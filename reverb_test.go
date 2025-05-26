package gosfzplayer

import (
	"testing"
)

func TestFreeverb(t *testing.T) {
	// Create a Freeverb instance
	reverb := NewFreeverb(44100)

	// Set some test parameters
	reverb.SetRoomSize(0.5)
	reverb.SetDamping(0.3)
	reverb.SetWet(0.8)
	reverb.SetDry(0.2)
	reverb.SetWidth(1.0)

	// Test parameter retrieval
	if reverb.GetRoomSize() != 0.5 {
		t.Errorf("Expected room size 0.5, got %.2f", reverb.GetRoomSize())
	}
	if reverb.GetDamping() != 0.3 {
		t.Errorf("Expected damping 0.3, got %.2f", reverb.GetDamping())
	}

	// Test basic audio processing (reverb initialization)
	input := 0.5
	output := reverb.ProcessMono(input)

	// Just verify the function doesn't crash and returns a reasonable value
	if output < -2.0 || output > 2.0 {
		t.Errorf("Reverb output out of reasonable range: %.3f", output)
	}

	t.Logf("Reverb basic functionality test passed")
}

func TestReverbSfzPlayerIntegration(t *testing.T) {
	// Skip if piano samples not available
	if !fileExists("testdata/piano.sfz") {
		t.Skip("piano.sfz not found, skipping reverb integration test")
	}

	// Create SFZ player
	player, err := NewSfzPlayer("testdata/piano.sfz", "")
	if err != nil {
		t.Fatalf("Failed to create piano SFZ player: %v", err)
	}
	defer player.StopAndClose()

	// Test reverb parameter setting
	player.SetReverbSend(0.5)
	player.SetReverbRoomSize(0.7)
	player.SetReverbDamping(0.4)

	// Verify parameters were set
	if player.GetReverbSend() != 0.5 {
		t.Errorf("Expected reverb send 0.5, got %.2f", player.GetReverbSend())
	}
	if player.GetReverbRoomSize() != 0.7 {
		t.Errorf("Expected room size 0.7, got %.2f", player.GetReverbRoomSize())
	}
	if player.GetReverbDamping() != 0.4 {
		t.Errorf("Expected damping 0.4, got %.2f", player.GetReverbDamping())
	}

	t.Logf("Reverb SFZ integration test passed")
}

func TestReverbParameterBounds(t *testing.T) {
	reverb := NewFreeverb(44100)

	// Test parameter bounds
	reverb.SetRoomSize(-0.5) // Should clamp to 0.0
	if reverb.GetRoomSize() != 0.0 {
		t.Errorf("Room size should be clamped to 0.0, got %.2f", reverb.GetRoomSize())
	}

	reverb.SetRoomSize(1.5) // Should clamp to 1.0
	if reverb.GetRoomSize() != 1.0 {
		t.Errorf("Room size should be clamped to 1.0, got %.2f", reverb.GetRoomSize())
	}

	reverb.SetDamping(-0.1) // Should clamp to 0.0
	if reverb.GetDamping() != 0.0 {
		t.Errorf("Damping should be clamped to 0.0, got %.2f", reverb.GetDamping())
	}

	reverb.SetDamping(1.1) // Should clamp to 1.0
	if reverb.GetDamping() != 1.0 {
		t.Errorf("Damping should be clamped to 1.0, got %.2f", reverb.GetDamping())
	}

	t.Logf("Reverb parameter bounds test passed")
}

// Helper function to check if file exists
func fileExists(filename string) bool {
	_, err := NewSfzPlayer(filename, "")
	return err == nil
}
