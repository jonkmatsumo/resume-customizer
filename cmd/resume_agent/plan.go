// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/jonathan/resume-customizer/internal/selection"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Select optimal stories and bullets for a resume plan",
	Long:  "Uses dynamic programming to optimally select experience stories and bullets under space budget constraints, maximizing relevance and skill coverage.",
	RunE:  runPlan,
}

var (
	planRanked      string
	planJobProfile  string
	planUserID      string
	planRunID       string
	planDatabaseURL string
	planMaxBullets  int
	planMaxLines    int
	planOutput      string
)

func init() {
	planCmd.Flags().StringVarP(&planRanked, "ranked", "r", "", "Path to RankedStories JSON file (deprecated: use --run-id)")
	planCmd.Flags().StringVarP(&planJobProfile, "job-profile", "j", "", "Path to JobProfile JSON file (deprecated: use --run-id)")
	planCmd.Flags().StringVarP(&planUserID, "user-id", "u", "", "User ID (required)")
	planCmd.Flags().StringVar(&planRunID, "run-id", "", "Run ID to load data from database (required if not using --ranked/--job-profile)")
	planCmd.Flags().StringVar(&planDatabaseURL, "db-url", "", "Database URL (required with --run-id)")
	planCmd.Flags().IntVar(&planMaxBullets, "max-bullets", 0, "Maximum bullets allowed (required)")
	planCmd.Flags().IntVar(&planMaxLines, "max-lines", 0, "Maximum lines allowed (required)")
	planCmd.Flags().StringVarP(&planOutput, "out", "o", "", "Path to output ResumePlan JSON file (deprecated: use --run-id)")

	rootCmd.AddCommand(planCmd)
}

