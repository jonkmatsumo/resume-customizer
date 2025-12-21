// Package ranking provides functionality to rank and score experience stories and education.
package ranking

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
	"github.com/jonathan/resume-customizer/internal/types"
)

// EducationScore represents the relevance score for an education entry
type EducationScore struct {
	EducationID  string   `json:"education_id"`
	TotalScore   float64  `json:"total_score"`         // 0-1 combined score
	RuleScore    float64  `json:"rule_score"`          // Score from rule-based matching
	LLMScore     *float64 `json:"llm_score,omitempty"` // Score from LLM (nil if not available)
	LLMReasoning string   `json:"llm_reasoning,omitempty"`
	Included     bool     `json:"included"` // Whether to include in resume
	Reason       string   `json:"reason"`   // Why included/excluded
}

// degreeRank maps degree types to numeric ranks for comparison
var degreeRank = map[string]int{
	"associate": 1,
	"bachelor":  2,
	"master":    3,
	"phd":       4,
}

// ScoreEducation scores each education entry against job requirements.
// Implements fallback strategy:
// - No rules + No LLM → Include all
// - Rules + No LLM → Use only rule-based
// - No rules + LLM → Use only LLM
// - Rules + LLM → Weighted average (50/50)
func ScoreEducation(
	ctx context.Context,
	education []types.Education,
	requirements *types.EducationRequirements,
	fullJobText string,
	apiKey string,
) ([]EducationScore, error) {
	if len(education) == 0 {
		return []EducationScore{}, nil
	}

	scores := make([]EducationScore, len(education))
	hasRules := requirements != nil && (requirements.MinDegree != "" || len(requirements.PreferredFields) > 0)
	hasAPIKey := apiKey != ""

	for i, edu := range education {
		score := EducationScore{
			EducationID: edu.ID,
		}

		// Rule-based scoring
		ruleScore := computeEducationRuleScore(edu, requirements)
		score.RuleScore = ruleScore

		// LLM-based scoring (if API key available)
		var llmScore *float64
		var llmReasoning string
		if hasAPIKey && fullJobText != "" {
			llmResult, err := judgeEducationRelevance(ctx, edu, requirements, fullJobText, apiKey)
			if err == nil && llmResult != nil {
				llmScore = &llmResult.Score
				llmReasoning = llmResult.Reasoning
			}
		}
		score.LLMScore = llmScore
		score.LLMReasoning = llmReasoning

		// Combine scores based on availability
		switch {
		case !hasRules && llmScore == nil:
			// No rules, no LLM → Include all
			score.TotalScore = 1.0
			score.Included = true
			score.Reason = "included (no filtering criteria available)"
		case hasRules && llmScore == nil:
			// Rules only
			score.TotalScore = ruleScore
			score.Included = ruleScore >= 0.3
			score.Reason = "rule-based scoring (LLM unavailable)"
		case !hasRules && llmScore != nil:
			// LLM only
			score.TotalScore = *llmScore
			score.Included = *llmScore >= 0.3
			score.Reason = "LLM-based scoring (no explicit requirements)"
		default:
			// Both available → weighted average
			score.TotalScore = (ruleScore + *llmScore) / 2.0
			score.Included = score.TotalScore >= 0.3
			score.Reason = "hybrid scoring (50% rule + 50% LLM)"
		}

		scores[i] = score
	}

	// Fallback: If no education is included, include the most recent one
	anyIncluded := false
	for _, s := range scores {
		if s.Included {
			anyIncluded = true
			break
		}
	}

	if !anyIncluded && len(scores) > 0 {
		// Find most recent education
		mostRecentIdx := 0
		latestEnd := ""

		for i, edu := range education {
			if edu.EndDate == "present" || (edu.EndDate > latestEnd && latestEnd != "present") {
				latestEnd = edu.EndDate
				mostRecentIdx = i
			}
		}

		scores[mostRecentIdx].Included = true
		scores[mostRecentIdx].Reason = "fallback: included most recent education (none met relevance threshold)"
	}

	return scores, nil
}

