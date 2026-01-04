// Package repair provides functionality to automatically fix violations in LaTeX resumes.
package repair

import (
	"context"
	"fmt"
	"os"

	"github.com/jonathan/resume-customizer/internal/rendering"
	"github.com/jonathan/resume-customizer/internal/rewriting"
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
		updatedPlan, updatedBullets, bulletsToRewrite, err := ApplyRepairs(repairActions, currentPlan, currentBullets, rankedStories, experienceBank)
		if err != nil {
			return nil, nil, "", currentViolations, iterationsUsed, fmt.Errorf("failed to apply repairs at iteration %d: %w", iterationsUsed, err)
		}

		// 3. Handle bullet rewriting if needed
		planChanged := plansDiffer(currentPlan, updatedPlan)

		// Determine which bullets need rewriting
		allBulletsToRewrite := make([]string, 0, len(bulletsToRewrite))
		allBulletsToRewrite = append(allBulletsToRewrite, bulletsToRewrite...)

		// Handle plan changes (swap_story adds new bullets)
		if planChanged {
			// Find new bullets not in currentBullets
			newBulletIDs := findNewBulletIDs(updatedPlan, currentBullets)
			allBulletsToRewrite = append(allBulletsToRewrite, newBulletIDs...)
		}

		// Rewrite only if there are bullets to rewrite
		if len(allBulletsToRewrite) > 0 {
			rewritten, err := rewriting.RewriteBulletsSelective(
				ctx,
				updatedBullets, // Current bullets (may have been modified by ApplyRepairs)
				allBulletsToRewrite,
				jobProfile,
				companyProfile,
				experienceBank,
				apiKey,
			)
			if err != nil {
				return nil, nil, "", currentViolations, iterationsUsed, fmt.Errorf("failed to rewrite bullets at iteration %d: %w", iterationsUsed, err)
			}
			updatedBullets = rewritten
		}
		// If no bullets to rewrite and plan didn't change, use updatedBullets from ApplyRepairs (which may have dropped bullets)

		// 5. Render LaTeX
		latex, lineMap, err := rendering.RenderLaTeX(updatedPlan, updatedBullets, templatePath, candidateInfo.Name, candidateInfo.Email, candidateInfo.Phone, experienceBank, selectedEducation)
		if err != nil {
			return nil, nil, "", currentViolations, iterationsUsed, fmt.Errorf("failed to render LaTeX at iteration %d: %w", iterationsUsed, err)
		}

		// Write LaTeX to temporary file for validation
		tempTexPath, err := writeTempLaTeX(latex)
		if err != nil {
			return nil, nil, "", currentViolations, iterationsUsed, fmt.Errorf("failed to write temporary LaTeX file at iteration %d: %w", iterationsUsed, err)
		}
		defer func() { _ = cleanupTempFile(tempTexPath) }()

		// 6. Validate LaTeX with line-to-bullet mapping
		var validationOpts *validation.Options
		if lineMap != nil {
			validationOpts = &validation.Options{
				LineToBulletMap: lineMap.LineToBullet,
				Bullets:         updatedBullets,
				Plan:            updatedPlan,
			}
		}
		updatedViolations, err := validation.ValidateConstraints(tempTexPath, companyProfile, maxPages, maxCharsPerLine, validationOpts)
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

// extractBulletIDsFromActions extracts bullet IDs from repair actions that require rewriting
func extractBulletIDsFromActions(actions *types.RepairActions) []string {
	bulletIDs := make([]string, 0)
	for _, action := range actions.Actions {
		switch action.Type {
		case "shorten_bullet":
			if action.BulletID != "" {
				bulletIDs = append(bulletIDs, action.BulletID)
			}
		case "swap_story":
			// Bullet IDs from swap_story are handled in ApplyRepairs
			// This function is for extracting from actions directly if needed
		}
	}
	return bulletIDs
}

// findNewBulletIDs finds bullet IDs in the plan that are not in currentBullets
func findNewBulletIDs(plan *types.ResumePlan, currentBullets *types.RewrittenBullets) []string {
	// Build set of existing bullet IDs
	existingBulletIDs := make(map[string]bool)
	for _, bullet := range currentBullets.Bullets {
		existingBulletIDs[bullet.OriginalBulletID] = true
	}

	// Find bullet IDs in plan that don't exist in currentBullets
	newBulletIDs := make([]string, 0)
	for _, story := range plan.SelectedStories {
		for _, bulletID := range story.BulletIDs {
			if !existingBulletIDs[bulletID] {
				newBulletIDs = append(newBulletIDs, bulletID)
			}
		}
	}

	return newBulletIDs
}
