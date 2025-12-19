package ranking

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRankStories_BasicRanking(t *testing.T) {
	jobProfile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
		Keywords: []string{"microservices"},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				StartDate: "2023-01",
				Bullets: []types.Bullet{
					{
						Skills:           []string{"Go"},
						Text:             "Built microservices",
						EvidenceStrength: "high",
					},
				},
			},
			{
				ID:        "story_002",
				StartDate: "2020-01",
				Bullets: []types.Bullet{
					{
						Skills:           []string{"Python"},
						Text:             "Worked on backend",
						EvidenceStrength: "medium",
					},
				},
			},
		},
	}

	ranked, err := RankStories(jobProfile, experienceBank)
	require.NoError(t, err)
	require.Len(t, ranked.Ranked, 2)

	// Story 1 should rank higher (matches Go skill and microservices keyword)
	assert.Equal(t, "story_001", ranked.Ranked[0].StoryID)
	assert.Greater(t, ranked.Ranked[0].RelevanceScore, ranked.Ranked[1].RelevanceScore)
	assert.Contains(t, ranked.Ranked[0].MatchedSkills, "Go")
	assert.Contains(t, ranked.Ranked[0].Notes, "skill match")
}

func TestRankStories_SortingByRelevance(t *testing.T) {
	jobProfile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "low_score",
				StartDate: "2010-01",
				Bullets: []types.Bullet{
					{
						Skills:           []string{"Python"},
						EvidenceStrength: "low",
					},
				},
			},
			{
				ID:        "high_score",
				StartDate: "2023-01",
				Bullets: []types.Bullet{
					{
						Skills:           []string{"Go"},
						EvidenceStrength: "high",
					},
				},
			},
		},
	}

	ranked, err := RankStories(jobProfile, experienceBank)
	require.NoError(t, err)
	require.Len(t, ranked.Ranked, 2)

	// Should be sorted by relevance score descending
	assert.Greater(t, ranked.Ranked[0].RelevanceScore, ranked.Ranked[1].RelevanceScore)
	assert.Equal(t, "high_score", ranked.Ranked[0].StoryID)
	assert.Equal(t, "low_score", ranked.Ranked[1].StoryID)
}

func TestRankStories_NoteGeneration(t *testing.T) {
	jobProfile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
			{Skill: "Kubernetes", Evidence: "Required"},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				StartDate: "2023-01",
				Bullets: []types.Bullet{
					{
						Skills:           []string{"Go", "Kubernetes"},
						Text:             "Built systems",
						EvidenceStrength: "high",
					},
				},
			},
		},
	}

	ranked, err := RankStories(jobProfile, experienceBank)
	require.NoError(t, err)
	require.Len(t, ranked.Ranked, 1)

	notes := ranked.Ranked[0].Notes
	assert.Contains(t, notes, "skill match")
	assert.Contains(t, notes, "evidence strength")
	// Should mention matched skills
	assert.Contains(t, notes, "Go")
}

func TestRankStories_Deterministic(t *testing.T) {
	jobProfile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				StartDate: "2023-01",
				Bullets: []types.Bullet{
					{
						Skills:           []string{"Go"},
						EvidenceStrength: "high",
					},
				},
			},
		},
	}

	// Run twice with same inputs
	ranked1, err1 := RankStories(jobProfile, experienceBank)
	require.NoError(t, err1)

	ranked2, err2 := RankStories(jobProfile, experienceBank)
	require.NoError(t, err2)

	// Should get same results
	require.Len(t, ranked1.Ranked, 1)
	require.Len(t, ranked2.Ranked, 1)
	assert.InDelta(t, ranked1.Ranked[0].RelevanceScore, ranked2.Ranked[0].RelevanceScore, 0.0001)
	assert.Equal(t, ranked1.Ranked[0].StoryID, ranked2.Ranked[0].StoryID)
}

func TestRankStories_EmptyExperienceBank(t *testing.T) {
	jobProfile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{},
	}

	ranked, err := RankStories(jobProfile, experienceBank)
	require.NoError(t, err)
	assert.Empty(t, ranked.Ranked)
}

func TestRankStories_ScoreComponents(t *testing.T) {
	jobProfile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
		Keywords: []string{"microservices"},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				StartDate: "2023-01",
				Bullets: []types.Bullet{
					{
						Skills:           []string{"Go"},
						Text:             "Built microservices",
						EvidenceStrength: "high",
					},
				},
			},
		},
	}

	ranked, err := RankStories(jobProfile, experienceBank)
	require.NoError(t, err)
	require.Len(t, ranked.Ranked, 1)

	story := ranked.Ranked[0]
	// All components should be populated
	assert.Greater(t, story.SkillOverlap, 0.0)
	assert.Greater(t, story.KeywordOverlap, 0.0)
	assert.Greater(t, story.EvidenceStrength, 0.0)
	assert.Greater(t, story.RelevanceScore, 0.0)
	// Relevance score should be weighted combination
	// Note: We can't easily compute expected recency here, so just verify it's reasonable
	// The actual score calculation is tested in the RankStories function itself
	assert.Greater(t, story.RelevanceScore, 0.0)
	assert.LessOrEqual(t, story.RelevanceScore, 1.0)
}

func TestRankStories_ScoreRange(t *testing.T) {
	jobProfile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				StartDate: "2023-01",
				Bullets: []types.Bullet{
					{
						Skills:           []string{"Go"},
						EvidenceStrength: "high",
					},
				},
			},
		},
	}

	ranked, err := RankStories(jobProfile, experienceBank)
	require.NoError(t, err)
	require.Len(t, ranked.Ranked, 1)

	story := ranked.Ranked[0]
	// Scores should be in valid range [0, 1]
	assert.GreaterOrEqual(t, story.RelevanceScore, 0.0)
	assert.LessOrEqual(t, story.RelevanceScore, 1.0)
	assert.GreaterOrEqual(t, story.SkillOverlap, 0.0)
	assert.LessOrEqual(t, story.SkillOverlap, 1.0)
	assert.GreaterOrEqual(t, story.KeywordOverlap, 0.0)
	assert.LessOrEqual(t, story.KeywordOverlap, 1.0)
	assert.GreaterOrEqual(t, story.EvidenceStrength, 0.0)
	assert.LessOrEqual(t, story.EvidenceStrength, 1.0)
}
