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

	// Fields for tracking which bullet caused the violation
	BulletID   *string `json:"bullet_id,omitempty"`   // Which bullet caused this
	StoryID    *string `json:"story_id,omitempty"`    // Which story contains the bullet
	BulletText *string `json:"bullet_text,omitempty"` // Original bullet text (for context)
}

// Violations represents a collection of validation failures
type Violations struct {
	Violations []Violation `json:"violations"`
}
