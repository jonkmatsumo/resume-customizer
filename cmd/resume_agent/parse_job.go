package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/parsing"
	"github.com/spf13/cobra"
)

var parseJobCmd = &cobra.Command{
	Use:   "parse-job",
	Short: "Parse a job posting into structured JobProfile",
	Long:  "Parses a job posting from the database (by run-id) into a structured JobProfile and saves it back to the database.",
	RunE:  runParseJob,
}

var (
	parseRunID       string
	parseDatabaseURL string
	parseAPIKey      string
)

func init() {
	parseJobCmd.Flags().StringVar(&parseRunID, "run-id", "", "Run ID to load job posting from database (required)")
	parseJobCmd.Flags().StringVar(&parseDatabaseURL, "db-url", "", "Database URL (required)")
	parseJobCmd.Flags().StringVar(&parseAPIKey, "api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")

	if err := parseJobCmd.MarkFlagRequired("run-id"); err != nil {
		panic(fmt.Sprintf("failed to mark run-id flag as required: %v", err))
	}
	if err := parseJobCmd.MarkFlagRequired("db-url"); err != nil {
		panic(fmt.Sprintf("failed to mark db-url flag as required: %v", err))
	}

	rootCmd.AddCommand(parseJobCmd)
}

func runParseJob(_ *cobra.Command, _ []string) error {
	// Get API key
	apiKey := parseAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required (set GEMINI_API_KEY environment variable or use --api-key flag)")
	}

	ctx := context.Background()

	runID, err := uuid.Parse(parseRunID)
	if err != nil {
		return fmt.Errorf("invalid run-id: %w", err)
	}

	// Connect to database
	if parseDatabaseURL == "" {
		parseDatabaseURL = os.Getenv("DATABASE_URL")
	}
	if parseDatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL required (set DATABASE_URL environment variable or use --db-url flag)")
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
