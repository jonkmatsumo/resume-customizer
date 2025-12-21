// Package fetch provides generic URL fetching and HTML-to-text processing.
// This package centralizes HTTP fetching logic used by ingestion and crawling.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net"
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

// Retry configuration
const (
	DefaultMaxRetries     = 3
	DefaultInitialBackoff = 500 * time.Millisecond
	DefaultMaxBackoff     = 10 * time.Second
	BackoffMultiplier     = 2.0
)

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
	URL       string
	Message   string
	Cause     error
	Retryable bool // Whether this error is retryable
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
	Timeout        time.Duration
	UserAgent      string
	Headers        map[string]string
	MaxRetries     int           // Maximum number of retry attempts (0 = no retries)
	InitialBackoff time.Duration // Initial backoff duration
	MaxBackoff     time.Duration // Maximum backoff duration
}

// DefaultOptions returns sensible defaults for fetching.
func DefaultOptions() *Options {
	return &Options{
		Timeout:        DefaultTimeout,
		UserAgent:      DefaultUserAgent,
		MaxRetries:     DefaultMaxRetries,
		InitialBackoff: DefaultInitialBackoff,
		MaxBackoff:     DefaultMaxBackoff,
	}
}

// isRetryableStatusCode returns true if the HTTP status code is retryable.
func isRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// isRetryableError returns true if the error is a transient network error.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network timeout errors
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}

	// Check for context deadline exceeded
	if err == context.DeadlineExceeded {
		return true
	}

	return false
}

// URL retrieves HTML content from a URL with retry support.
func URL(ctx context.Context, urlStr string, opts *Options) (*Result, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, &Error{
			URL:       urlStr,
			Message:   "invalid URL",
			Cause:     err,
			Retryable: false,
		}
	}

	var lastErr error
	backoff := opts.InitialBackoff

	for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
		// Wait before retry (skip on first attempt)
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, &Error{
					URL:       urlStr,
					Message:   "context cancelled during retry",
					Cause:     ctx.Err(),
					Retryable: false,
				}
			case <-time.After(backoff):
				// Continue with retry
			}

			// Exponential backoff with cap
			backoff = time.Duration(float64(backoff) * BackoffMultiplier)
			if backoff > opts.MaxBackoff {
				backoff = opts.MaxBackoff
			}
		}

		result, err := fetchOnce(ctx, urlStr, opts)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if fetchErr, ok := err.(*Error); ok {
			if !fetchErr.Retryable {
				return result, err // Non-retryable error, return immediately
			}
		}

		// Also check for retryable network errors
		if !isRetryableError(err) {
			if fetchErr, ok := err.(*Error); ok && !fetchErr.Retryable {
				return nil, err
			}
		}
	}

	// All retries exhausted
	return nil, &Error{
		URL:       urlStr,
		Message:   fmt.Sprintf("all %d retries exhausted", opts.MaxRetries),
		Cause:     lastErr,
		Retryable: false,
	}
}

// fetchOnce performs a single fetch attempt.
func fetchOnce(ctx context.Context, urlStr string, opts *Options) (*Result, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: opts.Timeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, &Error{
			URL:       urlStr,
			Message:   "failed to create request",
			Cause:     err,
			Retryable: false,
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
		retryable := isRetryableError(err)
		return nil, &Error{
			URL:       urlStr,
			Message:   "HTTP request failed",
			Cause:     err,
			Retryable: retryable,
		}
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &Error{
			URL:       urlStr,
			Message:   "failed to read response body",
			Cause:     err,
			Retryable: true, // Read errors can be transient
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
		retryable := isRetryableStatusCode(resp.StatusCode)
		return result, &Error{
			URL:       urlStr,
			Message:   fmt.Sprintf("HTTP status %d", resp.StatusCode),
			Retryable: retryable,
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
