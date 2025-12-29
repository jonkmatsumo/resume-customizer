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
	"github.com/jonathan/resume-customizer/internal/rewriting"
	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/spf13/cobra"
)

var rewriteCmd = &cobra.Command{
	Use:   "rewrite",
	Short: "Rewrite selected bullets to match job requirements and company voice",
	Long:  "Rewrites selected bullets to align with job requirements and company brand voice, applying style constraints and length targets.",
	RunE:  runRewrite,
}

var (
	rewriteSelectedFile       string
	rewriteJobProfileFile     string
	rewriteCompanyProfileFile string
	rewriteRunID              string
	rewriteDatabaseURL        string
	rewriteOutputFile         string
	rewriteAPIKey             string
)

func init() {
	rewriteCmd.Flags().StringVarP(&rewriteSelectedFile, "selected", "s", "", "Path to SelectedBullets JSON file (deprecated: use --run-id)")
	rewriteCmd.Flags().StringVarP(&rewriteJobProfileFile, "job-profile", "j", "", "Path to JobProfile JSON file (deprecated: use --run-id)")
	rewriteCmd.Flags().StringVarP(&rewriteCompanyProfileFile, "company-profile", "c", "", "Path to CompanyProfile JSON file (deprecated: use --run-id)")
	rewriteCmd.Flags().StringVar(&rewriteRunID, "run-id", "", "Run ID to load data from database (required if not using file flags)")
	rewriteCmd.Flags().StringVar(&rewriteDatabaseURL, "db-url", "", "Database URL (required with --run-id)")
	rewriteCmd.Flags().StringVarP(&rewriteOutputFile, "out", "o", "", "Path to output RewrittenBullets JSON file (deprecated: use --run-id)")
	rewriteCmd.Flags().StringVar(&rewriteAPIKey, "api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")

	rootCmd.AddCommand(rewriteCmd)
}

func runRewrite(_ *cobra.Command, _ []string) error {
	// Determine mode: database or file
	useDatabase := rewriteRunID != ""
	useFiles := rewriteSelectedFile != "" || rewriteJobProfileFile != "" || rewriteCompanyProfileFile != ""

	if useDatabase && useFiles {
		return fmt.Errorf("cannot use --run-id with --selected/--job-profile/--company-profile/--out flags")
	}
	if !useDatabase && !useFiles {
		return fmt.Errorf("must provide either --run-id or --selected/--job-profile/--company-profile/--out flags")
	}

	// Get API key from flag or environment
	apiKey := rewriteAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required (set GEMINI_API_KEY environment variable or use --api-key flag)")
	}

	ctx := context.Background()

	// Load data
	var selectedBullets *types.SelectedBullets
	var jobProfile *types.JobProfile
	var companyProfile *types.CompanyProfile
	var runID uuid.UUID

	if useFiles {
		// File mode (deprecated)
		fmt.Fprintf(os.Stderr, "Warning: File-based mode is deprecated. Use --run-id instead.\n")

		selectedContent, err := os.ReadFile(rewriteSelectedFile)
		if err != nil {
			return fmt.Errorf("failed to read selected bullets file: %w", err)
		}

		var selected types.SelectedBullets
		if err := json.Unmarshal(selectedContent, &selected); err != nil {
			return fmt.Errorf("failed to unmarshal selected bullets JSON: %w", err)
		}
		selectedBullets = &selected

		jobProfileContent, err := os.ReadFile(rewriteJobProfileFile)
		if err != nil {
			return fmt.Errorf("failed to read job profile file: %w", err)
		}

		var job types.JobProfile
		if err := json.Unmarshal(jobProfileContent, &job); err != nil {
			return fmt.Errorf("failed to unmarshal job profile JSON: %w", err)
		}
		jobProfile = &job

		companyProfileContent, err := os.ReadFile(rewriteCompanyProfileFile)
		if err != nil {
			return fmt.Errorf("failed to read company profile file: %w", err)
		}

		var company types.CompanyProfile
		if err := json.Unmarshal(companyProfileContent, &company); err != nil {
			return fmt.Errorf("failed to unmarshal company profile JSON: %w", err)
		}
		companyProfile = &company
	} else {
		// Database mode
		var err error
		runID, err = uuid.Parse(rewriteRunID)
		if err != nil {
			return fmt.Errorf("invalid run-id: %w", err)
		}

		// Connect to database
		if rewriteDatabaseURL == "" {
			rewriteDatabaseURL = os.Getenv("DATABASE_URL")
		}
		if rewriteDatabaseURL == "" {
			return fmt.Errorf("DATABASE_URL required when using --run-id")
		}

		database, err := db.Connect(ctx, rewriteDatabaseURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer database.Close()

		selectedBullets, err = database.GetSelectedBulletsByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load selected bullets from database: %w", err)
		}
		if selectedBullets == nil {
			return fmt.Errorf("selected bullets not found for run %s", runID)
		}

		jobProfile, err = database.GetJobProfileByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load job profile from database: %w", err)
		}
		if jobProfile == nil {
			return fmt.Errorf("job profile not found for run %s", runID)
		}

		companyProfile, err = database.GetCompanyProfileByRunID(ctx, runID)
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

	if useFiles {
		// Rewrite bullets
		rewritten, err := rewriting.RewriteBullets(ctx, selectedBullets, jobProfile, companyProfile, apiKey)
		if err != nil {
			return fmt.Errorf("failed to rewrite bullets: %w", err)
		}
		// File mode: write to file
		outputDir := filepath.Dir(rewriteOutputFile)
		if outputDir != "" && outputDir != "." {
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		jsonBytes, err := json.MarshalIndent(rewritten, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		if err := os.WriteFile(rewriteOutputFile, jsonBytes, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		// Validate against schema
		schemaPath := schemas.ResolveSchemaPath("schemas/bullets.schema.json")
		if schemaPath != "" {
			if err := schemas.ValidateJSON(schemaPath, rewriteOutputFile); err != nil {
				var validationErr *schemas.ValidationError
				var schemaLoadErr *schemas.SchemaLoadError
				if errors.As(err, &validationErr) {
					return fmt.Errorf("generated JSON does not validate against schema: %w", err)
				} else if errors.As(err, &schemaLoadErr) {
					_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema (schema loading failed): %v\n", err)
				} else {
					_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema: %v\n", err)
				}
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully rewritten %d bullets\n", len(rewritten.Bullets))
		_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", rewriteOutputFile)
	}

	return nil
}
