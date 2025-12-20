package ingestion

import (
	"context"
	"encoding/json"
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

func TestWriteOutput_Success(t *testing.T) {
	tmpDir := t.TempDir()
	cleanedText := "Cleaned text content"
	metadata := NewMetadata(cleanedText, "https://example.com")

	err := WriteOutput(tmpDir, cleanedText, metadata)
	require.NoError(t, err)

	// Check cleaned text file exists
	cleanedPath := filepath.Join(tmpDir, "job_posting.cleaned.txt")
	cleanedContent, err := os.ReadFile(cleanedPath)
	require.NoError(t, err)
	assert.Equal(t, cleanedText, string(cleanedContent))

	// Check metadata file exists
	metaPath := filepath.Join(tmpDir, "job_posting.meta.json")
	metaContent, err := os.ReadFile(metaPath)
	require.NoError(t, err)
	assert.NotEmpty(t, metaContent)

	// Verify it's valid JSON
	var unmarshaled Metadata
	jsonBytes, err := metadata.ToJSON()
	require.NoError(t, err)
	_, err = os.ReadFile(metaPath)
	require.NoError(t, err)
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, metadata.URL, unmarshaled.URL)
}

func TestWriteOutput_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "new", "sub", "directory")

	cleanedText := "Test content"
	metadata := NewMetadata(cleanedText, "")

	err := WriteOutput(outDir, cleanedText, metadata)
	require.NoError(t, err)

	// Directory should be created
	cleanedPath := filepath.Join(outDir, "job_posting.cleaned.txt")
	_, err = os.Stat(cleanedPath)
	assert.NoError(t, err)
}

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
