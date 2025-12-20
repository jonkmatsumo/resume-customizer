// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/jonathan/resume-customizer/internal/rewriting"
	"github.com/spf13/cobra"
)

var rewriteCmd = &cobra.Command{
	Use:   "rewrite",
	Short: "Rewrite selected bullets to match job requirements and company voice",
	Long:  "Rewrites selected bullets to align with job requirements and company brand voice, applying style constraints and length targets.",
	RunE:  runRewrite,
}

var (
	rewriteSelectedFile      string
	rewriteJobProfileFile    string
	rewriteCompanyProfileFile string
	rewriteOutputFile        string
	rewriteAPIKey            string
)

func init() {
	rewriteCmd.Flags().StringVarP(&rewriteSelectedFile, "selected", "s", "", "Path to SelectedBullets JSON file (required)")
	rewriteCmd.Flags().StringVarP(&rewriteJobProfileFile, "job-profile", "j", "", "Path to JobProfile JSON file (required)")
	rewriteCmd.Flags().StringVarP(&rewriteCompanyProfileFile, "company-profile", "c", "", "Path to CompanyProfile JSON file (required)")
	rewriteCmd.Flags().StringVarP(&rewriteOutputFile, "out", "o", "", "Path to output RewrittenBullets JSON file (required)")
	rewriteCmd.Flags().StringVar(&rewriteAPIKey, "api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")

	if err := rewriteCmd.MarkFlagRequired("selected"); err != nil {
		panic(fmt.Sprintf("failed to mark selected flag as required: %v", err))
	}
	if err := rewriteCmd.MarkFlagRequired("job-profile"); err != nil {
		panic(fmt.Sprintf("failed to mark job-profile flag as required: %v", err))
	}
	if err := rewriteCmd.MarkFlagRequired("company-profile"); err != nil {
		panic(fmt.Sprintf("failed to mark company-profile flag as required: %v", err))
	}
	if err := rewriteCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(rewriteCmd)
}

func runRewrite(_ *cobra.Command, _ []string) error {
	// Read and unmarshal SelectedBullets JSON file
	selectedContent, err := os.ReadFile(rewriteSelectedFile)
	if err != nil {
		return fmt.Errorf("failed to read selected bullets file: %w", err)
	}

	var selectedBullets types.SelectedBullets
	if err := json.Unmarshal(selectedContent, &selectedBullets); err != nil {
		return fmt.Errorf("failed to unmarshal selected bullets JSON: %w", err)
	}

	// Read and unmarshal JobProfile JSON file
	jobProfileContent, err := os.ReadFile(rewriteJobProfileFile)
	if err != nil {
		return fmt.Errorf("failed to read job profile file: %w", err)
	}

	var jobProfile types.JobProfile
	if err := json.Unmarshal(jobProfileContent, &jobProfile); err != nil {
		return fmt.Errorf("failed to unmarshal job profile JSON: %w", err)
	}

	// Read and unmarshal CompanyProfile JSON file
	companyProfileContent, err := os.ReadFile(rewriteCompanyProfileFile)
	if err != nil {
		return fmt.Errorf("failed to read company profile file: %w", err)
	}

	var companyProfile types.CompanyProfile
	if err := json.Unmarshal(companyProfileContent, &companyProfile); err != nil {
		return fmt.Errorf("failed to unmarshal company profile JSON: %w", err)
	}

	// Get API key from flag or environment
	apiKey := rewriteAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required (set GEMINI_API_KEY environment variable or use --api-key flag)")
	}

	// Ensure output directory exists (create early, before API call)
	outputDir := filepath.Dir(rewriteOutputFile)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Rewrite bullets
	ctx := context.Background()
	rewritten, err := rewriting.RewriteBullets(ctx, &selectedBullets, &jobProfile, &companyProfile, apiKey)
	if err != nil {
		return fmt.Errorf("failed to rewrite bullets: %w", err)
	}

	// Marshal to JSON with indentation
	jsonBytes, err := json.MarshalIndent(rewritten, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write output file
	if err := os.WriteFile(rewriteOutputFile, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// Validate against schema (if schema file exists)
	schemaPath := schemas.ResolveSchemaPath("schemas/bullets.schema.json")
	if schemaPath != "" {
		if err := schemas.ValidateJSON(schemaPath, rewriteOutputFile); err != nil {
			// Distinguish between validation errors (data doesn't match schema) and schema load errors
			var validationErr *schemas.ValidationError
			var schemaLoadErr *schemas.SchemaLoadError
			if errors.As(err, &validationErr) {
				// Actual validation failure - return error
				return fmt.Errorf("generated JSON does not validate against schema: %w", err)
			} else if errors.As(err, &schemaLoadErr) {
				// Schema loading issue - log warning and continue
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema (schema loading failed): %v\n", err)
			} else {
				// Other errors - log warning and continue
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema: %v\n", err)
			}
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully rewritten %d bullets\n", len(rewritten.Bullets))
	_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", rewriteOutputFile)

	return nil
}

