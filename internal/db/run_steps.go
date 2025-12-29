package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// -----------------------------------------------------------------------------
// Run Steps Methods
// -----------------------------------------------------------------------------

// CreateRunStep creates a new run step record
func (db *DB) CreateRunStep(ctx context.Context, runID uuid.UUID, input *RunStepInput) (*RunStep, error) {
	var step RunStep
	var parametersJSON []byte
	if input.Parameters != nil {
		var err error
		parametersJSON, err = json.Marshal(input.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}
	}

	err := db.pool.QueryRow(ctx,
		`INSERT INTO run_steps (run_id, step, category, status, parameters)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, run_id, step, category, status, started_at, completed_at,
		           duration_ms, artifact_id, error_message, parameters, created_at, updated_at`,
		runID, input.Step, input.Category, input.Status, parametersJSON,
	).Scan(&step.ID, &step.RunID, &step.Step, &step.Category, &step.Status,
		&step.StartedAt, &step.CompletedAt, &step.DurationMs, &step.ArtifactID,
		&step.ErrorMessage, &parametersJSON, &step.CreatedAt, &step.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create run step: %w", err)
	}

	if parametersJSON != nil {
		_ = json.Unmarshal(parametersJSON, &step.Parameters)
	}

	return &step, nil
}

// GetRunStep retrieves a run step by run_id and step name
func (db *DB) GetRunStep(ctx context.Context, runID uuid.UUID, stepName string) (*RunStep, error) {
	var step RunStep
	var parametersJSON []byte

	err := db.pool.QueryRow(ctx,
		`SELECT id, run_id, step, category, status, started_at, completed_at,
		        duration_ms, artifact_id, error_message, parameters, created_at, updated_at
		 FROM run_steps
		 WHERE run_id = $1 AND step = $2`,
		runID, stepName,
	).Scan(&step.ID, &step.RunID, &step.Step, &step.Category, &step.Status,
		&step.StartedAt, &step.CompletedAt, &step.DurationMs, &step.ArtifactID,
		&step.ErrorMessage, &parametersJSON, &step.CreatedAt, &step.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get run step: %w", err)
	}

	if parametersJSON != nil {
		_ = json.Unmarshal(parametersJSON, &step.Parameters)
	}

	return &step, nil
}

// ListRunSteps retrieves all steps for a run, optionally filtered by status or category
func (db *DB) ListRunSteps(ctx context.Context, runID uuid.UUID, status, category *string) ([]RunStep, error) {
	query := `SELECT id, run_id, step, category, status, started_at, completed_at,
	                 duration_ms, artifact_id, error_message, parameters, created_at, updated_at
	          FROM run_steps
	          WHERE run_id = $1`
	args := []interface{}{runID}
	argPos := 2

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *status)
		argPos++
	}

	if category != nil {
		query += fmt.Sprintf(" AND category = $%d", argPos)
		args = append(args, *category)
	}

	query += " ORDER BY created_at"

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list run steps: %w", err)
	}
	defer rows.Close()

	var steps []RunStep
	for rows.Next() {
		var step RunStep
		var parametersJSON []byte

		if err := rows.Scan(&step.ID, &step.RunID, &step.Step, &step.Category, &step.Status,
			&step.StartedAt, &step.CompletedAt, &step.DurationMs, &step.ArtifactID,
			&step.ErrorMessage, &parametersJSON, &step.CreatedAt, &step.UpdatedAt); err != nil {
			return nil, err
		}

		if parametersJSON != nil {
			_ = json.Unmarshal(parametersJSON, &step.Parameters)
		}

		steps = append(steps, step)
	}

	return steps, nil
}

