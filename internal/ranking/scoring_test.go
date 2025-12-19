package ranking

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestComputeSkillOverlapScore_ExactMatch(t *testing.T) {
	story := &types.Story{
		ID: "story_001",
		Bullets: []types.Bullet{
			{Skills: []string{"Go", "Kubernetes"}},
			{Skills: []string{"Python"}},
		},
	}

	skillTargets := &types.SkillTargets{
		Skills: []types.Skill{
			{Name: "Go", Weight: 1.0},
			{Name: "Kubernetes", Weight: 0.5},
			{Name: "Docker", Weight: 0.3},
		},
	}

	score, matched := computeSkillOverlapScore(story, skillTargets)

	// Should match Go (1.0) and Kubernetes (0.5), total 1.5 out of 1.8 possible
	expectedScore := 1.5 / 1.8
	assert.InDelta(t, expectedScore, score, 0.01)
	assert.ElementsMatch(t, []string{"Go", "Kubernetes"}, matched)
}

func TestComputeSkillOverlapScore_NormalizedMatch(t *testing.T) {
	story := &types.Story{
		ID: "story_002",
		Bullets: []types.Bullet{
			{Skills: []string{"golang", "JS"}}, // Should normalize to Go, JavaScript
		},
	}

	skillTargets := &types.SkillTargets{
		Skills: []types.Skill{
			{Name: "Go", Weight: 1.0},
			{Name: "JavaScript", Weight: 0.5},
		},
	}

	score, matched := computeSkillOverlapScore(story, skillTargets)

	// Should match both after normalization (score should be 1.0 since both match)
	assert.InDelta(t, 1.0, score, 0.01)
	assert.ElementsMatch(t, []string{"Go", "JavaScript"}, matched)
}

func TestComputeSkillOverlapScore_NoMatches(t *testing.T) {
	story := &types.Story{
		ID: "story_003",
		Bullets: []types.Bullet{
			{Skills: []string{"PHP", "Ruby"}},
		},
	}

	skillTargets := &types.SkillTargets{
		Skills: []types.Skill{
			{Name: "Go", Weight: 1.0},
		},
	}

	score, matched := computeSkillOverlapScore(story, skillTargets)

	assert.Equal(t, 0.0, score)
	assert.Empty(t, matched)
}

func TestComputeSkillOverlapScore_EmptyTargets(t *testing.T) {
	story := &types.Story{
		ID: "story_004",
		Bullets: []types.Bullet{
			{Skills: []string{"Go"}},
		},
	}

	skillTargets := &types.SkillTargets{
		Skills: []types.Skill{},
	}

	score, matched := computeSkillOverlapScore(story, skillTargets)

	assert.Equal(t, 0.0, score)
	assert.Empty(t, matched)
}

func TestComputeSkillOverlapScore_EmptyStorySkills(t *testing.T) {
	story := &types.Story{
		ID:      "story_005",
		Bullets: []types.Bullet{{Skills: []string{}}},
	}

	skillTargets := &types.SkillTargets{
		Skills: []types.Skill{
			{Name: "Go", Weight: 1.0},
		},
	}

	score, matched := computeSkillOverlapScore(story, skillTargets)

	assert.Equal(t, 0.0, score)
	assert.Empty(t, matched)
}

func TestComputeKeywordOverlapScore_Matches(t *testing.T) {
	story := &types.Story{
		ID: "story_006",
		Bullets: []types.Bullet{
			{Text: "Built distributed systems using microservices"},
			{Text: "Worked with cloud infrastructure"},
		},
	}

	jobProfile := &types.JobProfile{
		Keywords: []string{"microservices", "distributed systems", "cloud"},
	}

	score := computeKeywordOverlapScore(story, jobProfile)

	// Should match all 3 keywords
	assert.InDelta(t, 1.0, score, 0.01)
}

func TestComputeKeywordOverlapScore_PartialMatches(t *testing.T) {
	story := &types.Story{
		ID: "story_007",
		Bullets: []types.Bullet{
			{Text: "Worked with microservices"},
		},
	}

	jobProfile := &types.JobProfile{
		Keywords: []string{"microservices", "distributed systems", "cloud"},
	}

	score := computeKeywordOverlapScore(story, jobProfile)

	// Should match 1 out of 3 keywords
	assert.InDelta(t, 1.0/3.0, score, 0.01)
}

func TestComputeKeywordOverlapScore_NoMatches(t *testing.T) {
	story := &types.Story{
		ID: "story_008",
		Bullets: []types.Bullet{
			{Text: "Worked with PHP"},
		},
	}

	jobProfile := &types.JobProfile{
		Keywords: []string{"microservices", "distributed systems"},
	}

	score := computeKeywordOverlapScore(story, jobProfile)

	assert.Equal(t, 0.0, score)
}

