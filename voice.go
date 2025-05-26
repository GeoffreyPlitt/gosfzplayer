package gosfzplayer

import (
	"github.com/GeoffreyPlitt/debuggo"
)

var voiceDebug = debuggo.Debug("sfzplayer:voice")

// EnvelopeState represents the current state of an ADSR envelope
type EnvelopeState int

const (
	EnvelopeAttack EnvelopeState = iota
	EnvelopeDecay
	EnvelopeSustain
	EnvelopeRelease
	EnvelopeOff
)

// Voice represents an active playing voice/note
type Voice struct {
	sample     *Sample
	region     *SfzSection
	midiNote   uint8
	velocity   uint8
	position   float64 // Current playback position in samples (float for pitch adjustment)
	volume     float64
	pan        float64
	pitchRatio float64 // Pitch adjustment ratio (1.0 = no change, 2.0 = octave up)
	isActive   bool
	noteOn     bool

	// ADSR Envelope
	envelopeState  EnvelopeState
	envelopeLevel  float64 // Current envelope level (0.0 to 1.0)
	envelopeTime   float64 // Time in current envelope stage (in samples)
	attackSamples  float64 // Attack time in samples
	decaySamples   float64 // Decay time in samples
	sustainLevel   float64 // Sustain level (0.0 to 1.0)
	releaseSamples float64 // Release time in samples

	// Loop Support
	loopMode  string  // Loop mode: no_loop, one_shot, loop_continuous, loop_sustain
	loopStart float64 // Loop start point in samples
	loopEnd   float64 // Loop end point in samples

	// Advanced Features
	groupID     int    // Group number for exclusion
	offByGroup  int    // Group that can stop this voice
	triggerMode string // Trigger mode: attack, release, first, legato
}

// InitializeEnvelope sets up the ADSR envelope for a voice
func (v *Voice) InitializeEnvelope(sampleRate uint32) {
	// Default ADSR values (in seconds)
	defaultAttack := 0.001 // 1ms
	defaultDecay := 0.1    // 100ms
	defaultSustain := 1.0  // 100%
	defaultRelease := 0.1  // 100ms

	// Parse envelope opcodes with inheritance (Region → Group → Global)
	attack := v.region.GetInheritedFloatOpcode("ampeg_attack", defaultAttack)
	if attack < 0 {
		attack = defaultAttack
	}

	decay := v.region.GetInheritedFloatOpcode("ampeg_decay", defaultDecay)
	if decay < 0 {
		decay = defaultDecay
	}

	sustain := v.region.GetInheritedFloatOpcode("ampeg_sustain", defaultSustain*100) / 100.0 // Convert percentage to 0-1
	if sustain < 0 || sustain > 1 {
		sustain = defaultSustain
	}

	release := v.region.GetInheritedFloatOpcode("ampeg_release", defaultRelease)
	if release < 0 {
		release = defaultRelease
	}

	// Convert times to samples
	v.attackSamples = attack * float64(sampleRate)
	v.decaySamples = decay * float64(sampleRate)
	v.sustainLevel = sustain
	v.releaseSamples = release * float64(sampleRate)

	// Initialize envelope state
	v.envelopeState = EnvelopeAttack
	v.envelopeLevel = 0.0
	v.envelopeTime = 0.0

	voiceDebug("Initialized envelope: attack=%.3fs (%d samples), decay=%.3fs (%d samples), sustain=%.1f%%, release=%.3fs (%d samples)",
		attack, int(v.attackSamples), decay, int(v.decaySamples), sustain*100, release, int(v.releaseSamples))
}

