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

var materializeCmd = &cobra.Command{
	Use:   "materialize",
	Short: "Extract selected bullets from a resume plan",
	Long:  "Materializes (extracts) the actual bullet data from ExperienceBank based on a ResumePlan, pulling the exact bullets for rewriting.",
	RunE:  runMaterialize,
}

var (
	materializePlan       string
	materializeExperience string
	materializeOutput     string
)

func init() {
	materializeCmd.Flags().StringVarP(&materializePlan, "plan", "p", "", "Path to ResumePlan JSON file (required)")
	materializeCmd.Flags().StringVarP(&materializeExperience, "experience", "e", "", "Path to ExperienceBank JSON file (required)")
	materializeCmd.Flags().StringVarP(&materializeOutput, "out", "o", "", "Path to output SelectedBullets JSON file (required)")

	if err := materializeCmd.MarkFlagRequired("plan"); err != nil {
		panic(fmt.Sprintf("failed to mark plan flag as required: %v", err))
	}
	if err := materializeCmd.MarkFlagRequired("experience"); err != nil {
		panic(fmt.Sprintf("failed to mark experience flag as required: %v", err))
	}
	if err := materializeCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(materializeCmd)
}

func runMaterialize(_ *cobra.Command, _ []string) error {
	// 1. Load ResumePlan
	planContent, err := os.ReadFile(materializePlan)
	if err != nil {
		return fmt.Errorf("failed to read plan file %s: %w", materializePlan, err)
	}

	var plan types.ResumePlan
	if err := json.Unmarshal(planContent, &plan); err != nil {
		return fmt.Errorf("failed to unmarshal plan JSON: %w", err)
	}

	// 2. Load ExperienceBank
	experienceBank, err := experience.LoadExperienceBank(materializeExperience)
	if err != nil {
		return fmt.Errorf("failed to load experience bank: %w", err)
	}

	// 3. Materialize bullets
	selectedBullets, err := selection.MaterializeBullets(&plan, experienceBank)
	if err != nil {
		var selectionErr *selection.Error
		if errors.As(err, &selectionErr) {
			return fmt.Errorf("failed to materialize bullets: %w", err)
		}
		return fmt.Errorf("failed to materialize bullets: %w", err)
	}

	// 4. Marshal to JSON with indentation
	jsonOutput, err := json.MarshalIndent(selectedBullets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal selected bullets to JSON: %w", err)
	}

	// 5. Ensure output directory exists
	outputDir := filepath.Dir(materializeOutput)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
		}
	}

	// 6. Write to output file
	if err := os.WriteFile(materializeOutput, jsonOutput, 0644); err != nil {
		return fmt.Errorf("failed to write selected bullets to output file %s: %w", materializeOutput, err)
	}

	// 7. Validate output against schema (if schema file exists)
	schemaPath := schemas.ResolveSchemaPath("schemas/bullets.schema.json")
	if schemaPath != "" {
		if err := schemas.ValidateJSON(schemaPath, materializeOutput); err != nil {
			// Distinguish between validation errors and schema load errors
			var validationErr *schemas.ValidationError
			var schemaLoadErr *schemas.SchemaLoadError
			if errors.As(err, &validationErr) {
				// Actual validation failure - return error
				return fmt.Errorf("generated selected bullets do not validate against schema: %w", err)
			} else if errors.As(err, &schemaLoadErr) {
				// Schema loading issue - log warning and continue
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema (schema loading failed): %v\n", err)
			} else {
				// Other errors - log warning and continue
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema: %v\n", err)
			}
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully materialized %d bullets\n", len(selectedBullets.Bullets))
	_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", materializeOutput)

	return nil
}

