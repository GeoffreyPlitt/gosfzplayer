//go:build jack
// +build jack

package gosfzplayer

import (
	"fmt"
	"math"
	"sync"

	"github.com/GeoffreyPlitt/debuggo"
	"github.com/xthexder/go-jack"
)

var jackDebug = debuggo.Debug("sfzplayer:jack")

// Helper function to clamp float64 values
func clampFloat64(value, min, max float64) float64 {
	if value > max {
		return max
	}
	if value < min {
		return min
	}
	return value
}

// JackClient represents a JACK audio client for the SFZ player
type JackClient struct {
	client       *jack.Client
	player       *SfzPlayer
	audioOutPort *jack.Port
	midiInPort   *jack.Port
	sampleRate   uint32
	bufferSize   uint32
	mu           sync.RWMutex

	// Audio rendering state
	activeVoices []*Voice
	maxVoices    int

	// Advanced Features
	currentKeyswitch uint8 // Currently active keyswitch
	activeNoteCount  int   // Count of active notes for trigger modes
	pitchBendValue   int16 // Current pitch bend value (-8192 to +8191)
}

// NewJackClient creates a new JACK client for the SFZ player
func NewJackClient(player *SfzPlayer, clientName string) (*JackClient, error) {
	jackDebug("Creating JACK client: %s", clientName)

	// Open JACK client
	client, err := jack.ClientOpen(clientName, jack.NoStartServer)
	if err != nil {
		return nil, fmt.Errorf("failed to open JACK client: %w", err)
	}

	jackClient := &JackClient{
		client:       client,
		player:       player,
		sampleRate:   uint32(client.GetSampleRate()),
		bufferSize:   uint32(client.GetBufferSize()),
		activeVoices: make([]*Voice, 0),
		maxVoices:    32, // Limit polyphony
	}

	// Register audio output port
	audioOutPort, err := client.PortRegister("audio_out", jack.DEFAULT_AUDIO_TYPE, jack.PortIsOutput, 0)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to register audio output port: %w", err)
	}
	jackClient.audioOutPort = audioOutPort

	// Register MIDI input port
	midiInPort, err := client.PortRegister("midi_in", jack.DEFAULT_MIDI_TYPE, jack.PortIsInput, 0)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to register MIDI input port: %w", err)
	}
	jackClient.midiInPort = midiInPort

	// Set process callback
	client.SetProcessCallback(jackClient.processCallback)

	jackDebug("JACK client created successfully (sample rate: %d Hz, buffer size: %d)",
		jackClient.sampleRate, jackClient.bufferSize)

	return jackClient, nil
}

// Start activates the JACK client and begins audio processing
func (jc *JackClient) Start() error {
	jackDebug("Starting JACK client")

	err := jc.client.Activate()
	if err != nil {
		return fmt.Errorf("failed to activate JACK client: %w", err)
	}

	jackDebug("JACK client activated successfully")
	return nil
}

// Stop deactivates the JACK client
func (jc *JackClient) Stop() error {
	jackDebug("Stopping JACK client")

	err := jc.client.Deactivate()
	if err != nil {
		return fmt.Errorf("failed to deactivate JACK client: %w", err)
	}

	jackDebug("JACK client deactivated")
	return nil
}

// Close closes the JACK client connection
func (jc *JackClient) Close() error {
	jackDebug("Closing JACK client")

	err := jc.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close JACK client: %w", err)
	}

	jackDebug("JACK client closed")
	return nil
}

// processCallback is called by JACK for each audio buffer
func (jc *JackClient) processCallback(nframes uint32) int {
	// Get audio output buffer
	audioOut := jc.audioOutPort.GetBuffer(nframes)
	audioOutSamples := jack.GetAudioSamples(audioOut, nframes)

	// Clear output buffer
	for i := range audioOutSamples {
		audioOutSamples[i] = 0.0
	}

	// Process MIDI input
	midiIn := jc.midiInPort.GetBuffer(nframes)
	jc.processMidiEvents(midiIn, nframes)

	// Render active voices
	jc.renderVoices(audioOutSamples, nframes)

	// Apply reverb if enabled
	if jc.player.reverbSend > 0.0 {
		jc.applyReverb(audioOutSamples, nframes)
	}

	return 0
}

