//go:build !jack
// +build !jack

package gosfzplayer

import (
	"testing"
)

func TestJackStubFunctionality(t *testing.T) {
	// Create a test SFZ player - JACK client creation should fail silently
	player, err := NewSfzPlayer("testdata/test.sfz", "test-client")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}

	// Player should still be created successfully even without JACK
	if player == nil {
		t.Error("Expected player to be created even without JACK support")
	}

	// JACK client should be nil in stub mode
	if player.jackClient != nil {
		t.Error("Expected JACK client to be nil when JACK support is disabled")
	}

	// StopAndClose should work fine with no JACK client
	err = player.StopAndClose()
	if err != nil {
		t.Errorf("StopAndClose should not error when no JACK client exists: %v", err)
	}
}

func TestJackStubMethods(t *testing.T) {
	// Create stub client directly
	client := &JackClient{}

	// Test all methods return errors
	if err := client.Start(); err == nil {
		t.Error("Expected Start() to return error for stub client")
	}

	if err := client.Stop(); err == nil {
		t.Error("Expected Stop() to return error for stub client")
	}

	if err := client.Close(); err == nil {
		t.Error("Expected Close() to return error for stub client")
	}
}