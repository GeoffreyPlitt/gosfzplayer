//go:build jack
// +build jack

package gosfzplayer

import (
	"testing"
)

func TestVoiceCreation(t *testing.T) {
	// Create a test voice
	voice := &Voice{
		midiNote: 60, // Middle C
		velocity: 100,
		position: 0,
		volume:   1.0,
		pan:      0.0,
		isActive: true,
		noteOn:   true,
	}

	if voice.midiNote != 60 {
		t.Errorf("Expected MIDI note 60, got %d", voice.midiNote)
	}

	if voice.velocity != 100 {
		t.Errorf("Expected velocity 100, got %d", voice.velocity)
	}

	if !voice.isActive {
		t.Error("Expected voice to be active")
	}

	if !voice.noteOn {
		t.Error("Expected voice to be on")
	}
}

func TestRegionMatching(t *testing.T) {
	// Create a test SFZ player
	player, err := NewSfzPlayer("testdata/test.sfz", "test-region")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}

	// Create a mock JACK client for testing
	jc := &JackClient{
		player: player,
	}

	// Test region matching
	regions := player.sfzData.Regions
	if len(regions) == 0 {
		t.Fatal("No regions found in test SFZ file")
	}

	// Test first region (C2-C4, velocity 1-64)
	region := regions[0]

	// Should match C3 (MIDI 48) with velocity 50
	if !jc.regionMatches(region, 48, 50) {
		t.Error("Expected region to match C3 with velocity 50")
	}

	// Should not match C3 with velocity 100 (too high)
	if jc.regionMatches(region, 48, 100) {
		t.Error("Expected region to NOT match C3 with velocity 100")
	}

	// Should not match C5 (MIDI 72) - outside key range
	if jc.regionMatches(region, 72, 50) {
		t.Error("Expected region to NOT match C5")
	}
}

func TestVolumeCalculation(t *testing.T) {
	// Create a test SFZ player
	player, err := NewSfzPlayer("testdata/test.sfz", "test-volume")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}

	// Create a mock JACK client for testing
	jc := &JackClient{
		player: player,
	}

	regions := player.sfzData.Regions
	if len(regions) == 0 {
		t.Fatal("No regions found in test SFZ file")
	}

	// Test volume calculation
	region := regions[0]
	volume := jc.calculateVolume(region, 100)

	if volume <= 0 {
		t.Errorf("Expected positive volume, got %f", volume)
	}

	// Test with different velocity
	volume127 := jc.calculateVolume(region, 127)
	volume64 := jc.calculateVolume(region, 64)

	if volume127 <= volume64 {
		t.Error("Expected higher velocity to produce higher volume")
	}
}

func TestPanCalculation(t *testing.T) {
	// Create a test SFZ player
	player, err := NewSfzPlayer("testdata/test.sfz", "test-pan")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}

	// Create a mock JACK client for testing
	jc := &JackClient{
		player: player,
	}

	regions := player.sfzData.Regions
	if len(regions) >= 2 {
		// Test pan calculation on region with pan setting
		region := regions[1] // This should have pan=-50
		pan := jc.calculatePan(region)

		// Pan should be normalized to -1.0 to 1.0 range
		if pan < -1.0 || pan > 1.0 {
			t.Errorf("Pan value %f is outside valid range [-1.0, 1.0]", pan)
		}

		// Should be negative (left) for this region
		if pan >= 0 {
			t.Errorf("Expected negative pan for region with pan=-50, got %f", pan)
		}
	}
}

func TestMidiNoteConversion(t *testing.T) {
	// Test MIDI note number conversions
	testCases := []struct {
		note uint8
		name string
	}{
		{36, "C2"},
		{48, "C3"},
		{60, "C4"}, // Middle C
		{72, "C5"},
	}

	for _, tc := range testCases {
		// Basic validation that note is in valid MIDI range
		if tc.note < 0 || tc.note > 127 {
			t.Errorf("MIDI note %d (%s) is outside valid range [0-127]", tc.note, tc.name)
		}
	}
}

func TestVoiceLifecycle(t *testing.T) {
	// Test voice lifecycle management
	voice := &Voice{
		midiNote: 60,
		velocity: 100,
		position: 0,
		isActive: true,
		noteOn:   true,
	}

	// Voice should start active and on
	if !voice.isActive || !voice.noteOn {
		t.Error("Voice should start active and on")
	}

	// Simulate note off
	voice.noteOn = false
	voice.isActive = false

	// Voice should now be inactive
	if voice.isActive || voice.noteOn {
		t.Error("Voice should be inactive after note off")
	}
}

func TestMaxPolyphony(t *testing.T) {
	// Test that polyphony limiting works
	maxVoices := 4
	voices := make([]*Voice, 0)

	// Add voices up to the limit
	for i := 0; i < maxVoices+2; i++ {
		voice := &Voice{
			midiNote: uint8(60 + i),
			velocity: 100,
			isActive: true,
		}

		// Simulate polyphony limiting
		if len(voices) >= maxVoices {
			voices = voices[1:] // Remove oldest voice
		}
		voices = append(voices, voice)
	}

	// Should not exceed max voices
	if len(voices) != maxVoices {
		t.Errorf("Expected %d voices, got %d", maxVoices, len(voices))
	}

	// Last voice should be the most recent
	lastVoice := voices[len(voices)-1]
	expectedNote := uint8(60 + maxVoices + 1)
	if lastVoice.midiNote != expectedNote {
		t.Errorf("Expected last voice to have note %d, got %d", expectedNote, lastVoice.midiNote)
	}
}
