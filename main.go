package gosfzplayer

import (
	"fmt"
	"path/filepath"

	"github.com/GeoffreyPlitt/debuggo"
)

var debug = debuggo.Debug("sfzplayer:main")

// SfzPlayer represents an SFZ sampler that can parse SFZ files and play samples
type SfzPlayer struct {
	sfzData     *SfzData
	sampleCache *SampleCache
	sfzDir      string // Directory containing the SFZ file for relative sample paths
	jackClient  *JackClient // Internal JACK client (nil if JACK not available)
}

// NewSfzPlayer creates a new SFZ player from an SFZ file
func NewSfzPlayer(sfzPath string, jackClientName string) (*SfzPlayer, error) {
	debug("Creating new SFZ player for file: %s", sfzPath)

	// Parse the SFZ file
	sfzData, err := ParseSfzFile(sfzPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFZ player: %w", err)
	}

	debug("Successfully parsed SFZ file with %d regions", len(sfzData.Regions))

	// Get the directory of the SFZ file for relative sample paths
	sfzDir := filepath.Dir(sfzPath)

	player := &SfzPlayer{
		sfzData:     sfzData,
		sampleCache: NewSampleCache(),
		sfzDir:      sfzDir,
	}

	// Load all samples referenced in the SFZ file
	err = player.loadAllSamples()
	if err != nil {
		return nil, fmt.Errorf("failed to load samples: %w", err)
	}

	// Try to create and start JACK client automatically if client name provided
	var jackClient *JackClient
	if jackClientName != "" {
		jackClient, err = NewJackClient(player, jackClientName)
		if err != nil {
			debug("Warning: Could not create JACK client: %v", err)
			// Continue without JACK - player still works for sample access
		} else {
			err = jackClient.Start()
			if err != nil {
				debug("Warning: Could not start JACK client: %v", err)
				jackClient.Close()
				jackClient = nil
			} else {
				player.jackClient = jackClient
				debug("JACK client started successfully as '%s'", jackClientName)
			}
		}
	}

	return player, nil
}

// loadAllSamples loads all sample files referenced in the SFZ regions
func (p *SfzPlayer) loadAllSamples() error {
	debug("Loading all samples referenced in SFZ file")

	for i, region := range p.sfzData.Regions {
		samplePath := region.GetStringOpcode("sample")
		if samplePath == "" {
			debug("Warning: Region %d has no sample opcode", i)
			continue
		}

		debug("Loading sample for region %d: %s", i, samplePath)
		_, err := p.sampleCache.LoadSampleRelative(p.sfzDir, samplePath)
		if err != nil {
			return fmt.Errorf("failed to load sample '%s' for region %d: %w", samplePath, i, err)
		}
	}

	debug("Successfully loaded %d unique samples", p.sampleCache.Size())
	return nil
}

// GetSample returns the loaded sample for a given file path
func (p *SfzPlayer) GetSample(samplePath string) (*Sample, error) {
	sample, exists := p.sampleCache.GetSample(filepath.Join(p.sfzDir, samplePath))
	if !exists {
		return nil, fmt.Errorf("sample not found: %s", samplePath)
	}
	return sample, nil
}

// GetSfzData returns the parsed SFZ data
func (p *SfzPlayer) GetSfzData() *SfzData {
	return p.sfzData
}

// StopAndClose stops and closes the internal JACK client if it's running
func (p *SfzPlayer) StopAndClose() error {
	if p.jackClient != nil {
		debug("Stopping and closing JACK client")
		
		// Stop first
		if err := p.jackClient.Stop(); err != nil {
			debug("Warning: Error stopping JACK client: %v", err)
		}
		
		// Then close
		if err := p.jackClient.Close(); err != nil {
			debug("Warning: Error closing JACK client: %v", err)
			return fmt.Errorf("failed to close JACK client: %w", err)
		}
		
		p.jackClient = nil
		debug("JACK client stopped and closed")
	}
	return nil
}
