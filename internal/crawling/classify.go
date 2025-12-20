package crawling

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const (
	// DefaultClassificationModel is the Gemini model to use for link classification
	DefaultClassificationModel = "gemini-1.5-flash"
	// DefaultClassificationTemperature is the temperature setting for classification
	DefaultClassificationTemperature = 0.1
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

	// Initialize Gemini client
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, &ClassificationError{
			Message: "failed to create Gemini client",
			Cause:   err,
		}
	}
	defer func() { _ = client.Close() }()

	model := client.GenerativeModel(DefaultClassificationModel)
	model.SetTemperature(DefaultClassificationTemperature)

	// Construct classification prompt
	prompt := buildClassificationPrompt(links)

	// Call Gemini API
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, &ClassificationError{
			Message: "failed to generate content from Gemini API",
			Cause:   err,
		}
	}

	// Extract text from response
	responseText, err := extractTextFromResponse(resp)
	if err != nil {
		return nil, &ClassificationError{
			Message: "failed to extract text from API response",
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
	return fmt.Sprintf(`Classify the following URLs into categories. Return a JSON array of objects with "url" and "category" fields.

Categories:
- values: Values, principles, mission statements
- careers: Careers, culture, team, jobs pages
- press: Press, news, blog, articles, media
- product: Product, features, solutions pages
- about: About, company information
- other: Anything that doesn't fit the above categories

URLs to classify:
%s

Return ONLY valid JSON array, no markdown, no explanation. Example format:
[
  {"url": "https://example.com/about", "category": "about"},
  {"url": "https://example.com/careers", "category": "careers"}
]`, linksList)
}

// extractTextFromResponse extracts text content from Gemini API response
func extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil {
		return "", fmt.Errorf("no content in candidate")
	}

	var textParts []string
	for _, part := range candidate.Content.Parts {
		if text, ok := part.(genai.Text); ok {
			textParts = append(textParts, string(text))
		}
	}

	if len(textParts) == 0 {
		return "", fmt.Errorf("no text parts in response")
	}

	return strings.Join(textParts, ""), nil
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
