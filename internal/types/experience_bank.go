// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

// ExperienceBank represents a canonical store of reusable experience stories
type ExperienceBank struct {
	Stories []Story `json:"stories"`
}

// Story represents a single work experience story with stable ID
type Story struct {
	ID        string   `json:"id"`
	Company   string   `json:"company"`
	Role      string   `json:"role"`
	StartDate string   `json:"start_date"`
	EndDate   string   `json:"end_date"`
	Bullets   []Bullet `json:"bullets"`
}

// Bullet represents a single bullet point with skills, metrics, and metadata
type Bullet struct {
	ID               string   `json:"id"`
	Text             string   `json:"text"`
	Skills           []string `json:"skills"`
	Metrics          string   `json:"metrics,omitempty"`
	LengthChars      int      `json:"length_chars"`
	EvidenceStrength string   `json:"evidence_strength"`
	RiskFlags        []string `json:"risk_flags"`
}
