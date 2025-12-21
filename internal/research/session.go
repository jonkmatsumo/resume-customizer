// Package research - session.go provides iterative research capabilities.
package research

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/jonathan/resume-customizer/internal/fetch"
	"github.com/jonathan/resume-customizer/internal/ingestion"
	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
)

// RunResearchOptions configures the research session
type RunResearchOptions struct {
	SeedURLs      []string
	Company       string
	Domain        string
	InitialCorpus string // Pre-extracted company context (e.g., "About Us" from job post)
	MaxPages      int
	APIKey        string
	Verbose       bool
	UseBrowser    bool
}

// RunResearch executes an iterative research loop to build company corpus
func RunResearch(ctx context.Context, opts RunResearchOptions) (*Session, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("API key required for research")
	}

	session := &Session{
		Company:      opts.Company,
		Domain:       opts.Domain,
		CrawledURLs:  []string{},
		Frontier:     []RankedURL{},
		SkippedURLs:  []SkippedURL{},
		BrandSignals: []BrandSignal{},
		Corpus:       opts.InitialCorpus,
	}

	// Filter seed URLs first
	if opts.Verbose {
		log.Printf("[RESEARCH] Filtering %d seed URLs...", len(opts.SeedURLs))
	}

	filterResult, err := FilterLinks(ctx, opts.SeedURLs, opts.Company, opts.Domain, opts.APIKey)
	if err != nil {
		// Fallback to basic filtering
		for _, u := range opts.SeedURLs {
			if IsThirdParty(u) {
				session.SkippedURLs = append(session.SkippedURLs, SkippedURL{URL: u, Reason: "third-party"})
			} else {
				session.Frontier = append(session.Frontier, RankedURL{URL: u, Priority: 0.5, Reason: "seed"})
			}
		}
	} else {
		session.Frontier = filterResult.Kept
		session.SkippedURLs = filterResult.Skipped
	}

	if opts.Verbose {
		log.Printf("[RESEARCH] Kept %d URLs, skipped %d", len(session.Frontier), len(session.SkippedURLs))
	}

	// Add search-based URLs (also filter them)
	searchURLs, err := searchHighValuePages(ctx, opts.Company, opts.Domain, opts.APIKey)
	if err == nil && len(searchURLs) > 0 {
		// Convert to string list for filtering
		searchLinks := make([]string, 0, len(searchURLs))
		for _, u := range searchURLs {
			if !isInList(u.URL, session.Frontier) && !isInList(u.URL, session.CrawledURLs) {
				searchLinks = append(searchLinks, u.URL)
			}
		}

		// Filter search results through LLM
		if len(searchLinks) > 0 {
			filterResult, err = FilterLinks(ctx, searchLinks, opts.Company, opts.Domain, opts.APIKey)
			if err == nil {
				for _, kept := range filterResult.Kept {
					if !isInList(kept.URL, session.Frontier) {
						session.Frontier = append(session.Frontier, kept)
					}
				}
				session.SkippedURLs = append(session.SkippedURLs, filterResult.Skipped...)
				if opts.Verbose {
					log.Printf("[RESEARCH] Filtered search URLs: kept %d, skipped %d",
						len(filterResult.Kept), len(filterResult.Skipped))
				}
			} else {
				// Fallback: use basic third-party filtering
				for _, u := range searchURLs {
					if !isInList(u.URL, session.Frontier) && !IsThirdParty(u.URL) {
						session.Frontier = append(session.Frontier, u)
					}
				}
			}
		}
	}

	// Sort frontier by priority
	sortFrontierByPriority(session)

	// Crawl loop
	pagesProcessed := 0
	for pagesProcessed < opts.MaxPages && len(session.Frontier) > 0 {
		// Get highest priority URL
		target := session.Frontier[0]
		session.Frontier = session.Frontier[1:]

		if isInList(target.URL, session.CrawledURLs) {
			continue
		}

		if opts.Verbose {
			log.Printf("[RESEARCH] Crawling %s (priority: %.2f, type: %s)", target.URL, target.Priority, target.Type)
		}

		// Fetch page
		html, err := fetchPage(ctx, target.URL, opts.UseBrowser, opts.Verbose)
		if err != nil {
			if opts.Verbose {
				log.Printf("[RESEARCH] Failed to fetch %s: %v", target.URL, err)
			}
			continue
		}

		// Extract text
		text, err := fetch.ExtractMainText(html, fetch.CompanyPageSelectors())
		if err != nil {
			continue
		}
		text = ingestion.CleanText(text)

		if len(text) < 100 {
			if opts.Verbose {
				log.Printf("[RESEARCH] Skipping %s - insufficient content (%d chars)", target.URL, len(text))
			}
			continue
		}

		// Extract brand signals
		signal, err := ExtractBrandSignals(ctx, text, target.URL, opts.APIKey)
		if err == nil && signal != nil {
			session.BrandSignals = append(session.BrandSignals, *signal)
			if opts.Verbose {
				log.Printf("[RESEARCH] Extracted %d key points from %s", len(signal.KeyPoints), target.URL)
			}
		}

		session.CrawledURLs = append(session.CrawledURLs, target.URL)
		pagesProcessed++

		// Rate limiting
		time.Sleep(500 * time.Millisecond)
	}

	// Aggregate corpus from signals
	session.Corpus = AggregateSignals(session.BrandSignals)

	if opts.Verbose {
		log.Printf("[RESEARCH] Complete: crawled %d pages, extracted %d signals, corpus: %d chars",
			len(session.CrawledURLs), len(session.BrandSignals), len(session.Corpus))
	}

	return session, nil
}

