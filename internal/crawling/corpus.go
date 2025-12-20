package crawling

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"time"

	"github.com/jonathan/resume-customizer/internal/fetch"
	"github.com/jonathan/resume-customizer/internal/ingestion"
	"github.com/jonathan/resume-customizer/internal/types"
)

const (
	// MaxPagesLimit is the hard maximum number of pages to crawl
	MaxPagesLimit = 15
	// DefaultRateLimitDelay is the delay between HTTP requests
	DefaultRateLimitDelay = 1 * time.Second
)

// CrawlBrandCorpus crawls a company website and builds a text corpus.
// It now delegates to the generic fetch package for URL fetching and HTML extraction.
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

		// Fetch seed page using the generic fetch package
		result, err := fetch.URL(ctx, seed, nil)
		if err != nil {
			// Log error but continue
			continue
		}
		visited[seed] = true

		// Add text to corpus using company page selectors
		text, err := fetch.ExtractMainText(result.HTML, fetch.CompanyPageSelectors())
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
		pageLinks, err := ExtractLinks(result.HTML, seed)
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

					result, err := fetch.URL(ctx, pageURL, nil)
					if err != nil {
						continue
					}

					text, err := fetch.ExtractMainText(result.HTML, fetch.CompanyPageSelectors())
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
			u := urls[0]
			if !selectedSet[u] {
				selected = append(selected, u)
				selectedSet[u] = true
			}
		}
	}

	// Second pass: fill remaining slots
	for _, category := range categoryPriority {
		if len(selected) >= maxPages-1 {
			break
		}
		if urls, exists := categoryMap[category]; exists {
			for _, u := range urls {
				if len(selected) >= maxPages-1 {
					break
				}
				if !selectedSet[u] {
					selected = append(selected, u)
					selectedSet[u] = true
				}
			}
		}
	}

	return selected
}

// computeHash computes SHA256 hash of content and returns hex string
func computeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
