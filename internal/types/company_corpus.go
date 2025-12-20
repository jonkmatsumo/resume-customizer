// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

// Source represents a single crawled page with metadata
type Source struct {
	URL       string `json:"url"`
	Timestamp string `json:"timestamp"` // RFC3339 format
	Hash      string `json:"hash"`      // SHA256 hex digest
}

// CompanyCorpus represents the collected corpus with sources
// Note: The Corpus field is primarily for testing/metadata purposes.
// The CLI writes the corpus text directly to a .txt file.
type CompanyCorpus struct {
	Corpus  string   `json:"corpus"` // Concatenated text (for JSON representation, but primary output is .txt file)
	Sources []Source `json:"sources"`
}
