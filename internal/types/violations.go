// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

// Violation represents a single validation failure
type Violation struct {
	Type             string   `json:"type"`
	Severity         string   `json:"severity"`
	Details          string   `json:"details"`
	AffectedSections []string `json:"affected_sections,omitempty"`
	LineNumber       *int     `json:"line_number,omitempty"`
	CharCount        *int     `json:"char_count,omitempty"`
}

// Violations represents a collection of validation failures
type Violations struct {
	Violations []Violation `json:"violations"`
}
