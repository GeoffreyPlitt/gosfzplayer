package gosfzplayer

import (
	"github.com/GeoffreyPlitt/debuggo"
)

var debug = debuggo.Debug("sfzplayer:main")

// SfzPlayer represents an SFZ sampler that can parse SFZ files and play samples
type SfzPlayer struct {
	sfzData *SfzData
}

// NewSfzPlayer creates a new SFZ player from an SFZ file
func NewSfzPlayer(sfzPath string) (*SfzPlayer, error) {
	debug("Creating new SFZ player for file: %s", sfzPath)
	
	// Parse the SFZ file
	sfzData, err := ParseSfzFile(sfzPath)
	if err != nil {
		debug("Failed to parse SFZ file: %v", err)
		return nil, err
	}
	
	debug("Successfully parsed SFZ file with %d regions", len(sfzData.Regions))
	
	return &SfzPlayer{
		sfzData: sfzData,
	}, nil
}