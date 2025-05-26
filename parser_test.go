package gosfzplayer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSfzFile(t *testing.T) {
	// Test parsing the comprehensive test.sfz file
	testPath := filepath.Join("testdata", "test.sfz")

	sfzData, err := ParseSfzFile(testPath)
	if err != nil {
		t.Fatalf("Failed to parse test.sfz: %v", err)
	}

	// Verify global section
	if sfzData.Global == nil {
		t.Fatal("Expected global section to be parsed")
	}

	if sfzData.Global.Type != "global" {
		t.Errorf("Expected global section type to be 'global', got '%s'", sfzData.Global.Type)
	}

	// Check global opcodes
	expectedGlobalOpcodes := map[string]string{
		"volume": "-6.0",
		"tune":   "+10",
		"pan":    "0",
	}

	for opcode, expectedValue := range expectedGlobalOpcodes {
		if value := sfzData.Global.GetStringOpcode(opcode); value != expectedValue {
			t.Errorf("Expected global %s to be '%s', got '%s'", opcode, expectedValue, value)
		}
	}

	// Verify groups (we now have 4 groups with advanced opcodes)
	if len(sfzData.Groups) != 4 {
		t.Errorf("Expected 4 groups, got %d", len(sfzData.Groups))
	}

	// Check first group opcodes
	if len(sfzData.Groups) > 0 {
		group1 := sfzData.Groups[0]
		expectedGroup1Opcodes := map[string]string{
			"transpose":     "0",
			"pitch":         "100",
			"ampeg_attack":  "0.01",
			"ampeg_decay":   "0.1",
			"ampeg_sustain": "80",
			"ampeg_release": "0.2",
		}

		for opcode, expectedValue := range expectedGroup1Opcodes {
			if value := group1.GetStringOpcode(opcode); value != expectedValue {
				t.Errorf("Expected group1 %s to be '%s', got '%s'", opcode, expectedValue, value)
			}
		}
	}

	// Verify regions (we now have 10 regions with advanced opcodes)
	if len(sfzData.Regions) != 10 {
		t.Errorf("Expected 10 regions, got %d", len(sfzData.Regions))
	}

	// Test first region in detail
	if len(sfzData.Regions) > 0 {
		region1 := sfzData.Regions[0]

		expectedRegion1Opcodes := map[string]string{
			"sample":          "sample1.wav",
			"lokey":           "c2",
			"hikey":           "c4",
			"lovel":           "1",
			"hivel":           "64",
			"key":             "c3",
			"pitch_keycenter": "c3",
			"volume":          "0.0",
			"loop_mode":       "no_loop",
		}

		for opcode, expectedValue := range expectedRegion1Opcodes {
			if value := region1.GetStringOpcode(opcode); value != expectedValue {
				t.Errorf("Expected region1 %s to be '%s', got '%s'", opcode, expectedValue, value)
			}
		}
	}

	// Test second region for different opcodes
	if len(sfzData.Regions) > 1 {
		region2 := sfzData.Regions[1]

		expectedRegion2Opcodes := map[string]string{
			"sample":          "sample2.wav",
			"key":             "d3",
			"lovel":           "65",
			"hivel":           "127",
			"pitch_keycenter": "d3",
			"volume":          "-3.0",
			"pan":             "-50",
			"tune":            "-20",
			"loop_mode":       "loop_continuous",
			"loop_start":      "1000",
			"loop_end":        "8000",
		}

		for opcode, expectedValue := range expectedRegion2Opcodes {
			if value := region2.GetStringOpcode(opcode); value != expectedValue {
				t.Errorf("Expected region2 %s to be '%s', got '%s'", opcode, expectedValue, value)
			}
		}
	}
}

