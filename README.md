# SFZ Player Library

[![Go Report Card](https://goreportcard.com/badge/github.com/GeoffreyPlitt/gosfzplayer)](https://goreportcard.com/report/github.com/GeoffreyPlitt/gosfzplayer)
[![Go Reference](https://pkg.go.dev/badge/github.com/GeoffreyPlitt/gosfzplayer.svg)](https://pkg.go.dev/github.com/GeoffreyPlitt/gosfzplayer)
[![Test](https://github.com/GeoffreyPlitt/gosfzplayer/workflows/Test/badge.svg)](https://github.com/GeoffreyPlitt/gosfzplayer/actions?query=workflow%3ATest)
[![codecov](https://codecov.io/gh/GeoffreyPlitt/gosfzplayer/branch/main/graph/badge.svg)](https://codecov.io/gh/GeoffreyPlitt/gosfzplayer)
[![Go Version](https://img.shields.io/github/go-mod/go-version/GeoffreyPlitt/gosfzplayer)](https://github.com/GeoffreyPlitt/gosfzplayer)

A Go library that implements a simple SFZ sampler with WAV sample loading.

## API

**Primary Interface:**
```go
func NewSfzPlayer(sfzPath string, jackClientName string) (*SfzPlayer, error)
```


## Features

- **SFZ File Parsing**: Complete parser for SFZ files with structured data representation
- **WAV Sample Loading**: Automatic loading and caching of WAV audio samples
- **Sample Caching**: Efficient caching system to avoid duplicate sample loads
- **Normalized Audio Data**: Audio samples normalized to float64 range (-1.0 to 1.0)
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
- github.com/go-audio/wav - for WAV file loading

## Usage

```go
package main

import (
    "fmt"
    "gosfzplayer"
)

func main() {
    // Create SFZ player - parses file, loads samples, starts JACK client
    player, err := gosfzplayer.NewSfzPlayer("path/to/instrument.sfz", "MyInstrument")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Println("SFZ Player created successfully!")
    
    // Player is now ready for MIDI input and audio output
    // Use JACK connection tools like QJackCtl to connect MIDI and audio
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

