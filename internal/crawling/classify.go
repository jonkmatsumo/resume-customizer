package crawling

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
)

// ClassifiedLink represents a link with its classification category
type ClassifiedLink struct {
	URL      string `json:"url"`
	Category string `json:"category"`
}

// ClassifyLinks classifies URLs into categories using LLM
func ClassifyLinks(ctx context.Context, links []string, apiKey string) ([]ClassifiedLink, error) {
	if apiKey == "" {
		return nil, &ClassificationError{Message: "API key is required"}
	}

	if len(links) == 0 {
		return []ClassifiedLink{}, nil
	}

	// Initialize LLM client with default config
	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, &ClassificationError{
			Message: "failed to create LLM client",
			Cause:   err,
		}
	}
	defer func() { _ = client.Close() }()

	// Construct classification prompt
	prompt := buildClassificationPrompt(links)

	// Use TierLite for simple classification task
	responseText, err := client.GenerateContent(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, &ClassificationError{
			Message: "failed to generate content from LLM",
			Cause:   err,
		}
	}

	// Parse JSON response
	classified, err := parseClassificationResponse(responseText, links)
	if err != nil {
		return nil, &ClassificationError{
			Message: "failed to parse classification response",
			Cause:   err,
		}
	}

	return classified, nil
}

// buildClassificationPrompt constructs the prompt for link classification
func buildClassificationPrompt(links []string) string {
	linksList := strings.Join(links, "\n")
	template := prompts.MustGet("crawling.json", "classify-links")
	return prompts.Format(template, map[string]string{
		"Links": linksList,
	})
}

// parseClassificationResponse parses the JSON response from classification
func parseClassificationResponse(responseText string, originalLinks []string) ([]ClassifiedLink, error) {
	// Clean response text (remove markdown code blocks if present)
	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	// Parse JSON
	var classified []ClassifiedLink
	if err := json.Unmarshal([]byte(responseText), &classified); err != nil {
		return nil, fmt.Errorf("failed to unmarshal classification JSON: %w", err)
	}

	// Validate that all original links are classified
	classifiedMap := make(map[string]string)
	for _, cl := range classified {
		classifiedMap[cl.URL] = cl.Category
	}

	// Fill in any missing links with "other" category
	result := make([]ClassifiedLink, 0, len(originalLinks))
	for _, link := range originalLinks {
		category, exists := classifiedMap[link]
		if !exists {
			category = "other"
		}
		result = append(result, ClassifiedLink{
			URL:      link,
			Category: category,
		})
	}

	return result, nil
}
