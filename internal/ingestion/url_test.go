package ingestion

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jonathan/resume-customizer/internal/fetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestFromURL_InvalidURL(t *testing.T) {
	tests := []struct {
		name    string
		urlStr  string
		wantErr bool
	}{
		{"empty URL", "", true},
		{"malformed URL", "not-a-url", true},
		{"no scheme", "example.com", true},
		{"no host", "http://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := IngestFromURL(context.Background(), tt.urlStr, "", false, false)
			if tt.wantErr {
				assert.Error(t, err)
			}
		})
	}
}

func TestIngestFromURL_Success(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		html := `<!DOCTYPE html>
<html>
<body>
<nav>Nav</nav>
<main>
<h1>Job Title</h1>
<p>Job description</p>
</main>
<footer>Footer</footer>
</body>
</html>`
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	cleanedText, metadata, err := IngestFromURL(context.Background(), server.URL, "", false, false)
	require.NoError(t, err)

	assert.NotEmpty(t, cleanedText)
	assert.NotNil(t, metadata)
	assert.Equal(t, server.URL, metadata.URL)
	assert.Contains(t, cleanedText, "Job Title")
	assert.Contains(t, cleanedText, "Job description")
	// Should not contain nav/footer
	assert.NotContains(t, cleanedText, "Nav")
	assert.NotContains(t, cleanedText, "Footer")
}

func TestIngestFromURL_HTTPError(t *testing.T) {
	// Create mock server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, _, err := IngestFromURL(context.Background(), server.URL, "", false, false)
	assert.Error(t, err)
}

func TestIngestFromURL_NetworkError(t *testing.T) {
	// Use invalid URL that will fail to connect
	_, _, err := IngestFromURL(context.Background(), "http://localhost:99999/nonexistent", "", false, false)
	assert.Error(t, err)
}

func TestExtractMainText_GreenhouseLike(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
<nav>Navigation</nav>
<main>
<h1>Senior Software Engineer</h1>
<article>
<h2>About the Role</h2>
<p>We are looking for a Senior Software Engineer.</p>
</article>
</main>
<footer>Footer</footer>
</body>
</html>`

	text, err := fetch.ExtractMainText(html, fetch.JobPostingSelectors())
	require.NoError(t, err)

	assert.Contains(t, text, "Senior Software Engineer")
	assert.Contains(t, text, "About the Role")
	assert.Contains(t, text, "We are looking for")
	// Should not contain nav/footer
	assert.NotContains(t, text, "Navigation")
	assert.NotContains(t, text, "Footer")
}

func TestExtractMainText_LeverLike(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
<div class="sidebar">Sidebar</div>
<div class="job-description">
<h1>Senior Software Engineer</h1>
<p>Job description here</p>
</div>
<div class="advertisement">Ad</div>
</body>
</html>`

	text, err := fetch.ExtractMainText(html, fetch.JobPostingSelectors())
	require.NoError(t, err)

	assert.Contains(t, text, "Senior Software Engineer")
	assert.Contains(t, text, "Job description here")
	// Should not contain sidebar/ad
	assert.NotContains(t, text, "Sidebar")
	assert.NotContains(t, text, "Ad")
}

func TestExtractMainText_RemovesScriptAndStyle(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
<style>body { color: red; }</style>
</head>
<body>
<main>
<p>Content here</p>
<script>alert('test');</script>
</main>
</body>
</html>`

	text, err := fetch.ExtractMainText(html, fetch.JobPostingSelectors())
	require.NoError(t, err)

	assert.Contains(t, text, "Content here")
	assert.NotContains(t, text, "alert")
	assert.NotContains(t, text, "color: red")
}

func TestExtractMainText_FallbackToBody(t *testing.T) {
	// HTML without main/article elements
	html := `<!DOCTYPE html>
<html>
<body>
<h1>Title</h1>
<p>Content</p>
</body>
</html>`

	text, err := fetch.ExtractMainText(html, fetch.DefaultTextSelectors())
	require.NoError(t, err)

	assert.Contains(t, text, "Title")
	assert.Contains(t, text, "Content")
}

func TestURL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>Test</body></html>"))
	}))
	defer server.Close()

	result, err := fetch.URL(context.Background(), server.URL, nil)
	require.NoError(t, err)

	assert.Contains(t, result.HTML, "Test")
}

func TestURL_404Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := fetch.URL(context.Background(), server.URL, nil)
	assert.Error(t, err)
}

func TestURL_500Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := fetch.URL(context.Background(), server.URL, nil)
	assert.Error(t, err)
}

func TestIngestFromURL_WithTestFixtures(t *testing.T) {
	// Test with HTML fixture
	testFile := "testdata/sample_job_html.html"
	htmlContent, err := os.ReadFile(testFile)
	require.NoError(t, err)

	// Create mock server serving the HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(htmlContent)
	}))
	defer server.Close()

	cleanedText, metadata, err := IngestFromURL(context.Background(), server.URL, "", false, false)
	require.NoError(t, err)

	assert.NotEmpty(t, cleanedText)
	assert.NotNil(t, metadata)
	assert.Contains(t, cleanedText, "Senior Software Engineer")
	assert.Contains(t, cleanedText, "About the Role")
	assert.Contains(t, cleanedText, "Requirements")
}
