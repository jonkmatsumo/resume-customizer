package selection

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolveKnapsack_SimpleCase(t *testing.T) {
	// Simple test case: 2 stories, can fit both
	stories := []*types.Story{
		{
			ID: "story_001",
			Bullets: []types.Bullet{
				{ID: "bullet_001", LengthChars: 50, Skills: []string{"Go"}},
			},
		},
		{
			ID: "story_002",
			Bullets: []types.Bullet{
				{ID: "bullet_002", LengthChars: 50, Skills: []string{"Python"}},
			},
		},
	}

	rankedStories := []*types.RankedStory{
		{StoryID: "story_001", RelevanceScore: 0.8},
		{StoryID: "story_002", RelevanceScore: 0.7},
	}

	// Build story values (one combination per story: all bullets)
	storyValues := make(map[int][]storyValue)
	for i, story := range stories {
		rankedStory := rankedStories[i]
		combinations := generateBulletCombinations(story.Bullets)
		values := make([]storyValue, 0, len(combinations))
		for _, combo := range combinations {
			value := computeStoryValue(rankedStory, combo, nil)
			values = append(values, value)
		}
		storyValues[i] = values
	}

	selections, score, err := solveKnapsack(stories, storyValues, 10, 10)
	require.NoError(t, err)
	assert.Greater(t, len(selections), 0)
	assert.Greater(t, score, 0.0)
}

func TestSolveKnapsack_ConstraintViolation(t *testing.T) {
	// Test case: stories that don't fit within constraints
	stories := []*types.Story{
		{
			ID: "story_001",
			Bullets: []types.Bullet{
				{ID: "bullet_001", LengthChars: 1000}, // Very long, exceeds line limit
			},
		},
	}

	rankedStories := []*types.RankedStory{
		{StoryID: "story_001", RelevanceScore: 0.8},
	}

	storyValues := make(map[int][]storyValue)
	combinations := generateBulletCombinations(stories[0].Bullets)
	values := make([]storyValue, 0, len(combinations))
	for _, combo := range combinations {
		value := computeStoryValue(rankedStories[0], combo, nil)
		values = append(values, value)
	}
	storyValues[0] = values

	// Try with very small constraints
	selections, score, err := solveKnapsack(stories, storyValues, 1, 1)
	// Should return empty selection (no valid solution)
	if err != nil {
		assert.Contains(t, err.Error(), "no valid solution")
	} else {
		assert.Empty(t, selections)
		assert.Equal(t, 0.0, score)
	}
}

func TestSolveKnapsack_OptimalSelection(t *testing.T) {
	// Test case: 2 stories, only one fits
	// story_001 has higher value but costs more
	// story_002 has lower value but costs less
	// With tight constraints, should choose story_002
	stories := []*types.Story{
		{
			ID: "story_001",
			Bullets: []types.Bullet{
				{ID: "bullet_001", LengthChars: 250, Skills: []string{"Go"}},
				{ID: "bullet_002", LengthChars: 250, Skills: []string{"Kubernetes"}},
			},
		},
		{
			ID: "story_002",
			Bullets: []types.Bullet{
				{ID: "bullet_003", LengthChars: 50, Skills: []string{"Python"}},
			},
		},
	}

	rankedStories := []*types.RankedStory{
		{StoryID: "story_001", RelevanceScore: 0.9}, // Higher relevance
		{StoryID: "story_002", RelevanceScore: 0.5}, // Lower relevance
	}

	storyValues := make(map[int][]storyValue)
	for i, story := range stories {
		rankedStory := rankedStories[i]
		combinations := generateBulletCombinations(story.Bullets)
		values := make([]storyValue, 0, len(combinations))
		for _, combo := range combinations {
			value := computeStoryValue(rankedStory, combo, nil)
			values = append(values, value)
		}
		storyValues[i] = values
	}

	// Tight constraints: can only fit story_002
	selections, _, err := solveKnapsack(stories, storyValues, 2, 2)
	require.NoError(t, err)
	// Should select story_002 (fits) rather than story_001 (doesn't fit)
	if len(selections) > 0 {
		assert.Equal(t, "story_002", selections[0].storyID)
	}
}

func TestBacktrack(t *testing.T) {
	// Create a chain of states
	state1 := &dpState{
		score:  1.0,
		parent: nil,
		selection: &StorySelection{
			storyID:   "story_001",
			bulletIDs: []string{"bullet_001"},
		},
	}

	state2 := &dpState{
		score:  2.0,
		parent: state1,
		selection: &StorySelection{
			storyID:   "story_002",
			bulletIDs: []string{"bullet_002"},
		},
	}

	selections := backtrack(state2)
	assert.Len(t, selections, 2)
	assert.Equal(t, "story_001", selections[0].storyID)
	assert.Equal(t, "story_002", selections[1].storyID)
}

func TestBacktrack_SingleSelection(t *testing.T) {
	state := &dpState{
		score:  1.0,
		parent: nil,
		selection: &StorySelection{
			storyID:   "story_001",
			bulletIDs: []string{"bullet_001"},
		},
	}

	selections := backtrack(state)
	assert.Len(t, selections, 1)
	assert.Equal(t, "story_001", selections[0].storyID)
}

func TestBacktrack_NoSelection(t *testing.T) {
	state := &dpState{
		score:     1.0,
		parent:    nil,
		selection: nil,
	}

	selections := backtrack(state)
	assert.Empty(t, selections)
}
