package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getBinaryPath(t *testing.T) string {
	// Build the binary for testing
	binaryName := "resume_agent"
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	// Try to find or build the binary
	binaryPath := filepath.Join("..", "..", "bin", binaryName)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Binary doesn't exist, we'll need to build it
		// For now, we'll skip if it doesn't exist
		t.Skipf("Binary not found at %s, build it first with 'go build -o bin/resume_agent ./cmd/resume_agent'", binaryPath)
	}

	return binaryPath
}

func TestValidateCommand_Success(t *testing.T) {
	binaryPath := getBinaryPath(t)

	schemaPath := filepath.Join("..", "..", "schemas", "job_profile.schema.json")
	jsonPath := filepath.Join("..", "..", "testdata", "valid", "job_profile.json")

	cmd := exec.Command(binaryPath, "validate", "--schema", schemaPath, "--json", jsonPath)
	output, err := cmd.CombinedOutput()

	assert.NoError(t, err, "command should succeed")
	assert.Contains(t, string(output), "Validation passed", "output should indicate success")
}

func TestValidateCommand_Failure(t *testing.T) {
	binaryPath := getBinaryPath(t)

	schemaPath := filepath.Join("..", "..", "schemas", "job_profile.schema.json")
	jsonPath := filepath.Join("..", "..", "testdata", "invalid", "missing_field.json")

	cmd := exec.Command(binaryPath, "validate", "--schema", schemaPath, "--json", jsonPath)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err, "command should fail")
	assert.Contains(t, string(output), "Validation failed", "output should indicate failure")
	// Check exit code
	if exitError, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 1, exitError.ExitCode(), "should exit with code 1 on validation failure")
	}
}

func TestValidateCommand_MissingSchemaFlag(t *testing.T) {
	binaryPath := getBinaryPath(t)

	jsonPath := filepath.Join("..", "..", "testdata", "valid", "job_profile.json")

	cmd := exec.Command(binaryPath, "validate", "--json", jsonPath)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err, "command should fail")
	assert.Contains(t, string(output), "required", "should indicate flag is required")
	if exitError, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 2, exitError.ExitCode(), "should exit with code 2 on usage error")
	}
}

func TestValidateCommand_MissingJSONFlag(t *testing.T) {
	binaryPath := getBinaryPath(t)

	schemaPath := filepath.Join("..", "..", "schemas", "job_profile.schema.json")

	cmd := exec.Command(binaryPath, "validate", "--schema", schemaPath)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err, "command should fail")
	assert.Contains(t, string(output), "required", "should indicate flag is required")
}

func TestValidateCommand_InvalidSchemaPath(t *testing.T) {
	binaryPath := getBinaryPath(t)

	schemaPath := "nonexistent_schema.json"
	jsonPath := filepath.Join("..", "..", "testdata", "valid", "job_profile.json")

	cmd := exec.Command(binaryPath, "validate", "--schema", schemaPath, "--json", jsonPath)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err, "command should fail")
	assert.Contains(t, string(output), "not found", "should indicate file not found")
}

func TestValidateCommand_InvalidJSONPath(t *testing.T) {
	binaryPath := getBinaryPath(t)

	schemaPath := filepath.Join("..", "..", "schemas", "job_profile.schema.json")
	jsonPath := "nonexistent.json"

	cmd := exec.Command(binaryPath, "validate", "--schema", schemaPath, "--json", jsonPath)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err, "command should fail")
	assert.Contains(t, string(output), "not found", "should indicate file not found")
}

