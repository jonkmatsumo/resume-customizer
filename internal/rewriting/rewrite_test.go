// Package rewriting provides functionality to rewrite bullet points to match job requirements and company brand voice.
package rewriting

import (
	"context"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRewritingPrompt(t *testing.T) {
	bullet := types.SelectedBullet{
		ID:          "bullet_001",
		StoryID:     "story_001",
		Text:        "Built a system",
		Skills:      []string{"Go", "Kubernetes"},
		LengthChars: 15,
	}

	jobProfile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
		Keywords: []string{"microservices", "distributed systems"},
	}

	companyProfile := &types.CompanyProfile{
		Tone:         "direct, metric-driven",
		StyleRules:   []string{"Lead with metrics", "Avoid hype"},
		TabooPhrases: []string{"synergy"},
	}

	prompt := buildRewritingPrompt(bullet, jobProfile, companyProfile, []string{})

	assert.Contains(t, prompt, "Built a system")
	assert.Contains(t, prompt, "Go")
	assert.Contains(t, prompt, "microservices")
	assert.Contains(t, prompt, "direct, metric-driven")
	assert.Contains(t, prompt, "Lead with metrics")
	assert.Contains(t, prompt, "synergy")
	assert.Contains(t, prompt, "200 characters")
}

func TestBuildRewritingPrompt_NilProfiles(t *testing.T) {
	bullet := types.SelectedBullet{
		ID:          "bullet_001",
		Text:        "Built a system",
		LengthChars: 15,
	}

	prompt := buildRewritingPrompt(bullet, nil, nil, []string{})

	assert.Contains(t, prompt, "Built a system")
	assert.Contains(t, prompt, "200 characters")
}

func TestParseBulletResponse_PlainText(t *testing.T) {
	responseText := "Built a scalable system handling 1M requests/day"

	text, err := parseBulletResponse(responseText)
	require.NoError(t, err)
	assert.Equal(t, "Built a scalable system handling 1M requests/day", text)
}

func TestParseBulletResponse_JSONWrapped(t *testing.T) {
	responseText := `{"text": "Built a scalable system"}`

	text, err := parseBulletResponse(responseText)
	require.NoError(t, err)
	assert.Equal(t, "Built a scalable system", text)
}

func TestParseBulletResponse_Whitespace(t *testing.T) {
	responseText := "   Built a system   "

	text, err := parseBulletResponse(responseText)
	require.NoError(t, err)
	assert.Equal(t, "Built a system", text)
}

func TestParseBulletResponse_WithCodeBlocks(t *testing.T) {
	responseText := "```\nBuilt a scalable system\n```"

	text, err := parseBulletResponse(responseText)
	require.NoError(t, err)
	assert.Contains(t, text, "Built a scalable system")
	assert.NotContains(t, text, "```")
}

func TestPostProcessBullet_ValidBullet(t *testing.T) {
	rewrittenText := "Built a scalable system handling 1M requests/day"
	originalBullet := types.SelectedBullet{
		ID:          "bullet_001",
		StoryID:     "story_001",
		Text:        "Built a system",
		Skills:      []string{"Go"},
		LengthChars: 15,
	}

	companyProfile := &types.CompanyProfile{
		TabooPhrases: []string{"synergy"},
	}

	rewritten, err := postProcessBullet(rewrittenText, originalBullet, companyProfile)
	require.NoError(t, err)

	assert.Equal(t, "bullet_001", rewritten.OriginalBulletID)
	assert.Equal(t, rewrittenText, rewritten.FinalText)
	assert.Equal(t, len(rewrittenText), rewritten.LengthChars)
	assert.GreaterOrEqual(t, rewritten.EstimatedLines, 1)
	assert.NotNil(t, rewritten.StyleChecks)
}

func TestPostProcessBullet_NoCompanyProfile(t *testing.T) {
	rewrittenText := "Built a system"
	originalBullet := types.SelectedBullet{
		ID:          "bullet_001",
		LengthChars: 15,
	}

	rewritten, err := postProcessBullet(rewrittenText, originalBullet, nil)
	require.NoError(t, err)

	assert.Equal(t, "bullet_001", rewritten.OriginalBulletID)
	assert.Equal(t, rewrittenText, rewritten.FinalText)
	assert.NotNil(t, rewritten.StyleChecks)
}

func TestExtractLeadingVerb(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "Simple verb",
			text:     "Built a system",
			expected: "Built",
		},
		{
			name:     "Verb with comma",
			text:     "Designed, a system",
			expected: "Designed",
		},
		{
			name:     "Empty string",
			text:     "",
			expected: "",
		},
		{
			name:     "Whitespace only",
			text:     "   ",
			expected: "",
		},
		{
			name:     "Single word",
			text:     "Achieved",
			expected: "Achieved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLeadingVerb(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRewriteBulletsSelective_EmptyBulletsToRewrite tests that no rewriting occurs when list is empty
func TestRewriteBulletsSelective_EmptyBulletsToRewrite(t *testing.T) {
	currentBullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Preserved bullet",
				LengthChars:      50,
			},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{},
	}

	// Empty bulletsToRewrite should return preserved bullets unchanged
	result, err := RewriteBulletsSelective(
		context.TODO(), // context not needed for empty case
		currentBullets,
		[]string{}, // No bullets to rewrite
		nil,
		nil,
		experienceBank,
		"", // API key not needed for empty case
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, len(result.Bullets))
	assert.Equal(t, "bullet_001", result.Bullets[0].OriginalBulletID)
	assert.Equal(t, "Preserved bullet", result.Bullets[0].FinalText)
}

// TestRewriteBulletsSelective_MissingBulletInExperienceBank tests handling of bullets not in experienceBank
func TestRewriteBulletsSelective_MissingBulletInExperienceBank(t *testing.T) {
	currentBullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Existing bullet",
				LengthChars:      50,
			},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{}, // Empty - bullet_002 not found
	}

	// Try to rewrite bullet_002 which doesn't exist in experienceBank
	// Should skip it and return preserved bullets
	result, err := RewriteBulletsSelective(
		context.TODO(),
		currentBullets,
		[]string{"bullet_002"}, // Bullet not in experienceBank
		nil,
		nil,
		experienceBank,
		"", // API key not needed since no bullets will be rewritten
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	// Should return preserved bullets since bullet_002 wasn't found
	assert.Equal(t, 1, len(result.Bullets))
	assert.Equal(t, "bullet_001", result.Bullets[0].OriginalBulletID)
}
