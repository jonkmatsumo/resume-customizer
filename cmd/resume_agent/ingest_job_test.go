package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestJobCommand_TextFileSuccess(t *testing.T) {
	binaryPath := getBinaryPath(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_job.txt")
	testContent := "# Job Title\n\nDescription"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tmpDir, "output")

	cmd := exec.Command(binaryPath, "ingest-job", "--text-file", testFile, "--out", outDir)
	output, err := cmd.CombinedOutput()

	assert.NoError(t, err, "command should succeed: %s", string(output))
	assert.Contains(t, string(output), "Successfully ingested")

	// Verify output files exist
	cleanedPath := filepath.Join(outDir, "job_posting.cleaned.txt")
	metaPath := filepath.Join(outDir, "job_posting.meta.json")

	_, err = os.Stat(cleanedPath)
	assert.NoError(t, err, "cleaned text file should exist")

	_, err = os.Stat(metaPath)
	assert.NoError(t, err, "metadata file should exist")
}

func TestIngestJobCommand_URLSuccess(t *testing.T) {
	binaryPath := getBinaryPath(t)

	// Skip this test if we can't make network requests
	// In real CI, we'd use a mock server
	t.Skip("Skipping URL test - requires network access or mock server setup")

	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "output")

	cmd := exec.Command(binaryPath, "ingest-job", "--url", "https://example.com", "--out", outDir)
	_, err := cmd.CombinedOutput()

	// This will likely fail without a real URL, but we test the flag parsing
	_ = err
}

func TestIngestJobCommand_MissingFlags(t *testing.T) {
	binaryPath := getBinaryPath(t)

	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "output")

	// Neither --text-file nor --url provided
	cmd := exec.Command(binaryPath, "ingest-job", "--out", outDir)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "either --text-file or --url must be provided")
}

func TestIngestJobCommand_BothFlagsProvided(t *testing.T) {
	binaryPath := getBinaryPath(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tmpDir, "output")

	cmd := exec.Command(binaryPath, "ingest-job", "--text-file", testFile, "--url", "https://example.com", "--out", outDir)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "mutually exclusive")
}

func TestIngestJobCommand_MissingOutFlag(t *testing.T) {
	binaryPath := getBinaryPath(t)

	cmd := exec.Command(binaryPath, "ingest-job", "--text-file", "test.txt")
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required")
}

func TestIngestJobCommand_InvalidTextFile(t *testing.T) {
	binaryPath := getBinaryPath(t)

	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "output")

	cmd := exec.Command(binaryPath, "ingest-job", "--text-file", "/nonexistent/file.txt", "--out", outDir)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "file not found")
}

func TestIngestJobCommand_InvalidURL(t *testing.T) {
	binaryPath := getBinaryPath(t)

	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "output")

	cmd := exec.Command(binaryPath, "ingest-job", "--url", "not-a-url", "--out", outDir)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "invalid URL")
}

func TestIngestJobCommand_CreatesOutputDirectory(t *testing.T) {
	binaryPath := getBinaryPath(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Output directory doesn't exist
	outDir := filepath.Join(tmpDir, "new", "output", "dir")

	cmd := exec.Command(binaryPath, "ingest-job", "--text-file", testFile, "--out", outDir)
	output, err := cmd.CombinedOutput()

	assert.NoError(t, err, "command should succeed and create directory: %s", string(output))

	// Directory should exist
	_, err = os.Stat(outDir)
	assert.NoError(t, err, "output directory should be created")
}

func TestIngestJobCommand_OutputFilesExist(t *testing.T) {
	binaryPath := getBinaryPath(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "# Test Job\n\nDescription"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tmpDir, "output")

	cmd := exec.Command(binaryPath, "ingest-job", "--text-file", testFile, "--out", outDir)
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	// Verify files exist and have content
	cleanedPath := filepath.Join(outDir, "job_posting.cleaned.txt")
	cleanedContent, err := os.ReadFile(cleanedPath)
	require.NoError(t, err)
	assert.NotEmpty(t, cleanedContent)
	assert.Contains(t, string(cleanedContent), "Test Job")

	metaPath := filepath.Join(outDir, "job_posting.meta.json")
	metaContent, err := os.ReadFile(metaPath)
	require.NoError(t, err)
	assert.NotEmpty(t, metaContent)
	assert.Contains(t, string(metaContent), "timestamp")
	assert.Contains(t, string(metaContent), "hash")
}

func TestIngestJobCommand_ExitCode(t *testing.T) {
	binaryPath := getBinaryPath(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tmpDir, "output")

	// Success case
	cmd := exec.Command(binaryPath, "ingest-job", "--text-file", testFile, "--out", outDir)
	err = cmd.Run()
	assert.NoError(t, err)

	// Failure case - invalid file
	cmd = exec.Command(binaryPath, "ingest-job", "--text-file", "/nonexistent/file.txt", "--out", outDir)
	err = cmd.Run()
	if exitError, ok := err.(*exec.ExitError); ok {
		assert.NotEqual(t, 0, exitError.ExitCode())
	} else {
		assert.Error(t, err)
	}
}
