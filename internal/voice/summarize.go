// Package voice provides functionality to extract brand voice and style rules from company corpus text.
package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
	"github.com/jonathan/resume-customizer/internal/types"
)

// SummarizeOptions adds database support for caching
type SummarizeOptions struct {
	Database  *db.DB
	CompanyID *uuid.UUID
	MaxAge    time.Duration // How old cached profiles can be
}

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

// SummarizeVoiceWithCache attempts to use cached profile first, falling back to LLM generation
func SummarizeVoiceWithCache(ctx context.Context, opts SummarizeOptions, corpusText string, sources []types.Source, apiKey string) (*types.CompanyProfile, error) {
	// Try to get fresh cached profile
	if opts.Database != nil && opts.CompanyID != nil {
		maxAge := opts.MaxAge
		if maxAge == 0 {
			maxAge = db.DefaultProfileCacheTTL
		}

		cached, err := opts.Database.GetFreshCompanyProfile(ctx, *opts.CompanyID, maxAge)
		if err == nil && cached != nil {
			// Convert db.CompanyProfile to types.CompanyProfile
			return &types.CompanyProfile{
				Tone:          cached.Tone,
				DomainContext: derefStr(cached.DomainContext),
				StyleRules:    cached.StyleRules,
				TabooPhrases:  cached.TabooPhrases,
				Values:        cached.Values,
				EvidenceURLs:  cached.EvidenceURLs,
			}, nil
		}
	}

	// Generate fresh profile
	profile, err := SummarizeVoice(ctx, corpusText, sources, apiKey)
	if err != nil {
		return nil, err
	}

	// Store in database if connected
	if opts.Database != nil && opts.CompanyID != nil {
		input := &db.ProfileCreateInput{
			CompanyID:     *opts.CompanyID,
			Tone:          profile.Tone,
			DomainContext: profile.DomainContext,
			SourceCorpus:  corpusText,
			StyleRules:    profile.StyleRules,
			Values:        profile.Values,
		}

		// Convert taboo phrases
		for _, phrase := range profile.TabooPhrases {
			input.TabooPhrases = append(input.TabooPhrases, db.TabooPhraseInput{
				Phrase: phrase,
			})
		}

		// Convert evidence URLs
		for _, url := range profile.EvidenceURLs {
			input.EvidenceURLs = append(input.EvidenceURLs, db.ProfileSourceInput{
				URL: url,
			})
		}

		_, _ = opts.Database.CreateCompanyProfile(ctx, input)
	}

	return profile, nil
}

// derefStr returns the value of a string pointer, or empty string if nil
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// buildExtractionPrompt constructs the prompt for structured voice extraction
func buildExtractionPrompt(corpusText string, sourceURLs []string) string {
	// Build sources section if URLs are provided
	var sourcesSection string
	if len(sourceURLs) > 0 {
		var sb strings.Builder
		sb.WriteString("Sources (for context):\n")
		for _, url := range sourceURLs {
			sb.WriteString(fmt.Sprintf("- %s\n", url))
		}
		sb.WriteString("\n")
		sourcesSection = sb.String()
	}

	template := prompts.MustGet("voice.json", "extract-brand-voice")
	return prompts.Format(template, map[string]string{
		"Sources":    sourcesSection,
		"CorpusText": corpusText,
	})
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
