package selection

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaterializeBullets_Success(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_001", "bullet_002"},
				Section:        "experience",
				EstimatedLines: 2,
			},
			{
				StoryID:        "story_002",
				BulletIDs:      []string{"bullet_003"},
				Section:        "experience",
				EstimatedLines: 1,
			},
		},
		SpaceBudget: types.SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: types.Coverage{
			TopSkillsCovered: []string{"Go", "Python"},
			CoverageScore:    0.85,
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				Company:   "Company A",
				Role:      "Senior Software Engineer",
				StartDate: "2022-01",
				EndDate:   "2024-01",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_001",
						Text:        "Built Go microservices processing 1M+ requests/day",
						Skills:      []string{"Go", "Kubernetes"},
						Metrics:     "1M+ requests/day",
						LengthChars: 60,
					},
					{
						ID:          "bullet_002",
						Text:        "Designed distributed system architecture",
						Skills:      []string{"Go", "distributed systems"},
						LengthChars: 50,
					},
				},
			},
			{
				ID:        "story_002",
				Company:   "Company B",
				Role:      "Software Engineer",
				StartDate: "2020-01",
				EndDate:   "2022-01",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_003",
						Text:        "Developed Python backend services",
						Skills:      []string{"Python", "Django"},
						LengthChars: 45,
					},
				},
			},
		},
	}

	result, err := MaterializeBullets(plan, experienceBank)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Bullets, 3)

	// Verify first bullet
	assert.Equal(t, "bullet_001", result.Bullets[0].ID)
	assert.Equal(t, "story_001", result.Bullets[0].StoryID)
	assert.Equal(t, "Built Go microservices processing 1M+ requests/day", result.Bullets[0].Text)
	assert.Equal(t, []string{"Go", "Kubernetes"}, result.Bullets[0].Skills)
	assert.Equal(t, "1M+ requests/day", result.Bullets[0].Metrics)
	assert.Equal(t, 60, result.Bullets[0].LengthChars)

	// Verify second bullet
	assert.Equal(t, "bullet_002", result.Bullets[1].ID)
	assert.Equal(t, "story_001", result.Bullets[1].StoryID)
	assert.Equal(t, "Designed distributed system architecture", result.Bullets[1].Text)
	assert.Equal(t, []string{"Go", "distributed systems"}, result.Bullets[1].Skills)
	assert.Equal(t, 50, result.Bullets[1].LengthChars)

	// Verify third bullet
	assert.Equal(t, "bullet_003", result.Bullets[2].ID)
	assert.Equal(t, "story_002", result.Bullets[2].StoryID)
	assert.Equal(t, "Developed Python backend services", result.Bullets[2].Text)
	assert.Equal(t, []string{"Python", "Django"}, result.Bullets[2].Skills)
	assert.Equal(t, 45, result.Bullets[2].LengthChars)
}

func TestMaterializeBullets_StoryNotFound(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_nonexistent",
				BulletIDs:      []string{"bullet_001"},
				Section:        "experience",
				EstimatedLines: 1,
			},
		},
		SpaceBudget: types.SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: types.Coverage{
			TopSkillsCovered: []string{},
			CoverageScore:    0.0,
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Company A",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{ID: "bullet_001", Text: "Test", Skills: []string{}, LengthChars: 10},
				},
			},
		},
	}

	result, err := MaterializeBullets(plan, experienceBank)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "story not found")
	assert.Contains(t, err.Error(), "story_nonexistent")
}

func TestMaterializeBullets_BulletNotFound(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_nonexistent"},
				Section:        "experience",
				EstimatedLines: 1,
			},
		},
		SpaceBudget: types.SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: types.Coverage{
			TopSkillsCovered: []string{},
			CoverageScore:    0.0,
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Company A",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{ID: "bullet_001", Text: "Test", Skills: []string{}, LengthChars: 10},
				},
			},
		},
	}

	result, err := MaterializeBullets(plan, experienceBank)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "bullet not found")
	assert.Contains(t, err.Error(), "story_001")
	assert.Contains(t, err.Error(), "bullet_nonexistent")
}

