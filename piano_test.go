//go:generate go run scripts/download_piano.go

package gosfzplayer

import (
	"os"
	"testing"
)

func TestPianoSamplesDownload(t *testing.T) {
	// This test verifies that the piano samples are available (FLAC format)
	expectedFiles := []string{
		"testdata/samples/A0vL.flac",
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Piano sample file missing: %s (run 'go generate' to download)", file)
		}
	}
}

func TestPianoSfzPlayer(t *testing.T) {
	// Check if piano.sfz exists, if not skip this test
	if _, err := os.Stat("testdata/piano.sfz"); os.IsNotExist(err) {
		t.Skip("piano.sfz not found, skipping piano player test")
	}

	// Create SFZ player with piano samples
	player, err := NewSfzPlayer("testdata/piano.sfz", "test-piano")
	if err != nil {
		t.Fatalf("Failed to create piano SFZ player: %v", err)
	}
	defer player.StopAndClose()

	// Verify that piano samples were loaded
	if player.sampleCache.Size() == 0 {
		t.Error("Expected piano samples to be loaded, but cache is empty")
	}

	// Test getting a specific piano sample (relative to SFZ file)
	sample, err := player.GetSample("samples/C1vH.flac")
	if err != nil {
		t.Errorf("Failed to get piano sample samples/C1vH.flac: %v", err)
	}

	if sample == nil {
		t.Error("Expected non-nil piano sample")
	}
}
