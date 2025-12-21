// Package ranking provides functionality to rank experience stories against job profiles.
package ranking

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
	"github.com/jonathan/resume-customizer/internal/types"
)

// LLMScoreResult contains the result of LLM relevance scoring for a single story.
type LLMScoreResult struct {
	Score     float64 // 0.0-1.0 relevance score
	Reasoning string  // Brief explanation from LLM
}

// llmJudgeResponse represents the expected JSON response from the LLM.
type llmJudgeResponse struct {
	RelevanceScore float64 `json:"relevance_score"`
	Reasoning      string  `json:"reasoning"`
}

// JudgeStoryRelevance uses LLM to assess how relevant a story is to a job profile.
// Returns nil result and error if evaluation fails.
func JudgeStoryRelevance(ctx context.Context, story *types.Story, jobProfile *types.JobProfile, client llm.Client) (*LLMScoreResult, error) {
	prompt := buildJudgePrompt(story, jobProfile)

	jsonResp, err := client.GenerateJSON(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	jsonResp = llm.CleanJSONBlock(jsonResp)

	var response llmJudgeResponse
	if err := json.Unmarshal([]byte(jsonResp), &response); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w (content: %s)", err, jsonResp)
	}

	// Validate score is in valid range
	if response.RelevanceScore < 0.0 {
		response.RelevanceScore = 0.0
	}
	if response.RelevanceScore > 1.0 {
		response.RelevanceScore = 1.0
	}

	return &LLMScoreResult{
		Score:     response.RelevanceScore,
		Reasoning: response.Reasoning,
	}, nil
}

// JudgeStoriesRelevance batch-evaluates all stories against a job profile.
// Returns a map of story ID -> LLMScoreResult. Stories that fail evaluation
// will have nil results in the map (caller should handle fallback).
func JudgeStoriesRelevance(ctx context.Context, stories []types.Story, jobProfile *types.JobProfile, client llm.Client) map[string]*LLMScoreResult {
	results := make(map[string]*LLMScoreResult)

	for i := range stories {
		story := &stories[i]

		// Check context cancellation before each evaluation
		select {
		case <-ctx.Done():
			// Context cancelled, mark remaining as nil
			return results
		default:
		}

		result, err := JudgeStoryRelevance(ctx, story, jobProfile, client)
		if err != nil {
			// Log error but continue with other stories
			// Caller will handle nil result as fallback to heuristic
			results[story.ID] = nil
			continue
		}
		results[story.ID] = result
	}

	return results
}

// buildJudgePrompt constructs the prompt for LLM story evaluation.
func buildJudgePrompt(story *types.Story, jobProfile *types.JobProfile) string {
	// Format story bullets
	var bulletLines []string
	for _, bullet := range story.Bullets {
		bulletLine := fmt.Sprintf("  - %s", bullet.Text)
		if len(bullet.Skills) > 0 {
			bulletLine += fmt.Sprintf(" [Skills: %s]", strings.Join(bullet.Skills, ", "))
		}
		bulletLines = append(bulletLines, bulletLine)
	}
	storyBullets := strings.Join(bulletLines, "\n")

	// Format requirements
	var requirements []string
	for _, req := range jobProfile.HardRequirements {
		requirements = append(requirements, req.Skill)
	}
	for _, req := range jobProfile.NiceToHaves {
		requirements = append(requirements, req.Skill+" (nice to have)")
	}
	requirementsStr := strings.Join(requirements, ", ")
	if requirementsStr == "" {
		requirementsStr = "Not specified"
	}

	// Format keywords
	keywordsStr := strings.Join(jobProfile.Keywords, ", ")
	if keywordsStr == "" {
		keywordsStr = "Not specified"
	}

	// Get company name, fallback to empty string
	company := jobProfile.Company
	if company == "" {
		company = "Not specified"
	}

	// Get role title
	roleTitle := jobProfile.RoleTitle
	if roleTitle == "" {
		roleTitle = "Not specified"
	}

	template := prompts.MustGet("ranking.json", "judge-story-relevance")
	return prompts.Format(template, map[string]string{
		"Company":      company,
		"RoleTitle":    roleTitle,
		"Requirements": requirementsStr,
		"Keywords":     keywordsStr,
		"StoryCompany": story.Company,
		"StoryRole":    story.Role,
		"StoryBullets": storyBullets,
	})
}
