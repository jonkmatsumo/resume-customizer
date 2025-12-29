// Package ingestion provides functionality to ingest job postings from files or URLs and clean the text content.
package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// CleanHTML extracts meaningful text and links from HTML content
func CleanHTML(htmlContent string) (string, []string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract links before removing elements
	var links []string
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && href != "" && !strings.HasPrefix(href, "javascript:") && !strings.HasPrefix(href, "#") {
			links = append(links, strings.TrimSpace(href))
		}
	})

	// Remove noise
	// Remove noise tags
	doc.Find("script, style, nav, footer, header, meta, link, svg, path, noscript, iframe, aside").Remove()

	// Remove noise classes/ids
	// This is a heuristic list of common non-content selectors
	noiseSelectors := []string{
		".sidebar", "#sidebar",
		".nav", "#nav", ".navigation",
		".footer", "#footer",
		".header", "#header",
		".ad", ".advertisement", ".banner",
		".cookie-banner", ".cookie-consent",
		".menu", "#menu",
		".social-media", ".share-buttons",
	}
	doc.Find(strings.Join(noiseSelectors, ", ")).Remove()

	// Extract text. Use a custom walker or just standard .Text() but .Text() joins tightly.
	// We want to preserve basic structure.
	// Better approach: Iterate over relevant block elements and extract text with newlines.
	var sb strings.Builder
	doc.Find("body").Each(func(_ int, s *goquery.Selection) {
		// Just getting all text often squashes things.
		// Let's try reasonable approach: separate block tags with newlines.
		text := s.Text()
		sb.WriteString(text)
	})

	rawText := doc.Text()

	// Post-process the text using the existing text logic
	cleaned := CleanText(rawText)
	return cleaned, links, nil
}

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

// removeExcessiveBlankLines reduces consecutive blank lines to max 2
func removeExcessiveBlankLines(content string) string {
	// Replace 3+ consecutive newlines with 2 newlines
	re := regexp.MustCompile(`\n\n\n+`)
	return re.ReplaceAllString(content, "\n\n")
}

// IngestFromFile reads a text file, cleans it, and returns cleaned text with metadata
func IngestFromFile(ctx context.Context, path string, apiKey string) (string, *Metadata, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf("file not found: %w", err)
		}
		return "", nil, fmt.Errorf("failed to read file: %w", err)
	}

	var cleanedText string
	var links []string
	var adminInfo map[string]string

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".html" || ext == ".htm" {
		cleanedText, links, err = CleanHTML(string(content))
		if err != nil {
			return "", nil, fmt.Errorf("failed to clean HTML: %w", err)
		}
	} else {
		cleanedText = CleanText(string(content))
	}

	// If API key is provided, use LLM to separate core content from metadata
	if apiKey != "" {
		extracted, err := ExtractWithLLM(ctx, cleanedText, apiKey)
		if err == nil {
			// Success! Use extracted content
			var sb strings.Builder
			sb.WriteString("Requirements:\n")
			for _, req := range extracted.Requirements {
				sb.WriteString("- " + req + "\n")
			}
			sb.WriteString("\nResponsibilities:\n")
			for _, resp := range extracted.Responsibilities {
				sb.WriteString("- " + resp + "\n")
			}
			cleanedText = sb.String()
			adminInfo = extracted.AdminInfo
		} else {
			return "", nil, fmt.Errorf("LLM extraction failed: %w", err)
		}
	}

	metadata := NewMetadata(cleanedText, "")
	metadata.ExtractedLinks = links
	metadata.AdminInfo = adminInfo

	// Attempt to load corresponding metadata if it exists (e.g., job_posting.meta.json)
	metaPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".meta.json"
	if _, err := os.Stat(metaPath); err == nil {
		if metaData, err := os.ReadFile(metaPath); err == nil {
			var m Metadata
			if err := json.Unmarshal(metaData, &m); err == nil {
				// Merge loaded metadata
				if m.URL != "" {
					metadata.URL = m.URL
				}
				if m.Platform != "" {
					metadata.Platform = m.Platform
				}
				if m.Company != "" {
					metadata.Company = m.Company
				}
				if m.AboutCompany != "" {
					metadata.AboutCompany = m.AboutCompany
				}
				if len(m.ExtractedLinks) > 0 {
					metadata.ExtractedLinks = m.ExtractedLinks
				}
				if len(m.AdminInfo) > 0 {
					metadata.AdminInfo = m.AdminInfo
				}
			}
		}
	} else {
		// Try .json if .meta.json doesn't exist
		metaPath = strings.TrimSuffix(path, filepath.Ext(path)) + ".json"
		if _, err := os.Stat(metaPath); err == nil {
			if metaData, err := os.ReadFile(metaPath); err == nil {
				var m Metadata
				if err := json.Unmarshal(metaData, &m); err == nil {
					if m.URL != "" {
						metadata.URL = m.URL
					}
					if m.Platform != "" {
						metadata.Platform = m.Platform
					}
					if m.Company != "" {
						metadata.Company = m.Company
					}
					if m.AboutCompany != "" {
						metadata.AboutCompany = m.AboutCompany
					}
					if len(m.ExtractedLinks) > 0 {
						metadata.ExtractedLinks = m.ExtractedLinks
					}
					if len(m.AdminInfo) > 0 {
						metadata.AdminInfo = m.AdminInfo
					}
				}
			}
		}
	}

	return cleanedText, metadata, nil
}
