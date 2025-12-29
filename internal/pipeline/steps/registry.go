// Package steps provides step definitions, dependency validation, and step execution
// for the resume customization pipeline.
package steps

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	dbpkg "github.com/jonathan/resume-customizer/internal/db"
)

// StepDefinition defines metadata for a pipeline step
type StepDefinition struct {
	Name         string
	Category     string
	Dependencies []string
	Optional     []string
}

// StepExecutor defines the interface for executing pipeline steps
type StepExecutor interface {
	Name() string
	Category() string
	Dependencies() []string
	Execute(ctx context.Context, runID uuid.UUID, params map[string]interface{}) (*StepResult, error)
	ValidateDependencies(ctx context.Context, db *dbpkg.DB, runID uuid.UUID) error
}

// StepResult represents the result of executing a step
type StepResult struct {
	Step       string
	Status     string
	ArtifactID *uuid.UUID
	Duration   int64 // milliseconds
	Error      error
	Metadata   map[string]interface{}
}

// StepRegistry holds all step definitions
var StepRegistry = map[string]StepDefinition{
	"ingest_job": {
		Name:         "ingest_job",
		Category:     dbpkg.StepCategoryIngestion,
		Dependencies: []string{},
		Optional:     []string{},
	},
	"parse_job": {
		Name:         "parse_job",
		Category:     dbpkg.StepCategoryIngestion,
		Dependencies: []string{"ingest_job"},
		Optional:     []string{},
	},
	"extract_education": {
		Name:         "extract_education",
		Category:     dbpkg.StepCategoryIngestion,
		Dependencies: []string{"parse_job"},
		Optional:     []string{},
	},
	"load_experience": {
		Name:         "load_experience",
		Category:     dbpkg.StepCategoryExperience,
		Dependencies: []string{},
		Optional:     []string{},
	},
	"rank_stories": {
		Name:         "rank_stories",
		Category:     dbpkg.StepCategoryExperience,
		Dependencies: []string{"parse_job", "load_experience"},
		Optional:     []string{},
	},
	"score_education": {
		Name:         "score_education",
		Category:     dbpkg.StepCategoryExperience,
		Dependencies: []string{"parse_job", "load_experience"},
		Optional:     []string{},
	},
	"select_plan": {
		Name:         "select_plan",
		Category:     dbpkg.StepCategoryExperience,
		Dependencies: []string{"rank_stories"},
		Optional:     []string{"score_education"},
	},
	"materialize_bullets": {
		Name:         "materialize_bullets",
		Category:     dbpkg.StepCategoryExperience,
		Dependencies: []string{"select_plan"},
		Optional:     []string{},
	},
	"research_company": {
		Name:         "research_company",
		Category:     dbpkg.StepCategoryResearch,
		Dependencies: []string{"parse_job"},
		Optional:     []string{},
	},
	"summarize_voice": {
		Name:         "summarize_voice",
		Category:     dbpkg.StepCategoryResearch,
		Dependencies: []string{"research_company"},
		Optional:     []string{},
	},
	"rewrite_bullets": {
		Name:         "rewrite_bullets",
		Category:     dbpkg.StepCategoryRewriting,
		Dependencies: []string{"materialize_bullets", "summarize_voice"},
		Optional:     []string{},
	},
	"render_latex": {
		Name:         "render_latex",
		Category:     dbpkg.StepCategoryValidation,
		Dependencies: []string{"rewrite_bullets"},
		Optional:     []string{},
	},
	"validate_latex": {
		Name:         "validate_latex",
		Category:     dbpkg.StepCategoryValidation,
		Dependencies: []string{"render_latex"},
		Optional:     []string{},
	},
	"repair_violations": {
		Name:         "repair_violations",
		Category:     dbpkg.StepCategoryValidation,
		Dependencies: []string{"validate_latex"},
		Optional:     []string{},
	},
}

// DependencyError represents a dependency validation error
type DependencyError struct {
	Step                string
	MissingDependencies []string
}

func (e *DependencyError) Error() string {
	return fmt.Sprintf("missing dependencies: %v", e.MissingDependencies)
}

// ValidateDependencies checks if all required dependencies for a step are completed
func ValidateDependencies(ctx context.Context, db *dbpkg.DB, runID uuid.UUID, stepName string) error {
	def, ok := StepRegistry[stepName]
	if !ok {
		return fmt.Errorf("unknown step: %s", stepName)
	}

	var missing []string

	// Check each required dependency
	for _, dep := range def.Dependencies {
		step, err := db.GetRunStep(ctx, runID, dep)
		if err != nil {
			return fmt.Errorf("failed to check dependency %s: %w", dep, err)
		}
		if step == nil {
			missing = append(missing, dep)
			continue
		}
		if step.Status != dbpkg.StepStatusCompleted {
			missing = append(missing, dep)
		}
	}

	if len(missing) > 0 {
		return &DependencyError{
			Step:                stepName,
			MissingDependencies: missing,
		}
	}

	return nil
}

// GetAvailableSteps returns steps that can be executed (dependencies met)
func GetAvailableSteps(ctx context.Context, db *dbpkg.DB, runID uuid.UUID) ([]string, error) {
	var available []string

	for stepName := range StepRegistry {
		// Check if step already exists
		existing, err := db.GetRunStep(ctx, runID, stepName)
		if err != nil {
			return nil, fmt.Errorf("failed to check step %s: %w", stepName, err)
		}
		if existing != nil && existing.Status == dbpkg.StepStatusCompleted {
			continue // Already completed
		}
		if existing != nil && existing.Status == dbpkg.StepStatusInProgress {
			continue // Currently in progress
		}

		// Check dependencies
		if err := ValidateDependencies(ctx, db, runID, stepName); err != nil {
			continue // Dependencies not met
		}

		available = append(available, stepName)
	}

	return available, nil
}

// GetBlockedSteps returns steps that are blocked (dependencies not met)
func GetBlockedSteps(ctx context.Context, db *dbpkg.DB, runID uuid.UUID) ([]string, error) {
	var blocked []string

	for stepName := range StepRegistry {
		// Check if step already exists and is not completed
		existing, err := db.GetRunStep(ctx, runID, stepName)
		if err != nil {
			return nil, fmt.Errorf("failed to check step %s: %w", stepName, err)
		}
		if existing != nil && (existing.Status == dbpkg.StepStatusCompleted || existing.Status == dbpkg.StepStatusInProgress) {
			continue // Already completed or in progress
		}

		// Check dependencies
		if err := ValidateDependencies(ctx, db, runID, stepName); err != nil {
			blocked = append(blocked, stepName)
		}
	}

	return blocked, nil
}
