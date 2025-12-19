// Package ranking provides functionality to rank experience stories against job profiles.
package ranking

import (
	"strings"
	"time"

	"github.com/jonathan/resume-customizer/internal/parsing"
	"github.com/jonathan/resume-customizer/internal/types"
)

// Default weights for scoring components
const (
	skillOverlapWeight     = 0.5
	keywordOverlapWeight   = 0.2
	evidenceStrengthWeight = 0.2
	recencyWeight          = 0.1
)

// computeSkillOverlapScore calculates the skill overlap score between a story and skill targets.
// Returns the score (0-1) and list of matched skill names.
func computeSkillOverlapScore(story *types.Story, skillTargets *types.SkillTargets) (float64, []string) {
	if len(skillTargets.Skills) == 0 {
		return 0.0, nil
	}

	// Collect all normalized skills from story bullets
	storySkillsSet := make(map[string]bool)
	for _, bullet := range story.Bullets {
		for _, skill := range bullet.Skills {
			normalized := parsing.NormalizeSkillName(skill)
			if normalized != "" {
				storySkillsSet[normalized] = true
			}
		}
	}

	if len(storySkillsSet) == 0 {
		return 0.0, nil
	}

	// Build a map of skill target names to weights for efficient lookup
	targetMap := make(map[string]float64)
	totalWeight := 0.0
	for _, target := range skillTargets.Skills {
		normalizedTarget := parsing.NormalizeSkillName(target.Name)
		targetMap[normalizedTarget] = target.Weight
		totalWeight += target.Weight
	}

	// Find matches and sum weights
	matchedWeight := 0.0
	matchedSkills := make([]string, 0)
	for storySkill := range storySkillsSet {
		if weight, found := targetMap[storySkill]; found {
			matchedWeight += weight
			matchedSkills = append(matchedSkills, storySkill)
		}
	}

	// Normalize by total possible weight
	score := 0.0
	if totalWeight > 0 {
		score = matchedWeight / totalWeight
	}

	return score, matchedSkills
}

// computeKeywordOverlapScore calculates keyword overlap score by matching job keywords against story text.
func computeKeywordOverlapScore(story *types.Story, jobProfile *types.JobProfile) float64 {
	if len(jobProfile.Keywords) == 0 {
		return 0.0
	}

	// Build story text from all bullets
	var storyText strings.Builder
	for _, bullet := range story.Bullets {
		storyText.WriteString(bullet.Text)
		storyText.WriteString(" ")
	}
	storyTextLower := strings.ToLower(storyText.String())

	// Count keyword matches (case-insensitive)
	matches := 0
	for _, keyword := range jobProfile.Keywords {
		keywordLower := strings.ToLower(keyword)
		// Simple substring matching (could be enhanced with word boundary checks)
		if strings.Contains(storyTextLower, keywordLower) {
			matches++
		}
	}

	// Normalize by number of keywords
	score := float64(matches) / float64(len(jobProfile.Keywords))
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// computeEvidenceStrengthScore calculates the average evidence strength score across all bullets.
func computeEvidenceStrengthScore(story *types.Story) float64 {
	if len(story.Bullets) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, bullet := range story.Bullets {
		strength := strings.ToLower(bullet.EvidenceStrength)
		switch strength {
		case "high":
			totalScore += 1.0
		case "medium":
			totalScore += 0.6
		case "low":
			totalScore += 0.3
		default:
			// Unknown strength defaults to medium
			totalScore += 0.6
		}
	}

	return totalScore / float64(len(story.Bullets))
}

// computeRecencyScore calculates a recency score based on story start date.
// Returns 0.5 as default if date parsing fails (neutral score).
func computeRecencyScore(story *types.Story) float64 {
	if story.StartDate == "" {
		return 0.5 // Neutral score if no date
	}

	// Parse date in format "YYYY-MM"
	parts := strings.Split(story.StartDate, "-")
	if len(parts) != 2 {
		return 0.5
	}

	date, err := time.Parse("2006-01", story.StartDate)
	if err != nil {
		return 0.5
	}

	// Calculate years since start date
	now := time.Now()
	yearsSince := now.Sub(date).Hours() / (24 * 365.25)

	// Score: more recent = higher score
	// Linear decay: 0 years = 1.0, 10 years = 0.0
	maxYears := 10.0
	if yearsSince < 0 {
		return 1.0 // Future dates get max score
	}
	if yearsSince >= maxYears {
		return 0.0
	}

	score := 1.0 - (yearsSince / maxYears)
	if score < 0 {
		score = 0
	}
	if score > 1.0 {
		score = 1.0
	}

	return score
}
