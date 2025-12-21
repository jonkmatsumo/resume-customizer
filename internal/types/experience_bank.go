// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

// ExperienceBank represents a canonical store of reusable experience stories and education
type ExperienceBank struct {
	Stories   []Story     `json:"stories"`
	Education []Education `json:"education,omitempty"`
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

// Education represents an educational entry (degree, school, etc.)
type Education struct {
	ID         string   `json:"id"`
	School     string   `json:"school"`
	Degree     string   `json:"degree"` // bachelor, master, phd, associate, other
	Field      string   `json:"field"`  // e.g., "Computer Science"
	StartDate  string   `json:"start_date,omitempty"`
	EndDate    string   `json:"end_date,omitempty"`
	GPA        string   `json:"gpa,omitempty"`
	Highlights []string `json:"highlights,omitempty"` // Scholarships, research, achievements
}
