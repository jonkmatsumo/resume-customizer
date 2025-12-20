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

func TestRewriteCommand_MissingSelectedFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "rewrite",
		"--job-profile", "job.json",
		"--company-profile", "company.json",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"selected\" not set")
}

func TestRewriteCommand_MissingJobProfileFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "rewrite",
		"--selected", "bullets.json",
		"--company-profile", "company.json",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"job-profile\" not set")
}

func TestRewriteCommand_MissingCompanyProfileFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cmd := exec.Command(binaryPath, "rewrite",
		"--selected", "bullets.json",
		"--job-profile", "job.json",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"company-profile\" not set")
}

func TestRewriteCommand_MissingOutputFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)

	cmd := exec.Command(binaryPath, "rewrite",
		"--selected", "bullets.json",
		"--job-profile", "job.json",
		"--company-profile", "company.json")
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"out\" not set")
}

func TestRewriteCommand_MissingAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	selectedFile := filepath.Join(tmpDir, "selected.json")
	jobProfileFile := filepath.Join(tmpDir, "job.json")
	companyProfileFile := filepath.Join(tmpDir, "company.json")
	outputFile := filepath.Join(tmpDir, "output.json")

	_ = os.WriteFile(selectedFile, []byte(`{"bullets":[]}`), 0644)
	_ = os.WriteFile(jobProfileFile, []byte(`{"hard_requirements":[]}`), 0644)
	_ = os.WriteFile(companyProfileFile, []byte(`{"tone":"test","style_rules":[],"taboo_phrases":[],"values":[],"evidence_urls":[]}`), 0644)

	// Unset API key if set
	oldKey := os.Getenv("GEMINI_API_KEY")
	_ = os.Unsetenv("GEMINI_API_KEY")
	defer func() {
		if oldKey != "" {
			_ = os.Setenv("GEMINI_API_KEY", oldKey)
		}
	}()

	cmd := exec.Command(binaryPath, "rewrite",
		"--selected", selectedFile,
		"--job-profile", jobProfileFile,
		"--company-profile", companyProfileFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "API key is required")
}

func TestRewriteCommand_InvalidSelectedFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	jobProfileFile := filepath.Join(tmpDir, "job.json")
	companyProfileFile := filepath.Join(tmpDir, "company.json")
	outputFile := filepath.Join(tmpDir, "output.json")

	_ = os.WriteFile(jobProfileFile, []byte(`{"hard_requirements":[]}`), 0644)
	_ = os.WriteFile(companyProfileFile, []byte(`{"tone":"test","style_rules":[],"taboo_phrases":[],"values":[],"evidence_urls":[]}`), 0644)

	cmd := exec.Command(binaryPath, "rewrite",
		"--selected", "/nonexistent/selected.json",
		"--job-profile", jobProfileFile,
		"--company-profile", companyProfileFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to read selected bullets file")
}

func TestRewriteCommand_InvalidJobProfileFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	selectedFile := filepath.Join(tmpDir, "selected.json")
	companyProfileFile := filepath.Join(tmpDir, "company.json")
	outputFile := filepath.Join(tmpDir, "output.json")

	_ = os.WriteFile(selectedFile, []byte(`{"bullets":[]}`), 0644)
	_ = os.WriteFile(companyProfileFile, []byte(`{"tone":"test","style_rules":[],"taboo_phrases":[],"values":[],"evidence_urls":[]}`), 0644)

	cmd := exec.Command(binaryPath, "rewrite",
		"--selected", selectedFile,
		"--job-profile", "/nonexistent/job.json",
		"--company-profile", companyProfileFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to read job profile file")
}

func TestRewriteCommand_InvalidCompanyProfileFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	selectedFile := filepath.Join(tmpDir, "selected.json")
	jobProfileFile := filepath.Join(tmpDir, "job.json")
	outputFile := filepath.Join(tmpDir, "output.json")

	_ = os.WriteFile(selectedFile, []byte(`{"bullets":[]}`), 0644)
	_ = os.WriteFile(jobProfileFile, []byte(`{"hard_requirements":[]}`), 0644)

	cmd := exec.Command(binaryPath, "rewrite",
		"--selected", selectedFile,
		"--job-profile", jobProfileFile,
		"--company-profile", "/nonexistent/company.json",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to read company profile file")
}

