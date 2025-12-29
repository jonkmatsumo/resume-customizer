package db

import (
	"time"

	"github.com/google/uuid"
)

// ViolationSeverity constants
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// ViolationType constants for common violation types
const (
	ViolationPageOverflow  = "page_overflow"
	ViolationLineTooLong   = "line_too_long"
	ViolationMissingSkill  = "missing_skill"
	ViolationTooFewBullets = "too_few_bullets"
	ViolationTooManyPages  = "too_many_pages"
)

// Section constants
const (
	SectionExperience = "experience"
	SectionProjects   = "projects"
	SectionEducation  = "education"
	SectionSkills     = "skills"
)

// RunRankedStory represents a ranked story for a specific pipeline run
type RunRankedStory struct {
	ID               uuid.UUID  `json:"id"`
	RunID            uuid.UUID  `json:"run_id"`
	StoryID          *uuid.UUID `json:"story_id,omitempty"` // FK to stories table
	StoryIDText      string     `json:"story_id_text"`      // original string ID
	RelevanceScore   *float64   `json:"relevance_score,omitempty"`
	SkillOverlap     *float64   `json:"skill_overlap,omitempty"`
	KeywordOverlap   *float64   `json:"keyword_overlap,omitempty"`
	EvidenceStrength *float64   `json:"evidence_strength,omitempty"`
	HeuristicScore   *float64   `json:"heuristic_score,omitempty"`
	LLMScore         *float64   `json:"llm_score,omitempty"`
	LLMReasoning     *string    `json:"llm_reasoning,omitempty"`
	MatchedSkills    []string   `json:"matched_skills,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
	Ordinal          int        `json:"ordinal"`
	CreatedAt        time.Time  `json:"created_at"`
}

// RunResumePlan represents a resume plan for a specific pipeline run
type RunResumePlan struct {
	ID               uuid.UUID      `json:"id"`
	RunID            uuid.UUID      `json:"run_id"`
	MaxBullets       int            `json:"max_bullets"`
	MaxLines         int            `json:"max_lines"`
	SkillMatchRatio  float64        `json:"skill_match_ratio"`
	SectionBudgets   map[string]int `json:"section_budgets,omitempty"`
	TopSkillsCovered []string       `json:"top_skills_covered,omitempty"`
	CoverageScore    float64        `json:"coverage_score"`
	CreatedAt        time.Time      `json:"created_at"`
}

// RunSelectedBullet represents a selected bullet for a specific pipeline run
type RunSelectedBullet struct {
	ID           uuid.UUID  `json:"id"`
	RunID        uuid.UUID  `json:"run_id"`
	PlanID       *uuid.UUID `json:"plan_id,omitempty"`
	BulletID     *uuid.UUID `json:"bullet_id,omitempty"` // FK to bullets table
	BulletIDText string     `json:"bullet_id_text"`      // original string ID
	StoryID      *uuid.UUID `json:"story_id,omitempty"`  // FK to stories table
	StoryIDText  string     `json:"story_id_text"`       // original string ID
	Text         string     `json:"text"`
	Skills       []string   `json:"skills,omitempty"`
	Metrics      *string    `json:"metrics,omitempty"`
	LengthChars  int        `json:"length_chars"`
	Section      string     `json:"section"`
	Ordinal      int        `json:"ordinal"`
	CreatedAt    time.Time  `json:"created_at"`
}

// RunRewrittenBullet represents a rewritten bullet for a specific pipeline run
type RunRewrittenBullet struct {
	ID                   uuid.UUID  `json:"id"`
	RunID                uuid.UUID  `json:"run_id"`
	SelectedBulletID     *uuid.UUID `json:"selected_bullet_id,omitempty"`
	OriginalBulletIDText string     `json:"original_bullet_id_text"`
	FinalText            string     `json:"final_text"`
	LengthChars          int        `json:"length_chars"`
	EstimatedLines       int        `json:"estimated_lines"`
	StyleStrongVerb      bool       `json:"style_strong_verb"`
	StyleQuantified      bool       `json:"style_quantified"`
	StyleNoTaboo         bool       `json:"style_no_taboo"`
	StyleTargetLength    bool       `json:"style_target_length"`
	Ordinal              int        `json:"ordinal"`
	CreatedAt            time.Time  `json:"created_at"`
}

// RunViolation represents a validation violation for a specific pipeline run
type RunViolation struct {
	ID               uuid.UUID `json:"id"`
	RunID            uuid.UUID `json:"run_id"`
	ViolationType    string    `json:"violation_type"`
	Severity         string    `json:"severity"`
	Details          *string   `json:"details,omitempty"`
	LineNumber       *int      `json:"line_number,omitempty"`
	CharCount        *int      `json:"char_count,omitempty"`
	AffectedSections []string  `json:"affected_sections,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// RunRankedStoryInput is used when saving ranked stories
type RunRankedStoryInput struct {
	StoryIDText      string
	StoryID          *uuid.UUID // optional FK
	RelevanceScore   float64
	SkillOverlap     float64
	KeywordOverlap   float64
	EvidenceStrength float64
	HeuristicScore   float64
	LLMScore         *float64
	LLMReasoning     string
	MatchedSkills    []string
	Notes            string
	Ordinal          int
}

// RunResumePlanInput is used when saving a resume plan
type RunResumePlanInput struct {
	MaxBullets       int
	MaxLines         int
	SkillMatchRatio  float64
	SectionBudgets   map[string]int
	TopSkillsCovered []string
	CoverageScore    float64
}

// RunSelectedBulletInput is used when saving selected bullets
type RunSelectedBulletInput struct {
	BulletIDText string
	BulletID     *uuid.UUID // optional FK
	StoryIDText  string
	StoryID      *uuid.UUID // optional FK
	Text         string
	Skills       []string
	Metrics      string
	LengthChars  int
	Section      string
	Ordinal      int
}

// RunRewrittenBulletInput is used when saving rewritten bullets
type RunRewrittenBulletInput struct {
	SelectedBulletID     *uuid.UUID // optional FK
	OriginalBulletIDText string
	FinalText            string
	LengthChars          int
	EstimatedLines       int
	StyleStrongVerb      bool
	StyleQuantified      bool
	StyleNoTaboo         bool
	StyleTargetLength    bool
	Ordinal              int
}

// RunViolationInput is used when saving violations
type RunViolationInput struct {
	ViolationType    string
	Severity         string
	Details          string
	LineNumber       *int
	CharCount        *int
	AffectedSections []string
}

// ValidSeverity checks if a severity value is valid
func ValidSeverity(severity string) bool {
	return severity == SeverityError || severity == SeverityWarning
}
