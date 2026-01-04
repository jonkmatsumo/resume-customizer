// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapViolationsToBullets_LineTooLong(t *testing.T) {
	lineNum := 42
	charCount := 100

	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:       "line_too_long",
				Severity:   "error",
				Details:    "Line exceeds maximum character count",
				LineNumber: &lineNum,
				CharCount:  &charCount,
			},
		},
	}

	lineToBulletMap := map[int]string{
		42: "bullet_001",
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "This is a very long bullet point that exceeds the maximum character count",
			},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
		},
	}

	mapped := MapViolationsToBullets(violations, lineToBulletMap, bullets, plan, nil)
	require.NotNil(t, mapped)
	require.Equal(t, 1, len(mapped.Violations))

	violation := mapped.Violations[0]
	assert.NotNil(t, violation.BulletID)
	assert.Equal(t, "bullet_001", *violation.BulletID)
	assert.NotNil(t, violation.StoryID)
	assert.Equal(t, "story_001", *violation.StoryID)
	assert.NotNil(t, violation.BulletText)
	assert.Contains(t, *violation.BulletText, "very long bullet point")
}

func TestMapViolationsToBullets_ForbiddenPhrase(t *testing.T) {
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

	mapped := MapViolationsToBullets(violations, lineToBulletMap, bullets, plan, nil)
	require.NotNil(t, mapped)
	require.Equal(t, 1, len(mapped.Violations))

	violation := mapped.Violations[0]
	assert.NotNil(t, violation.BulletID)
	assert.Equal(t, "bullet_002", *violation.BulletID)
	assert.NotNil(t, violation.StoryID)
	assert.Equal(t, "story_002", *violation.StoryID)
}

func TestMapViolationsToBullets_NoLineNumber(t *testing.T) {
	// Page overflow violations don't have line numbers
	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:     "page_overflow",
				Severity: "error",
				Details:  "Resume has 2 pages, maximum allowed is 1",
				// No LineNumber
			},
		},
	}

	lineToBulletMap := map[int]string{
		42: "bullet_001",
	}

	mapped := MapViolationsToBullets(violations, lineToBulletMap, nil, nil, nil)
	require.NotNil(t, mapped)
	require.Equal(t, 1, len(mapped.Violations))

	violation := mapped.Violations[0]
	// Should remain unmapped (no bullet_id)
	assert.Nil(t, violation.BulletID)
	assert.Nil(t, violation.StoryID)
}

func TestMapViolationsToBullets_UnmappedLineNumber(t *testing.T) {
	lineNum := 99 // Line number that doesn't map to any bullet

	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:       "line_too_long",
				Severity:   "error",
				Details:    "Line exceeds maximum character count",
				LineNumber: &lineNum,
			},
		},
	}

	lineToBulletMap := map[int]string{
		42: "bullet_001", // Different line number
	}

	mapped := MapViolationsToBullets(violations, lineToBulletMap, nil, nil, nil)
	require.NotNil(t, mapped)
	require.Equal(t, 1, len(mapped.Violations))

	violation := mapped.Violations[0]
	// Should remain unmapped (line 99 not in map)
	assert.Nil(t, violation.BulletID)
	assert.Nil(t, violation.StoryID)
}

func TestMapViolationsToBullets_MultipleViolations(t *testing.T) {
	lineNum1 := 42
	lineNum2 := 50

	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:       "line_too_long",
				Severity:   "error",
				Details:    "Line 42 too long",
				LineNumber: &lineNum1,
			},
			{
				Type:       "forbidden_phrase",
				Severity:   "error",
				Details:    "Line 50 has forbidden phrase",
				LineNumber: &lineNum2,
			},
		},
	}

	lineToBulletMap := map[int]string{
		42: "bullet_001",
		50: "bullet_002",
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "First bullet",
			},
			{
				OriginalBulletID: "bullet_002",
				FinalText:        "Second bullet",
			},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
			{
				StoryID:   "story_002",
				BulletIDs: []string{"bullet_002"},
			},
		},
	}

	mapped := MapViolationsToBullets(violations, lineToBulletMap, bullets, plan, nil)
	require.NotNil(t, mapped)
	require.Equal(t, 2, len(mapped.Violations))

	// First violation
	v1 := mapped.Violations[0]
	assert.NotNil(t, v1.BulletID)
	assert.Equal(t, "bullet_001", *v1.BulletID)
	assert.Equal(t, "story_001", *v1.StoryID)

	// Second violation
	v2 := mapped.Violations[1]
	assert.NotNil(t, v2.BulletID)
	assert.Equal(t, "bullet_002", *v2.BulletID)
	assert.Equal(t, "story_002", *v2.StoryID)
}