func TestRewriteCommand_InvalidSelectedJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	selectedFile := filepath.Join(tmpDir, "selected.json")
	jobProfileFile := filepath.Join(tmpDir, "job.json")
	companyProfileFile := filepath.Join(tmpDir, "company.json")
	outputFile := filepath.Join(tmpDir, "output.json")

	_ = os.WriteFile(selectedFile, []byte(`{invalid json`), 0644)
	_ = os.WriteFile(jobProfileFile, []byte(`{"hard_requirements":[]}`), 0644)
	_ = os.WriteFile(companyProfileFile, []byte(`{"tone":"test","style_rules":[],"taboo_phrases":[],"values":[],"evidence_urls":[]}`), 0644)

	cmd := exec.Command(binaryPath, "rewrite",
		"--selected", selectedFile,
		"--job-profile", jobProfileFile,
		"--company-profile", companyProfileFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to unmarshal selected bullets JSON")
}

func TestRewriteCommand_CreatesOutputDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	selectedFile := filepath.Join(tmpDir, "selected.json")
	jobProfileFile := filepath.Join(tmpDir, "job.json")
	companyProfileFile := filepath.Join(tmpDir, "company.json")
	outputFile := filepath.Join(tmpDir, "nested", "output", "rewritten.json")

	selectedBullets := types.SelectedBullets{
		Bullets: []types.SelectedBullet{
			{
				ID:          "bullet_001",
				StoryID:     "story_001",
				Text:        "Built a system",
				Skills:      []string{"Go"},
				LengthChars: 15,
			},
		},
	}
	selectedBytes, _ := json.Marshal(selectedBullets)
	_ = os.WriteFile(selectedFile, selectedBytes, 0644)

	jobProfile := types.JobProfile{
		HardRequirements: []types.Requirement{},
	}
	jobBytes, _ := json.Marshal(jobProfile)
	_ = os.WriteFile(jobProfileFile, jobBytes, 0644)

	companyProfile := types.CompanyProfile{
		Tone:         "professional",
		StyleRules:   []string{},
		TabooPhrases: []string{},
		Values:       []string{},
		EvidenceURLs: []string{},
	}
	companyBytes, _ := json.Marshal(companyProfile)
	_ = os.WriteFile(companyProfileFile, companyBytes, 0644)

	_ = os.Setenv("GEMINI_API_KEY", "test-key")
	defer func() { _ = os.Unsetenv("GEMINI_API_KEY") }()

	// This will fail with API errors, but should create directory first
	cmd := exec.Command(binaryPath, "rewrite",
		"--selected", selectedFile,
		"--job-profile", jobProfileFile,
		"--company-profile", companyProfileFile,
		"--out", outputFile)
	_ = cmd.Run() // Ignore error, just check directory creation

	// Directory should be created (even if command fails later)
	outputDir := filepath.Dir(outputFile)
	_, err := os.Stat(outputDir)
	assert.NoError(t, err, "output directory should be created")
}

func TestRewriteCommand_ValidInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	// Use test fixtures from testdata
	selectedFile := filepath.Join("..", "..", "testdata", "rewriting", "sample_selected_bullets.json")
	jobProfileFile := filepath.Join("..", "..", "testdata", "rewriting", "sample_job_profile.json")
	companyProfileFile := filepath.Join("..", "..", "testdata", "rewriting", "sample_company_profile.json")
	outputFile := filepath.Join(tmpDir, "rewritten.json")

	cmd := exec.Command(binaryPath, "rewrite",
		"--selected", selectedFile,
		"--job-profile", jobProfileFile,
		"--company-profile", companyProfileFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "command should succeed: %s", string(output))
	assert.Contains(t, string(output), "Successfully rewritten")

	// Verify output file exists and is valid JSON
	outputContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var rewritten types.RewrittenBullets
	err = json.Unmarshal(outputContent, &rewritten)
	require.NoError(t, err)

	// Verify structure
	assert.NotEmpty(t, rewritten.Bullets)
	for _, bullet := range rewritten.Bullets {
		assert.NotEmpty(t, bullet.OriginalBulletID)
		assert.NotEmpty(t, bullet.FinalText)
		assert.Greater(t, bullet.LengthChars, 0)
		assert.GreaterOrEqual(t, bullet.EstimatedLines, 1)
		assert.NotNil(t, bullet.StyleChecks)
	}
}

