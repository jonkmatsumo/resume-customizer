// Package parsing provides functionality to parse job postings into structured JobProfile JSON using LLM extraction.
package parsing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
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
	return fmt.Sprintf(`Extract structured information from the following job posting. Return ONLY valid JSON matching this exact structure:

{
  "company": "string (company name, best-effort)",
  "role_title": "string (job title)",
  "responsibilities": ["string (list of responsibilities)"],
  "hard_requirements": [
    {
      "skill": "string (skill name)",
      "level": "string (e.g., '3+ years', optional)",
      "evidence": "string (exact quote from job posting)"
    }
  ],
  "nice_to_haves": [
    {
      "skill": "string (skill name)",
      "level": "string (optional)",
      "evidence": "string (exact quote from job posting)"
    }
  ],
  "keywords": ["string (domain-specific terms)"],
  "eval_signals": {
    "latency": boolean,
    "reliability": boolean,
    "ownership": boolean,
    "scale": boolean,
    "collaboration": boolean
  }
}

IMPORTANT:
- Include exact quotes from the job posting as evidence snippets
- Set eval_signals based on what the posting emphasizes (e.g., latency if performance mentioned, ownership if autonomy/ownership mentioned)
- Extract all mentioned skills, even if implicit
- Return ONLY the JSON object, no markdown, no explanation, no code blocks

Job posting:
%s`, jobText)
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