// processMidiEvents processes incoming MIDI events
func (jc *JackClient) processMidiEvents(midiBuffer *jack.PortBuffer, nframes uint32) {
	eventCount := jack.MidiGetEventCount(midiBuffer)

	for i := uint32(0); i < eventCount; i++ {
		event, err := jack.MidiEventGet(midiBuffer, i)
		if err != nil {
			continue
		}

		if len(event.Buffer) < 1 {
			continue
		}

		// Parse MIDI message
		status := event.Buffer[0]

		switch status & 0xF0 {
		case 0x90: // Note On
			if len(event.Buffer) >= 3 {
				note := event.Buffer[1]
				velocity := event.Buffer[2]
				if velocity > 0 {
					jc.noteOn(note, velocity)
				} else {
					jc.noteOff(note)
				}
			}
		case 0x80: // Note Off
			if len(event.Buffer) >= 2 {
				note := event.Buffer[1]
				jc.noteOff(note)
			}
		case 0xB0: // Control Change (MIDI CC)
			if len(event.Buffer) >= 3 {
				cc := event.Buffer[1]
				value := event.Buffer[2]
				jc.processControlChange(cc, value)
			}
		case 0xE0: // Pitch Bend
			if len(event.Buffer) >= 3 {
				lsb := event.Buffer[1]
				msb := event.Buffer[2]
				jc.processPitchBend(lsb, msb)
			}
		}
	}
}

// noteOn handles MIDI note on events
func (jc *JackClient) noteOn(note, velocity uint8) {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	jackDebug("Note on: note=%d, velocity=%d", note, velocity)

	// Update keyswitch state - check if this note is in any keyswitch range
	jc.updateKeyswitchState(note)

	// Increment active note count for trigger modes
	jc.activeNoteCount++

	// Find matching regions
	for _, region := range jc.player.sfzData.Regions {
		if jc.regionMatches(region, note, velocity) {
			// Get sample for this region
			samplePath := region.GetStringOpcode("sample")
			if samplePath == "" {
				continue
			}

			sample, err := jc.player.GetSample(samplePath)
			if err != nil {
				jackDebug("Failed to get sample %s: %v", samplePath, err)
				continue
			}

			// Get advanced opcodes
			groupID := region.GetInheritedIntOpcode("group", 0)
			offByGroup := region.GetInheritedIntOpcode("off_by", 0)
			triggerMode := region.GetInheritedStringOpcode("trigger")
			if triggerMode == "" {
				triggerMode = "attack"
			}

			// Handle group exclusion - stop voices that should be stopped by this group
			if groupID > 0 {
				jc.stopVoicesByOffBy(groupID)
			}

			// Create new voice
			voice := &Voice{
				sample:      sample,
				region:      region,
				midiNote:    note,
				velocity:    velocity,
				position:    0.0,
				volume:      jc.calculateVolume(region, velocity),
				pan:         jc.calculatePan(region),
				pitchRatio:  jc.calculatePitchRatio(region, note),
				isActive:    true,
				noteOn:      true,
				groupID:     groupID,
				offByGroup:  offByGroup,
				triggerMode: triggerMode,
			}

			// Initialize ADSR envelope and loop parameters
			voice.InitializeEnvelope(jc.sampleRate)
			voice.InitializeLoop()

			// Add voice (replace oldest if at max polyphony)
			if len(jc.activeVoices) >= jc.maxVoices {
				jc.activeVoices = jc.activeVoices[1:] // Remove oldest voice
			}
			jc.activeVoices = append(jc.activeVoices, voice)

			jackDebug("Started voice for note %d, sample: %s", note, samplePath)
		}
	}
}

