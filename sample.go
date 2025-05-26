package gosfzplayer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GeoffreyPlitt/debuggo"
	"github.com/go-audio/wav"
	"github.com/mewkiz/flac"
)

var sampleDebug = debuggo.Debug("sfzplayer:sample")

// Sample represents a loaded audio sample
type Sample struct {
	FilePath   string    // Original file path
	Data       []float64 // Audio data as float64 samples
	SampleRate int       // Sample rate in Hz
	Channels   int       // Number of audio channels
	Length     int       // Number of samples per channel
}

// SampleCache manages loaded samples to avoid duplicate loading
type SampleCache struct {
	samples map[string]*Sample // File path -> Sample
}

// NewSampleCache creates a new sample cache
func NewSampleCache() *SampleCache {
	return &SampleCache{
		samples: make(map[string]*Sample),
	}
}

// LoadSample loads a WAV or FLAC file and returns a Sample, using cache if available
func (sc *SampleCache) LoadSample(filePath string) (*Sample, error) {
	// Check cache first
	if sample, exists := sc.samples[filePath]; exists {
		sampleDebug("Sample already cached: %s", filePath)
		return sample, nil
	}

	sampleDebug("Loading new sample: %s", filePath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("sample file not found: %s", filePath)
	}

	// Determine file type based on extension
	ext := strings.ToLower(filepath.Ext(filePath))

	var sample *Sample
	var err error

	switch ext {
	case ".wav":
		sample, err = sc.loadWAV(filePath)
	case ".flac":
		sample, err = sc.loadFLAC(filePath)
	default:
		return nil, fmt.Errorf("unsupported audio format: %s (supported: .wav, .flac)", ext)
	}

	if err != nil {
		return nil, err
	}

	// Cache the sample
	sc.samples[filePath] = sample

	sampleDebug("Loaded sample: %s (rate: %d Hz, channels: %d, length: %d samples)",
		filePath, sample.SampleRate, sample.Channels, sample.Length)

	return sample, nil
}

// loadWAV loads a WAV file
func (sc *SampleCache) loadWAV(filePath string) (*Sample, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAV file %s: %w", filePath, err)
	}
	defer file.Close()

	// Create WAV decoder
	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV file: %s", filePath)
	}

	// Read audio data
	audioData, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data from %s: %w", filePath, err)
	}

	// Convert to float64 samples
	samples := make([]float64, len(audioData.Data))
	for i, sample := range audioData.Data {
		// Convert from int to float64, normalize based on bit depth
		switch decoder.BitDepth {
		case 16:
			samples[i] = float64(sample) / 32768.0
		case 24:
			samples[i] = float64(sample) / 8388608.0
		case 32:
			samples[i] = float64(sample) / 2147483648.0
		default:
			samples[i] = float64(sample) / 32768.0 // Default to 16-bit
		}
	}

	return &Sample{
		FilePath:   filePath,
		Data:       samples,
		SampleRate: int(audioData.Format.SampleRate),
		Channels:   int(audioData.Format.NumChannels),
		Length:     len(samples) / int(audioData.Format.NumChannels),
	}, nil
}

// loadFLAC loads a FLAC file
func (sc *SampleCache) loadFLAC(filePath string) (*Sample, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open FLAC file %s: %w", filePath, err)
	}
	defer file.Close()

	// Create FLAC decoder
	stream, err := flac.New(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create FLAC decoder for %s: %w", filePath, err)
	}
	defer stream.Close()

	// Get stream info
	info := stream.Info
	if info == nil {
		return nil, fmt.Errorf("no stream info available for FLAC file: %s", filePath)
	}

	sampleRate := int(info.SampleRate)
	channels := int(info.NChannels)
	bitsPerSample := int(info.BitsPerSample)

	// Read all audio frames
	var allSamples []float64
	for {
		frame, err := stream.ParseNext()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to read FLAC frame from %s: %w", filePath, err)
		}

		// Convert frame samples to float64
		for i := 0; i < len(frame.Subframes[0].Samples); i++ {
			for ch := 0; ch < channels; ch++ {
				sample := frame.Subframes[ch].Samples[i]

				// Normalize based on bit depth
				var normalizedSample float64
				switch bitsPerSample {
				case 16:
					normalizedSample = float64(sample) / 32768.0
				case 24:
					normalizedSample = float64(sample) / 8388608.0
				case 32:
					normalizedSample = float64(sample) / 2147483648.0
				default:
					normalizedSample = float64(sample) / 32768.0 // Default to 16-bit
				}

				allSamples = append(allSamples, normalizedSample)
			}
		}
	}

	return &Sample{
		FilePath:   filePath,
		Data:       allSamples,
		SampleRate: sampleRate,
		Channels:   channels,
		Length:     len(allSamples) / channels,
	}, nil
}

// LoadSampleRelative loads a sample with a path relative to the SFZ file directory
func (sc *SampleCache) LoadSampleRelative(sfzDir, relativePath string) (*Sample, error) {
	absolutePath := filepath.Join(sfzDir, relativePath)
	return sc.LoadSample(absolutePath)
}

// GetSample returns a cached sample if it exists
func (sc *SampleCache) GetSample(filePath string) (*Sample, bool) {
	sample, exists := sc.samples[filePath]
	return sample, exists
}

// Clear removes all samples from the cache
func (sc *SampleCache) Clear() {
	sc.samples = make(map[string]*Sample)
	sampleDebug("Sample cache cleared")
}

// Size returns the number of cached samples
func (sc *SampleCache) Size() int {
	return len(sc.samples)
}
