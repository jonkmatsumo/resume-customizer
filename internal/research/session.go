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
		Company:        opts.Company,
		Domain:         opts.Domain,
		CrawledURLs:    []string{},
		Frontier:       []RankedURL{},
		SkippedURLs:    []SkippedURL{},
		BrandSignals:   []BrandSignal{},
		Corpus:         opts.InitialCorpus,
		CompanyDomains: []string{},
	}

	// Step 1: Identify company domains from seed URLs
	if opts.Verbose {
		log.Printf("[RESEARCH] Identifying company domains from %d seed URLs...", len(opts.SeedURLs))
	}

	companyDomains, err := IdentifyCompanyDomains(ctx, opts.SeedURLs, opts.Company, opts.APIKey)
	if err != nil {
		if opts.Verbose {
			log.Printf("[RESEARCH] Domain identification failed: %v, falling back to provided domain", err)
		}
		// Fallback to the provided domain if identification fails
		if opts.Domain != "" {
			companyDomains = []string{opts.Domain}
		}
	}

	if len(companyDomains) > 0 {
		session.CompanyDomains = companyDomains
		if opts.Verbose {
			log.Printf("[RESEARCH] Identified company domains: %v", companyDomains)
		}
	}

	// Step 2: Pre-filter seeds to only company domains (if we have them)
	filteredSeeds := opts.SeedURLs
	if len(companyDomains) > 0 {
		filteredSeeds = FilterToCompanyDomains(opts.SeedURLs, companyDomains)
		if opts.Verbose {
			log.Printf("[RESEARCH] Pre-filtered from %d to %d company domain URLs",
				len(opts.SeedURLs), len(filteredSeeds))
		}

		// Track skipped non-company URLs
		for _, u := range opts.SeedURLs {
			if !IsFromCompanyDomain(u, companyDomains) && !IsThirdParty(u) {
				session.SkippedURLs = append(session.SkippedURLs, SkippedURL{
					URL:    u,
					Reason: "not company domain",
				})
			} else if IsThirdParty(u) {
				session.SkippedURLs = append(session.SkippedURLs, SkippedURL{
					URL:    u,
					Reason: "third-party platform",
				})
			}
		}
	}

	// Step 3: LLM filter for relevance + priority
	if opts.Verbose {
		log.Printf("[RESEARCH] Filtering %d URLs for relevance...", len(filteredSeeds))
	}

	domainsStr := strings.Join(companyDomains, ", ")
	filterResult, err := FilterLinks(ctx, filteredSeeds, opts.Company, domainsStr, opts.APIKey)
	if err != nil {
		// Fallback to basic filtering with path priority
		if opts.Verbose {
			log.Printf("[RESEARCH] LLM filtering failed: %v, using path-based priority", err)
		}
		for _, u := range filteredSeeds {
			if IsThirdParty(u) {
				session.SkippedURLs = append(session.SkippedURLs, SkippedURL{URL: u, Reason: "third-party"})
			} else {
				priority := AssignPathPriority(u)
				session.Frontier = append(session.Frontier, RankedURL{
					URL:      u,
					Priority: priority,
					Reason:   "fallback path-based",
				})
			}
		}
	} else {
		session.Frontier = filterResult.Kept
		session.SkippedURLs = append(session.SkippedURLs, filterResult.Skipped...)
	}

	if opts.Verbose {
		log.Printf("[RESEARCH] After filtering: kept %d URLs, skipped %d",
			len(session.Frontier), len(session.SkippedURLs))
	}

	// Step 4: Add high-value pattern URLs for company domains
	if len(companyDomains) > 0 {
		patternURLs := generateHighValueURLs(companyDomains)
		for _, pu := range patternURLs {
			if !isInList(pu.URL, session.Frontier) && !isInList(pu.URL, session.CrawledURLs) {
				session.Frontier = append(session.Frontier, pu)
			}
		}
		if opts.Verbose {
			log.Printf("[RESEARCH] Added %d high-value pattern URLs", len(patternURLs))
		}
	}

	// Step 5: Sort frontier by priority (highest first)
	sortFrontierByPriority(session)

	if opts.Verbose && len(session.Frontier) > 0 {
		log.Printf("[RESEARCH] Top 5 frontier URLs:")
		for i, u := range session.Frontier {
			if i >= 5 {
				break
			}
			log.Printf("  [%.2f] %s (%s)", u.Priority, u.URL, u.Type)
		}
	}

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

// generateHighValueURLs creates URLs for common high-value paths across company domains
func generateHighValueURLs(companyDomains []string) []RankedURL {
	patterns := HighValuePatterns()
	var results []RankedURL

	for _, domain := range companyDomains {
		for pattern, priority := range patterns {
			expectedURL := fmt.Sprintf("https://%s/%s", domain, pattern)
			results = append(results, RankedURL{
				URL:      expectedURL,
				Priority: priority,
				Reason:   "expected high-value pattern: " + pattern,
				Type:     categorizePattern(pattern),
			})
		}
	}

	return results
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
	case strings.Contains(pattern, "mission"):
		return "values"
	case strings.Contains(pattern, "careers"):
		return "careers"
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
