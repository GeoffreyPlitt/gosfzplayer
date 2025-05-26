package gosfzplayer

import (
	"os"
	"testing"
)

func TestLoopInitialization(t *testing.T) {
	// Create test sample data
	sampleData := make([]float64, 1000)
	for i := range sampleData {
		sampleData[i] = float64(i) / 1000.0 // Linear ramp
	}

	sample := &Sample{
		Data:     sampleData,
		Channels: 1,
	}

	// Test different loop modes
	testCases := []struct {
		name          string
		loopMode      string
		loopStart     string
		loopEnd       string
		expectedMode  string
		expectedStart float64
		expectedEnd   float64
	}{
		{
			name:          "no_loop mode",
			loopMode:      "no_loop",
			expectedMode:  "no_loop",
			expectedStart: 0,
			expectedEnd:   999, // Sample length - 1
		},
		{
			name:          "loop_continuous with explicit points",
			loopMode:      "loop_continuous",
			loopStart:     "100",
			loopEnd:       "500",
			expectedMode:  "loop_continuous",
			expectedStart: 100,
			expectedEnd:   500,
		},
		{
			name:          "loop_sustain mode",
			loopMode:      "loop_sustain",
			loopStart:     "200",
			loopEnd:       "800",
			expectedMode:  "loop_sustain",
			expectedStart: 200,
			expectedEnd:   800,
		},
		{
			name:          "one_shot mode",
			loopMode:      "one_shot",
			expectedMode:  "one_shot",
			expectedStart: 0,
			expectedEnd:   999,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create region with loop opcodes
			opcodes := map[string]string{
				"loop_mode": tc.loopMode,
			}
			if tc.loopStart != "" {
				opcodes["loop_start"] = tc.loopStart
			}
			if tc.loopEnd != "" {
				opcodes["loop_end"] = tc.loopEnd
			}

			region := &SfzSection{
				Type:    "region",
				Opcodes: opcodes,
			}

			// Create voice and initialize loop
			voice := &Voice{
				sample: sample,
				region: region,
			}

			voice.InitializeLoop()

			// Verify loop parameters
			if voice.loopMode != tc.expectedMode {
				t.Errorf("Expected loop mode %s, got %s", tc.expectedMode, voice.loopMode)
			}

			if voice.loopStart != tc.expectedStart {
				t.Errorf("Expected loop start %f, got %f", tc.expectedStart, voice.loopStart)
			}

			if voice.loopEnd != tc.expectedEnd {
				t.Errorf("Expected loop end %f, got %f", tc.expectedEnd, voice.loopEnd)
			}
		})
	}
}

func TestLoopProcessing(t *testing.T) {
	// Create test sample
	sampleData := make([]float64, 100)
	sample := &Sample{
		Data:     sampleData,
		Channels: 1,
	}

	// Test no_loop mode
	t.Run("no_loop stops at end", func(t *testing.T) {
		region := &SfzSection{
			Type:    "region",
			Opcodes: map[string]string{"loop_mode": "no_loop"},
		}

		voice := &Voice{
			sample:   sample,
			region:   region,
			position: 98, // Before end
			noteOn:   true,
		}

		voice.InitializeLoop()

		// Should continue at position 98
		if !voice.ProcessLoop() {
			t.Error("Expected voice to continue at position 98")
		}

		// Move to boundary (sampleLength-1 = 99)
		voice.position = 99
		if voice.ProcessLoop() {
			t.Error("Expected voice to stop at end of sample (position 99)")
		}
	})

	// Test loop_continuous mode
	t.Run("loop_continuous loops back", func(t *testing.T) {
		region := &SfzSection{
			Type: "region",
			Opcodes: map[string]string{
				"loop_mode":  "loop_continuous",
				"loop_start": "20",
				"loop_end":   "80",
			},
		}

		voice := &Voice{
			sample:   sample,
			region:   region,
			position: 75, // Before loop end
			noteOn:   true,
		}

		voice.InitializeLoop()

		// Should continue before loop end
		if !voice.ProcessLoop() {
			t.Error("Expected voice to continue before loop end")
		}

		// Move to loop end
		voice.position = 80
		if !voice.ProcessLoop() {
			t.Error("Expected voice to continue and loop back")
		}

		// Should have looped back to start
		if voice.position < 20 || voice.position > 30 {
			t.Errorf("Expected position to be near loop start (20), got %f", voice.position)
		}
	})

	// Test loop_sustain mode
	t.Run("loop_sustain behavior", func(t *testing.T) {
		region := &SfzSection{
			Type: "region",
			Opcodes: map[string]string{
				"loop_mode":  "loop_sustain",
				"loop_start": "10",
				"loop_end":   "50",
			},
		}

		voice := &Voice{
			sample:   sample,
			region:   region,
			position: 45,
			noteOn:   true, // Note held
		}

		voice.InitializeLoop()

		// Should loop while note is held
		voice.position = 50
		if !voice.ProcessLoop() {
			t.Error("Expected voice to continue and loop while note held")
		}

		// Should have looped back
		if voice.position < 10 || voice.position > 20 {
			t.Errorf("Expected position to be near loop start (10), got %f", voice.position)
		}

		// Release note
		voice.noteOn = false
		voice.position = 98 // Before end of sample

		// Should continue until end without looping
		if !voice.ProcessLoop() {
			t.Error("Expected voice to continue after note release")
		}

		// Move to boundary - should stop
		voice.position = 99
		if voice.ProcessLoop() {
			t.Error("Expected voice to stop at end after note release")
		}
	})
}