// ProcessEnvelope updates the envelope state and returns the current envelope level
func (v *Voice) ProcessEnvelope() float64 {
	switch v.envelopeState {
	case EnvelopeAttack:
		if v.attackSamples <= 0 {
			// Instant attack
			v.envelopeLevel = 1.0
			v.envelopeState = EnvelopeDecay
			v.envelopeTime = 0.0
		} else {
			// Linear attack
			v.envelopeLevel = v.envelopeTime / v.attackSamples
			if v.envelopeLevel >= 1.0 {
				v.envelopeLevel = 1.0
				v.envelopeState = EnvelopeDecay
				v.envelopeTime = 0.0
			}
		}

	case EnvelopeDecay:
		if v.decaySamples <= 0 {
			// Instant decay
			v.envelopeLevel = v.sustainLevel
			v.envelopeState = EnvelopeSustain
		} else {
			// Linear decay from 1.0 to sustain level
			progress := v.envelopeTime / v.decaySamples
			if progress >= 1.0 {
				v.envelopeLevel = v.sustainLevel
				v.envelopeState = EnvelopeSustain
			} else {
				v.envelopeLevel = 1.0 - progress*(1.0-v.sustainLevel)
			}
		}

	case EnvelopeSustain:
		// Hold at sustain level while note is on
		v.envelopeLevel = v.sustainLevel

	case EnvelopeRelease:
		if v.releaseSamples <= 0 {
			// Instant release
			v.envelopeLevel = 0.0
			v.envelopeState = EnvelopeOff
		} else {
			// Linear release from current level to 0
			startLevel := v.sustainLevel
			progress := v.envelopeTime / v.releaseSamples
			if progress >= 1.0 {
				v.envelopeLevel = 0.0
				v.envelopeState = EnvelopeOff
			} else {
				v.envelopeLevel = startLevel * (1.0 - progress)
			}
		}

	case EnvelopeOff:
		v.envelopeLevel = 0.0
		v.isActive = false
	}

	v.envelopeTime++
	return v.envelopeLevel
}

// TriggerRelease starts the release phase of the envelope
func (v *Voice) TriggerRelease() {
	if v.envelopeState != EnvelopeRelease && v.envelopeState != EnvelopeOff {
		v.envelopeState = EnvelopeRelease
		v.envelopeTime = 0.0
		v.noteOn = false

		// For loop_sustain mode, stop looping when note is released
		if v.loopMode == "loop_sustain" {
			v.loopMode = "no_loop"
			voiceDebug("Voice note off: switching from loop_sustain to no_loop for note %d", v.midiNote)
		}

		voiceDebug("Voice release triggered for note %d", v.midiNote)
	}
}

// InitializeLoop sets up loop parameters for a voice
func (v *Voice) InitializeLoop() {
	// Get loop mode with inheritance (default: no_loop)
	v.loopMode = v.region.GetInheritedStringOpcode("loop_mode")
	if v.loopMode == "" {
		v.loopMode = "no_loop"
	}

	// Get loop points with inheritance (default: 0 to end of sample)
	v.loopStart = float64(v.region.GetInheritedIntOpcode("loop_start", 0))
	v.loopEnd = float64(v.region.GetInheritedIntOpcode("loop_end", -1))

	// Validate and set defaults for loop end
	sampleLength := float64(len(v.sample.Data))
	if v.loopEnd < 0 || v.loopEnd >= sampleLength {
		v.loopEnd = sampleLength - 1
	}

	// Validate loop start
	if v.loopStart < 0 {
		v.loopStart = 0
	}

	// Ensure loop_start < loop_end
	if v.loopStart >= v.loopEnd {
		v.loopStart = 0
		v.loopEnd = sampleLength - 1
		voiceDebug("Invalid loop points for note %d, using full sample", v.midiNote)
	}

	voiceDebug("Initialized loop: mode=%s, start=%.0f, end=%.0f (sample length=%.0f)",
		v.loopMode, v.loopStart, v.loopEnd, sampleLength)
}

// ProcessLoop handles loop behavior and returns true if voice should continue playing
func (v *Voice) ProcessLoop() bool {
	sampleLength := float64(len(v.sample.Data))

	switch v.loopMode {
	case "no_loop":
		// Stop when reaching end of sample
		if v.position >= sampleLength-1 {
			return false
		}

	case "one_shot":
		// Play once, ignore note off, stop at end
		if v.position >= sampleLength-1 {
			return false
		}

	case "loop_continuous":
		// Loop indefinitely between loop points
		if v.position >= v.loopEnd {
			// Jump back to loop start
			v.position = v.loopStart + (v.position - v.loopEnd)
			voiceDebug("Voice %d: looping from %.0f back to %.0f", v.midiNote, v.loopEnd, v.position)
		}

	case "loop_sustain":
		// Loop while note is held, stop looping on note off
		if v.noteOn && v.position >= v.loopEnd {
			// Jump back to loop start while note is held
			v.position = v.loopStart + (v.position - v.loopEnd)
			voiceDebug("Voice %d: sustain looping from %.0f back to %.0f", v.midiNote, v.loopEnd, v.position)
		} else if !v.noteOn && v.position >= sampleLength-1 {
			// Stop when reaching end after note off
			return false
		}

	default:
		// Unknown loop mode, treat as no_loop
		if v.position >= sampleLength-1 {
			return false
		}
	}

	return true
}
