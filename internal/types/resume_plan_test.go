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

func TestResumePlan_JSONMarshaling(t *testing.T) {
	plan := ResumePlan{
		SelectedStories: []SelectedStory{
			{
				StoryID:        "story_001",
				BulletIDs:      []string{"bullet_001", "bullet_002"},
				Section:        "experience",
				EstimatedLines: 2,
			},
		},
		SpaceBudget: SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
			Sections: map[string]int{
				"experience": 30,
			},
		},
		Coverage: Coverage{
			TopSkillsCovered: []string{"Go", "Kubernetes"},
			CoverageScore:    0.85,
		},
	}

	jsonBytes, err := json.MarshalIndent(plan, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"story_id": "story_001"`)
	assert.Contains(t, string(jsonBytes), `"bullet_ids":`)
	assert.Contains(t, string(jsonBytes), `"section": "experience"`)
	assert.Contains(t, string(jsonBytes), `"max_bullets": 8`)
	assert.Contains(t, string(jsonBytes), `"top_skills_covered":`)
	assert.Contains(t, string(jsonBytes), `"coverage_score": 0.85`)
}

func TestResumePlan_JSONUnmarshaling(t *testing.T) {
	jsonInput := `{
		"selected_stories": [
			{
				"story_id": "story_001",
				"bullet_ids": ["bullet_001"],
				"section": "experience",
				"estimated_lines": 2
			}
		],
		"space_budget": {
			"max_bullets": 8,
			"max_lines": 45
		},
		"coverage": {
			"top_skills_covered": ["Go"],
			"coverage_score": 0.75
		}
	}`

	var plan ResumePlan
	err := json.Unmarshal([]byte(jsonInput), &plan)
	require.NoError(t, err)
	assert.Len(t, plan.SelectedStories, 1)
	assert.Equal(t, "story_001", plan.SelectedStories[0].StoryID)
	assert.Equal(t, 8, plan.SpaceBudget.MaxBullets)
	assert.Equal(t, 45, plan.SpaceBudget.MaxLines)
	assert.Contains(t, plan.Coverage.TopSkillsCovered, "Go")
	assert.Equal(t, 0.75, plan.Coverage.CoverageScore)
}

func TestResumePlan_EmptyStories(t *testing.T) {
	plan := ResumePlan{
		SelectedStories: []SelectedStory{},
		SpaceBudget: SpaceBudget{
			MaxBullets: 8,
			MaxLines:   45,
		},
		Coverage: Coverage{
			TopSkillsCovered: []string{},
			CoverageScore:    0.0,
		},
	}

	jsonBytes, err := json.Marshal(plan)
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"selected_stories":[]`)
}

func TestSpaceBudget_WithSections(t *testing.T) {
	budget := SpaceBudget{
		MaxBullets: 10,
		MaxLines:   50,
		Sections: map[string]int{
			"experience": 30,
			"skills":     10,
		},
	}

	jsonBytes, err := json.Marshal(budget)
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"sections"`)
	assert.Contains(t, string(jsonBytes), `"experience":30`)
}

func TestSpaceBudget_WithoutSections(t *testing.T) {
	budget := SpaceBudget{
		MaxBullets: 10,
		MaxLines:   50,
	}

	jsonBytes, err := json.Marshal(budget)
	require.NoError(t, err)
	// Sections field should be omitted when empty
	jsonStr := string(jsonBytes)
	assert.NotContains(t, jsonStr, `"sections"`)
}
