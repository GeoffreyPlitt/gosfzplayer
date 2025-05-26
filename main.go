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
	sfzDir      string      // Directory containing the SFZ file for relative sample paths
	jackClient  *JackClient // Internal JACK client (nil if JACK not available)
	reverb      *Freeverb   // Master reverb processor
	reverbSend  float64     // Global reverb send level (0.0 to 1.0)
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
		reverb:      NewFreeverb(44100), // Initialize with default sample rate
		reverbSend:  0.0,                // Start with no reverb
	}

	// Load all samples referenced in the SFZ file
	err = player.loadAllSamples()
	if err != nil {
		return nil, fmt.Errorf("failed to load samples: %w", err)
	}

	// Load reverb settings from SFZ file
	player.loadReverbSettings()

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

// Reverb Control Methods

// SetReverbSend sets the global reverb send level (0.0 to 1.0)
func (p *SfzPlayer) SetReverbSend(send float64) {
	if send < 0.0 {
		send = 0.0
	}
	if send > 1.0 {
		send = 1.0
	}
	p.reverbSend = send
	debug("Reverb send set to %.2f", send)
}

// GetReverbSend returns the current reverb send level
func (p *SfzPlayer) GetReverbSend() float64 {
	return p.reverbSend
}

// SetReverbRoomSize sets the reverb room size (0.0 to 1.0)
func (p *SfzPlayer) SetReverbRoomSize(size float64) {
	p.reverb.SetRoomSize(size)
	debug("Reverb room size set to %.2f", size)
}

// GetReverbRoomSize returns the current reverb room size
func (p *SfzPlayer) GetReverbRoomSize() float64 {
	return p.reverb.GetRoomSize()
}

// SetReverbDamping sets the reverb damping (0.0 to 1.0)
func (p *SfzPlayer) SetReverbDamping(damp float64) {
	p.reverb.SetDamping(damp)
	debug("Reverb damping set to %.2f", damp)
}

// GetReverbDamping returns the current reverb damping
func (p *SfzPlayer) GetReverbDamping() float64 {
	return p.reverb.GetDamping()
}

// SetReverbWet sets the reverb wet level (0.0 to 1.0)
func (p *SfzPlayer) SetReverbWet(wet float64) {
	p.reverb.SetWet(wet)
	debug("Reverb wet level set to %.2f", wet)
}

// GetReverbWet returns the current reverb wet level
func (p *SfzPlayer) GetReverbWet() float64 {
	return p.reverb.GetWet()
}

// SetReverbDry sets the reverb dry level (0.0 to 1.0)
func (p *SfzPlayer) SetReverbDry(dry float64) {
	p.reverb.SetDry(dry)
	debug("Reverb dry level set to %.2f", dry)
}

// GetReverbDry returns the current reverb dry level
func (p *SfzPlayer) GetReverbDry() float64 {
	return p.reverb.GetDry()
}

// SetReverbWidth sets the reverb stereo width (0.0 to 1.0)
func (p *SfzPlayer) SetReverbWidth(width float64) {
	p.reverb.SetWidth(width)
	debug("Reverb width set to %.2f", width)
}

// GetReverbWidth returns the current reverb stereo width
func (p *SfzPlayer) GetReverbWidth() float64 {
	return p.reverb.GetWidth()
}

// loadReverbSettings reads reverb opcodes from the SFZ file and applies them
func (p *SfzPlayer) loadReverbSettings() {
	// Check global section first
	if p.sfzData.Global != nil {
		p.applyReverbOpcodes(p.sfzData.Global)
	}

	// Apply reverb settings from groups and regions as defaults
	// Note: In a full implementation, reverb would typically be per-voice
	// but for simplicity, we're using global reverb here
	for _, group := range p.sfzData.Groups {
		p.applyReverbOpcodes(group)
		break // Use first group's settings as global default
	}

	debug("Reverb settings loaded from SFZ file")
}

// applyReverbOpcodes applies reverb opcodes from an SFZ section
func (p *SfzPlayer) applyReverbOpcodes(section *SfzSection) {
	// Standard SFZ reverb opcodes
	if reverbSend := section.GetStringOpcode("reverb_send"); reverbSend != "" {
		if value := section.GetFloatOpcode("reverb_send", -1); value >= 0 {
			// Convert from dB to linear if needed, or use as percentage
			p.SetReverbSend(value / 100.0) // Assuming percentage
		}
	}

	// Custom reverb opcodes (non-standard but useful)
	if roomSize := section.GetStringOpcode("reverb_room_size"); roomSize != "" {
		if value := section.GetFloatOpcode("reverb_room_size", -1); value >= 0 {
			p.SetReverbRoomSize(value / 100.0)
		}
	}

	if damping := section.GetStringOpcode("reverb_damping"); damping != "" {
		if value := section.GetFloatOpcode("reverb_damping", -1); value >= 0 {
			p.SetReverbDamping(value / 100.0)
		}
	}

	if wet := section.GetStringOpcode("reverb_wet"); wet != "" {
		if value := section.GetFloatOpcode("reverb_wet", -1); value >= 0 {
			p.SetReverbWet(value / 100.0)
		}
	}

	if dry := section.GetStringOpcode("reverb_dry"); dry != "" {
		if value := section.GetFloatOpcode("reverb_dry", -1); value >= 0 {
			p.SetReverbDry(value / 100.0)
		}
	}

	if width := section.GetStringOpcode("reverb_width"); width != "" {
		if value := section.GetFloatOpcode("reverb_width", -1); value >= 0 {
			p.SetReverbWidth(value / 100.0)
		}
	}
}
