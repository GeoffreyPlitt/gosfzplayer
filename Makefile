# GoSFZPlayer Makefile

.PHONY: all test clean clean-dry-run clean-piano clean-all generate help build

# Default target
all: build test

# Build the project
build:
	go build ./...

# Run tests
test:
	go test -v

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.txt ./...

# Run tests with debug output
test-debug:
	DEBUG=sfzplayer:* go test -v

# Generate piano samples
generate:
	go generate ./...

# Dry run - show what would be cleaned without actually deleting
clean-dry-run:
	@echo "Files that would be deleted by 'make clean':"
	@git clean -xdn

# Clean generated files (safe - only removes gitignored files)
clean:
	@echo "Cleaning generated files..."
	git clean -xdf

# Clean only piano samples (useful if you want to re-download)
clean-piano:
	@echo "Cleaning piano samples..."
	rm -f testdata/piano/*.wav
	rm -f testdata/piano_arpeggio.wav

# Nuclear clean - removes ALL untracked files (be careful!)
clean-all:
	@echo "WARNING: This will remove ALL untracked files!"
	@echo "Are you sure? This will delete everything not in git!"
	@echo "Press Ctrl+C to cancel, or Enter to continue"
	@read
	git clean -xdf

# Show help
help:
	@echo "Available targets:"
	@echo "  all           - Build and test (default)"
	@echo "  build         - Build the project"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-debug    - Run tests with debug output"
	@echo "  generate      - Download piano samples"
	@echo "  clean-dry-run - Show what would be cleaned"
	@echo "  clean         - Remove gitignored generated files"
	@echo "  clean-piano   - Remove only piano samples"
	@echo "  clean-all     - Remove ALL untracked files (dangerous!)"
	@echo "  help          - Show this help"