// Package repair provides functionality to automatically fix violations in LaTeX resumes.
package repair

import (
	"fmt"
	"testing"

	"github.com/jonathan/resume-customizer/internal/selection"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/jonathan/resume-customizer/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposeBulletDrops_NoOverflow(t *testing.T) {
	overflow := &validation.OverflowAnalysis{
		ExcessPages:   0,
		ExcessBullets: 0,
		MustDrop:      false,
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{OriginalBulletID: "b1"},
		},
	}

	actions := ProposeBulletDrops(overflow, bullets, nil, nil, nil, nil)

	assert.Empty(t, actions)
}

func TestProposeBulletDrops_NilOverflow(t *testing.T) {
	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{OriginalBulletID: "b1"},
		},
	}

	actions := ProposeBulletDrops(nil, bullets, nil, nil, nil, nil)

	assert.Empty(t, actions)
}

func TestProposeBulletDrops_DropsLowestScored(t *testing.T) {
	overflow := &validation.OverflowAnalysis{
		ExcessPages:   1.0,
		ExcessLines:   50,
		ExcessBullets: 2.0, // Need to drop 2 bullets
		MustDrop:      true,
	}

	// Create bullets with different quality levels
	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "high_quality",
				LengthChars:      50,
				StyleChecks: types.StyleChecks{
					StrongVerb:   true,
					Quantified:   true,
					NoTaboo:      true,
					TargetLength: true,
				},
			},
			{
				OriginalBulletID: "medium_quality",
				LengthChars:      150,
				StyleChecks: types.StyleChecks{
					StrongVerb: true,
				},
			},
			{
				OriginalBulletID: "low_quality",
				LengthChars:      300, // Very long = low efficiency
				StyleChecks:      types.StyleChecks{},
			},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{StoryID: "story1", BulletIDs: []string{"high_quality", "medium_quality", "low_quality"}},
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
					{ID: "high_quality", Skills: []string{"Go", "Python", "AWS"}},
					{ID: "medium_quality", Skills: []string{"Go"}},
					{ID: "low_quality", Skills: []string{}},
				},
			},
		},
	}

	actions := ProposeBulletDrops(overflow, bullets, plan, nil, rankedStories, experienceBank)

	require.Len(t, actions, 2)
	// First drop should be lowest scored (low_quality)
	assert.Equal(t, "drop_bullet", actions[0].Type)
	assert.Equal(t, "low_quality", actions[0].BulletID)
	// Second drop should be medium_quality
	assert.Equal(t, "medium_quality", actions[1].BulletID)
	// High quality should NOT be dropped
	for _, action := range actions {
		assert.NotEqual(t, "high_quality", action.BulletID)
	}
}

func TestProposeBulletDrops_CorrectNumberOfDrops(t *testing.T) {
	tests := []struct {
		name          string
		excessBullets float64
		numBullets    int
		wantDrops     int
	}{
		{"drop 1", 1.0, 5, 1},
		{"drop 2 (ceil)", 1.5, 5, 2},
		{"drop 3", 3.0, 5, 3},
		{"limited by bullets", 10.0, 3, 3}, // Can't drop more than we have
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overflow := &validation.OverflowAnalysis{
				ExcessPages:   1.0,
				ExcessBullets: tt.excessBullets,
				MustDrop:      true,
			}

			// Create enough bullets
			bullets := &types.RewrittenBullets{
				Bullets: make([]types.RewrittenBullet, tt.numBullets),
			}
			for i := 0; i < tt.numBullets; i++ {
				bullets.Bullets[i] = types.RewrittenBullet{
					OriginalBulletID: fmt.Sprintf("b%d", i),
					LengthChars:      100,
				}
			}

			actions := ProposeBulletDrops(overflow, bullets, nil, nil, nil, nil)

			assert.Len(t, actions, tt.wantDrops)
		})
	}
}

func TestProposeBulletDrops_GeneratesValidActions(t *testing.T) {
	overflow := &validation.OverflowAnalysis{
		ExcessPages:   1.0,
		ExcessBullets: 1.0,
		MustDrop:      true,
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{OriginalBulletID: "b1", LengthChars: 100},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{StoryID: "story1", BulletIDs: []string{"b1"}},
		},
	}

	actions := ProposeBulletDrops(overflow, bullets, plan, nil, nil, nil)

	require.Len(t, actions, 1)
	assert.Equal(t, "drop_bullet", actions[0].Type)
	assert.Equal(t, "b1", actions[0].BulletID)
	assert.Equal(t, "story1", actions[0].StoryID)
	assert.NotEmpty(t, actions[0].Reason)
	assert.Contains(t, actions[0].Reason, "page overflow")
}

func TestProposeBulletDrops_EmptyBullets(t *testing.T) {
	overflow := &validation.OverflowAnalysis{
		ExcessPages:   1.0,
		ExcessBullets: 2.0,
		MustDrop:      true,
	}

	bullets := &types.RewrittenBullets{Bullets: []types.RewrittenBullet{}}

	actions := ProposeBulletDrops(overflow, bullets, nil, nil, nil, nil)

	assert.Empty(t, actions)
}

func TestProposeBulletDrops_NilBullets(t *testing.T) {
	overflow := &validation.OverflowAnalysis{
		ExcessPages:   1.0,
		ExcessBullets: 2.0,
		MustDrop:      true,
	}

	actions := ProposeBulletDrops(overflow, nil, nil, nil, nil, nil)

	assert.Empty(t, actions)
}

func TestFormatDropReason(t *testing.T) {
	bullet := selection.ScoredBullet{
		BulletID:       "b1",
		StoryID:        "story1",
		RelevanceScore: 0.45,
		Components: selection.ScoreComponents{
			StoryRelevance:   0.5,
			SkillCoverage:    0.3,
			LengthEfficiency: 0.6,
			StyleQuality:     0.25,
		},
	}

	overflow := &validation.OverflowAnalysis{
		ExcessPages: 1.5,
	}

	reason := formatDropReason(bullet, overflow)

	assert.Contains(t, reason, "1.5 excess pages")
	assert.Contains(t, reason, "0.45")
	assert.Contains(t, reason, "story: 0.50")
}
