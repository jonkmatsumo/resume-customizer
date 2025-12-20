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

func TestPlanCommand_MissingRankedFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "plan",
		"--job-profile", "test.json",
		"--experience", "test.json",
		"--max-bullets", "8",
		"--max-lines", "45",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required")
}

func TestPlanCommand_MissingJobProfileFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "plan",
		"--ranked", "test.json",
		"--experience", "test.json",
		"--max-bullets", "8",
		"--max-lines", "45",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required")
}

func TestPlanCommand_InvalidMaxBullets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")
	rankedFile := filepath.Join(tmpDir, "ranked.json")
	jobProfileFile := filepath.Join(tmpDir, "job_profile.json")
	experienceFile := filepath.Join(tmpDir, "experience.json")

	// Create minimal valid JSON files
	_ = os.WriteFile(rankedFile, []byte(`{"ranked":[]}`), 0644)
	_ = os.WriteFile(jobProfileFile, []byte(`{"hard_requirements":[]}`), 0644)
	_ = os.WriteFile(experienceFile, []byte(`{"stories":[]}`), 0644)

	cmd := exec.Command(binaryPath, "plan",
		"--ranked", rankedFile,
		"--job-profile", jobProfileFile,
		"--experience", experienceFile,
		"--max-bullets", "0",
		"--max-lines", "45",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "max-bullets must be greater than 0")
}

func TestPlanCommand_InvalidMaxLines(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")
	rankedFile := filepath.Join(tmpDir, "ranked.json")
	jobProfileFile := filepath.Join(tmpDir, "job_profile.json")
	experienceFile := filepath.Join(tmpDir, "experience.json")

	// Create minimal valid JSON files
	_ = os.WriteFile(rankedFile, []byte(`{"ranked":[]}`), 0644)
	_ = os.WriteFile(jobProfileFile, []byte(`{"hard_requirements":[]}`), 0644)
	_ = os.WriteFile(experienceFile, []byte(`{"stories":[]}`), 0644)

	cmd := exec.Command(binaryPath, "plan",
		"--ranked", rankedFile,
		"--job-profile", jobProfileFile,
		"--experience", experienceFile,
		"--max-bullets", "8",
		"--max-lines", "0",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "max-lines must be greater than 0")
}

func TestPlanCommand_InvalidInputFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "plan",
		"--ranked", "/nonexistent/ranked.json",
		"--job-profile", "/nonexistent/job_profile.json",
		"--experience", "/nonexistent/experience.json",
		"--max-bullets", "8",
		"--max-lines", "45",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to read")
}

func TestPlanCommand_ValidInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	// Create test input files
	rankedFile := filepath.Join(tmpDir, "ranked.json")
	rankedStories := types.RankedStories{
		Ranked: []types.RankedStory{
			{
				StoryID:        "story_001",
				RelevanceScore: 0.8,
				MatchedSkills:  []string{"Go"},
			},
		},
	}
	rankedBytes, _ := json.Marshal(rankedStories)
	_ = os.WriteFile(rankedFile, rankedBytes, 0644)

	jobProfileFile := filepath.Join(tmpDir, "job_profile.json")
	jobProfile := types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
	}
	jobProfileBytes, _ := json.Marshal(jobProfile)
	_ = os.WriteFile(jobProfileFile, jobProfileBytes, 0644)

	experienceFile := filepath.Join(tmpDir, "experience.json")
	experienceBank := types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test Company",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_001",
						Text:        "Built Go services",
						LengthChars: 50,
						Skills:      []string{"Go"},
					},
				},
			},
		},
	}
	experienceBytes, _ := json.Marshal(experienceBank)
	_ = os.WriteFile(experienceFile, experienceBytes, 0644)

	cmd := exec.Command(binaryPath, "plan",
		"--ranked", rankedFile,
		"--job-profile", jobProfileFile,
		"--experience", experienceFile,
		"--max-bullets", "8",
		"--max-lines", "45",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "command should succeed: %s", string(output))
	assert.Contains(t, string(output), "Successfully created resume plan")

	// Verify output file exists and is valid JSON
	outputContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var plan types.ResumePlan
	err = json.Unmarshal(outputContent, &plan)
	require.NoError(t, err)
	assert.NotNil(t, plan.SelectedStories)
	assert.Equal(t, 8, plan.SpaceBudget.MaxBullets)
	assert.Equal(t, 45, plan.SpaceBudget.MaxLines)
}
