//go:build integration
// +build integration

package rewriting

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteBullets_RealAPI(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	selectedBullets := &types.SelectedBullets{
		Bullets: []types.SelectedBullet{
			{
				ID:          "bullet_001",
				StoryID:     "story_001",
				Text:        "Built a system with Go",
				Skills:      []string{"Go", "Kubernetes"},
				LengthChars: 25,
			},
		},
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
		TabooPhrases: []string{"synergy", "ninja"},
		Values:       []string{"Ownership"},
	}

	ctx := context.Background()
	rewritten, err := RewriteBullets(ctx, selectedBullets, jobProfile, companyProfile, apiKey)
	require.NoError(t, err)
	require.NotNil(t, rewritten)
	require.Len(t, rewritten.Bullets, 1)

	bullet := rewritten.Bullets[0]
	assert.Equal(t, "bullet_001", bullet.OriginalBulletID)
	assert.NotEmpty(t, bullet.FinalText)
	assert.Greater(t, bullet.LengthChars, 0)
	assert.GreaterOrEqual(t, bullet.EstimatedLines, 1)
	assert.NotNil(t, bullet.StyleChecks)

	// Verify style checks are populated
	assert.NotNil(t, bullet.StyleChecks)
}

func TestRewriteBullets_SchemaValidation(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	selectedBullets := &types.SelectedBullets{
		Bullets: []types.SelectedBullet{
			{
				ID:          "bullet_001",
				StoryID:     "story_001",
				Text:        "Worked on a project",
				Skills:      []string{"Python"},
				LengthChars: 20,
			},
		},
	}

	jobProfile := &types.JobProfile{
		Keywords: []string{"data"},
	}

	companyProfile := &types.CompanyProfile{
		Tone:       "professional",
		StyleRules: []string{"Use active voice"},
	}

	ctx := context.Background()
	rewritten, err := RewriteBullets(ctx, selectedBullets, jobProfile, companyProfile, apiKey)
	require.NoError(t, err)

	// Marshal to JSON
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "rewritten_bullets.json")

	jsonBytes, err := json.MarshalIndent(rewritten, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(outputPath, jsonBytes, 0644)
	require.NoError(t, err)

	// Validate against schema
	schemaPath := filepath.Join("..", "..", "schemas", "bullets.schema.json")
	err = schemas.ValidateJSON(schemaPath, outputPath)
	assert.NoError(t, err, "generated rewritten bullets should validate against schema")
}

func TestRewriteBullets_MultipleBullets(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	selectedBullets := &types.SelectedBullets{
		Bullets: []types.SelectedBullet{
			{
				ID:          "bullet_001",
				StoryID:     "story_001",
				Text:        "Built a system",
				Skills:      []string{"Go"},
				LengthChars: 15,
			},
			{
				ID:          "bullet_002",
				StoryID:     "story_001",
				Text:        "Designed architecture",
				Skills:      []string{"Architecture"},
				LengthChars: 22,
			},
		},
	}

	jobProfile := &types.JobProfile{
		Keywords: []string{"system"},
	}

	companyProfile := &types.CompanyProfile{
		Tone: "professional",
	}

	ctx := context.Background()
	rewritten, err := RewriteBullets(ctx, selectedBullets, jobProfile, companyProfile, apiKey)
	require.NoError(t, err)
	require.Len(t, rewritten.Bullets, 2)

	assert.Equal(t, "bullet_001", rewritten.Bullets[0].OriginalBulletID)
	assert.Equal(t, "bullet_002", rewritten.Bullets[1].OriginalBulletID)
}

func TestRewriteBullets_MissingAPIKey(t *testing.T) {
	selectedBullets := &types.SelectedBullets{
		Bullets: []types.SelectedBullet{},
	}

	ctx := context.Background()
	_, err := RewriteBullets(ctx, selectedBullets, nil, nil, "")
	assert.Error(t, err)
	var apiErr *APICallError
	assert.ErrorAs(t, err, &apiErr)
	assert.Contains(t, err.Error(), "API key is required")
}

