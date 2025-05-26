package gosfzplayer

import (
	"testing"
)

func TestGlobalGroupInheritance(t *testing.T) {
	// Create a test SFZ structure with Global → Group → Region inheritance
	global := &SfzSection{
		Type: "global",
		Opcodes: map[string]string{
			"volume":     "-6.0",   // Global volume
			"tune":       "+10",    // Global tune
			"transpose":  "0",      // Global transpose
			"ampeg_attack": "0.1",  // Global attack
		},
	}

	group := &SfzSection{
		Type: "group",
		Opcodes: map[string]string{
			"volume":       "-3.0",  // Override global volume
			"pitch":        "+50",   // Group-specific pitch
			"ampeg_decay":  "0.2",   // Group-specific decay
		},
		GlobalRef: global,
	}

	region := &SfzSection{
		Type: "region",
		Opcodes: map[string]string{
			"pitch_keycenter": "60", // Region-specific pitch_keycenter
			"ampeg_release":   "0.5", // Region-specific release
		},
		ParentGroup: group,
		GlobalRef:   global,
	}

	// Test inheritance order: Region → Group → Global
	tests := []struct {
		name     string
		opcode   string
		expected interface{}
		method   string
	}{
		// Volume: Group overrides Global
		{"volume inheritance", "volume", -3.0, "float"},
		
		// Tune: Only in Global
		{"tune inheritance", "tune", 10.0, "float"},
		
		// Pitch: Only in Group
		{"pitch inheritance", "pitch", 50.0, "float"},
		
		// Transpose: Only in Global
		{"transpose inheritance", "transpose", 0, "int"},
		
		// Pitch_keycenter: Only in Region
		{"pitch_keycenter inheritance", "pitch_keycenter", 60, "int"},
		
		// ADSR: Mixed inheritance
		{"ampeg_attack inheritance", "ampeg_attack", 0.1, "float"},   // From Global
		{"ampeg_decay inheritance", "ampeg_decay", 0.2, "float"},     // From Group  
		{"ampeg_release inheritance", "ampeg_release", 0.5, "float"}, // From Region
		{"ampeg_sustain inheritance", "ampeg_sustain", 80.0, "float"}, // Default (not defined anywhere)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			switch test.method {
			case "float":
				expected := test.expected.(float64)
				actual := region.GetInheritedFloatOpcode(test.opcode, 80.0) // Default for sustain
				if actual != expected {
					t.Errorf("Expected %s = %f, got %f", test.opcode, expected, actual)
				}

			case "int":
				expected := test.expected.(int)
				actual := region.GetInheritedIntOpcode(test.opcode, -1)
				if actual != expected {
					t.Errorf("Expected %s = %d, got %d", test.opcode, expected, actual)
				}

			case "string":
				expected := test.expected.(string)
				actual := region.GetInheritedStringOpcode(test.opcode)
				if actual != expected {
					t.Errorf("Expected %s = %s, got %s", test.opcode, expected, actual)
				}
			}
		})
	}
}

func TestInheritanceWithoutParents(t *testing.T) {
	// Test region without group or global
	region := &SfzSection{
		Type: "region",
		Opcodes: map[string]string{
			"volume": "3.0",
		},
	}

	// Should use region value
	if volume := region.GetInheritedFloatOpcode("volume", 0.0); volume != 3.0 {
		t.Errorf("Expected volume = 3.0, got %f", volume)
	}

	// Should use default for missing opcode
	if tune := region.GetInheritedFloatOpcode("tune", -5.0); tune != -5.0 {
		t.Errorf("Expected tune default = -5.0, got %f", tune)
	}
}

func TestInheritanceGroupOnly(t *testing.T) {
	// Test region with group but no global
	group := &SfzSection{
		Type: "group",
		Opcodes: map[string]string{
			"transpose": "12",
		},
	}

	region := &SfzSection{
		Type: "region",
		Opcodes: map[string]string{
			"volume": "-1.0",
		},
		ParentGroup: group,
	}

	// Should inherit from group
	if transpose := region.GetInheritedIntOpcode("transpose", 0); transpose != 12 {
		t.Errorf("Expected transpose = 12, got %d", transpose)
	}

	// Should use region value
	if volume := region.GetInheritedFloatOpcode("volume", 0.0); volume != -1.0 {
		t.Errorf("Expected volume = -1.0, got %f", volume)
	}
}

func TestComplexPitchCalculation(t *testing.T) {
	// Test all pitch opcodes working together
	global := &SfzSection{
		Type: "global",
		Opcodes: map[string]string{
			"tune": "+20", // +20 cents
		},
	}

	group := &SfzSection{
		Type: "group",
		Opcodes: map[string]string{
			"transpose": "12", // +1 octave (12 semitones)
			"pitch":     "-10", // -10 cents
		},
		GlobalRef: global,
	}

	region := &SfzSection{
		Type: "region",
		Opcodes: map[string]string{
			"pitch_keycenter": "60", // Middle C
		},
		ParentGroup: group,
		GlobalRef:   global,
	}

	// Create mock client to test pitch calculation
	player := &SfzPlayer{}
	mockClient := &MockJackClient{player: player}

	// Test MIDI note 72 (C5) with pitch_keycenter=60 (C4)
	// Expected calculation:
	// - Base semitones: 72 - 60 = 12 semitones (1 octave up)
	// - Transpose: +12 semitones (another octave up) 
	// - Tune: +20 cents = +0.2 semitones
	// - Pitch: -10 cents = -0.1 semitones
	// - Total: 12 + 12 + 0.2 - 0.1 = 24.1 semitones
	// - Ratio: 2^(24.1/12) ≈ 4.014 (about 4x = 2 octaves)

	ratio := mockClient.calculatePitchRatio(region, 72)
	expectedRatio := 4.014 // Approximately 2^(24.1/12)

	if ratio < expectedRatio-0.1 || ratio > expectedRatio+0.1 {
		t.Errorf("Expected pitch ratio ≈ %f, got %f", expectedRatio, ratio)
	}

	t.Logf("Pitch calculation test: note=72, keycenter=60, transpose=+12, tune=+20c, pitch=-10c → ratio=%f", ratio)
}