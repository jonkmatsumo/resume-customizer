package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jonathan/resume-customizer/internal/parsing"
	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/spf13/cobra"
)

var parseJobCmd = &cobra.Command{
	Use:   "parse-job",
	Short: "Parse a cleaned job posting into structured JobProfile JSON",
	Long:  "Parse a cleaned job posting text file into a structured JobProfile JSON that validates against the job_profile schema.",
	RunE:  runParseJob,
}

var (
	parseInputFile  string
	parseOutputFile string
	parseAPIKey     string
)

func init() {
	parseJobCmd.Flags().StringVarP(&parseInputFile, "in", "i", "", "Path to cleaned text file (required)")
	parseJobCmd.Flags().StringVarP(&parseOutputFile, "out", "o", "", "Path to output JSON file (required)")
	parseJobCmd.Flags().StringVar(&parseAPIKey, "api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")

	parseJobCmd.MarkFlagRequired("in")
	parseJobCmd.MarkFlagRequired("out")

	rootCmd.AddCommand(parseJobCmd)
}

func runParseJob(cmd *cobra.Command, args []string) error {
	// Read input file
	inputContent, err := os.ReadFile(parseInputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Get API key
	apiKey := parseAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required (set GEMINI_API_KEY environment variable or use --api-key flag)")
	}

	// Parse job profile
	ctx := context.Background()
	profile, err := parsing.ParseJobProfile(ctx, string(inputContent), apiKey)
	if err != nil {
		return fmt.Errorf("failed to parse job profile: %w", err)
	}

	// Marshal to JSON with indentation
	jsonBytes, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write output file
	if err := os.WriteFile(parseOutputFile, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// Validate against schema
	schemaPath := "schemas/job_profile.schema.json"
	if err := schemas.ValidateJSON(schemaPath, parseOutputFile); err != nil {
		return fmt.Errorf("generated JSON does not validate against schema: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Successfully parsed job profile\n")
	fmt.Fprintf(os.Stdout, "Output: %s\n", parseOutputFile)

	return nil
}
