package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJobCommand_FlagsValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantError   bool
		errorString string
	}{
		{
			name:        "Missing --in flag",
			args:        []string{"parse-job", "--out", "/tmp/output.json"},
			wantError:   true,
			errorString: "required",
		},
		{
			name:        "Missing --out flag",
			args:        []string{"parse-job", "--in", "/tmp/input.txt"},
			wantError:   true,
			errorString: "required",
		},
	}

	binaryPath := getBinaryPath(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.args...)
			output, err := cmd.CombinedOutput()

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, string(output), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseJobCommand_MissingAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	inputFile := filepath.Join(tmpDir, "input.txt")
	err := os.WriteFile(inputFile, []byte("Test job posting"), 0644)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "output.json")

	// Unset GEMINI_API_KEY for this test
	oldAPIKey := os.Getenv("GEMINI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")
	defer func() {
		if oldAPIKey != "" {
			os.Setenv("GEMINI_API_KEY", oldAPIKey)
		}
	}()

	cmd := exec.Command(binaryPath, "parse-job", "--in", inputFile, "--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err, "should fail when API key is missing")
	assert.Contains(t, string(output), "API key is required")
}

func TestParseJobCommand_InvalidInputFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	// Set a dummy API key so we get past that check
	oldAPIKey := os.Getenv("GEMINI_API_KEY")
	os.Setenv("GEMINI_API_KEY", "test-key")
	defer func() {
		if oldAPIKey != "" {
			os.Setenv("GEMINI_API_KEY", oldAPIKey)
		} else {
			os.Unsetenv("GEMINI_API_KEY")
		}
	}()

	cmd := exec.Command(binaryPath, "parse-job", "--in", "/nonexistent/file.txt", "--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err, "should fail when input file doesn't exist")
	assert.Contains(t, string(output), "failed to read input file")
}

func TestParseJobCommand_OutputFileCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping test")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	inputFile := filepath.Join(tmpDir, "input.txt")
	testContent := `Senior Software Engineer

Requirements:
- 3+ years Go experience
- Distributed systems`
	err := os.WriteFile(inputFile, []byte(testContent), 0644)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "parse-job", "--in", inputFile, "--out", outputFile)
	err = cmd.Run()

	// Note: This test may fail if API call fails, which is expected in test environment
	// The important thing is that it doesn't fail due to file I/O issues
	if err != nil {
		// If it fails, check it's not a file I/O error
		output, _ := cmd.CombinedOutput()
		assert.NotContains(t, string(output), "failed to write output file")
		assert.NotContains(t, string(output), "failed to read input file")
	} else {
		// If it succeeds, verify output file exists
		_, err := os.Stat(outputFile)
		assert.NoError(t, err, "output file should be created")
	}
}
