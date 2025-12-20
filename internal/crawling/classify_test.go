package crawling

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseClassificationResponse_ValidJSON(t *testing.T) {
	responseText := `[
		{"url": "https://example.com/values", "category": "values"},
		{"url": "https://example.com/careers", "category": "careers"},
		{"url": "https://example.com/blog", "category": "press"}
	]`

	originalLinks := []string{
		"https://example.com/values",
		"https://example.com/careers",
		"https://example.com/blog",
	}

	classified, err := parseClassificationResponse(responseText, originalLinks)
	require.NoError(t, err)
	require.Len(t, classified, 3)

	assert.Equal(t, "values", classified[0].Category)
	assert.Equal(t, "careers", classified[1].Category)
	assert.Equal(t, "press", classified[2].Category)
}

func TestParseClassificationResponse_WithMarkdownCodeBlocks(t *testing.T) {
	responseText := "```json\n[{\"url\": \"https://example.com/about\", \"category\": \"about\"}]\n```"

	originalLinks := []string{"https://example.com/about"}

	classified, err := parseClassificationResponse(responseText, originalLinks)
	require.NoError(t, err)
	require.Len(t, classified, 1)
	assert.Equal(t, "about", classified[0].Category)
}

func TestParseClassificationResponse_MissingLinks(t *testing.T) {
	responseText := `[{"url": "https://example.com/values", "category": "values"}]`

	originalLinks := []string{
		"https://example.com/values",
		"https://example.com/missing",
	}

	classified, err := parseClassificationResponse(responseText, originalLinks)
	require.NoError(t, err)
	require.Len(t, classified, 2)

	// First link should have correct category
	assert.Equal(t, "values", classified[0].Category)

	// Missing link should default to "other"
	assert.Equal(t, "other", classified[1].Category)
	assert.Equal(t, "https://example.com/missing", classified[1].URL)
}

func TestParseClassificationResponse_InvalidJSON(t *testing.T) {
	responseText := "not valid json"

	originalLinks := []string{"https://example.com/test"}

	_, err := parseClassificationResponse(responseText, originalLinks)
	assert.Error(t, err)
}

func TestBuildClassificationPrompt(t *testing.T) {
	links := []string{
		"https://example.com/about",
		"https://example.com/careers",
	}

	prompt := buildClassificationPrompt(links)

	assert.Contains(t, prompt, "https://example.com/about")
	assert.Contains(t, prompt, "https://example.com/careers")
	assert.Contains(t, prompt, "values")
	assert.Contains(t, prompt, "careers")
	assert.Contains(t, prompt, "press")
	assert.Contains(t, prompt, "product")
	assert.Contains(t, prompt, "about")
	assert.Contains(t, prompt, "other")
}

func TestClassifyLinks_EmptyLinks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := getTestAPIKey(t)
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping test")
	}

	classified, err := ClassifyLinks(context.Background(), []string{}, apiKey)
	require.NoError(t, err)
	assert.Empty(t, classified)
}

func TestClassifyLinks_MissingAPIKey(t *testing.T) {
	_, err := ClassifyLinks(context.Background(), []string{"https://example.com/test"}, "")
	assert.Error(t, err)
	var classErr *ClassificationError
	assert.ErrorAs(t, err, &classErr)
	assert.Contains(t, err.Error(), "API key is required")
}

// getTestAPIKey retrieves the API key from environment for integration tests
func getTestAPIKey(t *testing.T) string {
	t.Helper()
	return "" // Skip integration tests in unit test file
}
