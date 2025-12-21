// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

// RankedStories represents a collection of ranked experience stories
type RankedStories struct {
	Ranked []RankedStory `json:"ranked"`
}

// RankedStory represents a single ranked story with scores and metadata
type RankedStory struct {
	StoryID          string   `json:"story_id"`
	RelevanceScore   float64  `json:"relevance_score"`
	SkillOverlap     float64  `json:"skill_overlap"`
	KeywordOverlap   float64  `json:"keyword_overlap"`
	EvidenceStrength float64  `json:"evidence_strength"`
	MatchedSkills    []string `json:"matched_skills"`
	Notes            string   `json:"notes"`
	// HeuristicScore is the score from deterministic heuristic evaluation
	HeuristicScore float64 `json:"heuristic_score,omitempty"`
	// LLMScore is the score from LLM relevance evaluation (nil if not evaluated)
	LLMScore *float64 `json:"llm_score,omitempty"`
	// LLMReasoning is the LLM's explanation for the score
	LLMReasoning string `json:"llm_reasoning,omitempty"`
}
