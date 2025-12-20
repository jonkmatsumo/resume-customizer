// Package selection provides functionality to select optimal stories and bullets for a resume plan.
package selection

import (
	"fmt"
	"math"

	"github.com/jonathan/resume-customizer/internal/types"
)

// dpState represents a state in the DP table with backtracking information
type dpState struct {
	score       float64
	parent      *dpState
	storyIndex  int
	bulletsUsed int
	linesUsed   int
	selection   *storySelection // Selection that led to this state
}

// storySelection represents which bullets were selected from a story
type storySelection struct {
	storyID   string
	bulletIDs []string
}

// solveKnapsack solves the knapsack problem using dynamic programming
// Returns the optimal selection of stories and bullets
func solveKnapsack(
	stories []*types.Story,
	storyValues map[int][]storyValue, // story index -> list of value options
	maxBullets, maxLines int,
) ([]storySelection, float64, error) {
	if len(stories) == 0 {
		return nil, 0.0, nil
	}

	// Initialize DP table: dp[story_idx][bullets][lines] = best score
	// We use a map-based approach for flexibility, but could use 3D array for performance
	dp := make(map[int]map[int]map[int]*dpState)

	// Initialize base state: no stories selected, no bullets/lines used
	dp[-1] = make(map[int]map[int]*dpState)
	dp[-1][0] = make(map[int]*dpState)
	dp[-1][0][0] = &dpState{
		score:       0.0,
		parent:      nil,
		storyIndex:  -1,
		bulletsUsed: 0,
		linesUsed:   0,
		selection:   nil,
	}

	// Fill DP table
	for i := 0; i < len(stories); i++ {
		dp[i] = make(map[int]map[int]*dpState)

		// Try including story i with different bullet combinations
		values, hasValues := storyValues[i]
		if hasValues {
			for _, valueOption := range values {
				// Try all previous states that can accommodate this selection
				if prevStates, exists := dp[i-1]; exists {
					for prevBullets := range prevStates {
						for prevLines := range prevStates[prevBullets] {
							newBullets := prevBullets + valueOption.CostBullets
							newLines := prevLines + valueOption.CostLines

							// Check constraints
							if newBullets > maxBullets || newLines > maxLines {
								continue
							}

							prevState := prevStates[prevBullets][prevLines]
							if prevState == nil {
								continue
							}

							newScore := prevState.score + valueOption.Value

							// Initialize new state slots if needed
							if dp[i][newBullets] == nil {
								dp[i][newBullets] = make(map[int]*dpState)
							}

							// Update if this is better than existing state
							existing := dp[i][newBullets][newLines]
							if existing == nil || newScore > existing.score {
								dp[i][newBullets][newLines] = &dpState{
									score:       newScore,
									parent:      prevState,
									storyIndex:  i,
									bulletsUsed: newBullets,
									linesUsed:   newLines,
									selection: &storySelection{
										storyID:   stories[i].ID,
										bulletIDs: valueOption.BulletIDs,
									},
								}
							}
						}
					}
				}
			}
		}

		// Copy previous states (option: don't include story i)
		// This must happen after trying to include story i, so we can compare
		if prevStates, exists := dp[i-1]; exists {
			for bullets := range prevStates {
				if dp[i][bullets] == nil {
					dp[i][bullets] = make(map[int]*dpState)
				}
				for lines := range prevStates[bullets] {
					if prevState := prevStates[bullets][lines]; prevState != nil {
						// Only copy if we haven't already set a better state by including story i
						existing := dp[i][bullets][lines]
						if existing == nil || prevState.score > existing.score {
							// Create a new state object (don't just copy pointer) to avoid aliasing
							dp[i][bullets][lines] = &dpState{
								score:       prevState.score,
								parent:      prevState.parent,
								storyIndex:  prevState.storyIndex,
								bulletsUsed: prevState.bulletsUsed,
								linesUsed:   prevState.linesUsed,
								selection:   prevState.selection,
							}
						}
					}
				}
			}
		}
	}

	// Find best final state
	bestState := findBestState(dp[len(stories)-1], maxBullets, maxLines)
	if bestState == nil {
		return nil, 0.0, fmt.Errorf("no valid solution found")
	}

	// Backtrack to reconstruct solution
	selections := backtrack(bestState)

	return selections, bestState.score, nil
}

// findBestState finds the state with the highest score in the final DP layer
func findBestState(finalLayer map[int]map[int]*dpState, maxBullets, maxLines int) *dpState {
	var best *dpState
	bestScore := math.Inf(-1)

	for bullets := 0; bullets <= maxBullets; bullets++ {
		if linesMap, exists := finalLayer[bullets]; exists {
			for lines := 0; lines <= maxLines; lines++ {
				if state := linesMap[lines]; state != nil {
					if state.score > bestScore {
						bestScore = state.score
						best = state
					}
				}
			}
		}
	}

	return best
}

// backtrack reconstructs the solution by following parent pointers
func backtrack(finalState *dpState) []storySelection {
	var selections []storySelection

	current := finalState
	for current != nil && current.selection != nil {
		selections = append(selections, *current.selection)
		current = current.parent
	}

	// Reverse to get chronological order
	for i, j := 0, len(selections)-1; i < j; i, j = i+1, j-1 {
		selections[i], selections[j] = selections[j], selections[i]
	}

	return selections
}
