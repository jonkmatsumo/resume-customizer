package research

import (
	"context"
	"fmt"
	"strings"

	"github.com/jonathan/resume-customizer/internal/types"
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
)

// Researcher handles external company research
type Researcher struct {
	svc *customsearch.Service
	cx  string
}

// NewResearcher creates a new Researcher instance
func NewResearcher(ctx context.Context, apiKey string, cx string) (*Researcher, error) {
	svc, err := customsearch.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create customsearch service: %w", err)
	}
	return &Researcher{
		svc: svc,
		cx:  cx,
	}, nil
}

// DiscoverCompanyWebsite attempts to find the company's main website URL
func (r *Researcher) DiscoverCompanyWebsite(ctx context.Context, jobProfile *types.JobProfile) (string, error) {
	// 1. Check if URL is already in the parsed text (if we had it, but mostly we don't in structured profile)
	// For now, assume it's not and we need to search.

	query := fmt.Sprintf("%s official website", jobProfile.Company)
	resp, err := r.svc.Cse.List().Cx(r.cx).Q(query).Do()
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(resp.Items) == 0 {
		return "", fmt.Errorf("no search results found for %s", jobProfile.Company)
	}

	// Return the first result's link
	return resp.Items[0].Link, nil
}

// FindVoiceSeeds discovers relevant pages for analyzing brand voice (Careers, Culture, Blog)
// It accepts a base company name and/or website.
func (r *Researcher) FindVoiceSeeds(ctx context.Context, companyName string, websiteURL string) ([]string, error) {
	var seeds []string

	// Always include the main website if provided
	if websiteURL != "" {
		seeds = append(seeds, websiteURL)
	}

	// Queries to find specific pages
	queries := []string{
		fmt.Sprintf("site:%s culture values", getDomain(websiteURL)),
		fmt.Sprintf("site:%s engineering blog", getDomain(websiteURL)),
		fmt.Sprintf("%s company values mission", companyName),
		fmt.Sprintf("%s engineering culture principles", companyName),
	}

	for _, q := range queries {
		// Be gentle with rate limits if needed, but standard quota is okay for low volume
		resp, err := r.svc.Cse.List().Cx(r.cx).Q(q).Num(3).Do() // Get top 3 for each
		if err != nil {
			continue // Skip failed queries gracefully
		}

		for _, item := range resp.Items {
			seeds = append(seeds, item.Link)
		}
	}

	// Dedup
	uniqueSeeds := make([]string, 0)
	seen := make(map[string]bool)
	for _, s := range seeds {
		if !seen[s] {
			uniqueSeeds = append(uniqueSeeds, s)
			seen[s] = true
		}
	}

	return uniqueSeeds, nil
}

func getDomain(url string) string {
	// improved domain extraction could go here
	// for now, simple strip
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return url
}
