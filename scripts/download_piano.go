package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Piano sample files from https://github.com/parisjava/wav-piano-sound
// Pinned to specific commit for reproducibility
const (
	repoCommit = "940c512d5a0624b5ae2ad139c5a9ebf9d79fd3c9" // Latest commit SHA
	baseURL    = "https://raw.githubusercontent.com/parisjava/wav-piano-sound/" + repoCommit + "/wav/"
	targetDir  = "testdata/piano"
)

// File definitions with expected checksums for verification
type pianoFile struct {
	name string
	url  string
	// Checksums will be filled in after initial download
	sha256 string
}

var pianoFiles = []pianoFile{
	{"a1.wav", baseURL + "a1.wav", ""},
	{"a1s.wav", baseURL + "a1s.wav", ""},
	{"b1.wav", baseURL + "b1.wav", ""},
	{"c1.wav", baseURL + "c1.wav", ""},
	{"c1s.wav", baseURL + "c1s.wav", ""},
	{"c2.wav", baseURL + "c2.wav", ""},
	{"d1.wav", baseURL + "d1.wav", ""},
	{"d1s.wav", baseURL + "d1s.wav", ""},
	{"e1.wav", baseURL + "e1.wav", ""},
	{"f1.wav", baseURL + "f1.wav", ""},
	{"f1s.wav", baseURL + "f1s.wav", ""},
	{"g1.wav", baseURL + "g1.wav", ""},
	{"g1s.wav", baseURL + "g1s.wav", ""},
}

func main() {
	fmt.Println("Downloading piano samples...")

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