package selection

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestComputeStoryValue(t *testing.T) {
	story := &types.Story{
		ID: "story_001",
		Bullets: []types.Bullet{
			{
				ID:          "bullet_001",
				LengthChars: 90,
				Skills:      []string{"Go"},
			},
			{
				ID:          "bullet_002",
				LengthChars: 100,
				Skills:      []string{"Kubernetes"},
			},
		},
	}

	rankedStory := &types.RankedStory{
		StoryID:        "story_001",
		RelevanceScore: 0.8,
	}

	skillTargets := &types.SkillTargets{
		Skills: []types.Skill{
			{Name: "Go", Weight: 1.0},
			{Name: "Kubernetes", Weight: 0.7},
		},
	}

	// Test with single bullet
	value1 := computeStoryValue(rankedStory, story.Bullets[0:1], skillTargets)
	assert.Equal(t, 1, value1.CostBullets)
	assert.Equal(t, 1, value1.CostLines) // 90 chars = 1 line
	assert.Equal(t, []string{"bullet_001"}, value1.BulletIDs)
	assert.Greater(t, value1.Value, 0.0)

	// Test with both bullets
	value2 := computeStoryValue(rankedStory, story.Bullets, skillTargets)
	assert.Equal(t, 2, value2.CostBullets)
	assert.Equal(t, 3, value2.CostLines) // 90 + 100 = 190 chars = 3 lines
	assert.Equal(t, []string{"bullet_001", "bullet_002"}, value2.BulletIDs)
	assert.Greater(t, value2.Value, value1.Value) // More bullets should have higher value
}

func TestGenerateBulletCombinations(t *testing.T) {
	bullets := []types.Bullet{
		{ID: "bullet_001"},
		{ID: "bullet_002"},
		{ID: "bullet_003"},
	}

	combinations := generateBulletCombinations(bullets)

	// Should have 2^n - 1 combinations (all non-empty subsets)
	expectedCount := (1 << len(bullets)) - 1
	assert.Equal(t, expectedCount, len(combinations))

	// Verify all combinations are unique and non-empty
	seen := make(map[string]bool)
	for _, combo := range combinations {
		assert.Greater(t, len(combo), 0, "combination should not be empty")

		// Create a key for this combination
		key := ""
		for _, bullet := range combo {
			key += bullet.ID + ","
		}
		assert.False(t, seen[key], "combination should be unique: %v", key)
		seen[key] = true
	}
}

func TestGenerateBulletCombinations_Empty(t *testing.T) {
	combinations := generateBulletCombinations([]types.Bullet{})
	assert.Nil(t, combinations)
}

func TestGenerateBulletCombinations_SingleBullet(t *testing.T) {
	bullets := []types.Bullet{
		{ID: "bullet_001"},
	}

	combinations := generateBulletCombinations(bullets)
	assert.Len(t, combinations, 1)
	assert.Equal(t, []types.Bullet{{ID: "bullet_001"}}, combinations[0])
}
