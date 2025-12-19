package main

import (
	"os"
	"path/filepath"
	"testing"
)

// getBinaryPath returns the path to the resume_agent binary for testing
func getBinaryPath(t *testing.T) string {
	binaryName := "resume_agent"
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := filepath.Join("..", "..", "bin", binaryName)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Binary not found at %s, build it first with 'make build'", binaryPath)
	}

	return binaryPath
}

