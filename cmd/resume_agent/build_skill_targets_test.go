package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSkillTargetsCommand_FlagsValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantError   bool
		errorString string
	}{
		{
			name:        "Missing --job-profile flag",
			args:        []string{"build-skill-targets", "--out", "/tmp/output.json"},
			wantError:   true,
			errorString: "required",
		},
		{
			name:        "Missing --out flag",
			args:        []string{"build-skill-targets", "--job-profile", "/tmp/input.json"},
			wantError:   true,
			errorString: "required",
		},
	}

	binaryPath := getBinaryPath(t)

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
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

func TestBuildSkillTargetsCommand_Success(t *testing.T) {
	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	// Create a valid job profile file
	inputContent := `{
		"company": "TestCo",
		"role_title": "Developer",
		"responsibilities": [],
		"hard_requirements": [
			{
				"skill": "Go",
				"evidence": "Required"
			}
		],
		"nice_to_haves": [
			{
				"skill": "Kubernetes",
				"evidence": "Preferred"
			}
		],
		"keywords": ["microservices"]
	}`
	inputFile := filepath.Join(tmpDir, "input.json")
	err := os.WriteFile(inputFile, []byte(inputContent), 0644)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "build-skill-targets", "--job-profile", inputFile, "--out", outputFile)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Command failed with output: %s", string(output))

	assert.Contains(t, string(output), "Successfully built skill targets")

	// Verify output file exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Output file should exist")

	// Verify output is valid JSON
	outputContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(outputContent), `"skills"`)
	assert.Contains(t, string(outputContent), `"name": "Go"`)
	assert.Contains(t, string(outputContent), `"weight": 1`)
}

func TestBuildSkillTargetsCommand_InvalidInputFile(t *testing.T) {
	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "build-skill-targets", "--job-profile", "/nonexistent/file.json", "--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to read job profile file")
}

func TestBuildSkillTargetsCommand_InvalidJSON(t *testing.T) {
	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	inputFile := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(inputFile, []byte(`{ invalid json }`), 0644)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "build-skill-targets", "--job-profile", inputFile, "--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to unmarshal job profile JSON")
}

func TestBuildSkillTargetsCommand_RealJobProfile(t *testing.T) {
	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	// Use real job profile from testdata
	jobProfilePath := "testdata/valid/job_profile.json"
	if _, err := os.Stat(jobProfilePath); os.IsNotExist(err) {
		t.Skip("testdata/valid/job_profile.json not found, skipping test")
	}

	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "build-skill-targets", "--job-profile", jobProfilePath, "--out", outputFile)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Command failed with output: %s", string(output))

	assert.Contains(t, string(output), "Successfully built skill targets")

	// Verify output file exists and contains expected content
	outputContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(outputContent), `"skills"`)
	// The job profile has "Go" in hard_requirements, so it should appear in output
	assert.Contains(t, string(outputContent), `"Go"`)
}

func TestBuildSkillTargetsCommand_EmptyJobProfile(t *testing.T) {
	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	inputContent := `{
		"company": "TestCo",
		"role_title": "Developer",
		"responsibilities": [],
		"hard_requirements": [],
		"nice_to_haves": [],
		"keywords": []
	}`
	inputFile := filepath.Join(tmpDir, "empty.json")
	err := os.WriteFile(inputFile, []byte(inputContent), 0644)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "build-skill-targets", "--job-profile", inputFile, "--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to build skill targets")
	assert.Contains(t, string(output), "no skills found")
}

