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

var materializeCmd = &cobra.Command{
	Use:   "materialize",
	Short: "Extract selected bullets from a resume plan",
	Long:  "Materializes (extracts) the actual bullet data from ExperienceBank based on a ResumePlan, pulling the exact bullets for rewriting.",
	RunE:  runMaterialize,
}

var (
	materializePlan        string
	materializeUserID      string
	materializeRunID       string
	materializeDatabaseURL string
	materializeOutput      string
)

func init() {
	materializeCmd.Flags().StringVarP(&materializePlan, "plan", "p", "", "Path to ResumePlan JSON file (deprecated: use --run-id)")
	materializeCmd.Flags().StringVarP(&materializeUserID, "user-id", "u", "", "User ID (required)")
	materializeCmd.Flags().StringVar(&materializeRunID, "run-id", "", "Run ID to load resume plan from database (required if not using --plan)")
	materializeCmd.Flags().StringVar(&materializeDatabaseURL, "db-url", "", "Database URL (required with --run-id)")
	materializeCmd.Flags().StringVarP(&materializeOutput, "out", "o", "", "Path to output SelectedBullets JSON file (deprecated: use --run-id)")

	rootCmd.AddCommand(materializeCmd)
}

func runMaterialize(_ *cobra.Command, _ []string) error {
	// Determine mode: database or file
	useDatabase := materializeRunID != ""
	useFiles := materializePlan != "" || materializeOutput != ""

	if useDatabase && useFiles {
		return fmt.Errorf("cannot use --run-id with --plan/--out flags")
	}
	if !useDatabase && !useFiles {
		return fmt.Errorf("must provide either --run-id or --plan/--out flags")
	}

	ctx := context.Background()

	// Connect to database (required for both modes - experience bank is always from DB)
	if materializeDatabaseURL == "" {
		materializeDatabaseURL = os.Getenv("DATABASE_URL")
	}
	if materializeDatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL not set and --db-url not provided (required for DB access)")
	}

	database, err := db.Connect(ctx, materializeDatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	// Parse user ID
	uid, err := uuid.Parse(materializeUserID)
	if err != nil {
		return fmt.Errorf("invalid user-id: %w", err)
	}

	// Load ExperienceBank from DB (always from database)
	experienceBank, err := database.GetExperienceBank(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to load experience bank from DB: %w", err)
	}

	// Load resume plan
	var plan *types.ResumePlan
	var runID uuid.UUID

	if useFiles {
		// File mode (deprecated)
		fmt.Fprintf(os.Stderr, "Warning: File-based mode is deprecated. Use --run-id instead.\n")

		planContent, err := os.ReadFile(materializePlan)
		if err != nil {
			return fmt.Errorf("failed to read plan file %s: %w", materializePlan, err)
		}

		var p types.ResumePlan
		if err := json.Unmarshal(planContent, &p); err != nil {
			return fmt.Errorf("failed to unmarshal plan JSON: %w", err)
		}
		plan = &p
	} else {
		// Database mode
		runID, err = uuid.Parse(materializeRunID)
		if err != nil {
			return fmt.Errorf("invalid run-id: %w", err)
		}

		plan, err = database.GetResumePlanByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load resume plan from database: %w", err)
		}
		if plan == nil {
			return fmt.Errorf("resume plan not found for run %s", runID)
		}
	}

	// Materialize bullets
	selectedBullets, err := selection.MaterializeBullets(plan, experienceBank)
	if err != nil {
		var selectionErr *selection.Error
		if errors.As(err, &selectionErr) {
			return fmt.Errorf("failed to materialize bullets: %w", err)
		}
		return fmt.Errorf("failed to materialize bullets: %w", err)
	}

	if useFiles {
		// File mode: write to file
		jsonOutput, err := json.MarshalIndent(selectedBullets, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal selected bullets to JSON: %w", err)
		}

		outputDir := filepath.Dir(materializeOutput)
		if outputDir != "" && outputDir != "." {
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
			}
		}

		if err := os.WriteFile(materializeOutput, jsonOutput, 0644); err != nil {
			return fmt.Errorf("failed to write selected bullets to output file %s: %w", materializeOutput, err)
		}

		// Validate output against schema
		schemaPath := schemas.ResolveSchemaPath("schemas/bullets.schema.json")
		if schemaPath != "" {
			if err := schemas.ValidateJSON(schemaPath, materializeOutput); err != nil {
				var validationErr *schemas.ValidationError
				var schemaLoadErr *schemas.SchemaLoadError
				if errors.As(err, &validationErr) {
					return fmt.Errorf("generated selected bullets do not validate against schema: %w", err)
				} else if errors.As(err, &schemaLoadErr) {
					_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema (schema loading failed): %v\n", err)
				} else {
					_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema: %v\n", err)
				}
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully materialized %d bullets\n", len(selectedBullets.Bullets))
		_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", materializeOutput)
	} else {
		// Database mode: save to database
		// Get resume plan ID for linking
		resumePlan, err := database.GetRunResumePlan(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to get resume plan: %w", err)
		}
		var planID *uuid.UUID
		if resumePlan != nil {
			planID = &resumePlan.ID
		}

		// Convert to RunSelectedBulletInput format
		var inputs []db.RunSelectedBulletInput
		for i, bullet := range selectedBullets.Bullets {
			input := db.RunSelectedBulletInput{
				BulletIDText: bullet.ID,
				StoryIDText:  bullet.StoryID,
				Text:         bullet.Text,
				Skills:       bullet.Skills,
				Metrics:      bullet.Metrics,
				LengthChars:  bullet.LengthChars,
				Section:      db.SectionExperience, // Default section
				Ordinal:      i + 1,
			}
			inputs = append(inputs, input)
		}

		_, err = database.SaveRunSelectedBullets(ctx, runID, planID, inputs)
		if err != nil {
			return fmt.Errorf("failed to save selected bullets to database: %w", err)
		}

		// Also save as artifact
		if err := database.SaveArtifact(ctx, runID, db.StepSelectedBullets, db.CategoryExperience, selectedBullets); err != nil {
			return fmt.Errorf("failed to save selected bullets artifact: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully materialized %d bullets and saved to database (run: %s)\n", len(selectedBullets.Bullets), runID)
	}

	return nil
}
