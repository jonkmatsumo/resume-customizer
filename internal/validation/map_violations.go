// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"github.com/jonathan/resume-customizer/internal/types"
)

// MapViolationsToBullets maps violations to specific bullet IDs using line-to-bullet mapping
func MapViolationsToBullets(
	violations *types.Violations,
	lineToBulletMap map[int]string, // Line number → bullet_id
	bullets *types.RewrittenBullets, // For bullet text and story ID lookup
	plan *types.ResumePlan, // For story ID lookup
	forbiddenPhraseMap map[string][]string, // bulletID → list of forbidden phrases found (optional)
) *types.Violations {
	if violations == nil {
		return nil
	}

	// If no mapping provided, return violations as-is
	if len(lineToBulletMap) == 0 {
		return violations
	}

	// Build a map of rewritten bullets by original bullet ID for quick lookup
	bulletMap := make(map[string]*types.RewrittenBullet)
	if bullets != nil {
		for i := range bullets.Bullets {
			bullet := &bullets.Bullets[i]
			bulletMap[bullet.OriginalBulletID] = bullet
		}
	}

	// Build a map of story IDs by bullet ID from the plan
	storyIDByBulletID := make(map[string]string)
	if plan != nil {
		for _, selectedStory := range plan.SelectedStories {
			for _, bulletID := range selectedStory.BulletIDs {
				storyIDByBulletID[bulletID] = selectedStory.StoryID
			}
		}
	}

	// Map violations to bullets
	mappedViolations := make([]types.Violation, 0, len(violations.Violations))
	for _, violation := range violations.Violations {
		mappedViolation := violation // Copy

		// If violation has a line number, try to map it to a bullet
		if violation.LineNumber != nil {
			bulletID, found := lineToBulletMap[*violation.LineNumber]
			if found {
				// For forbidden_phrase violations, verify the bullet is in the forbidden phrase map
				// if the map is provided (for additional confirmation)
				if violation.Type == "forbidden_phrase" && forbiddenPhraseMap != nil && len(forbiddenPhraseMap) > 0 {
					// Only map if bullet is in forbidden phrase map (confirms we detected it during rewriting)
					if _, hasPhrases := forbiddenPhraseMap[bulletID]; !hasPhrases {
						// Bullet not in forbidden phrase map - skip mapping for forbidden_phrase violations
						// This ensures we only map violations for bullets we detected during rewriting
						mappedViolations = append(mappedViolations, mappedViolation)
						continue
					}
				}

				// Set bullet ID
				mappedViolation.BulletID = &bulletID

				// Set story ID if available
				if storyID, ok := storyIDByBulletID[bulletID]; ok {
					mappedViolation.StoryID = &storyID
				}

				// Set bullet text if available
				if bullet, ok := bulletMap[bulletID]; ok {
					mappedViolation.BulletText = &bullet.FinalText
				}
			}
		}
		// If no line number, leave bullet_id as nil (e.g., page_overflow violations)

		mappedViolations = append(mappedViolations, mappedViolation)
	}

	return &types.Violations{Violations: mappedViolations}
}
