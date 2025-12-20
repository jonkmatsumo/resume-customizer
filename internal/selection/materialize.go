// Package selection provides functionality to select optimal stories and bullets for a resume plan.
package selection

import (
	"fmt"

	"github.com/jonathan/resume-customizer/internal/types"
)

// MaterializeBullets extracts the actual bullet data from ExperienceBank based on the ResumePlan.
// It preserves the order from the plan and verifies all referenced stories and bullets exist.
func MaterializeBullets(plan *types.ResumePlan, experienceBank *types.ExperienceBank) (*types.SelectedBullets, error) {
	// Build story ID → Story map for O(1) lookup
	storyMap := make(map[string]*types.Story)
	for i := range experienceBank.Stories {
		storyMap[experienceBank.Stories[i].ID] = &experienceBank.Stories[i]
	}

	// Build bullet lookup maps: story ID → (bullet ID → Bullet)
	bulletMaps := make(map[string]map[string]*types.Bullet)
	for i := range experienceBank.Stories {
		story := &experienceBank.Stories[i]
		bulletMap := make(map[string]*types.Bullet)
		for j := range story.Bullets {
			bulletMap[story.Bullets[j].ID] = &story.Bullets[j]
		}
		bulletMaps[story.ID] = bulletMap
	}

	// Materialize bullets in order from plan
	result := make([]types.SelectedBullet, 0)

	for _, selectedStory := range plan.SelectedStories {
		// Verify story exists
		story, exists := storyMap[selectedStory.StoryID]
		if !exists {
			return nil, &Error{
				Message: fmt.Sprintf("story not found in experience bank (story_id: %s)", selectedStory.StoryID),
				Cause:   nil,
			}
		}

		// Get bullet map for this story
		bulletMap, exists := bulletMaps[selectedStory.StoryID]
		if !exists {
			return nil, &Error{
				Message: "bullet map not found for story",
				Cause:   nil,
			}
		}

		// Extract bullets in order
		for _, bulletID := range selectedStory.BulletIDs {
			bullet, exists := bulletMap[bulletID]
			if !exists {
				return nil, &Error{
					Message: fmt.Sprintf("bullet not found in story (story_id: %s, bullet_id: %s)", selectedStory.StoryID, bulletID),
					Cause:   nil,
				}
			}

			// Copy skills slice to avoid sharing references
			skills := make([]string, len(bullet.Skills))
			copy(skills, bullet.Skills)

			selectedBullet := types.SelectedBullet{
				ID:          bullet.ID,
				StoryID:     story.ID,
				Text:        bullet.Text,
				Skills:      skills,
				Metrics:     bullet.Metrics,
				LengthChars: bullet.LengthChars,
			}

			result = append(result, selectedBullet)
		}
	}

	return &types.SelectedBullets{
		Bullets: result,
	}, nil
}
