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
	"github.com/jonathan/resume-customizer/internal/voice"
	"github.com/spf13/cobra"
)

var summarizeVoiceCmd = &cobra.Command{
	Use:   "summarize-voice",
	Short: "Extract brand voice and style rules from company corpus",
	Long:  "Analyzes a company corpus text and extracts structured brand voice information (tone, style rules, taboo phrases, domain context, values) using the Gemini API.",
	RunE:  runSummarizeVoice,
}

var (
	summarizeVoiceInputFile   string
	summarizeVoiceSourcesFile string
	summarizeVoiceOutputFile  string
	summarizeVoiceAPIKey      string
)

func init() {
	summarizeVoiceCmd.Flags().StringVarP(&summarizeVoiceInputFile, "in", "i", "", "Path to corpus text file (required)")
	summarizeVoiceCmd.Flags().StringVarP(&summarizeVoiceSourcesFile, "sources", "s", "", "Path to sources JSON file (required)")
	summarizeVoiceCmd.Flags().StringVarP(&summarizeVoiceOutputFile, "out", "o", "", "Path to output CompanyProfile JSON file (required)")
	summarizeVoiceCmd.Flags().StringVar(&summarizeVoiceAPIKey, "api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")

	if err := summarizeVoiceCmd.MarkFlagRequired("in"); err != nil {
		panic(fmt.Sprintf("failed to mark in flag as required: %v", err))
	}
	if err := summarizeVoiceCmd.MarkFlagRequired("sources"); err != nil {
		panic(fmt.Sprintf("failed to mark sources flag as required: %v", err))
	}
	if err := summarizeVoiceCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(summarizeVoiceCmd)
}

func runSummarizeVoice(_ *cobra.Command, _ []string) error {
	// Read corpus text file
	corpusContent, err := os.ReadFile(summarizeVoiceInputFile)
	if err != nil {
		return fmt.Errorf("failed to read corpus file: %w", err)
	}

	// Read and unmarshal sources JSON file
	sourcesContent, err := os.ReadFile(summarizeVoiceSourcesFile)
	if err != nil {
		return fmt.Errorf("failed to read sources file: %w", err)
	}

	var sources []types.Source
	if err := json.Unmarshal(sourcesContent, &sources); err != nil {
		return fmt.Errorf("failed to unmarshal sources JSON: %w", err)
	}

	// Get API key from flag or environment
	apiKey := summarizeVoiceAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required (set GEMINI_API_KEY environment variable or use --api-key flag)")
	}

	// Ensure output directory exists (create early, before API call)
	outputDir := filepath.Dir(summarizeVoiceOutputFile)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Summarize voice
	ctx := context.Background()
	profile, err := voice.SummarizeVoice(ctx, string(corpusContent), sources, apiKey)
	if err != nil {
		return fmt.Errorf("failed to summarize voice: %w", err)
	}

	// Marshal to JSON with indentation
	jsonBytes, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write output file
	if err := os.WriteFile(summarizeVoiceOutputFile, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// Validate against schema (if schema file exists)
	schemaPath := schemas.ResolveSchemaPath("schemas/company_profile.schema.json")
	if schemaPath != "" {
		if err := schemas.ValidateJSON(schemaPath, summarizeVoiceOutputFile); err != nil {
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

	_, _ = fmt.Fprintf(os.Stdout, "Successfully summarized brand voice\n")
	_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", summarizeVoiceOutputFile)

	return nil
}
