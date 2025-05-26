//go:build !jack
// +build !jack

package gosfzplayer

import (
	"os"
	"testing"
)

func TestIndividualNoteRendering(t *testing.T) {
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

	// Test notes: all should work due to pitch-shifting capabilities
	testNotes := []struct {
		midiNote uint8
		name     string
		expected string // "should work" or "should fail"
	}{
		{57, "A3 (lowest sample)", "should work"},    // a1.wav mapped to MIDI 57
		{60, "C4 (middle C)", "should work"},         // c1.wav mapped to MIDI 60
		{64, "E4", "should work"},                    // e1.wav mapped to MIDI 64
		{67, "G4", "should work"},                    // g1.wav mapped to MIDI 67
		{69, "A4 (highest sample)", "should work"},   // c2.wav mapped to MIDI 69
		{72, "C5 (now works with pitch-shift)", "should work"}, // Works via pitch-shifting
		{76, "E5 (now works with pitch-shift)", "should work"}, // Works via pitch-shifting
		{79, "G5 (now works with pitch-shift)", "should work"}, // Works via pitch-shifting
		{84, "C6 (now works with pitch-shift)", "should work"}, // Works via pitch-shifting
	}

	for _, test := range testNotes {
		t.Run(test.name, func(t *testing.T) {
			// Clear any existing voices
			mockClient.activeVoices = make([]*Voice, 0)

			// Try to trigger the note
			mockClient.noteOn(test.midiNote, 100)

			// Check if any voices were created
			voiceCount := len(mockClient.activeVoices)

			if test.expected == "should work" {
				if voiceCount == 0 {
					t.Errorf("MIDI %d (%s) %s but no voice was created", 
						test.midiNote, test.name, test.expected)
				} else {
					t.Logf("✅ MIDI %d (%s) correctly created %d voice(s)", 
						test.midiNote, test.name, voiceCount)
				}
			} else { // should fail
				if voiceCount > 0 {
					t.Errorf("MIDI %d (%s) %s but %d voice(s) were created", 
						test.midiNote, test.name, test.expected, voiceCount)
				} else {
					t.Logf("✅ MIDI %d (%s) correctly failed to create voices", 
						test.midiNote, test.name)
				}
			}

			// If voice was created, verify it has valid properties
			if voiceCount > 0 {
				voice := mockClient.activeVoices[0]
				if voice.sample == nil {
					t.Errorf("Voice for MIDI %d has nil sample", test.midiNote)
				}
				if voice.pitchRatio <= 0 {
					t.Errorf("Voice for MIDI %d has invalid pitch ratio: %f", 
						test.midiNote, voice.pitchRatio)
				}
				t.Logf("   Voice details: sample=%s, pitchRatio=%.3f, volume=%.3f", 
					voice.sample.FilePath, voice.pitchRatio, voice.volume)
			}
		})
	}
}

func TestArpeggioNotesByNote(t *testing.T) {
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

	// Original arpeggio notes: C-E-G-C-E-G-C
	arpeggioNotes := []uint8{60, 64, 67, 72, 76, 79, 84}
	noteNames := []string{"C4", "E4", "G4", "C5", "E5", "G5", "C6"}

	t.Log("Testing original arpeggio notes individually:")
	
	workingNotes := []uint8{}
	failingNotes := []uint8{}

	for i, note := range arpeggioNotes {
		// Clear voices
		mockClient.activeVoices = make([]*Voice, 0)
		
		// Try to create voice for this note
		mockClient.noteOn(note, 100)
		
		voiceCount := len(mockClient.activeVoices)
		if voiceCount > 0 {
			t.Logf("✅ %s (MIDI %d): Voice created successfully", noteNames[i], note)
			workingNotes = append(workingNotes, note)
		} else {
			t.Logf("❌ %s (MIDI %d): NO VOICE CREATED", noteNames[i], note)
			failingNotes = append(failingNotes, note)
		}
	}

	t.Logf("\nSUMMARY:")
	t.Logf("Working notes (%d): %v", len(workingNotes), workingNotes)
	t.Logf("Failing notes (%d): %v", len(failingNotes), failingNotes)
	
	if len(failingNotes) > 0 {
		t.Logf("\nSome notes still failing: %d out of %d arpeggio notes cannot be played", 
			len(failingNotes), len(arpeggioNotes))
	} else {
		t.Logf("\n✅ SUCCESS: All %d arpeggio notes can now be played thanks to pitch-shifting!", len(arpeggioNotes))
	}
}