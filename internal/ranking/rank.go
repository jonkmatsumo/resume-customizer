// Package ranking provides functionality to rank experience stories against job profiles.
package ranking

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/skills"
	"github.com/jonathan/resume-customizer/internal/types"
)

// Weights for hybrid scoring when LLM is available
const (
	heuristicWeight = 0.5
	llmWeight       = 0.5
)

// RankStories ranks experience stories against a job profile using heuristic scoring only.
// This is the original deterministic ranking function maintained for backward compatibility.
func RankStories(jobProfile *types.JobProfile, experienceBank *types.ExperienceBank) (*types.RankedStories, error) {
	return rankStoriesHeuristic(jobProfile, experienceBank)
}

// RankStoriesWithLLM ranks experience stories using hybrid heuristic + LLM scoring.
// If apiKey is empty or LLM evaluation fails, falls back to heuristic-only scoring.
// The final score is 50% heuristic + 50% LLM when LLM is available.
func RankStoriesWithLLM(ctx context.Context, jobProfile *types.JobProfile, experienceBank *types.ExperienceBank, apiKey string) (*types.RankedStories, error) {
	// Build skill targets from job profile
	skillTargets, err := skills.BuildSkillTargets(jobProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to build skill targets: %w", err)
	}

	// Compute heuristic scores for all stories first
	rankedStories := make([]types.RankedStory, 0, len(experienceBank.Stories))
	for _, story := range experienceBank.Stories {
		rankedStory := computeHeuristicScore(&story, jobProfile, skillTargets)
		rankedStories = append(rankedStories, rankedStory)
	}

	// Attempt LLM scoring if API key is provided
	var llmScores map[string]*LLMScoreResult
	if apiKey != "" {
		config := llm.DefaultConfig()
		client, err := llm.NewClient(ctx, config, apiKey)
		if err == nil {
			defer func() { _ = client.Close() }()
			llmScores = JudgeStoriesRelevance(ctx, experienceBank.Stories, jobProfile, client)
		}
		// If client creation fails, llmScores remains nil and we use heuristic only
	}

	// Apply hybrid scoring
	for i := range rankedStories {
		story := &rankedStories[i]
		heuristicScore := story.HeuristicScore

		if llmResult, ok := llmScores[story.StoryID]; ok && llmResult != nil {
			// Hybrid scoring: 50% heuristic + 50% LLM
			llmScore := llmResult.Score
			story.LLMScore = &llmScore
			story.LLMReasoning = llmResult.Reasoning
			story.RelevanceScore = (heuristicWeight * heuristicScore) + (llmWeight * llmScore)
		} else {
			// Fallback to heuristic-only scoring
			story.RelevanceScore = heuristicScore
		}

		// Ensure score is in valid range
		if story.RelevanceScore > 1.0 {
			story.RelevanceScore = 1.0
		}
		if story.RelevanceScore < 0.0 {
			story.RelevanceScore = 0.0
		}
	}

	// Sort by relevance score (descending)
	sort.Slice(rankedStories, func(i, j int) bool {
		return rankedStories[i].RelevanceScore > rankedStories[j].RelevanceScore
	})

	return &types.RankedStories{Ranked: rankedStories}, nil
}

// rankStoriesHeuristic performs heuristic-only ranking (internal implementation).
func rankStoriesHeuristic(jobProfile *types.JobProfile, experienceBank *types.ExperienceBank) (*types.RankedStories, error) {
	// Build skill targets from job profile
	skillTargets, err := skills.BuildSkillTargets(jobProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to build skill targets: %w", err)
	}

	// Score each story
	rankedStories := make([]types.RankedStory, 0, len(experienceBank.Stories))
	for _, story := range experienceBank.Stories {
		rankedStory := computeHeuristicScore(&story, jobProfile, skillTargets)
		rankedStory.RelevanceScore = rankedStory.HeuristicScore
		rankedStories = append(rankedStories, rankedStory)
	}

	// Sort by relevance score (descending)
	sort.Slice(rankedStories, func(i, j int) bool {
		return rankedStories[i].RelevanceScore > rankedStories[j].RelevanceScore
	})

	return &types.RankedStories{Ranked: rankedStories}, nil
}

// computeHeuristicScore calculates the heuristic score for a single story.
func computeHeuristicScore(story *types.Story, jobProfile *types.JobProfile, skillTargets *types.SkillTargets) types.RankedStory {
	skillOverlap, matchedSkills := computeSkillOverlapScore(story, skillTargets)
	keywordOverlap := computeKeywordOverlapScore(story, jobProfile)
	evidenceStrength := computeEvidenceStrengthScore(story)
	recency := computeRecencyScore(story)

	// Calculate weighted heuristic score
	heuristicScore := (skillOverlapWeight * skillOverlap) +
		(keywordOverlapWeight * keywordOverlap) +
		(evidenceStrengthWeight * evidenceStrength) +
		(recencyWeight * recency)

	// Ensure score is in valid range
	if heuristicScore > 1.0 {
		heuristicScore = 1.0
	}
	if heuristicScore < 0.0 {
		heuristicScore = 0.0
	}

	return types.RankedStory{
		StoryID:          story.ID,
		HeuristicScore:   heuristicScore,
		SkillOverlap:     skillOverlap,
		KeywordOverlap:   keywordOverlap,
		EvidenceStrength: evidenceStrength,
		MatchedSkills:    matchedSkills,
		Notes:            generateNotes(skillOverlap, keywordOverlap, evidenceStrength, matchedSkills),
	}
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
