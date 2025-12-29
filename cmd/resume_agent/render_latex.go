// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/rendering"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/spf13/cobra"
)

var renderLaTeXCmd = &cobra.Command{
	Use:   "render-latex",
	Short: "Render a LaTeX resume from a template",
	Long:  "Generates a strictly formatted one-page LaTeX resume from a ResumePlan and RewrittenBullets using a locked template.",
	RunE:  runRenderLaTeX,
}

var (
	renderLaTeXPlanFile     string
	renderLaTeXBulletsFile  string
	renderLaTeXRunID        string
	renderLaTeXTemplateFile string
	renderLaTeXName         string
	renderLaTeXEmail        string
	renderLaTeXPhone        string
	renderLaTeXOutputFile   string
	renderLaTeXUserID       string
	renderLaTeXDatabaseURL  string
)

func init() {
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXPlanFile, "plan", "p", "", "Path to ResumePlan JSON file (deprecated: use --run-id)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXBulletsFile, "bullets", "b", "", "Path to RewrittenBullets JSON file (deprecated: use --run-id)")
	renderLaTeXCmd.Flags().StringVar(&renderLaTeXRunID, "run-id", "", "Run ID to load data from database (required if not using --plan/--bullets)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXTemplateFile, "template", "t", "templates/one_page_resume.tex", "Path to LaTeX template file")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXName, "name", "n", "", "Candidate name (required)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXEmail, "email", "e", "", "Candidate email (required)")
	renderLaTeXCmd.Flags().StringVar(&renderLaTeXPhone, "phone", "", "Candidate phone number (optional)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXOutputFile, "out", "o", "", "Path to output LaTeX file (optional with --run-id, for debugging)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXUserID, "user-id", "u", "", "User ID (required with --run-id)")
	renderLaTeXCmd.Flags().StringVar(&renderLaTeXDatabaseURL, "db-url", "", "Database URL (required with --run-id)")

	rootCmd.AddCommand(renderLaTeXCmd)
}