func TestMaterializeBullets_OrderPreserved(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_002",
				BulletIDs:      []string{"bullet_003"},
				Section:        "experience",
				EstimatedLines: 1,
			},
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_001", "bullet_002"},
				Section:        "experience",
				EstimatedLines: 2,
			},
		},
		SpaceBudget: types.SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: types.Coverage{
			TopSkillsCovered: []string{},
			CoverageScore:    0.0,
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Company A",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{ID: "bullet_001", Text: "First", Skills: []string{}, LengthChars: 10},
					{ID: "bullet_002", Text: "Second", Skills: []string{}, LengthChars: 10},
				},
			},
			{
				ID:      "story_002",
				Company: "Company B",
				Role:    "Developer",
				Bullets: []types.Bullet{
					{ID: "bullet_003", Text: "Third", Skills: []string{}, LengthChars: 10},
				},
			},
		},
	}

	result, err := MaterializeBullets(plan, experienceBank)
	require.NoError(t, err)
	require.Len(t, result.Bullets, 3)

	// Verify order: story_002 (bullet_003) comes first, then story_001 (bullet_001, bullet_002)
	assert.Equal(t, "bullet_003", result.Bullets[0].ID)
	assert.Equal(t, "bullet_001", result.Bullets[1].ID)
	assert.Equal(t, "bullet_002", result.Bullets[2].ID)
}

func TestMaterializeBullets_EmptyPlan(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{},
		SpaceBudget: types.SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: types.Coverage{
			TopSkillsCovered: []string{},
			CoverageScore:    0.0,
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Company A",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{ID: "bullet_001", Text: "Test", Skills: []string{}, LengthChars: 10},
				},
			},
		},
	}

	result, err := MaterializeBullets(plan, experienceBank)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Bullets)
}

func TestMaterializeBullets_EmptyBulletIDs(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{},
				Section:        "experience",
				EstimatedLines: 0,
			},
		},
		SpaceBudget: types.SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: types.Coverage{
			TopSkillsCovered: []string{},
			CoverageScore:    0.0,
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Company A",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{ID: "bullet_001", Text: "Test", Skills: []string{}, LengthChars: 10},
				},
			},
		},
	}

	result, err := MaterializeBullets(plan, experienceBank)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Bullets)
}

func TestMaterializeBullets_AllFields(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_001"},
				Section:        "experience",
				EstimatedLines: 1,
			},
		},
		SpaceBudget: types.SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: types.Coverage{
			TopSkillsCovered: []string{},
			CoverageScore:    0.0,
		},
	}

	originalSkills := []string{"Go", "Kubernetes"}
	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Company A",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_001",
						Text:        "Built system",
						Skills:      originalSkills,
						Metrics:     "1M+ requests",
						LengthChars: 50,
					},
				},
			},
		},
	}

	result, err := MaterializeBullets(plan, experienceBank)
	require.NoError(t, err)
	require.Len(t, result.Bullets, 1)

	bullet := result.Bullets[0]
	assert.Equal(t, "bullet_001", bullet.ID)
	assert.Equal(t, "story_001", bullet.StoryID)
	assert.Equal(t, "Built system", bullet.Text)
	assert.Equal(t, []string{"Go", "Kubernetes"}, bullet.Skills)
	assert.Equal(t, "1M+ requests", bullet.Metrics)
	assert.Equal(t, 50, bullet.LengthChars)

	// Verify skills slice is copied (not shared reference)
	originalSkills[0] = "Modified"
	assert.Equal(t, []string{"Go", "Kubernetes"}, bullet.Skills) // Should not be modified
}

func TestMaterializeBullets_EmptyMetrics(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_001"},
				Section:        "experience",
				EstimatedLines: 1,
			},
		},
		SpaceBudget: types.SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: types.Coverage{
			TopSkillsCovered: []string{},
			CoverageScore:    0.0,
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Company A",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_001",
						Text:        "Built system",
						Skills:      []string{"Go"},
						Metrics:     "",
						LengthChars: 30,
					},
				},
			},
		},
	}

	result, err := MaterializeBullets(plan, experienceBank)
	require.NoError(t, err)
	require.Len(t, result.Bullets, 1)
	assert.Equal(t, "", result.Bullets[0].Metrics)
}

