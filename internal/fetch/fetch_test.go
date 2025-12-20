package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURL_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body><h1>Test</h1></body></html>"))
	}))
	defer server.Close()

	result, err := URL(context.Background(), server.URL, nil)
	require.NoError(t, err)
	assert.Equal(t, server.URL, result.URL)
	assert.Contains(t, result.HTML, "<h1>Test</h1>")
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestURL_InvalidURL(t *testing.T) {
	_, err := URL(context.Background(), "not-a-valid-url", nil)
	require.Error(t, err)

	var fetchErr *Error
	assert.ErrorAs(t, err, &fetchErr)
	assert.Contains(t, err.Error(), "invalid URL")
}

func TestURL_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result, err := URL(context.Background(), server.URL, nil)
	require.Error(t, err)
	assert.NotNil(t, result) // Result is returned even on error
	assert.Equal(t, http.StatusNotFound, result.StatusCode)

	var fetchErr *Error
	assert.ErrorAs(t, err, &fetchErr)
	assert.Contains(t, err.Error(), "404")
}

func TestExtractMainText_WithMainElement(t *testing.T) {
	html := `
	<html>
		<body>
			<nav>Navigation</nav>
			<main>
				<h1>Main Content</h1>
				<p>This is the important text.</p>
			</main>
			<footer>Footer</footer>
		</body>
	</html>`

	text, err := ExtractMainText(html, DefaultTextSelectors())
	require.NoError(t, err)
	assert.Contains(t, text, "Main Content")
	assert.Contains(t, text, "important text")
	assert.NotContains(t, text, "Navigation")
	assert.NotContains(t, text, "Footer")
}

func TestExtractMainText_WithArticleElement(t *testing.T) {
	html := `
	<html>
		<body>
			<article>
				<h1>Article Title</h1>
				<p>Article body.</p>
			</article>
		</body>
	</html>`

	text, err := ExtractMainText(html, DefaultTextSelectors())
	require.NoError(t, err)
	assert.Contains(t, text, "Article Title")
	assert.Contains(t, text, "Article body")
}

func TestExtractMainText_FallbackToBody(t *testing.T) {
	html := `
	<html>
		<body>
			<div>Some content here.</div>
		</body>
	</html>`

	text, err := ExtractMainText(html, DefaultTextSelectors())
	require.NoError(t, err)
	assert.Contains(t, text, "Some content here")
}

func TestExtractMainText_JobPostingSelectors(t *testing.T) {
	html := `
	<html>
		<body>
			<div class="sidebar">Sidebar junk</div>
			<div class="job-description">
				<h2>Requirements</h2>
				<p>5 years experience in Go</p>
			</div>
		</body>
	</html>`

	text, err := ExtractMainText(html, JobPostingSelectors())
	require.NoError(t, err)
	assert.Contains(t, text, "Requirements")
	assert.Contains(t, text, "5 years experience")
	assert.NotContains(t, text, "Sidebar junk")
}

func TestDefaultTextSelectors(t *testing.T) {
	selectors := DefaultTextSelectors()
	assert.Contains(t, selectors, "main")
	assert.Contains(t, selectors, "article")
}

func TestJobPostingSelectors(t *testing.T) {
	selectors := JobPostingSelectors()
	assert.Contains(t, selectors, ".job-description")
	assert.Contains(t, selectors, "#job-content")
}

func TestCompanyPageSelectors(t *testing.T) {
	selectors := CompanyPageSelectors()
	assert.Contains(t, selectors, "main")
	assert.Contains(t, selectors, ".about-content")
}
