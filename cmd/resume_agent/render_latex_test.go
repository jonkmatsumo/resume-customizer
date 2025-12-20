package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderLaTeXCommand_MissingPlanFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.tex")

	cmd := exec.Command(binaryPath, "render-latex",
		"--bullets", "bullets.json",
		"--name", "John Doe",
		"--email", "john@example.com",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"plan\" not set")
}

func TestRenderLaTeXCommand_MissingBulletsFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.tex")

	cmd := exec.Command(binaryPath, "render-latex",
		"--plan", "plan.json",
		"--name", "John Doe",
		"--email", "john@example.com",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"bullets\" not set")
}

func TestRenderLaTeXCommand_MissingNameFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.tex")

	cmd := exec.Command(binaryPath, "render-latex",
		"--plan", "plan.json",
		"--bullets", "bullets.json",
		"--email", "john@example.com",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"name\" not set")
}

func TestRenderLaTeXCommand_MissingEmailFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.tex")

	cmd := exec.Command(binaryPath, "render-latex",
		"--plan", "plan.json",
		"--bullets", "bullets.json",
		"--name", "John Doe",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"email\" not set")
}

func TestRenderLaTeXCommand_MissingOutputFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)

	cmd := exec.Command(binaryPath, "render-latex",
		"--plan", "plan.json",
		"--bullets", "bullets.json",
		"--name", "John Doe",
		"--email", "john@example.com")
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"out\" not set")
}

func TestRenderLaTeXCommand_InvalidPlanFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	bulletsFile := filepath.Join(tmpDir, "bullets.json")
	templateFile := filepath.Join(tmpDir, "template.tex")
	outputFile := filepath.Join(tmpDir, "output.tex")

	_ = os.WriteFile(bulletsFile, []byte(`{"bullets":[]}`), 0644)
	_ = os.WriteFile(templateFile, []byte(`\documentclass{article}\begin{document}Test\end{document}`), 0644)

	cmd := exec.Command(binaryPath, "render-latex",
		"--plan", "/nonexistent/plan.json",
		"--bullets", bulletsFile,
		"--template", templateFile,
		"--name", "John Doe",
		"--email", "john@example.com",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to read plan file")
}

func TestRenderLaTeXCommand_InvalidBulletsFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.json")
	templateFile := filepath.Join(tmpDir, "template.tex")
	outputFile := filepath.Join(tmpDir, "output.tex")

	_ = os.WriteFile(planFile, []byte(`{"selected_stories":[]}`), 0644)
	_ = os.WriteFile(templateFile, []byte(`\documentclass{article}\begin{document}Test\end{document}`), 0644)

	cmd := exec.Command(binaryPath, "render-latex",
		"--plan", planFile,
		"--bullets", "/nonexistent/bullets.json",
		"--template", templateFile,
		"--name", "John Doe",
		"--email", "john@example.com",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to read bullets file")
}

func TestRenderLaTeXCommand_InvalidTemplateFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.json")
	bulletsFile := filepath.Join(tmpDir, "bullets.json")
	outputFile := filepath.Join(tmpDir, "output.tex")

	_ = os.WriteFile(planFile, []byte(`{"selected_stories":[]}`), 0644)
	_ = os.WriteFile(bulletsFile, []byte(`{"bullets":[]}`), 0644)

	cmd := exec.Command(binaryPath, "render-latex",
		"--plan", planFile,
		"--bullets", bulletsFile,
		"--template", "/nonexistent/template.tex",
		"--name", "John Doe",
		"--email", "john@example.com",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to render LaTeX")
}

func TestRenderLaTeXCommand_CreatesOutputDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.json")
	bulletsFile := filepath.Join(tmpDir, "bullets.json")
	templateFile := filepath.Join(tmpDir, "template.tex")
	outputFile := filepath.Join(tmpDir, "nested", "output", "resume.tex")

	plan := types.ResumePlan{SelectedStories: []types.SelectedStory{}}
	planBytes, _ := json.Marshal(plan)
	_ = os.WriteFile(planFile, planBytes, 0644)

	bullets := types.RewrittenBullets{Bullets: []types.RewrittenBullet{}}
	bulletsBytes, _ := json.Marshal(bullets)
	_ = os.WriteFile(bulletsFile, bulletsBytes, 0644)

	templateContent := `\documentclass{article}\begin{document}Test\end{document}`
	_ = os.WriteFile(templateFile, []byte(templateContent), 0644)

	cmd := exec.Command(binaryPath, "render-latex",
		"--plan", planFile,
		"--bullets", bulletsFile,
		"--template", templateFile,
		"--name", "John Doe",
		"--email", "john@example.com",
		"--out", outputFile)
	_ = cmd.Run() // Command may succeed or fail depending on template, but directory should be created

	// Check directory was created
	outputDir := filepath.Dir(outputFile)
	_, err := os.Stat(outputDir)
	assert.NoError(t, err, "output directory should be created")
}

