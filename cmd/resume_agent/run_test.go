package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunCommand_MissingFlags(t *testing.T) {
	binaryPath := getBinaryPath(t)

	// Missing all required flags for 'run'
	cmd := exec.Command(binaryPath, "run")
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	// After config file support, the error message changed
	assert.Contains(t, string(output), "either --job or --job-url must be provided")
}

func TestRunCommand_MissingUserID(t *testing.T) {
	// Test that run command requires user_id
	binaryPath := getBinaryPath(t)

	// Provide job but no user_id
	cmd := exec.Command(binaryPath, "run",
		"--job", "testdata/parsing/sample_job_plain.txt",
		"--company-seed", "https://example.com")

	// Clear environment to ensure no API Key or DATABASE_URL
	cmd.Env = os.Environ()
	var env []string
	for _, e := range cmd.Env {
		if !strings.HasPrefix(e, "GEMINI_API_KEY=") && !strings.HasPrefix(e, "DATABASE_URL=") {
			env = append(env, e)
		}
	}
	cmd.Env = env

	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	// Should fail because user_id is required
	assert.Contains(t, string(output), "--user-id is required")
}

func TestRunCommand_MissingDatabaseURL(t *testing.T) {
	// Test that run command requires DATABASE_URL when user_id is provided
	binaryPath := getBinaryPath(t)

	tmpDir := t.TempDir()

	// Create a config file with user_id
	configFile := filepath.Join(tmpDir, "config.json")
	configJSON := `{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "job_url": "https://example.com/job"
}`
	_ = os.WriteFile(configFile, []byte(configJSON), 0644)

	cmd := exec.Command(binaryPath, "run",
		"--config", configFile,
		"--api-key", "dummy-key")

	// Clear environment to ensure no DATABASE_URL
	cmd.Env = os.Environ()
	var env []string
	for _, e := range cmd.Env {
		if !strings.HasPrefix(e, "DATABASE_URL=") {
			env = append(env, e)
		}
	}
	cmd.Env = env

	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	// Should fail because DATABASE_URL is required
	assert.Contains(t, string(output), "DATABASE_URL environment variable or --db-url flag is required")
}