func runRenderLaTeX(_ *cobra.Command, _ []string) error {
	// Determine mode: database or file
	useDatabase := renderLaTeXRunID != ""
	useFiles := renderLaTeXPlanFile != "" || renderLaTeXBulletsFile != ""

	if useDatabase && useFiles {
		return fmt.Errorf("cannot use --run-id with --plan/--bullets/--out flags")
	}
	if !useDatabase && !useFiles {
		return fmt.Errorf("must provide either --run-id or --plan/--bullets/--out flags")
	}

	ctx := context.Background()

	// Get template path (use default if not provided)
	templatePath := renderLaTeXTemplateFile
	if templatePath == "" {
		templatePath = "templates/one_page_resume.tex"
	}

	// Load data
	var plan *types.ResumePlan
	var rewrittenBullets *types.RewrittenBullets
	var experienceBank *types.ExperienceBank
	var runID uuid.UUID
	var database *db.DB

	if useFiles {
		// File mode (deprecated)
		fmt.Fprintf(os.Stderr, "Warning: File-based mode is deprecated. Use --run-id instead.\n")

		planContent, err := os.ReadFile(renderLaTeXPlanFile)
		if err != nil {
			return fmt.Errorf("failed to read plan file: %w", err)
		}

		var p types.ResumePlan
		if err := json.Unmarshal(planContent, &p); err != nil {
			return fmt.Errorf("failed to unmarshal plan JSON: %w", err)
		}
		plan = &p

		bulletsContent, err := os.ReadFile(renderLaTeXBulletsFile)
		if err != nil {
			return fmt.Errorf("failed to read bullets file: %w", err)
		}

		var bullets types.RewrittenBullets
		if err := json.Unmarshal(bulletsContent, &bullets); err != nil {
			return fmt.Errorf("failed to unmarshal bullets JSON: %w", err)
		}
		rewrittenBullets = &bullets

		// Load ExperienceBank from DB if UserID is provided
		if renderLaTeXUserID != "" {
			if renderLaTeXDatabaseURL == "" {
				renderLaTeXDatabaseURL = os.Getenv("DATABASE_URL")
			}
			if renderLaTeXDatabaseURL == "" {
				return fmt.Errorf("DATABASE_URL not set and --db-url not provided (required for DB access)")
			}

			var err error
			database, err = db.Connect(ctx, renderLaTeXDatabaseURL)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer database.Close()

			uid, err := uuid.Parse(renderLaTeXUserID)
			if err != nil {
				return fmt.Errorf("invalid user-id: %w", err)
			}

			experienceBank, err = database.GetExperienceBank(ctx, uid)
			if err != nil {
				return fmt.Errorf("failed to load experience bank from DB: %w", err)
			}
		}
	} else {
		// Database mode
		var err error
		runID, err = uuid.Parse(renderLaTeXRunID)
		if err != nil {
			return fmt.Errorf("invalid run-id: %w", err)
		}

		if renderLaTeXUserID == "" {
			return fmt.Errorf("--user-id is required with --run-id")
		}

		// Connect to database
		if renderLaTeXDatabaseURL == "" {
			renderLaTeXDatabaseURL = os.Getenv("DATABASE_URL")
		}
		if renderLaTeXDatabaseURL == "" {
			return fmt.Errorf("DATABASE_URL required when using --run-id")
		}

		database, err = db.Connect(ctx, renderLaTeXDatabaseURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer database.Close()

		plan, err = database.GetResumePlanByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load resume plan from database: %w", err)
		}
		if plan == nil {
			return fmt.Errorf("resume plan not found for run %s", runID)
		}

		rewrittenBullets, err = database.GetRewrittenBulletsByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load rewritten bullets from database: %w", err)
		}
		if rewrittenBullets == nil {
			return fmt.Errorf("rewritten bullets not found for run %s", runID)
		}

		// Load experience bank
		uid, err := uuid.Parse(renderLaTeXUserID)
		if err != nil {
			return fmt.Errorf("invalid user-id: %w", err)
		}

		experienceBank, err = database.GetExperienceBank(ctx, uid)
		if err != nil {
			return fmt.Errorf("failed to load experience bank from DB: %w", err)
		}
	}

	// Render LaTeX
	var selectedEducation []types.Education
	if experienceBank != nil {
		selectedEducation = experienceBank.Education
	}
	latex, err := rendering.RenderLaTeX(plan, rewrittenBullets, templatePath, renderLaTeXName, renderLaTeXEmail, renderLaTeXPhone, experienceBank, selectedEducation)
	if err != nil {
		return fmt.Errorf("failed to render LaTeX: %w", err)
	}

	if useFiles {
		// File mode: write to file
		outputDir := filepath.Dir(renderLaTeXOutputFile)
		if outputDir != "" && outputDir != "." {
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		if err := os.WriteFile(renderLaTeXOutputFile, []byte(latex), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully rendered LaTeX resume\n")
		_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", renderLaTeXOutputFile)
	} else {
		// Database mode: save to database
		if database == nil {
			return fmt.Errorf("database connection not available")
		}

		if err := database.SaveTextArtifact(ctx, runID, db.StepResumeTex, db.CategoryValidation, latex); err != nil {
			return fmt.Errorf("failed to save LaTeX to database: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully rendered LaTeX resume and saved to database (run: %s)\n", runID)

		// Optionally write to file if --out provided (for debugging)
		if renderLaTeXOutputFile != "" {
			outputDir := filepath.Dir(renderLaTeXOutputFile)
			if outputDir != "" && outputDir != "." {
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return fmt.Errorf("failed to create output directory: %w", err)
				}
			}

			if err := os.WriteFile(renderLaTeXOutputFile, []byte(latex), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Also written to: %s\n", renderLaTeXOutputFile)
		}
	}

	return nil
}
