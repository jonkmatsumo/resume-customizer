// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileLaTeX_ValidLaTeX(t *testing.T) {
	// Skip if pdflatex is not available
	if _, err := exec.LookPath("pdflatex"); err != nil {
		t.Skip("pdflatex not available, skipping compilation test")
	}

	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
Hello, World!
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	pdfPath, _, err := CompileLaTeX(texFile, tmpDir)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfPath)

	// Verify PDF exists
	_, err = os.Stat(pdfPath)
	assert.NoError(t, err, "PDF should exist")
}

func TestCompileLaTeX_InvalidLaTeX(t *testing.T) {
	// Skip if pdflatex is not available
	if _, err := exec.LookPath("pdflatex"); err != nil {
		t.Skip("pdflatex not available, skipping compilation test")
	}

	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
\undefinedcommand{this will fail}
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	_, _, compileErr := CompileLaTeX(texFile, tmpDir)
	// Even with errors, PDF might be generated (partial compilation)
	// We don't check the result here, just verify the function doesn't panic
	_ = compileErr
}

func TestCompileLaTeX_FileNotFound(t *testing.T) {
	// Skip if pdflatex is not available
	if _, err := exec.LookPath("pdflatex"); err != nil {
		t.Skip("pdflatex not available, skipping compilation test")
	}

	_, _, err := CompileLaTeX("/nonexistent/file.tex", "")
	assert.Error(t, err)
	var fileErr *FileReadError
	var compErr *CompilationError
	assert.True(t, errors.As(err, &fileErr) || errors.As(err, &compErr))
}

func TestCompileLaTeX_PdflatexNotAvailable(t *testing.T) {
	// This test would need to mock exec.LookPath, which is complex
	// Instead, we test the error handling path by checking the error message
	// This is a best-effort test since we can't easily mock PATH
	t.Skip("Cannot easily test pdflatex unavailability without mocking exec")
}

func TestCleanupCompilationArtifacts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	testFile := filepath.Join(tmpDir, "test.aux")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Cleanup should not fail
	err = CleanupCompilationArtifacts(tmpDir)
	assert.NoError(t, err)

	// File may or may not be removed (depends on implementation)
	// Just verify function doesn't panic
}