func runPlan(_ *cobra.Command, _ []string) error {
	// Determine mode: database or file
	useDatabase := planRunID != ""
	useFiles := planRanked != "" || planJobProfile != ""

	if useDatabase && useFiles {
		return fmt.Errorf("cannot use --run-id with --ranked/--job-profile/--out flags")
	}
	if !useDatabase && !useFiles {
		return fmt.Errorf("must provide either --run-id or --ranked/--job-profile/--out flags")
	}

	// Validate flags
	if planMaxBullets <= 0 {
		return fmt.Errorf("max-bullets must be greater than 0, got %d", planMaxBullets)
	}
	if planMaxLines <= 0 {
		return fmt.Errorf("max-lines must be greater than 0, got %d", planMaxLines)
	}

	ctx := context.Background()

	// Connect to database (required for both modes - experience bank is always from DB)
	if planDatabaseURL == "" {
		planDatabaseURL = os.Getenv("DATABASE_URL")
	}
	if planDatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL not set and --db-url not provided")
	}

	database, err := db.Connect(ctx, planDatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	// Parse user ID
	uid, err := uuid.Parse(planUserID)
	if err != nil {
		return fmt.Errorf("invalid user-id: %w", err)
	}

	// Load ExperienceBank from DB (always from database)
	experienceBank, err := database.GetExperienceBank(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to load experience bank from DB: %w", err)
	}

	// Load ranked stories and job profile
	var rankedStories *types.RankedStories
	var jobProfile *types.JobProfile
	var runID uuid.UUID

	if useFiles {
		// File mode (deprecated)
		fmt.Fprintf(os.Stderr, "Warning: File-based mode is deprecated. Use --run-id instead.\n")

		rankedContent, err := os.ReadFile(planRanked)
		if err != nil {
			return fmt.Errorf("failed to read ranked stories file %s: %w", planRanked, err)
		}

		var stories types.RankedStories
		if err := json.Unmarshal(rankedContent, &stories); err != nil {
			return fmt.Errorf("failed to unmarshal ranked stories JSON: %w", err)
		}
		rankedStories = &stories

		jobProfileContent, err := os.ReadFile(planJobProfile)
		if err != nil {
			return fmt.Errorf("failed to read job profile file %s: %w", planJobProfile, err)
		}

		var profile types.JobProfile
		if err := json.Unmarshal(jobProfileContent, &profile); err != nil {
			return fmt.Errorf("failed to unmarshal job profile JSON: %w", err)
		}
		jobProfile = &profile
	} else {
		// Database mode
		runID, err = uuid.Parse(planRunID)
		if err != nil {
			return fmt.Errorf("invalid run-id: %w", err)
		}

		rankedStories, err = database.GetRankedStoriesByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load ranked stories from database: %w", err)
		}
		if rankedStories == nil {
			return fmt.Errorf("ranked stories not found for run %s", runID)
		}

		jobProfile, err = database.GetJobProfileByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load job profile from database: %w", err)
		}
		if jobProfile == nil {
			return fmt.Errorf("job profile not found for run %s", runID)
		}
	}

	// Create SpaceBudget
	spaceBudget := &types.SpaceBudget{
		MaxBullets: planMaxBullets,
		MaxLines:   planMaxLines,
	}

	// Select plan
	resumePlan, err := selection.SelectPlan(rankedStories, jobProfile, experienceBank, spaceBudget)
	if err != nil {
		return fmt.Errorf("failed to select plan: %w", err)
	}

	if useFiles {
		// File mode: write to file
		jsonOutput, err := json.MarshalIndent(resumePlan, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal resume plan to JSON: %w", err)
		}

		outputDir := filepath.Dir(planOutput)
		if outputDir != "" && outputDir != "." {
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
			}
		}

		if err := os.WriteFile(planOutput, jsonOutput, 0644); err != nil {
			return fmt.Errorf("failed to write resume plan to output file %s: %w", planOutput, err)
		}

		// Validate output against schema
		schemaPath := schemas.ResolveSchemaPath("schemas/resume_plan.schema.json")
		if schemaPath != "" {
			if err := schemas.ValidateJSON(schemaPath, planOutput); err != nil {
				var validationErr *schemas.ValidationError
				var schemaLoadErr *schemas.SchemaLoadError
				if errors.As(err, &validationErr) {
					return fmt.Errorf("generated resume plan does not validate against schema: %w", err)
				} else if errors.As(err, &schemaLoadErr) {
					_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema (schema loading failed): %v\n", err)
				} else {
					_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema: %v\n", err)
				}
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully created resume plan with %d selected stories\n", len(resumePlan.SelectedStories))
		_, _ = fmt.Fprintf(os.Stdout, "Coverage score: %.2f\n", resumePlan.Coverage.CoverageScore)
		_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", planOutput)
	} else {
		// Database mode: save to database
		// Convert to RunResumePlanInput format
		input := &db.RunResumePlanInput{
			MaxBullets:       resumePlan.SpaceBudget.MaxBullets,
			MaxLines:         resumePlan.SpaceBudget.MaxLines,
			SkillMatchRatio:  resumePlan.SpaceBudget.SkillMatchRatio,
			SectionBudgets:   resumePlan.SpaceBudget.Sections,
			TopSkillsCovered: resumePlan.Coverage.TopSkillsCovered,
			CoverageScore:    resumePlan.Coverage.CoverageScore,
		}

		_, err = database.SaveRunResumePlan(ctx, runID, input)
		if err != nil {
			return fmt.Errorf("failed to save resume plan to database: %w", err)
		}

		// Also save as artifact
		if err := database.SaveArtifact(ctx, runID, db.StepResumePlan, db.CategoryExperience, resumePlan); err != nil {
			return fmt.Errorf("failed to save resume plan artifact: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully created resume plan with %d selected stories\n", len(resumePlan.SelectedStories))
		_, _ = fmt.Fprintf(os.Stdout, "Coverage score: %.2f\n", resumePlan.Coverage.CoverageScore)
		_, _ = fmt.Fprintf(os.Stdout, "Saved to database (run: %s)\n", runID)
	}

	return nil
}
