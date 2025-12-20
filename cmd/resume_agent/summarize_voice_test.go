package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeVoiceCommand_MissingInputFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "summarize-voice",
		"--sources", "sources.json",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"in\" not set")
}

func TestSummarizeVoiceCommand_MissingSourcesFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")
	inputFile := filepath.Join(tmpDir, "corpus.txt")
	_ = os.WriteFile(inputFile, []byte("test corpus"), 0644)

	cmd := exec.Command(binaryPath, "summarize-voice",
		"--in", inputFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"sources\" not set")
}

func TestSummarizeVoiceCommand_MissingOutputFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "corpus.txt")
	sourcesFile := filepath.Join(tmpDir, "sources.json")
	_ = os.WriteFile(inputFile, []byte("test corpus"), 0644)
	_ = os.WriteFile(sourcesFile, []byte(`[]`), 0644)

	cmd := exec.Command(binaryPath, "summarize-voice",
		"--in", inputFile,
		"--sources", sourcesFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"out\" not set")
}

func TestSummarizeVoiceCommand_MissingAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "corpus.txt")
	sourcesFile := filepath.Join(tmpDir, "sources.json")
	outputFile := filepath.Join(tmpDir, "output.json")

	_ = os.WriteFile(inputFile, []byte("test corpus"), 0644)
	_ = os.WriteFile(sourcesFile, []byte(`[]`), 0644)

	// Unset API key if set
	oldKey := os.Getenv("GEMINI_API_KEY")
	_ = os.Unsetenv("GEMINI_API_KEY")
	defer func() {
		if oldKey != "" {
			_ = os.Setenv("GEMINI_API_KEY", oldKey)
		}
	}()

	cmd := exec.Command(binaryPath, "summarize-voice",
		"--in", inputFile,
		"--sources", sourcesFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "API key is required")
}

func TestSummarizeVoiceCommand_InvalidInputFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	sourcesFile := filepath.Join(tmpDir, "sources.json")
	outputFile := filepath.Join(tmpDir, "output.json")

	_ = os.WriteFile(sourcesFile, []byte(`[]`), 0644)

	cmd := exec.Command(binaryPath, "summarize-voice",
		"--in", "/nonexistent/corpus.txt",
		"--sources", sourcesFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to read corpus file")
}

func TestSummarizeVoiceCommand_InvalidSourcesFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "corpus.txt")
	outputFile := filepath.Join(tmpDir, "output.json")

	_ = os.WriteFile(inputFile, []byte("test corpus"), 0644)

	cmd := exec.Command(binaryPath, "summarize-voice",
		"--in", inputFile,
		"--sources", "/nonexistent/sources.json",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to read sources file")
}

func TestSummarizeVoiceCommand_InvalidSourcesJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "corpus.txt")
	sourcesFile := filepath.Join(tmpDir, "sources.json")
	outputFile := filepath.Join(tmpDir, "output.json")

	_ = os.WriteFile(inputFile, []byte("test corpus"), 0644)
	_ = os.WriteFile(sourcesFile, []byte(`{invalid json`), 0644)

	cmd := exec.Command(binaryPath, "summarize-voice",
		"--in", inputFile,
		"--sources", sourcesFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to unmarshal sources JSON")
}

func TestSummarizeVoiceCommand_CreatesOutputDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "corpus.txt")
	sourcesFile := filepath.Join(tmpDir, "sources.json")
	outputFile := filepath.Join(tmpDir, "nested", "output", "profile.json")

	_ = os.WriteFile(inputFile, []byte("test corpus"), 0644)
	sources := []types.Source{
		{URL: "https://example.com/values", Timestamp: "2023-10-27T10:00:00Z", Hash: "hash1"},
	}
	sourcesBytes, _ := json.Marshal(sources)
	_ = os.WriteFile(sourcesFile, sourcesBytes, 0644)

	_ = os.Setenv("GEMINI_API_KEY", "test-key")
	defer func() { _ = os.Unsetenv("GEMINI_API_KEY") }()

	// This will fail with API errors, but should create directory first
	cmd := exec.Command(binaryPath, "summarize-voice",
		"--in", inputFile,
		"--sources", sourcesFile,
		"--out", outputFile)
	_ = cmd.Run() // Ignore error, just check directory creation

	// Directory should be created (even if command fails later)
	outputDir := filepath.Dir(outputFile)
	_, err := os.Stat(outputDir)
	// Note: This might not pass if command fails before directory creation
	assert.NoError(t, err, "output directory should be created")
}

func TestSummarizeVoiceCommand_ValidInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "corpus.txt")
	sourcesFile := filepath.Join(tmpDir, "sources.json")
	outputFile := filepath.Join(tmpDir, "profile.json")

	// Create test corpus
	corpusText := `Our company values ownership and customer obsession. We communicate directly with metrics.
We avoid marketing jargon. Domain: B2B SaaS infrastructure.`
	_ = os.WriteFile(inputFile, []byte(corpusText), 0644)

	// Create test sources
	sources := []types.Source{
		{URL: "https://example.com/values", Timestamp: "2023-10-27T10:00:00Z", Hash: "hash1"},
		{URL: "https://example.com/culture", Timestamp: "2023-10-27T10:05:00Z", Hash: "hash2"},
	}
	sourcesBytes, _ := json.Marshal(sources)
	_ = os.WriteFile(sourcesFile, sourcesBytes, 0644)

	cmd := exec.Command(binaryPath, "summarize-voice",
		"--in", inputFile,
		"--sources", sourcesFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "command should succeed: %s", string(output))
	assert.Contains(t, string(output), "Successfully summarized brand voice")

	// Verify output file exists and is valid JSON
	outputContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var profile types.CompanyProfile
	err = json.Unmarshal(outputContent, &profile)
	require.NoError(t, err)

	// Verify required fields are present
	assert.NotEmpty(t, profile.Company)
	assert.NotEmpty(t, profile.Tone)
	assert.NotEmpty(t, profile.StyleRules)
	assert.NotEmpty(t, profile.TabooPhrases)
	assert.NotEmpty(t, profile.DomainContext)
	assert.NotEmpty(t, profile.Values)
	assert.NotEmpty(t, profile.EvidenceURLs)
	assert.Len(t, profile.EvidenceURLs, 2)
}

