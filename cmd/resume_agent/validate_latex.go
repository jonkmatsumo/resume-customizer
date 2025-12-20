// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/jonathan/resume-customizer/internal/validation"
	"github.com/spf13/cobra"
)

var validateLatexCmd = &cobra.Command{
	Use:   "validate-latex",
	Short: "Validate LaTeX resume against constraints",
	Long:  "Validates a LaTeX resume file for page count, line length, forbidden phrases, and compilation errors.",
	RunE:  runValidateLatex,
}

var (
	validateLatexInput        string
	validateLatexCompanyProfile string
	validateLatexMaxPages     int
	validateLatexMaxChars     int
	validateLatexOutput       string
)

func init() {
	validateLatexCmd.Flags().StringVarP(&validateLatexInput, "in", "i", "", "Path to LaTeX file (required)")
	validateLatexCmd.Flags().StringVarP(&validateLatexCompanyProfile, "company-profile", "c", "", "Path to CompanyProfile JSON file (optional)")
	validateLatexCmd.Flags().IntVar(&validateLatexMaxPages, "max-pages", 1, "Maximum page count")
	validateLatexCmd.Flags().IntVar(&validateLatexMaxChars, "max-chars", 90, "Maximum characters per line")
	validateLatexCmd.Flags().StringVarP(&validateLatexOutput, "out", "o", "", "Path to output Violations JSON file (required)")

	if err := validateLatexCmd.MarkFlagRequired("in"); err != nil {
		panic(fmt.Sprintf("failed to mark in flag as required: %v", err))
	}
	if err := validateLatexCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(validateLatexCmd)
}

func runValidateLatex(_ *cobra.Command, _ []string) error {
	// Validate input file exists
	if _, err := os.Stat(validateLatexInput); os.IsNotExist(err) {
		return fmt.Errorf("LaTeX file not found: %s", validateLatexInput)
	}

	// Load company profile if provided
	var companyProfile *types.CompanyProfile
	if validateLatexCompanyProfile != "" {
		content, err := os.ReadFile(validateLatexCompanyProfile)
		if err != nil {
			return fmt.Errorf("failed to read company profile file: %w", err)
		}

		var profile types.CompanyProfile
		if err := json.Unmarshal(content, &profile); err != nil {
			return fmt.Errorf("failed to unmarshal company profile JSON: %w", err)
		}
		companyProfile = &profile
	}

	// Validate constraints
	violations, err := validation.ValidateConstraints(validateLatexInput, companyProfile, validateLatexMaxPages, validateLatexMaxChars)
	if err != nil {
		var validationErr *validation.Error
		var compilationErr *validation.CompilationError
		var fileErr *validation.FileReadError
		if errors.As(err, &validationErr) || errors.As(err, &compilationErr) || errors.As(err, &fileErr) {
			return fmt.Errorf("validation failed: %w", err)
		}
		return fmt.Errorf("failed to validate LaTeX: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(validateLatexOutput)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Marshal violations to JSON
	jsonBytes, err := json.MarshalIndent(violations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal violations to JSON: %w", err)
	}

	// Write output file
	if err := os.WriteFile(validateLatexOutput, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write violations to output file: %w", err)
	}

	// Validate output against schema (non-fatal)
	schemaPath := schemas.ResolveSchemaPath("schemas/violations.schema.json")
	if schemaPath != "" {
		if err := schemas.ValidateJSON(schemaPath, validateLatexOutput); err != nil {
			var validationErr *schemas.ValidationError
			var schemaLoadErr *schemas.SchemaLoadError
			if errors.As(err, &validationErr) {
				// Actual validation failure - log warning but continue
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Generated violations do not validate against schema: %v\n", err)
			} else if errors.As(err, &schemaLoadErr) {
				// Schema loading issue - log warning and continue
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema (schema loading failed): %v\n", err)
			} else {
				// Other errors - log warning and continue
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema: %v\n", err)
			}
		}
	}

	// Output results
	if len(violations.Violations) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Validation passed: No violations found\n")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Validation found %d violation(s)\n", len(violations.Violations))
	_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", validateLatexOutput)

	// Return error to indicate violations were found (exit code 1)
	return fmt.Errorf("validation found %d violation(s)", len(violations.Violations))
}

