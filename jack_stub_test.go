//go:build !jack
// +build !jack

package gosfzplayer

import (
	"strings"
	"testing"
)

func TestJackStubFunctionality(t *testing.T) {
	// Create a test SFZ player
	player, err := NewSfzPlayer("testdata/test.sfz")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}

	// Try to create JACK client (should fail with stub)
	jackClient, err := player.NewJackClient("Test Client")
	if err == nil {
		t.Error("Expected error when creating JACK client without JACK support")
	}

	if jackClient != nil {
		t.Error("Expected nil JACK client when JACK support is disabled")
	}

	expectedError := "JACK support not enabled"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
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