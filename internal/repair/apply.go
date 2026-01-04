// Package repair provides functionality to automatically fix violations in LaTeX resumes.
package repair

import (
	"fmt"

	"github.com/jonathan/resume-customizer/internal/types"
)

// ApplyRepairs applies repair actions deterministically to a plan and rewritten bullets
func ApplyRepairs(actions *types.RepairActions, plan *types.ResumePlan, rewrittenBullets *types.RewrittenBullets, rankedStories *types.RankedStories, experienceBank *types.ExperienceBank) (updatedPlan *types.ResumePlan, updatedBullets *types.RewrittenBullets, bulletsToRewrite []string, err error) {
	// Create deep copies to avoid mutating inputs
	planCopy := deepCopyPlan(plan)
	bulletsCopy := deepCopyRewrittenBullets(rewrittenBullets)
	bulletsToRewriteList := make([]string, 0)

	// Process each action in order
	for i, action := range actions.Actions {
		switch action.Type {
		case "shorten_bullet":
			if err := applyShortenBullet(&action, bulletsCopy); err != nil {
				return nil, nil, nil, &ApplyError{
					Message: fmt.Sprintf("failed to apply shorten_bullet action at index %d", i),
					Cause:   err,
				}
			}
			// Add bullet ID to rewrite list
			bulletsToRewriteList = append(bulletsToRewriteList, action.BulletID)

		case "drop_bullet":
			var planChanged bool
			if err := applyDropBullet(&action, planCopy, bulletsCopy); err != nil {
				return nil, nil, nil, &ApplyError{
					Message: fmt.Sprintf("failed to apply drop_bullet action at index %d", i),
					Cause:   err,
				}
			}
			_ = planChanged // Plan may have changed but we don't need to track it here
			// drop_bullet doesn't require rewriting (bullet is removed)

		case "swap_story":
			newBulletIDs, err := applySwapStory(&action, planCopy, bulletsCopy, rankedStories, experienceBank)
			if err != nil {
				return nil, nil, nil, &ApplyError{
					Message: fmt.Sprintf("failed to apply swap_story action at index %d", i),
					Cause:   err,
				}
			}
			// Add all new bullet IDs from replacement story to rewrite list
			bulletsToRewriteList = append(bulletsToRewriteList, newBulletIDs...)

		case "tighten_section":
			// Not implemented yet - skip with warning
			// Could return error in strict mode, but for now we skip

		case "adjust_template_params":
			// Not implemented yet - skip with warning
			// Could return error in strict mode, but for now we skip

		default:
			return nil, nil, nil, &ApplyError{
				Message: fmt.Sprintf("unknown repair action type: %s", action.Type),
			}
		}
	}

	// Recalculate estimated lines for plan after changes
	recalculateEstimatedLines(planCopy, bulletsCopy)

	return planCopy, bulletsCopy, bulletsToRewriteList, nil
}