func TestLoopEdgeCases(t *testing.T) {
	sampleData := make([]float64, 100)
	sample := &Sample{
		Data:     sampleData,
		Channels: 1,
	}

	// Test invalid loop points
	t.Run("invalid loop points", func(t *testing.T) {
		region := &SfzSection{
			Type: "region",
			Opcodes: map[string]string{
				"loop_mode":  "loop_continuous",
				"loop_start": "80", // Start after end
				"loop_end":   "20", // End before start
			},
		}

		voice := &Voice{
			sample: sample,
			region: region,
		}

		voice.InitializeLoop()

		// Should fallback to full sample
		if voice.loopStart != 0 || voice.loopEnd != 99 {
			t.Errorf("Expected fallback to full sample loop (0-99), got %f-%f", voice.loopStart, voice.loopEnd)
		}
	})

	// Test unknown loop mode
	t.Run("unknown loop mode", func(t *testing.T) {
		region := &SfzSection{
			Type: "region",
			Opcodes: map[string]string{
				"loop_mode": "unknown_mode",
			},
		}

		voice := &Voice{
			sample:   sample,
			region:   region,
			position: 98,
		}

		voice.InitializeLoop()

		// Should treat as no_loop
		voice.position = 99
		if voice.ProcessLoop() {
			t.Error("Expected unknown mode to behave like no_loop and stop at end")
		}
	})
}

func TestLoopAudioDemo(t *testing.T) {
	// Skip if piano samples not available
	if _, err := os.Stat("testdata/piano.sfz"); os.IsNotExist(err) {
		t.Skip("piano.sfz not found, run 'go generate' to download piano samples")
	}

	// Create a simple SFZ content with loop settings
	sfzContent := `<global>
ampeg_attack=0.01
ampeg_decay=0.1
ampeg_sustain=80
ampeg_release=0.5

<region>
sample=samples/C4vL.flac
key=60
loop_mode=loop_continuous
loop_start=5000
loop_end=15000
`

	// Create SFZ file in testdata directory
	sfzPath := "testdata/loop_demo.sfz"
	err := os.WriteFile(sfzPath, []byte(sfzContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create SFZ file: %v", err)
	}
	defer os.Remove(sfzPath)

	// Create SFZ player
	player, err := NewSfzPlayer(sfzPath, "")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}
	defer player.StopAndClose()

	// Create mock client for rendering
	mockClient := createTestMockClient(player, 44100, 512)

	// Render a sustained note to demonstrate looping
	sampleRate := 44100
	duration := 3.0 // 3 seconds to hear the loop
	totalSamples := int(float64(sampleRate) * duration)
	outputBuffer := make([]float32, totalSamples)

	// Trigger note and hold it
	mockClient.noteOn(60, 100) // C4, velocity 100

	// Render audio
	bufferSize := 512
	currentSample := 0

	for currentSample < totalSamples {
		framesToRender := bufferSize
		if currentSample+bufferSize > totalSamples {
			framesToRender = totalSamples - currentSample
		}

		audioBuffer := make([]float32, framesToRender)
		mockClient.renderVoices(audioBuffer, uint32(framesToRender))

		copy(outputBuffer[currentSample:currentSample+framesToRender], audioBuffer)
		currentSample += framesToRender
	}

	// Release note at the end
	mockClient.noteOff(60)

	// Save as WAV file
	outputPath := "testdata/loop_demo.wav"
	err = saveWAV(outputPath, outputBuffer, sampleRate)
	if err != nil {
		t.Fatalf("Failed to save WAV file: %v", err)
	}

	t.Logf("Generated loop demo: %s (%.1f seconds)", outputPath, duration)
}
