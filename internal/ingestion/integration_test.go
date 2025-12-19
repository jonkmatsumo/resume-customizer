package ingestion

import (
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
	cleanedText, metadata, err := IngestFromFile(testFile)
	require.NoError(t, err)

	// Write output
	outDir := filepath.Join(tmpDir, "output")
	err = WriteOutput(outDir, cleanedText, metadata)
	require.NoError(t, err)

	// Verify output files
	cleanedPath := filepath.Join(outDir, "job_posting.cleaned.txt")
	cleanedFileContent, err := os.ReadFile(cleanedPath)
	require.NoError(t, err)
	assert.Contains(t, string(cleanedFileContent), "Senior Software Engineer")
	assert.Contains(t, string(cleanedFileContent), "Requirements")

	metaPath := filepath.Join(outDir, "job_posting.meta.json")
	metaFileContent, err := os.ReadFile(metaPath)
	require.NoError(t, err)
	assert.Contains(t, string(metaFileContent), "timestamp")
	assert.Contains(t, string(metaFileContent), "hash")
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	// Ingest from URL
	cleanedText, metadata, err := IngestFromURL(server.URL)
	require.NoError(t, err)

	// Write output
	outDir := filepath.Join(tmpDir, "output")
	err = WriteOutput(outDir, cleanedText, metadata)
	require.NoError(t, err)

	// Verify output
	cleanedPath := filepath.Join(outDir, "job_posting.cleaned.txt")
	cleanedFileContent, err := os.ReadFile(cleanedPath)
	require.NoError(t, err)
	assert.Contains(t, string(cleanedFileContent), "Senior Software Engineer")
	assert.Contains(t, string(cleanedFileContent), "Requirements")
	assert.NotContains(t, string(cleanedFileContent), "Nav")
	assert.NotContains(t, string(cleanedFileContent), "Footer")

	metaPath := filepath.Join(outDir, "job_posting.meta.json")
	metaFileContent, err := os.ReadFile(metaPath)
	require.NoError(t, err)
	assert.Contains(t, string(metaFileContent), server.URL)
}

func TestOutputFileFormat_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cleanedText := "Test content"
	metadata := NewMetadata(cleanedText, "https://example.com/job")

	err := WriteOutput(tmpDir, cleanedText, metadata)
	require.NoError(t, err)

	metaPath := filepath.Join(tmpDir, "job_posting.meta.json")
	metaFileContent, err := os.ReadFile(metaPath)
	require.NoError(t, err)

	// Verify it's valid JSON
	var unmarshaled Metadata
	err = json.Unmarshal(metaFileContent, &unmarshaled)
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
			content, err := os.ReadFile(tt.fixture)
			require.NoError(t, err)

			var cleanedText string
			// For HTML files, extract text first, then clean
			if filepath.Ext(tt.fixture) == ".html" {
				extractedText, err := extractTextFromHTML(string(content))
				require.NoError(t, err)
				cleanedText = CleanText(extractedText)
			} else {
				// For text files, clean directly
				cleanedText = CleanText(string(content))
			}

			for _, expected := range tt.expected {
				assert.Contains(t, cleanedText, expected, "should contain expected text")
			}

			for _, notIn := range tt.notIn {
				assert.NotContains(t, cleanedText, notIn, "should not contain unwanted text")
			}
		})
	}
}
