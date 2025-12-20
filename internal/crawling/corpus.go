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
func CrawlBrandCorpus(ctx context.Context, seedURLs []string, maxPages int, apiKey string) (*types.CompanyCorpus, error) {
	if len(seedURLs) == 0 {
		return nil, &CrawlError{
			Message: "no seed URLs provided",
		}
	}

	// Validate all seed URLs
	validSeeds := make([]string, 0)
	for _, seed := range seedURLs {
		parsedURL, err := url.Parse(seed)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			continue // Skip invalid seeds
		}
		validSeeds = append(validSeeds, seed)
	}

	if len(validSeeds) == 0 {
		return nil, &CrawlError{
			Message: "no valid seed URLs provided",
		}
	}

	// Enforce max pages limit
	if maxPages > MaxPagesLimit {
		maxPages = MaxPagesLimit
	}
	if maxPages < 1 {
		maxPages = 10 // Default
	}

	var corpusParts []string
	sources := make([]types.Source, 0)
	visited := make(map[string]bool)
	allLinks := make([]string, 0)

	// Phase 1: Fetch all seeds first
	for _, seed := range validSeeds {
		if visited[seed] {
			continue
		}

		// Fetch seed page
		html, err := fetchHTML(seed)
		if err != nil {
			// Log error but continue
			continue
		}
		visited[seed] = true

		// Add text to corpus
		text, err := extractTextFromHTML(html)
		if err == nil {
			cleanedText := ingestion.CleanText(text)
			hash := computeHash(cleanedText)
			corpusParts = append(corpusParts, cleanedText)
			sources = append(sources, types.Source{
				URL:       seed,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Hash:      hash,
			})
		}

		// Extract links for Phase 2
		pageLinks, err := ExtractLinks(html, seed)
		if err == nil {
			allLinks = append(allLinks, pageLinks...)
		}
	}

	// If we've reached maxPages just with seeds, return early
	if len(sources) >= maxPages {
		corpus := strings.Join(corpusParts, "\n\n---\n\n")
		return &types.CompanyCorpus{
			Corpus:  corpus,
			Sources: sources,
		}, nil
	}

	// Phase 2: Classify and select more pages if needed
	if len(allLinks) > 0 {
		// Limit candidate links to top 50, deduped
		uniqueLinks := make([]string, 0)
		linkSeen := make(map[string]bool)
		for _, l := range allLinks {
			if !linkSeen[l] && !visited[l] {
				uniqueLinks = append(uniqueLinks, l)
				linkSeen[l] = true
			}
		}

		maxCandidates := 30
		if len(uniqueLinks) > maxCandidates {
			uniqueLinks = uniqueLinks[:maxCandidates]
		}

		if len(uniqueLinks) > 0 {
			classified, err := ClassifyLinks(ctx, uniqueLinks, apiKey)
			if err == nil {
				// Use the first valid seed as "homepage" for exclusion logic (best effort)
				selectedURLs := selectPages(classified, maxPages-len(sources), validSeeds[0])

				for _, pageURL := range selectedURLs {
					if visited[pageURL] {
						continue
					}
					visited[pageURL] = true

					time.Sleep(DefaultRateLimitDelay)

					html, err := fetchHTML(pageURL)
					if err != nil {
						continue
					}

					text, err := extractTextFromHTML(html)
					if err == nil {
						cleanedText := ingestion.CleanText(text)
						hash := computeHash(cleanedText)
						corpusParts = append(corpusParts, cleanedText)
						sources = append(sources, types.Source{
							URL:       pageURL,
							Timestamp: time.Now().UTC().Format(time.RFC3339),
							Hash:      hash,
						})
					}
				}
			}
		}
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
