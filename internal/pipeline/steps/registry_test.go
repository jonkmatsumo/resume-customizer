package steps

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbpkg "github.com/jonathan/resume-customizer/internal/db"
)

func TestStepRegistry(t *testing.T) {
	// Verify all expected steps are in the registry
	expectedSteps := []string{
		"ingest_job", "parse_job", "extract_education",
		"load_experience", "rank_stories", "score_education",
		"select_plan", "materialize_bullets",
		"research_company", "summarize_voice",
		"rewrite_bullets", "render_latex", "validate_latex",
		"repair_violations",
	}

	for _, stepName := range expectedSteps {
		def, ok := StepRegistry[stepName]
		require.True(t, ok, "Step %s should be in registry", stepName)
		assert.Equal(t, stepName, def.Name)
		assert.NotEmpty(t, def.Category)
	}
}

func TestStepRegistryCategories(t *testing.T) {
	categories := map[string][]string{
		dbpkg.StepCategoryIngestion:  {"ingest_job", "parse_job", "extract_education"},
		dbpkg.StepCategoryExperience: {"load_experience", "rank_stories", "score_education", "select_plan", "materialize_bullets"},
		dbpkg.StepCategoryResearch:   {"research_company", "summarize_voice"},
		dbpkg.StepCategoryRewriting:  {"rewrite_bullets"},
		dbpkg.StepCategoryValidation: {"render_latex", "validate_latex", "repair_violations"},
	}

	for category, stepNames := range categories {
		for _, stepName := range stepNames {
			def, ok := StepRegistry[stepName]
			require.True(t, ok)
			assert.Equal(t, category, def.Category, "Step %s should be in category %s", stepName, category)
		}
	}
}

func TestDependencyError(t *testing.T) {
	err := &DependencyError{
		Step:                "test_step",
		MissingDependencies: []string{"dep1", "dep2"},
	}

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing dependencies")
	assert.Equal(t, "test_step", err.Step)
	assert.Equal(t, []string{"dep1", "dep2"}, err.MissingDependencies)
}

func TestValidateDependencies_UnknownStep(t *testing.T) {
	// This test doesn't require a database connection
	// We'll test the actual validation logic in integration tests
	err := ValidateDependencies(context.Background(), nil, uuid.Nil, "unknown_step")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown step")
}