// computeEducationRuleScore computes rule-based score for education
func computeEducationRuleScore(edu types.Education, req *types.EducationRequirements) float64 {
	if req == nil {
		return 1.0 // No requirements = full score
	}

	score := 0.0
	weights := 0.0

	// Degree level matching (60% weight)
	if req.MinDegree != "" {
		weights += 0.6
		reqRank := degreeRank[strings.ToLower(req.MinDegree)]
		eduRank := degreeRank[strings.ToLower(edu.Degree)]

		if eduRank >= reqRank {
			// Meets or exceeds requirement
			score += 0.6
		} else if eduRank == reqRank-1 {
			// One level below
			score += 0.3
		}
		// Else: 0 points
	}

	// Field matching (40% weight)
	if len(req.PreferredFields) > 0 {
		weights += 0.4
		fieldScore := computeFieldMatchScore(edu.Field, req.PreferredFields)
		score += 0.4 * fieldScore
	}

	if weights == 0 {
		return 1.0 // No requirements = full score
	}

	return score / weights * 1.0 // Normalize to 0-1
}

// computeFieldMatchScore computes how well the education field matches preferred fields
func computeFieldMatchScore(field string, preferredFields []string) float64 {
	fieldLower := strings.ToLower(field)

	for _, preferred := range preferredFields {
		preferredLower := strings.ToLower(preferred)

		// Exact or substring match
		if fieldLower == preferredLower || strings.Contains(fieldLower, preferredLower) || strings.Contains(preferredLower, fieldLower) {
			return 1.0
		}
	}

	// Check for related fields
	relatedFields := map[string][]string{
		"computer science":       {"software engineering", "computer engineering", "information technology", "cs"},
		"software engineering":   {"computer science", "computer engineering", "cs"},
		"data science":           {"statistics", "mathematics", "computer science", "machine learning"},
		"statistics":             {"mathematics", "data science", "economics"},
		"mathematics":            {"statistics", "physics", "computer science"},
		"electrical engineering": {"computer engineering", "electronics"},
	}

	for _, preferred := range preferredFields {
		preferredLower := strings.ToLower(preferred)
		if related, ok := relatedFields[preferredLower]; ok {
			for _, r := range related {
				if strings.Contains(fieldLower, r) || strings.Contains(r, fieldLower) {
					return 0.7 // Related field
				}
			}
		}
	}

	return 0.2 // Unrelated field
}

// judgeEducationResult holds the LLM response for education relevance
type judgeEducationResult struct {
	Score     float64 `json:"relevance_score"`
	Reasoning string  `json:"reasoning"`
}

// judgeEducationRelevance uses LLM to evaluate education relevance
func judgeEducationRelevance(
	ctx context.Context,
	edu types.Education,
	req *types.EducationRequirements,
	jobSummary string,
	apiKey string,
) (*judgeEducationResult, error) {
	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Close() }()

	// Build requirements string
	reqStr := "None specified"
	if req != nil {
		parts := []string{}
		if req.MinDegree != "" {
			parts = append(parts, "Min degree: "+req.MinDegree)
		}
		if len(req.PreferredFields) > 0 {
			parts = append(parts, "Preferred fields: "+strings.Join(req.PreferredFields, ", "))
		}
		if len(parts) > 0 {
			reqStr = strings.Join(parts, "; ")
		}
	}

	// Build highlights string
	highlightsStr := "None"
	if len(edu.Highlights) > 0 {
		highlightsStr = strings.Join(edu.Highlights, "; ")
	}

	template := prompts.MustGet("ranking.json", "judge-education-relevance")
	prompt := prompts.Format(template, map[string]string{
		"RoleTitle":             "", // Will be filled from job context
		"EducationRequirements": reqStr,
		"JobSummary":            truncateText(jobSummary, 500),
		"School":                edu.School,
		"Degree":                edu.Degree,
		"Field":                 edu.Field,
		"Highlights":            highlightsStr,
	})

	responseText, err := client.GenerateContent(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, err
	}

	responseText = llm.CleanJSONBlock(responseText)

	var result judgeEducationResult
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, err
	}

	// Clamp score to 0-1
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 1 {
		result.Score = 1
	}

	return &result, nil
}

// truncateText truncates text to maxLen characters
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
