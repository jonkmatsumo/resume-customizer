// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/rewriting"
	"github.com/spf13/cobra"
)

var rewriteCmd = &cobra.Command{
	Use:   "rewrite",
	Short: "Rewrite selected bullets to match job requirements and company voice",
	Long:  "Rewrites selected bullets to align with job requirements and company brand voice, applying style constraints and length targets. Loads data from and saves results to the database.",
	RunE:  runRewrite,
}

var (
	rewriteRunID       string
	rewriteDatabaseURL string
	rewriteAPIKey      string
)

func init() {
	rewriteCmd.Flags().StringVar(&rewriteRunID, "run-id", "", "Run ID to load data from database (required)")
	rewriteCmd.Flags().StringVar(&rewriteDatabaseURL, "db-url", "", "Database URL (required)")
	rewriteCmd.Flags().StringVar(&rewriteAPIKey, "api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")

	if err := rewriteCmd.MarkFlagRequired("run-id"); err != nil {
		panic(fmt.Sprintf("failed to mark run-id flag as required: %v", err))
	}
	if err := rewriteCmd.MarkFlagRequired("db-url"); err != nil {
		panic(fmt.Sprintf("failed to mark db-url flag as required: %v", err))
	}

	rootCmd.AddCommand(rewriteCmd)
}

func runRewrite(_ *cobra.Command, _ []string) error {
	// Get API key from flag or environment
	apiKey := rewriteAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required (set GEMINI_API_KEY environment variable or use --api-key flag)")
	}

	ctx := context.Background()

	// Parse run ID
	runID, err := uuid.Parse(rewriteRunID)
	if err != nil {
		return fmt.Errorf("invalid run-id: %w", err)
	}

	// Connect to database
	if rewriteDatabaseURL == "" {
		rewriteDatabaseURL = os.Getenv("DATABASE_URL")
	}
	if rewriteDatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL not set (set DATABASE_URL environment variable or use --db-url flag)")
	}

	database, err := db.Connect(ctx, rewriteDatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	// Load data from database
	selectedBullets, err := database.GetSelectedBulletsByRunID(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load selected bullets from database: %w", err)
	}
	if selectedBullets == nil {
		return fmt.Errorf("selected bullets not found for run %s", runID)
	}

	jobProfile, err := database.GetJobProfileByRunID(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load job profile from database: %w", err)
	}
	if jobProfile == nil {
		return fmt.Errorf("job profile not found for run %s", runID)
	}

	companyProfile, err := database.GetCompanyProfileByRunID(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load company profile from database: %w", err)
	}
	if companyProfile == nil {
		return fmt.Errorf("company profile not found for run %s", runID)
	}

	// Rewrite bullets
	rewritten, err := rewriting.RewriteBullets(ctx, selectedBullets, jobProfile, companyProfile, apiKey)
	if err != nil {
		return fmt.Errorf("failed to rewrite bullets: %w", err)
	}

	// Get selected bullets from database to link rewritten bullets
	selectedBulletsDB, err := database.GetRunSelectedBullets(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to get selected bullets from database: %w", err)
	}

	// Create a map of bullet_id_text to selected bullet ID
	bulletIDMap := make(map[string]uuid.UUID)
	for _, sb := range selectedBulletsDB {
		bulletIDMap[sb.BulletIDText] = sb.ID
	}

	// Convert to RunRewrittenBulletInput format
	var inputs []db.RunRewrittenBulletInput
	for i, bullet := range rewritten.Bullets {
		selectedBulletID, ok := bulletIDMap[bullet.OriginalBulletID]
		var selectedID *uuid.UUID
		if ok {
			selectedID = &selectedBulletID
		}

		input := db.RunRewrittenBulletInput{
			SelectedBulletID:     selectedID,
			OriginalBulletIDText: bullet.OriginalBulletID,
			FinalText:            bullet.FinalText,
			LengthChars:          bullet.LengthChars,
			EstimatedLines:       bullet.EstimatedLines,
			StyleStrongVerb:      bullet.StyleChecks.StrongVerb,
			StyleQuantified:      bullet.StyleChecks.Quantified,
			StyleNoTaboo:         bullet.StyleChecks.NoTaboo,
			StyleTargetLength:    bullet.StyleChecks.TargetLength,
			Ordinal:              i + 1,
		}
		inputs = append(inputs, input)
	}

	_, err = database.SaveRunRewrittenBullets(ctx, runID, inputs)
	if err != nil {
		return fmt.Errorf("failed to save rewritten bullets to database: %w", err)
	}

	// Also save as artifact
	if err := database.SaveArtifact(ctx, runID, db.StepRewrittenBullets, db.CategoryRewriting, rewritten); err != nil {
		return fmt.Errorf("failed to save rewritten bullets artifact: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully rewritten %d bullets and saved to database (run: %s)\n", len(rewritten.Bullets), runID)
	return nil
}
