// Package voice provides functionality to extract brand voice and style rules from company corpus text.
package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/types"
)

// SummarizeVoice extracts brand voice and style rules from company corpus text
func SummarizeVoice(ctx context.Context, corpusText string, sources []types.Source, apiKey string) (*types.CompanyProfile, error) {
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

	// Extract source URLs for context in prompt
	sourceURLs := make([]string, len(sources))
	for i, source := range sources {
		sourceURLs[i] = source.URL
	}

	// Construct extraction prompt
	prompt := buildExtractionPrompt(corpusText, sourceURLs)

	// Use TierAdvanced for voice analysis (requires nuance and understanding)
	responseText, err := client.GenerateContent(ctx, prompt, llm.TierAdvanced)
	if err != nil {
		return nil, &APICallError{
			Message: "failed to generate content from LLM",
			Cause:   err,
		}
	}

	// Clean markdown code blocks if present using shared utility
	responseText = llm.CleanJSONBlock(responseText)

	// Parse JSON response
	profile, err := parseJSONResponse(responseText)
	if err != nil {
		return nil, err
	}

	// Post-process the profile (validate and populate evidence URLs)
	if err := postProcessProfile(profile, sources); err != nil {
		return nil, err
	}

	return profile, nil
}

// buildExtractionPrompt constructs the prompt for structured voice extraction
func buildExtractionPrompt(corpusText string, sourceURLs []string) string {
	var sb strings.Builder
	sb.WriteString("Extract brand voice and style rules from the following company corpus text. Return ONLY valid JSON matching this exact structure:\n\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"company\": \"string (company name)\",\n")
	sb.WriteString("  \"tone\": \"string (brand tone, e.g., 'direct, metric-driven', 'collaborative, values-driven')\",\n")
	sb.WriteString("  \"style_rules\": [\"string (actionable style guidelines, e.g., 'lead with metrics', 'avoid hype', 'use active voice')\"],\n")
	sb.WriteString("  \"taboo_phrases\": [\"string (words/phrases to avoid)\"],\n")
	sb.WriteString("  \"domain_context\": \"string (domain/industry context, e.g., 'B2B SaaS, infrastructure')\",\n")
	sb.WriteString("  \"values\": [\"string (core company values)\"]\n")
	sb.WriteString("}\n\n")
	sb.WriteString("IMPORTANT:\n")
	sb.WriteString("- Style rules must be actionable and specific (e.g., 'lead with quantified impact', 'avoid marketing jargon')\n")
	sb.WriteString("- Extract values directly from the corpus text\n")
	sb.WriteString("- Taboo phrases should include words/phrases the company explicitly avoids or criticizes\n")
	sb.WriteString("- Tone should capture the overall communication style\n")
	sb.WriteString("- Domain context should summarize the industry/domain\n")
	sb.WriteString("- Return ONLY the JSON object, no markdown, no explanation, no code blocks\n\n")
	if len(sourceURLs) > 0 {
		sb.WriteString("Sources (for context):\n")
		for _, url := range sourceURLs {
			sb.WriteString(fmt.Sprintf("- %s\n", url))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("Company corpus text:\n")
	sb.WriteString(corpusText)

	return sb.String()
}

// parseJSONResponse parses the JSON response into a CompanyProfile
func parseJSONResponse(jsonText string) (*types.CompanyProfile, error) {
	var profile types.CompanyProfile
	if err := json.Unmarshal([]byte(jsonText), &profile); err != nil {
		return nil, &ParseError{
			Message: "failed to parse JSON response",
			Cause:   err,
		}
	}

	return &profile, nil
}

// postProcessProfile validates the profile and populates evidence URLs from sources
func postProcessProfile(profile *types.CompanyProfile, sources []types.Source) error {
	// Validate required fields
	if profile.Company == "" {
		return &ValidationError{
			Field:   "company",
			Message: "company name is required",
		}
	}
	if profile.Tone == "" {
		return &ValidationError{
			Field:   "tone",
			Message: "tone is required",
		}
	}
	if len(profile.StyleRules) == 0 {
		return &ValidationError{
			Field:   "style_rules",
			Message: "at least one style rule is required",
		}
	}
	if len(profile.TabooPhrases) == 0 {
		return &ValidationError{
			Field:   "taboo_phrases",
			Message: "at least one taboo phrase is required",
		}
	}
	if profile.DomainContext == "" {
		return &ValidationError{
			Field:   "domain_context",
			Message: "domain context is required",
		}
	}
	if len(profile.Values) == 0 {
		return &ValidationError{
			Field:   "values",
			Message: "at least one value is required",
		}
	}

	// Populate evidence URLs from sources (not from LLM extraction)
	evidenceURLs := make([]string, len(sources))
	for i, source := range sources {
		evidenceURLs[i] = source.URL
	}
	profile.EvidenceURLs = evidenceURLs

	return nil
}
