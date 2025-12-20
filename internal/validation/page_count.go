// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// CountPDFPages counts the number of pages in a PDF file
// It tries pdfinfo first, then falls back to ghostscript
func CountPDFPages(pdfPath string) (int, error) {
	// Try pdfinfo first (from poppler-utils)
	if count, err := countPagesWithPdfinfo(pdfPath); err == nil {
		return count, nil
	}

	// Fallback to ghostscript
	if count, err := countPagesWithGhostscript(pdfPath); err == nil {
		return count, nil
	}

	// If both methods fail, return error
	return 0, &Error{
		Message: "failed to count PDF pages: neither pdfinfo nor ghostscript available. Please install poppler-utils (pdfinfo) or ghostscript",
	}
}

// countPagesWithPdfinfo uses pdfinfo to count PDF pages
func countPagesWithPdfinfo(pdfPath string) (int, error) {
	cmd := exec.Command("pdfinfo", pdfPath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("pdfinfo command failed: %w", err)
	}

	// Parse output looking for "Pages: N"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Pages:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				count, err := strconv.Atoi(parts[1])
				if err == nil {
					return count, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("could not parse page count from pdfinfo output")
}

// countPagesWithGhostscript uses ghostscript to count PDF pages
func countPagesWithGhostscript(pdfPath string) (int, error) {
	// Use ghostscript to count pages
	// Command: gs -q -dNODISPLAY -c "(filename.pdf) (r) file runpdfbegin pdfpagecount = quit"
	script := fmt.Sprintf("(%s) (r) file runpdfbegin pdfpagecount = quit", pdfPath)
	cmd := exec.Command("gs", "-q", "-dNODISPLAY", "-c", script)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ghostscript command failed: %w", err)
	}

	// Output should be just the page count number
	outputStr := strings.TrimSpace(string(output))
	count, err := strconv.Atoi(outputStr)
	if err != nil {
		return 0, fmt.Errorf("could not parse page count from ghostscript output: %s", outputStr)
	}

	return count, nil
}
