// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

// RepairAction represents a single repair action to fix a violation
type RepairAction struct {
	Type           string                 `json:"type"`
	BulletID       string                 `json:"bullet_id,omitempty"`
	StoryID        string                 `json:"story_id,omitempty"`
	TargetChars    *int                   `json:"target_chars,omitempty"`
	Section        string                 `json:"section,omitempty"`
	TemplateParams map[string]interface{} `json:"template_params,omitempty"`
	Reason         string                 `json:"reason"`
}

// RepairActions represents a collection of repair actions
type RepairActions struct {
	Actions []RepairAction `json:"actions"`
}
