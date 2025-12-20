package crawling

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jonathan/resume-customizer/internal/ingestion"
	"github.com/jonathan/resume-customizer/internal/types"
)

const (
	// MaxPagesLimit is the hard maximum number of pages to crawl
	MaxPagesLimit = 15
	// DefaultRateLimitDelay is the delay between HTTP requests
	DefaultRateLimitDelay = 1 * time.Second
)

// CrawlBrandCorpus crawls a company website and builds a text corpus
func CrawlBrandCorpus(ctx context.Context, seedURL string, maxPages int, apiKey string) (*types.CompanyCorpus, error) {
	// Validate seed URL
	parsedURL, err := url.Parse(seedURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, &CrawlError{
			Message: fmt.Sprintf("invalid seed URL: %s", seedURL),
			Cause:   err,
		}
	}

	// Enforce max pages limit
	if maxPages > MaxPagesLimit {
		maxPages = MaxPagesLimit
	}
	if maxPages < 1 {
		maxPages = 10 // Default
	}

	// Fetch homepage
	homepageHTML, err := fetchHTML(seedURL)
	if err != nil {
		return nil, &CrawlError{
			Message: "failed to fetch homepage",
			Cause:   err,
		}
	}

	// Extract links
	links, err := ExtractLinks(homepageHTML, seedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract links: %w", err)
	}

	// Limit candidate links to top 30 for classification efficiency
	maxCandidates := 30
	if len(links) > maxCandidates {
		links = links[:maxCandidates]
	}

	if len(links) == 0 {
		// Only homepage available
		text, err := extractTextFromHTML(homepageHTML)
		if err != nil {
			return nil, &CrawlError{
				Message: "failed to extract text from homepage",
				Cause:   err,
			}
		}

		cleanedText := ingestion.CleanText(text)
		hash := computeHash(cleanedText)

		return &types.CompanyCorpus{
			Corpus: cleanedText,
			Sources: []types.Source{
				{
					URL:       seedURL,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Hash:      hash,
				},
			},
		}, nil
	}

	// Classify links
	classified, err := ClassifyLinks(ctx, links, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to classify links: %w", err)
	}

	// Select pages to crawl based on classification
	selectedURLs := selectPages(classified, maxPages, seedURL)

	// Crawl selected pages
	var corpusParts []string
	sources := make([]types.Source, 0)

	// Always include homepage
	homepageText, err := extractTextFromHTML(homepageHTML)
	if err != nil {
		return nil, &CrawlError{
			Message: "failed to extract text from homepage",
			Cause:   err,
		}
	}
	cleanedHomepageText := ingestion.CleanText(homepageText)
	homepageHash := computeHash(cleanedHomepageText)
	corpusParts = append(corpusParts, cleanedHomepageText)
	sources = append(sources, types.Source{
		URL:       seedURL,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Hash:      homepageHash,
	})

	// Crawl additional pages
	visited := map[string]bool{seedURL: true}
	for _, pageURL := range selectedURLs {
		if visited[pageURL] {
			continue
		}
		visited[pageURL] = true

		// Rate limiting
		time.Sleep(DefaultRateLimitDelay)

		// Fetch page
		pageHTML, err := fetchHTML(pageURL)
		if err != nil {
			// Log error but continue with other pages
			continue
		}

		// Extract text
		pageText, err := extractTextFromHTML(pageHTML)
		if err != nil {
			continue
		}

		cleanedPageText := ingestion.CleanText(pageText)
		pageHash := computeHash(cleanedPageText)

		corpusParts = append(corpusParts, cleanedPageText)
		sources = append(sources, types.Source{
			URL:       pageURL,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Hash:      pageHash,
		})
	}

	// Concatenate corpus with separators
	corpus := strings.Join(corpusParts, "\n\n---\n\n")

	return &types.CompanyCorpus{
		Corpus:  corpus,
		Sources: sources,
	}, nil
}

// selectPages selects pages to crawl based on classification
func selectPages(classified []ClassifiedLink, maxPages int, homepageURL string) []string {
	// Prioritize categories: values, careers, press (one each minimum)
	// Then fill remaining slots
	categoryPriority := []string{"values", "careers", "press", "product", "about", "other"}

	// Group links by category
	categoryMap := make(map[string][]string)
	for _, cl := range classified {
		if cl.URL == homepageURL {
			continue // Skip homepage, already included
		}
		categoryMap[cl.Category] = append(categoryMap[cl.Category], cl.URL)
	}

	selected := make([]string, 0)
	selectedSet := make(map[string]bool)

	// First pass: ensure at least one from each priority category
	for _, category := range categoryPriority {
		if len(selected) >= maxPages-1 { // -1 because homepage is always included
			break
		}
		if urls, exists := categoryMap[category]; exists && len(urls) > 0 {
			// Take first URL from this category
			url := urls[0]
			if !selectedSet[url] {
				selected = append(selected, url)
				selectedSet[url] = true
			}
		}
	}

	// Second pass: fill remaining slots
	for _, category := range categoryPriority {
		if len(selected) >= maxPages-1 {
			break
		}
		if urls, exists := categoryMap[category]; exists {
			for _, url := range urls {
				if len(selected) >= maxPages-1 {
					break
				}
				if !selectedSet[url] {
					selected = append(selected, url)
					selectedSet[url] = true
				}
			}
		}
	}

	return selected
}

// fetchHTML fetches HTML content from a URL (reused from ingestion pattern)
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

// extractTextFromHTML extracts main content text from HTML (reused from ingestion pattern)
func extractTextFromHTML(htmlContent string) (string, error) {
	// Use goquery to extract text (same pattern as ingestion)
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

// computeHash computes SHA256 hash of content and returns hex string
func computeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
