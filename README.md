# SFZ Player Library

A Go 1.24 library that implements a simple SFZ sampler as a JACK client.

## Supported SFZ Opcodes

### Structure
- `<global>` - global settings
- `<group>` - group definition  
- `<region>` - region definition

### Critical Core
- `sample=` - audio file path

## Dependencies

- Go 1.24
- github.com/GeoffreyPlitt/debuggo - for debug logging
- JACK client library for Go (planned: github.com/xthexder/go-jack)
- WAV file reader (planned: github.com/go-audio/wav)

## API Design

Primary interface (planned):
```go
func NewSfzPlayer(sfzPath string) (*SfzPlayer, error)
```

The SfzPlayer will:
- Parse the provided SFZ file
- Load referenced audio samples
- Create a JACK client with one MIDI input port and one stereo audio output port
- Render audio in response to MIDI note messages