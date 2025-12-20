// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

// CompanyProfile represents brand voice and style rules for a company
type CompanyProfile struct {
	Company       string   `json:"company"`
	Tone          string   `json:"tone"`
	StyleRules    []string `json:"style_rules"`
	TabooPhrases  []string `json:"taboo_phrases"`
	DomainContext string   `json:"domain_context"`
	Values        []string `json:"values"`
	EvidenceURLs  []string `json:"evidence_urls"`
}
