package gosfzplayer

import (
	"testing"
)

func TestSampleCache(t *testing.T) {
	cache := NewSampleCache()

	// Test initial state
	if cache.Size() != 0 {
		t.Errorf("Expected empty cache, got size %d", cache.Size())
	}

	// Test cache miss
	sample, exists := cache.GetSample("nonexistent.wav")
	if exists || sample != nil {
		t.Error("Expected cache miss for nonexistent sample")
	}
}

func TestLoadSampleNotFound(t *testing.T) {
	cache := NewSampleCache()

	_, err := cache.LoadSample("nonexistent.wav")
	if err == nil {
		t.Error("Expected error for nonexistent sample file")
	}
}

func TestLoadSampleRelative(t *testing.T) {
	cache := NewSampleCache()

	// Try to load a test sample (should exist in testdata/)
	_, err := cache.LoadSampleRelative("testdata", "sample1.wav")
	if err != nil {
		t.Errorf("Failed to load sample1.wav: %v", err)
	}

	// Check cache hit
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}

	// Load same sample again (should hit cache)
	_, err = cache.LoadSampleRelative("testdata", "sample1.wav")
	if err != nil {
		t.Errorf("Failed to load cached sample: %v", err)
	}

	// Cache size should remain 1
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1 after second load, got %d", cache.Size())
	}
}

func TestCacheClearAndSize(t *testing.T) {
	cache := NewSampleCache()

	// Load multiple samples
	testSamples := []string{"sample1.wav", "sample2.wav", "sample3.wav"}
	for _, sample := range testSamples {
		_, err := cache.LoadSampleRelative("testdata", sample)
		if err != nil {
			t.Errorf("Failed to load %s: %v", sample, err)
		}
	}

	// Check size
	if cache.Size() != len(testSamples) {
		t.Errorf("Expected cache size %d, got %d", len(testSamples), cache.Size())
	}

	// Clear cache
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Expected empty cache after clear, got size %d", cache.Size())
	}
}

func TestSfzPlayerWithSamples(t *testing.T) {
	// Create player with test SFZ file
	player, err := NewSfzPlayer("testdata/test.sfz", "test-player")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}

	// Check that samples were loaded
	if player.sampleCache.Size() == 0 {
		t.Error("Expected samples to be loaded, but cache is empty")
	}

	// Check that we can get a specific sample
	sample, err := player.GetSample("sample1.wav")
	if err != nil {
		t.Errorf("Failed to get sample1.wav: %v", err)
	}

	if sample == nil {
		t.Error("Expected non-nil sample")
	}

	// Verify sample properties
	if sample.SampleRate <= 0 {
		t.Errorf("Invalid sample rate: %d", sample.SampleRate)
	}

	if sample.Channels <= 0 {
		t.Errorf("Invalid channel count: %d", sample.Channels)
	}

	if len(sample.Data) == 0 {
		t.Error("Expected sample data, got empty slice")
	}

	if sample.Length != len(sample.Data)/sample.Channels {
		t.Errorf("Sample length mismatch: expected %d, got %d",
			len(sample.Data)/sample.Channels, sample.Length)
	}
}

func TestSfzPlayerGetNonexistentSample(t *testing.T) {
	player, err := NewSfzPlayer("testdata/test.sfz", "test-player-2")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}

	_, err = player.GetSample("nonexistent.wav")
	if err == nil {
		t.Error("Expected error for nonexistent sample")
	}
}

func TestSampleDataNormalization(t *testing.T) {
	cache := NewSampleCache()

	sample, err := cache.LoadSampleRelative("testdata", "sample1.wav")
	if err != nil {
		t.Fatalf("Failed to load sample: %v", err)
	}

	// Check that sample data is normalized between -1.0 and 1.0
	for i, value := range sample.Data {
		if value < -1.0 || value > 1.0 {
			t.Errorf("Sample data[%d] = %f is outside normalized range [-1.0, 1.0]", i, value)
		}
	}
}
