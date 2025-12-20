// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestRepairCommand_MissingRequiredFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)

	tests := []struct {
		name        string
		args        []string
		contains    string
		flagName    string
	}{
		{
			name:     "missing plan flag",
			args:     []string{"repair", "--bullets", "test.json", "--violations", "test.json", "--out", "/tmp"},
			contains: "required",
			flagName: "plan",
		},
		{
			name:     "missing bullets flag",
			args:     []string{"repair", "--plan", "test.json", "--violations", "test.json", "--out", "/tmp"},
			contains: "required",
			flagName: "bullets",
		},
		{
			name:     "missing violations flag",
			args:     []string{"repair", "--plan", "test.json", "--bullets", "test.json", "--out", "/tmp"},
			contains: "required",
			flagName: "violations",
		},
		{
			name:     "missing out flag",
			args:     []string{"repair", "--plan", "test.json", "--bullets", "test.json", "--violations", "test.json"},
			contains: "required",
			flagName: "out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.args...)
			output, err := cmd.CombinedOutput()

			assert.Error(t, err)
			outputStr := string(output)
			assert.Contains(t, outputStr, tt.contains, "should mention required flags")
			assert.Contains(t, outputStr, tt.flagName, "should mention the missing flag name")
		})
	}
}

func TestRepairCommand_InvalidInputFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	cmd := exec.Command(binaryPath, "repair",
		"--plan", "/nonexistent/plan.json",
		"--bullets", "/nonexistent/bullets.json",
		"--violations", "/nonexistent/violations.json",
		"--ranked", "/nonexistent/ranked.json",
		"--job-profile", "/nonexistent/job.json",
		"--company-profile", "/nonexistent/company.json",
		"--experience", "/nonexistent/experience.json",
		"--name", "Test User",
		"--email", "test@example.com",
		"--out", tmpDir,
	)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to read")
}

func TestRepairCommand_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	invalidJSONFile := filepath.Join(tmpDir, "invalid.json")
	_ = os.WriteFile(invalidJSONFile, []byte(`{invalid json`), 0644)

	cmd := exec.Command(binaryPath, "repair",
		"--plan", invalidJSONFile,
		"--bullets", invalidJSONFile,
		"--violations", invalidJSONFile,
		"--ranked", invalidJSONFile,
		"--job-profile", invalidJSONFile,
		"--company-profile", invalidJSONFile,
		"--experience", invalidJSONFile,
		"--name", "Test User",
		"--email", "test@example.com",
		"--out", tmpDir,
	)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to unmarshal")
}

func TestRepairCommand_ValidInputs_NoViolations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode (may require API key)")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

	// Create minimal valid input files
	plan := types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
				Section:   "experience",
			},
		},
	}
	planFile := filepath.Join(tmpDir, "plan.json")
	planBytes, _ := json.MarshalIndent(plan, "", "  ")
	_ = os.WriteFile(planFile, planBytes, 0644)

	bullets := types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Test bullet",
				LengthChars:      50,
				EstimatedLines:   1,
			},
		},
	}
	bulletsFile := filepath.Join(tmpDir, "bullets.json")
	bulletsBytes, _ := json.MarshalIndent(bullets, "", "  ")
	_ = os.WriteFile(bulletsFile, bulletsBytes, 0644)

	violations := types.Violations{Violations: []types.Violation{}}
	violationsFile := filepath.Join(tmpDir, "violations.json")
	violationsBytes, _ := json.MarshalIndent(violations, "", "  ")
	_ = os.WriteFile(violationsFile, violationsBytes, 0644)

	rankedStories := types.RankedStories{Ranked: []types.RankedStory{}}
	rankedFile := filepath.Join(tmpDir, "ranked.json")
	rankedBytes, _ := json.MarshalIndent(rankedStories, "", "  ")
	_ = os.WriteFile(rankedFile, rankedBytes, 0644)

	jobProfile := types.JobProfile{
		RoleTitle: "Engineer",
		Company:   "Test Corp",
	}
	jobFile := filepath.Join(tmpDir, "job.json")
	jobBytes, _ := json.MarshalIndent(jobProfile, "", "  ")
	_ = os.WriteFile(jobFile, jobBytes, 0644)

	companyProfile := types.CompanyProfile{
		Company: "Test Corp",
	}
	companyFile := filepath.Join(tmpDir, "company.json")
	companyBytes, _ := json.MarshalIndent(companyProfile, "", "  ")
	_ = os.WriteFile(companyFile, companyBytes, 0644)

	experienceBank := types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:   "story_001",
				Role: "Engineer",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_001",
						Text:        "Test bullet",
						LengthChars: 50,
					},
				},
			},
		},
	}
	experienceFile := filepath.Join(tmpDir, "experience.json")
	experienceBytes, _ := json.MarshalIndent(experienceBank, "", "  ")
	_ = os.WriteFile(experienceFile, experienceBytes, 0644)

	// Note: This test may fail if API key is not set or if validation fails
	// It's primarily checking that the command structure is correct
	cmd := exec.Command(binaryPath, "repair",
		"--plan", planFile,
		"--bullets", bulletsFile,
		"--violations", violationsFile,
		"--ranked", rankedFile,
		"--job-profile", jobFile,
		"--company-profile", companyFile,
		"--experience", experienceFile,
		"--name", "Test User",
		"--email", "test@example.com",
		"--out", outputDir,
	)
	output, err := cmd.CombinedOutput()

	// Command may succeed (no violations) or fail (API key missing, etc.)
	// Just verify it doesn't fail due to invalid command structure
	if err != nil {
		// Should not fail due to missing flags or file reading
		assert.NotContains(t, string(output), "required flag(s)")
		assert.NotContains(t, string(output), "failed to read")
	}
}

