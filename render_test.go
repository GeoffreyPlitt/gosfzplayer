//go:build !jack
// +build !jack

package gosfzplayer

import (
	"encoding/binary"
	"math"
	"os"
	"testing"
	"time"
)

func TestRenderPianoArpeggio(t *testing.T) {
	// Skip if piano samples not available
	if _, err := os.Stat("testdata/piano.sfz"); os.IsNotExist(err) {
		t.Skip("piano.sfz not found, run 'go generate' to download piano samples")
	}

	// Create SFZ player
	player, err := NewSfzPlayer("testdata/piano.sfz", "")
	if err != nil {
		t.Fatalf("Failed to create piano SFZ player: %v", err)
	}
	defer player.StopAndClose()

	// Create a mock JACK client for offline rendering
	mockClient := &MockJackClient{
		player:       player,
		sampleRate:   44100,
		bufferSize:   512,
		activeVoices: make([]*Voice, 0),
		maxVoices:    32,
	}

	// Define 4-octave C major arpeggio pattern: C-E-G-C-E-G-C-E-G-C-E-G-C (up 4 octaves)
	// MIDI notes: 48(C3) 52(E3) 55(G3) 60(C4) 64(E4) 67(G4) 72(C5) 76(E5) 79(G5) 84(C6) 88(E6) 91(G6) 96(C7)
	arpeggioNotes := []uint8{48, 52, 55, 60, 64, 67, 72, 76, 79, 84, 88, 91, 96}

	// Timing parameters
	sampleRate := 44100
	noteLength := time.Second * 1                                               // 1 second per quarter note
	totalDuration := time.Duration(len(arpeggioNotes))*noteLength + time.Second // Extra second for decay
	totalSamples := int(float64(sampleRate) * totalDuration.Seconds())

	// Prepare output buffer
	outputBuffer := make([]float32, totalSamples)

	// Render the arpeggio
	bufferSize := 512
	currentSample := 0

	// Track which notes have been triggered/released to avoid timing window misses
	noteTriggered := make([]bool, len(arpeggioNotes))
	noteReleased := make([]bool, len(arpeggioNotes))

	for currentSample < totalSamples {
		// Calculate current time position
		currentTime := float64(currentSample) / float64(sampleRate)

		// Trigger notes at appropriate times
		for i, note := range arpeggioNotes {
			noteStartTime := float64(i) * noteLength.Seconds()
			noteEndTime := noteStartTime + 0.8 // 800ms note duration (80% of quarter note)

			// Check if we should trigger this note (trigger once when time is reached)
			if !noteTriggered[i] && currentTime >= noteStartTime {
				mockClient.noteOn(note, 100) // velocity 100
				noteTriggered[i] = true
			}

			// Check if we should release this note (release once when time is reached)
			if !noteReleased[i] && noteTriggered[i] && currentTime >= noteEndTime {
				mockClient.noteOff(note)
				noteReleased[i] = true
			}
		}

		// Render this buffer
		framesToRender := bufferSize
		if currentSample+bufferSize > totalSamples {
			framesToRender = totalSamples - currentSample
		}

		audioBuffer := make([]float32, framesToRender)
		mockClient.renderVoices(audioBuffer, uint32(framesToRender))

		// Copy to output buffer
		copy(outputBuffer[currentSample:currentSample+framesToRender], audioBuffer)
		currentSample += framesToRender
	}

	// Save as WAV file
	outputPath := "testdata/piano_arpeggio.wav"
	err = saveWAV(outputPath, outputBuffer, sampleRate)
	if err != nil {
		t.Fatalf("Failed to save WAV file: %v", err)
	}

	t.Logf("Rendered piano arpeggio to %s (%.2f seconds, %d samples)",
		outputPath, totalDuration.Seconds(), totalSamples)
}

// MockJackClient implements the core JACK rendering logic without actual JACK dependency
type MockJackClient struct {
	player       *SfzPlayer
	sampleRate   uint32
	bufferSize   uint32
	activeVoices []*Voice
	maxVoices    int
}

func (mjc *MockJackClient) noteOn(note, velocity uint8) {
	// Find matching regions (copied from jack.go)
	for _, region := range mjc.player.sfzData.Regions {
		if mjc.regionMatches(region, note, velocity) {
			samplePath := region.GetStringOpcode("sample")
			if samplePath == "" {
				continue
			}

			sample, err := mjc.player.GetSample(samplePath)
			if err != nil {
				continue
			}

			// Create new voice
			voice := &Voice{
				sample:     sample,
				region:     region,
				midiNote:   note,
				velocity:   velocity,
				position:   0.0,
				volume:     mjc.calculateVolume(region, velocity),
				pan:        mjc.calculatePan(region),
				pitchRatio: mjc.calculatePitchRatio(region, note),
				isActive:   true,
				noteOn:     true,
			}

			// Initialize ADSR envelope and loop parameters
			voice.InitializeEnvelope(mjc.sampleRate)
			voice.InitializeLoop()

			// Add voice (replace oldest if at max polyphony)
			if len(mjc.activeVoices) >= mjc.maxVoices {
				mjc.activeVoices = mjc.activeVoices[1:]
			}
			mjc.activeVoices = append(mjc.activeVoices, voice)
		}
	}
}

