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

func TestRunCommand_MissingAPIKey(t *testing.T) {
	// Only run this test if GEMINI_API_KEY is NOT set in the environment
	// OR if we explicitly unset it for the command.
	binaryPath := getBinaryPath(t)

	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "output")

	// Provide all required flags but ensure NO API KEY
	cmd := exec.Command(binaryPath, "run",
		"--job", "testdata/parsing/sample_job_plain.txt",
		"--experience", "testdata/valid/experience_bank.json",
		"--company-seed", "https://example.com",
		"--out", outDir)

	// Clear environment to ensure no API Key
	cmd.Env = os.Environ()
	// Filter out GEMINI_API_KEY
	var env []string
	for _, e := range cmd.Env {
		if !strings.HasPrefix(e, "GEMINI_API_KEY=") {
			env = append(env, e)
		}
	}
	cmd.Env = env

	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	// We expect the command to fail because of missing API key in RunPipeline wrapper
	assert.Contains(t, string(output), "GEMINI_API_KEY environment variable or --api-key flag is required")
}

func TestRunCommand_APIKeyProvided(t *testing.T) {
	// This test provides a dummy API key and expects the pipeline to START (and fail later)
	binaryPath := getBinaryPath(t)

	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "output")

	// Create dummy input files if they don't exist in CWD (test runs in package dir, but binary runs in CWD usually)
	// Actually exec runs in the Cwd of the parent process unless specified.
	// We typically run tests from the project root in the provided makefile or `go test ./...`
	// Let's create dummy files in tmpDir to be safe.
	jobFile := filepath.Join(tmpDir, "job.txt")
	_ = os.WriteFile(jobFile, []byte("Job Description"), 0644)

	expFile := filepath.Join(tmpDir, "exp.json")
	// Minimal valid experience bank JSON
	expJSON := `{
  "stories": []
}`
	_ = os.WriteFile(expFile, []byte(expJSON), 0644)

	cmd := exec.Command(binaryPath, "run",
		"--job", jobFile,
		"--experience", expFile,
		"--company-seed", "https://example.com",
		"--out", outDir,
		"--api-key", "dummy-key")

	output, err := cmd.CombinedOutput()

	// It should fail, but NOT because of missing API key.
	// It will likely fail at parsing or ingestion since "job.txt" is minimal.
	// Or it might fail at ingestion Step 1 if the file is valid but Step 2 parsing fails with 400 from API (invalid key).

	assert.Error(t, err)
	// Check that it started the pipeline
	assert.Contains(t, string(output), "Step 1/12: Ingesting job posting")
}
