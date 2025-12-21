// Package parsing provides functionality to parse job postings into structured JobProfile JSON using LLM extraction.
package parsing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
	"github.com/jonathan/resume-customizer/internal/types"
)

// ParseJobProfile extracts a structured JobProfile from cleaned job posting text
func ParseJobProfile(ctx context.Context, cleanedText string, apiKey string) (*types.JobProfile, error) {
	if apiKey == "" {
		return nil, &APICallError{Message: "API key is required"}
	}

	// Initialize LLM client with default config
	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, &APICallError{
			Message: "failed to create LLM client",
			Cause:   err,
		}
	}
	defer func() { _ = client.Close() }()

	// Construct extraction prompt
	prompt := buildExtractionPrompt(cleanedText)

	// Use TierAdvanced for structured job parsing (requires reasoning)
	responseText, err := client.GenerateContent(ctx, prompt, llm.TierAdvanced)
	if err != nil {
		return nil, &APICallError{
			Message: "failed to generate content from LLM",
			Cause:   err,
		}
	}

	// Clean markdown code blocks if present
	responseText = cleanJSONBlock(responseText)

	// Parse JSON response
	profile, err := parseJSONResponse(responseText)
	if err != nil {
		return nil, err
	}

	// Post-process the profile
	if err := postProcessProfile(profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// buildExtractionPrompt constructs the prompt for structured extraction
func buildExtractionPrompt(jobText string) string {
	template := prompts.MustGet("parsing.json", "extract-job-profile")
	return prompts.Format(template, map[string]string{
		"JobText": jobText,
	})
}

// cleanJSONBlock removes markdown code block wrappers from JSON
func cleanJSONBlock(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}
	return text
}

// parseJSONResponse parses the JSON response into a JobProfile
func parseJSONResponse(jsonText string) (*types.JobProfile, error) {
	var profile types.JobProfile
	if err := json.Unmarshal([]byte(jsonText), &profile); err != nil {
		return nil, &ParseError{
			Message: "failed to parse JSON response",
			Cause:   err,
		}
	}

	return &profile, nil
}

// postProcessProfile applies normalization and validation
func postProcessProfile(profile *types.JobProfile) error {
	// Normalize skill names in hard_requirements
	profile.HardRequirements = NormalizeRequirements(profile.HardRequirements)

	// Normalize skill names in nice_to_haves
	profile.NiceToHaves = NormalizeRequirements(profile.NiceToHaves)

	// Validate evidence snippets
	for i, req := range profile.HardRequirements {
		if strings.TrimSpace(req.Evidence) == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("hard_requirements[%d].evidence", i),
				Message: "evidence snippet is required",
			}
		}
	}

	for i, req := range profile.NiceToHaves {
		if strings.TrimSpace(req.Evidence) == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("nice_to_haves[%d].evidence", i),
				Message: "evidence snippet is required",
			}
		}
	}

	// Normalize keywords (lowercase, trim)
	normalizedKeywords := make([]string, 0, len(profile.Keywords))
	seenKeywords := make(map[string]bool)
	for _, keyword := range profile.Keywords {
		normalized := strings.ToLower(strings.TrimSpace(keyword))
		if normalized != "" && !seenKeywords[normalized] {
			normalizedKeywords = append(normalizedKeywords, normalized)
			seenKeywords[normalized] = true
		}
	}
	profile.Keywords = normalizedKeywords

	// Ensure eval_signals is initialized
	if profile.EvalSignals == nil {
		profile.EvalSignals = &types.EvalSignals{}
	}

	return nil
}

// ExtractEducationRequirements extracts education requirements from job posting text.
// This is called separately from ParseJobProfile to allow for graceful degradation.
func ExtractEducationRequirements(ctx context.Context, jobText string, apiKey string) (*types.EducationRequirements, error) {
	if apiKey == "" {
		return nil, &APICallError{Message: "API key is required"}
	}

	// Initialize LLM client
	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, &APICallError{
			Message: "failed to create LLM client",
			Cause:   err,
		}
	}
	defer func() { _ = client.Close() }()

	// Build prompt
	template := prompts.MustGet("parsing.json", "extract-education-requirements")
	prompt := prompts.Format(template, map[string]string{
		"JobText": jobText,
	})

	// Use TierLite for simple extraction
	responseText, err := client.GenerateContent(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, &APICallError{
			Message: "failed to extract education requirements",
			Cause:   err,
		}
	}

	// Clean and parse response
	responseText = cleanJSONBlock(responseText)

	var eduReq types.EducationRequirements
	if err := json.Unmarshal([]byte(responseText), &eduReq); err != nil {
		return nil, &ParseError{
			Message: "failed to parse education requirements JSON",
			Cause:   err,
		}
	}

	// Normalize degree level
	eduReq.MinDegree = normalizeDegreeLevel(eduReq.MinDegree)

	return &eduReq, nil
}

// normalizeDegreeLevel normalizes degree level strings to standard values
func normalizeDegreeLevel(degree string) string {
	degree = strings.ToLower(strings.TrimSpace(degree))

	switch {
	case strings.Contains(degree, "phd") || strings.Contains(degree, "doctor"):
		return "phd"
	case strings.Contains(degree, "master"):
		return "master"
	case strings.Contains(degree, "bachelor"):
		return "bachelor"
	case strings.Contains(degree, "associate"):
		return "associate"
	default:
		return degree
	}
}
