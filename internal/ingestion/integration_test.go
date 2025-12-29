package ingestion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndToEnd_TextFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create input file
	testFile := filepath.Join(tmpDir, "input.txt")
	testContent := "# Senior Software Engineer\n\n## Requirements\n- Go experience\n- Distributed systems"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Ingest
	cleanedText, metadata, err := IngestFromFile(context.Background(), testFile, "")
	require.NoError(t, err)

	// Verify ingestion results (file writing removed - data should be saved to database)
	assert.Contains(t, cleanedText, "Senior Software Engineer")
	assert.Contains(t, cleanedText, "Requirements")
	assert.NotNil(t, metadata)
	assert.NotEmpty(t, metadata.Timestamp)
	assert.NotEmpty(t, metadata.Hash)
}

func TestEndToEnd_URL_MockServer(t *testing.T) {
	// Create mock HTTP server
	htmlContent := `<!DOCTYPE html>
<html>
<body>
<nav>Nav</nav>
<main>
<h1>Senior Software Engineer</h1>
<article>
<h2>Requirements</h2>
<ul>
<li>Go experience</li>
<li>Distributed systems</li>
</ul>
</article>
</main>
<footer>Footer</footer>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	// Ingest from URL
	cleanedText, metadata, err := IngestFromURL(context.Background(), server.URL, "", false, false)
	require.NoError(t, err)

	// Verify ingestion results (file writing removed - data should be saved to database)
	assert.Contains(t, cleanedText, "Senior Software Engineer")
	assert.Contains(t, cleanedText, "Requirements")
	assert.NotContains(t, cleanedText, "Nav")
	assert.NotContains(t, cleanedText, "Footer")
	assert.NotNil(t, metadata)
	assert.Equal(t, server.URL, metadata.URL)
}

func TestMetadata_ValidJSON(t *testing.T) {
	// Test that metadata can be serialized to valid JSON
	// (file writing removed - data should be saved to database)
	cleanedText := "Test content"
	metadata := NewMetadata(cleanedText, "https://example.com/job")

	// Verify metadata can be serialized to JSON
	metaJSON, err := metadata.ToJSON()
	require.NoError(t, err)

	// Verify it's valid JSON
	var unmarshaled Metadata
	err = json.Unmarshal(metaJSON, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, metadata.URL, unmarshaled.URL)
	assert.Equal(t, metadata.Timestamp, unmarshaled.Timestamp)
	assert.Equal(t, metadata.Hash, unmarshaled.Hash)
}

func TestRealJobBoardFormats(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		expected []string
		notIn    []string
	}{
		{
			name:     "Markdown format",
			fixture:  "testdata/sample_job_markdown.txt",
			expected: []string{"Senior Software Engineer", "About the Role", "Requirements"},
		},
		{
			name:     "Plain text format",
			fixture:  "testdata/sample_job_plain.txt",
			expected: []string{"Senior Software Engineer", "About the Role", "Requirements"},
		},
		{
			name:     "HTML format (Greenhouse-like)",
			fixture:  "testdata/sample_job_html.html",
			expected: []string{"Senior Software Engineer", "About the Role", "Requirements"},
			notIn:    []string{"Navigation", "Header", "Footer"},
		},
		{
			name:     "Lever format",
			fixture:  "testdata/sample_job_lever.html",
			expected: []string{"Senior Software Engineer", "About the Role", "Requirements"},
			notIn:    []string{"Sidebar", "Ad content"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var cleanedText string
			// Use IngestFromFile which now handles HTML and text files
			cleaned, _, err := IngestFromFile(context.Background(), tt.fixture, "")
			require.NoError(t, err)
			cleanedText = cleaned

			for _, expected := range tt.expected {
				assert.Contains(t, cleanedText, expected, "should contain expected text")
			}

			for _, notIn := range tt.notIn {
				assert.NotContains(t, cleanedText, notIn, "should not contain unwanted text")
			}
		})
	}
}
