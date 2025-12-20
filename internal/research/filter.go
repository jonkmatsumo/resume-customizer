// Package research - filter.go provides LLM-guided link filtering.
package research

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
)

// FilterLinksResult contains the results of link filtering
type FilterLinksResult struct {
	Kept    []RankedURL  `json:"kept"`
	Skipped []SkippedURL `json:"skipped"`
}

// FilterLinks uses LLM to filter and rank links by relevance to the company
func FilterLinks(ctx context.Context, links []string, companyName string, companyDomain string, apiKey string) (*FilterLinksResult, error) {
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

	prompt := buildFilterPrompt(links, companyName, companyDomain)

	jsonResp, err := client.GenerateJSON(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	jsonResp = llm.CleanJSONBlock(jsonResp)

	var result FilterLinksResult
	if err := json.Unmarshal([]byte(jsonResp), &result); err != nil {
		return nil, fmt.Errorf("failed to parse filter response: %w (content: %s)", err, jsonResp)
	}

	return &result, nil
}

func buildFilterPrompt(links []string, companyName string, companyDomain string) string {
	linksList := strings.Join(links, "\n")

	template := prompts.MustGet("research.json", "filter-links")
	return prompts.Format(template, map[string]string{
		"Company": companyName,
		"Domain":  companyDomain,
		"Links":   linksList,
	})
}

// IsThirdParty checks if a URL is from a known third-party platform
func IsThirdParty(url string) bool {
	thirdPartyDomains := []string{
		"greenhouse.io",
		"lever.co",
		"workday.com",
		"myworkdayjobs.com",
		"linkedin.com",
		"indeed.com",
		"glassdoor.com",
		"ziprecruiter.com",
	}

	for _, domain := range thirdPartyDomains {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}
