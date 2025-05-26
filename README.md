# SFZ Player Library

[![Go Report Card](https://goreportcard.com/badge/github.com/GeoffreyPlitt/gosfzplayer)](https://goreportcard.com/report/github.com/GeoffreyPlitt/gosfzplayer)
[![Go Reference](https://pkg.go.dev/badge/github.com/GeoffreyPlitt/gosfzplayer.svg)](https://pkg.go.dev/github.com/GeoffreyPlitt/gosfzplayer)
[![Test](https://github.com/GeoffreyPlitt/gosfzplayer/workflows/Test/badge.svg)](https://github.com/GeoffreyPlitt/gosfzplayer/actions?query=workflow%3ATest)
[![Lint](https://github.com/GeoffreyPlitt/gosfzplayer/workflows/Lint/badge.svg)](https://github.com/GeoffreyPlitt/gosfzplayer/actions?query=workflow%3ALint)
[![codecov](https://codecov.io/gh/GeoffreyPlitt/gosfzplayer/branch/main/graph/badge.svg)](https://codecov.io/gh/GeoffreyPlitt/gosfzplayer)
[![Go Version](https://img.shields.io/github/go-mod/go-version/GeoffreyPlitt/gosfzplayer)](https://github.com/GeoffreyPlitt/gosfzplayer)

A Go library that implements a simple SFZ sampler as a JACK client.

## API

**Primary Interface:**
```go
func NewSfzPlayer(sfzPath string) (*SfzPlayer, error)
```

**Parser Functions:**
```go
func ParseSfzFile(filePath string) (*SfzData, error)
```

**Type Conversion Helpers:**
```go
func (s *SfzSection) GetStringOpcode(opcode string) string
func (s *SfzSection) GetIntOpcode(opcode string, defaultValue int) int
func (s *SfzSection) GetFloatOpcode(opcode string, defaultValue float64) float64
```

## Data Structures

```go
type SfzData struct {
    Global  *SfzSection
    Groups  []*SfzSection
    Regions []*SfzSection
}

type SfzSection struct {
    Type    string            // "global", "group", or "region"
    Opcodes map[string]string // opcode name -> value
}
```

## Features

- **SFZ File Parsing**: Complete parser for SFZ files with structured data representation
- **Error Handling**: Graceful handling of missing files and invalid syntax
- **Debug Logging**: Comprehensive logging with configurable namespaces
- **Type Conversion**: Helper functions for string to numeric type conversion

## Test Assets

The `testdata/` directory contains:

- **`test.sfz`** - Comprehensive SFZ test file that exercises all supported opcodes
- **`sample1.wav`**, **`sample2.wav`**, **`sample3.wav`** - Test audio samples
- **`test-behavior.md`** - Detailed documentation of expected behavior for each opcode

## Dependencies

- Go 1.21+ (tested on 1.21, 1.22, 1.23, 1.24)
- github.com/GeoffreyPlitt/debuggo - for debug logging
- JACK client library for Go (planned: github.com/xthexder/go-jack)
- WAV file reader (planned: github.com/go-audio/wav)

## Usage

```go
package main

import (
    "fmt"
    "gosfzplayer"
)

func main() {
    // Parse an SFZ file
    player, err := gosfzplayer.NewSfzPlayer("path/to/instrument.sfz")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    // Player is ready (audio playback not yet implemented)
    fmt.Println("SFZ file parsed successfully!")
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

