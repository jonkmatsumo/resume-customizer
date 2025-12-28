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
	renderLaTeXTemplateFile string
	renderLaTeXName         string
	renderLaTeXEmail        string
	renderLaTeXPhone        string
	renderLaTeXOutputFile   string
	renderLaTeXUserID       string
	renderLaTeXDatabaseURL  string
)

func init() {
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXPlanFile, "plan", "p", "", "Path to ResumePlan JSON file (required)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXBulletsFile, "bullets", "b", "", "Path to RewrittenBullets JSON file (required)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXTemplateFile, "template", "t", "templates/one_page_resume.tex", "Path to LaTeX template file")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXName, "name", "n", "", "Candidate name (required)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXEmail, "email", "e", "", "Candidate email (required)")
	renderLaTeXCmd.Flags().StringVar(&renderLaTeXPhone, "phone", "", "Candidate phone number (optional)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXOutputFile, "out", "o", "", "Path to output LaTeX file (required)")
	renderLaTeXCmd.Flags().StringVarP(&renderLaTeXUserID, "user-id", "u", "", "User ID (optional, but recommended for full data)")
	renderLaTeXCmd.Flags().StringVar(&renderLaTeXDatabaseURL, "db-url", "", "Database URL (optional)")

	if err := renderLaTeXCmd.MarkFlagRequired("plan"); err != nil {
		panic(fmt.Sprintf("failed to mark plan flag as required: %v", err))
	}
	if err := renderLaTeXCmd.MarkFlagRequired("bullets"); err != nil {
		panic(fmt.Sprintf("failed to mark bullets flag as required: %v", err))
	}
	if err := renderLaTeXCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Sprintf("failed to mark name flag as required: %v", err))
	}
	if err := renderLaTeXCmd.MarkFlagRequired("email"); err != nil {
		panic(fmt.Sprintf("failed to mark email flag as required: %v", err))
	}
	if err := renderLaTeXCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(renderLaTeXCmd)
}

func runRenderLaTeX(_ *cobra.Command, _ []string) error {
	// Read and unmarshal ResumePlan JSON file
	planContent, err := os.ReadFile(renderLaTeXPlanFile)
	if err != nil {
		return fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan types.ResumePlan
	if err := json.Unmarshal(planContent, &plan); err != nil {
		return fmt.Errorf("failed to unmarshal plan JSON: %w", err)
	}

	// Read and unmarshal RewrittenBullets JSON file
	bulletsContent, err := os.ReadFile(renderLaTeXBulletsFile)
	if err != nil {
		return fmt.Errorf("failed to read bullets file: %w", err)
	}

	var rewrittenBullets types.RewrittenBullets
	if err := json.Unmarshal(bulletsContent, &rewrittenBullets); err != nil {
		return fmt.Errorf("failed to unmarshal bullets JSON: %w", err)
	}

	// Get template path (use default if not provided)
	templatePath := renderLaTeXTemplateFile
	if templatePath == "" {
		templatePath = "templates/one_page_resume.tex"
	}

	// Load ExperienceBank from DB if UserID is provided
	var experienceBank *types.ExperienceBank
	if renderLaTeXUserID != "" {
		ctx := context.Background()

		if renderLaTeXDatabaseURL == "" {
			renderLaTeXDatabaseURL = os.Getenv("DATABASE_URL")
		}
		if renderLaTeXDatabaseURL == "" {
			return fmt.Errorf("DATABASE_URL not set and --db-url not provided (required for DB access)")
		}

		database, err := db.Connect(ctx, renderLaTeXDatabaseURL)
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

	// Ensure output directory exists
	outputDir := filepath.Dir(renderLaTeXOutputFile)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Render LaTeX
	var selectedEducation []types.Education
	if experienceBank != nil {
		selectedEducation = experienceBank.Education
	}
	latex, err := rendering.RenderLaTeX(&plan, &rewrittenBullets, templatePath, renderLaTeXName, renderLaTeXEmail, renderLaTeXPhone, experienceBank, selectedEducation)
	if err != nil {
		return fmt.Errorf("failed to render LaTeX: %w", err)
	}

	// Write output file
	if err := os.WriteFile(renderLaTeXOutputFile, []byte(latex), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully rendered LaTeX resume\n")
	_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", renderLaTeXOutputFile)

	return nil
}
