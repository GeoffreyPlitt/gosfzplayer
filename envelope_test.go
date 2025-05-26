package gosfzplayer

import (
	"os"
	"testing"
)

func TestEnvelopeInitialization(t *testing.T) {
	// Create a test region with ADSR opcodes
	region := &SfzSection{
		Type: "region",
		Opcodes: map[string]string{
			"ampeg_attack":  "0.5", // 500ms attack
			"ampeg_decay":   "0.2", // 200ms decay
			"ampeg_sustain": "75",  // 75% sustain level
			"ampeg_release": "1.0", // 1000ms release
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
			"ampeg_attack":  "0", // Instant attack
			"ampeg_decay":   "0", // Instant decay
			"ampeg_sustain": "0", // 0% sustain
			"ampeg_release": "0", // Instant release
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

func TestEnvelopeAudioDemo(t *testing.T) {
	// Skip if piano samples not available
	if _, err := os.Stat("testdata/piano.sfz"); os.IsNotExist(err) {
		t.Skip("piano.sfz not found, run 'go generate' to download piano samples")
	}

	// Create SFZ content with pronounced envelope settings
	sfzContent := `<global>
ampeg_attack=0.5
ampeg_decay=0.8
ampeg_sustain=40
ampeg_release=1.2

<region>
sample=samples/C4vL.flac
key=60
`

	// Create SFZ file in testdata directory
	sfzPath := "testdata/envelope_demo.sfz"
	err := os.WriteFile(sfzPath, []byte(sfzContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create SFZ file: %v", err)
	}
	defer os.Remove(sfzPath)

	// Create SFZ player
	player, err := NewSfzPlayer(sfzPath, "")
	if err != nil {
		t.Fatalf("Failed to create SFZ player: %v", err)
	}
	defer player.StopAndClose()

	// Create mock client for rendering
	mockClient := createTestMockClient(player, 44100, 512)

	// Render a sequence of notes with different envelope phases
	sampleRate := 44100
	duration := 6.0 // 6 seconds to hear full envelope cycle
	totalSamples := int(float64(sampleRate) * duration)
	outputBuffer := make([]float32, totalSamples)

	// Timing: 2 seconds note on, 4 seconds release
	noteOnDuration := 2.0
	noteOnSamples := int(float64(sampleRate) * noteOnDuration)

	// Render audio
	bufferSize := 512
	currentSample := 0

	// Trigger note at start
	mockClient.noteOn(60, 100) // C4, velocity 100

	for currentSample < totalSamples {
		// Release note after 2 seconds
		if currentSample >= noteOnSamples {
			mockClient.noteOff(60)
		}

		framesToRender := bufferSize
		if currentSample+bufferSize > totalSamples {
			framesToRender = totalSamples - currentSample
		}

		audioBuffer := make([]float32, framesToRender)
		mockClient.renderVoices(audioBuffer, uint32(framesToRender))

		copy(outputBuffer[currentSample:currentSample+framesToRender], audioBuffer)
		currentSample += framesToRender
	}

	// Save as WAV file
	outputPath := "testdata/envelope_demo.wav"
	err = saveWAV(outputPath, outputBuffer, sampleRate)
	if err != nil {
		t.Fatalf("Failed to save WAV file: %v", err)
	}

	t.Logf("Generated envelope demo: %s (%.1f seconds)", outputPath, duration)
	t.Logf("Envelope phases: Attack=0.5s, Decay=0.8s, Sustain=40%%, Release=1.2s")
}
