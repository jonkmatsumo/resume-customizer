// Package repair provides functionality to automatically fix violations in LaTeX resumes.
package repair

import (
	"fmt"

	"github.com/jonathan/resume-customizer/internal/selection"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/jonathan/resume-customizer/internal/validation"
)

// ProposeBulletDrops proposes which bullets to drop based on overflow analysis and relevance scoring.
// It returns a list of drop_bullet repair actions for the least relevant bullets.
// Returns an empty list if no drops are needed or if the inputs are invalid.
func ProposeBulletDrops(
	overflow *validation.OverflowAnalysis,
	bullets *types.RewrittenBullets,
	plan *types.ResumePlan,
	jobProfile *types.JobProfile,
	rankedStories *types.RankedStories,
	experienceBank *types.ExperienceBank,
) []types.RepairAction {
	// Return empty if no overflow or no drops needed
	if overflow == nil || !overflow.MustDrop {
		return []types.RepairAction{}
	}

	// Return empty if no bullets to drop
	if bullets == nil || len(bullets.Bullets) == 0 {
		return []types.RepairAction{}
	}

	// Determine how many bullets to drop
	numToDrop := overflow.BulletsToDropCount()
	if numToDrop <= 0 {
		return []types.RepairAction{}
	}

	// Don't drop more bullets than we have
	if numToDrop > len(bullets.Bullets) {
		numToDrop = len(bullets.Bullets)
	}

	// Score all bullets and get them sorted by relevance (lowest first)
	scoredBullets := selection.ScoreAllBullets(bullets, plan, jobProfile, rankedStories, experienceBank)
	if len(scoredBullets) == 0 {
		return []types.RepairAction{}
	}

	// Create drop actions for the least relevant bullets
	dropActions := make([]types.RepairAction, 0, numToDrop)
	for i := 0; i < numToDrop && i < len(scoredBullets); i++ {
		bullet := scoredBullets[i]
		dropActions = append(dropActions, types.RepairAction{
			Type:     "drop_bullet",
			BulletID: bullet.BulletID,
			StoryID:  bullet.StoryID,
			Reason:   formatDropReason(bullet, overflow),
		})
	}

	return dropActions
}

// formatDropReason creates a human-readable reason for dropping a bullet
func formatDropReason(bullet selection.ScoredBullet, overflow *validation.OverflowAnalysis) string {
	return fmt.Sprintf(
		"Dropping to resolve page overflow (%.1f excess pages). Bullet scored %.2f (story: %.2f, skills: %.2f, efficiency: %.2f, style: %.2f)",
		overflow.ExcessPages,
		bullet.RelevanceScore,
		bullet.Components.StoryRelevance,
		bullet.Components.SkillCoverage,
		bullet.Components.LengthEfficiency,
		bullet.Components.StyleQuality,
	)
}
