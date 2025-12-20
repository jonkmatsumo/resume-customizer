// Package selection provides functionality to select optimal stories and bullets for a resume plan.
package selection

import (
	"github.com/jonathan/resume-customizer/internal/types"
)

// storyValue represents the value and cost of selecting a story with a specific bullet combination
type storyValue struct {
	Value       float64
	CostBullets int
	CostLines   int
	BulletIDs   []string
}

// computeStoryValue computes the value of selecting a story with a specific set of bullets
func computeStoryValue(
	rankedStory *types.RankedStory,
	bulletCombination []types.Bullet,
	skillTargets *types.SkillTargets,
) storyValue {
	// Compute skill coverage score for the selected bullets
	skillCoverage := computeSkillCoverageScore(bulletCombination, skillTargets)

	// Compute total value using weighted combination
	value := defaultRelevanceWeight*rankedStory.RelevanceScore + defaultSkillWeight*skillCoverage

	// Compute costs
	costBullets := len(bulletCombination)
	costLines := 0
	bulletIDs := make([]string, 0, len(bulletCombination))
	for _, bullet := range bulletCombination {
		costLines += estimateLines(bullet.LengthChars)
		bulletIDs = append(bulletIDs, bullet.ID)
	}

	return storyValue{
		Value:       value,
		CostBullets: costBullets,
		CostLines:   costLines,
		BulletIDs:   bulletIDs,
	}
}

// generateBulletCombinations generates all valid non-empty combinations of bullets for a story
// This is used to try different subsets of bullets from each story
func generateBulletCombinations(bullets []types.Bullet) [][]types.Bullet {
	if len(bullets) == 0 {
		return nil
	}

	// For now, we'll use a simple approach: try all bullets, or individual bullets
	// In the future, this could be optimized to try only promising combinations
	combinations := make([][]types.Bullet, 0)

	// Generate all non-empty subsets (power set)
	n := len(bullets)
	total := 1 << n // 2^n combinations

	for i := 1; i < total; i++ { // Start from 1 to exclude empty set
		combination := make([]types.Bullet, 0)
		for j := 0; j < n; j++ {
			if i&(1<<j) != 0 {
				combination = append(combination, bullets[j])
			}
		}
		combinations = append(combinations, combination)
	}

	return combinations
}
