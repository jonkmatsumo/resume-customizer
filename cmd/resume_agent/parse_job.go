package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
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
	parseInputFile   string
	parseOutputFile  string
	parseRunID       string
	parseDatabaseURL string
	parseAPIKey      string
)

func init() {
	parseJobCmd.Flags().StringVarP(&parseInputFile, "in", "i", "", "Path to cleaned text file (deprecated: use --run-id)")
	parseJobCmd.Flags().StringVarP(&parseOutputFile, "out", "o", "", "Path to output JSON file (deprecated: use --run-id)")
	parseJobCmd.Flags().StringVar(&parseRunID, "run-id", "", "Run ID to load job posting from database (required if not using --in)")
	parseJobCmd.Flags().StringVar(&parseDatabaseURL, "db-url", "", "Database URL (required with --run-id)")
	parseJobCmd.Flags().StringVar(&parseAPIKey, "api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")

	rootCmd.AddCommand(parseJobCmd)
}

func runParseJob(_ *cobra.Command, _ []string) error {
	// Determine mode: database or file
	useDatabase := parseRunID != ""
	useFiles := parseInputFile != "" || parseOutputFile != ""

	if useDatabase && useFiles {
		return fmt.Errorf("cannot use --run-id with --in/--out flags")
	}
	if !useDatabase && !useFiles {
		return fmt.Errorf("must provide either --run-id or --in/--out flags")
	}

	// Get API key
	apiKey := parseAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required (set GEMINI_API_KEY environment variable or use --api-key flag)")
	}

	ctx := context.Background()

	if useFiles {
		// File mode (deprecated)
		fmt.Fprintf(os.Stderr, "Warning: File-based mode is deprecated. Use --run-id instead.\n")

		// Read input file
		inputContent, err := os.ReadFile(parseInputFile)
		if err != nil {
			return fmt.Errorf("failed to read input file: %w", err)
		}

		// Parse job profile
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

		// Validate against schema (if schema file exists)
		schemaPath := schemas.ResolveSchemaPath("schemas/job_profile.schema.json")
		if schemaPath != "" {
			if err := schemas.ValidateJSON(schemaPath, parseOutputFile); err != nil {
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

		_, _ = fmt.Fprintf(os.Stdout, "Successfully parsed job profile\n")
		_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", parseOutputFile)

		return nil
	}

	// Database mode
	runID, err := uuid.Parse(parseRunID)
	if err != nil {
		return fmt.Errorf("invalid run-id: %w", err)
	}

	// Connect to database
	if parseDatabaseURL == "" {
		parseDatabaseURL = os.Getenv("DATABASE_URL")
	}
	if parseDatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL required when using --run-id")
	}

	database, err := db.Connect(ctx, parseDatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	// Load run
	run, err := database.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to get run: %w", err)
	}
	if run == nil {
		return fmt.Errorf("run not found: %s", runID)
	}

	// Get job posting text from database
	var cleanedText string
	if run.JobURL != "" {
		// Try to get from job_postings by URL
		jobPosting, err := database.GetJobPostingByURL(ctx, run.JobURL)
		if err != nil {
			return fmt.Errorf("failed to get job posting by URL: %w", err)
		}
		if jobPosting != nil && jobPosting.CleanedText != nil {
			cleanedText = *jobPosting.CleanedText
		} else {
			// Fallback: try to get from text artifact
			cleanedText, err = database.GetTextArtifact(ctx, runID, db.StepJobPosting)
			if err != nil {
				return fmt.Errorf("failed to get job posting text: %w", err)
			}
		}
	} else {
		// Fallback: get from text artifact
		cleanedText, err = database.GetTextArtifact(ctx, runID, db.StepJobPosting)
		if err != nil {
			return fmt.Errorf("failed to get job posting text: %w", err)
		}
	}

	if cleanedText == "" {
		return fmt.Errorf("no cleaned text found for run")
	}

	// Parse job profile
	profile, err := parsing.ParseJobProfile(ctx, cleanedText, apiKey)
	if err != nil {
		return fmt.Errorf("failed to parse job profile: %w", err)
	}

	// Save to database as artifact
	if err := database.SaveArtifact(ctx, runID, db.StepJobProfile, db.CategoryIngestion, profile); err != nil {
		return fmt.Errorf("failed to save job profile to database: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully parsed job profile and saved to database (run: %s)\n", runID)

	return nil
}
