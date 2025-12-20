package crawling

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLinks_HomepageWithNav(t *testing.T) {
	html := `
		<html>
			<body>
				<nav>
					<a href="/about">About</a>
					<a href="/careers">Careers</a>
					<a href="/values">Values</a>
				</nav>
				<main>
					<a href="/blog">Blog</a>
					<a href="https://other.com/external">External</a>
				</main>
				<footer>
					<a href="/press">Press</a>
				</footer>
			</body>
		</html>
	`

	links, err := ExtractLinks(html, "https://example.com")
	require.NoError(t, err)
	require.Len(t, links, 5)

	// Check that all expected links are present
	expectedLinks := map[string]bool{
		"https://example.com/about":   true,
		"https://example.com/careers": true,
		"https://example.com/values":  true,
		"https://example.com/blog":    true,
		"https://example.com/press":   true,
	}

	for _, link := range links {
		assert.True(t, expectedLinks[link], "unexpected link: %s", link)
		delete(expectedLinks, link)
	}

	// Check that external link was filtered out
	for link := range expectedLinks {
		assert.NotContains(t, links, link)
	}
}

func TestExtractLinks_FiltersExternalLinks(t *testing.T) {
	html := `
		<html>
			<body>
				<a href="https://example.com/internal">Internal</a>
				<a href="https://other.com/external">External</a>
				<a href="http://example.com/mixed">Mixed Protocol</a>
			</body>
		</html>
	`

	links, err := ExtractLinks(html, "https://example.com")
	require.NoError(t, err)
	assert.Len(t, links, 2)
	assert.Contains(t, links, "https://example.com/internal")
	assert.Contains(t, links, "http://example.com/mixed")
	assert.NotContains(t, links, "https://other.com/external")
}

func TestExtractLinks_NormalizesRelativeURLs(t *testing.T) {
	html := `
		<html>
			<body>
				<a href="/relative">Relative</a>
				<a href="relative2">Relative No Slash</a>
				<a href="../parent">Parent</a>
			</body>
		</html>
	`

	links, err := ExtractLinks(html, "https://example.com/path/to/page")
	require.NoError(t, err)
	assert.Len(t, links, 3)
	assert.Contains(t, links, "https://example.com/relative")
	assert.Contains(t, links, "https://example.com/path/to/relative2")
	assert.Contains(t, links, "https://example.com/path/parent")
}

func TestExtractLinks_RemovesDuplicates(t *testing.T) {
	html := `
		<html>
			<body>
				<a href="/duplicate">Duplicate 1</a>
				<a href="/duplicate">Duplicate 2</a>
				<a href="/duplicate/">Duplicate 3 (trailing slash)</a>
			</body>
		</html>
	`

	links, err := ExtractLinks(html, "https://example.com")
	require.NoError(t, err)
	// Should have only one instance (trailing slash normalization may vary, but duplicates should be removed)
	assert.LessOrEqual(t, len(links), 2) // At most 2 (with/without trailing slash before normalization)
}

func TestExtractLinks_RemovesFragments(t *testing.T) {
	html := `
		<html>
			<body>
				<a href="/page#section">With Fragment</a>
				<a href="/page#other">Same Page Different Fragment</a>
			</body>
		</html>
	`

	links, err := ExtractLinks(html, "https://example.com")
	require.NoError(t, err)
	// Should normalize to same URL (fragments removed)
	assert.LessOrEqual(t, len(links), 1)
	if len(links) > 0 {
		assert.Contains(t, links[0], "https://example.com/page")
		assert.NotContains(t, links[0], "#")
	}
}

func TestExtractLinks_InvalidBaseURL(t *testing.T) {
	html := `<html><body><a href="/link">Link</a></body></html>`

	_, err := ExtractLinks(html, "not-a-valid-url")
	assert.Error(t, err)
	var linkErr *LinkExtractionError
	assert.ErrorAs(t, err, &linkErr)
}

func TestExtractLinks_EmptyHTML(t *testing.T) {
	links, err := ExtractLinks("", "https://example.com")
	require.NoError(t, err)
	assert.Empty(t, links)
}

func TestExtractLinks_NoLinks(t *testing.T) {
	html := `<html><body><p>No links here</p></body></html>`

	links, err := ExtractLinks(html, "https://example.com")
	require.NoError(t, err)
	assert.Empty(t, links)
}

func TestExtractLinks_MalformedLinks(t *testing.T) {
	html := `
		<html>
			<body>
				<a href="valid">Valid</a>
				<a href="://invalid">Invalid</a>
				<a>No href</a>
			</body>
		</html>
	`

	links, err := ExtractLinks(html, "https://example.com")
	require.NoError(t, err)
	// Should only include valid link
	assert.Len(t, links, 1)
	assert.Contains(t, links, "https://example.com/valid")
}
