// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

// SelectedBullet represents a bullet selected from the experience bank
type SelectedBullet struct {
	ID          string   `json:"id"`
	StoryID     string   `json:"story_id"`
	Text        string   `json:"text"`
	Skills      []string `json:"skills"`
	Metrics     string   `json:"metrics,omitempty"`
	LengthChars int      `json:"length_chars"`
}

// SelectedBullets represents a collection of selected bullets (wrapper for schema)
type SelectedBullets struct {
	Bullets []SelectedBullet `json:"bullets"`
}

