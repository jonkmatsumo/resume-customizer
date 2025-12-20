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
	"github.com/stretchr/testify/require"
)

func TestValidateLatexCommand_MissingInputFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "violations.json")

	cmd := exec.Command(binaryPath, "validate-latex",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"in\" not set")
}

func TestValidateLatexCommand_MissingOutputFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	_ = os.WriteFile(texFile, []byte(`\documentclass{article}\begin{document}Test\end{document}`), 0644)

	cmd := exec.Command(binaryPath, "validate-latex",
		"--in", texFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "required flag(s) \"out\" not set")
}

func TestValidateLatexCommand_InvalidInputFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "violations.json")

	cmd := exec.Command(binaryPath, "validate-latex",
		"--in", "/nonexistent/file.tex",
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "LaTeX file not found")
}

func TestValidateLatexCommand_ValidInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	outputFile := filepath.Join(tmpDir, "violations.json")

	// Create a valid LaTeX file
	content := `\documentclass{article}
\begin{document}
Short line content
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	cmd := exec.Command(binaryPath, "validate-latex",
		"--in", texFile,
		"--out", outputFile,
		"--max-pages", "1",
		"--max-chars", "90")
	output, err := cmd.CombinedOutput()

	// Command may succeed (no violations) or fail (violations found)
	// But output file should be created in either case
	if err != nil {
		// If there are violations, that's expected (exit code 1)
		assert.Contains(t, string(output), "violation")
	} else {
		assert.Contains(t, string(output), "Validation passed")
	}

	// Verify output file exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "violations file should be created")

	// Verify output is valid JSON
	contentBytes, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var violations types.Violations
	err = json.Unmarshal(contentBytes, &violations)
	require.NoError(t, err)
	assert.NotNil(t, violations.Violations)
}

func TestValidateLatexCommand_WithCompanyProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	outputFile := filepath.Join(tmpDir, "violations.json")
	companyProfileFile := filepath.Join(tmpDir, "company_profile.json")

	// Create LaTeX file with forbidden phrase
	texContent := `\documentclass{article}
\begin{document}
I am a coding ninja
\end{document}`
	err := os.WriteFile(texFile, []byte(texContent), 0644)
	require.NoError(t, err)

	// Create company profile with taboo phrase
	companyProfile := types.CompanyProfile{
		TabooPhrases: []string{"ninja"},
	}
	profileBytes, err := json.MarshalIndent(companyProfile, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(companyProfileFile, profileBytes, 0644)
	require.NoError(t, err)

	cmd := exec.Command(binaryPath, "validate-latex",
		"--in", texFile,
		"--company-profile", companyProfileFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	// Should find violations (forbidden phrase)
	assert.Error(t, err) // Exit code 1 for violations
	assert.Contains(t, string(output), "violation")

	// Verify violations contain forbidden_phrase
	contentBytes, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var violations types.Violations
	err = json.Unmarshal(contentBytes, &violations)
	require.NoError(t, err)
	
	hasForbidden := false
	for _, v := range violations.Violations {
		if v.Type == "forbidden_phrase" {
			hasForbidden = true
			break
		}
	}
	assert.True(t, hasForbidden, "should have forbidden_phrase violation")
}

func TestValidateLatexCommand_InvalidCompanyProfileJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	binaryPath := getBinaryPath(t)
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	outputFile := filepath.Join(tmpDir, "violations.json")
	companyProfileFile := filepath.Join(tmpDir, "invalid.json")

	_ = os.WriteFile(texFile, []byte(`\documentclass{article}\begin{document}Test\end{document}`), 0644)
	_ = os.WriteFile(companyProfileFile, []byte(`{invalid json`), 0644)

	cmd := exec.Command(binaryPath, "validate-latex",
		"--in", texFile,
		"--company-profile", companyProfileFile,
		"--out", outputFile)
	output, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(output), "failed to unmarshal company profile JSON")
}

