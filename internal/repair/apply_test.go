// Package repair provides functionality to automatically fix violations in LaTeX resumes.
package repair

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyRepairs_ShortenBullet(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_001", "bullet_002"},
				Section:        "experience",
				EstimatedLines: 4,
			},
		},
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Long bullet text that needs shortening",
				LengthChars:      100,
				EstimatedLines:   2,
			},
			{
				OriginalBulletID: "bullet_002",
				FinalText:        "Another bullet",
				LengthChars:      50,
				EstimatedLines:   1,
			},
		},
	}

	targetChars := 80
	actions := &types.RepairActions{
		Actions: []types.RepairAction{
			{
				Type:        "shorten_bullet",
				BulletID:    "bullet_001",
				TargetChars: &targetChars,
				Reason:      "Bullet is too long",
			},
		},
	}

	rankedStories := &types.RankedStories{Ranked: []types.RankedStory{}}
	experienceBank := &types.ExperienceBank{Stories: []types.Story{}}

	updatedPlan, updatedBullets, bulletsToRewrite, err := ApplyRepairs(actions, plan, bullets, rankedStories, experienceBank)

	require.NoError(t, err)
	assert.NotNil(t, updatedPlan)
	assert.NotNil(t, updatedBullets)
	assert.Equal(t, 1, len(bulletsToRewrite), "should have one bullet to rewrite")
	assert.Equal(t, "bullet_001", bulletsToRewrite[0], "should mark bullet_001 for rewrite")
	assert.Equal(t, 2, len(updatedBullets.Bullets))
}

func TestApplyRepairs_DropBullet(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_001", "bullet_002"},
				Section:        "experience",
				EstimatedLines: 4,
			},
		},
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "First bullet",
				LengthChars:      50,
				EstimatedLines:   1,
			},
			{
				OriginalBulletID: "bullet_002",
				FinalText:        "Second bullet",
				LengthChars:      60,
				EstimatedLines:   1,
			},
		},
	}

	actions := &types.RepairActions{
		Actions: []types.RepairAction{
			{
				Type:     "drop_bullet",
				BulletID: "bullet_001",
				Reason:   "Remove to save space",
			},
		},
	}

	rankedStories := &types.RankedStories{Ranked: []types.RankedStory{}}
	experienceBank := &types.ExperienceBank{Stories: []types.Story{}}

	updatedPlan, updatedBullets, bulletsToRewrite, err := ApplyRepairs(actions, plan, bullets, rankedStories, experienceBank)

	require.NoError(t, err)
	assert.NotNil(t, updatedPlan)
	assert.NotNil(t, updatedBullets)
	assert.Equal(t, 0, len(bulletsToRewrite), "should not need rewrite after drop_bullet action")
	assert.Equal(t, 1, len(updatedBullets.Bullets))
	assert.Equal(t, "bullet_002", updatedBullets.Bullets[0].OriginalBulletID)
	assert.Equal(t, 1, len(updatedPlan.SelectedStories[0].BulletIDs))
	assert.Equal(t, "bullet_002", updatedPlan.SelectedStories[0].BulletIDs[0])
}

func TestApplyRepairs_DropBullet_RemovesStoryWhenEmpty(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_001"},
				Section:        "experience",
				EstimatedLines: 2,
			},
		},
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Only bullet",
				LengthChars:      50,
				EstimatedLines:   1,
			},
		},
	}

	actions := &types.RepairActions{
		Actions: []types.RepairAction{
			{
				Type:     "drop_bullet",
				BulletID: "bullet_001",
				Reason:   "Remove last bullet",
			},
		},
	}

	rankedStories := &types.RankedStories{Ranked: []types.RankedStory{}}
	experienceBank := &types.ExperienceBank{Stories: []types.Story{}}

	updatedPlan, updatedBullets, _, err := ApplyRepairs(actions, plan, bullets, rankedStories, experienceBank)

	require.NoError(t, err)
	assert.Equal(t, 0, len(updatedPlan.SelectedStories), "story should be removed when it has no bullets")
	assert.Equal(t, 0, len(updatedBullets.Bullets), "all bullets should be removed")
}