func fetchPage(ctx context.Context, pageURL string, useBrowser bool, verbose bool) (string, error) {
	result, err := fetch.URL(ctx, pageURL, nil)
	if err != nil {
		return "", err
	}

	// Check if we need browser fallback
	text, _ := fetch.ExtractMainText(result.HTML, fetch.CompanyPageSelectors())
	if useBrowser && fetch.ShouldUseBrowser(text) {
		return fetch.BrowserSimple(ctx, pageURL, verbose)
	}

	return result.HTML, nil
}

func searchHighValuePages(_ context.Context, _ string, domain string, _ string) ([]RankedURL, error) {
	if domain == "" {
		return nil, nil
	}
	// This would use Google Custom Search API in production
	// For now, generate expected URLs based on patterns
	var results []RankedURL

	patterns := HighValuePatterns()
	for pattern, priority := range patterns {
		expectedURL := fmt.Sprintf("https://%s/%s", domain, pattern)
		results = append(results, RankedURL{
			URL:      expectedURL,
			Priority: priority,
			Reason:   "expected pattern: " + pattern,
			Type:     categorizePattern(pattern),
		})
	}

	return results, nil
}

func categorizePattern(pattern string) string {
	switch {
	case strings.Contains(pattern, "values") || strings.Contains(pattern, "principles"):
		return "values"
	case strings.Contains(pattern, "culture"):
		return "culture"
	case strings.Contains(pattern, "engineering") || strings.Contains(pattern, "blog"):
		return "engineering"
	case strings.Contains(pattern, "about"):
		return "about"
	default:
		return "other"
	}
}

func isInList(urlStr string, list interface{}) bool {
	switch v := list.(type) {
	case []string:
		for _, u := range v {
			if u == urlStr {
				return true
			}
		}
	case []RankedURL:
		for _, u := range v {
			if u.URL == urlStr {
				return true
			}
		}
	}
	return false
}

func sortFrontierByPriority(session *Session) {
	// Simple bubble sort for small lists
	for i := 0; i < len(session.Frontier); i++ {
		for j := i + 1; j < len(session.Frontier); j++ {
			if session.Frontier[j].Priority > session.Frontier[i].Priority {
				session.Frontier[i], session.Frontier[j] = session.Frontier[j], session.Frontier[i]
			}
		}
	}
}

// SuggestNextQueries uses LLM to suggest what to search for next
func SuggestNextQueries(ctx context.Context, session *Session, apiKey string) ([]string, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key required")
	}

	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Close() }()

	template := prompts.MustGet("research.json", "suggest-search-queries")
	prompt := prompts.Format(template, map[string]string{
		"Company":         session.Company,
		"CurrentFindings": summarizeFindingsBrief(session),
	})

	jsonResp, err := client.GenerateJSON(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, err
	}

	jsonResp = llm.CleanJSONBlock(jsonResp)

	var queries []string
	if err := json.Unmarshal([]byte(jsonResp), &queries); err != nil {
		return nil, err
	}

	return queries, nil
}

func summarizeFindingsBrief(session *Session) string {
	var summary string
	for _, signal := range session.BrandSignals {
		summary += fmt.Sprintf("- %s (%s): %d key points\n", signal.URL, signal.Type, len(signal.KeyPoints))
	}
	return summary
}

// ExtractDomain extracts the domain from a URL. It handles schemeless URLs by prepending https://.
func ExtractDomain(urlStr string) string {
	if urlStr == "" {
		return ""
	}

	// Prepend scheme if missing
	if !strings.Contains(urlStr, "://") {
		urlStr = "https://" + urlStr
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	host := parsed.Host
	host = strings.TrimPrefix(host, "www.")

	// Handle subdomains for job boards
	hostLower := strings.ToLower(host)
	if strings.Contains(hostLower, "greenhouse.io") ||
		strings.Contains(hostLower, "lever.co") ||
		strings.Contains(hostLower, "workday.com") ||
		strings.Contains(hostLower, "myworkdayjobs.com") {
		return ""
	}

	return host
}
