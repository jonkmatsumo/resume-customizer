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

// StyleChecks represents style validation results for a rewritten bullet
type StyleChecks struct {
	StrongVerb   bool `json:"strong_verb"`
	Quantified   bool `json:"quantified"`
	NoTaboo      bool `json:"no_taboo"`
	TargetLength bool `json:"target_length"`
}

// RewrittenBullet represents a rewritten bullet with style validation
type RewrittenBullet struct {
	OriginalBulletID string      `json:"original_bullet_id"`
	FinalText        string      `json:"final_text"`
	LengthChars      int         `json:"length_chars"`
	EstimatedLines   int         `json:"estimated_lines"`
	StyleChecks      StyleChecks `json:"style_checks"`
}

// RewrittenBullets represents a collection of rewritten bullets (wrapper for schema)
type RewrittenBullets struct {
	Bullets []RewrittenBullet `json:"bullets"`
}
