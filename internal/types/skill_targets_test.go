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

func TestSkillTargets_JSONMarshaling(t *testing.T) {
	targets := SkillTargets{
		Skills: []Skill{
			{
				Name:   "Go",
				Weight: 1.0,
				Source: "hard_requirement",
			},
			{
				Name:   "Distributed Systems",
				Weight: 0.5,
				Source: "nice_to_have",
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(targets, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"name": "Go"`)
	assert.Contains(t, string(jsonBytes), `"weight": 1`)
	assert.Contains(t, string(jsonBytes), `"source": "hard_requirement"`)
}

func TestSkillTargets_JSONUnmarshaling(t *testing.T) {
	jsonInput := `{
		"skills": [
			{
				"name": "Go",
				"weight": 1.0,
				"source": "hard_requirement"
			},
			{
				"name": "Kubernetes",
				"weight": 0.5,
				"source": "nice_to_have"
			}
		]
	}`

	var targets SkillTargets
	err := json.Unmarshal([]byte(jsonInput), &targets)
	require.NoError(t, err)
	assert.Len(t, targets.Skills, 2)
	assert.Equal(t, "Go", targets.Skills[0].Name)
	assert.Equal(t, 1.0, targets.Skills[0].Weight)
	assert.Equal(t, "hard_requirement", targets.Skills[0].Source)
	assert.Equal(t, "Kubernetes", targets.Skills[1].Name)
	assert.Equal(t, 0.5, targets.Skills[1].Weight)
	assert.Equal(t, "nice_to_have", targets.Skills[1].Source)
}

func TestSkillTargets_EmptySkills(t *testing.T) {
	targets := SkillTargets{
		Skills: []Skill{},
	}

	jsonBytes, err := json.Marshal(targets)
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"skills":[]`)
}

func TestSkill_RequiredFields(t *testing.T) {
	skill := Skill{
		Name:   "JavaScript",
		Weight: 0.3,
		Source: "keyword",
	}

	jsonBytes, err := json.Marshal(skill)
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"name":"JavaScript"`)
	assert.Contains(t, string(jsonBytes), `"weight":0.3`)
	assert.Contains(t, string(jsonBytes), `"source":"keyword"`)
}
