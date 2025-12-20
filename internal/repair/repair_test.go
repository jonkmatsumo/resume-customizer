// Package repair provides functionality to automatically fix violations in LaTeX resumes.
package repair

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRepairResponse_ValidJSON(t *testing.T) {
	jsonText := `{
  "actions": [
    {
      "type": "shorten_bullet",
      "bullet_id": "bullet_001",
      "target_chars": 80,
      "reason": "Bullet is too long"
    }
  ]
}`

	actions, err := parseRepairResponse(jsonText)
	require.NoError(t, err)
	require.NotNil(t, actions)
	require.Len(t, actions.Actions, 1)
	assert.Equal(t, "shorten_bullet", actions.Actions[0].Type)
	assert.Equal(t, "bullet_001", actions.Actions[0].BulletID)
	assert.NotNil(t, actions.Actions[0].TargetChars)
	assert.Equal(t, 80, *actions.Actions[0].TargetChars)
}

func TestParseRepairResponse_MarkdownCodeBlock(t *testing.T) {
	jsonText := "```json\n{\n  \"actions\": [\n    {\n      \"type\": \"drop_bullet\",\n      \"bullet_id\": \"bullet_002\",\n      \"reason\": \"Remove to save space\"\n    }\n  ]\n}\n```"

	actions, err := parseRepairResponse(jsonText)
	require.NoError(t, err)
	require.NotNil(t, actions)
	require.Len(t, actions.Actions, 1)
	assert.Equal(t, "drop_bullet", actions.Actions[0].Type)
	assert.Equal(t, "bullet_002", actions.Actions[0].BulletID)
}

func TestExtractJSONFromText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "JSON in markdown code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON in generic code block",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Plain JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateProposedActions_Valid(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001", "bullet_002"},
			},
		},
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				LengthChars:      100,
			},
		},
	}

	rankedStories := &types.RankedStories{
		Ranked: []types.RankedStory{
			{
				StoryID:        "story_002",
				RelevanceScore: 0.8,
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
				Reason:      "Too long",
			},
		},
	}

	err := validateProposedActions(actions, plan, bullets, rankedStories)
	assert.NoError(t, err)
}

func TestValidateProposedActions_InvalidBulletID(t *testing.T) {
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
				LengthChars:      100,
			},
		},
	}

	rankedStories := &types.RankedStories{Ranked: []types.RankedStory{}}

	targetChars := 80
	actions := &types.RepairActions{
		Actions: []types.RepairAction{
			{
				Type:        "shorten_bullet",
				BulletID:    "nonexistent_bullet",
				TargetChars: &targetChars,
				Reason:      "Invalid",
			},
		},
	}

	err := validateProposedActions(actions, plan, bullets, rankedStories)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in rewritten bullets")
}

func TestValidateProposedActions_InvalidStoryID(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
		},
	}

	bullets := &types.RewrittenBullets{Bullets: []types.RewrittenBullet{}}
	rankedStories := &types.RankedStories{Ranked: []types.RankedStory{}}

	actions := &types.RepairActions{
		Actions: []types.RepairAction{
			{
				Type:    "swap_story",
				StoryID: "nonexistent_story",
				Reason:  "Invalid",
			},
		},
	}

	err := validateProposedActions(actions, plan, bullets, rankedStories)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in plan")
}

func TestBuildRepairPrompt_IncludesViolations(t *testing.T) {
	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:     "page_overflow",
				Severity: "error",
				Details:  "Resume has 2 pages, maximum is 1",
			},
		},
	}

	plan := &types.ResumePlan{SelectedStories: []types.SelectedStory{}}
	bullets := &types.RewrittenBullets{Bullets: []types.RewrittenBullet{}}
	rankedStories := &types.RankedStories{Ranked: []types.RankedStory{}}
	jobProfile := &types.JobProfile{RoleTitle: "Engineer"}
	companyProfile := &types.CompanyProfile{}

	prompt := buildRepairPrompt(violations, plan, bullets, rankedStories, jobProfile, companyProfile)

	assert.Contains(t, prompt, "page_overflow")
	assert.Contains(t, prompt, "Resume has 2 pages")
	assert.Contains(t, prompt, "Engineer")
}