// noteOff handles MIDI note off events
func (jc *JackClient) noteOff(note uint8) {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	jackDebug("Note off: note=%d", note)

	// Decrement active note count
	jc.activeNoteCount--
	if jc.activeNoteCount < 0 {
		jc.activeNoteCount = 0
	}

	// Trigger release envelope for voices playing this note
	for _, voice := range jc.activeVoices {
		if voice.midiNote == note && voice.noteOn {
			voice.TriggerRelease()
		}
	}

	// Handle release trigger regions
	jc.handleReleaseTriggers(note)
}

// regionMatches checks if a region should respond to the given note and velocity
func (jc *JackClient) regionMatches(region *SfzSection, note, velocity uint8) bool {
	// Check key range
	lokey := region.GetInheritedIntOpcode("lokey", 0)
	hikey := region.GetInheritedIntOpcode("hikey", 127)
	key := region.GetInheritedIntOpcode("key", -1)

	// If key is specified, use it as both lokey and hikey
	if key >= 0 {
		lokey = key
		hikey = key
	}

	if int(note) < lokey || int(note) > hikey {
		return false
	}

	// Check velocity range
	lovel := region.GetInheritedIntOpcode("lovel", 1)
	hivel := region.GetInheritedIntOpcode("hivel", 127)

	if int(velocity) < lovel || int(velocity) > hivel {
		return false
	}

	// Check keyswitch range
	swLokey := region.GetInheritedIntOpcode("sw_lokey", -1)
	swHikey := region.GetInheritedIntOpcode("sw_hikey", -1)

	if swLokey >= 0 && swHikey >= 0 {
		// This region has keyswitch requirement
		if int(jc.currentKeyswitch) < swLokey || int(jc.currentKeyswitch) > swHikey {
			return false
		}
	}

	// Check trigger mode
	triggerMode := region.GetInheritedStringOpcode("trigger")
	if triggerMode == "" {
		triggerMode = "attack"
	}

	switch triggerMode {
	case "first":
		if jc.activeNoteCount > 1 { // We already incremented, so >1 means other notes are active
			return false
		}
	case "legato":
		if jc.activeNoteCount <= 1 { // No other notes active
			return false
		}
	case "release":
		return false // Release triggers are handled separately in noteOff
	}

	return true
}

// calculateVolume calculates the final volume for a voice
func (jc *JackClient) calculateVolume(region *SfzSection, velocity uint8) float64 {
	// Get volume with inheritance (Region → Group → Global)
	volume := region.GetInheritedFloatOpcode("volume", 0.0)

	// Clamp volume to reasonable range
	volume = clampFloat64(volume, -60.0, 6.0)

	// Convert dB to linear gain: linear = 10^(dB/20)
	linear := math.Pow(10.0, volume/20.0)

	// Velocity scaling (simplified)
	velocityScale := float64(velocity) / 127.0

	return linear * velocityScale
}

// calculatePan calculates the pan position for a voice
func (jc *JackClient) calculatePan(region *SfzSection) float64 {
	// Get pan with inheritance (Region → Group → Global)
	pan := region.GetInheritedFloatOpcode("pan", 0.0)

	// Clamp pan to valid range
	pan = clampFloat64(pan, -100.0, 100.0)

	return pan / 100.0 // Normalize to -1.0 to 1.0
}

