// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/types"
)

// ValidateConstraints validates a LaTeX resume file against the specified constraints
func ValidateConstraints(texPath string, companyProfile *types.CompanyProfile, maxPages int, maxCharsPerLine int) (*types.Violations, error) {
	var allViolations []types.Violation

	// 1. Validate line lengths
	lineViolations, err := ValidateLineLengths(texPath, maxCharsPerLine)
	if err != nil {
		return nil, fmt.Errorf("failed to validate line lengths: %w", err)
	}
	allViolations = append(allViolations, lineViolations...)

	// 2. Check forbidden phrases (if company profile provided)
	if companyProfile != nil && len(companyProfile.TabooPhrases) > 0 {
		phraseViolations, err := CheckForbiddenPhrases(texPath, companyProfile.TabooPhrases)
		if err != nil {
			return nil, fmt.Errorf("failed to check forbidden phrases: %w", err)
		}
		allViolations = append(allViolations, phraseViolations...)
	}

	// 3. Compile LaTeX and check page count
	workDir := filepath.Dir(texPath)
	pdfPath, logOutput, err := CompileLaTeX(texPath, workDir)
	
	// Handle compilation errors
	if err != nil {
		var compErr *CompilationError
		if errors.As(err, &compErr) {
			// Add compilation error as a violation
			allViolations = append(allViolations, types.Violation{
				Type:     "latex_error",
				Severity: "error",
				Details:  fmt.Sprintf("LaTeX compilation failed: %s", compErr.Message),
			})
			// If compilation failed, we can't check page count, so return violations so far
			return &types.Violations{Violations: allViolations}, nil
		}
		// Other errors (file read, etc.) should be returned
		return nil, fmt.Errorf("failed to compile LaTeX: %w", err)
	}

	// 4. Check page count (only if compilation succeeded)
	pageCount, err := CountPDFPages(pdfPath)
	if err != nil {
		// If page counting fails, add as a warning violation but continue
		allViolations = append(allViolations, types.Violation{
			Type:     "page_overflow",
			Severity: "warning",
			Details:  fmt.Sprintf("Could not determine page count: %v", err),
		})
	} else if pageCount > maxPages {
		allViolations = append(allViolations, types.Violation{
			Type:     "page_overflow",
			Severity: "error",
			Details:  fmt.Sprintf("Resume has %d pages, maximum allowed is %d", pageCount, maxPages),
		})
	}

	// Clean up compilation artifacts (best effort, ignore errors)
	_ = CleanupCompilationArtifacts(workDir)

	// Log compilation output for debugging (if there were warnings)
	if logOutput != "" && pageCount > maxPages {
		// Compilation may have warnings even if successful
		// Log output is captured but not included in violations
		_ = logOutput
	}

	return &types.Violations{Violations: allViolations}, nil
}