// UpdateRunStepStatus updates the status and related fields of a run step
func (db *DB) UpdateRunStepStatus(ctx context.Context, runID uuid.UUID, stepName string, status string, errorMsg *string, artifactID *uuid.UUID) error {
	now := time.Now()
	var durationMs *int

	// Get current step to calculate duration
	currentStep, err := db.GetRunStep(ctx, runID, stepName)
	if err != nil {
		return err
	}
	if currentStep == nil {
		return fmt.Errorf("step not found: %s", stepName)
	}

	// Calculate duration if step is being completed
	if status == StepStatusCompleted && currentStep.StartedAt != nil {
		dur := int(now.Sub(*currentStep.StartedAt).Milliseconds())
		durationMs = &dur
	}

	var startedAt *time.Time
	if status == StepStatusInProgress && currentStep.StartedAt == nil {
		startedAt = &now
	}

	var completedAt *time.Time
	if status == StepStatusCompleted || status == StepStatusFailed || status == StepStatusSkipped {
		completedAt = &now
	}

	_, err = db.pool.Exec(ctx,
		`UPDATE run_steps
		 SET status = $1, started_at = COALESCE($2, started_at), completed_at = $3,
		     duration_ms = $4, error_message = $5, artifact_id = COALESCE($6, artifact_id),
		     updated_at = NOW()
		 WHERE run_id = $7 AND step = $8`,
		status, startedAt, completedAt, durationMs, errorMsg, artifactID, runID, stepName,
	)
	if err != nil {
		return fmt.Errorf("failed to update run step status: %w", err)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Run Checkpoints Methods
// -----------------------------------------------------------------------------

// CreateRunCheckpoint creates a new checkpoint after a step completion
func (db *DB) CreateRunCheckpoint(ctx context.Context, runID uuid.UUID, input *RunCheckpointInput) (*RunCheckpoint, error) {
	var checkpoint RunCheckpoint
	artifactsJSON, _ := json.Marshal(input.Artifacts)
	metadataJSON, _ := json.Marshal(input.Metadata)

	err := db.pool.QueryRow(ctx,
		`INSERT INTO run_checkpoints (run_id, step, artifacts, metadata)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (run_id, step) DO UPDATE
		 SET step = EXCLUDED.step, artifacts = EXCLUDED.artifacts,
		     metadata = EXCLUDED.metadata, completed_at = NOW()
		 RETURNING id, run_id, step, completed_at, artifacts, metadata, created_at`,
		runID, input.Step, artifactsJSON, metadataJSON,
	).Scan(&checkpoint.ID, &checkpoint.RunID, &checkpoint.Step, &checkpoint.CompletedAt,
		&artifactsJSON, &metadataJSON, &checkpoint.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkpoint: %w", err)
	}

	_ = json.Unmarshal(artifactsJSON, &checkpoint.Artifacts)
	_ = json.Unmarshal(metadataJSON, &checkpoint.Metadata)

	return &checkpoint, nil
}

// GetRunCheckpoint retrieves the latest checkpoint for a run
func (db *DB) GetRunCheckpoint(ctx context.Context, runID uuid.UUID) (*RunCheckpoint, error) {
	var checkpoint RunCheckpoint
	var artifactsJSON, metadataJSON []byte

	err := db.pool.QueryRow(ctx,
		`SELECT id, run_id, step, completed_at, artifacts, metadata, created_at
		 FROM run_checkpoints
		 WHERE run_id = $1
		 ORDER BY completed_at DESC
		 LIMIT 1`,
		runID,
	).Scan(&checkpoint.ID, &checkpoint.RunID, &checkpoint.Step, &checkpoint.CompletedAt,
		&artifactsJSON, &metadataJSON, &checkpoint.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}

	if artifactsJSON != nil {
		_ = json.Unmarshal(artifactsJSON, &checkpoint.Artifacts)
	}
	if metadataJSON != nil {
		_ = json.Unmarshal(metadataJSON, &checkpoint.Metadata)
	}

	return &checkpoint, nil
}

// GetRunCheckpointByStep retrieves a checkpoint for a specific step
func (db *DB) GetRunCheckpointByStep(ctx context.Context, runID uuid.UUID, stepName string) (*RunCheckpoint, error) {
	var checkpoint RunCheckpoint
	var artifactsJSON, metadataJSON []byte

	err := db.pool.QueryRow(ctx,
		`SELECT id, run_id, step, completed_at, artifacts, metadata, created_at
		 FROM run_checkpoints
		 WHERE run_id = $1 AND step = $2`,
		runID, stepName,
	).Scan(&checkpoint.ID, &checkpoint.RunID, &checkpoint.Step, &checkpoint.CompletedAt,
		&artifactsJSON, &metadataJSON, &checkpoint.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}

	if artifactsJSON != nil {
		_ = json.Unmarshal(artifactsJSON, &checkpoint.Artifacts)
	}
	if metadataJSON != nil {
		_ = json.Unmarshal(metadataJSON, &checkpoint.Metadata)
	}

	return &checkpoint, nil
}
