//go:build !jack
// +build !jack

package gosfzplayer

import (
	"os"
	"testing"
)

func TestNote2DebugAnalysis(t *testing.T) {
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

	// Test Note 2 (E4, MIDI 64) in isolation
	t.Run("Note2_Isolation", func(t *testing.T) {
		mockClient.activeVoices = make([]*Voice, 0)
		
		t.Log("Testing Note 2 (E4, MIDI 64) in isolation:")
		
		// Try to create voice
		mockClient.noteOn(64, 100)
		
		if len(mockClient.activeVoices) == 0 {
			t.Fatal("❌ NO VOICE CREATED for Note 2")
		}
		
		voice := mockClient.activeVoices[0]
		t.Logf("✅ Voice created for Note 2:")
		t.Logf("   Sample: %s", voice.sample.FilePath)
		t.Logf("   Pitch ratio: %.3f", voice.pitchRatio)
		t.Logf("   Volume: %.3f", voice.volume)
		t.Logf("   Sample length: %d samples", len(voice.sample.Data))
		t.Logf("   Sample channels: %d", voice.sample.Channels)
		
		// Check envelope settings
		t.Logf("   ADSR Envelope:")
		t.Logf("     Attack samples: %.1f (%.3fs)", voice.attackSamples, voice.attackSamples/44100.0)
		t.Logf("     Decay samples: %.1f (%.3fs)", voice.decaySamples, voice.decaySamples/44100.0)
		t.Logf("     Sustain level: %.3f", voice.sustainLevel)
		t.Logf("     Release samples: %.1f (%.3fs)", voice.releaseSamples, voice.releaseSamples/44100.0)
		t.Logf("     Initial state: %v", voice.envelopeState)
		
		// Check loop settings
		t.Logf("   Loop settings:")
		t.Logf("     Loop mode: %s", voice.loopMode)
		t.Logf("     Loop start: %.1f", voice.loopStart)
		t.Logf("     Loop end: %.1f", voice.loopEnd)
	})

	// Compare with working notes
	t.Run("Compare_With_Working_Notes", func(t *testing.T) {
		workingNotes := []struct {
			midi uint8
			name string
		}{
			{60, "C4"},
			{64, "E4"}, // Our problem note
			{67, "G4"},
		}
		
		for _, note := range workingNotes {
			mockClient.activeVoices = make([]*Voice, 0)
			mockClient.noteOn(note.midi, 100)
			
			if len(mockClient.activeVoices) == 0 {
				t.Errorf("❌ No voice for %s (MIDI %d)", note.name, note.midi)
				continue
			}
			
			voice := mockClient.activeVoices[0]
			t.Logf("%s (MIDI %d): attack=%.1f, decay=%.1f, sustain=%.3f, release=%.1f, loop=%s", 
				note.name, note.midi,
				voice.attackSamples, voice.decaySamples, voice.sustainLevel, voice.releaseSamples,
				voice.loopMode)
		}
	})
}

func TestNote2VoiceLifecycle(t *testing.T) {
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

	// Test Note 2 voice lifecycle
	t.Log("Testing Note 2 voice lifecycle:")
	
	// Create voice
	mockClient.noteOn(64, 100)
	if len(mockClient.activeVoices) == 0 {
		t.Fatal("❌ No voice created")
	}
	
	voice := mockClient.activeVoices[0]
	t.Logf("✅ Voice created, initial state: active=%v, noteOn=%v, envState=%v", 
		voice.isActive, voice.noteOn, voice.envelopeState)
	
	// Simulate rendering frames to see if voice dies early
	testFrames := []uint32{512, 512, 512, 512} // 4 buffers = ~46ms at 44.1kHz
	
	for i, frames := range testFrames {
		output := make([]float32, frames)
		mockClient.renderVoices(output, frames)
		
		if len(mockClient.activeVoices) == 0 {
			t.Fatalf("❌ Voice died after buffer %d (%.1fms)", i+1, float64(i+1)*float64(frames)/44.1)
		}
		
		voice = mockClient.activeVoices[0]
		envelopeLevel := voice.ProcessEnvelope()
		
		t.Logf("Buffer %d: active=%v, noteOn=%v, envState=%v, envLevel=%.3f, position=%.1f", 
			i+1, voice.isActive, voice.noteOn, voice.envelopeState, envelopeLevel, voice.position)
		
		// Check for silence in output
		maxSample := float32(0.0)
		for _, sample := range output {
			if abs32(sample) > maxSample {
				maxSample = abs32(sample)
			}
		}
		t.Logf("   Output max amplitude: %.6f", maxSample)
		
		if maxSample < 0.000001 {
			t.Logf("   ⚠️  Buffer %d is essentially silent!", i+1)
		}
	}
}

