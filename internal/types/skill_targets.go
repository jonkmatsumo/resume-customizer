// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

// SkillTargets represents a weighted list of target skills
type SkillTargets struct {
	Skills []Skill `json:"skills"`
}

// Skill represents a single target skill with weight and source
type Skill struct {
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	Source      string  `json:"source"`
	Specificity float64 `json:"specificity,omitempty"` // 0.0 (generic) to 1.0 (highly specific)
}
