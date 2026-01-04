// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/types"
)

// Options provides optional parameters for violation mapping
type Options struct {
	LineToBulletMap    map[int]string          // Line number → bullet_id
	Bullets            *types.RewrittenBullets // For bullet text and story ID lookup
	Plan               *types.ResumePlan       // For story ID lookup
	ForbiddenPhraseMap map[string][]string     // bulletID → list of forbidden phrases found (optional)
}

// ValidateFromContent validates LaTeX content against the specified constraints.
// It writes the content to a temp file for LaTeX compilation.
// If opts is provided and contains line-to-bullet mapping, violations will be mapped to bullet IDs.
func ValidateFromContent(latexContent string, companyProfile *types.CompanyProfile, maxPages int, maxCharsPerLine int, opts *Options) (*types.Violations, error) {
	// Create temp directory for validation
	tmpDir, err := os.MkdirTemp("", "resume-validation-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Write LaTeX content to temp file
	texPath := filepath.Join(tmpDir, "resume.tex")
	if err := os.WriteFile(texPath, []byte(latexContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp LaTeX file: %w", err)
	}

	return ValidateConstraints(texPath, companyProfile, maxPages, maxCharsPerLine, opts)
}

// ValidateConstraints validates a LaTeX resume file against the specified constraints.
// If opts is provided and contains line-to-bullet mapping, violations will be mapped to bullet IDs.
func ValidateConstraints(texPath string, companyProfile *types.CompanyProfile, maxPages int, maxCharsPerLine int, opts *Options) (*types.Violations, error) {
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

	violations := &types.Violations{Violations: allViolations}

	// Map violations to bullets if mapping is provided
	if opts != nil && opts.LineToBulletMap != nil && len(opts.LineToBulletMap) > 0 {
		forbiddenPhraseMap := opts.ForbiddenPhraseMap // Can be nil
		violations = MapViolationsToBullets(violations, opts.LineToBulletMap, opts.Bullets, opts.Plan, forbiddenPhraseMap)
	}

	return violations, nil
}
