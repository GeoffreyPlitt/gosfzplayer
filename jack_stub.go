//go:build !jack
// +build !jack

package gosfzplayer

import "fmt"

// JackClient stub for builds without JACK support
type JackClient struct{}

// NewJackClient creates a stub JACK client that returns an error
func NewJackClient(player *SfzPlayer, clientName string) (*JackClient, error) {
	return nil, fmt.Errorf("JACK support not enabled - rebuild with '-tags jack' and ensure JACK development headers are installed")
}

// Start returns an error for stub client
func (jc *JackClient) Start() error {
	return fmt.Errorf("JACK support not enabled")
}

// Stop returns an error for stub client
func (jc *JackClient) Stop() error {
	return fmt.Errorf("JACK support not enabled")
}

// Close returns an error for stub client
func (jc *JackClient) Close() error {
	return fmt.Errorf("JACK support not enabled")
}