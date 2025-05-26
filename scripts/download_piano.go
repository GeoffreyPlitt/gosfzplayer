package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Upright Piano KW samples from FreePats project
// Source: https://github.com/freepats/upright-piano-KW
// License: Creative Commons CC0 1.0 Public Domain Dedication
const (
	repoCommit = "main" // Using main branch - CC0 licensed samples
	baseURL    = "https://raw.githubusercontent.com/freepats/upright-piano-KW/" + repoCommit + "/samples/"
	targetDir  = "testdata/samples"
)

// File definitions with expected checksums for verification
type pianoFile struct {
	name string
	url  string
	// Checksums will be filled in after initial download
	sha256 string
}

// Only the samples actually used in the SFZ file
var pianoFiles = []pianoFile{
	{"A0vH.flac", baseURL + "A0vH.flac", ""},
	{"A0vL.flac", baseURL + "A0vL.flac", ""},
	{"A1vH.flac", baseURL + "A1vH.flac", ""},
	{"A1vL.flac", baseURL + "A1vL.flac", ""},
	{"A2vL.flac", baseURL + "A2vL.flac", ""},
	{"A3vH.flac", baseURL + "A3vH.flac", ""},
	{"A3vL.flac", baseURL + "A3vL.flac", ""},
	{"A4vH.flac", baseURL + "A4vH.flac", ""},
	{"A4vL.flac", baseURL + "A4vL.flac", ""},
	{"A5vH.flac", baseURL + "A5vH.flac", ""},
	{"A5vL.flac", baseURL + "A5vL.flac", ""},
	{"A6vH.flac", baseURL + "A6vH.flac", ""},
	{"A6vL.flac", baseURL + "A6vL.flac", ""},
	{"A7vH.flac", baseURL + "A7vH.flac", ""},
	{"A7vL.flac", baseURL + "A7vL.flac", ""},
	{"B0vH.flac", baseURL + "B0vH.flac", ""},
	{"B1vH.flac", baseURL + "B1vH.flac", ""},
	{"B2vH.flac", baseURL + "B2vH.flac", ""},
	{"B3vH.flac", baseURL + "B3vH.flac", ""},
	{"B4vH.flac", baseURL + "B4vH.flac", ""},
	{"B5vH.flac", baseURL + "B5vH.flac", ""},
	{"B6vH.flac", baseURL + "B6vH.flac", ""},
	{"B7vH.flac", baseURL + "B7vH.flac", ""},
	{"C1vH.flac", baseURL + "C1vH.flac", ""},
	{"C1vL.flac", baseURL + "C1vL.flac", ""},
	{"C2vH.flac", baseURL + "C2vH.flac", ""},
	{"C2vL.flac", baseURL + "C2vL.flac", ""},
	{"C3vH.flac", baseURL + "C3vH.flac", ""},
	{"C3vL.flac", baseURL + "C3vL.flac", ""},
	{"C4vL.flac", baseURL + "C4vL.flac", ""},
	{"C5vH.flac", baseURL + "C5vH.flac", ""},
	{"C5vL.flac", baseURL + "C5vL.flac", ""},
	{"C6vH.flac", baseURL + "C6vH.flac", ""},
	{"C6vL.flac", baseURL + "C6vL.flac", ""},
	{"C7vH.flac", baseURL + "C7vH.flac", ""},
	{"C7vL.flac", baseURL + "C7vL.flac", ""},
	{"C8vH.flac", baseURL + "C8vH.flac", ""},
	{"C8vL.flac", baseURL + "C8vL.flac", ""},
	{"D#1vH.flac", baseURL + "D%231vH.flac", ""},
	{"D#1vL.flac", baseURL + "D%231vL.flac", ""},
	{"D#2vH.flac", baseURL + "D%232vH.flac", ""},
	{"D#2vL.flac", baseURL + "D%232vL.flac", ""},
	{"D#3vH.flac", baseURL + "D%233vH.flac", ""},
	{"D#3vL.flac", baseURL + "D%233vL.flac", ""},
	{"D#4vH.flac", baseURL + "D%234vH.flac", ""},
	{"D#4vL.flac", baseURL + "D%234vL.flac", ""},
	{"D#5vH.flac", baseURL + "D%235vH.flac", ""},
	{"D#5vL.flac", baseURL + "D%235vL.flac", ""},
	{"D#6vH.flac", baseURL + "D%236vH.flac", ""},
	{"D#6vL.flac", baseURL + "D%236vL.flac", ""},
	{"D#7vH.flac", baseURL + "D%237vH.flac", ""},
	{"D#7vL.flac", baseURL + "D%237vL.flac", ""},
	{"F#1vH.flac", baseURL + "F%231vH.flac", ""},
	{"F#1vL.flac", baseURL + "F%231vL.flac", ""},
	{"F#2vH.flac", baseURL + "F%232vH.flac", ""},
	{"F#2vL.flac", baseURL + "F%232vL.flac", ""},
	{"F#3vH.flac", baseURL + "F%233vH.flac", ""},
	{"F#3vL.flac", baseURL + "F%233vL.flac", ""},
	{"F#4vH.flac", baseURL + "F%234vH.flac", ""},
	{"F#4vL.flac", baseURL + "F%234vL.flac", ""},
	{"F#5vH.flac", baseURL + "F%235vH.flac", ""},
	{"F#5vL.flac", baseURL + "F%235vL.flac", ""},
	{"F#6vH.flac", baseURL + "F%236vH.flac", ""},
	{"F#6vL.flac", baseURL + "F%236vL.flac", ""},
	{"F#7vH.flac", baseURL + "F%237vH.flac", ""},
	{"F#7vL.flac", baseURL + "F%237vL.flac", ""},
}

func main() {
	fmt.Println("Downloading Upright Piano KW samples (CC0 licensed)...")

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Printf("Error creating directory %s: %v\n", targetDir, err)
		os.Exit(1)
	}

	// Download each file
	for _, file := range pianoFiles {
		targetPath := filepath.Join(targetDir, file.name)

		// Skip if file already exists
		if _, err := os.Stat(targetPath); err == nil {
			fmt.Printf("  %s already exists, skipping\n", file.name)
			continue
		}

		fmt.Printf("  Downloading %s...", file.name)

		if err := downloadFile(file.url, targetPath); err != nil {
			fmt.Printf(" FAILED: %v\n", err)
			os.Exit(1)
		}

		// Verify checksum if provided
		if file.sha256 != "" {
			if err := verifyChecksum(targetPath, file.sha256); err != nil {
				fmt.Printf(" CHECKSUM FAILED: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Printf(" OK\n")
	}

	fmt.Printf("Successfully downloaded %d piano samples to %s\n", len(pianoFiles), targetDir)
}

func downloadFile(url, targetPath string) error {
	// Create HTTP client with timeout
	client := &http.Client{}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create target file
	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetPath, err)
	}
	defer out.Close()

	// Copy content
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", targetPath, err)
	}

	return nil
}

func verifyChecksum(filePath, expectedSHA256 string) error {
	if expectedSHA256 == "" {
		return nil // No checksum to verify
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actualSHA256 := fmt.Sprintf("%x", hash.Sum(nil))
	if actualSHA256 != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
	}

	return nil
}
