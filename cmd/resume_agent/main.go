package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "resume_agent",
	Short: "A schema-first, CLI-driven resume generation agent",
	Long:  "Resume Agent generates strictly formatted, one-page LaTeX resumes tailored to job postings and company brand voice.",
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a JSON file against a JSON Schema",
	Long:  "Validate a JSON file against a JSON Schema file. Returns exit code 0 on success, 1 on validation failure, 2 on usage error.",
	RunE:  runValidate,
}

var (
	schemaPath string
	jsonPath   string
)

func init() {
	validateCmd.Flags().StringVarP(&schemaPath, "schema", "s", "", "Path to JSON Schema file (required)")
	validateCmd.Flags().StringVarP(&jsonPath, "json", "j", "", "Path to JSON file to validate (required)")
	if err := validateCmd.MarkFlagRequired("schema"); err != nil {
		panic(fmt.Sprintf("failed to mark schema flag as required: %v", err))
	}
	if err := validateCmd.MarkFlagRequired("json"); err != nil {
		panic(fmt.Sprintf("failed to mark json flag as required: %v", err))
	}

	rootCmd.AddCommand(validateCmd)
}

func runValidate(_ *cobra.Command, _ []string) error {
	if schemaPath == "" {
		return fmt.Errorf("--schema flag is required")
	}
	if jsonPath == "" {
		return fmt.Errorf("--json flag is required")
	}

	err := schemas.ValidateJSON(schemaPath, jsonPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Validation failed:\n%s\n", err.Error())
		return err
	}

	fmt.Println("Validation passed")
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		// Determine exit code based on error type
		// Use errors.As to traverse the error chain and find the original error type
		var validationErr *schemas.ValidationError
		var schemaLoadErr *schemas.SchemaLoadError
		if errors.As(err, &validationErr) {
			os.Exit(1) // Validation failure
		}
		if errors.As(err, &schemaLoadErr) {
			os.Exit(2) // Schema load error (usage/configuration issue)
		}
		os.Exit(2) // Generic error
	}
}
