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

func TestRankedStories_JSONMarshaling(t *testing.T) {
	ranked := RankedStories{
		Ranked: []RankedStory{
			{
				StoryID:          "story_001",
				RelevanceScore:   0.85,
				SkillOverlap:     0.9,
				KeywordOverlap:   0.7,
				EvidenceStrength: 1.0,
				MatchedSkills:    []string{"Go", "Kubernetes"},
				Notes:            "Strong skill match (Go, Kubernetes). High evidence strength. Good keyword overlap.",
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(ranked, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"story_id": "story_001"`)
	assert.Contains(t, string(jsonBytes), `"relevance_score": 0.85`)
	assert.Contains(t, string(jsonBytes), `"skill_overlap": 0.9`)
	assert.Contains(t, string(jsonBytes), `"Go"`)
	assert.Contains(t, string(jsonBytes), `"Kubernetes"`)
	assert.Contains(t, string(jsonBytes), `"matched_skills"`)
}

func TestRankedStories_JSONUnmarshaling(t *testing.T) {
	jsonInput := `{
		"ranked": [
			{
				"story_id": "story_002",
				"relevance_score": 0.65,
				"skill_overlap": 0.5,
				"keyword_overlap": 0.3,
				"evidence_strength": 0.6,
				"matched_skills": ["Python"],
				"notes": "Moderate skill match (Python). Medium evidence strength."
			}
		]
	}`

	var ranked RankedStories
	err := json.Unmarshal([]byte(jsonInput), &ranked)
	require.NoError(t, err)
	assert.Len(t, ranked.Ranked, 1)
	assert.Equal(t, "story_002", ranked.Ranked[0].StoryID)
	assert.Equal(t, 0.65, ranked.Ranked[0].RelevanceScore)
	assert.Equal(t, 0.5, ranked.Ranked[0].SkillOverlap)
	assert.Equal(t, []string{"Python"}, ranked.Ranked[0].MatchedSkills)
	assert.Equal(t, "Moderate skill match (Python). Medium evidence strength.", ranked.Ranked[0].Notes)
}

func TestRankedStories_EmptyRanked(t *testing.T) {
	ranked := RankedStories{
		Ranked: []RankedStory{},
	}

	jsonBytes, err := json.Marshal(ranked)
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"ranked":[]`)
}

func TestRankedStory_AllFields(t *testing.T) {
	story := RankedStory{
		StoryID:          "story_003",
		RelevanceScore:   0.75,
		SkillOverlap:     0.8,
		KeywordOverlap:   0.6,
		EvidenceStrength: 0.9,
		MatchedSkills:    []string{"Go", "Distributed Systems"},
		Notes:            "Test notes",
	}

	jsonBytes, err := json.Marshal(story)
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"story_id":"story_003"`)
	assert.Contains(t, string(jsonBytes), `"relevance_score":0.75`)
	assert.Contains(t, string(jsonBytes), `"skill_overlap":0.8`)
	assert.Contains(t, string(jsonBytes), `"keyword_overlap":0.6`)
	assert.Contains(t, string(jsonBytes), `"evidence_strength":0.9`)
	assert.Contains(t, string(jsonBytes), `"matched_skills":["Go","Distributed Systems"]`)
	assert.Contains(t, string(jsonBytes), `"notes":"Test notes"`)
}