// applyShortenBullet marks a bullet for rewriting with a new target length
func applyShortenBullet(action *types.RepairAction, bullets *types.RewrittenBullets) error {
	if action.BulletID == "" {
		return fmt.Errorf("bullet_id is required for shorten_bullet action")
	}
	if action.TargetChars == nil {
		return fmt.Errorf("target_chars is required for shorten_bullet action")
	}

	// Find bullet in rewritten bullets
	found := false
	for i := range bullets.Bullets {
		if bullets.Bullets[i].OriginalBulletID == action.BulletID {
			// Mark for rewrite by storing target_chars in a way we can track
			// For now, we'll just note that it needs rewrite - the actual rewriting
			// will happen in the loop with the target length from the action
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("bullet_id %s not found in rewritten bullets", action.BulletID)
	}

	return nil
}

// applyDropBullet removes a bullet from the plan and rewritten bullets
func applyDropBullet(action *types.RepairAction, plan *types.ResumePlan, bullets *types.RewrittenBullets) error {
	if action.BulletID == "" {
		return fmt.Errorf("bullet_id is required for drop_bullet action")
	}

	// Find and remove bullet from plan
	for i := range plan.SelectedStories {
		story := &plan.SelectedStories[i]
		bulletIdx := -1
		for j, bulletID := range story.BulletIDs {
			if bulletID == action.BulletID {
				bulletIdx = j
				break
			}
		}

		if bulletIdx >= 0 {
			// Remove bullet ID from story
			story.BulletIDs = append(story.BulletIDs[:bulletIdx], story.BulletIDs[bulletIdx+1:]...)

			// If story has no bullets left, remove story from plan
			if len(story.BulletIDs) == 0 {
				plan.SelectedStories = append(plan.SelectedStories[:i], plan.SelectedStories[i+1:]...)
			}
			break
		}
	}

	// Remove bullet from rewritten bullets
	bulletIdx := -1
	for i, bullet := range bullets.Bullets {
		if bullet.OriginalBulletID == action.BulletID {
			bulletIdx = i
			break
		}
	}

	if bulletIdx >= 0 {
		bullets.Bullets = append(bullets.Bullets[:bulletIdx], bullets.Bullets[bulletIdx+1:]...)
	}

	return nil
}

// applySwapStory replaces a story in the plan with an alternative from ranked stories
// Returns the bullet IDs from the replacement story that need to be rewritten
func applySwapStory(action *types.RepairAction, plan *types.ResumePlan, bullets *types.RewrittenBullets, rankedStories *types.RankedStories, experienceBank *types.ExperienceBank) ([]string, error) {
	if action.StoryID == "" {
		return nil, fmt.Errorf("story_id is required for swap_story action")
	}

	// Find story in plan
	storyIdx := -1
	for i := range plan.SelectedStories {
		if plan.SelectedStories[i].StoryID == action.StoryID {
			storyIdx = i
			break
		}
	}

	if storyIdx < 0 {
		return nil, fmt.Errorf("story_id %s not found in plan", action.StoryID)
	}

	// Capture old story info before modifying plan
	oldStory := plan.SelectedStories[storyIdx]
	oldStoryBulletIDs := make(map[string]bool)
	for _, bulletID := range oldStory.BulletIDs {
		oldStoryBulletIDs[bulletID] = true
	}

	// Find replacement story from ranked stories (next best alternative not already in plan)
	// Build set of existing story IDs
	existingStoryIDs := make(map[string]bool)
	for _, selectedStory := range plan.SelectedStories {
		existingStoryIDs[selectedStory.StoryID] = true
	}

	var replacementStory *types.RankedStory
	for i := range rankedStories.Ranked {
		rankedStory := &rankedStories.Ranked[i]
		if !existingStoryIDs[rankedStory.StoryID] && rankedStory.StoryID != action.StoryID {
			replacementStory = rankedStory
			break
		}
	}

	if replacementStory == nil {
		return nil, fmt.Errorf("no suitable replacement story found in ranked stories")
	}

	// Find replacement story in experience bank to get bullet IDs
	var replacementExperienceStory *types.Story
	for i := range experienceBank.Stories {
		if experienceBank.Stories[i].ID == replacementStory.StoryID {
			replacementExperienceStory = &experienceBank.Stories[i]
			break
		}
	}

	if replacementExperienceStory == nil {
		return nil, fmt.Errorf("replacement story %s not found in experience bank", replacementStory.StoryID)
	}

	// Determine which bullets to use from replacement story
	// For now, use all bullets from replacement story, but could be smarter
	replacementBulletIDs := make([]string, 0, len(replacementExperienceStory.Bullets))
	for _, bullet := range replacementExperienceStory.Bullets {
		replacementBulletIDs = append(replacementBulletIDs, bullet.ID)
	}

	// Replace story in plan
	plan.SelectedStories[storyIdx] = types.SelectedStory{
		StoryID:        replacementStory.StoryID,
		BulletIDs:      replacementBulletIDs,
		Section:        oldStory.Section,        // Preserve section
		EstimatedLines: oldStory.EstimatedLines, // Will be recalculated
	}

	// Remove old story's bullets from rewritten bullets
	// (oldStoryBulletIDs was already built above)

	// Remove bullets that belong to the old story
	newBulletsList := make([]types.RewrittenBullet, 0, len(bullets.Bullets))
	for _, bullet := range bullets.Bullets {
		if !oldStoryBulletIDs[bullet.OriginalBulletID] {
			newBulletsList = append(newBulletsList, bullet)
		}
	}

	bullets.Bullets = newBulletsList

	// Return the bullet IDs from the replacement story that need to be rewritten
	return replacementBulletIDs, nil
}

// recalculateEstimatedLines updates estimated lines in the plan based on current bullets
func recalculateEstimatedLines(plan *types.ResumePlan, bullets *types.RewrittenBullets) {
	// Build map of bullet_id -> estimated_lines
	bulletLinesMap := make(map[string]int)
	for _, bullet := range bullets.Bullets {
		bulletLinesMap[bullet.OriginalBulletID] = bullet.EstimatedLines
	}

	// Update estimated lines for each story in plan
	for i := range plan.SelectedStories {
		story := &plan.SelectedStories[i]
		totalLines := 0
		for _, bulletID := range story.BulletIDs {
			if lines, exists := bulletLinesMap[bulletID]; exists {
				totalLines += lines
			}
		}
		story.EstimatedLines = totalLines
	}
}

// deepCopyPlan creates a deep copy of a ResumePlan
func deepCopyPlan(plan *types.ResumePlan) *types.ResumePlan {
	copyPlan := &types.ResumePlan{
		SelectedStories: make([]types.SelectedStory, len(plan.SelectedStories)),
		SpaceBudget:     plan.SpaceBudget, // SpaceBudget contains basic types, shallow copy is OK
		Coverage:        plan.Coverage,    // Coverage contains basic types, shallow copy is OK
	}

	for i, story := range plan.SelectedStories {
		copyPlan.SelectedStories[i] = types.SelectedStory{
			StoryID:        story.StoryID,
			BulletIDs:      make([]string, len(story.BulletIDs)),
			Section:        story.Section,
			EstimatedLines: story.EstimatedLines,
		}
		copy(copyPlan.SelectedStories[i].BulletIDs, story.BulletIDs)
	}

	// Deep copy sections map if present
	if plan.SpaceBudget.Sections != nil {
		copyPlan.SpaceBudget.Sections = make(map[string]int)
		for k, v := range plan.SpaceBudget.Sections {
			copyPlan.SpaceBudget.Sections[k] = v
		}
	}

	return copyPlan
}

// deepCopyRewrittenBullets creates a deep copy of RewrittenBullets
func deepCopyRewrittenBullets(bullets *types.RewrittenBullets) *types.RewrittenBullets {
	copyBullets := &types.RewrittenBullets{
		Bullets: make([]types.RewrittenBullet, len(bullets.Bullets)),
	}

	for i, bullet := range bullets.Bullets {
		copyBullets.Bullets[i] = types.RewrittenBullet{
			OriginalBulletID: bullet.OriginalBulletID,
			FinalText:        bullet.FinalText,
			LengthChars:      bullet.LengthChars,
			EstimatedLines:   bullet.EstimatedLines,
			StyleChecks:      bullet.StyleChecks, // StyleChecks contains basic types, shallow copy is OK
		}
	}

	return copyBullets
}
