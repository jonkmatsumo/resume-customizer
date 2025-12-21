// Package research - filter.go provides LLM-guided link filtering.
package research

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
)

// FilterLinksResult contains the results of link filtering
type FilterLinksResult struct {
	Kept    []RankedURL  `json:"kept"`
	Skipped []SkippedURL `json:"skipped"`
}

// identifyDomainsResponse is the expected JSON response from the LLM
type identifyDomainsResponse struct {
	CompanyDomains []string `json:"company_domains"`
}

// IdentifyCompanyDomains uses LLM to identify which domains belong to the company
func IdentifyCompanyDomains(ctx context.Context, urls []string, companyName string, apiKey string) ([]string, error) {
	if len(urls) == 0 {
		return nil, nil
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key required for domain identification")
	}

	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Extract unique domains from URLs
	domains := extractUniqueDomains(urls)
	if len(domains) == 0 {
		return nil, nil
	}

	linksList := strings.Join(domains, "\n")

	template := prompts.MustGet("research.json", "identify-company-domains")
	prompt := prompts.Format(template, map[string]string{
		"Company": companyName,
		"Links":   linksList,
	})

	jsonResp, err := client.GenerateJSON(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	jsonResp = llm.CleanJSONBlock(jsonResp)

	var result identifyDomainsResponse
	if err := json.Unmarshal([]byte(jsonResp), &result); err != nil {
		return nil, fmt.Errorf("failed to parse domain response: %w (content: %s)", err, jsonResp)
	}

	return result.CompanyDomains, nil
}

// extractUniqueDomains extracts unique domain names from a list of URLs
func extractUniqueDomains(urls []string) []string {
	seen := make(map[string]bool)
	var domains []string

	for _, urlStr := range urls {
		domain := extractDomainFromURL(urlStr)
		if domain != "" && !seen[domain] {
			seen[domain] = true
			domains = append(domains, domain)
		}
	}

	return domains
}

// extractDomainFromURL extracts the domain from a URL
func extractDomainFromURL(urlStr string) string {
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

	return host
}

// FilterToCompanyDomains filters URLs to only include those from company domains
func FilterToCompanyDomains(urls []string, companyDomains []string) []string {
	if len(companyDomains) == 0 {
		return urls // No filtering if no domains identified
	}

	var filtered []string
	for _, urlStr := range urls {
		if IsFromCompanyDomain(urlStr, companyDomains) {
			filtered = append(filtered, urlStr)
		}
	}

	return filtered
}

// IsFromCompanyDomain checks if a URL is from one of the company domains
func IsFromCompanyDomain(urlStr string, companyDomains []string) bool {
	urlDomain := extractDomainFromURL(urlStr)
	if urlDomain == "" {
		return false
	}

	urlDomainLower := strings.ToLower(urlDomain)
	for _, companyDomain := range companyDomains {
		companyDomainLower := strings.ToLower(companyDomain)
		// Check exact match or subdomain match
		if urlDomainLower == companyDomainLower ||
			strings.HasSuffix(urlDomainLower, "."+companyDomainLower) {
			return true
		}
	}

	return false
}

// AssignPathPriority returns a priority boost based on URL path patterns
func AssignPathPriority(urlStr string) float64 {
	urlLower := strings.ToLower(urlStr)

	// Highest priority: leadership principles, values, mission
	highValuePatterns := []string{
		"leadership-principles", "values", "mission-and-values",
		"mission", "principles", "culture-memo", "our-values",
	}
	for _, pattern := range highValuePatterns {
		if strings.Contains(urlLower, pattern) {
			return 0.95
		}
	}

	// High priority: culture, about, careers, engineering
	goodPatterns := []string{
		"culture", "about", "careers", "engineering", "blog/engineering",
		"who-we-are", "our-story", "team",
	}
	for _, pattern := range goodPatterns {
		if strings.Contains(urlLower, pattern) {
			return 0.85
		}
	}

	// Medium priority: press, news
	mediumPatterns := []string{"press", "news", "announcements"}
	for _, pattern := range mediumPatterns {
		if strings.Contains(urlLower, pattern) {
			return 0.7
		}
	}

	// Low priority patterns to skip: promotional/product pages
	skipPatterns := []string{
		"/p/", "/product/", "/catering", "/delivery", "/alcohol",
		"/chips", "/stores", "/near-me", "/order",
	}
	for _, pattern := range skipPatterns {
		if strings.Contains(urlLower, pattern) {
			return 0.1 // Very low priority
		}
	}

	// Default priority
	return 0.5
}

// FilterLinks uses LLM to filter and rank links by relevance to the company
func FilterLinks(ctx context.Context, links []string, companyName string, companyDomains string, apiKey string) (*FilterLinksResult, error) {
	if len(links) == 0 {
		return &FilterLinksResult{}, nil
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key required for link filtering")
	}

	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}
	defer func() { _ = client.Close() }()

	prompt := buildFilterPrompt(links, companyName, companyDomains)

	jsonResp, err := client.GenerateJSON(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	jsonResp = llm.CleanJSONBlock(jsonResp)

	var result FilterLinksResult
	if err := json.Unmarshal([]byte(jsonResp), &result); err != nil {
		return nil, fmt.Errorf("failed to parse filter response: %w (content: %s)", err, jsonResp)
	}

	// Apply path-based priority boost to results
	for i := range result.Kept {
		pathPriority := AssignPathPriority(result.Kept[i].URL)
		if pathPriority > result.Kept[i].Priority {
			result.Kept[i].Priority = pathPriority
		}
	}

	return &result, nil
}

func buildFilterPrompt(links []string, companyName string, companyDomains string) string {
	linksList := strings.Join(links, "\n")

	template := prompts.MustGet("research.json", "filter-links")
	return prompts.Format(template, map[string]string{
		"Company": companyName,
		"Domain":  companyDomains,
		"Links":   linksList,
	})
}

// IsThirdParty checks if a URL is from a known third-party platform
func IsThirdParty(urlStr string) bool {
	thirdPartyDomains := []string{
		"greenhouse.io",
		"lever.co",
		"workday.com",
		"myworkdayjobs.com",
		"linkedin.com",
		"indeed.com",
		"glassdoor.com",
		"ziprecruiter.com",
		"getcovey.com",
		"usa.gov",
		"go.usa.gov",
		"medium.com",
	}

	urlLower := strings.ToLower(urlStr)
	for _, domain := range thirdPartyDomains {
		if strings.Contains(urlLower, domain) {
			return true
		}
	}
	return false
}
