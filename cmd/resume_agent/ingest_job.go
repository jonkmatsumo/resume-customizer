// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/ingestion"
	"github.com/spf13/cobra"
)

var ingestJobCmd = &cobra.Command{
	Use:   "ingest-job",
	Short: "Ingest a job posting from a text file or URL",
	Long:  "Ingest a job posting from either a text file or URL, clean the content, and save to the database.",
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
	ingestJobCmd.Flags().StringVar(&ingestRunID, "run-id", "", "Run ID to save job posting to database (required)")
	ingestJobCmd.Flags().StringVar(&ingestUserID, "user-id", "", "User ID (required)")
	ingestJobCmd.Flags().StringVar(&ingestDatabaseURL, "db-url", "", "Database URL (required)")
	ingestJobCmd.Flags().StringVarP(&ingestOutDir, "out", "o", "", "Output directory (optional, for debugging)")
	ingestJobCmd.Flags().BoolVarP(&ingestVerbose, "verbose", "v", false, "Print detailed debug information")
	ingestJobCmd.Flags().BoolVar(&ingestUseBrowser, "use-browser", false, "Use headless browser for SPA sites (requires Chrome)")

	ingestJobCmd.Flags().StringVar(&runAPIKey, "api-key", "", "Gemini API Key (optional, defaults to GEMINI_API_KEY env var) - required for HTML extraction")

	if err := ingestJobCmd.MarkFlagRequired("run-id"); err != nil {
		panic(fmt.Sprintf("failed to mark run-id flag as required: %v", err))
	}
	if err := ingestJobCmd.MarkFlagRequired("user-id"); err != nil {
		panic(fmt.Sprintf("failed to mark user-id flag as required: %v", err))
	}
	if err := ingestJobCmd.MarkFlagRequired("db-url"); err != nil {
		panic(fmt.Sprintf("failed to mark db-url flag as required: %v", err))
	}

	rootCmd.AddCommand(ingestJobCmd)
}

func runIngestJob(cmd *cobra.Command, _ []string) error {
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

	// Parse run ID
	runID, err := uuid.Parse(ingestRunID)
	if err != nil {
		return fmt.Errorf("invalid run-id: %w", err)
	}

	// Connect to database
	if ingestDatabaseURL == "" {
		ingestDatabaseURL = os.Getenv("DATABASE_URL")
	}
	if ingestDatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL not set (set DATABASE_URL environment variable or use --db-url flag)")
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
		// Create output directory if it doesn't exist
		if err := os.MkdirAll(ingestOutDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Write cleaned text file
		cleanedPath := filepath.Join(ingestOutDir, "job_posting.cleaned.txt")
		if err := os.WriteFile(cleanedPath, []byte(cleanedText), 0644); err != nil {
			return fmt.Errorf("failed to write cleaned text file: %w", err)
		}

		// Write metadata JSON file
		if metadata != nil {
			metaPath := filepath.Join(ingestOutDir, "job_posting.cleaned.meta.json")
			metaJSON, err := metadata.ToJSON()
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
			if err := os.WriteFile(metaPath, metaJSON, 0644); err != nil {
				return fmt.Errorf("failed to write metadata file: %w", err)
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "Debug files written to: %s\n", ingestOutDir)
	}

	return nil
}
