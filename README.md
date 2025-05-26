# SFZ Player Library

A Go 1.24 library that implements a simple SFZ sampler as a JACK client.

## Project Status

Currently in development. Milestone 0 (Prologue) is complete.

### Completed Milestones

#### Milestone 0: Prologue ✅
- Created comprehensive test assets in `testdata/` directory
- `test.sfz` - SFZ file using all supported opcodes
- `sample1.wav`, `sample2.wav`, `sample3.wav` - Test audio samples (16-bit mono PCM, 44.1kHz)
- Documented expected behavior for all opcode usage

## Test Assets

The `testdata/` directory contains:

- **`test.sfz`** - Comprehensive SFZ test file that exercises all supported opcodes including:
  - Structural elements: `<global>`, `<group>`, `<region>`
  - Key/velocity mapping: `lokey`, `hikey`, `lovel`, `hivel`, `key`
  - Basic playback: `volume`, `pitch_keycenter`
  - Envelope: `ampeg_attack`, `ampeg_decay`, `ampeg_sustain`, `ampeg_release`
  - Tuning: `tune`, `transpose`, `pitch`
  - Panning: `pan`
  - Looping: `loop_mode`, `loop_start`, `loop_end`

- **`sample1.wav`**, **`sample2.wav`**, **`sample3.wav`** - Test audio samples

- **`test-behavior.md`** - Detailed documentation of expected behavior for each opcode

## Supported SFZ Opcodes

The implementation will support these opcodes in priority order:

### Critical Core
- `<region>` - region definition
- `sample=` - audio file path

### Key/Velocity Mapping
- `lokey=`, `hikey=` - key range
- `lovel=`, `hivel=` - velocity range
- `key=` - single key mapping

### Basic Playback
- `volume=` - volume in dB
- `pitch_keycenter=` - root note

### Envelope
- `ampeg_attack=`, `ampeg_decay=`, `ampeg_sustain=`, `ampeg_release=` - ADSR envelope

### Structure
- `<group>` - group definition
- `<global>` - global settings

### Common Adjustments
- `tune=` - fine tuning in cents
- `pan=` - stereo position (-100 to 100)
- `transpose=` - semitone transpose
- `pitch=` - another tuning opcode

### Looping
- `loop_mode=` - no_loop/one_shot/loop_continuous/loop_sustain
- `loop_start=`, `loop_end=` - loop points in samples

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