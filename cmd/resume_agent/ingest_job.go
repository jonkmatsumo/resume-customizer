// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/ingestion"
	"github.com/spf13/cobra"
)

var ingestJobCmd = &cobra.Command{
	Use:   "ingest-job",
	Short: "Ingest a job posting from a text file or URL",
	Long:  "Ingest a job posting from either a text file or URL, clean the content, and output cleaned text with metadata.",
	RunE:  runIngestJob,
}

var (
	ingestTextFile    string
	ingestURL         string
	ingestRunID       string
	ingestUserID      string
	ingestDatabaseURL string
	ingestOutDir      string
	ingestVerbose     bool
	ingestUseBrowser  bool
)

func init() {
	ingestJobCmd.Flags().StringVarP(&ingestTextFile, "text-file", "t", "", "Path to text file containing job posting")
	ingestJobCmd.Flags().StringVarP(&ingestURL, "url", "u", "", "URL to fetch job posting from")
	ingestJobCmd.Flags().StringVar(&ingestRunID, "run-id", "", "Run ID to save job posting to database (required if not using --out)")
	ingestJobCmd.Flags().StringVar(&ingestUserID, "user-id", "", "User ID (required with --run-id)")
	ingestJobCmd.Flags().StringVar(&ingestDatabaseURL, "db-url", "", "Database URL (required with --run-id)")
	ingestJobCmd.Flags().StringVarP(&ingestOutDir, "out", "o", "", "Output directory (deprecated: use --run-id)")
	ingestJobCmd.Flags().BoolVarP(&ingestVerbose, "verbose", "v", false, "Print detailed debug information")
	ingestJobCmd.Flags().BoolVar(&ingestUseBrowser, "use-browser", false, "Use headless browser for SPA sites (requires Chrome)")

	ingestJobCmd.Flags().StringVar(&runAPIKey, "api-key", "", "Gemini API Key (optional, defaults to GEMINI_API_KEY env var) - required for HTML extraction")

	rootCmd.AddCommand(ingestJobCmd)
}

