package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArtifactStepConstants(t *testing.T) {
	// Verify step constants are defined
	steps := []string{
		StepJobPosting,
		StepJobMetadata,
		StepJobProfile,
		StepRankedStories,
		StepResumePlan,
		StepSelectedBullets,
		StepCompanyProfile,
		StepRewrittenBullets,
		StepViolations,
		StepResumeTex,
	}

	for _, step := range steps {
		assert.NotEmpty(t, step, "step constant should not be empty")
	}
}

func TestRunType(t *testing.T) {
	// Verify Run struct can be instantiated
	run := Run{
		Company:   "TestCorp",
		RoleTitle: "Engineer",
		Status:    "running",
	}

	assert.Equal(t, "TestCorp", run.Company)
	assert.Equal(t, "Engineer", run.RoleTitle)
	assert.Equal(t, "running", run.Status)
	assert.Nil(t, run.CompletedAt)
}
