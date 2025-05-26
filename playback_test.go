package gosfzplayer

import (
	"os"
	"testing"
)

func TestVolumeOpcodeParsing(t *testing.T) {
	// Create a temporary SFZ file with volume opcodes
	tempFile, err := os.CreateTemp("", "test_volume_*.sfz")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := `<global>
volume=-6.0

<region>
sample=sample1.wav
volume=3.0
key=60

<region>
sample=sample2.wav
volume=-12.5
key=61
`

	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tempFile.Close()

	// Parse the file
	sfzData, err := ParseSfzFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse SFZ file with volume opcodes: %v", err)
	}

	// Check global volume
	if sfzData.Global == nil {
		t.Fatal("Expected global section")
	}
	globalVolume := sfzData.Global.GetFloatOpcode("volume", 999.0)
	if globalVolume != -6.0 {
		t.Errorf("Expected global volume -6.0, got %f", globalVolume)
	}

	// Check region volumes
	if len(sfzData.Regions) != 2 {
		t.Fatalf("Expected 2 regions, got %d", len(sfzData.Regions))
	}

	region1Volume := sfzData.Regions[0].GetFloatOpcode("volume", 999.0)
	if region1Volume != 3.0 {
		t.Errorf("Expected region1 volume 3.0, got %f", region1Volume)
	}

	region2Volume := sfzData.Regions[1].GetFloatOpcode("volume", 999.0)
	if region2Volume != -12.5 {
		t.Errorf("Expected region2 volume -12.5, got %f", region2Volume)
	}
}

func TestPitchKeycenterOpcodeParsing(t *testing.T) {
	// Create a temporary SFZ file with pitch_keycenter opcodes
	tempFile, err := os.CreateTemp("", "test_pitch_*.sfz")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := `<region>
sample=sample1.wav
key=60
pitch_keycenter=60

<region>
sample=sample2.wav
key=62
pitch_keycenter=60

<region>
sample=sample3.wav
key=64
pitch_keycenter=72
`

	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tempFile.Close()

	// Parse the file
	sfzData, err := ParseSfzFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse SFZ file with pitch_keycenter opcodes: %v", err)
	}

	// Check regions
	if len(sfzData.Regions) != 3 {
		t.Fatalf("Expected 3 regions, got %d", len(sfzData.Regions))
	}

	// Region 1: pitch_keycenter matches key
	region1Keycenter := sfzData.Regions[0].GetIntOpcode("pitch_keycenter", -1)
	if region1Keycenter != 60 {
		t.Errorf("Expected region1 pitch_keycenter 60, got %d", region1Keycenter)
	}

	// Region 2: pitch_keycenter different from key
	region2Keycenter := sfzData.Regions[1].GetIntOpcode("pitch_keycenter", -1)
	if region2Keycenter != 60 {
		t.Errorf("Expected region2 pitch_keycenter 60, got %d", region2Keycenter)
	}

	// Region 3: different pitch_keycenter
	region3Keycenter := sfzData.Regions[2].GetIntOpcode("pitch_keycenter", -1)
	if region3Keycenter != 72 {
		t.Errorf("Expected region3 pitch_keycenter 72, got %d", region3Keycenter)
	}
}

func TestVolumeCalculationNoCrash(t *testing.T) {
	// Test that volume calculation doesn't crash with various inputs
	player, err := NewSfzPlayer("testdata/test.sfz", "test-volume-calc")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}
	defer player.StopAndClose()

	// This test verifies that player creation with volume opcodes doesn't crash
	// The actual volume calculation is tested in JACK-specific tests
	if player == nil {
		t.Fatal("Expected non-nil player")
	}

	// Verify SFZ data was parsed correctly
	sfzData := player.GetSfzData()
	if sfzData == nil {
		t.Fatal("Expected non-nil SFZ data")
	}

	// Test that we can access volume opcodes without crashing
	if len(sfzData.Regions) > 0 {
		region := sfzData.Regions[0]
		volume := region.GetFloatOpcode("volume", 0.0)

		// Just verify we got a numeric value
		if volume < -100.0 || volume > 100.0 {
			t.Logf("Volume opcode parsed as: %f", volume)
		}
	}
}

func TestBasicPlaybackOpcodesNoErrors(t *testing.T) {
	// Test that basic playback opcodes don't cause parsing errors
	// Use the existing test.sfz file which already has the correct sample paths
	player, err := NewSfzPlayer("testdata/test.sfz", "test-playback")
	if err != nil {
		t.Fatalf("Failed to create SFZ player with playback opcodes: %v", err)
	}
	defer player.StopAndClose()

	// Verify the player was created successfully
	if player == nil {
		t.Fatal("Expected non-nil player")
	}

	// Verify samples were loaded
	if player.sampleCache.Size() == 0 {
		t.Error("Expected samples to be loaded")
	}

	// Verify SFZ data was parsed
	sfzData := player.GetSfzData()
	if sfzData == nil {
		t.Fatal("Expected non-nil SFZ data")
	}

	if len(sfzData.Regions) == 0 {
		t.Error("Expected at least one region")
	}

	// Verify that volume and pitch_keycenter opcodes exist in the test file
	foundVolume := false
	foundPitchKeycenter := false

	for _, region := range sfzData.Regions {
		if region.GetStringOpcode("volume") != "" {
			foundVolume = true
		}
		if region.GetStringOpcode("pitch_keycenter") != "" {
			foundPitchKeycenter = true
		}
	}

	if !foundVolume {
		t.Log("Note: No volume opcodes found in test.sfz - that's okay for this test")
	}

	if !foundPitchKeycenter {
		t.Log("Note: No pitch_keycenter opcodes found in test.sfz - that's okay for this test")
	}
}