func TestApplyRepairs_SwapStory(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_001"},
				Section:        "experience",
				EstimatedLines: 2,
			},
		},
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Old bullet",
				LengthChars:      50,
				EstimatedLines:   1,
			},
		},
	}

	rankedStories := &types.RankedStories{
		Ranked: []types.RankedStory{
			{
				StoryID:        "story_002",
				RelevanceScore: 0.9,
			},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:   "story_002",
				Role: "Senior Engineer",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_002",
						Text:        "New bullet",
						LengthChars: 40,
					},
				},
			},
		},
	}

	actions := &types.RepairActions{
		Actions: []types.RepairAction{
			{
				Type:    "swap_story",
				StoryID: "story_001",
				Reason:  "Swap to better story",
			},
		},
	}

	updatedPlan, updatedBullets, bulletsToRewrite, err := ApplyRepairs(actions, plan, bullets, rankedStories, experienceBank)

	require.NoError(t, err)
	assert.NotNil(t, updatedPlan)
	assert.NotNil(t, updatedBullets)
	assert.Equal(t, 1, len(bulletsToRewrite), "should have one bullet to rewrite after swap_story")
	assert.Equal(t, "bullet_002", bulletsToRewrite[0], "should mark new bullet_002 for rewrite")
	assert.Equal(t, "story_002", updatedPlan.SelectedStories[0].StoryID)
	assert.Equal(t, "bullet_002", updatedPlan.SelectedStories[0].BulletIDs[0])
	// Old bullets should be removed from rewritten bullets (bullet_001 was from story_001)
	// New bullets (bullet_002) will need to be materialized and rewritten in the loop
	assert.Equal(t, 0, len(updatedBullets.Bullets), "old story bullets should be removed")
}

func TestApplyRepairs_DropNonexistentBullet(t *testing.T) {
	// Dropping a nonexistent bullet should not error - it's idempotent
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
		},
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Valid bullet",
				LengthChars:      50,
			},
		},
	}

	actions := &types.RepairActions{
		Actions: []types.RepairAction{
			{
				Type:     "drop_bullet",
				BulletID: "nonexistent_bullet",
				Reason:   "Drop nonexistent bullet (idempotent)",
			},
		},
	}

	rankedStories := &types.RankedStories{Ranked: []types.RankedStory{}}
	experienceBank := &types.ExperienceBank{Stories: []types.Story{}}

	updatedPlan, updatedBullets, _, err := ApplyRepairs(actions, plan, bullets, rankedStories, experienceBank)

	require.NoError(t, err)
	// Plan and bullets should be unchanged
	assert.Equal(t, 1, len(updatedPlan.SelectedStories))
	assert.Equal(t, 1, len(updatedBullets.Bullets))
}

func TestApplyRepairs_MultipleActions(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_001", "bullet_002", "bullet_003"},
				Section:        "experience",
				EstimatedLines: 6,
			},
		},
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "First",
				LengthChars:      50,
				EstimatedLines:   1,
			},
			{
				OriginalBulletID: "bullet_002",
				FinalText:        "Second",
				LengthChars:      60,
				EstimatedLines:   1,
			},
			{
				OriginalBulletID: "bullet_003",
				FinalText:        "Third",
				LengthChars:      70,
				EstimatedLines:   1,
			},
		},
	}

	targetChars := 60
	actions := &types.RepairActions{
		Actions: []types.RepairAction{
			{
				Type:     "drop_bullet",
				BulletID: "bullet_002",
				Reason:   "Remove middle bullet",
			},
			{
				Type:        "shorten_bullet",
				BulletID:    "bullet_003",
				TargetChars: &targetChars,
				Reason:      "Shorten third bullet",
			},
		},
	}

	rankedStories := &types.RankedStories{Ranked: []types.RankedStory{}}
	experienceBank := &types.ExperienceBank{Stories: []types.Story{}}

	updatedPlan, updatedBullets, bulletsToRewrite, err := ApplyRepairs(actions, plan, bullets, rankedStories, experienceBank)

	require.NoError(t, err)
	assert.Equal(t, 2, len(updatedBullets.Bullets))
	assert.Equal(t, 2, len(updatedPlan.SelectedStories[0].BulletIDs))
	assert.Contains(t, updatedPlan.SelectedStories[0].BulletIDs, "bullet_001")
	assert.Contains(t, updatedPlan.SelectedStories[0].BulletIDs, "bullet_003")
	assert.NotContains(t, updatedPlan.SelectedStories[0].BulletIDs, "bullet_002")
	assert.Equal(t, 1, len(bulletsToRewrite), "should have one bullet to rewrite")
	assert.Equal(t, "bullet_003", bulletsToRewrite[0], "should mark bullet_003 for rewrite")
}

func TestApplyRepairs_UnknownActionType(t *testing.T) {
	plan := &types.ResumePlan{SelectedStories: []types.SelectedStory{}}
	bullets := &types.RewrittenBullets{Bullets: []types.RewrittenBullet{}}
	actions := &types.RepairActions{
		Actions: []types.RepairAction{
			{
				Type:   "unknown_action",
				Reason: "Invalid",
			},
		},
	}

	rankedStories := &types.RankedStories{Ranked: []types.RankedStory{}}
	experienceBank := &types.ExperienceBank{Stories: []types.Story{}}

	_, _, _, err := ApplyRepairs(actions, plan, bullets, rankedStories, experienceBank)

	assert.Error(t, err)
	var applyErr *ApplyError
	assert.ErrorAs(t, err, &applyErr)
	assert.Contains(t, err.Error(), "unknown repair action type")
}
