// Package selection provides functionality to select and score resume content.
package selection

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScoreBulletRelevance_HighRelevanceStory(t *testing.T) {
	bullet := types.RewrittenBullet{
		OriginalBulletID: "b1",
		LengthChars:      100,
		StyleChecks: types.StyleChecks{
			StrongVerb:   true,
			Quantified:   true,
			NoTaboo:      true,
			TargetLength: true,
		},
	}

	rankedStories := &types.RankedStories{
		Ranked: []types.RankedStory{
			{StoryID: "story1", RelevanceScore: 0.9},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID: "story1",
				Bullets: []types.Bullet{
					{ID: "b1", Skills: []string{"Go", "Python", "AWS", "Docker"}},
				},
			},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{StoryID: "story1", BulletIDs: []string{"b1"}},
		},
	}

	score := ScoreBulletRelevance(bullet, "story1", plan, nil, rankedStories, experienceBank)

	// High relevance story (0.9 * 0.4) + good skill coverage (4/5 * 0.3) +
	// good efficiency (0.75 * 0.2) + perfect style (1.0 * 0.1)
	assert.Greater(t, score, 0.7)
}

func TestScoreBulletRelevance_LowRelevanceStory(t *testing.T) {
	bullet := types.RewrittenBullet{
		OriginalBulletID: "b1",
		LengthChars:      100,
		StyleChecks:      types.StyleChecks{}, // All false
	}

	rankedStories := &types.RankedStories{
		Ranked: []types.RankedStory{
			{StoryID: "story1", RelevanceScore: 0.1},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID: "story1",
				Bullets: []types.Bullet{
					{ID: "b1", Skills: []string{}}, // No skills
				},
			},
		},
	}

	plan := &types.ResumePlan{}

	score := ScoreBulletRelevance(bullet, "story1", plan, nil, rankedStories, experienceBank)

	// Low relevance story should score lower
	assert.Less(t, score, 0.5)
}

func TestScoreBulletRelevance_ManySkills(t *testing.T) {
	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID: "story1",
				Bullets: []types.Bullet{
					{ID: "b1", Skills: []string{"Go", "Python", "AWS", "Docker", "K8s", "Redis"}},
				},
			},
		},
	}

	// Skill coverage should be capped at maxSkillsNorm (5)
	skillCoverage := calculateSkillCoverage("b1", experienceBank)
	assert.Equal(t, 1.0, skillCoverage) // Capped at 1.0
}

func TestScoreBulletRelevance_GoodStyleChecks(t *testing.T) {
	styleChecks := types.StyleChecks{
		StrongVerb:   true,
		Quantified:   true,
		NoTaboo:      true,
		TargetLength: true,
	}

	quality := calculateStyleQuality(styleChecks)
	assert.Equal(t, 1.0, quality)
}

func TestScoreBulletRelevance_PartialStyleChecks(t *testing.T) {
	styleChecks := types.StyleChecks{
		StrongVerb:   true,
		Quantified:   false,
		NoTaboo:      true,
		TargetLength: false,
	}

	quality := calculateStyleQuality(styleChecks)
	assert.Equal(t, 0.5, quality) // 2/4
}

func TestScoreAllBullets_SortsLowestFirst(t *testing.T) {
	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{OriginalBulletID: "high", LengthChars: 50, StyleChecks: types.StyleChecks{StrongVerb: true, Quantified: true, NoTaboo: true, TargetLength: true}},
			{OriginalBulletID: "low", LengthChars: 300, StyleChecks: types.StyleChecks{}},
			{OriginalBulletID: "medium", LengthChars: 150, StyleChecks: types.StyleChecks{StrongVerb: true}},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{StoryID: "story1", BulletIDs: []string{"high", "low", "medium"}},
		},
	}

	rankedStories := &types.RankedStories{
		Ranked: []types.RankedStory{
			{StoryID: "story1", RelevanceScore: 0.5},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID: "story1",
				Bullets: []types.Bullet{
					{ID: "high", Skills: []string{"Go", "Python"}},
					{ID: "low", Skills: []string{}},
					{ID: "medium", Skills: []string{"Go"}},
				},
			},
		},
	}

	scored := ScoreAllBullets(bullets, plan, nil, rankedStories, experienceBank)

	require.Len(t, scored, 3)
	// Lowest scored should be first (candidate for dropping)
	assert.Equal(t, "low", scored[0].BulletID)
	// Highest scored should be last (should keep)
	assert.Equal(t, "high", scored[2].BulletID)
}

func TestScoreAllBullets_MissingStoryInRanked(t *testing.T) {
	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{OriginalBulletID: "b1", LengthChars: 100},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{StoryID: "unknown_story", BulletIDs: []string{"b1"}},
		},
	}

	rankedStories := &types.RankedStories{
		Ranked: []types.RankedStory{}, // Empty - story not in ranked list
	}

	scored := ScoreAllBullets(bullets, plan, nil, rankedStories, nil)

	require.Len(t, scored, 1)
	// Should use default 0.5 story relevance
	assert.Equal(t, "unknown_story", scored[0].StoryID)
}

func TestScoreAllBullets_EmptyBullets(t *testing.T) {
	bullets := &types.RewrittenBullets{Bullets: []types.RewrittenBullet{}}

	scored := ScoreAllBullets(bullets, nil, nil, nil, nil)

	assert.Empty(t, scored)
}

func TestScoreAllBullets_NilBullets(t *testing.T) {
	scored := ScoreAllBullets(nil, nil, nil, nil, nil)

	assert.Empty(t, scored)
}

func TestCalculateLengthEfficiency(t *testing.T) {
	tests := []struct {
		name        string
		lengthChars int
		wantMin     float64
		wantMax     float64
	}{
		{"zero length", 0, 0.4, 0.6}, // Returns 0.5 for zero
		{"very short", 50, 0.8, 1.0}, // Short = high efficiency
		{"average", 150, 0.5, 0.7},   // Average length
		{"at max", 200, 0.4, 0.6},    // At maxLengthChars = 0.5
		{"very long", 400, 0.0, 0.1}, // Long = low efficiency
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			efficiency := calculateLengthEfficiency(tt.lengthChars)
			assert.GreaterOrEqual(t, efficiency, tt.wantMin)
			assert.LessOrEqual(t, efficiency, tt.wantMax)
		})
	}
}

func TestBuildBulletToStoryMap(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{StoryID: "story1", BulletIDs: []string{"b1", "b2"}},
			{StoryID: "story2", BulletIDs: []string{"b3"}},
		},
	}

	bulletMap := buildBulletToStoryMap(plan)

	assert.Equal(t, "story1", bulletMap["b1"])
	assert.Equal(t, "story1", bulletMap["b2"])
	assert.Equal(t, "story2", bulletMap["b3"])
	assert.Empty(t, bulletMap["unknown"])
}

func TestBuildBulletToStoryMap_NilPlan(t *testing.T) {
	bulletMap := buildBulletToStoryMap(nil)

	assert.Empty(t, bulletMap)
}
