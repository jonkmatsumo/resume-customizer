// Package repair provides functionality to automatically fix violations in LaTeX resumes.
package repair

import (
	"context"
	"fmt"
	"os"

	"github.com/jonathan/resume-customizer/internal/rendering"
	"github.com/jonathan/resume-customizer/internal/rewriting"
	"github.com/jonathan/resume-customizer/internal/selection"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/jonathan/resume-customizer/internal/validation"
)

// CandidateInfo holds candidate information for resume rendering
type CandidateInfo struct {
	Name  string
	Email string
	Phone string
}

// RunRepairLoop runs the repair loop to fix violations iteratively
func RunRepairLoop(ctx context.Context, initialPlan *types.ResumePlan, initialBullets *types.RewrittenBullets, violations *types.Violations, rankedStories *types.RankedStories, jobProfile *types.JobProfile, companyProfile *types.CompanyProfile, experienceBank *types.ExperienceBank, templatePath string, candidateInfo CandidateInfo, selectedEducation []types.Education, maxPages int, maxCharsPerLine int, maxIterations int, apiKey string) (finalPlan *types.ResumePlan, finalBullets *types.RewrittenBullets, finalLaTeX string, finalViolations *types.Violations, iterations int, err error) {
	// Initialize loop state
	currentPlan := initialPlan
	currentBullets := initialBullets
	currentViolations := violations
	iterationsUsed := 0

	// Helper to check if we have any violations
	hasViolations := func(v *types.Violations) bool {
		return v != nil && len(v.Violations) > 0
	}

	// Loop until no violations or max iterations reached
	for hasViolations(currentViolations) && iterationsUsed < maxIterations {
		iterationsUsed++

		// 1. Propose repair actions
		repairActions, err := ProposeRepairs(ctx, currentViolations, currentPlan, currentBullets, rankedStories, jobProfile, companyProfile, apiKey)
		if err != nil {
			return nil, nil, "", currentViolations, iterationsUsed, fmt.Errorf("failed to propose repairs at iteration %d: %w", iterationsUsed, err)
		}

		// 2. Apply repairs
		updatedPlan, updatedBullets, needsRewrite, err := ApplyRepairs(repairActions, currentPlan, currentBullets, rankedStories, experienceBank)
		if err != nil {
			return nil, nil, "", currentViolations, iterationsUsed, fmt.Errorf("failed to apply repairs at iteration %d: %w", iterationsUsed, err)
		}

		// 3. Handle bullet rewriting if needed
		planChanged := plansDiffer(currentPlan, updatedPlan)
		if needsRewrite || planChanged {
			// Need to materialize and rewrite bullets
			// Materialize bullets from updated plan
			selectedBullets, err := selection.MaterializeBullets(updatedPlan, experienceBank)
			if err != nil {
				return nil, nil, "", currentViolations, iterationsUsed, fmt.Errorf("failed to materialize bullets at iteration %d: %w", iterationsUsed, err)
			}

			// Rewrite bullets
			rewritten, err := rewriting.RewriteBullets(ctx, selectedBullets, jobProfile, companyProfile, apiKey)
			if err != nil {
				return nil, nil, "", currentViolations, iterationsUsed, fmt.Errorf("failed to rewrite bullets at iteration %d: %w", iterationsUsed, err)
			}
			updatedBullets = rewritten
		}
		// If not needsRewrite and plan didn't change, use updatedBullets from ApplyRepairs (which may have dropped bullets)

		// 5. Render LaTeX
		latex, err := rendering.RenderLaTeX(updatedPlan, updatedBullets, templatePath, candidateInfo.Name, candidateInfo.Email, candidateInfo.Phone, experienceBank, selectedEducation)
		if err != nil {
			return nil, nil, "", currentViolations, iterationsUsed, fmt.Errorf("failed to render LaTeX at iteration %d: %w", iterationsUsed, err)
		}

		// Write LaTeX to temporary file for validation
		tempTexPath, err := writeTempLaTeX(latex)
		if err != nil {
			return nil, nil, "", currentViolations, iterationsUsed, fmt.Errorf("failed to write temporary LaTeX file at iteration %d: %w", iterationsUsed, err)
		}
		defer func() { _ = cleanupTempFile(tempTexPath) }()

		// 6. Validate LaTeX
		updatedViolations, err := validation.ValidateConstraints(tempTexPath, companyProfile, maxPages, maxCharsPerLine)
		if err != nil {
			return nil, nil, "", currentViolations, iterationsUsed, fmt.Errorf("failed to validate LaTeX at iteration %d: %w", iterationsUsed, err)
		}

		// Update state for next iteration
		currentPlan = updatedPlan
		currentBullets = updatedBullets
		currentViolations = updatedViolations
		finalLaTeX = latex
	}

	return currentPlan, currentBullets, finalLaTeX, currentViolations, iterationsUsed, nil
}

// plansDiffer checks if two plans are different
func plansDiffer(plan1, plan2 *types.ResumePlan) bool {
	if len(plan1.SelectedStories) != len(plan2.SelectedStories) {
		return true
	}

	for i, story1 := range plan1.SelectedStories {
		story2 := plan2.SelectedStories[i]
		if story1.StoryID != story2.StoryID {
			return true
		}
		if len(story1.BulletIDs) != len(story2.BulletIDs) {
			return true
		}
		for j, bulletID1 := range story1.BulletIDs {
			if bulletID1 != story2.BulletIDs[j] {
				return true
			}
		}
	}

	return false
}

// writeTempLaTeX writes LaTeX content to a temporary file and returns the path
func writeTempLaTeX(latex string) (string, error) {
	tmpFile, err := os.CreateTemp("", "resume-*.tex")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = tmpFile.WriteString(latex)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// cleanupTempFile removes a temporary file (best effort)
func cleanupTempFile(path string) error {
	return os.Remove(path)
}
