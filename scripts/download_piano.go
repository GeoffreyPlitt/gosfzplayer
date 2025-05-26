package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Upright Piano KW samples from FreePats project
// Source: https://github.com/freepats/upright-piano-KW
// License: Creative Commons CC0 1.0 Public Domain Dedication
const (
	repoCommit = "main" // Using main branch - CC0 licensed samples
	baseURL    = "https://raw.githubusercontent.com/freepats/upright-piano-KW/" + repoCommit + "/samples/"
	sfzFile    = "testdata/piano.sfz"
	targetDir  = "testdata/samples"
)

func main() {
	fmt.Println("Downloading Upright Piano KW samples (CC0 licensed)...")

	// Parse SFZ file to find required samples
	samples, err := findSamplesInSfz(sfzFile)
	if err != nil {
		fmt.Printf("Error parsing SFZ file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d unique samples in %s\n", len(samples), sfzFile)

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Printf("Error creating directory %s: %v\n", targetDir, err)
		os.Exit(1)
	}

	// Download each sample
	downloaded := 0
	for _, sample := range samples {
		// Extract just the filename from the sample path
		filename := filepath.Base(sample)
		targetPath := filepath.Join(targetDir, filename)

		// Skip if file already exists
		if _, err := os.Stat(targetPath); err == nil {
			fmt.Printf("  %s already exists, skipping\n", filename)
			continue
		}

		// URL encode the filename for download (handle # characters)
		encodedFilename := strings.ReplaceAll(filename, "#", "%23")
		downloadURL := baseURL + encodedFilename

		fmt.Printf("  Downloading %s...", filename)

		if err := downloadFile(downloadURL, targetPath); err != nil {
			fmt.Printf(" FAILED: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf(" OK\n")
		downloaded++
	}

	fmt.Printf("Successfully downloaded %d new samples to %s\n", downloaded, targetDir)
}

// findSamplesInSfz parses an SFZ file and extracts all unique sample paths
func findSamplesInSfz(sfzPath string) ([]string, error) {
	file, err := os.Open(sfzPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SFZ file: %w", err)
	}
	defer file.Close()

	var samples []string
	sampleSet := make(map[string]bool) // To avoid duplicates
	sampleRegex := regexp.MustCompile(`^\s*sample\s*=\s*(.+)$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "//") || line == "" {
			continue
		}

		// Look for sample= lines
		if matches := sampleRegex.FindStringSubmatch(line); matches != nil {
			samplePath := strings.TrimSpace(matches[1])

			// Only process samples in the samples/ directory
			if strings.HasPrefix(samplePath, "samples/") {
				if !sampleSet[samplePath] {
					samples = append(samples, samplePath)
					sampleSet[samplePath] = true
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SFZ file: %w", err)
	}

	return samples, nil
}

func downloadFile(url, targetPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", targetPath, err)
	}

	return nil
}
