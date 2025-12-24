package selection

import (
	"math"

	"github.com/jonathan/resume-customizer/internal/types"
)

// SelectHybrid selects bullets using a two-phase hybrid approach:
// 1. Greedy Phase: Selects best skill matches up to (maxLines * ratio).
// 2. Knapsack Phase: Selects highest value content for remaining space.
func SelectHybrid(
	stories []*types.Story,
	rankedStories *types.RankedStories,
	skillTargets *types.SkillTargets,
	maxLines int,
	skillMatchRatio float64,
) ([]storySelection, float64, error) {

	// Calculate line budget for each phase
	greedyBudget := int(math.Floor(float64(maxLines) * skillMatchRatio))

	// Phase 1: Greedy Selection for Skills
	greedySelections, greedyScore, err := SelectGreedy(stories, skillTargets, greedyBudget)
	if err != nil {
		return nil, 0.0, err
	}

	// Calculate used lines in Phase 1
	usedLines := 0
	selectedBulletMap := make(map[string]bool) // Track selected bullets to avoid duplicates

	for _, sel := range greedySelections {
		// Find story to get bullet lengths
		var story *types.Story
		for _, s := range stories { // Inefficient lookup but acceptable for small N
			if s.ID == sel.storyID {
				story = s
				break
			}
		}

		if story != nil {
			for _, bid := range sel.bulletIDs {
				selectedBulletMap[bid] = true
				for _, b := range story.Bullets {
					if b.ID == bid {
						usedLines += estimateLines(b.LengthChars)
						break
					}
				}
			}
		}
	}

	remainingLines := maxLines - usedLines
	if remainingLines <= 0 {
		return greedySelections, greedyScore, nil
	}

	// Phase 2: Knapsack for Overall Value
	// We need to construct a filtered view of stories/bullets that excludes
	// what was already selected in Phase 1.

	// Create filtered stories (deep copy structure but point to same data)
	// We only want to include bullets that start NOT selected.
	// NOTE: This modifies the problem space for knapsack.
	// Knapsack expects full control. We can simply mark selected bullets as "already taken"
	// but knapsack implementation assumes it's building from scratch.
	// Easier Strategy: run Knapsack on the SUBSET of available bullets.

	filteredStories := make([]*types.Story, 0, len(stories))
	filteredRanked := make([]*types.RankedStory, 0, len(rankedStories.Ranked))

	// Map ranked stories for alignment
	rankedMap := make(map[string]*types.RankedStory)
	for i := range rankedStories.Ranked {
		rankedMap[rankedStories.Ranked[i].StoryID] = &rankedStories.Ranked[i]
	}

	for _, story := range stories {
		// Create a new story instance with only unselected bullets
		newBullets := make([]types.Bullet, 0)
		for _, b := range story.Bullets {
			if !selectedBulletMap[b.ID] {
				newBullets = append(newBullets, b)
			}
		}

		if len(newBullets) > 0 {
			filteredStories = append(filteredStories, &types.Story{
				ID:        story.ID,
				Role:      story.Role,
				Company:   story.Company,
				StartDate: story.StartDate,
				EndDate:   story.EndDate,
				Bullets:   newBullets,
			})
			filteredRanked = append(filteredRanked, rankedMap[story.ID])
		}
	}

	// Prepare data for Knapsack
	storyValues := make(map[int][]storyValue)
	for i, story := range filteredStories {
		combinations := generateBulletCombinations(story.Bullets)
		values := make([]storyValue, 0, len(combinations))
		for _, combo := range combinations {
			// We need a ranked story object to compute value.
			// We use the original ranked story metrics.
			value := computeStoryValue(filteredRanked[i], combo, skillTargets)
			values = append(values, value)
		}
		storyValues[i] = values
	}

	// Solve Knapsack for remaining space
	knapsackSelections, knapsackScore, err := solveKnapsack(
		filteredStories,
		storyValues,
		1000,
		remainingLines,
	)

	// Handle case where no solution found (e.g. no bullets left)
	if err != nil {
		// If it's just "no valid solution found" because maybe remaining space is too small
		// for any bullet, we just return greedy result.
		return greedySelections, greedyScore, nil
	}

	// Merge selections
	// We need to merge knapsack selections back into greedy selections.
	// Since both return []storySelection (storyID -> bullets), we need to consolidate.

	finalMap := make(map[string][]string)

	// Add Greedy
	for _, sel := range greedySelections {
		finalMap[sel.storyID] = append(finalMap[sel.storyID], sel.bulletIDs...)
	}

	// Add Knapsack
	for _, sel := range knapsackSelections {
		finalMap[sel.storyID] = append(finalMap[sel.storyID], sel.bulletIDs...)
	}

	// Convert back to slice
	finalSelections := make([]storySelection, 0, len(finalMap))
	for sID, bIDs := range finalMap {
		finalSelections = append(finalSelections, storySelection{
			storyID:   sID,
			bulletIDs: bIDs,
		})
	}

	return finalSelections, greedyScore + knapsackScore, nil
}
