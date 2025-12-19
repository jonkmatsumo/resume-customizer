// Package ranking provides functionality to rank experience stories against job profiles.
package ranking

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jonathan/resume-customizer/internal/skills"
	"github.com/jonathan/resume-customizer/internal/types"
)

// RankStories ranks experience stories against a job profile, returning sorted stories by relevance score.
func RankStories(jobProfile *types.JobProfile, experienceBank *types.ExperienceBank) (*types.RankedStories, error) {
	// Build skill targets from job profile
	skillTargets, err := skills.BuildSkillTargets(jobProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to build skill targets: %w", err)
	}

	// Score each story
	rankedStories := make([]types.RankedStory, 0, len(experienceBank.Stories))
	for _, story := range experienceBank.Stories {
		skillOverlap, matchedSkills := computeSkillOverlapScore(&story, skillTargets)
		keywordOverlap := computeKeywordOverlapScore(&story, jobProfile)
		evidenceStrength := computeEvidenceStrengthScore(&story)
		recency := computeRecencyScore(&story)

		// Calculate weighted relevance score
		relevanceScore := (skillOverlapWeight * skillOverlap) +
			(keywordOverlapWeight * keywordOverlap) +
			(evidenceStrengthWeight * evidenceStrength) +
			(recencyWeight * recency)

		// Ensure score is in valid range
		if relevanceScore > 1.0 {
			relevanceScore = 1.0
		}
		if relevanceScore < 0.0 {
			relevanceScore = 0.0
		}

		rankedStory := types.RankedStory{
			StoryID:          story.ID,
			RelevanceScore:   relevanceScore,
			SkillOverlap:     skillOverlap,
			KeywordOverlap:   keywordOverlap,
			EvidenceStrength: evidenceStrength,
			MatchedSkills:    matchedSkills,
			Notes:            generateNotes(skillOverlap, keywordOverlap, evidenceStrength, matchedSkills),
		}

		rankedStories = append(rankedStories, rankedStory)
	}

	// Sort by relevance score (descending)
	sort.Slice(rankedStories, func(i, j int) bool {
		return rankedStories[i].RelevanceScore > rankedStories[j].RelevanceScore
	})

	return &types.RankedStories{Ranked: rankedStories}, nil
}

// generateNotes creates a brief explanation of the ranking.
func generateNotes(skillOverlap, keywordOverlap, evidenceStrength float64, matchedSkills []string) string {
	var parts []string

	// Skill match description
	if len(matchedSkills) > 0 {
		if skillOverlap >= 0.7 {
			parts = append(parts, fmt.Sprintf("Strong skill match (%s)", strings.Join(matchedSkills, ", ")))
		} else if skillOverlap >= 0.4 {
			parts = append(parts, fmt.Sprintf("Moderate skill match (%s)", strings.Join(matchedSkills, ", ")))
		} else if skillOverlap > 0 {
			parts = append(parts, fmt.Sprintf("Weak skill match (%s)", strings.Join(matchedSkills, ", ")))
		} else {
			parts = append(parts, "No skill matches")
		}
	} else {
		parts = append(parts, "No skill matches")
	}

	// Evidence strength description
	if evidenceStrength >= 0.8 {
		parts = append(parts, "High evidence strength")
	} else if evidenceStrength >= 0.5 {
		parts = append(parts, "Medium evidence strength")
	} else {
		parts = append(parts, "Low evidence strength")
	}

	// Keyword match description
	if keywordOverlap >= 0.5 {
		parts = append(parts, "Good keyword overlap")
	} else if keywordOverlap > 0 {
		parts = append(parts, "Some keyword overlap")
	}

	return strings.Join(parts, ". ")
}