func TestComputeKeywordOverlapScore_EmptyKeywords(t *testing.T) {
	story := &types.Story{
		ID: "story_009",
		Bullets: []types.Bullet{
			{Text: "Some text"},
		},
	}

	jobProfile := &types.JobProfile{
		Keywords: []string{},
	}

	score := computeKeywordOverlapScore(story, jobProfile)

	assert.Equal(t, 0.0, score)
}

func TestComputeKeywordOverlapScore_CaseInsensitive(t *testing.T) {
	story := &types.Story{
		ID: "story_010",
		Bullets: []types.Bullet{
			{Text: "Worked with MICROSERVICES"},
		},
	}

	jobProfile := &types.JobProfile{
		Keywords: []string{"microservices"},
	}

	score := computeKeywordOverlapScore(story, jobProfile)

	assert.InDelta(t, 1.0, score, 0.01)
}

func TestComputeEvidenceStrengthScore_AllHigh(t *testing.T) {
	story := &types.Story{
		ID: "story_011",
		Bullets: []types.Bullet{
			{EvidenceStrength: "high"},
			{EvidenceStrength: "HIGH"},
			{EvidenceStrength: "high"},
		},
	}

	score := computeEvidenceStrengthScore(story)

	assert.InDelta(t, 1.0, score, 0.01)
}

func TestComputeEvidenceStrengthScore_Mixed(t *testing.T) {
	story := &types.Story{
		ID: "story_012",
		Bullets: []types.Bullet{
			{EvidenceStrength: "high"},   // 1.0
			{EvidenceStrength: "medium"}, // 0.6
			{EvidenceStrength: "low"},    // 0.3
		},
	}

	score := computeEvidenceStrengthScore(story)

	expectedScore := (1.0 + 0.6 + 0.3) / 3.0
	assert.InDelta(t, expectedScore, score, 0.01)
}

func TestComputeEvidenceStrengthScore_AllLow(t *testing.T) {
	story := &types.Story{
		ID: "story_013",
		Bullets: []types.Bullet{
			{EvidenceStrength: "low"},
			{EvidenceStrength: "low"},
		},
	}

	score := computeEvidenceStrengthScore(story)

	assert.InDelta(t, 0.3, score, 0.01)
}

func TestComputeEvidenceStrengthScore_EmptyBullets(t *testing.T) {
	story := &types.Story{
		ID:      "story_014",
		Bullets: []types.Bullet{},
	}

	score := computeEvidenceStrengthScore(story)

	assert.Equal(t, 0.0, score)
}

func TestComputeEvidenceStrengthScore_UnknownStrength(t *testing.T) {
	story := &types.Story{
		ID: "story_015",
		Bullets: []types.Bullet{
			{EvidenceStrength: "unknown"}, // Should default to 0.6 (medium)
		},
	}

	score := computeEvidenceStrengthScore(story)

	assert.InDelta(t, 0.6, score, 0.01)
}

func TestComputeRecencyScore_Recent(t *testing.T) {
	story := &types.Story{
		ID:        "story_016",
		StartDate: "2023-01", // Recent date
	}

	score := computeRecencyScore(story)

	// Should be reasonable score for recent dates (2023-01 is ~2.9 years ago, so score should be ~0.70)
	// Use a reasonable range check instead of exact value
	assert.Greater(t, score, 0.6) // Should be above 0.6 for dates less than 4 years old
	assert.LessOrEqual(t, score, 1.0)
}

func TestComputeRecencyScore_Old(t *testing.T) {
	story := &types.Story{
		ID:        "story_017",
		StartDate: "2010-01", // Old date (more than 10 years)
	}

	score := computeRecencyScore(story)

	// Should be low (close to 0.0) for old dates
	assert.Less(t, score, 0.2)
	assert.GreaterOrEqual(t, score, 0.0)
}

func TestComputeRecencyScore_EmptyDate(t *testing.T) {
	story := &types.Story{
		ID:        "story_018",
		StartDate: "",
	}

	score := computeRecencyScore(story)

	assert.Equal(t, 0.5, score) // Neutral score
}

func TestComputeRecencyScore_InvalidDate(t *testing.T) {
	story := &types.Story{
		ID:        "story_019",
		StartDate: "invalid",
	}

	score := computeRecencyScore(story)

	assert.Equal(t, 0.5, score) // Neutral score for invalid dates
}

func TestComputeRecencyScore_InvalidFormat(t *testing.T) {
	story := &types.Story{
		ID:        "story_020",
		StartDate: "2023",
	}

	score := computeRecencyScore(story)

	assert.Equal(t, 0.5, score) // Neutral score for invalid format
}
