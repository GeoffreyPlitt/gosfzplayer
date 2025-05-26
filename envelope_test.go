package gosfzplayer

import (
	"testing"
)

func TestEnvelopeInitialization(t *testing.T) {
	// Create a test region with ADSR opcodes
	region := &SfzSection{
		Type: "region",
		Opcodes: map[string]string{
			"ampeg_attack":  "0.5",  // 500ms attack
			"ampeg_decay":   "0.2",  // 200ms decay
			"ampeg_sustain": "75",   // 75% sustain level
			"ampeg_release": "1.0",  // 1000ms release
		},
	}

	// Create a test voice
	voice := &Voice{
		region: region,
	}

	// Initialize envelope with test sample rate
	sampleRate := uint32(44100)
	voice.InitializeEnvelope(sampleRate)

	// Verify envelope parameters
	expectedAttackSamples := 0.5 * float64(sampleRate)  // 22050 samples
	expectedDecaySamples := 0.2 * float64(sampleRate)   // 8820 samples
	expectedSustainLevel := 0.75                        // 75%
	expectedReleaseSamples := 1.0 * float64(sampleRate) // 44100 samples

	if voice.attackSamples != expectedAttackSamples {
		t.Errorf("Expected attack samples %f, got %f", expectedAttackSamples, voice.attackSamples)
	}

	if voice.decaySamples != expectedDecaySamples {
		t.Errorf("Expected decay samples %f, got %f", expectedDecaySamples, voice.decaySamples)
	}

	if voice.sustainLevel != expectedSustainLevel {
		t.Errorf("Expected sustain level %f, got %f", expectedSustainLevel, voice.sustainLevel)
	}

	if voice.releaseSamples != expectedReleaseSamples {
		t.Errorf("Expected release samples %f, got %f", expectedReleaseSamples, voice.releaseSamples)
	}

	// Verify initial envelope state
	if voice.envelopeState != EnvelopeAttack {
		t.Errorf("Expected initial envelope state to be Attack, got %v", voice.envelopeState)
	}

	if voice.envelopeLevel != 0.0 {
		t.Errorf("Expected initial envelope level to be 0.0, got %f", voice.envelopeLevel)
	}
}

func TestEnvelopeDefaults(t *testing.T) {
	// Create a test region with no ADSR opcodes (should use defaults)
	region := &SfzSection{
		Type:    "region",
		Opcodes: map[string]string{},
	}

	// Create a test voice
	voice := &Voice{
		region: region,
	}

	// Initialize envelope with test sample rate
	sampleRate := uint32(44100)
	voice.InitializeEnvelope(sampleRate)

	// Verify default values are applied
	if voice.envelopeState != EnvelopeAttack {
		t.Errorf("Expected initial envelope state to be Attack, got %v", voice.envelopeState)
	}

	if voice.sustainLevel != 1.0 {
		t.Errorf("Expected default sustain level to be 1.0, got %f", voice.sustainLevel)
	}

	// Verify envelope starts properly
	if voice.envelopeLevel != 0.0 {
		t.Errorf("Expected initial envelope level to be 0.0, got %f", voice.envelopeLevel)
	}
}

func TestEnvelopeProcessing(t *testing.T) {
	// Create a simple test region
	region := &SfzSection{
		Type: "region",
		Opcodes: map[string]string{
			"ampeg_attack":  "0.001", // Very short attack (44.1 samples)
			"ampeg_decay":   "0.001", // Very short decay  
			"ampeg_sustain": "50",    // 50% sustain
			"ampeg_release": "0.001", // Very short release
		},
	}

	voice := &Voice{
		region: region,
	}

	sampleRate := uint32(44100)
	voice.InitializeEnvelope(sampleRate)

	// Process envelope through attack phase
	initialLevel := voice.ProcessEnvelope()
	if initialLevel < 0.0 || initialLevel > 1.0 {
		t.Errorf("Envelope level should be between 0 and 1, got %f", initialLevel)
	}

	// Process several samples to advance through phases
	for i := 0; i < 200; i++ {
		level := voice.ProcessEnvelope()
		if level < 0.0 || level > 1.0 {
			t.Errorf("Envelope level should be between 0 and 1, got %f at sample %d", level, i)
		}
	}

	// Trigger release
	voice.TriggerRelease()
	if voice.envelopeState != EnvelopeRelease {
		t.Errorf("Expected envelope state to be Release after TriggerRelease, got %v", voice.envelopeState)
	}

	// Process release phase
	for i := 0; i < 100; i++ {
		level := voice.ProcessEnvelope()
		if level < 0.0 || level > 1.0 {
			t.Errorf("Envelope level should be between 0 and 1 during release, got %f at sample %d", level, i)
		}
	}
}

func TestEnvelopeDoesNotCrash(t *testing.T) {
	// Test that envelope processing doesn't crash with extreme values
	region := &SfzSection{
		Type: "region",
		Opcodes: map[string]string{
			"ampeg_attack":  "0",     // Instant attack
			"ampeg_decay":   "0",     // Instant decay
			"ampeg_sustain": "0",     // 0% sustain
			"ampeg_release": "0",     // Instant release
		},
	}

	voice := &Voice{
		region: region,
	}

	sampleRate := uint32(44100)
	voice.InitializeEnvelope(sampleRate)

	// Process envelope - should not crash
	for i := 0; i < 100; i++ {
		level := voice.ProcessEnvelope()
		if level < 0.0 || level > 1.0 {
			t.Errorf("Envelope level should be between 0 and 1, got %f", level)
		}
	}

	// Trigger release and continue processing
	voice.TriggerRelease()
	for i := 0; i < 100; i++ {
		level := voice.ProcessEnvelope()
		if level < 0.0 || level > 1.0 {
			t.Errorf("Envelope level should be between 0 and 1 during release, got %f", level)
		}
	}
}