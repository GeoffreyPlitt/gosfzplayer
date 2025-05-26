//go:build jack
// +build jack

package gosfzplayer

import (
	"testing"
)

func TestPitchRatioCalculation(t *testing.T) {
	// Create a test SFZ player
	player, err := NewSfzPlayer("testdata/test.sfz", "test-pitch")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}
	defer player.StopAndClose()

	// Skip if JACK client not available
	if player.jackClient == nil {
		t.Skip("JACK client not available, skipping pitch ratio test")
	}

	jc := player.jackClient

	// Create a test region with pitch_keycenter
	region := &SfzSection{
		Type: "region",
		Opcodes: map[string]string{
			"pitch_keycenter": "60", // Middle C
		},
	}

	// Test that pitch ratio calculation doesn't crash
	ratio := jc.calculatePitchRatio(region, 60)
	if ratio <= 0 {
		t.Errorf("Expected positive pitch ratio, got %f", ratio)
	}

	// Test with different notes
	ratio = jc.calculatePitchRatio(region, 72) // Octave up
	if ratio <= 0 {
		t.Errorf("Expected positive pitch ratio for octave up, got %f", ratio)
	}

	ratio = jc.calculatePitchRatio(region, 48) // Octave down
	if ratio <= 0 {
		t.Errorf("Expected positive pitch ratio for octave down, got %f", ratio)
	}
}

func TestVoiceWithPitchShift(t *testing.T) {
	// Test that voices are created with pitch ratios and don't crash
	player, err := NewSfzPlayer("testdata/test.sfz", "test-voice-pitch")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}
	defer player.StopAndClose()

	// Skip if JACK client not available
	if player.jackClient == nil {
		t.Skip("JACK client not available, skipping voice pitch test")
	}

	jc := player.jackClient

	// Simulate a note on event
	jc.noteOn(60, 100) // Middle C, velocity 100

	// Check that voices have valid pitch ratios
	jc.mu.RLock()
	for i, voice := range jc.activeVoices {
		if voice.pitchRatio <= 0 {
			t.Errorf("Voice %d has invalid pitch ratio: %f", i, voice.pitchRatio)
		}
		if voice.position < 0 {
			t.Errorf("Voice %d has invalid position: %f", i, voice.position)
		}
	}
	jc.mu.RUnlock()
}
