// Package rewriting provides functionality to rewrite bullet points to match job requirements and company brand voice.
package rewriting

import (
	"strings"

	"github.com/jonathan/resume-customizer/internal/types"
)

// checkForbiddenPhrasesInText checks plain text for forbidden phrases
// Returns a list of forbidden phrases found in the text (case-insensitive)
func checkForbiddenPhrasesInText(text string, tabooPhrases []string) []string {
	if len(tabooPhrases) == 0 {
		return nil
	}

	// Normalize text to lowercase for case-insensitive matching
	normalizedText := strings.ToLower(text)

	var foundPhrases []string
	seen := make(map[string]bool) // Track seen phrases to avoid duplicates

	for _, phrase := range tabooPhrases {
		normalizedPhrase := strings.ToLower(strings.TrimSpace(phrase))
		if normalizedPhrase == "" {
			continue
		}

		// Check if phrase is in text (case-insensitive)
		if strings.Contains(normalizedText, normalizedPhrase) {
			if !seen[normalizedPhrase] {
				foundPhrases = append(foundPhrases, phrase) // Use original phrase, not normalized
				seen[normalizedPhrase] = true
			}
		}
	}

	// Return nil if no phrases found (idiomatic Go - nil slice vs empty slice)
	if len(foundPhrases) == 0 {
		return nil
	}

	return foundPhrases
}

// CheckForbiddenPhrasesInBullets checks all bullets for forbidden phrases
// Returns a map of bulletID → list of forbidden phrases found
func CheckForbiddenPhrasesInBullets(bullets *types.RewrittenBullets, companyProfile *types.CompanyProfile) map[string][]string {
	if bullets == nil || companyProfile == nil {
		return nil
	}

	if len(companyProfile.TabooPhrases) == 0 {
		return map[string][]string{}
	}

	result := make(map[string][]string)

	for i := range bullets.Bullets {
		bullet := &bullets.Bullets[i]
		foundPhrases := checkForbiddenPhrasesInText(bullet.FinalText, companyProfile.TabooPhrases)
		if len(foundPhrases) > 0 {
			result[bullet.OriginalBulletID] = foundPhrases
		}
	}

	return result
}
