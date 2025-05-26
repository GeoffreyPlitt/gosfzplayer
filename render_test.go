//go:build !jack
// +build !jack

package gosfzplayer

import (
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

	// Configure reverb for a nice hall sound
	player.SetReverbSend(0.3)        // 30% reverb send
	player.SetReverbRoomSize(0.7)    // Large room
	player.SetReverbDamping(0.4)     // Moderate damping
	player.SetReverbWet(0.8)         // High wet level
	player.SetReverbDry(0.6)         // Moderate dry level  
	player.SetReverbWidth(1.0)       // Full stereo width

	t.Logf("Reverb configured: Send=%.1f%%, Room=%.1f%%, Damping=%.1f%%, Wet=%.1f%%, Dry=%.1f%%",
		player.GetReverbSend()*100, player.GetReverbRoomSize()*100, 
		player.GetReverbDamping()*100, player.GetReverbWet()*100, player.GetReverbDry()*100)

	// Create a mock JACK client for offline rendering
	mockClient := createTestMockClient(player, 44100, 512)

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
