// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

// JobProfile represents a structured job posting extracted from raw text
type JobProfile struct {
	Company               string                 `json:"company"`
	RoleTitle             string                 `json:"role_title"`
	Responsibilities      []string               `json:"responsibilities"`
	HardRequirements      []Requirement          `json:"hard_requirements"`
	NiceToHaves           []Requirement          `json:"nice_to_haves"`
	Keywords              []string               `json:"keywords"`
	EvalSignals           *EvalSignals           `json:"eval_signals"`
	EducationRequirements *EducationRequirements `json:"education_requirements,omitempty"`
}

// Requirement represents a skill requirement with evidence
type Requirement struct {
	Skill    string `json:"skill"`
	Level    string `json:"level,omitempty"`
	Evidence string `json:"evidence"`
}

// EvalSignals represents inferred evaluation criteria signals
type EvalSignals struct {
	Latency       bool `json:"latency,omitempty"`
	Reliability   bool `json:"reliability,omitempty"`
	Ownership     bool `json:"ownership,omitempty"`
	Scale         bool `json:"scale,omitempty"`
	Collaboration bool `json:"collaboration,omitempty"`
}

// EducationRequirements represents the education requirements extracted from a job posting
type EducationRequirements struct {
	MinDegree       string   `json:"min_degree,omitempty"`       // bachelor, master, phd, or empty
	PreferredFields []string `json:"preferred_fields,omitempty"` // e.g., ["Computer Science", "Data Science"]
	Evidence        string   `json:"evidence,omitempty"`         // Quote from job posting
	IsRequired      bool     `json:"is_required,omitempty"`      // True if degree is required, false if preferred
}
