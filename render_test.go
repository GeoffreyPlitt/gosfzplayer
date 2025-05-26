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

	// Define 2-octave C major arpeggio pattern: C-E-G-C-E-G-C (up 2 octaves)
	// MIDI notes: 60(C4) 64(E4) 67(G4) 72(C5) 76(E5) 79(G5) 84(C6)
	arpeggioNotes := []uint8{60, 64, 67, 72, 76, 79, 84}

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

	for currentSample < totalSamples {
		// Calculate current time position
		currentTime := float64(currentSample) / float64(sampleRate)

		// Trigger notes at appropriate times
		for i, note := range arpeggioNotes {
			noteStartTime := float64(i) * noteLength.Seconds()
			noteEndTime := noteStartTime + 0.8 // 800ms note duration (80% of quarter note)

			// Check if we should trigger this note
			if currentTime >= noteStartTime && currentTime < noteStartTime+0.01 { // 10ms trigger window
				mockClient.noteOn(note, 100) // velocity 100
			}

			// Check if we should release this note
			if currentTime >= noteEndTime && currentTime < noteEndTime+0.01 {
				mockClient.noteOff(note)
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
	// Check key range
	lokey := region.GetIntOpcode("lokey", 0)
	hikey := region.GetIntOpcode("hikey", 127)
	key := region.GetIntOpcode("key", -1)

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

	return int(velocity) >= lovel && int(velocity) <= hivel
}

func (mjc *MockJackClient) calculateVolume(region *SfzSection, velocity uint8) float64 {
	volume := region.GetFloatOpcode("volume", 0.0)

	if mjc.player.sfzData.Global != nil {
		globalVolume := mjc.player.sfzData.Global.GetFloatOpcode("volume", 0.0)
		volume += globalVolume
	}

	if volume > 6.0 {
		volume = 6.0
	}
	if volume < -60.0 {
		volume = -60.0
	}

	linear := math.Pow(10.0, volume/20.0)
	velocityScale := float64(velocity) / 127.0

	return linear * velocityScale
}

func (mjc *MockJackClient) calculatePan(region *SfzSection) float64 {
	pan := region.GetFloatOpcode("pan", 0.0)

	if mjc.player.sfzData.Global != nil {
		globalPan := mjc.player.sfzData.Global.GetFloatOpcode("pan", 0.0)
		pan += globalPan
	}

	if pan > 100.0 {
		pan = 100.0
	}
	if pan < -100.0 {
		pan = -100.0
	}

	return pan / 100.0
}

func (mjc *MockJackClient) calculatePitchRatio(region *SfzSection, midiNote uint8) float64 {
	pitchKeycenter := region.GetIntOpcode("pitch_keycenter", int(midiNote))
	semitones := int(midiNote) - pitchKeycenter
	return math.Pow(2.0, float64(semitones)/12.0)
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
