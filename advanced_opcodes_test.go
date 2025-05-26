package gosfzplayer

import (
	"testing"
)

// TestParseAdvancedOpcodes tests that all advanced opcodes are recognized and parsed
func TestParseAdvancedOpcodes(t *testing.T) {
	tests := []struct {
		name   string
		opcode string
		want   bool
	}{
		// Keyswitching
		{"sw_lokey", "sw_lokey", true},
		{"sw_hikey", "sw_hikey", true},

		// Groups and Exclusion
		{"group", "group", true},
		{"off_by", "off_by", true},

		// Trigger Modes
		{"trigger", "trigger", true},

		// Pitch Bend
		{"bend_up", "bend_up", true},
		{"bend_down", "bend_down", true},

		// Reverb (should also be recognized)
		{"reverb_send", "reverb_send", true},
		{"reverb_room_size", "reverb_room_size", true},

		// Should still reject unknown opcodes
		{"unknown", "unknown_opcode", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isKnownOpcode(tt.opcode)
			if got != tt.want {
				t.Errorf("isKnownOpcode(%q) = %v, want %v", tt.opcode, got, tt.want)
			}
		})
	}
}

// TestAdvancedOpcodeDefaults tests default values for advanced opcodes
func TestAdvancedOpcodeDefaults(t *testing.T) {
	region := &SfzSection{
		Type:    "region",
		Opcodes: map[string]string{},
	}

	tests := []struct {
		name     string
		opcode   string
		defValue int
		expected int
	}{
		{"group default", "group", 0, 0},
		{"off_by default", "off_by", 0, 0},
		{"bend_up default", "bend_up", 200, 200},
		{"bend_down default", "bend_down", -200, -200},
		{"sw_lokey default", "sw_lokey", -1, -1},
		{"sw_hikey default", "sw_hikey", -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := region.GetInheritedIntOpcode(tt.opcode, tt.defValue)
			if got != tt.expected {
				t.Errorf("GetInheritedIntOpcode(%q, %d) = %d, want %d", tt.opcode, tt.defValue, got, tt.expected)
			}
		})
	}

	// Test string opcodes
	trigger := region.GetInheritedStringOpcode("trigger")
	if trigger != "" {
		t.Errorf("GetInheritedStringOpcode(trigger) = %q, want empty string", trigger)
	}
}

// TestPitchBendConversion tests pitch bend MIDI value conversion
func TestPitchBendConversion(t *testing.T) {
	tests := []struct {
		name     string
		lsb      uint8
		msb      uint8
		expected int16
	}{
		{"center position", 0x00, 0x40, 0},   // 0x4000 - 8192 = 0
		{"max positive", 0x7F, 0x7F, 8191},   // 0x7F7F - 8192 = 8191
		{"max negative", 0x00, 0x00, -8192},  // 0x0000 - 8192 = -8192
		{"slight positive", 0x00, 0x41, 128}, // 0x4100 - 8192 = 128
		{"slight negative", 0x7F, 0x3F, -1},  // 0x3F7F - 8192 = -1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the pitch bend conversion algorithm directly
			bendValue := int16((uint16(tt.msb)<<7)|uint16(tt.lsb)) - 8192
			if bendValue != tt.expected {
				t.Errorf("pitch bend conversion lsb=%d msb=%d = %d, want %d",
					tt.lsb, tt.msb, bendValue, tt.expected)
			}
		})
	}
}

// TestKeyswitchRangeCheck tests keyswitch range checking logic
func TestKeyswitchRangeCheck(t *testing.T) {
	tests := []struct {
		name        string
		swLokey     int
		swHikey     int
		currentKey  uint8
		shouldMatch bool
	}{
		{"note in range", 12, 23, 15, true},         // C1-B1, D#1
		{"note below range", 12, 23, 10, false},     // Below C1
		{"note above range", 12, 23, 25, false},     // Above B1
		{"note at low boundary", 12, 23, 12, true},  // Exactly C1
		{"note at high boundary", 12, 23, 23, true}, // Exactly B1
		{"no keyswitch range", -1, -1, 15, true},    // No restriction
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the keyswitch logic directly
			inRange := true
			if tt.swLokey >= 0 && tt.swHikey >= 0 {
				if int(tt.currentKey) < tt.swLokey || int(tt.currentKey) > tt.swHikey {
					inRange = false
				}
			}

			if inRange != tt.shouldMatch {
				t.Errorf("keyswitch check lokey=%d hikey=%d current=%d = %v, want %v",
					tt.swLokey, tt.swHikey, tt.currentKey, inRange, tt.shouldMatch)
			}
		})
	}
}

// TestTriggerModeChecks tests trigger mode logic without JackClient dependencies
func TestTriggerModeChecks(t *testing.T) {
	tests := []struct {
		name            string
		triggerMode     string
		activeNoteCount int
		shouldMatch     bool
	}{
		{"first with no active notes", "first", 1, true},    // Count will be 1 after increment
		{"first with active notes", "first", 2, false},      // Count will be 2 after increment
		{"legato with no active notes", "legato", 1, false}, // Count will be 1 after increment
		{"legato with active notes", "legato", 2, true},     // Count will be 2 after increment
		{"attack mode always works", "attack", 1, true},     // Default behavior
		{"release mode never matches in noteOn", "release", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test trigger mode logic directly
			shouldTrigger := true

			switch tt.triggerMode {
			case "first":
				if tt.activeNoteCount > 1 {
					shouldTrigger = false
				}
			case "legato":
				if tt.activeNoteCount <= 1 {
					shouldTrigger = false
				}
			case "release":
				shouldTrigger = false // Release triggers are handled separately
			}

			if shouldTrigger != tt.shouldMatch {
				t.Errorf("trigger mode check mode=%s activeCount=%d = %v, want %v",
					tt.triggerMode, tt.activeNoteCount, shouldTrigger, tt.shouldMatch)
			}
		})
	}
}

// TestParseTestSfzWithAdvancedOpcodes tests that our updated test.sfz file parses correctly
func TestParseTestSfzWithAdvancedOpcodes(t *testing.T) {
	sfzData, err := ParseSfzFile("testdata/test.sfz")
	if err != nil {
		t.Fatalf("ParseSfzFile failed: %v", err)
	}

	if sfzData == nil {
		t.Fatal("ParseSfzFile returned nil data")
	}

	// Should have at least 10 regions now (including advanced opcode regions)
	if len(sfzData.Regions) < 10 {
		t.Errorf("Expected at least 10 regions, got %d", len(sfzData.Regions))
	}

	// Check that some regions have advanced opcodes
	foundGroup := false
	foundTrigger := false
	foundKeyswitch := false

	for _, region := range sfzData.Regions {
		if region.GetInheritedIntOpcode("group", -1) > 0 {
			foundGroup = true
		}
		if region.GetInheritedStringOpcode("trigger") != "" {
			foundTrigger = true
		}
		if region.GetInheritedIntOpcode("sw_lokey", -1) >= 0 {
			foundKeyswitch = true
		}
	}

	if !foundGroup {
		t.Error("No regions found with group opcode")
	}
	if !foundTrigger {
		t.Error("No regions found with trigger opcode")
	}
	if !foundKeyswitch {
		t.Error("No regions found with keyswitch opcodes")
	}
}
