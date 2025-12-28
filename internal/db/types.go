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
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ArtifactStep constants for known artifact types
const (
	StepJobPosting       = "job_posting"
	StepJobMetadata      = "job_metadata"
	StepJobProfile       = "job_profile"
	StepRankedStories    = "ranked_stories"
	StepResumePlan       = "resume_plan"
	StepSelectedBullets  = "selected_bullets"
	StepCompanyProfile   = "company_profile"
	StepRewrittenBullets = "rewritten_bullets"
	StepViolations       = "violations"
	StepResumeTex        = "resume_tex"
)
