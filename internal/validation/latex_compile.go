// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	// CompilationTimeout is the maximum time to wait for LaTeX compilation
	CompilationTimeout = 30 * time.Second
)

// CompileLaTeX compiles a LaTeX file using pdflatex
func CompileLaTeX(texPath string, workDir string) (pdfPath string, logOutput string, err error) {
	// Check if pdflatex is available
	if _, err := exec.LookPath("pdflatex"); err != nil {
		return "", "", &CompilationError{
			Message: "pdflatex not found in PATH. Please install a LaTeX distribution (e.g., TeX Live, MiKTeX)",
			Cause:   err,
		}
	}

	// Create working directory if it doesn't exist
	if workDir == "" {
		var err error
		workDir, err = os.MkdirTemp("", "latex-compile-*")
		if err != nil {
			return "", "", &CompilationError{
				Message: "failed to create temporary working directory",
				Cause:   err,
			}
		}
	} else {
		if err := os.MkdirAll(workDir, 0755); err != nil {
			return "", "", &CompilationError{
				Message: fmt.Sprintf("failed to create working directory: %s", workDir),
				Cause:   err,
			}
		}
	}

	// Copy LaTeX file to working directory (or use original if already there)
	texBaseName := filepath.Base(texPath)
	workTexPath := filepath.Join(workDir, texBaseName)

	// If source and destination are different, copy the file
	if texPath != workTexPath {
		texContent, err := os.ReadFile(texPath)
		if err != nil {
			return "", "", &FileReadError{
				Message: fmt.Sprintf("failed to read LaTeX file: %s", texPath),
				Cause:   err,
			}
		}
		if err := os.WriteFile(workTexPath, texContent, 0644); err != nil {
			return "", "", &CompilationError{
				Message: fmt.Sprintf("failed to write LaTeX file to working directory: %s", workDir),
				Cause:   err,
			}
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), CompilationTimeout)
	defer cancel()

	// Run pdflatex
	// Use -interaction=nonstopmode to prevent interactive prompts
	// Use -output-directory to specify where to put output files
	cmd := exec.CommandContext(ctx, "pdflatex", "-interaction=nonstopmode", "-output-directory", workDir, workTexPath)

	// Capture both stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	runErr := cmd.Run()

	// Combine stdout and stderr for log output
	logOutput = stdout.String() + stderr.String()

	// Check if PDF was created
	pdfPath = filepath.Join(workDir, strings.TrimSuffix(texBaseName, ".tex")+".pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		// PDF was not created, compilation failed
		return "", logOutput, &CompilationError{
			Message:   "LaTeX compilation failed: PDF was not generated",
			LogOutput: logOutput,
			Cause:     runErr,
		}
	}

	// Even if command returned an error, if PDF exists, consider it a partial success
	// (LaTeX can produce PDFs with errors/warnings)
	if runErr != nil {
		return pdfPath, logOutput, &CompilationError{
			Message:   "LaTeX compilation completed with errors (PDF may be incomplete)",
			LogOutput: logOutput,
			Cause:     runErr,
		}
	}

	return pdfPath, logOutput, nil
}

// CleanupCompilationArtifacts removes temporary files created during compilation
func CleanupCompilationArtifacts(workDir string) error {
	if workDir == "" {
		return nil
	}

	// Check if this is a temporary directory we created
	if strings.Contains(workDir, "latex-compile-") {
		return os.RemoveAll(workDir)
	}

	// Otherwise, just remove common LaTeX auxiliary files
	auxFiles := []string{".aux", ".log", ".out", ".toc", ".lof", ".lot"}
	texBaseName := filepath.Base(workDir)
	for _, ext := range auxFiles {
		auxPath := filepath.Join(workDir, texBaseName+ext)
		_ = os.Remove(auxPath) // Ignore errors for missing files
	}

	return nil
}