// calculatePitchRatio calculates the pitch adjustment ratio for a voice
func (jc *JackClient) calculatePitchRatio(region *SfzSection, midiNote uint8) float64 {
	// Get pitch_keycenter (root note) with inheritance - default to played note if not specified
	pitchKeycenter := region.GetInheritedIntOpcode("pitch_keycenter", int(midiNote))

	// Calculate semitone difference from pitch_keycenter
	semitones := float64(int(midiNote) - pitchKeycenter)

	// Apply transpose (in semitones) with inheritance
	transpose := region.GetInheritedIntOpcode("transpose", 0)
	semitones += float64(transpose)

	// Apply tune (in cents) with inheritance - convert cents to semitones
	tune := region.GetInheritedFloatOpcode("tune", 0.0)
	semitones += tune / 100.0 // 100 cents = 1 semitone

	// Apply pitch (in cents) with inheritance - convert cents to semitones
	pitch := region.GetInheritedFloatOpcode("pitch", 0.0)
	semitones += pitch / 100.0 // 100 cents = 1 semitone

	// Apply pitch bend
	if jc.pitchBendValue != 0 {
		bendUp := region.GetInheritedIntOpcode("bend_up", 200)      // Default 200 cents up
		bendDown := region.GetInheritedIntOpcode("bend_down", -200) // Default 200 cents down

		// Calculate pitch bend range and apply
		if jc.pitchBendValue > 0 {
			// Positive pitch bend - scale to bend_up range
			bendSemitones := float64(jc.pitchBendValue) / 8192.0 * float64(bendUp) / 100.0
			semitones += bendSemitones
		} else {
			// Negative pitch bend - scale to bend_down range
			bendSemitones := float64(jc.pitchBendValue) / 8192.0 * float64(-bendDown) / 100.0
			semitones += bendSemitones
		}
	}

	// Convert semitones to pitch ratio: ratio = 2^(semitones/12)
	pitchRatio := math.Pow(2.0, semitones/12.0)

	// Clamp pitch ratio to reasonable range (avoid extreme values)
	pitchRatio = clampFloat64(pitchRatio, 0.1, 10.0)

	jackDebug("Pitch adjustment: note=%d, keycenter=%d, transpose=%d, tune=%.1fc, pitch=%.1fc, total_semitones=%.2f, ratio=%f",
		midiNote, pitchKeycenter, transpose, tune, pitch, semitones, pitchRatio)

	return pitchRatio
}

// renderVoices renders all active voices to the output buffer
func (jc *JackClient) renderVoices(output []jack.AudioSample, nframes uint32) {
	jc.mu.RLock()
	defer jc.mu.RUnlock()

	// Process each active voice
	for i := len(jc.activeVoices) - 1; i >= 0; i-- {
		voice := jc.activeVoices[i]

		if !voice.isActive {
			// Remove inactive voice
			jc.activeVoices = append(jc.activeVoices[:i], jc.activeVoices[i+1:]...)
			continue
		}

		jc.renderVoice(voice, output, nframes)
	}
}

// renderVoice renders a single voice to the output buffer with pitch-shifting
func (jc *JackClient) renderVoice(voice *Voice, output []jack.AudioSample, nframes uint32) {
	sample := voice.sample
	maxSamples := len(sample.Data)

	// Handle mono vs stereo sample indexing
	var samplesPerFrame int
	if sample.Channels == 1 {
		samplesPerFrame = 1
	} else {
		samplesPerFrame = 2
		maxSamples = maxSamples / 2 // For stereo, we count frames not individual samples
	}

	for i := uint32(0); i < nframes; i++ {
		// Process envelope
		envelopeLevel := voice.ProcessEnvelope()

		// Check if envelope is finished
		if envelopeLevel <= 0.0 && voice.envelopeState == EnvelopeOff {
			voice.isActive = false
			break
		}

		// Get the interpolated sample value
		sampleValue := jc.getInterpolatedSample(sample, voice.position, samplesPerFrame)

		// Apply volume and envelope
		sampleValue *= voice.volume * envelopeLevel

		// For now, output to mono (ignore panning)
		output[i] += jack.AudioSample(sampleValue)

		// Advance position by pitch ratio
		voice.position += voice.pitchRatio

		// Process loop behavior
		if !voice.ProcessLoop() {
			voice.isActive = false
			break
		}
	}
}

