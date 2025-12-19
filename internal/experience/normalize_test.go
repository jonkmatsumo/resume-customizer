package experience

import (
	"strings"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSkills_NormalizesSkillNames(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:     "bullet_001",
						Text:   "Built system",
						Skills: []string{"golang", "javascript", "typescript", "k8s"},
					},
				},
			},
		},
	}

	NormalizeSkills(bank)

	skills := bank.Stories[0].Bullets[0].Skills
	// golang -> Go, javascript -> JavaScript, typescript -> TypeScript, k8s -> Kubernetes
	assert.Contains(t, skills, "Go")
	assert.Contains(t, skills, "JavaScript")
	assert.Contains(t, skills, "TypeScript")
	assert.Contains(t, skills, "Kubernetes")
}

func TestNormalizeSkills_DeduplicatesSkills(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:     "bullet_001",
						Text:   "Built system",
						Skills: []string{"Go", "go", "golang", "JavaScript", "javascript"},
					},
				},
			},
		},
	}

	NormalizeSkills(bank)

	skills := bank.Stories[0].Bullets[0].Skills
	// All variations should normalize to the same skills and be deduplicated
	assert.Len(t, skills, 2)
	assert.Contains(t, skills, "Go")
	assert.Contains(t, skills, "JavaScript")
}

func TestNormalizeSkills_HandlesEmptySkills(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:     "bullet_001",
						Text:   "Built system",
						Skills: []string{},
					},
				},
			},
		},
	}

	NormalizeSkills(bank)

	assert.Len(t, bank.Stories[0].Bullets[0].Skills, 0)
}

func TestNormalizeSkills_HandlesEmptySkillStrings(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:     "bullet_001",
						Text:   "Built system",
						Skills: []string{"Go", "", "  ", "JavaScript"},
					},
				},
			},
		},
	}

	NormalizeSkills(bank)

	skills := bank.Stories[0].Bullets[0].Skills
	// Empty strings should be filtered out
	assert.Len(t, skills, 2)
	assert.Contains(t, skills, "Go")
	assert.Contains(t, skills, "JavaScript")
}

func TestNormalizeSkills_MultipleBullets(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:     "bullet_001",
						Text:   "Built system",
						Skills: []string{"golang"},
					},
					{
						ID:     "bullet_002",
						Text:   "Optimized code",
						Skills: []string{"javascript"},
					},
				},
			},
		},
	}

	NormalizeSkills(bank)

	assert.Equal(t, []string{"Go"}, bank.Stories[0].Bullets[0].Skills)
	assert.Equal(t, []string{"JavaScript"}, bank.Stories[0].Bullets[1].Skills)
}

func TestComputeLengthChars_ComputesMissingLength(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_001",
						Text:        "This is a test bullet",
						LengthChars: 0, // Missing/zero
					},
				},
			},
		},
	}

	ComputeLengthChars(bank)

	assert.Equal(t, len("This is a test bullet"), bank.Stories[0].Bullets[0].LengthChars)
}

func TestComputeLengthChars_DoesNotOverwriteExisting(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_001",
						Text:        "Short text",
						LengthChars: 100, // Existing non-zero value
					},
				},
			},
		},
	}

	ComputeLengthChars(bank)

	// Should not overwrite existing non-zero value
	assert.Equal(t, 100, bank.Stories[0].Bullets[0].LengthChars)
}

func TestComputeLengthChars_HandlesEmptyText(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_001",
						Text:        "",
						LengthChars: 0,
					},
				},
			},
		},
	}

	ComputeLengthChars(bank)

	assert.Equal(t, 0, bank.Stories[0].Bullets[0].LengthChars)
}

func TestComputeLengthChars_MultipleBullets(t *testing.T) {
	bank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test",
				Role:    "Engineer",
				Bullets: []types.Bullet{
					{
						ID:          "bullet_001",
						Text:        "First bullet",
						LengthChars: 0,
					},
					{
						ID:          "bullet_002",
						Text:        "Second bullet text",
						LengthChars: 50, // Existing value
					},
					{
						ID:          "bullet_003",
						Text:        "Third",
						LengthChars: 0,
					},
				},
			},
		},
	}

	ComputeLengthChars(bank)

	assert.Equal(t, len("First bullet"), bank.Stories[0].Bullets[0].LengthChars)
	assert.Equal(t, 50, bank.Stories[0].Bullets[1].LengthChars) // Unchanged
	assert.Equal(t, len("Third"), bank.Stories[0].Bullets[2].LengthChars)
}

func TestValidateEvidenceStrength_ValidValues(t *testing.T) {
	tests := []struct {
		name   string
		values []string
	}{
		{"high", []string{"high"}},
		{"medium", []string{"medium"}},
		{"low", []string{"low"}},
		{"mixed case", []string{"HIGH", "Medium", "low"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bank := &types.ExperienceBank{
				Stories: []types.Story{
					{
						ID:      "story_001",
						Company: "Test",
						Role:    "Engineer",
						Bullets: make([]types.Bullet, len(tt.values)),
					},
				},
			}

			for i, value := range tt.values {
				bank.Stories[0].Bullets[i] = types.Bullet{
					ID:               "bullet_001",
					Text:             "Test",
					EvidenceStrength: value,
				}
			}

			err := ValidateEvidenceStrength(bank)
			assert.NoError(t, err)

			// All values should be normalized to lowercase
			for i := range bank.Stories[0].Bullets {
				assert.Equal(t, strings.ToLower(tt.values[i]), bank.Stories[0].Bullets[i].EvidenceStrength)
			}
		})
	}
}

func TestValidateEvidenceStrength_InvalidValue(t *testing.T) {
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
						EvidenceStrength: "invalid",
					},
				},
			},
		},
	}

	err := ValidateEvidenceStrength(bank)
	require.Error(t, err)

	normErr, ok := err.(*NormalizationError)
	require.True(t, ok, "error should be NormalizationError type")
	assert.Contains(t, normErr.Error(), "invalid evidence_strength")
	assert.Contains(t, normErr.Error(), "story_001")
	assert.Contains(t, normErr.Error(), "bullet_001")
}

func TestValidateEvidenceStrength_MultipleStories(t *testing.T) {
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
						EvidenceStrength: "high",
					},
				},
			},
			{
				ID:      "story_002",
				Company: "Test2",
				Role:    "Engineer2",
				Bullets: []types.Bullet{
					{
						ID:               "bullet_002",
						Text:             "Test2",
						EvidenceStrength: "bad_value",
					},
				},
			},
		},
	}

	err := ValidateEvidenceStrength(bank)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "story_002")
	assert.Contains(t, err.Error(), "bullet_002")
}
