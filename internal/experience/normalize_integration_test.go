package experience

import (
	"encoding/json"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeExperienceBank_EndToEnd(t *testing.T) {
	// Create a bank with various normalization needs
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				Company:   "Test Company",
				Role:      "Software Engineer",
				StartDate: "2020-01",
				EndDate:   "2023-06",
				Bullets: []types.Bullet{
					{
						ID:               "bullet_001",
						Text:             "Built distributed system with golang and javascript",
						Skills:           []string{"golang", "javascript", "golang"}, // Duplicates and synonyms
						Metrics:          "1M requests/day",
						LengthChars:      0,      // Missing
						EvidenceStrength: "HIGH", // Mixed case
						RiskFlags:        []string{},
					},
					{
						ID:               "bullet_002",
						Text:             "Optimized performance",
						Skills:           []string{"typescript", "k8s"},
						LengthChars:      20, // Already set
						EvidenceStrength: "medium",
						RiskFlags:        []string{"needs_citation"},
					},
				},
			},
		},
	}

	err := NormalizeExperienceBank(bank)
	require.NoError(t, err)

	// Verify skill normalization
	bullet1 := bank.Stories[0].Bullets[0]
	assert.Equal(t, []string{"Go", "JavaScript"}, bullet1.Skills)                                    // Normalized and deduplicated
	assert.Equal(t, "high", bullet1.EvidenceStrength)                                                // Normalized to lowercase
	assert.Equal(t, len("Built distributed system with golang and javascript"), bullet1.LengthChars) // Computed

	// Verify second bullet
	bullet2 := bank.Stories[0].Bullets[1]
	assert.Equal(t, []string{"TypeScript", "Kubernetes"}, bullet2.Skills)
	assert.Equal(t, "medium", bullet2.EvidenceStrength)
	assert.Equal(t, 20, bullet2.LengthChars) // Unchanged (was non-zero)

	// Verify structure is still valid
	assert.Equal(t, "story_001", bank.Stories[0].ID)
	assert.Len(t, bank.Stories[0].Bullets, 2)
}

func TestNormalizeExperienceBank_JSONRoundTrip(t *testing.T) {
	// Test that normalized bank can be marshaled and unmarshaled
	originalJSON := `{
		"stories": [
			{
				"id": "story_001",
				"company": "Test Company",
				"role": "Engineer",
				"start_date": "2020-01",
				"end_date": "2023-06",
				"bullets": [
					{
						"id": "bullet_001",
						"text": "Built system with golang",
						"skills": ["golang", "javascript"],
						"length_chars": 0,
						"evidence_strength": "HIGH",
						"risk_flags": []
					}
				]
			}
		]
	}`

	var bank types.ExperienceBank
	err := json.Unmarshal([]byte(originalJSON), &bank)
	require.NoError(t, err)

	// Normalize
	err = NormalizeExperienceBank(&bank)
	require.NoError(t, err)

	// Marshal back to JSON
	normalizedJSON, err := json.MarshalIndent(&bank, "", "  ")
	require.NoError(t, err)

	// Verify normalized JSON contains expected changes
	normalizedStr := string(normalizedJSON)
	assert.Contains(t, normalizedStr, `"Go"`)                        // Normalized skill
	assert.Contains(t, normalizedStr, `"JavaScript"`)                // Normalized skill
	assert.Contains(t, normalizedStr, `"evidence_strength": "high"`) // Lowercase
	assert.Contains(t, normalizedStr, `"length_chars": 24`)          // Computed length ("Built system with golang" = 24 chars)

	// Unmarshal again to verify it's still valid
	var roundTripBank types.ExperienceBank
	err = json.Unmarshal(normalizedJSON, &roundTripBank)
	require.NoError(t, err)

	assert.Equal(t, bank.Stories[0].Bullets[0].Skills, roundTripBank.Stories[0].Bullets[0].Skills)
	assert.Equal(t, bank.Stories[0].Bullets[0].EvidenceStrength, roundTripBank.Stories[0].Bullets[0].EvidenceStrength)
	assert.Equal(t, bank.Stories[0].Bullets[0].LengthChars, roundTripBank.Stories[0].Bullets[0].LengthChars)
}

func TestNormalizeExperienceBank_EmptyBank(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{},
	}

	err := NormalizeExperienceBank(bank)
	assert.NoError(t, err)
	assert.Len(t, bank.Stories, 0)
}

func TestNormalizeExperienceBank_StoryWithNoBullets(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				Company:   "Test",
				Role:      "Engineer",
				StartDate: "2020-01",
				EndDate:   "2023-06",
				Bullets:   []types.Bullet{},
			},
		},
	}

	err := NormalizeExperienceBank(bank)
	assert.NoError(t, err)
	assert.Len(t, bank.Stories[0].Bullets, 0)
}

func TestNormalizeExperienceBank_InvalidEvidenceStrength(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:               "bullet_001",
						Text:             "Test",
						Skills:           []string{"Go"},
						LengthChars:      4,
						EvidenceStrength: "invalid",
						RiskFlags:        []string{},
					},
				},
			},
		},
	}

	err := NormalizeExperienceBank(bank)
	require.Error(t, err)

	normErr, ok := err.(*NormalizationError)
	require.True(t, ok, "error should be NormalizationError type")
	assert.Contains(t, normErr.Error(), "invalid evidence_strength")
}

func TestNormalizeExperienceBank_MultipleStories(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Company A",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:               "bullet_001",
						Text:             "Built with golang",
						Skills:           []string{"golang"},
						LengthChars:      0,
						EvidenceStrength: "high",
						RiskFlags:        []string{},
					},
				},
			},
			{
				ID:      "story_002",
				Company: "Company B",
				Role:    "Senior Engineer",
				Bullets: []types.Bullet{
					{
						ID:               "bullet_002",
						Text:             "Optimized with typescript",
						Skills:           []string{"typescript"},
						LengthChars:      0,
						EvidenceStrength: "medium",
						RiskFlags:        []string{},
					},
				},
			},
		},
	}

	err := NormalizeExperienceBank(bank)
	require.NoError(t, err)

	// Verify both stories are normalized
	assert.Equal(t, []string{"Go"}, bank.Stories[0].Bullets[0].Skills)
	assert.Equal(t, []string{"TypeScript"}, bank.Stories[1].Bullets[0].Skills)
	assert.Equal(t, len("Built with golang"), bank.Stories[0].Bullets[0].LengthChars)
	assert.Equal(t, len("Optimized with typescript"), bank.Stories[1].Bullets[0].LengthChars)
}
