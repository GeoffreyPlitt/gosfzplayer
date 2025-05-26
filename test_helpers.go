//go:build !jack
// +build !jack

package gosfzplayer

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"testing"
)

// MockJackClient implements the core JACK rendering logic without actual JACK dependency
type MockJackClient struct {
	player       *SfzPlayer
	sampleRate   uint32
	bufferSize   uint32
	activeVoices []*Voice
	maxVoices    int
}

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

// Helper function to get sample value accounting for stereo/mono
func getSampleValue(sample *Sample, frameIndex int, channel int) float64 {
	if sample.Channels == 1 {
		if frameIndex >= len(sample.Data) {
			return 0.0
		}
		return sample.Data[frameIndex]
	} else {
		// Stereo
		sampleIndex := frameIndex*2 + channel
		if sampleIndex >= len(sample.Data) {
			return 0.0
		}
		return sample.Data[sampleIndex]
	}
}

// Helper function to validate MIDI message buffer
func validateMidiMessage(buffer []byte, minLength int) bool {
	return len(buffer) >= minLength
}

// createTestSfzFile creates a temporary SFZ file with given content and returns cleanup function
func createTestSfzFile(t *testing.T, content string) (string, func()) {
	tmpFile, err := ioutil.TempFile("", "test_*.sfz")
	if err != nil {
		t.Fatalf("Failed to create temp SFZ file: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to write to temp SFZ file: %v", err)
	}
	tmpFile.Close()

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return tmpFile.Name(), cleanup
}

// createTestPlayer creates a test SFZ player with error handling
func createTestPlayer(t *testing.T) *SfzPlayer {
	player, err := NewSfzPlayer("testdata/test.sfz", "test-client")
	if err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}
	return player
}

// createTestVoice creates a test voice with given opcodes
func createTestVoice(opcodes map[string]string, sampleRate uint32) *Voice {
	// Create a test section with the opcodes
	section := &SfzSection{
		Type:    "region",
		Opcodes: opcodes,
	}

	// Create a test sample
	testSample := &Sample{
		FilePath:   "test.wav",
		Data:       make([]float64, 1000), // 1000 sample frames
		SampleRate: int(sampleRate),
		Channels:   1,
		Length:     1000,
	}

	// Create voice
	voice := &Voice{
		sample:     testSample,
		region:     section,
		midiNote:   60, // Middle C
		velocity:   100,
		position:   0.0,
		volume:     1.0,
		pan:        0.0,
		pitchRatio: 1.0,
		isActive:   true,
		noteOn:     true,
	}

	// Initialize envelope and loop
	voice.InitializeEnvelope(sampleRate)
	voice.InitializeLoop()

	return voice
}

// createTestSample creates a test sample with specified parameters
func createTestSample(size int, channels int) *Sample {
	data := make([]float64, size*channels)

	// Fill with a simple sine wave for testing
	for i := 0; i < size; i++ {
		sampleValue := math.Sin(float64(i)*2.0*math.Pi*440.0/44100.0) * 0.5
		for ch := 0; ch < channels; ch++ {
			data[i*channels+ch] = sampleValue
		}
	}

	return &Sample{
		FilePath:   "test.wav",
		Data:       data,
		SampleRate: 44100,
		Channels:   channels,
		Length:     size,
	}
}

// assertOpcode checks that a section has the expected opcode value
func assertOpcode(t *testing.T, section *SfzSection, opcode, expected string) {
	t.Helper()
	actual := section.GetStringOpcode(opcode)
	if actual != expected {
		t.Errorf("Expected %s=%s, got %s", opcode, expected, actual)
	}
}

// assertIntOpcode checks that a section has the expected int opcode value
func assertIntOpcode(t *testing.T, section *SfzSection, opcode string, expected int) {
	t.Helper()
	actual := section.GetIntOpcode(opcode, -999)
	if actual != expected {
		t.Errorf("Expected %s=%d, got %d", opcode, expected, actual)
	}
}

// assertFloatOpcode checks that a section has the expected float opcode value
func assertFloatOpcode(t *testing.T, section *SfzSection, opcode string, expected float64) {
	t.Helper()
	actual := section.GetFloatOpcode(opcode, -999.0)
	if math.Abs(actual-expected) > 0.001 {
		t.Errorf("Expected %s=%.3f, got %.3f", opcode, expected, actual)
	}
}

// MockJackClient methods

