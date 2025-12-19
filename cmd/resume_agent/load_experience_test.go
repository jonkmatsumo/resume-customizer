package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadExperienceCommand_ValidInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	inputFile := filepath.Join("..", "..", "testdata", "valid", "experience_bank.json")
	outputFile := filepath.Join(tmpDir, "normalized.json")

	cmd := exec.Command(binaryPath, "load-experience", "--in", inputFile, "--out", outputFile)
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "command should succeed: %s", string(output))
	assert.Contains(t, string(output), "Successfully loaded and normalized")

	// Verify output file exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "output file should exist")

	// Verify output is valid JSON
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var bank interface{}
	err = json.Unmarshal(content, &bank)
	assert.NoError(t, err, "output should be valid JSON")
}

func TestLoadExperienceCommand_MissingInputFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "load-experience", "--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required")
}

func TestLoadExperienceCommand_MissingOutputFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	inputFile := filepath.Join("..", "..", "testdata", "valid", "experience_bank.json")

	cmd := exec.Command(binaryPath, "load-experience", "--in", inputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required")
}

func TestLoadExperienceCommand_InvalidInputFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "load-experience", "--in", "/nonexistent/file.json", "--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to load")
}

func TestLoadExperienceCommand_CreatesOutputDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	inputFile := filepath.Join("..", "..", "testdata", "valid", "experience_bank.json")
	outputFile := filepath.Join(tmpDir, "nested", "dir", "normalized.json")

	cmd := exec.Command(binaryPath, "load-experience", "--in", inputFile, "--out", outputFile)
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "command should succeed: %s", string(output))

	// Verify nested directory was created
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "output file should exist in nested directory")
}

func TestLoadExperienceCommand_NormalizesOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	// Create input file with skills that need normalization
	inputFile := filepath.Join(tmpDir, "input.json")
	inputContent := `{
		"stories": [
			{
				"id": "story_001",
				"company": "Test Company",
				"role": "Engineer",
				"start_date": "2020-01",
				"end_date": "2023-06",
				"bullets": [
					{
						"id": "bullet_001",
						"text": "Built system with golang",
						"skills": ["golang", "javascript"],
						"length_chars": 0,
						"evidence_strength": "HIGH",
						"risk_flags": []
					}
				]
			}
		]
	}`
	err := os.WriteFile(inputFile, []byte(inputContent), 0644)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "normalized.json")

	cmd := exec.Command(binaryPath, "load-experience", "--in", inputFile, "--out", outputFile)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "command should succeed: %s", string(output))

	// Read and verify normalized output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	normalizedStr := string(content)
	// Verify normalization occurred
	assert.Contains(t, normalizedStr, `"Go"`)                       // Normalized skill
	assert.Contains(t, normalizedStr, `"JavaScript"`)               // Normalized skill
	assert.Contains(t, normalizedStr, `"evidence_strength": "high"`) // Lowercase
	assert.Contains(t, normalizedStr, `"length_chars": 25`)          // Computed
}

func TestLoadExperienceCommand_ValidatesOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	inputFile := filepath.Join("..", "..", "testdata", "valid", "experience_bank.json")
	outputFile := filepath.Join(tmpDir, "normalized.json")

	cmd := exec.Command(binaryPath, "load-experience", "--in", inputFile, "--out", outputFile)
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "command should succeed: %s", string(output))

	// Verify output validates against schema by reading it back
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var bank interface{}
	err = json.Unmarshal(content, &bank)
	require.NoError(t, err)

	// Basic structure check
	bankMap, ok := bank.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, bankMap, "stories")
}

