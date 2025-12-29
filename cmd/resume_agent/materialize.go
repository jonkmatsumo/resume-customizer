// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/selection"
	"github.com/spf13/cobra"
)

var materializeCmd = &cobra.Command{
	Use:   "materialize",
	Short: "Extract selected bullets from a resume plan",
	Long:  "Materializes (extracts) the actual bullet data from ExperienceBank based on a ResumePlan from the database, pulling the exact bullets for rewriting and saving to the database.",
	RunE:  runMaterialize,
}

var (
	materializeUserID      string
	materializeRunID       string
	materializeDatabaseURL string
)

func init() {
	materializeCmd.Flags().StringVarP(&materializeUserID, "user-id", "u", "", "User ID (required)")
	materializeCmd.Flags().StringVar(&materializeRunID, "run-id", "", "Run ID to load resume plan from database (required)")
	materializeCmd.Flags().StringVar(&materializeDatabaseURL, "db-url", "", "Database URL (required)")

	if err := materializeCmd.MarkFlagRequired("user-id"); err != nil {
		panic(fmt.Sprintf("failed to mark user-id flag as required: %v", err))
	}
	if err := materializeCmd.MarkFlagRequired("run-id"); err != nil {
		panic(fmt.Sprintf("failed to mark run-id flag as required: %v", err))
	}
	if err := materializeCmd.MarkFlagRequired("db-url"); err != nil {
		panic(fmt.Sprintf("failed to mark db-url flag as required: %v", err))
	}

	rootCmd.AddCommand(materializeCmd)
}

func runMaterialize(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Connect to database
	if materializeDatabaseURL == "" {
		materializeDatabaseURL = os.Getenv("DATABASE_URL")
	}
	if materializeDatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL not set (set DATABASE_URL environment variable or use --db-url flag)")
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

	// Load ExperienceBank from DB
	experienceBank, err := database.GetExperienceBank(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to load experience bank from DB: %w", err)
	}

	// Parse run ID
	runID, err := uuid.Parse(materializeRunID)
	if err != nil {
		return fmt.Errorf("invalid run-id: %w", err)
	}

	// Load resume plan from database
	plan, err := database.GetResumePlanByRunID(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load resume plan from database: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("resume plan not found for run %s", runID)
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

	// Save to database
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

	return nil
}
