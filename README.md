# SFZ Player Library

[![Go Report Card](https://goreportcard.com/badge/github.com/GeoffreyPlitt/gosfzplayer)](https://goreportcard.com/report/github.com/GeoffreyPlitt/gosfzplayer)
[![Go Reference](https://pkg.go.dev/badge/github.com/GeoffreyPlitt/gosfzplayer.svg)](https://pkg.go.dev/github.com/GeoffreyPlitt/gosfzplayer)
[![Test](https://github.com/GeoffreyPlitt/gosfzplayer/workflows/Test/badge.svg)](https://github.com/GeoffreyPlitt/gosfzplayer/actions?query=workflow%3ATest)
[![codecov](https://codecov.io/gh/GeoffreyPlitt/gosfzplayer/branch/main/graph/badge.svg)](https://codecov.io/gh/GeoffreyPlitt/gosfzplayer)
[![Go Version](https://img.shields.io/github/go-mod/go-version/GeoffreyPlitt/gosfzplayer)](https://github.com/GeoffreyPlitt/gosfzplayer)

A Go library that implements a simple SFZ sampler as a JACK client.

## API

**Primary Interface:**
```go
func NewSfzPlayer(sfzPath string) (*SfzPlayer, error)
func (p *SfzPlayer) NewJackClient(clientName string) (*JackClient, error)
```

**JACK Audio Client:**
```go
func (jc *JackClient) Start() error
func (jc *JackClient) Stop() error
func (jc *JackClient) Close() error
```


## Features

- **SFZ File Parsing**: Complete parser for SFZ files with structured data representation
- **WAV Sample Loading**: Automatic loading and caching of WAV audio samples
- **JACK Audio Integration**: Real-time audio playback via JACK Audio Connection Kit
- **MIDI Input Processing**: Note on/off events trigger sample playback
- **Key/Velocity Mapping**: Regions respond to specific note and velocity ranges
- **Volume and Panning**: Support for volume (dB) and pan (-100 to +100) opcodes
- **Polyphonic Playback**: Multiple simultaneous voices with configurable polyphony limit
- **Debug Logging**: Comprehensive logging with configurable namespaces

## Test Assets

The `testdata/` directory contains:

- **`test.sfz`** - Comprehensive SFZ test file that exercises all supported opcodes
- **`sample1.wav`**, **`sample2.wav`**, **`sample3.wav`** - Test audio samples
- **`test-behavior.md`** - Detailed documentation of expected behavior for each opcode

## Dependencies

- Go 1.21+ (tested on 1.21, 1.22, 1.23, 1.24)
- JACK Audio Connection Kit development headers (libjack-jackd2-dev on Ubuntu)
- github.com/GeoffreyPlitt/debuggo - for debug logging
- github.com/go-audio/wav - for WAV file loading
- github.com/xthexder/go-jack - for JACK audio integration

## Installation

```bash
# Install JACK development headers (Ubuntu/Debian)
sudo apt-get install libjack-jackd2-dev

# Build with JACK support
go build -tags jack

# Or run tests with JACK support
go test -tags jack -v
```

## Usage

```go
package main

import (
    "fmt"
    "gosfzplayer"
)

func main() {
    // Parse an SFZ file and load all samples
    player, err := gosfzplayer.NewSfzPlayer("path/to/instrument.sfz")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    // Create JACK audio client
    jackClient, err := player.NewJackClient("SFZ Player")
    if err != nil {
        fmt.Printf("Error creating JACK client: %v\n", err)
        return
    }
    defer jackClient.Close()
    
    // Start audio processing
    err = jackClient.Start()
    if err != nil {
        fmt.Printf("Error starting JACK client: %v\n", err)
        return
    }
    defer jackClient.Stop()
    
    fmt.Println("SFZ Player running. Connect MIDI input and audio output in QJackCtl.")
    
    // Keep running until interrupted
    select {} // or use signal handling for graceful shutdown
}
```

## Debug Logging

Enable debug output with the `DEBUG` environment variable:

```bash
# Enable all debug output
DEBUG=sfzplayer:* go run main.go

# Enable only parser debug output  
DEBUG=sfzplayer:parser go run main.go
```

## Testing

Run tests:
```bash
go test -v
```

Run tests with debug output:
```bash
DEBUG=sfzplayer:* go test -v
```

