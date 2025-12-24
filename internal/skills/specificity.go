// Package skills provides functionality to build weighted skill targets from job profiles.
package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
)

// SkillSpecificityResult represents the LLM's specificity rating for a skill
type SkillSpecificityResult struct {
	SkillName   string  `json:"skill_name"`
	Specificity float64 `json:"specificity"`
	Reasoning   string  `json:"reasoning,omitempty"`
}

// JudgeSkillSpecificity uses LLM to assess how specific/concrete each skill is.
// Returns a map of skill name to specificity score (0.0 = very generic, 1.0 = highly specific).
func JudgeSkillSpecificity(ctx context.Context, skillNames []string, client llm.Client) (map[string]float64, error) {
	if len(skillNames) == 0 {
		return make(map[string]float64), nil
	}

	// Build the prompt
	prompt := buildSpecificityPrompt(skillNames)

	// Call LLM
	response, err := client.GenerateJSON(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	results, err := parseSpecificityResponse(response, skillNames)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return results, nil
}

// buildSpecificityPrompt creates the prompt for specificity judgment
func buildSpecificityPrompt(skillNames []string) string {
	systemPrompt, _ := prompts.Get("ranking.json", "skill_specificity_system")
	if systemPrompt == "" {
		systemPrompt = `You are an expert at analyzing job skills.
Rate each skill's specificity from 0.0 (very generic/soft skill) to 1.0 (highly specific/technical).

Specificity guidelines:
- 1.0: Concrete technology or tool (Kubernetes, Go, PostgreSQL, React)
- 0.8: Specific methodology or domain (microservices, CI/CD, machine learning)
- 0.5: Technical but broad (distributed systems, backend development, cloud infrastructure)
- 0.3: Industry-specific but non-technical (product development, agile, project management)
- 0.0: Generic soft skills (communication, teamwork, leadership, problem solving)

Respond in JSON format only.`
	}

	skillListJSON, _ := json.Marshal(skillNames)

	userPrompt := fmt.Sprintf(`Rate the specificity of these skills:
%s

Respond with a JSON array:
[{"skill_name": "...", "specificity": 0.0-1.0, "reasoning": "..."}]`, string(skillListJSON))

	return systemPrompt + "\n\n" + userPrompt
}

// parseSpecificityResponse parses the LLM response into a map
func parseSpecificityResponse(response string, skillNames []string) (map[string]float64, error) {
	// Clean response - find JSON array
	response = strings.TrimSpace(response)
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return nil, fmt.Errorf("no valid JSON array found in response")
	}
	jsonStr := response[startIdx : endIdx+1]

	var results []SkillSpecificityResult
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Build result map
	resultMap := make(map[string]float64)
	for _, r := range results {
		// Clamp to valid range
		specificity := r.Specificity
		if specificity < 0 {
			specificity = 0
		}
		if specificity > 1 {
			specificity = 1
		}
		resultMap[strings.ToLower(strings.TrimSpace(r.SkillName))] = specificity
	}

	// Ensure all requested skills have a score (default to 0.5 if missing)
	for _, name := range skillNames {
		normalized := strings.ToLower(strings.TrimSpace(name))
		if _, exists := resultMap[normalized]; !exists {
			resultMap[normalized] = 0.5 // Default mid-range
		}
	}

	return resultMap, nil
}
