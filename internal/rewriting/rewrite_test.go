// Package rewriting provides functionality to rewrite bullet points to match job requirements and company brand voice.
package rewriting

import (
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
	assert.Contains(t, prompt, "15 characters")
}

func TestBuildRewritingPrompt_NilProfiles(t *testing.T) {
	bullet := types.SelectedBullet{
		ID:          "bullet_001",
		Text:        "Built a system",
		LengthChars: 15,
	}

	prompt := buildRewritingPrompt(bullet, nil, nil, []string{})

	assert.Contains(t, prompt, "Built a system")
	assert.Contains(t, prompt, "15 characters")
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
