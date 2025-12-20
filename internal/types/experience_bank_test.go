package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExperienceBank_JSONMarshaling(t *testing.T) {
	bank := &ExperienceBank{
		Stories: []Story{
			{
				ID:        "story_001",
				Company:   "Test Company",
				Role:      "Software Engineer",
				StartDate: "2020-01",
				EndDate:   "2023-06",
				Bullets: []Bullet{
					{
						ID:               "bullet_001",
						Text:             "Built distributed system",
						Skills:           []string{"Go", "Distributed Systems"},
						Metrics:          "1M requests/day",
						LengthChars:      25,
						EvidenceStrength: "high",
						RiskFlags:        []string{},
					},
				},
			},
		},
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(bank)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	// Unmarshal back
	var unmarshaled ExperienceBank
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	// Verify structure
	assert.Len(t, unmarshaled.Stories, 1)
	assert.Equal(t, "story_001", unmarshaled.Stories[0].ID)
	assert.Equal(t, "Test Company", unmarshaled.Stories[0].Company)
	assert.Equal(t, "Software Engineer", unmarshaled.Stories[0].Role)
	assert.Equal(t, "2020-01", unmarshaled.Stories[0].StartDate)
	assert.Equal(t, "2023-06", unmarshaled.Stories[0].EndDate)

	assert.Len(t, unmarshaled.Stories[0].Bullets, 1)
	assert.Equal(t, "bullet_001", unmarshaled.Stories[0].Bullets[0].ID)
	assert.Equal(t, "Built distributed system", unmarshaled.Stories[0].Bullets[0].Text)
	assert.Equal(t, []string{"Go", "Distributed Systems"}, unmarshaled.Stories[0].Bullets[0].Skills)
	assert.Equal(t, "1M requests/day", unmarshaled.Stories[0].Bullets[0].Metrics)
	assert.Equal(t, 25, unmarshaled.Stories[0].Bullets[0].LengthChars)
	assert.Equal(t, "high", unmarshaled.Stories[0].Bullets[0].EvidenceStrength)
	assert.Equal(t, []string{}, unmarshaled.Stories[0].Bullets[0].RiskFlags)
}

func TestExperienceBank_OptionalMetrics(t *testing.T) {
	// Test that metrics field is optional (omitempty)
	bank := &ExperienceBank{
		Stories: []Story{
			{
				ID:        "story_001",
				Company:   "Test Company",
				Role:      "Software Engineer",
				StartDate: "2020-01",
				EndDate:   "2023-06",
				Bullets: []Bullet{
					{
						ID:               "bullet_001",
						Text:             "Built distributed system",
						Skills:           []string{"Go"},
						LengthChars:      25,
						EvidenceStrength: "high",
						RiskFlags:        []string{},
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(bank)
	require.NoError(t, err)

	// Metrics should not be in JSON if empty
	jsonStr := string(jsonBytes)
	assert.NotContains(t, jsonStr, `"metrics"`)
}

func TestExperienceBank_MultipleStories(t *testing.T) {
	bank := &ExperienceBank{
		Stories: []Story{
			{
				ID:        "story_001",
				Company:   "Company A",
				Role:      "Engineer",
				StartDate: "2020-01",
				EndDate:   "2021-01",
				Bullets:   []Bullet{},
			},
			{
				ID:        "story_002",
				Company:   "Company B",
				Role:      "Senior Engineer",
				StartDate: "2021-01",
				EndDate:   "present",
				Bullets: []Bullet{
					{
						ID:               "bullet_001",
						Text:             "Built system",
						Skills:           []string{"Go"},
						LengthChars:      12,
						EvidenceStrength: "medium",
						RiskFlags:        []string{"needs_citation"},
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(bank)
	require.NoError(t, err)

	var unmarshaled ExperienceBank
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	assert.Len(t, unmarshaled.Stories, 2)
	assert.Equal(t, "story_001", unmarshaled.Stories[0].ID)
	assert.Equal(t, "story_002", unmarshaled.Stories[1].ID)
	assert.Equal(t, "present", unmarshaled.Stories[1].EndDate)
	assert.Len(t, unmarshaled.Stories[0].Bullets, 0)
	assert.Len(t, unmarshaled.Stories[1].Bullets, 1)
	assert.Equal(t, []string{"needs_citation"}, unmarshaled.Stories[1].Bullets[0].RiskFlags)
}
