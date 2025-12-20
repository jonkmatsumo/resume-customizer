package crawling

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectPages_PrioritizesCategories(t *testing.T) {
	classified := []ClassifiedLink{
		{URL: "https://example.com/values", Category: "values"},
		{URL: "https://example.com/careers", Category: "careers"},
		{URL: "https://example.com/press", Category: "press"},
		{URL: "https://example.com/product", Category: "product"},
		{URL: "https://example.com/other", Category: "other"},
	}

	selected := selectPages(classified, 10, "https://example.com/home")
	require.Len(t, selected, 5) // All should be selected

	// Verify priority categories are included
	categories := make(map[string]bool)
	for _, cl := range classified {
		if contains(selected, cl.URL) {
			categories[cl.Category] = true
		}
	}

	assert.True(t, categories["values"], "values should be selected")
	assert.True(t, categories["careers"], "careers should be selected")
	assert.True(t, categories["press"], "press should be selected")
}

func TestSelectPages_RespectsMaxPages(t *testing.T) {
	classified := make([]ClassifiedLink, 20)
	for i := 0; i < 20; i++ {
		classified[i] = ClassifiedLink{
			URL:      fmt.Sprintf("https://example.com/page%d", i),
			Category: "other",
		}
	}

	selected := selectPages(classified, 5, "https://example.com/home")
	// Should respect maxPages-1 (homepage is always included separately)
	assert.LessOrEqual(t, len(selected), 4)
}

func TestSelectPages_SkipsHomepage(t *testing.T) {
	homepageURL := "https://example.com/home"
	classified := []ClassifiedLink{
		{URL: homepageURL, Category: "about"},
		{URL: "https://example.com/other", Category: "other"},
	}

	selected := selectPages(classified, 10, homepageURL)
	// Homepage should not be in selected (it's added separately)
	assert.NotContains(t, selected, homepageURL)
	assert.Contains(t, selected, "https://example.com/other")
}

func TestExtractTextFromHTML_ExtractsMainContent(t *testing.T) {
	html := `
		<html>
			<nav>Navigation</nav>
			<body>
				<main>
					<h1>Main Content</h1>
					<p>This is the main content of the page.</p>
				</main>
				<footer>Footer</footer>
			</body>
		</html>
	`

	text, err := extractTextFromHTML(html)
	require.NoError(t, err)

	assert.Contains(t, text, "Main Content")
	assert.Contains(t, text, "This is the main content")
	assert.NotContains(t, text, "Navigation")
	assert.NotContains(t, text, "Footer")
}

func TestExtractTextFromHTML_RemovesScriptsAndStyles(t *testing.T) {
	html := `
		<html>
			<head>
				<style>.class { color: red; }</style>
				<script>console.log('test');</script>
			</head>
			<body>
				<p>Content</p>
			</body>
		</html>
	`

	text, err := extractTextFromHTML(html)
	require.NoError(t, err)

	assert.Contains(t, text, "Content")
	assert.NotContains(t, text, "console.log")
	assert.NotContains(t, text, "color: red")
}

func TestComputeHash_ProducesConsistentHashes(t *testing.T) {
	content := "test content"
	hash1 := computeHash(content)
	hash2 := computeHash(content)

	assert.Equal(t, hash1, hash2)
	assert.Len(t, hash1, 64) // SHA256 hex is 64 characters
}

func TestCrawlBrandCorpus_InvalidURL(t *testing.T) {
	_, err := CrawlBrandCorpus(context.Background(), "not-a-url", 10, "api-key")
	assert.Error(t, err)
	var crawlErr *CrawlError
	assert.ErrorAs(t, err, &crawlErr)
}

func TestCrawlBrandCorpus_EnforcesMaxPagesLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	// This test would require mocking the classification function to properly verify
	// the max pages limit. Since classification requires API access, we skip this test
	// for now. The limit enforcement is tested implicitly through integration tests
	// and the logic is straightforward (simple comparison).
	t.Skip("Requires mocking classification function - skip for unit tests")
}

func TestCrawlBrandCorpus_HomepageOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create test server with no links
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><p>Homepage content</p></body></html>`))
	}))
	defer server.Close()

	// Mock classification won't be called if no links
	// But we need API key for the function, so we'll skip this test
	// or make it an integration test
	t.Skip("Requires mocking or real API for full test")
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