// getInterpolatedSample performs linear interpolation between sample points
func (jc *JackClient) getInterpolatedSample(sample *Sample, position float64, samplesPerFrame int) float64 {
	// Get integer and fractional parts of position
	intPos := int(position)
	fracPos := position - float64(intPos)

	// Ensure we don't go out of bounds
	maxFrames := len(sample.Data) / samplesPerFrame
	if intPos >= maxFrames {
		return 0.0
	}

	// Get current sample
	var sample1 float64
	if samplesPerFrame == 1 {
		// Mono
		sample1 = sample.Data[intPos]
	} else {
		// Stereo - use left channel for now
		sample1 = sample.Data[intPos*2]
	}

	// Get next sample for interpolation
	var sample2 float64
	if intPos+1 < maxFrames {
		if samplesPerFrame == 1 {
			// Mono
			sample2 = sample.Data[intPos+1]
		} else {
			// Stereo - use left channel for now
			sample2 = sample.Data[(intPos+1)*2]
		}
	} else {
		// At end of sample, use same value
		sample2 = sample1
	}

	// Linear interpolation: result = sample1 + fracPos * (sample2 - sample1)
	return sample1 + fracPos*(sample2-sample1)
}

// processControlChange handles MIDI Control Change messages
func (jc *JackClient) processControlChange(cc, value uint8) {
	// Convert MIDI value (0-127) to float (0.0-1.0)
	floatValue := float64(value) / 127.0

	switch cc {
	case 91: // Standard MIDI CC for reverb send/depth
		jc.player.SetReverbSend(floatValue)
		jackDebug("MIDI CC91 (Reverb Send): %.3f", floatValue)

	case 92: // Reverb room size (custom mapping)
		jc.player.SetReverbRoomSize(floatValue)
		jackDebug("MIDI CC92 (Reverb Room Size): %.3f", floatValue)

	case 93: // Reverb damping (custom mapping)
		jc.player.SetReverbDamping(floatValue)
		jackDebug("MIDI CC93 (Reverb Damping): %.3f", floatValue)

	case 94: // Reverb wet level (custom mapping)
		jc.player.SetReverbWet(floatValue)
		jackDebug("MIDI CC94 (Reverb Wet): %.3f", floatValue)

	case 95: // Reverb dry level (custom mapping)
		jc.player.SetReverbDry(floatValue)
		jackDebug("MIDI CC95 (Reverb Dry): %.3f", floatValue)

	default:
		// Log unknown CC for debugging
		jackDebug("Unknown MIDI CC%d: %d", cc, value)
	}
}

// processPitchBend handles MIDI Pitch Bend messages
func (jc *JackClient) processPitchBend(lsb, msb uint8) {
	// Convert 14-bit pitch bend value to signed 16-bit (-8192 to +8191)
	// LSB = low 7 bits, MSB = high 7 bits
	bendValue := int16((uint16(msb)<<7)|uint16(lsb)) - 8192

	jc.pitchBendValue = bendValue
	jackDebug("Pitch Bend: %d (%.3f semitones)", bendValue, float64(bendValue)/8192.0*2.0)
}

// applyReverb applies reverb processing to the audio buffer
func (jc *JackClient) applyReverb(audioBuffer []jack.AudioSample, nframes uint32) {
	// Convert jack.AudioSample to float64, process through reverb, and convert back
	for i := uint32(0); i < nframes; i++ {
		// Convert to float64
		input := float64(audioBuffer[i])

		// Apply reverb send level
		reverbInput := input * jc.player.reverbSend

		// Process through reverb (mono)
		reverbOutput := jc.player.reverb.ProcessMono(reverbInput)

		// Mix with dry signal
		dryLevel := 1.0 - jc.player.reverbSend
		output := (input * dryLevel) + reverbOutput

		// Convert back to jack.AudioSample and clamp
		audioBuffer[i] = jack.AudioSample(clampFloat64(output, -1.0, 1.0))
	}
}

