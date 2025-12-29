package db

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestStepStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", StepStatusPending)
	assert.Equal(t, "in_progress", StepStatusInProgress)
	assert.Equal(t, "completed", StepStatusCompleted)
	assert.Equal(t, "failed", StepStatusFailed)
	assert.Equal(t, "skipped", StepStatusSkipped)
	assert.Equal(t, "blocked", StepStatusBlocked)
}

func TestStepCategoryConstants(t *testing.T) {
	assert.Equal(t, "ingestion", StepCategoryIngestion)
	assert.Equal(t, "experience", StepCategoryExperience)
	assert.Equal(t, "research", StepCategoryResearch)
	assert.Equal(t, "rewriting", StepCategoryRewriting)
	assert.Equal(t, "validation", StepCategoryValidation)
}

func TestRunStepInput(t *testing.T) {
	input := &RunStepInput{
		Step:       "test_step",
		Category:   StepCategoryIngestion,
		Status:     StepStatusPending,
		Parameters: map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "test_step", input.Step)
	assert.Equal(t, StepCategoryIngestion, input.Category)
	assert.Equal(t, StepStatusPending, input.Status)
	assert.Equal(t, map[string]interface{}{"key": "value"}, input.Parameters)
}

func TestRunCheckpointInput(t *testing.T) {
	input := &RunCheckpointInput{
		Step:      "test_step",
		Artifacts: map[string]interface{}{"artifact1": uuid.New().String()},
		Metadata:  map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "test_step", input.Step)
	assert.NotNil(t, input.Artifacts)
	assert.NotNil(t, input.Metadata)
}
