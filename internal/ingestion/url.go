package ingestion

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	// ErrInvalidURL is returned when URL is malformed
	ErrInvalidURL = fmt.Errorf("invalid URL")
	// ErrHTTPRequestFailed is returned when HTTP request fails
	ErrHTTPRequestFailed = fmt.Errorf("HTTP request failed")
	// ErrContentExtractionFailed is returned when content extraction fails
	ErrContentExtractionFailed = fmt.Errorf("content extraction failed")
)

// IngestFromURL fetches content from a URL, extracts text, cleans it, and returns cleaned text with metadata
func IngestFromURL(urlStr string) (string, *Metadata, error) {
	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", nil, fmt.Errorf("%w: %s", ErrInvalidURL, urlStr)
	}

	// Fetch HTML
	htmlContent, err := fetchHTML(urlStr)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", ErrHTTPRequestFailed, err)
	}

	// Extract text from HTML
	textContent, err := extractTextFromHTML(htmlContent)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", ErrContentExtractionFailed, err)
	}

	// Clean text
	cleanedText := CleanText(textContent)

	// Generate metadata
	metadata := NewMetadata(cleanedText, urlStr)

	return cleanedText, metadata, nil
}

// fetchHTML fetches HTML content from a URL
func fetchHTML(urlStr string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}

	// Set user agent to avoid blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ResumeAgent/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

// extractTextFromHTML extracts main content text from HTML, removing nav, footer, etc.
func extractTextFromHTML(htmlContent string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Remove unwanted elements
	doc.Find("nav, footer, header, script, style, .ad, .advertisement, .ads, .sidebar").Remove()

	// Try to find main content using common selectors (in priority order)
	var mainContent *goquery.Selection

	selectors := []string{
		"main",
		"article",
		".job-description",
		".job-content",
		"#job-description",
		".content",
		"#content",
	}

	for _, selector := range selectors {
		if selection := doc.Find(selector); selection.Length() > 0 {
			mainContent = selection.First()
			break
		}
	}

	// Fallback to body (minus nav/footer which we already removed)
	if mainContent == nil {
		mainContent = doc.Find("body")
	}

	// Extract text content
	text := mainContent.Text()

	// Clean up extra whitespace
	text = strings.TrimSpace(text)

	return text, nil
}
