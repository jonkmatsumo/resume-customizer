package selection

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectPlan_EmptyRankedStories(t *testing.T) {
	rankedStories := &types.RankedStories{Ranked: []types.RankedStory{}}
	jobProfile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
	}
	experienceBank := &types.ExperienceBank{Stories: []types.Story{}}
	spaceBudget := &types.SpaceBudget{
		MaxBullets: 8,
		MaxLines:   45,
	}

	plan, err := SelectPlan(rankedStories, jobProfile, experienceBank, spaceBudget)
	require.NoError(t, err)
	assert.Empty(t, plan.SelectedStories)
	assert.Equal(t, 0.0, plan.Coverage.CoverageScore)
}

func TestSelectPlan_BasicSelection(t *testing.T) {
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
				Company:   "Company A",
				Role:      "Engineer",
				StartDate: "2023-01",
				EndDate:   "2024-01",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_001",
						Text:        "Built Go microservices",
						LengthChars: 90,
						Skills:      []string{"Go"},
					},
				},
			},
			{
				ID:        "story_002",
				Company:   "Company B",
				Role:      "Developer",
				StartDate: "2022-01",
				EndDate:   "2023-01",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_002",
						Text:        "Worked with Python",
						LengthChars: 80,
						Skills:      []string{"Python"},
					},
				},
			},
		},
	}

	rankedStories := &types.RankedStories{
		Ranked: []types.RankedStory{
			{
				StoryID:        "story_001",
				RelevanceScore: 0.9,
				MatchedSkills:  []string{"Go"},
			},
			{
				StoryID:        "story_002",
				RelevanceScore: 0.3,
				MatchedSkills:  []string{},
			},
		},
	}

	spaceBudget := &types.SpaceBudget{
		MaxBullets: 5,
		MaxLines:   10,
	}

	plan, err := SelectPlan(rankedStories, jobProfile, experienceBank, spaceBudget)
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Should select at least story_001 (higher relevance, matches Go skill)
	foundStory001 := false
	for _, selected := range plan.SelectedStories {
		if selected.StoryID == "story_001" {
			foundStory001 = true
			assert.Contains(t, selected.BulletIDs, "bullet_001")
			assert.Equal(t, "experience", selected.Section)
			assert.Greater(t, selected.EstimatedLines, 0)
		}
	}
	assert.True(t, foundStory001, "should select story_001")

	// Verify constraints are respected
	totalBullets := 0
	totalLines := 0
	for _, selected := range plan.SelectedStories {
		totalBullets += len(selected.BulletIDs)
		totalLines += selected.EstimatedLines
	}
	assert.LessOrEqual(t, totalBullets, spaceBudget.MaxBullets)
	assert.LessOrEqual(t, totalLines, spaceBudget.MaxLines)

	// Verify coverage metrics
	assert.Greater(t, plan.Coverage.CoverageScore, 0.0)
	assert.NotEmpty(t, plan.Coverage.TopSkillsCovered)
}

func TestSelectPlan_RespectsConstraints(t *testing.T) {
	jobProfile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID: "story_001",
				Bullets: []types.Bullet{
					{ID: "bullet_001", LengthChars: 50, Skills: []string{"Go"}},
					{ID: "bullet_002", LengthChars: 50, Skills: []string{"Go"}},
					{ID: "bullet_003", LengthChars: 50, Skills: []string{"Go"}},
				},
			},
		},
	}

	rankedStories := &types.RankedStories{
		Ranked: []types.RankedStory{
			{StoryID: "story_001", RelevanceScore: 0.8},
		},
	}

	spaceBudget := &types.SpaceBudget{
		MaxBullets: 2, // Can only fit 2 bullets
		MaxLines:   10,
	}

	plan, err := SelectPlan(rankedStories, jobProfile, experienceBank, spaceBudget)
	require.NoError(t, err)

	// Count total bullets selected
	totalBullets := 0
	for _, selected := range plan.SelectedStories {
		totalBullets += len(selected.BulletIDs)
	}
	assert.LessOrEqual(t, totalBullets, spaceBudget.MaxBullets)
}

func TestComputeCoverage(t *testing.T) {
	selectedBullets := []types.Bullet{
		{Skills: []string{"Go", "Kubernetes"}},
		{Skills: []string{"Go", "Python"}},
	}

	skillTargets := &types.SkillTargets{
		Skills: []types.Skill{
			{Name: "Go", Weight: 1.0},
			{Name: "Kubernetes", Weight: 0.7},
			{Name: "Python", Weight: 0.5},
			{Name: "Java", Weight: 0.3},
		},
	}

	coverage := computeCoverage(selectedBullets, skillTargets)

	assert.Greater(t, coverage.CoverageScore, 0.0)
	assert.NotEmpty(t, coverage.TopSkillsCovered)
	// Should include Go, Kubernetes, Python (covered skills)
	assert.Contains(t, coverage.TopSkillsCovered, "Go")
}

func TestComputeCoverage_EmptyBullets(t *testing.T) {
	skillTargets := &types.SkillTargets{
		Skills: []types.Skill{
			{Name: "Go", Weight: 1.0},
		},
	}

	coverage := computeCoverage([]types.Bullet{}, skillTargets)
	assert.Equal(t, 0.0, coverage.CoverageScore)
	assert.Empty(t, coverage.TopSkillsCovered)
}

func TestComputeCoverage_EmptySkillTargets(t *testing.T) {
	selectedBullets := []types.Bullet{
		{Skills: []string{"Go"}},
	}

	coverage := computeCoverage(selectedBullets, nil)
	assert.Equal(t, 0.0, coverage.CoverageScore)
	assert.Empty(t, coverage.TopSkillsCovered)
}
