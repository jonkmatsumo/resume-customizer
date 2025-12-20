// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectedBullet_JSONMarshaling(t *testing.T) {
	bullet := SelectedBullet{
		ID:          "bullet_001",
		StoryID:     "story_001",
		Text:        "Built Go microservices processing 1M+ requests/day",
		Skills:      []string{"Go", "Kubernetes"},
		Metrics:     "1M+ requests/day",
		LengthChars: 60,
	}

	jsonBytes, err := json.MarshalIndent(bullet, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"id": "bullet_001"`)
	assert.Contains(t, string(jsonBytes), `"story_id": "story_001"`)
	assert.Contains(t, string(jsonBytes), `"text": "Built Go microservices`)
	assert.Contains(t, string(jsonBytes), `"skills":`)
	assert.Contains(t, string(jsonBytes), `"Go"`)
	assert.Contains(t, string(jsonBytes), `"metrics": "1M+ requests/day"`)
	assert.Contains(t, string(jsonBytes), `"length_chars": 60`)
}

func TestSelectedBullet_JSONUnmarshaling(t *testing.T) {
	jsonInput := `{
		"id": "bullet_001",
		"story_id": "story_001",
		"text": "Built Go microservices",
		"skills": ["Go", "Kubernetes"],
		"metrics": "1M+ requests/day",
		"length_chars": 60
	}`

	var bullet SelectedBullet
	err := json.Unmarshal([]byte(jsonInput), &bullet)
	require.NoError(t, err)
	assert.Equal(t, "bullet_001", bullet.ID)
	assert.Equal(t, "story_001", bullet.StoryID)
	assert.Equal(t, "Built Go microservices", bullet.Text)
	assert.Equal(t, []string{"Go", "Kubernetes"}, bullet.Skills)
	assert.Equal(t, "1M+ requests/day", bullet.Metrics)
	assert.Equal(t, 60, bullet.LengthChars)
}

func TestSelectedBullet_OptionalFields(t *testing.T) {
	// Test with metrics present
	bulletWithMetrics := SelectedBullet{
		ID:          "bullet_001",
		StoryID:     "story_001",
		Text:        "Built system",
		Skills:      []string{"Go"},
		Metrics:     "1M+ requests",
		LengthChars: 20,
	}

	jsonBytes, err := json.Marshal(bulletWithMetrics)
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"metrics"`)

	// Test with metrics empty (should still be present in JSON, just empty string)
	bulletEmptyMetrics := SelectedBullet{
		ID:          "bullet_002",
		StoryID:     "story_001",
		Text:        "Built system",
		Skills:      []string{"Go"},
		Metrics:     "",
		LengthChars: 20,
	}

	jsonBytes2, err := json.Marshal(bulletEmptyMetrics)
	require.NoError(t, err)
	// Empty string metrics will be in JSON, but omitempty tag means it can be omitted if empty
	// However, since we're setting it to empty string explicitly, it may still appear
	var unmarshaled SelectedBullet
	err = json.Unmarshal(jsonBytes2, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, "", unmarshaled.Metrics)
}

func TestSelectedBullets_JSONMarshaling(t *testing.T) {
	bullets := SelectedBullets{
		Bullets: []SelectedBullet{
			{
				ID:          "bullet_001",
				StoryID:     "story_001",
				Text:        "First bullet",
				Skills:      []string{"Go"},
				LengthChars: 20,
			},
			{
				ID:          "bullet_002",
				StoryID:     "story_001",
				Text:        "Second bullet",
				Skills:      []string{"Kubernetes"},
				LengthChars: 25,
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(bullets, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"bullets"`)
	assert.Contains(t, string(jsonBytes), `"bullet_001"`)
	assert.Contains(t, string(jsonBytes), `"bullet_002"`)
}

func TestSelectedBullets_JSONUnmarshaling(t *testing.T) {
	jsonInput := `{
		"bullets": [
			{
				"id": "bullet_001",
				"story_id": "story_001",
				"text": "First bullet",
				"skills": ["Go"],
				"length_chars": 20
			}
		]
	}`

	var bullets SelectedBullets
	err := json.Unmarshal([]byte(jsonInput), &bullets)
	require.NoError(t, err)
	assert.Len(t, bullets.Bullets, 1)
	assert.Equal(t, "bullet_001", bullets.Bullets[0].ID)
}

func TestSelectedBullets_EmptyBullets(t *testing.T) {
	bullets := SelectedBullets{
		Bullets: []SelectedBullet{},
	}

	jsonBytes, err := json.Marshal(bullets)
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"bullets":[]`)
}

func TestRewrittenBullet_JSONMarshaling(t *testing.T) {
	bullet := RewrittenBullet{
		OriginalBulletID: "bullet_001",
		FinalText:        "Built Go microservices handling 1M+ requests/day",
		LengthChars:      45,
		EstimatedLines:   1,
		StyleChecks: StyleChecks{
			StrongVerb:   true,
			Quantified:   true,
			NoTaboo:      true,
			TargetLength: true,
		},
	}

	jsonBytes, err := json.MarshalIndent(bullet, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"original_bullet_id": "bullet_001"`)
	assert.Contains(t, string(jsonBytes), `"final_text": "Built Go microservices handling 1M+ requests/day"`)
	assert.Contains(t, string(jsonBytes), `"length_chars": 45`)
	assert.Contains(t, string(jsonBytes), `"estimated_lines": 1`)
	assert.Contains(t, string(jsonBytes), `"style_checks"`)
	assert.Contains(t, string(jsonBytes), `"strong_verb": true`)
	assert.Contains(t, string(jsonBytes), `"quantified": true`)
	assert.Contains(t, string(jsonBytes), `"no_taboo": true`)
	assert.Contains(t, string(jsonBytes), `"target_length": true`)

	var unmarshaledBullet RewrittenBullet
	err = json.Unmarshal(jsonBytes, &unmarshaledBullet)
	require.NoError(t, err)
	assert.Equal(t, bullet, unmarshaledBullet)
}

func TestRewrittenBullets_JSONMarshaling(t *testing.T) {
	bullets := RewrittenBullets{
		Bullets: []RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Bullet 1 rewritten",
				LengthChars:      20,
				EstimatedLines:   1,
				StyleChecks: StyleChecks{
					StrongVerb:   true,
					Quantified:   false,
					NoTaboo:      true,
					TargetLength: true,
				},
			},
			{
				OriginalBulletID: "bullet_002",
				FinalText:        "Bullet 2 rewritten",
				LengthChars:      25,
				EstimatedLines:   1,
				StyleChecks: StyleChecks{
					StrongVerb:   true,
					Quantified:   true,
					NoTaboo:      true,
					TargetLength: false,
				},
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(bullets, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"bullets": [`)
	assert.Contains(t, string(jsonBytes), `"original_bullet_id": "bullet_001"`)
	assert.Contains(t, string(jsonBytes), `"original_bullet_id": "bullet_002"`)

	var unmarshaledBullets RewrittenBullets
	err = json.Unmarshal(jsonBytes, &unmarshaledBullets)
	require.NoError(t, err)
	assert.Equal(t, bullets, unmarshaledBullets)
}

func TestStyleChecks_JSONMarshaling(t *testing.T) {
	styleChecks := StyleChecks{
		StrongVerb:   true,
		Quantified:   false,
		NoTaboo:      true,
		TargetLength: false,
	}

	jsonBytes, err := json.Marshal(styleChecks)
	require.NoError(t, err)

	var unmarshaled StyleChecks
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, styleChecks, unmarshaled)
}