func TestMapViolationsToBullets_MultipleViolationsSameBullet(t *testing.T) {
	lineNum1 := 42
	lineNum2 := 43

	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:       "line_too_long",
				Severity:   "error",
				Details:    "Line 42 too long",
				LineNumber: &lineNum1,
			},
			{
				Type:       "forbidden_phrase",
				Severity:   "error",
				Details:    "Line 43 has forbidden phrase",
				LineNumber: &lineNum2,
			},
		},
	}

	// Both lines map to the same bullet (bullet spans multiple lines)
	lineToBulletMap := map[int]string{
		42: "bullet_001",
		43: "bullet_001",
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "This is a problematic bullet with multiple issues",
			},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
		},
	}

	mapped := MapViolationsToBullets(violations, lineToBulletMap, bullets, plan, nil)
	require.NotNil(t, mapped)
	require.Equal(t, 2, len(mapped.Violations))

	// Both violations should map to the same bullet
	v1 := mapped.Violations[0]
	v2 := mapped.Violations[1]
	assert.Equal(t, "bullet_001", *v1.BulletID)
	assert.Equal(t, "bullet_001", *v2.BulletID)
	assert.Equal(t, "story_001", *v1.StoryID)
	assert.Equal(t, "story_001", *v2.StoryID)
}

func TestMapViolationsToBullets_NilMapping(t *testing.T) {
	lineNum := 42

	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:       "line_too_long",
				Severity:   "error",
				Details:    "Line exceeds maximum character count",
				LineNumber: &lineNum,
			},
		},
	}

	// No mapping provided
	mapped := MapViolationsToBullets(violations, nil, nil, nil, nil)
	require.NotNil(t, mapped)
	require.Equal(t, 1, len(mapped.Violations))

	// Should return violations as-is (unmapped)
	violation := mapped.Violations[0]
	assert.Nil(t, violation.BulletID)
	assert.Nil(t, violation.StoryID)
}

func TestMapViolationsToBullets_EmptyMapping(t *testing.T) {
	lineNum := 42

	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:       "line_too_long",
				Severity:   "error",
				Details:    "Line exceeds maximum character count",
				LineNumber: &lineNum,
			},
		},
	}

	// Empty mapping
	lineToBulletMap := map[int]string{}

	mapped := MapViolationsToBullets(violations, lineToBulletMap, nil, nil, nil)
	require.NotNil(t, mapped)
	require.Equal(t, 1, len(mapped.Violations))

	// Should return violations as-is (unmapped)
	violation := mapped.Violations[0]
	assert.Nil(t, violation.BulletID)
}

func TestMapViolationsToBullets_NilViolations(t *testing.T) {
	mapped := MapViolationsToBullets(nil, map[int]string{42: "bullet_001"}, nil, nil, nil)
	assert.Nil(t, mapped)
}

func TestMapViolationsToBullets_MissingBulletInMap(t *testing.T) {
	lineNum := 42

	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:       "line_too_long",
				Severity:   "error",
				Details:    "Line exceeds maximum character count",
				LineNumber: &lineNum,
			},
		},
	}

	lineToBulletMap := map[int]string{
		42: "bullet_001",
	}

	// Bullet not in bullets map
	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_002", // Different bullet
				FinalText:        "Other bullet",
			},
		},
	}

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
		},
	}

	mapped := MapViolationsToBullets(violations, lineToBulletMap, bullets, plan, nil)
	require.NotNil(t, mapped)
	require.Equal(t, 1, len(mapped.Violations))

	violation := mapped.Violations[0]
	// Should still have bullet_id and story_id, but no bullet_text
	assert.NotNil(t, violation.BulletID)
	assert.Equal(t, "bullet_001", *violation.BulletID)
	assert.NotNil(t, violation.StoryID)
	assert.Equal(t, "story_001", *violation.StoryID)
	assert.Nil(t, violation.BulletText) // Not found in bullets map
}