func (mjc *MockJackClient) noteOff(note uint8) {
	// Mark voices for this note as released
	for _, voice := range mjc.activeVoices {
		if voice.midiNote == note && voice.noteOn {
			voice.noteOn = false
			voice.isActive = false // Simple release for now
		}
	}
}

func (mjc *MockJackClient) regionMatches(region *SfzSection, note, velocity uint8) bool {
	// Check key range with inheritance
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

	// Check velocity range with inheritance
	lovel := region.GetInheritedIntOpcode("lovel", 1)
	hivel := region.GetInheritedIntOpcode("hivel", 127)

	return int(velocity) >= lovel && int(velocity) <= hivel
}

func (mjc *MockJackClient) calculateVolume(region *SfzSection, velocity uint8) float64 {
	// Get volume with inheritance (Region → Group → Global)
	volume := region.GetInheritedFloatOpcode("volume", 0.0)

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

func (mjc *MockJackClient) calculatePan(region *SfzSection) float64 {
	// Get pan with inheritance (Region → Group → Global)
	pan := region.GetInheritedFloatOpcode("pan", 0.0)

	// Clamp pan to valid range
	if pan > 100.0 {
		pan = 100.0
	}
	if pan < -100.0 {
		pan = -100.0
	}

	return pan / 100.0 // Normalize to -1.0 to 1.0
}

func (mjc *MockJackClient) calculatePitchRatio(region *SfzSection, midiNote uint8) float64 {
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

	// Convert semitones to pitch ratio: ratio = 2^(semitones/12)
	return math.Pow(2.0, semitones/12.0)
}

func (mjc *MockJackClient) renderVoices(output []float32, nframes uint32) {
	// Process each active voice
	for i := len(mjc.activeVoices) - 1; i >= 0; i-- {
		voice := mjc.activeVoices[i]

		if !voice.isActive {
			mjc.activeVoices = append(mjc.activeVoices[:i], mjc.activeVoices[i+1:]...)
			continue
		}

		mjc.renderVoice(voice, output, nframes)
	}
}

func (mjc *MockJackClient) renderVoice(voice *Voice, output []float32, nframes uint32) {
	sample := voice.sample
	maxSamples := len(sample.Data)

	samplesPerFrame := 1
	if sample.Channels == 2 {
		samplesPerFrame = 2
		maxSamples = maxSamples / 2
	}

	for i := uint32(0); i < nframes; i++ {
		// Process envelope
		envelopeLevel := voice.ProcessEnvelope()

		// Check if envelope is finished
		if envelopeLevel <= 0.0 && voice.envelopeState == EnvelopeOff {
			voice.isActive = false
			break
		}

		sampleValue := mjc.getInterpolatedSample(sample, voice.position, samplesPerFrame)
		sampleValue *= voice.volume * envelopeLevel

		output[i] += float32(sampleValue)
		voice.position += voice.pitchRatio

		// Process loop behavior
		if !voice.ProcessLoop() {
			voice.isActive = false
			break
		}
	}
}

func (mjc *MockJackClient) getInterpolatedSample(sample *Sample, position float64, samplesPerFrame int) float64 {
	intPos := int(position)
	fracPos := position - float64(intPos)

	// Ensure we don't go out of bounds
	maxFrames := len(sample.Data) / samplesPerFrame
	if intPos >= maxFrames {
		return 0.0
	}

	var sample1 float64
	if samplesPerFrame == 1 {
		sample1 = sample.Data[intPos]
	} else {
		sample1 = sample.Data[intPos*2]
	}

	var sample2 float64
	if intPos+1 < maxFrames {
		if samplesPerFrame == 1 {
			sample2 = sample.Data[intPos+1]
		} else {
			sample2 = sample.Data[(intPos+1)*2]
		}
	} else {
		sample2 = sample1
	}

	return sample1 + fracPos*(sample2-sample1)
}

// saveWAV saves audio data as a WAV file
func saveWAV(filename string, data []float32, sampleRate int) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// WAV header
	numSamples := len(data)
	numChannels := 1
	bitsPerSample := 16
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := numSamples * blockAlign

	// Write RIFF header
	file.WriteString("RIFF")
	binary.Write(file, binary.LittleEndian, uint32(36+dataSize))
	file.WriteString("WAVE")

	// Write fmt chunk
	file.WriteString("fmt ")
	binary.Write(file, binary.LittleEndian, uint32(16))            // Chunk size
	binary.Write(file, binary.LittleEndian, uint16(1))             // Audio format (PCM)
	binary.Write(file, binary.LittleEndian, uint16(numChannels))   // Number of channels
	binary.Write(file, binary.LittleEndian, uint32(sampleRate))    // Sample rate
	binary.Write(file, binary.LittleEndian, uint32(byteRate))      // Byte rate
	binary.Write(file, binary.LittleEndian, uint16(blockAlign))    // Block align
	binary.Write(file, binary.LittleEndian, uint16(bitsPerSample)) // Bits per sample

	// Write data chunk
	file.WriteString("data")
	binary.Write(file, binary.LittleEndian, uint32(dataSize))

	// Convert float32 to int16 and write
	for _, sample := range data {
		// Clamp to [-1, 1] and convert to int16
		if sample > 1.0 {
			sample = 1.0
		}
		if sample < -1.0 {
			sample = -1.0
		}
		int16Sample := int16(sample * 32767)
		binary.Write(file, binary.LittleEndian, int16Sample)
	}

	return nil
}
