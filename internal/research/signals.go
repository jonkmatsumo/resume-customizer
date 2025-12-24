// Package research - signals.go provides brand signal extraction from web pages.
package research

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
	"github.com/jonathan/resume-customizer/internal/validation"
)

// ExtractBrandSignals extracts brand-relevant information from page text
func ExtractBrandSignals(ctx context.Context, pageText string, url string, apiKey string) (*BrandSignal, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key required for signal extraction")
	}

	if len(pageText) < 100 {
		// Not enough content to extract signals
		return nil, nil
	}

	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}
	defer func() { _ = client.Close() }()

	prompt := buildSignalPrompt(pageText, url)

	jsonResp, err := client.GenerateJSON(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	jsonResp = llm.CleanJSONBlock(jsonResp)

	var signal BrandSignal
	if err := json.Unmarshal([]byte(jsonResp), &signal); err != nil {
		return nil, fmt.Errorf("failed to parse signal response: %w", err)
	}

	signal.URL = url
	return &signal, nil
}

func buildSignalPrompt(pageText string, url string) string {
	// Truncate if too long
	if len(pageText) > 8000 {
		pageText = pageText[:8000] + "..."
	}

	// Check for potential injection attempts and log warning (but continue processing)
	checkResult := validation.CheckBasicHeuristics(pageText)
	validation.LogInjectionWarning(checkResult, "brand signal page: "+url)

	// Wrap content in quote markers to signal non-executable content
	quotedContent := validation.QuoteExternalContentWithLabel(pageText, "WEB PAGE CONTENT")

	template := prompts.MustGet("research.json", "extract-brand-signals")
	return prompts.Format(template, map[string]string{
		"URL":         url,
		"PageContent": quotedContent,
	})
}

// AggregateSignals combines brand signals into a corpus
func AggregateSignals(signals []BrandSignal) string {
	if len(signals) == 0 {
		return ""
	}

	var result string
	for _, signal := range signals {
		if len(signal.KeyPoints) > 0 {
			result += fmt.Sprintf("Source: %s (type: %s)\n", signal.URL, signal.Type)
			for _, point := range signal.KeyPoints {
				result += "- " + point + "\n"
			}
			result += "\n"
		}
	}
	return result
}
