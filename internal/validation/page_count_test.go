// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountPDFPages_WithPdfinfo(t *testing.T) {
	// Skip if pdfinfo is not available
	if _, err := exec.LookPath("pdfinfo"); err != nil {
		t.Skip("pdfinfo not available, skipping test")
	}

	// We need a real PDF file for this test
	// Create a minimal PDF using pdflatex if available
	if _, err := exec.LookPath("pdflatex"); err != nil {
		t.Skip("pdflatex not available, cannot create test PDF")
	}

	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
Page 1
\newpage
Page 2
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	// Compile to PDF
	pdfPath, _, err := CompileLaTeX(texFile, tmpDir)
	require.NoError(t, err)

	// Count pages
	count, err := CountPDFPages(pdfPath)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestCountPDFPages_WithGhostscript(t *testing.T) {
	// Skip if ghostscript is not available
	if _, err := exec.LookPath("gs"); err != nil {
		t.Skip("ghostscript not available, skipping test")
	}

	// Skip if pdfinfo is available (we test that path separately)
	if _, err := exec.LookPath("pdfinfo"); err == nil {
		t.Skip("pdfinfo available, testing ghostscript fallback requires pdfinfo to be unavailable")
	}

	// We need a real PDF file for this test
	if _, err := exec.LookPath("pdflatex"); err != nil {
		t.Skip("pdflatex not available, cannot create test PDF")
	}

	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
Single page
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	// Compile to PDF
	pdfPath, _, err := CompileLaTeX(texFile, tmpDir)
	require.NoError(t, err)

	// Count pages (should use ghostscript fallback)
	count, err := CountPDFPages(pdfPath)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCountPDFPages_NoToolsAvailable(t *testing.T) {
	// This is hard to test without mocking exec.LookPath
	// We'll test the error case by using a non-existent PDF
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "nonexistent.pdf")

	_, err := CountPDFPages(pdfPath)
	assert.Error(t, err)
	// Error should indicate tools unavailable or file not found
}

func TestCountPDFPages_FileNotFound(t *testing.T) {
	_, err := CountPDFPages("/nonexistent/file.pdf")
	assert.Error(t, err)
}

