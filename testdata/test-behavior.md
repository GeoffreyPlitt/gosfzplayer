# Test SFZ Expected Behavior Documentation

This document describes the expected behavior for each opcode used in `test.sfz`.

## Global Section Opcodes
- `volume=-6.0`: All regions should be 6dB quieter than their individual volume settings
- `tune=+10`: All regions should be tuned 10 cents sharp (unless overridden)
- `pan=0`: Default center panning for all regions (unless overridden)

## Group 1 Opcodes
- `transpose=0`: No transposition (default behavior)
- `pitch=100`: Additional 100 cents (1 semitone) sharp tuning
- `ampeg_attack=0.01`: 10ms attack time for envelope
- `ampeg_decay=0.1`: 100ms decay time
- `ampeg_sustain=80`: Sustain at 80% of peak level
- `ampeg_release=0.2`: 200ms release time

## Region 1 (sample1.wav, C2-C4, velocity 1-64)
- `sample=sample1.wav`: Plays sample1.wav file
- `lokey=c2, hikey=c4`: Responds to MIDI notes 36-60
- `lovel=1, hivel=64`: Only responds to velocities 1-64
- `key=c3, pitch_keycenter=c3`: Root note is C3 (MIDI 48), no pitch adjustment when C3 is played
- `volume=0.0`: No additional volume change beyond global -6dB
- `loop_mode=no_loop`: Sample plays once from start to end

## Region 2 (sample2.wav, D3 only, velocity 65-127)
- `key=d3`: Only responds to D3 (MIDI note 50)
- `lovel=65, hivel=127`: Only responds to higher velocities
- `pitch_keycenter=d3`: Root note is D3, no pitch adjustment when D3 is played
- `volume=-3.0`: 3dB quieter than global setting (total -9dB)
- `pan=-50`: Panned 50% to the left
- `tune=-20`: Detuned 20 cents flat (overrides global +10 cents)
- `loop_mode=loop_continuous`: Sample loops continuously from loop_start to loop_end
- `loop_start=1000, loop_end=8000`: Loops between sample 1000 and 8000

## Group 2 Opcodes
- `ampeg_attack=0.5`: 500ms attack time (overrides Group 1)
- `ampeg_decay=0.3`: 300ms decay time
- `ampeg_sustain=60`: 60% sustain level
- `ampeg_release=1.0`: 1 second release time
- `transpose=12`: All regions in this group transposed up one octave

## Region 3 (sample3.wav, C5-C7, all velocities)
- `lokey=c5, hikey=c7`: Responds to MIDI notes 72-96
- `pitch_keycenter=c6`: Root note is C6 (MIDI 84)
- `volume=+3.0`: 3dB louder than global setting (total -3dB)
- `pan=+75`: Panned 75% to the right
- `tune=+50`: 50 cents sharp (adds to global +10 = +60 cents total)
- `loop_mode=loop_sustain`: Loops while key is held, stops looping on release
- `loop_start=500, loop_end=4000`: Loops between sample 500 and 4000
- `transpose=12`: Transposed up one octave from Group 2

## Region 4 (sample1.wav, C1-B1, all velocities)
- `lokey=c1, hikey=b1`: Responds to MIDI notes 24-35
- `pitch_keycenter=c1`: Root note is C1 (MIDI 24)
- `loop_mode=one_shot`: Plays once, ignores note-off messages
- `volume=-12.0`: 12dB quieter than global (total -18dB)
- `pitch=-100`: 100 cents flat (1 semitone down)

## Region 5 (sample2.wav, C7-C8, all velocities)
- `lokey=c7, hikey=c8`: Responds to MIDI notes 96-108
- `lovel=1, hivel=127`: Responds to all velocities
- `pitch_keycenter=c7`: Root note is C7 (MIDI 96)
- `transpose=-12`: Transposed down one octave
- `volume=+6.0`: 6dB louder than global setting (total 0dB)

## Expected Playback Behavior

### Key Mapping
- C1-B1: Region 4 (one_shot, very quiet, pitched down)
- C2-C4: Region 1 (no loop, medium volume, velocity 1-64 only)
- D3: Region 2 (continuous loop, panned left, velocity 65-127 only)
- C5-C7: Region 3 (sustain loop, panned right, transposed up)
- C7-C8: Region 5 (no loop, transposed down, full velocity)

### Overlapping Regions
- C7: Both Region 3 and Region 5 should trigger simultaneously
- D3 with velocity 1-64: Only Region 1 should trigger
- D3 with velocity 65-127: Only Region 2 should trigger

### Envelope Behavior
- Regions 1 & 2: Fast attack (10ms), quick decay (100ms), 80% sustain, short release (200ms)
- Regions 3, 4 & 5: Slow attack (500ms), longer decay (300ms), 60% sustain, long release (1s)

### Pitch Behavior
- All regions affected by global tuning and group transposition
- Individual pitch adjustments applied per region
- Pitch changes based on played note vs pitch_keycenter