func TestParseSfzFileNotFound(t *testing.T) {
	_, err := ParseSfzFile("nonexistent.sfz")
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

func TestGetOpcodeHelpers(t *testing.T) {
	section := &SfzSection{
		Type: "region",
		Opcodes: map[string]string{
			"volume":  "-6.5",
			"lokey":   "60",
			"sample":  "test.wav",
			"invalid": "not_a_number",
		},
	}

	// Test GetStringOpcode
	if value := section.GetStringOpcode("sample"); value != "test.wav" {
		t.Errorf("Expected 'test.wav', got '%s'", value)
	}

	if value := section.GetStringOpcode("nonexistent"); value != "" {
		t.Errorf("Expected empty string for nonexistent opcode, got '%s'", value)
	}

	// Test GetFloatOpcode
	if value := section.GetFloatOpcode("volume", 0.0); value != -6.5 {
		t.Errorf("Expected -6.5, got %f", value)
	}

	if value := section.GetFloatOpcode("nonexistent", 99.9); value != 99.9 {
		t.Errorf("Expected default value 99.9, got %f", value)
	}

	if value := section.GetFloatOpcode("invalid", 0.0); value != 0.0 {
		t.Errorf("Expected default value 0.0 for invalid float, got %f", value)
	}

	// Test GetIntOpcode
	if value := section.GetIntOpcode("lokey", 0); value != 60 {
		t.Errorf("Expected 60, got %d", value)
	}

	if value := section.GetIntOpcode("nonexistent", 42); value != 42 {
		t.Errorf("Expected default value 42, got %d", value)
	}

	if value := section.GetIntOpcode("invalid", 0); value != 0 {
		t.Errorf("Expected default value 0 for invalid int, got %d", value)
	}
}

func TestNilSectionHelpers(t *testing.T) {
	var section *SfzSection

	// Test that helper functions handle nil sections gracefully
	if value := section.GetStringOpcode("test"); value != "" {
		t.Errorf("Expected empty string for nil section, got '%s'", value)
	}

	if value := section.GetFloatOpcode("test", 5.5); value != 5.5 {
		t.Errorf("Expected default value 5.5 for nil section, got %f", value)
	}

	if value := section.GetIntOpcode("test", 10); value != 10 {
		t.Errorf("Expected default value 10 for nil section, got %d", value)
	}
}

func TestUnknownOpcodes(t *testing.T) {
	// Create a temporary SFZ file with unknown opcodes
	tempFile, err := os.CreateTemp("", "test_unknown_*.sfz")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := `<region>
sample=test.wav
unknown_opcode=value
valid_opcode=volume
another_unknown=123
`

	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tempFile.Close()

	// Parse the file - should succeed but log warnings for unknown opcodes
	sfzData, err := ParseSfzFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Parser should handle unknown opcodes gracefully: %v", err)
	}

	// Should have one region
	if len(sfzData.Regions) != 1 {
		t.Errorf("Expected 1 region, got %d", len(sfzData.Regions))
	}

	// Should have parsed known opcodes but not unknown ones
	region := sfzData.Regions[0]
	if value := region.GetStringOpcode("sample"); value != "test.wav" {
		t.Errorf("Expected known opcode to be parsed, got '%s'", value)
	}

	// Unknown opcodes should not be stored
	if value := region.GetStringOpcode("unknown_opcode"); value != "" {
		t.Errorf("Unknown opcode should not be stored, got '%s'", value)
	}
}

func TestEmptyAndCommentLines(t *testing.T) {
	// Create a temporary SFZ file with empty lines and comments
	tempFile, err := os.CreateTemp("", "test_comments_*.sfz")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := `// This is a comment
<region>

// Another comment
sample=test.wav   // Inline comment
volume=-6.0

// Final comment
`

	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tempFile.Close()

	// Parse the file
	sfzData, err := ParseSfzFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse file with comments: %v", err)
	}

	// Should have one region with correct opcodes
	if len(sfzData.Regions) != 1 {
		t.Errorf("Expected 1 region, got %d", len(sfzData.Regions))
	}

	region := sfzData.Regions[0]
	if value := region.GetStringOpcode("sample"); value != "test.wav" {
		t.Errorf("Expected 'test.wav', got '%s'", value)
	}

	if value := region.GetFloatOpcode("volume", 0.0); value != -6.0 {
		t.Errorf("Expected -6.0, got %f", value)
	}
}