func (mjc *MockJackClient) noteOn(note, velocity uint8) {
	// Find matching regions
	for _, region := range mjc.player.sfzData.Regions {
		if mjc.regionMatches(region, note, velocity) {
			// Get sample for this region
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
				mjc.activeVoices = mjc.activeVoices[1:] // Remove oldest voice
			}
			mjc.activeVoices = append(mjc.activeVoices, voice)
		}
	}
}

func (mjc *MockJackClient) noteOff(note uint8) {
	// Trigger release envelope for voices playing this note
	for _, voice := range mjc.activeVoices {
		if voice.midiNote == note && voice.noteOn {
			voice.TriggerRelease()
		}
	}
}

func (mjc *MockJackClient) regionMatches(region *SfzSection, note, velocity uint8) bool {
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

	return true
}

func (mjc *MockJackClient) calculateVolume(region *SfzSection, velocity uint8) float64 {
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

func (mjc *MockJackClient) calculatePan(region *SfzSection) float64 {
	// Get pan with inheritance (Region → Group → Global)
	pan := region.GetInheritedFloatOpcode("pan", 0.0)

	// Clamp pan to valid range
	pan = clampFloat64(pan, -100.0, 100.0)

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
			// Remove inactive voice
			mjc.activeVoices = append(mjc.activeVoices[:i], mjc.activeVoices[i+1:]...)
			continue
		}

		mjc.renderVoice(voice, output, nframes)
	}

	// Apply reverb if enabled
	if mjc.player.reverbSend > 0.0 {
		mjc.applyReverb(output, nframes)
	}
}

func (mjc *MockJackClient) renderVoice(voice *Voice, output []float32, nframes uint32) {
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
		sampleValue := mjc.getInterpolatedSample(sample, voice.position, samplesPerFrame)

		// Apply volume and envelope
		sampleValue *= voice.volume * envelopeLevel

		// For now, output to mono (ignore panning)
		output[i] += float32(sampleValue)

		// Advance position by pitch ratio
		voice.position += voice.pitchRatio

		// Process loop behavior
		if !voice.ProcessLoop() {
			voice.isActive = false
			break
		}
	}
}

func (mjc *MockJackClient) getInterpolatedSample(sample *Sample, position float64, samplesPerFrame int) float64 {
	// Get integer and fractional parts of position
	intPos := int(position)
	fracPos := position - float64(intPos)

	// Ensure we don't go out of bounds
	maxFrames := len(sample.Data) / samplesPerFrame
	if intPos >= maxFrames {
		return 0.0
	}

	// Get current sample
	sample1 := getSampleValue(sample, intPos, 0) // Use left channel for mono output

	// Get next sample for interpolation
	var sample2 float64
	if intPos+1 < maxFrames {
		sample2 = getSampleValue(sample, intPos+1, 0)
	} else {
		// At end of sample, use same value
		sample2 = sample1
	}

	// Linear interpolation: result = sample1 + fracPos * (sample2 - sample1)
	return sample1 + fracPos*(sample2-sample1)
}

// saveWAV saves float32 audio data as a WAV file
func saveWAV(filename string, data []float32, sampleRate int) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create WAV file: %w", err)
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

// createTestMockClient creates a mock JACK client for testing
func createTestMockClient(player *SfzPlayer, sampleRate uint32, bufferSize uint32) *MockJackClient {
	return &MockJackClient{
		player:       player,
		sampleRate:   sampleRate,
		bufferSize:   bufferSize,
		activeVoices: make([]*Voice, 0),
		maxVoices:    32,
	}
}

// applyReverb applies reverb processing to the audio buffer (MockJackClient version)
func (mjc *MockJackClient) applyReverb(audioBuffer []float32, nframes uint32) {
	// Convert float32 to float64, process through reverb, and convert back
	for i := uint32(0); i < nframes; i++ {
		// Convert to float64
		input := float64(audioBuffer[i])

		// Apply reverb send level
		reverbInput := input * mjc.player.reverbSend

		// Process through reverb (mono)
		reverbOutput := mjc.player.reverb.ProcessMono(reverbInput)

		// Mix with dry signal
		dryLevel := 1.0 - mjc.player.reverbSend
		output := (input * dryLevel) + reverbOutput

		// Convert back to float32 and clamp
		audioBuffer[i] = float32(clampFloat64(output, -1.0, 1.0))
	}
}
