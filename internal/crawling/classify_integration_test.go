//go:build !short

package crawling

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyLinks_RealAPI(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	links := []string{
		"https://example.com/about",
		"https://example.com/careers",
		"https://example.com/values",
		"https://example.com/blog",
		"https://example.com/product",
	}

	classified, err := ClassifyLinks(context.Background(), links, apiKey)
	require.NoError(t, err)
	require.Len(t, classified, len(links))

	// Verify all links are classified
	classifiedMap := make(map[string]string)
	for _, cl := range classified {
		classifiedMap[cl.URL] = cl.Category
		assert.NotEmpty(t, cl.Category)
		assert.Contains(t, []string{"values", "careers", "press", "product", "about", "other"}, cl.Category)
	}

	// Verify all original links are present
	for _, link := range links {
		assert.Contains(t, classifiedMap, link, "link %s should be classified", link)
	}
}

