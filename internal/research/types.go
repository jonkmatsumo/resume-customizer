// Package research provides types for company research and brand discovery.
package research

// Session tracks an iterative research process
type Session struct {
	// Company info
	Company string `json:"company"`
	Domain  string `json:"domain"`

	// URL management
	CrawledURLs []string     `json:"crawled_urls"`
	Frontier    []RankedURL  `json:"frontier"`     // URLs to try next
	SkippedURLs []SkippedURL `json:"skipped_urls"` // Filtered out

	// Extracted content
	BrandSignals []BrandSignal `json:"brand_signals"`
	Corpus       string        `json:"corpus"`
}

// RankedURL is a URL with priority for crawl ordering
type RankedURL struct {
	URL      string  `json:"url"`
	Priority float64 `json:"priority"` // 0.0-1.0, higher = more relevant
	Reason   string  `json:"reason"`   // Why it's relevant
	Type     string  `json:"type"`     // values, culture, engineering, press, other
}

// SkippedURL is a URL that was filtered out
type SkippedURL struct {
	URL    string `json:"url"`
	Reason string `json:"reason"` // third-party, irrelevant, promotional
}

// BrandSignal represents extracted brand information from a single page
type BrandSignal struct {
	URL       string   `json:"url"`
	Type      string   `json:"type"` // values, culture, engineering, press
	KeyPoints []string `json:"key_points"`
	Values    []string `json:"values,omitempty"` // Inferred values
}

// HighValuePatterns returns URL path patterns that indicate high-value pages
func HighValuePatterns() map[string]float64 {
	return map[string]float64{
		"leadership-principles": 1.0,
		"culture":               0.9,
		"values":                0.9,
		"principles":            0.9,
		"culture-memo":          0.9,
		"engineering":           0.8,
		"blog/engineering":      0.8,
		"about":                 0.7,
		"mission":               0.8,
		"careers":               0.6,
	}
}

// SearchQueries returns search queries for finding high-value company pages
func SearchQueries(companyName string, domain string) []string {
	return []string{
		companyName + " leadership principles",
		companyName + " culture memo",
		companyName + " values principles",
		"site:" + domain + " culture values",
		"site:" + domain + " engineering blog",
		companyName + " company values mission",
		companyName + " engineering culture",
	}
}
