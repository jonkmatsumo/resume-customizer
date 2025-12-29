package ingestion

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanText_PreserveMarkdownHeadings(t *testing.T) {
	input := "# Title\n## Subtitle\nContent here"
	result := CleanText(input)

	assert.Contains(t, result, "# Title")
	assert.Contains(t, result, "## Subtitle")
	assert.Contains(t, result, "Content here")
}

func TestCleanText_PreserveBulletLists(t *testing.T) {
	input := "- Item 1\n- Item 2\n* Item 3"
	result := CleanText(input)

	assert.Contains(t, result, "- Item 1")
	assert.Contains(t, result, "- Item 2")
	assert.Contains(t, result, "* Item 3")
}

func TestCleanText_NormalizeWhitespace(t *testing.T) {
	input := "Line    with    multiple    spaces"
	result := CleanText(input)

	assert.Contains(t, result, "Line with multiple spaces")
	assert.NotContains(t, result, "    ") // Should not have 4 spaces
}

func TestCleanText_RemoveExcessiveBlankLines(t *testing.T) {
	input := "Line 1\n\n\n\n\nLine 2"
	result := CleanText(input)

	// Should have max 2 consecutive newlines
	assert.NotContains(t, result, "\n\n\n\n")
	// But should preserve up to 2
	assert.Contains(t, result, "\n\n")
}

func TestCleanText_NormalizeLineEndings(t *testing.T) {
	input := "Line 1\r\nLine 2\rLine 3\nLine 4"
	result := CleanText(input)

	// All should be normalized to LF
	assert.NotContains(t, result, "\r\n")
	assert.NotContains(t, result, "\r")
	assert.Contains(t, result, "\n")
}

func TestCleanText_DeterministicOutput(t *testing.T) {
	input := "Test content   with   spaces\n\n\nMultiple   blank   lines"
	result1 := CleanText(input)
	result2 := CleanText(input)

	// Same input should produce identical output
	assert.Equal(t, result1, result2)
}

func TestCleanText_EmptyInput(t *testing.T) {
	result := CleanText("")
	assert.Empty(t, result)
}

func TestCleanText_OnlyWhitespace(t *testing.T) {
	result := CleanText("   \n  \n  ")
	assert.Empty(t, result)
}

func TestCleanText_SpecialCharacters(t *testing.T) {
	input := "Test with émojis 🚀 and spéciàl chàracters"
	result := CleanText(input)

	assert.Contains(t, result, "émojis")
	assert.Contains(t, result, "🚀")
	assert.Contains(t, result, "spéciàl chàracters")
}

func TestCleanText_PreserveIndentation(t *testing.T) {
	input := "    Indented line\n  Less indented"
	result := CleanText(input)

	// Should preserve relative indentation
	assert.Contains(t, result, "Indented")
	assert.Contains(t, result, "Less indented")
}

func TestIngestFromFile_Success(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "# Job Title\n\nDescription here"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	cleanedText, metadata, err := IngestFromFile(context.Background(), testFile, "")
	require.NoError(t, err)

	assert.NotEmpty(t, cleanedText)
	assert.NotNil(t, metadata)
	assert.Contains(t, cleanedText, "Job Title")
	assert.Len(t, metadata.Hash, 64)
	assert.NotEmpty(t, metadata.Timestamp)
}

func TestIngestFromFile_FileNotFound(t *testing.T) {
	cleanedText, metadata, err := IngestFromFile(context.Background(), "/nonexistent/file.txt", "")

	assert.Error(t, err)
	assert.Empty(t, cleanedText)
	assert.Nil(t, metadata)
	assert.Contains(t, err.Error(), "file not found")
}

func TestIngestFromFile_MetadataGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Test content"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	_, metadata1, err1 := IngestFromFile(context.Background(), testFile, "")
	require.NoError(t, err1)

	_, metadata2, err2 := IngestFromFile(context.Background(), testFile, "")
	require.NoError(t, err2)

	// Same file should produce same hash
	assert.Equal(t, metadata1.Hash, metadata2.Hash)

	// Timestamps may differ, but hashes should be the same
}

func TestIngestFromFile_HashUniqueness(t *testing.T) {
	tmpDir := t.TempDir()

	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")

	err := os.WriteFile(testFile1, []byte("Content 1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("Content 2"), 0644)
	require.NoError(t, err)

	_, metadata1, err1 := IngestFromFile(context.Background(), testFile1, "")
	require.NoError(t, err1)

	_, metadata2, err2 := IngestFromFile(context.Background(), testFile2, "")
	require.NoError(t, err2)

	// Different files should produce different hashes
	assert.NotEqual(t, metadata1.Hash, metadata2.Hash)
}

// TestWriteOutput_Success removed - WriteOutput() function has been removed
// File writing is now only used for debugging in ingest-job command

// TestWriteOutput_CreatesDirectory removed - WriteOutput() function has been removed

func TestCleanText_ComplexFormatting(t *testing.T) {
	// Read test fixture
	testFile := filepath.Join("testdata", "complex_formatting.txt")
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)

	result := CleanText(string(content))

	// Should preserve headings
	assert.Contains(t, result, "# Senior Software Engineer")
	assert.Contains(t, result, "## Responsibilities")

	// Should preserve bullets
	assert.Contains(t, result, "- Go experience")
	assert.Contains(t, result, "* Go (5+ years)")

	// Should normalize whitespace but preserve structure
	assert.NotEmpty(t, result)
}

func TestIngestFromFile_MergesCompanyFromMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	// Create job file
	testFile := filepath.Join(tmpDir, "job_posting.cleaned.txt")
	err := os.WriteFile(testFile, []byte("Job content"), 0644)
	require.NoError(t, err)

	// Create metadata file with Company and AboutCompany
	// Note: IngestFromFile strips the extension and adds .meta.json
	// So job_posting.cleaned.txt looks for job_posting.cleaned.meta.json
	metaFile := filepath.Join(tmpDir, "job_posting.cleaned.meta.json")
	metaJSON := `{
		"url": "https://example.com",
		"timestamp": "2024-01-01T00:00:00Z",
		"hash": "abc123",
		"company": "TestCorp",
		"about_company": "We build amazing things.",
		"extracted_links": ["https://example.com/about"]
	}`
	err = os.WriteFile(metaFile, []byte(metaJSON), 0644)
	require.NoError(t, err)

	_, metadata, err := IngestFromFile(context.Background(), testFile, "")
	require.NoError(t, err)

	assert.Equal(t, "TestCorp", metadata.Company)
	assert.Equal(t, "We build amazing things.", metadata.AboutCompany)
	assert.Equal(t, "https://example.com", metadata.URL)
	assert.Len(t, metadata.ExtractedLinks, 1)
}
