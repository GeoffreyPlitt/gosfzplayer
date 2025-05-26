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
}

// Voice represents an active playing voice/note
type Voice struct {
	sample   *Sample
	region   *SfzSection
	midiNote uint8
	velocity uint8
	position int64 // Current playback position in samples
	volume   float64
	pan      float64
	isActive bool
	noteOn   bool
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
		}
	}
}

// noteOn handles MIDI note on events
func (jc *JackClient) noteOn(note, velocity uint8) {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	jackDebug("Note on: note=%d, velocity=%d", note, velocity)

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

			// Create new voice
			voice := &Voice{
				sample:   sample,
				region:   region,
				midiNote: note,
				velocity: velocity,
				position: 0,
				volume:   jc.calculateVolume(region, velocity),
				pan:      jc.calculatePan(region),
				isActive: true,
				noteOn:   true,
			}

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

	// Mark voices for this note as released
	for _, voice := range jc.activeVoices {
		if voice.midiNote == note && voice.noteOn {
			voice.noteOn = false
			// For now, just stop the voice immediately
			// TODO: Implement proper release envelope
			voice.isActive = false
		}
	}
}

// regionMatches checks if a region should respond to the given note and velocity
func (jc *JackClient) regionMatches(region *SfzSection, note, velocity uint8) bool {
	// Check key range
	lokey := region.GetIntOpcode("lokey", 0)
	hikey := region.GetIntOpcode("hikey", 127)
	key := region.GetIntOpcode("key", -1)

	// If key is specified, use it as both lokey and hikey
	if key >= 0 {
		lokey = key
		hikey = key
	}

	if int(note) < lokey || int(note) > hikey {
		return false
	}

	// Check velocity range
	lovel := region.GetIntOpcode("lovel", 1)
	hivel := region.GetIntOpcode("hivel", 127)

	if int(velocity) < lovel || int(velocity) > hivel {
		return false
	}

	return true
}

// calculateVolume calculates the final volume for a voice
func (jc *JackClient) calculateVolume(region *SfzSection, velocity uint8) float64 {
	// Base volume from region
	volume := region.GetFloatOpcode("volume", 0.0)

	// Apply global volume if present
	if jc.player.sfzData.Global != nil {
		globalVolume := jc.player.sfzData.Global.GetFloatOpcode("volume", 0.0)
		volume += globalVolume
	}

	// Clamp volume to reasonable range
	if volume > 6.0 {
		volume = 6.0
	}
	if volume < -60.0 {
		volume = -60.0
	}

	// Convert dB to linear gain: linear = 10^(dB/20)
	linear := math.Pow(10.0, volume/20.0)

	// Velocity scaling (simplified)
	velocityScale := float64(velocity) / 127.0

	return linear * velocityScale
}

// calculatePan calculates the pan position for a voice
func (jc *JackClient) calculatePan(region *SfzSection) float64 {
	pan := region.GetFloatOpcode("pan", 0.0)

	// Apply global pan if present
	if jc.player.sfzData.Global != nil {
		globalPan := jc.player.sfzData.Global.GetFloatOpcode("pan", 0.0)
		pan += globalPan
	}

	// Clamp pan to valid range
	if pan > 100.0 {
		pan = 100.0
	}
	if pan < -100.0 {
		pan = -100.0
	}

	return pan / 100.0 // Normalize to -1.0 to 1.0
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

// renderVoice renders a single voice to the output buffer
func (jc *JackClient) renderVoice(voice *Voice, output []jack.AudioSample, nframes uint32) {
	sample := voice.sample

	for i := uint32(0); i < nframes; i++ {
		if voice.position >= int64(len(sample.Data)) {
			voice.isActive = false
			break
		}

		// Get sample data (handle mono/stereo)
		var sampleValue float64
		if sample.Channels == 1 {
			sampleValue = sample.Data[voice.position]
		} else {
			// For stereo, just use left channel for now
			sampleValue = sample.Data[voice.position*2]
		}

		// Apply volume
		sampleValue *= voice.volume

		// For now, output to mono (ignore panning)
		output[i] += jack.AudioSample(sampleValue)

		voice.position++
	}
}