// updateKeyswitchState updates the current keyswitch based on incoming note
func (jc *JackClient) updateKeyswitchState(note uint8) {
	// Check all regions for keyswitch ranges and update current keyswitch
	for _, region := range jc.player.sfzData.Regions {
		swLokey := region.GetInheritedIntOpcode("sw_lokey", -1)
		swHikey := region.GetInheritedIntOpcode("sw_hikey", -1)

		if swLokey >= 0 && swHikey >= 0 {
			if int(note) >= swLokey && int(note) <= swHikey {
				jc.currentKeyswitch = note
				jackDebug("Keyswitch updated: %d", note)
				return
			}
		}
	}
}

// stopVoicesByOffBy stops all active voices that should be stopped by the given group
func (jc *JackClient) stopVoicesByOffBy(groupID int) {
	for i := len(jc.activeVoices) - 1; i >= 0; i-- {
		voice := jc.activeVoices[i]
		if voice.offByGroup == groupID {
			jackDebug("Stopping voice (group exclusion): note=%d, stopped_by_group=%d", voice.midiNote, groupID)
			// Remove voice immediately
			jc.activeVoices = append(jc.activeVoices[:i], jc.activeVoices[i+1:]...)
		}
	}
}

// handleReleaseTriggers handles release trigger regions when a note is released
func (jc *JackClient) handleReleaseTriggers(note uint8) {
	// Find regions with trigger=release that match this note
	for _, region := range jc.player.sfzData.Regions {
		triggerMode := region.GetInheritedStringOpcode("trigger")
		if triggerMode == "release" {
			// Check if this region matches the released note (without trigger mode check)
			if jc.regionMatchesForRelease(region, note) {
				// Get sample for this region
				samplePath := region.GetStringOpcode("sample")
				if samplePath == "" {
					continue
				}

				sample, err := jc.player.GetSample(samplePath)
				if err != nil {
					jackDebug("Failed to get release sample %s: %v", samplePath, err)
					continue
				}

				// Create release voice
				voice := &Voice{
					sample:      sample,
					region:      region,
					midiNote:    note,
					velocity:    64, // Use moderate velocity for release triggers
					position:    0.0,
					volume:      jc.calculateVolume(region, 64),
					pan:         jc.calculatePan(region),
					pitchRatio:  jc.calculatePitchRatio(region, note),
					isActive:    true,
					noteOn:      false, // Release triggers don't respond to note-off
					groupID:     region.GetInheritedIntOpcode("group", 0),
					offByGroup:  region.GetInheritedIntOpcode("off_by", 0),
					triggerMode: "release",
				}

				// Initialize envelope and loop
				voice.InitializeEnvelope(jc.sampleRate)
				voice.InitializeLoop()

				// Add voice
				if len(jc.activeVoices) >= jc.maxVoices {
					jc.activeVoices = jc.activeVoices[1:]
				}
				jc.activeVoices = append(jc.activeVoices, voice)

				jackDebug("Started release voice for note %d", note)
			}
		}
	}
}

// regionMatchesForRelease checks if a region matches for release triggers (without trigger mode check)
func (jc *JackClient) regionMatchesForRelease(region *SfzSection, note uint8) bool {
	// Check key range
	lokey := region.GetInheritedIntOpcode("lokey", 0)
	hikey := region.GetInheritedIntOpcode("hikey", 127)
	key := region.GetInheritedIntOpcode("key", -1)

	if key >= 0 {
		lokey = key
		hikey = key
	}

	if int(note) < lokey || int(note) > hikey {
		return false
	}

	// Check keyswitch range (same as normal matching)
	swLokey := region.GetInheritedIntOpcode("sw_lokey", -1)
	swHikey := region.GetInheritedIntOpcode("sw_hikey", -1)

	if swLokey >= 0 && swHikey >= 0 {
		if int(jc.currentKeyswitch) < swLokey || int(jc.currentKeyswitch) > swHikey {
			return false
		}
	}

	return true
}