func TestSequentialNoteIssues(t *testing.T) {
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

	// Test notes in sequence like the arpeggio
	t.Log("Testing first 3 arpeggio notes in sequence:")
	arpeggioNotes := []uint8{60, 64, 67} // C4, E4, G4
	noteNames := []string{"C4", "E4", "G4"}
	
	for i, note := range arpeggioNotes {
		t.Logf("\n--- Playing %s (MIDI %d) ---", noteNames[i], note)
		
		// Trigger note
		mockClient.noteOn(note, 100)
		activeCount := len(mockClient.activeVoices)
		t.Logf("Active voices after note on: %d", activeCount)
		
		// Render a few frames to see what happens
		for frame := 0; frame < 3; frame++ {
			output := make([]float32, 512)
			mockClient.renderVoices(output, 512)
			
			stillActiveCount := len(mockClient.activeVoices)
			
			// Check for audio output
			maxSample := float32(0.0)
			for _, sample := range output {
				if abs32(sample) > maxSample {
					maxSample = abs32(sample)
				}
			}
			
			t.Logf("  Frame %d: voices=%d, max_amplitude=%.6f", 
				frame+1, stillActiveCount, maxSample)
			
			if maxSample < 0.000001 {
				t.Logf("    ⚠️  Frame %d is silent!", frame+1)
			}
		}
		
		// Release note after 200ms 
		mockClient.noteOff(note)
		t.Logf("Note off sent for %s", noteNames[i])
	}
}

func TestExactArpeggioTiming(t *testing.T) {
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

	// Arpeggio notes: C4-E4-G4-C5-E5-G5-C6
	arpeggioNotes := []uint8{60, 64, 67, 72, 76, 79, 84}
	noteNames := []string{"C4", "E4", "G4", "C5", "E5", "G5", "C6"}

	// Timing (quarter notes with 1 second duration)
	totalDuration := float64(8.0) // 8 seconds total
	sampleRate := uint32(44100)
	bufferSize := uint32(512)
	totalSamples := int(totalDuration * float64(sampleRate))

	t.Log("Testing exact arpeggio timing reproduction:")
	
	// Track which notes have been triggered/released to avoid timing window misses
	noteTriggered := make([]bool, len(arpeggioNotes))
	noteReleased := make([]bool, len(arpeggioNotes))
	
	currentSample := 0
	for currentSample < totalSamples {
		currentTime := float64(currentSample) / float64(sampleRate)
		
		// Check each note for triggering
		for i, note := range arpeggioNotes {
			noteStartTime := float64(i) * 1.0 // i-th second
			noteEndTime := noteStartTime + 0.8 // 800ms note duration

			// Check if we should trigger this note (trigger once when time is reached)
			if !noteTriggered[i] && currentTime >= noteStartTime {
				t.Logf("%.3fs: Triggering %s (MIDI %d)", currentTime, noteNames[i], note)
				mockClient.noteOn(note, 100)
				noteTriggered[i] = true
				t.Logf("  Active voices after note on: %d", len(mockClient.activeVoices))
			}

			// Check if we should release this note (release once when time is reached)
			if !noteReleased[i] && noteTriggered[i] && currentTime >= noteEndTime {
				t.Logf("%.3fs: Releasing %s (MIDI %d)", currentTime, noteNames[i], note)
				mockClient.noteOff(note)
				noteReleased[i] = true
			}
		}

		// Render this buffer
		framesToRender := bufferSize
		if currentSample+int(bufferSize) > totalSamples {
			framesToRender = uint32(totalSamples - currentSample)
		}

		audioBuffer := make([]float32, framesToRender)
		mockClient.renderVoices(audioBuffer, framesToRender)

		// Check for silence in critical time windows
		if currentTime >= 1.0 && currentTime < 1.2 { // Note 2 (E4) should be playing
			maxSample := float32(0.0)
			for _, sample := range audioBuffer {
				if abs32(sample) > maxSample {
					maxSample = abs32(sample)
				}
			}
			
			if maxSample < 0.000001 {
				t.Errorf("⚠️  Note 2 (E4) is SILENT at time %.3fs (voices=%d)", 
					currentTime, len(mockClient.activeVoices))
				
				// Debug voice states
				for j, voice := range mockClient.activeVoices {
					t.Errorf("  Voice %d: note=%d, active=%v, noteOn=%v, envState=%v, envLevel=%.3f, pos=%.1f", 
						j, voice.midiNote, voice.isActive, voice.noteOn, voice.envelopeState, 
						voice.envelopeLevel, voice.position)
				}
			} else {
				t.Logf("✅ Note 2 (E4) amplitude: %.6f at time %.3fs", maxSample, currentTime)
			}
		}

		currentSample += int(framesToRender)
	}
}

// Helper function
func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}