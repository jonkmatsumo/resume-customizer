// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"context"
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
	Long:  "Generates a strictly formatted one-page LaTeX resume from a ResumePlan and RewrittenBullets using a locked template. Loads data from and saves results to the database.",
	RunE:  runRenderLaTeX,
}

var (
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
	renderLaTeXCmd.Flags().StringVar(&renderLaTeXRunID, "run-id", "", "Run ID to load data from database (required)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXTemplateFile, "template", "t", "templates/one_page_resume.tex", "Path to LaTeX template file")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXName, "name", "n", "", "Candidate name (required)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXEmail, "email", "e", "", "Candidate email (required)")
	renderLaTeXCmd.Flags().StringVar(&renderLaTeXPhone, "phone", "", "Candidate phone number (optional)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXOutputFile, "out", "o", "", "Path to output LaTeX file (optional, for debugging)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXUserID, "user-id", "u", "", "User ID (required)")
	renderLaTeXCmd.Flags().StringVar(&renderLaTeXDatabaseURL, "db-url", "", "Database URL (required)")

	if err := renderLaTeXCmd.MarkFlagRequired("run-id"); err != nil {
		panic(fmt.Sprintf("failed to mark run-id flag as required: %v", err))
	}
	if err := renderLaTeXCmd.MarkFlagRequired("user-id"); err != nil {
		panic(fmt.Sprintf("failed to mark user-id flag as required: %v", err))
	}
	if err := renderLaTeXCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Sprintf("failed to mark name flag as required: %v", err))
	}
	if err := renderLaTeXCmd.MarkFlagRequired("email"); err != nil {
		panic(fmt.Sprintf("failed to mark email flag as required: %v", err))
	}
	if err := renderLaTeXCmd.MarkFlagRequired("db-url"); err != nil {
		panic(fmt.Sprintf("failed to mark db-url flag as required: %v", err))
	}

	rootCmd.AddCommand(renderLaTeXCmd)
}

func runRenderLaTeX(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Get template path (use default if not provided)
	templatePath := renderLaTeXTemplateFile
	if templatePath == "" {
		templatePath = "templates/one_page_resume.tex"
	}

	// Parse run ID
	runID, err := uuid.Parse(renderLaTeXRunID)
	if err != nil {
		return fmt.Errorf("invalid run-id: %w", err)
	}

	// Connect to database
	if renderLaTeXDatabaseURL == "" {
		renderLaTeXDatabaseURL = os.Getenv("DATABASE_URL")
	}
	if renderLaTeXDatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL not set (set DATABASE_URL environment variable or use --db-url flag)")
	}

	database, err := db.Connect(ctx, renderLaTeXDatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	// Load data from database
	plan, err := database.GetResumePlanByRunID(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load resume plan from database: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("resume plan not found for run %s", runID)
	}

	rewrittenBullets, err := database.GetRewrittenBulletsByRunID(ctx, runID)
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

	experienceBank, err := database.GetExperienceBank(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to load experience bank from DB: %w", err)
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

	// Save to database
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

	return nil
}
