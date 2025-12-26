// Package rewriting provides functionality to rewrite bullet points to match job requirements and company brand voice.
package rewriting

import (
	"math"
	"regexp"
	"strings"

	"github.com/jonathan/resume-customizer/internal/types"
)

const (
	// charsPerLine is the estimated number of characters per line in the resume (same as selection package)
	charsPerLine = 100
	// lengthTolerancePercent is the percentage tolerance for target length (within 20% is acceptable)
	lengthTolerancePercent = 0.2
)

// Common strong action verbs for resume bullets (heuristic check)
var strongVerbs = map[string]bool{
	"achieved": true, "architected": true, "built": true, "created": true,
	"delivered": true, "designed": true, "developed": true, "engineered": true,
	"implemented": true, "improved": true, "increased": true, "launched": true,
	"led": true, "optimized": true, "reduced": true, "scaled": true,
	"shipped": true, "transformed": true,
}

// StyleChecksResult holds the results of style validation
type StyleChecksResult struct {
	StrongVerb   bool
	Quantified   bool
	NoTaboo      bool
	TargetLength bool
}

// ValidateStyle checks if rewritten text meets style requirements
func ValidateStyle(rewrittenText string, companyProfile *types.CompanyProfile, originalLengthChars int) StyleChecksResult {
	result := StyleChecksResult{}

	textLower := strings.ToLower(strings.TrimSpace(rewrittenText))

	// Check for strong verb (first word should be an action verb)
	result.StrongVerb = checkStrongVerb(textLower)

	// Check for quantified impact (contains numbers or percentages)
	result.Quantified = checkQuantifiedImpact(rewrittenText)

	// Check for taboo phrases
	result.NoTaboo = checkNoTaboo(textLower, companyProfile)

	// Check target length (within tolerance of original)
	result.TargetLength = checkTargetLength(len(rewrittenText), originalLengthChars)

	return result
}

// checkStrongVerb checks if text starts with a strong action verb
func checkStrongVerb(textLower string) bool {
	// Get first word
	words := strings.Fields(textLower)
	if len(words) == 0 {
		return false
	}

	firstWord := words[0]
	// Remove punctuation
	firstWord = strings.TrimRight(firstWord, ".,!?;:")

	// Check if it's in our strong verb list
	if strongVerbs[firstWord] {
		return true
	}

	// Additional check: verbs ending in -ed are often action verbs (past tense)
	if strings.HasSuffix(firstWord, "ed") && len(firstWord) > 3 {
		// Simple heuristic: if it ends in -ed and is reasonable length, likely a verb
		return true
	}

	return false
}

// checkQuantifiedImpact checks if text contains numbers or metrics
func checkQuantifiedImpact(text string) bool {
	// Check for digits
	hasDigits := regexp.MustCompile(`\d`).MatchString(text)
	if hasDigits {
		return true
	}

	// Check for percentage symbol
	if strings.Contains(text, "%") {
		return true
	}

	return false
}

// checkNoTaboo checks if text contains any taboo phrases
func checkNoTaboo(textLower string, companyProfile *types.CompanyProfile) bool {
	if companyProfile == nil {
		return true // No taboo phrases to check
	}

	for _, taboo := range companyProfile.TabooPhrases {
		tabooLower := strings.ToLower(strings.TrimSpace(taboo))
		if tabooLower == "" {
			continue
		}
		// Check if taboo phrase appears in text (word boundary matching would be better, but simple substring works)
		if strings.Contains(textLower, tabooLower) {
			return false
		}
	}

	return true
}

// checkTargetLength checks if rewritten text length is within tolerance of original
func checkTargetLength(rewrittenLength int, originalLength int) bool {
	if originalLength == 0 {
		return rewrittenLength > 0 // Any length is acceptable if original was empty
	}

	tolerance := float64(originalLength) * lengthTolerancePercent
	minLength := float64(originalLength) - tolerance
	maxLength := float64(originalLength) + tolerance

	// Allow some flexibility - rewritten can be slightly shorter or longer
	return float64(rewrittenLength) >= minLength && float64(rewrittenLength) <= maxLength*1.5
}

// EstimateLines estimates the number of lines for a given text length
func EstimateLines(lengthChars int) int {
	if lengthChars <= 0 {
		return 1 // Minimum 1 line
	}
	return int(math.Ceil(float64(lengthChars) / charsPerLine))
}

// ComputeLengthChars computes the character length of text
func ComputeLengthChars(text string) int {
	return len(text)
}
