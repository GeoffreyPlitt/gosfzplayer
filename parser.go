package gosfzplayer

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/GeoffreyPlitt/debuggo"
)

var parserDebug = debuggo.Debug("sfzplayer:parser")

// SfzData represents the parsed SFZ file structure
type SfzData struct {
	Global  *SfzSection
	Groups  []*SfzSection
	Regions []*SfzSection
}

// SfzSection represents a section in the SFZ file (global, group, or region)
type SfzSection struct {
	Type    string            // "global", "group", or "region"
	Opcodes map[string]string // opcode name -> value
}

// ParseSfzFile parses an SFZ file and returns the structured data
func ParseSfzFile(filePath string) (*SfzData, error) {
	parserDebug("Starting to parse SFZ file: %s", filePath)

	// Check if file exists
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SFZ file: %w", err)
	}
	defer file.Close()

	sfzData := &SfzData{
		Groups:  make([]*SfzSection, 0),
		Regions: make([]*SfzSection, 0),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var currentSection *SfzSection

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		parserDebug("Parsing line %d: %s", lineNum, line)

		// Check for section headers
		if strings.HasPrefix(line, "<") && strings.HasSuffix(line, ">") {
			sectionType := strings.ToLower(strings.Trim(line, "<>"))
			parserDebug("Found section: %s", sectionType)
			
			currentSection = &SfzSection{
				Type:    sectionType,
				Opcodes: make(map[string]string),
			}

			switch sectionType {
			case "global":
				sfzData.Global = currentSection
			case "group":
				sfzData.Groups = append(sfzData.Groups, currentSection)
			case "region":
				sfzData.Regions = append(sfzData.Regions, currentSection)
			default:
				parserDebug("Warning: Unknown section type: %s", sectionType)
			}
			continue
		}

		// Parse opcodes
		if currentSection != nil {
			err := parseOpcodes(line, currentSection, lineNum)
			if err != nil {
				parserDebug("Warning: Failed to parse line %d: %v", lineNum, err)
			}
		} else {
			parserDebug("Warning: Opcode found outside of section at line %d: %s", lineNum, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SFZ file: %w", err)
	}

	parserDebug("Parsing complete. Found %d regions, %d groups", len(sfzData.Regions), len(sfzData.Groups))
	return sfzData, nil
}

// parseOpcodes parses a line containing opcodes and adds them to the section
func parseOpcodes(line string, section *SfzSection, lineNum int) error {
	// Split line by whitespace to get individual opcodes
	parts := strings.Fields(line)
	
	for _, part := range parts {
		// Skip comments that might appear inline
		if strings.HasPrefix(part, "//") {
			break
		}
		
		// Find the = separator
		equalIndex := strings.Index(part, "=")
		if equalIndex == -1 {
			continue // Skip parts without =
		}
		
		opcode := strings.ToLower(strings.TrimSpace(part[:equalIndex]))
		value := strings.TrimSpace(part[equalIndex+1:])
		
		// Validate and store the opcode
		if isKnownOpcode(opcode) {
			section.Opcodes[opcode] = value
			parserDebug("Parsed opcode: %s = %s", opcode, value)
		} else {
			parserDebug("Warning: Unknown opcode '%s' at line %d", opcode, lineNum)
		}
	}
	
	return nil
}

// isKnownOpcode checks if an opcode is in our supported list
func isKnownOpcode(opcode string) bool {
	knownOpcodes := map[string]bool{
		// Critical Core
		"sample": true,
		
		// Key/Velocity Mapping
		"lokey":  true,
		"hikey":  true,
		"lovel":  true,
		"hivel":  true,
		"key":    true,
		
		// Basic Playback
		"volume":          true,
		"pitch_keycenter": true,
		
		// Envelope
		"ampeg_attack":  true,
		"ampeg_decay":   true,
		"ampeg_sustain": true,
		"ampeg_release": true,
		
		// Common Adjustments
		"tune":      true,
		"pan":       true,
		"transpose": true,
		"pitch":     true,
		
		// Looping
		"loop_mode":  true,
		"loop_start": true,
		"loop_end":   true,
	}
	
	return knownOpcodes[opcode]
}

// Helper functions to extract specific opcode values with type conversion

// GetStringOpcode returns a string opcode value, or empty string if not found
func (s *SfzSection) GetStringOpcode(opcode string) string {
	if s == nil || s.Opcodes == nil {
		return ""
	}
	return s.Opcodes[opcode]
}

// GetIntOpcode returns an integer opcode value, or defaultValue if not found or invalid
func (s *SfzSection) GetIntOpcode(opcode string, defaultValue int) int {
	if s == nil || s.Opcodes == nil {
		return defaultValue
	}
	
	value, exists := s.Opcodes[opcode]
	if !exists {
		return defaultValue
	}
	
	intVal, err := strconv.Atoi(value)
	if err != nil {
		parserDebug("Warning: Invalid integer value for opcode %s: %s", opcode, value)
		return defaultValue
	}
	
	return intVal
}

// GetFloatOpcode returns a float opcode value, or defaultValue if not found or invalid
func (s *SfzSection) GetFloatOpcode(opcode string, defaultValue float64) float64 {
	if s == nil || s.Opcodes == nil {
		return defaultValue
	}
	
	value, exists := s.Opcodes[opcode]
	if !exists {
		return defaultValue
	}
	
	floatVal, err := strconv.ParseFloat(value, 64)
	if err != nil {
		parserDebug("Warning: Invalid float value for opcode %s: %s", opcode, value)
		return defaultValue
	}
	
	return floatVal
}