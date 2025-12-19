package ingestion

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CleanText cleans and normalizes text content while preserving structure
func CleanText(content string) string {
	if content == "" {
		return ""
	}

	// 1. Normalize line endings (CRLF → LF)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// 2. Split into lines for processing
	lines := strings.Split(content, "\n")

	// 3. Process each line
	cleanedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		cleaned := cleanLine(line)
		cleanedLines = append(cleanedLines, cleaned)
	}

	// 4. Join lines
	result := strings.Join(cleanedLines, "\n")

	// 5. Remove excessive blank lines (max 2 consecutive)
	result = removeExcessiveBlankLines(result)

	// 6. Trim leading/trailing whitespace from entire content
	result = strings.TrimSpace(result)

	return result
}

// cleanLine cleans a single line while preserving structure
func cleanLine(line string) string {
	// Trim trailing whitespace
	line = strings.TrimRight(line, " \t")

	// Handle empty lines
	if strings.TrimSpace(line) == "" {
		return ""
	}

	// Preserve headings (Markdown # or ## etc.)
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "#") {
		// Keep markdown headings as-is, normalize leading spaces to 0
		return trimmed
	}

	// Preserve bullet lists (Markdown - or *)
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		// Preserve indentation before bullet, but normalize
		indent := len(line) - len(trimmed)
		if indent > 0 {
			return strings.Repeat(" ", indent) + trimmed
		}
		return trimmed
	}

	// For regular lines, normalize multiple spaces to single space
	// but preserve intentional indentation at start of line
	leadingSpace := len(line) - len(trimmed)
	content := strings.TrimSpace(line)
	// Normalize spaces in content (multiple spaces → single)
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	if leadingSpace > 0 {
		return strings.Repeat(" ", leadingSpace) + content
	}
	return content
}

// isBulletLine checks if a line is a bullet list item
func isBulletLine(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") ||
		strings.HasPrefix(trimmed, "• ") || strings.HasPrefix(trimmed, "· ")
}

// removeExcessiveBlankLines reduces consecutive blank lines to max 2
func removeExcessiveBlankLines(content string) string {
	// Replace 3+ consecutive newlines with 2 newlines
	re := regexp.MustCompile(`\n\n\n+`)
	return re.ReplaceAllString(content, "\n\n")
}

// IngestFromFile reads a text file, cleans it, and returns cleaned text with metadata
func IngestFromFile(path string) (string, *Metadata, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf("file not found: %w", err)
		}
		return "", nil, fmt.Errorf("failed to read file: %w", err)
	}

	cleanedText := CleanText(string(content))
	metadata := NewMetadata(cleanedText, "")

	return cleanedText, metadata, nil
}

// WriteOutput writes the cleaned text and metadata to output files
func WriteOutput(outDir string, cleanedText string, metadata *Metadata) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write cleaned text file
	cleanedPath := filepath.Join(outDir, "job_posting.cleaned.txt")
	if err := os.WriteFile(cleanedPath, []byte(cleanedText), 0644); err != nil {
		return fmt.Errorf("failed to write cleaned text file: %w", err)
	}

	// Write metadata JSON file
	metaPath := filepath.Join(outDir, "job_posting.meta.json")
	metaJSON, err := metadata.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaJSON, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}
