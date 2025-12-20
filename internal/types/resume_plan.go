// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

// ResumePlan represents a selection contract defining which stories and bullets to use
type ResumePlan struct {
	SelectedStories []SelectedStory `json:"selected_stories"`
	SpaceBudget     SpaceBudget     `json:"space_budget"`
	Coverage        Coverage        `json:"coverage"`
}

// SelectedStory represents a selected story with its bullet IDs and metadata
type SelectedStory struct {
	StoryID        string   `json:"story_id"`
	BulletIDs      []string `json:"bullet_ids"`
	Section        string   `json:"section"`
	EstimatedLines int      `json:"estimated_lines"`
}

// SpaceBudget represents space budget constraints for the resume
type SpaceBudget struct {
	MaxBullets int            `json:"max_bullets"`
	MaxLines   int            `json:"max_lines"`
	Sections   map[string]int `json:"sections,omitempty"`
}

// Coverage represents skill coverage metrics for the selected plan
type Coverage struct {
	TopSkillsCovered []string `json:"top_skills_covered"`
	CoverageScore    float64  `json:"coverage_score"`
}