func runIngestJob(cmd *cobra.Command, _ []string) error {
	// Determine mode: database or file
	useDatabase := ingestRunID != ""
	useFiles := ingestOutDir != ""

	if useDatabase && useFiles {
		return fmt.Errorf("cannot use --run-id with --out flag (use --out only for debugging)")
	}
	if !useDatabase && !useFiles {
		return fmt.Errorf("must provide either --run-id or --out flag")
	}

	// Validate mutually exclusive flags
	if ingestTextFile == "" && ingestURL == "" {
		return fmt.Errorf("either --text-file or --url must be provided")
	}
	if ingestTextFile != "" && ingestURL != "" {
		return fmt.Errorf("--text-file and --url are mutually exclusive; provide only one")
	}

	// Get API key
	apiKey := runAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	ctx := cmd.Context()
	var cleanedText string
	var metadata *ingestion.Metadata
	var err error

	// Ingest from either text file or URL
	if ingestTextFile != "" {
		cleanedText, metadata, err = ingestion.IngestFromFile(ctx, ingestTextFile, apiKey)
		if err != nil {
			return fmt.Errorf("failed to ingest from file: %w", err)
		}
	} else {
		// URL ingestion with platform detection, LLM extraction, and optional browser fallback
		cleanedText, metadata, err = ingestion.IngestFromURL(ctx, ingestURL, apiKey, ingestUseBrowser, ingestVerbose)
		if err != nil {
			return fmt.Errorf("failed to ingest from URL: %w", err)
		}
	}

	if useFiles {
		// File mode (deprecated)
		fmt.Fprintf(os.Stderr, "Warning: File-based mode is deprecated. Use --run-id instead.\n")

		// Write output files
		if err := ingestion.WriteOutput(ingestOutDir, cleanedText, metadata); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully ingested job posting\n")
		_, _ = fmt.Fprintf(os.Stdout, "Cleaned text: %s/job_posting.cleaned.txt\n", ingestOutDir)
		_, _ = fmt.Fprintf(os.Stdout, "Metadata: %s/job_posting.cleaned.meta.json\n", ingestOutDir)
	} else {
		// Database mode
		runID, err := uuid.Parse(ingestRunID)
		if err != nil {
			return fmt.Errorf("invalid run-id: %w", err)
		}

		if ingestUserID == "" {
			return fmt.Errorf("--user-id is required with --run-id")
		}

		// Connect to database
		if ingestDatabaseURL == "" {
			ingestDatabaseURL = os.Getenv("DATABASE_URL")
		}
		if ingestDatabaseURL == "" {
			return fmt.Errorf("DATABASE_URL required when using --run-id")
		}

		database, err := db.Connect(ctx, ingestDatabaseURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer database.Close()

		// Get or create run
		run, err := database.GetRun(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to get run: %w", err)
		}
		if run == nil {
			// Create new run if it doesn't exist
			// We need company and role title - try to extract from metadata or use defaults
			company := "Unknown"
			roleTitle := "Unknown"
			if metadata != nil && metadata.Company != "" {
				company = metadata.Company
			}
			// Role title might not be available yet, will be updated after parsing
			runID, err = database.CreateRun(ctx, company, roleTitle, ingestURL)
			if err != nil {
				return fmt.Errorf("failed to create run: %w", err)
			}
		}

		// Prepare job posting input
		jobURL := ingestURL
		if jobURL == "" && metadata != nil {
			jobURL = metadata.URL
		}
		if jobURL == "" {
			jobURL = "file://" + ingestTextFile // Use file path as URL for file-based ingestion
		}

		platform := db.PlatformUnknown
		if metadata != nil && metadata.Platform != "" {
			platform = metadata.Platform
		}

		roleTitle := ""
		// Role title will be extracted during parsing, not from ingestion

		adminInfo := &db.AdminInfo{}
		if metadata != nil && metadata.AdminInfo != nil {
			salary := metadata.AdminInfo["salary"]
			location := metadata.AdminInfo["location"]
			remotePolicy := metadata.AdminInfo["remote_policy"]
			employmentType := metadata.AdminInfo["employment_type"]

			adminInfo = &db.AdminInfo{
				Salary:         &salary,
				Location:       &location,
				RemotePolicy:   &remotePolicy,
				EmploymentType: &employmentType,
			}
			// Only set non-empty values
			if salary == "" {
				adminInfo.Salary = nil
			}
			if location == "" {
				adminInfo.Location = nil
			}
			if remotePolicy == "" {
				adminInfo.RemotePolicy = nil
			}
			if employmentType == "" {
				adminInfo.EmploymentType = nil
			}
		}

		links := []string{}
		if metadata != nil {
			links = metadata.ExtractedLinks
		}

		// Create/upsert job posting
		jobPostingInput := &db.JobPostingCreateInput{
			URL:          jobURL,
			CompanyID:    nil, // Will be set after company is identified
			RoleTitle:    roleTitle,
			Platform:     platform,
			RawHTML:      "", // Not available from ingestion
			CleanedText:  cleanedText,
			AboutCompany: "",
			AdminInfo:    adminInfo,
			Links:        links,
			HTTPStatus:   200, // Assume success for now
		}

		jobPosting, err := database.UpsertJobPosting(ctx, jobPostingInput)
		if err != nil {
			return fmt.Errorf("failed to save job posting to database: %w", err)
		}

		// Save artifacts to database
		if err := database.SaveTextArtifact(ctx, runID, db.StepJobPosting, db.CategoryIngestion, cleanedText); err != nil {
			return fmt.Errorf("failed to save job posting text artifact: %w", err)
		}

		if err := database.SaveArtifact(ctx, runID, db.StepJobMetadata, db.CategoryIngestion, metadata); err != nil {
			return fmt.Errorf("failed to save job metadata artifact: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully ingested job posting and saved to database\n")
		_, _ = fmt.Fprintf(os.Stdout, "Job posting ID: %s\n", jobPosting.ID)
		_, _ = fmt.Fprintf(os.Stdout, "Run ID: %s\n", runID)

		// Optionally write to file if --out provided (for debugging)
		if ingestOutDir != "" {
			fmt.Fprintf(os.Stderr, "Warning: Writing files for debugging (deprecated). Use database artifacts instead.\n")
			if err := ingestion.WriteOutput(ingestOutDir, cleanedText, metadata); err != nil {
				return fmt.Errorf("failed to write debug output: %w", err)
			}
			_, _ = fmt.Fprintf(os.Stdout, "Debug files written to: %s\n", ingestOutDir)
		}
	}

	return nil
}
