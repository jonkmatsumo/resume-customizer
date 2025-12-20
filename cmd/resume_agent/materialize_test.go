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

func TestMaterializeCommand_MissingPlanFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "materialize",
		"--experience", "test.json",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required")
}

func TestMaterializeCommand_MissingExperienceFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "materialize",
		"--plan", "test.json",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required")
}

func TestMaterializeCommand_MissingOutputFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.json")
	experienceFile := filepath.Join(tmpDir, "experience.json")

	_ = os.WriteFile(planFile, []byte(`{"selected_stories":[]}`), 0644)
	_ = os.WriteFile(experienceFile, []byte(`{"stories":[]}`), 0644)

	cmd := exec.Command(binaryPath, "materialize",
		"--plan", planFile,
		"--experience", experienceFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required")
}

func TestMaterializeCommand_InvalidPlanFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")
	experienceFile := filepath.Join(tmpDir, "experience.json")

	_ = os.WriteFile(experienceFile, []byte(`{"stories":[]}`), 0644)

	cmd := exec.Command(binaryPath, "materialize",
		"--plan", filepath.Join(tmpDir, "nonexistent.json"),
		"--experience", experienceFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to read plan file")
}

func TestMaterializeCommand_InvalidExperienceFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")
	planFile := filepath.Join(tmpDir, "plan.json")

	_ = os.WriteFile(planFile, []byte(`{"selected_stories":[]}`), 0644)

	cmd := exec.Command(binaryPath, "materialize",
		"--plan", planFile,
		"--experience", filepath.Join(tmpDir, "nonexistent.json"),
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to load experience bank")
}

func TestMaterializeCommand_InvalidPlanJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")
	planFile := filepath.Join(tmpDir, "plan.json")
	experienceFile := filepath.Join(tmpDir, "experience.json")

	_ = os.WriteFile(planFile, []byte(`{invalid json`), 0644)
	_ = os.WriteFile(experienceFile, []byte(`{"stories":[]}`), 0644)

	cmd := exec.Command(binaryPath, "materialize",
		"--plan", planFile,
		"--experience", experienceFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to unmarshal plan JSON")
}

func TestMaterializeCommand_ValidInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")
	planFile := filepath.Join(tmpDir, "plan.json")
	experienceFile := filepath.Join(tmpDir, "experience.json")

	// Create valid plan
	plan := types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_001"},
				Section:        "experience",
				EstimatedLines: 1,
			},
		},
		SpaceBudget: types.SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: types.Coverage{
			TopSkillsCovered: []string{"Go"},
			CoverageScore:    0.85,
		},
	}
	planJSON, err := json.Marshal(plan)
	require.NoError(t, err)
	_ = os.WriteFile(planFile, planJSON, 0644)

	// Create valid experience bank
	experienceBank := types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				Company:   "Company A",
				Role:      "Engineer",
				StartDate: "2023-01",
				EndDate:   "2024-01",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_001",
						Text:        "Built Go microservices",
						Skills:      []string{"Go"},
						LengthChars: 30,
					},
				},
			},
		},
	}
	experienceJSON, err := json.Marshal(experienceBank)
	require.NoError(t, err)
	_ = os.WriteFile(experienceFile, experienceJSON, 0644)

	cmd := exec.Command(binaryPath, "materialize",
		"--plan", planFile,
		"--experience", experienceFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "Command should succeed, output: %s", string(output))
	assert.Contains(t, string(output), "Successfully materialized")

	// Verify output file exists and is valid JSON
	outputContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var selectedBullets types.SelectedBullets
	err = json.Unmarshal(outputContent, &selectedBullets)
	require.NoError(t, err)
	assert.NotNil(t, selectedBullets.Bullets)
	assert.Len(t, selectedBullets.Bullets, 1)
	assert.Equal(t, "bullet_001", selectedBullets.Bullets[0].ID)
	assert.Equal(t, "story_001", selectedBullets.Bullets[0].StoryID)
}

func TestMaterializeCommand_StoryNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")
	planFile := filepath.Join(tmpDir, "plan.json")
	experienceFile := filepath.Join(tmpDir, "experience.json")

	// Create plan with non-existent story
	plan := types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_nonexistent",
				BulletIDs:      []string{"bullet_001"},
				Section:        "experience",
				EstimatedLines: 1,
			},
		},
		SpaceBudget: types.SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: types.Coverage{
			TopSkillsCovered: []string{},
			CoverageScore:    0.0,
		},
	}
	planJSON, err := json.Marshal(plan)
	require.NoError(t, err)
	_ = os.WriteFile(planFile, planJSON, 0644)

	// Create experience bank without the referenced story
	experienceBank := types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Company A",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{ID: "bullet_001", Text: "Test", Skills: []string{}, LengthChars: 10},
				},
			},
		},
	}
	experienceJSON, err := json.Marshal(experienceBank)
	require.NoError(t, err)
	_ = os.WriteFile(experienceFile, experienceJSON, 0644)

	cmd := exec.Command(binaryPath, "materialize",
		"--plan", planFile,
		"--experience", experienceFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to materialize bullets")
	assert.Contains(t, string(output), "story not found")
}

func TestMaterializeCommand_BulletNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")
	planFile := filepath.Join(tmpDir, "plan.json")
	experienceFile := filepath.Join(tmpDir, "experience.json")

	// Create plan with non-existent bullet
	plan := types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_nonexistent"},
				Section:        "experience",
				EstimatedLines: 1,
			},
		},
		SpaceBudget: types.SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: types.Coverage{
			TopSkillsCovered: []string{},
			CoverageScore:    0.0,
		},
	}
	planJSON, err := json.Marshal(plan)
	require.NoError(t, err)
	_ = os.WriteFile(planFile, planJSON, 0644)

	// Create experience bank without the referenced bullet
	experienceBank := types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Company A",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{ID: "bullet_001", Text: "Test", Skills: []string{}, LengthChars: 10},
				},
			},
		},
	}
	experienceJSON, err := json.Marshal(experienceBank)
	require.NoError(t, err)
	_ = os.WriteFile(experienceFile, experienceJSON, 0644)

	cmd := exec.Command(binaryPath, "materialize",
		"--plan", planFile,
		"--experience", experienceFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to materialize bullets")
	assert.Contains(t, string(output), "bullet not found")
}

