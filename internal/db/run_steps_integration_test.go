//go:build integration
// +build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRunStep_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a run first
	runID, err := db.CreateRun(ctx, "Test Company", "Test Role", "https://example.com/job")
	require.NoError(t, err)

	// Create a step
	stepInput := &RunStepInput{
		Step:       "ingest_job",
		Category:   StepCategoryIngestion,
		Status:     StepStatusPending,
		Parameters: map[string]interface{}{"job_url": "https://example.com/job"},
	}

	step, err := db.CreateRunStep(ctx, runID, stepInput)
	require.NoError(t, err)
	assert.NotNil(t, step)
	assert.Equal(t, runID, step.RunID)
	assert.Equal(t, "ingest_job", step.Step)
	assert.Equal(t, StepCategoryIngestion, step.Category)
	assert.Equal(t, StepStatusPending, step.Status)
}

func TestGetRunStep_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a run and step
	runID, err := db.CreateRun(ctx, "Test Company", "Test Role", "https://example.com/job")
	require.NoError(t, err)

	stepInput := &RunStepInput{
		Step:     "ingest_job",
		Category: StepCategoryIngestion,
		Status:   StepStatusPending,
	}

	_, err = db.CreateRunStep(ctx, runID, stepInput)
	require.NoError(t, err)

	// Retrieve the step
	step, err := db.GetRunStep(ctx, runID, "ingest_job")
	require.NoError(t, err)
	assert.NotNil(t, step)
	assert.Equal(t, "ingest_job", step.Step)

	// Test non-existent step
	step, err = db.GetRunStep(ctx, runID, "nonexistent_step")
	require.NoError(t, err)
	assert.Nil(t, step)
}

func TestUpdateRunStepStatus_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a run and step
	runID, err := db.CreateRun(ctx, "Test Company", "Test Role", "https://example.com/job")
	require.NoError(t, err)

	stepInput := &RunStepInput{
		Step:     "ingest_job",
		Category: StepCategoryIngestion,
		Status:   StepStatusPending,
	}

	_, err = db.CreateRunStep(ctx, runID, stepInput)
	require.NoError(t, err)

	// Update to in_progress
	err = db.UpdateRunStepStatus(ctx, runID, "ingest_job", StepStatusInProgress, nil, nil)
	require.NoError(t, err)

	step, err := db.GetRunStep(ctx, runID, "ingest_job")
	require.NoError(t, err)
	assert.Equal(t, StepStatusInProgress, step.Status)
	assert.NotNil(t, step.StartedAt)

	// Update to completed
	err = db.UpdateRunStepStatus(ctx, runID, "ingest_job", StepStatusCompleted, nil, nil)
	require.NoError(t, err)

	step, err = db.GetRunStep(ctx, runID, "ingest_job")
	require.NoError(t, err)
	assert.Equal(t, StepStatusCompleted, step.Status)
	assert.NotNil(t, step.CompletedAt)
	assert.NotNil(t, step.DurationMs)
}

func TestListRunSteps_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a run
	runID, err := db.CreateRun(ctx, "Test Company", "Test Role", "https://example.com/job")
	require.NoError(t, err)

	// Create multiple steps
	steps := []*RunStepInput{
		{Step: "ingest_job", Category: StepCategoryIngestion, Status: StepStatusCompleted},
		{Step: "parse_job", Category: StepCategoryIngestion, Status: StepStatusInProgress},
		{Step: "load_experience", Category: StepCategoryExperience, Status: StepStatusPending},
	}

	for _, stepInput := range steps {
		_, err = db.CreateRunStep(ctx, runID, stepInput)
		require.NoError(t, err)
	}

	// List all steps
	allSteps, err := db.ListRunSteps(ctx, runID, nil, nil)
	require.NoError(t, err)
	assert.Len(t, allSteps, 3)

	// Filter by status
	completedSteps, err := db.ListRunSteps(ctx, runID, stringPtr(StepStatusCompleted), nil)
	require.NoError(t, err)
	assert.Len(t, completedSteps, 1)
	assert.Equal(t, "ingest_job", completedSteps[0].Step)

	// Filter by category
	ingestionSteps, err := db.ListRunSteps(ctx, runID, nil, stringPtr(StepCategoryIngestion))
	require.NoError(t, err)
	assert.Len(t, ingestionSteps, 2)
}

func TestCreateRunCheckpoint_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a run
	runID, err := db.CreateRun(ctx, "Test Company", "Test Role", "https://example.com/job")
	require.NoError(t, err)

	// Create a checkpoint
	checkpointInput := &RunCheckpointInput{
		Step:      "ingest_job",
		Artifacts: map[string]interface{}{"job_posting": uuid.New().String()},
		Metadata:  map[string]interface{}{"key": "value"},
	}

	checkpoint, err := db.CreateRunCheckpoint(ctx, runID, checkpointInput)
	require.NoError(t, err)
	assert.NotNil(t, checkpoint)
	assert.Equal(t, runID, checkpoint.RunID)
	assert.Equal(t, "ingest_job", checkpoint.Step)
	assert.NotEmpty(t, checkpoint.Artifacts)
}

func TestGetRunCheckpoint_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a run
	runID, err := db.CreateRun(ctx, "Test Company", "Test Role", "https://example.com/job")
	require.NoError(t, err)

	// Create multiple checkpoints
	checkpoint1 := &RunCheckpointInput{
		Step:      "ingest_job",
		Artifacts: map[string]interface{}{"artifact1": "value1"},
	}
	_, err = db.CreateRunCheckpoint(ctx, runID, checkpoint1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	checkpoint2 := &RunCheckpointInput{
		Step:      "parse_job",
		Artifacts: map[string]interface{}{"artifact2": "value2"},
	}
	_, err = db.CreateRunCheckpoint(ctx, runID, checkpoint2)
	require.NoError(t, err)

	// Get latest checkpoint (should be parse_job)
	checkpoint, err := db.GetRunCheckpoint(ctx, runID)
	require.NoError(t, err)
	assert.NotNil(t, checkpoint)
	assert.Equal(t, "parse_job", checkpoint.Step)

	// Get specific checkpoint
	checkpoint, err = db.GetRunCheckpointByStep(ctx, runID, "ingest_job")
	require.NoError(t, err)
	assert.NotNil(t, checkpoint)
	assert.Equal(t, "ingest_job", checkpoint.Step)
}

func stringPtr(s string) *string {
	return &s
}
