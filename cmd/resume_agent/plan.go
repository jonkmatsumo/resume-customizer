// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/experience"
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
	planRanked     string
	planJobProfile string
	planExperience string
	planMaxBullets int
	planMaxLines   int
	planOutput     string
)

func init() {
	planCmd.Flags().StringVarP(&planRanked, "ranked", "r", "", "Path to RankedStories JSON file (required)")
	planCmd.Flags().StringVarP(&planJobProfile, "job-profile", "j", "", "Path to JobProfile JSON file (required)")
	planCmd.Flags().StringVarP(&planExperience, "experience", "e", "", "Path to ExperienceBank JSON file (required)")
	planCmd.Flags().IntVar(&planMaxBullets, "max-bullets", 0, "Maximum bullets allowed (required)")
	planCmd.Flags().IntVar(&planMaxLines, "max-lines", 0, "Maximum lines allowed (required)")
	planCmd.Flags().StringVarP(&planOutput, "out", "o", "", "Path to output ResumePlan JSON file (required)")

	if err := planCmd.MarkFlagRequired("ranked"); err != nil {
		panic(fmt.Sprintf("failed to mark ranked flag as required: %v", err))
	}
	if err := planCmd.MarkFlagRequired("job-profile"); err != nil {
		panic(fmt.Sprintf("failed to mark job-profile flag as required: %v", err))
	}
	if err := planCmd.MarkFlagRequired("experience"); err != nil {
		panic(fmt.Sprintf("failed to mark experience flag as required: %v", err))
	}
	if err := planCmd.MarkFlagRequired("max-bullets"); err != nil {
		panic(fmt.Sprintf("failed to mark max-bullets flag as required: %v", err))
	}
	if err := planCmd.MarkFlagRequired("max-lines"); err != nil {
		panic(fmt.Sprintf("failed to mark max-lines flag as required: %v", err))
	}
	if err := planCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
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

	// 1. Load RankedStories
	rankedContent, err := os.ReadFile(planRanked)
	if err != nil {
		return fmt.Errorf("failed to read ranked stories file %s: %w", planRanked, err)
	}

	var rankedStories types.RankedStories
	if err := json.Unmarshal(rankedContent, &rankedStories); err != nil {
		return fmt.Errorf("failed to unmarshal ranked stories JSON: %w", err)
	}

	// 2. Load JobProfile
	jobProfileContent, err := os.ReadFile(planJobProfile)
	if err != nil {
		return fmt.Errorf("failed to read job profile file %s: %w", planJobProfile, err)
	}

	var jobProfile types.JobProfile
	if err := json.Unmarshal(jobProfileContent, &jobProfile); err != nil {
		return fmt.Errorf("failed to unmarshal job profile JSON: %w", err)
	}

	// 3. Load ExperienceBank
	experienceBank, err := experience.LoadExperienceBank(planExperience)
	if err != nil {
		return fmt.Errorf("failed to load experience bank: %w", err)
	}

	// 4. Create SpaceBudget
	spaceBudget := &types.SpaceBudget{
		MaxBullets: planMaxBullets,
		MaxLines:   planMaxLines,
	}

	// 5. Select plan
	resumePlan, err := selection.SelectPlan(&rankedStories, &jobProfile, experienceBank, spaceBudget)
	if err != nil {
		return fmt.Errorf("failed to select plan: %w", err)
	}

	// 6. Marshal to JSON with indentation
	jsonOutput, err := json.MarshalIndent(resumePlan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal resume plan to JSON: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(planOutput)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
		}
	}

	// 7. Write to output file
	if err := os.WriteFile(planOutput, jsonOutput, 0644); err != nil {
		return fmt.Errorf("failed to write resume plan to output file %s: %w", planOutput, err)
	}

	// 8. Validate output against schema (if schema file exists)
	schemaPath := schemas.ResolveSchemaPath("schemas/resume_plan.schema.json")
	if schemaPath != "" {
		if err := schemas.ValidateJSON(schemaPath, planOutput); err != nil {
			// Distinguish between validation errors and schema load errors
			var validationErr *schemas.ValidationError
			var schemaLoadErr *schemas.SchemaLoadError
			if errors.As(err, &validationErr) {
				// Actual validation failure - return error
				return fmt.Errorf("generated resume plan does not validate against schema: %w", err)
			} else if errors.As(err, &schemaLoadErr) {
				// Schema loading issue - log warning and continue
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema (schema loading failed): %v\n", err)
			} else {
				// Other errors - log warning and continue
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema: %v\n", err)
			}
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully created resume plan with %d selected stories\n", len(resumePlan.SelectedStories))
	_, _ = fmt.Fprintf(os.Stdout, "Coverage score: %.2f\n", resumePlan.Coverage.CoverageScore)
	_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", planOutput)

	return nil
}
