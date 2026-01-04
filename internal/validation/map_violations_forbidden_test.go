// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapViolationsToBullets_ForbiddenPhraseWithMap(t *testing.T) {
	lineNum := 15

	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:       "forbidden_phrase",
				Severity:   "error",
				Details:    "Contains forbidden phrase: ninja",
				LineNumber: &lineNum,
			},
		},
	}

	lineToBulletMap := map[int]string{
		15: "bullet_002",
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_002",
				FinalText:        "I am a coding ninja",
			},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_002",
				BulletIDs: []string{"bullet_002"},
			},
		},
	}

	forbiddenPhraseMap := map[string][]string{
		"bullet_002": []string{"ninja"},
	}

	mapped := MapViolationsToBullets(violations, lineToBulletMap, bullets, plan, forbiddenPhraseMap)
	require.NotNil(t, mapped)
	require.Equal(t, 1, len(mapped.Violations))

	violation := mapped.Violations[0]
	assert.NotNil(t, violation.BulletID)
	assert.Equal(t, "bullet_002", *violation.BulletID)
	assert.NotNil(t, violation.StoryID)
	assert.Equal(t, "story_002", *violation.StoryID)
	assert.NotNil(t, violation.BulletText)
	assert.Contains(t, *violation.BulletText, "coding ninja")
}

func TestMapViolationsToBullets_ForbiddenPhraseWithoutMap(t *testing.T) {
	lineNum := 15

	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:       "forbidden_phrase",
				Severity:   "error",
				Details:    "Contains forbidden phrase: ninja",
				LineNumber: &lineNum,
			},
		},
	}

	lineToBulletMap := map[int]string{
		15: "bullet_002",
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_002",
				FinalText:        "I am a coding ninja",
			},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_002",
				BulletIDs: []string{"bullet_002"},
			},
		},
	}

	// No forbidden phrase map - should still map via line-to-bullet
	mapped := MapViolationsToBullets(violations, lineToBulletMap, bullets, plan, nil)
	require.NotNil(t, mapped)
	require.Equal(t, 1, len(mapped.Violations))

	violation := mapped.Violations[0]
	// Should still map via line-to-bullet mapping
	assert.NotNil(t, violation.BulletID)
	assert.Equal(t, "bullet_002", *violation.BulletID)
}

func TestMapViolationsToBullets_ForbiddenPhraseMapButNotInBullet(t *testing.T) {
	lineNum := 15

	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:       "forbidden_phrase",
				Severity:   "error",
				Details:    "Contains forbidden phrase: ninja",
				LineNumber: &lineNum,
			},
		},
	}

	lineToBulletMap := map[int]string{
		15: "bullet_002",
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_002",
				FinalText:        "I am a software engineer", // No forbidden phrase
			},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_002",
				BulletIDs: []string{"bullet_002"},
			},
		},
	}

	// Forbidden phrase map doesn't include this bullet
	forbiddenPhraseMap := map[string][]string{
		"bullet_001": []string{"ninja"}, // Different bullet
	}

	mapped := MapViolationsToBullets(violations, lineToBulletMap, bullets, plan, forbiddenPhraseMap)
	require.NotNil(t, mapped)
	require.Equal(t, 1, len(mapped.Violations))

	violation := mapped.Violations[0]
	// Should not map since bullet not in forbidden phrase map (for forbidden_phrase violations)
	assert.Nil(t, violation.BulletID)
	assert.Nil(t, violation.StoryID)
}