func TestRenderLaTeXCommand_ValidInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	// Use test fixtures - use absolute paths to avoid relative path issues
	wd, err := os.Getwd()
	require.NoError(t, err)
	testdataDir := filepath.Join(wd, "..", "..", "testdata", "rendering")

	planFile := filepath.Join(testdataDir, "sample_resume_plan.json")
	bulletsFile := filepath.Join(testdataDir, "sample_rewritten_bullets.json")
	experienceFile := filepath.Join(testdataDir, "sample_experience_bank.json")
	templateFile := filepath.Join(testdataDir, "minimal_template.tex")
	outputFile := filepath.Join(tmpDir, "resume.tex")

	cmd := exec.Command(binaryPath, "render-latex",
		"--plan", planFile,
		"--bullets", bulletsFile,
		"--experience", experienceFile,
		"--template", templateFile,
		"--name", "John Doe",
		"--email", "john@example.com",
		"--phone", "555-1234",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "command should succeed: %s", string(output))
	assert.Contains(t, string(output), "Successfully rendered LaTeX resume")

	// Verify output file exists and contains LaTeX
	outputContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	latex := string(outputContent)
	assert.Contains(t, latex, "\\documentclass{article}")
	assert.Contains(t, latex, "John Doe")
	assert.Contains(t, latex, "john@example.com")
}

func TestRenderLaTeXCommand_WithExperienceBank(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	// Create test files
	planFile := filepath.Join(tmpDir, "plan.json")
	bulletsFile := filepath.Join(tmpDir, "bullets.json")
	experienceFile := filepath.Join(tmpDir, "experience.json")
	templateFile := filepath.Join(tmpDir, "template.tex")
	outputFile := filepath.Join(tmpDir, "resume.tex")

	plan := types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
		},
	}
	planBytes, _ := json.Marshal(plan)
	_ = os.WriteFile(planFile, planBytes, 0644)

	bullets := types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Built a system",
			},
		},
	}
	bulletsBytes, _ := json.Marshal(bullets)
	_ = os.WriteFile(bulletsFile, bulletsBytes, 0644)

	experienceBank := types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				Company:   "Test Company",
				Role:      "Engineer",
				StartDate: "2020-01",
				EndDate:   "2023-01",
			},
		},
	}
	experienceBytes, _ := json.Marshal(experienceBank)
	_ = os.WriteFile(experienceFile, experienceBytes, 0644)

	templateContent := `\documentclass{article}\begin{document}Name: {{.Name}}{{range .Experience}}Company: {{.Company}}{{end}}\end{document}`
	_ = os.WriteFile(templateFile, []byte(templateContent), 0644)

	cmd := exec.Command(binaryPath, "render-latex",
		"--plan", planFile,
		"--bullets", bulletsFile,
		"--experience", experienceFile,
		"--template", templateFile,
		"--name", "John Doe",
		"--email", "john@example.com",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "command should succeed: %s", string(output))

	// Verify output contains company name from experience bank
	outputContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(outputContent), "Test Company")
}

func TestRenderLaTeXCommand_EscapesSpecialCharacters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()

	planFile := filepath.Join(tmpDir, "plan.json")
	bulletsFile := filepath.Join(tmpDir, "bullets.json")
	templateFile := filepath.Join(tmpDir, "template.tex")
	outputFile := filepath.Join(tmpDir, "resume.tex")

	plan := types.ResumePlan{SelectedStories: []types.SelectedStory{}}
	planBytes, _ := json.Marshal(plan)
	_ = os.WriteFile(planFile, planBytes, 0644)

	bullets := types.RewrittenBullets{Bullets: []types.RewrittenBullet{}}
	bulletsBytes, _ := json.Marshal(bullets)
	_ = os.WriteFile(bulletsFile, bulletsBytes, 0644)

	templateContent := `\documentclass{article}\begin{document}Name: {{.Name}}\end{document}`
	_ = os.WriteFile(templateFile, []byte(templateContent), 0644)

	// Name with special LaTeX characters
	cmd := exec.Command(binaryPath, "render-latex",
		"--plan", planFile,
		"--bullets", bulletsFile,
		"--template", templateFile,
		"--name", "John & Jane",
		"--email", "test@example.com",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "command should succeed: %s", string(output))

	// Verify LaTeX characters are escaped
	outputContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	latex := string(outputContent)
	// Should escape ampersand
	assert.Contains(t, latex, `\&`)
	assert.False(t, strings.Contains(latex, "John & Jane"), "should not contain unescaped ampersand")
}
