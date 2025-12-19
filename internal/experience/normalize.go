// Package experience provides functionality to load and normalize experience bank files.
package experience

import (
	"fmt"
	"strings"

	"github.com/jonathan/resume-customizer/internal/parsing"
	"github.com/jonathan/resume-customizer/internal/types"
)

// NormalizeExperienceBank applies all normalization steps to an experience bank
func NormalizeExperienceBank(bank *types.ExperienceBank) error {
	// Normalize skills first (before computing length)
	NormalizeSkills(bank)

	// Compute missing length_chars
	ComputeLengthChars(bank)

	// Validate evidence strength
	if err := ValidateEvidenceStrength(bank); err != nil {
		return err
	}

	return nil
}

// NormalizeSkills normalizes skill names in all bullets and deduplicates them
func NormalizeSkills(bank *types.ExperienceBank) {
	for i := range bank.Stories {
		for j := range bank.Stories[i].Bullets {
			bullet := &bank.Stories[i].Bullets[j]
			normalized := make([]string, 0, len(bullet.Skills))
			seen := make(map[string]struct{})

			for _, skill := range bullet.Skills {
				normalizedSkill := parsing.NormalizeSkillName(skill)
				if normalizedSkill == "" {
					continue // Skip empty skills
				}
				if _, exists := seen[normalizedSkill]; !exists {
					normalized = append(normalized, normalizedSkill)
					seen[normalizedSkill] = struct{}{}
				}
			}

			bullet.Skills = normalized
		}
	}
}

// ComputeLengthChars computes the character length of bullet text if it's missing or zero
func ComputeLengthChars(bank *types.ExperienceBank) {
	for i := range bank.Stories {
		for j := range bank.Stories[i].Bullets {
			bullet := &bank.Stories[i].Bullets[j]
			if bullet.LengthChars == 0 {
				bullet.LengthChars = len(bullet.Text)
			}
		}
	}
}

// ValidateEvidenceStrength validates that all evidence strength values are valid
func ValidateEvidenceStrength(bank *types.ExperienceBank) error {
	validStrengths := map[string]bool{
		"high":   true,
		"medium": true,
		"low":    true,
	}

	for i, story := range bank.Stories {
		for j, bullet := range story.Bullets {
			if !validStrengths[strings.ToLower(bullet.EvidenceStrength)] {
				return &NormalizationError{
					Message: fmt.Sprintf("invalid evidence_strength '%s' in story '%s', bullet '%s'", bullet.EvidenceStrength, story.ID, bullet.ID),
				}
			}
			// Normalize to lowercase for consistency
			bank.Stories[i].Bullets[j].EvidenceStrength = strings.ToLower(bullet.EvidenceStrength)
		}
	}

	return nil
}

