// Package research - signals.go provides brand signal extraction from web pages.
package research

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jonathan/resume-customizer/internal/llm"
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

	return fmt.Sprintf(`Extract brand voice signals from this company page.

URL: %s

INSTRUCTIONS:
1. COPY KEY POINTS VERBATIM - do not paraphrase
2. Focus on statements about culture, values, principles, working style
3. Include memorable quotes that reveal company character
4. Note any repeated themes or emphasized points

Return ONLY valid JSON:
{
  "type": "values|culture|engineering|press|about|other",
  "key_points": ["exact quote 1", "exact quote 2", ...],
  "values": ["inferred value 1", "inferred value 2", ...]
}

Page content:
"""
%s
"""`, url, pageText)
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
