package db

import (
	"time"

	"github.com/google/uuid"
)

// StepStatus constants
const (
	StepStatusPending    = "pending"
	StepStatusInProgress = "in_progress"
	StepStatusCompleted  = "completed"
	StepStatusFailed     = "failed"
	StepStatusSkipped    = "skipped"
	StepStatusBlocked    = "blocked"
)

// StepCategory constants
const (
	StepCategoryIngestion  = "ingestion"
	StepCategoryExperience = "experience"
	StepCategoryResearch   = "research"
	StepCategoryRewriting  = "rewriting"
	StepCategoryValidation = "validation"
)

// RunStep represents a single step execution for a pipeline run
type RunStep struct {
	ID           uuid.UUID              `json:"id"`
	RunID        uuid.UUID              `json:"run_id"`
	Step         string                 `json:"step"`
	Category     string                 `json:"category"`
	Status       string                 `json:"status"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	DurationMs   *int                   `json:"duration_ms,omitempty"`
	ArtifactID   *uuid.UUID             `json:"artifact_id,omitempty"`
	ErrorMessage *string                `json:"error_message,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// RunStepInput represents input for creating/updating a run step
type RunStepInput struct {
	Step       string
	Category   string
	Status     string
	Parameters map[string]interface{}
}

// RunCheckpoint represents a checkpoint state after a completed step
type RunCheckpoint struct {
	ID          uuid.UUID              `json:"id"`
	RunID       uuid.UUID              `json:"run_id"`
	Step        string                 `json:"step"`
	CompletedAt time.Time              `json:"completed_at"`
	Artifacts   map[string]interface{} `json:"artifacts"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// RunCheckpointInput represents input for creating a checkpoint
type RunCheckpointInput struct {
	Step      string
	Artifacts map[string]interface{}
	Metadata  map[string]interface{}
}
