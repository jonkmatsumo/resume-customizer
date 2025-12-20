// Package fetch provides generic URL fetching and HTML-to-text processing.
// This package centralizes HTTP fetching logic used by ingestion and crawling.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// DefaultTimeout is the default HTTP request timeout.
const DefaultTimeout = 30 * time.Second

// DefaultUserAgent is the user agent string for HTTP requests.
const DefaultUserAgent = "Mozilla/5.0 (compatible; ResumeAgent/1.0)"

// Result holds the raw and processed content from a URL fetch.
type Result struct {
	URL         string
	HTML        string
	Text        string
	ContentType string
	StatusCode  int
}

// Error represents an error during URL fetching.
type Error struct {
	URL     string
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("fetch error for %s: %s: %v", e.URL, e.Message, e.Cause)
	}
	return fmt.Sprintf("fetch error for %s: %s", e.URL, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// Options configures the fetch behavior.
type Options struct {
	Timeout   time.Duration
	UserAgent string
	Headers   map[string]string
}

// DefaultOptions returns sensible defaults for fetching.
func DefaultOptions() *Options {
	return &Options{
		Timeout:   DefaultTimeout,
		UserAgent: DefaultUserAgent,
	}
}

// URL retrieves HTML content from a URL.
func URL(ctx context.Context, urlStr string, opts *Options) (*Result, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, &Error{
			URL:     urlStr,
			Message: "invalid URL",
			Cause:   err,
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: opts.Timeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, &Error{
			URL:     urlStr,
			Message: "failed to create request",
			Cause:   err,
		}
	}

	// Set headers
	req.Header.Set("User-Agent", opts.UserAgent)
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, &Error{
			URL:     urlStr,
			Message: "HTTP request failed",
			Cause:   err,
		}
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &Error{
			URL:     urlStr,
			Message: "failed to read response body",
			Cause:   err,
		}
	}

	result := &Result{
		URL:         urlStr,
		HTML:        string(bodyBytes),
		ContentType: resp.Header.Get("Content-Type"),
		StatusCode:  resp.StatusCode,
	}

	// Check for non-success status
	if resp.StatusCode != http.StatusOK {
		return result, &Error{
			URL:     urlStr,
			Message: fmt.Sprintf("HTTP status %d", resp.StatusCode),
		}
	}

	return result, nil
}

// ExtractMainText parses HTML and returns the main body text.
// It removes noise elements using noiseSelectors, then finds content using contentSelectors.
// If no content selectors match, it falls back to the body element.
func ExtractMainText(html string, contentSelectors []string, noiseSelectors ...string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Remove common unwanted elements (nav, footer, scripts, ads, etc.)
	doc.Find("nav, footer, header, script, style, noscript, .ad, .advertisement, .ads, .sidebar, .cookie-banner, .popup").Remove()

	// Remove platform-specific noise elements
	if len(noiseSelectors) > 0 {
		noiseSelector := strings.Join(noiseSelectors, ", ")
		if noiseSelector != "" {
			doc.Find(noiseSelector).Remove()
		}
	}

	// Try to find main content using provided selectors
	var mainContent *goquery.Selection
	for _, selector := range contentSelectors {
		if selection := doc.Find(selector); selection.Length() > 0 {
			mainContent = selection.First()
			break
		}
	}

	// Fallback to body if no selector matched
	if mainContent == nil {
		mainContent = doc.Find("body")
	}

	// Extract and clean text
	text := mainContent.Text()
	text = cleanWhitespace(text)

	return text, nil
}

// DefaultTextSelectors returns standard selectors for general web content.
func DefaultTextSelectors() []string {
	return []string{
		"main",
		"article",
		".content",
		"#content",
		".main-content",
		"#main-content",
	}
}

// JobPostingSelectors returns selectors optimized for job board pages.
func JobPostingSelectors() []string {
	return []string{
		".job-description",
		".job-content",
		"#job-description",
		"#job-content",
		".posting-content",
		".job-details",
		"[data-testid='job-description']",
		"main",
		"article",
		".content",
		"#content",
	}
}

// CompanyPageSelectors returns selectors for company pages (about, values, culture).
func CompanyPageSelectors() []string {
	return []string{
		"main",
		"article",
		".about-content",
		".values-content",
		".culture-content",
		".content",
		"#content",
	}
}

// cleanWhitespace normalizes whitespace in text.
func cleanWhitespace(text string) string {
	// Replace multiple whitespace characters with single space
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}
