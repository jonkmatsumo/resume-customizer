package db

import (
	"time"

	"github.com/google/uuid"
)

// Run represents a pipeline run record
type Run struct {
	ID          uuid.UUID  `json:"id"`
	Company     string     `json:"company"`
	RoleTitle   string     `json:"role_title"`
	JobURL      string     `json:"job_url"`
	Status      string     `json:"status"`
	UserID      *uuid.UUID `json:"user_id,omitempty"` // Nullable for backward compatibility
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ArtifactStep constants for known artifact types
const (
	// Pipeline lifecycle
	StepRunStarted = "run_started"

	// Ingestion phase
	StepJobPosting   = "job_posting"
	StepJobMetadata  = "job_metadata"
	StepJobProfile   = "job_profile"
	StepEducationReq = "education_requirements"

	// Experience branch
	StepExperienceBank  = "experience_bank"
	StepRankedStories   = "ranked_stories"
	StepEducationScores = "education_scores"
	StepResumePlan      = "resume_plan"
	StepSelectedBullets = "selected_bullets"

	// Research branch
	StepResearchSession = "research_session"
	StepCompanyCorpus   = "company_corpus"
	StepSources         = "sources"
	StepCompanyProfile  = "company_profile"

	// Final steps
	StepRewrittenBullets = "rewritten_bullets"
	StepResumeTex        = "resume_tex"
	StepViolations       = "violations"
)

// Category constants for grouping artifacts by pipeline phase
const (
	CategoryLifecycle  = "lifecycle"
	CategoryIngestion  = "ingestion"
	CategoryExperience = "experience"
	CategoryResearch   = "research"
	CategoryRewriting  = "rewriting"
	CategoryValidation = "validation"
)
