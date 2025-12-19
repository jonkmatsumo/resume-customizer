package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRankStoriesCommand_FlagsValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantError   bool
		errorString string
	}{
		{
			name:        "Missing --job-profile flag",
			args:        []string{"rank-stories", "--experience", "/tmp/exp.json", "--out", "/tmp/out.json"},
			wantError:   true,
			errorString: "required",
		},
		{
			name:        "Missing --experience flag",
			args:        []string{"rank-stories", "--job-profile", "/tmp/job.json", "--out", "/tmp/out.json"},
			wantError:   true,
			errorString: "required",
		},
		{
			name:        "Missing --out flag",
			args:        []string{"rank-stories", "--job-profile", "/tmp/job.json", "--experience", "/tmp/exp.json"},
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

func TestRankStoriesCommand_InvalidJobProfile(t *testing.T) {
	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	// Create invalid job profile
	jobProfileFile := filepath.Join(tmpDir, "invalid_job.json")
	err := os.WriteFile(jobProfileFile, []byte(`{ invalid json }`), 0644)
	require.NoError(t, err)

	experienceFile := filepath.Join(tmpDir, "experience.json")
	err = os.WriteFile(experienceFile, []byte(`{"stories": []}`), 0644)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "rank-stories", "--job-profile", jobProfileFile, "--experience", experienceFile, "--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to unmarshal job profile JSON")
}

func TestRankStoriesCommand_InvalidExperienceBank(t *testing.T) {
	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	jobProfileFile := filepath.Join(tmpDir, "job_profile.json")
	err := os.WriteFile(jobProfileFile, []byte(`{"hard_requirements": [], "keywords": []}`), 0644)
	require.NoError(t, err)

	// Create invalid experience bank file
	experienceFile := filepath.Join(tmpDir, "invalid_exp.json")
	err = os.WriteFile(experienceFile, []byte(`{ invalid json }`), 0644)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "rank-stories", "--job-profile", jobProfileFile, "--experience", experienceFile, "--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to load experience bank")
}

func TestRankStoriesCommand_RealFixtures(t *testing.T) {
	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	jobProfilePath := "testdata/valid/job_profile.json"
	experiencePath := "testdata/valid/experience_bank.json"

	if _, err := os.Stat(jobProfilePath); os.IsNotExist(err) {
		t.Skip("testdata/valid/job_profile.json not found")
	}
	if _, err := os.Stat(experiencePath); os.IsNotExist(err) {
		t.Skip("testdata/valid/experience_bank.json not found")
	}

	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "rank-stories", "--job-profile", jobProfilePath, "--experience", experiencePath, "--out", outputFile)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Command failed with output: %s", string(output))

	assert.Contains(t, string(output), "Successfully ranked")

	// Verify output file exists and has content
	outputContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(outputContent), `"ranked"`)
	assert.Greater(t, len(outputContent), 100, "Output should have substantial content")
}
