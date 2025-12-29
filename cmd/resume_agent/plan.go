// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/selection"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Select optimal stories and bullets for a resume plan",
	Long:  "Uses dynamic programming to optimally select experience stories and bullets under space budget constraints, maximizing relevance and skill coverage. Loads data from and saves results to the database.",
	RunE:  runPlan,
}

var (
	planUserID      string
	planRunID       string
	planDatabaseURL string
	planMaxBullets  int
	planMaxLines    int
)

func init() {
	planCmd.Flags().StringVarP(&planUserID, "user-id", "u", "", "User ID (required)")
	planCmd.Flags().StringVar(&planRunID, "run-id", "", "Run ID to load data from database (required)")
	planCmd.Flags().StringVar(&planDatabaseURL, "db-url", "", "Database URL (required)")
	planCmd.Flags().IntVar(&planMaxBullets, "max-bullets", 0, "Maximum bullets allowed (required)")
	planCmd.Flags().IntVar(&planMaxLines, "max-lines", 0, "Maximum lines allowed (required)")

	if err := planCmd.MarkFlagRequired("user-id"); err != nil {
		panic(fmt.Sprintf("failed to mark user-id flag as required: %v", err))
	}
	if err := planCmd.MarkFlagRequired("run-id"); err != nil {
		panic(fmt.Sprintf("failed to mark run-id flag as required: %v", err))
	}
	if err := planCmd.MarkFlagRequired("db-url"); err != nil {
		panic(fmt.Sprintf("failed to mark db-url flag as required: %v", err))
	}
	if err := planCmd.MarkFlagRequired("max-bullets"); err != nil {
		panic(fmt.Sprintf("failed to mark max-bullets flag as required: %v", err))
	}
	if err := planCmd.MarkFlagRequired("max-lines"); err != nil {
		panic(fmt.Sprintf("failed to mark max-lines flag as required: %v", err))
	}

	rootCmd.AddCommand(planCmd)
}

func runPlan(_ *cobra.Command, _ []string) error {
	// Validate flags
	if planMaxBullets <= 0 {
		return fmt.Errorf("max-bullets must be greater than 0, got %d", planMaxBullets)
	}
	if planMaxLines <= 0 {
		return fmt.Errorf("max-lines must be greater than 0, got %d", planMaxLines)
	}

	ctx := context.Background()

	// Connect to database
	if planDatabaseURL == "" {
		planDatabaseURL = os.Getenv("DATABASE_URL")
	}
	if planDatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL not set (set DATABASE_URL environment variable or use --db-url flag)")
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

	// Load ExperienceBank from DB
	experienceBank, err := database.GetExperienceBank(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to load experience bank from DB: %w", err)
	}

	// Parse run ID
	runID, err := uuid.Parse(planRunID)
	if err != nil {
		return fmt.Errorf("invalid run-id: %w", err)
	}

	// Load ranked stories and job profile from database
	rankedStories, err := database.GetRankedStoriesByRunID(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load ranked stories from database: %w", err)
	}
	if rankedStories == nil {
		return fmt.Errorf("ranked stories not found for run %s", runID)
	}

	jobProfile, err := database.GetJobProfileByRunID(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load job profile from database: %w", err)
	}
	if jobProfile == nil {
		return fmt.Errorf("job profile not found for run %s", runID)
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

	// Save to database
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

	return nil
}
