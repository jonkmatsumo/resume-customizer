package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/ingestion"
	"github.com/jonathan/resume-customizer/internal/pipeline"
	"github.com/spf13/cobra"
)

var runCommand = &cobra.Command{
	Use:   "run",
	Short: "Run the full resume generation pipeline end-to-end",
	Long:  "Orchestrates the entire resume generation process: ingestion -> parsing -> planning -> selection -> crawling -> voice -> rewriting -> rendering -> validation -> repair.",
	RunE:  runPipelineCmd,
}

var (
	runJob         string
	runJobURL      string
	runExperience  string
	runCompanySeed string
	runOut         string
	runName        string
	runEmail       string
	runPhone       string
	runTemplate    string
	runMaxBullets  int
	runMaxLines    int
	runAPIKey      string
	runUseBrowser  bool
	runVerbose     bool
)

func init() {
	runCommand.Flags().StringVarP(&runJob, "job", "j", "", "Path to job posting text file (mutually exclusive with --job-url)")
	runCommand.Flags().StringVar(&runJobURL, "job-url", "", "URL to fetch job posting from (mutually exclusive with --job)")
	runCommand.Flags().StringVarP(&runExperience, "experience", "e", "", "Path to experience bank JSON file (required)")
	runCommand.Flags().StringVarP(&runCompanySeed, "company-seed", "c", "", "Company seed URL (optional, auto-discovered if not provided)")
	runCommand.Flags().StringVarP(&runOut, "out", "o", "", "Output directory (required)")
	runCommand.Flags().StringVarP(&runName, "name", "n", "Candidate Name", "Candidate name")
	runCommand.Flags().StringVar(&runEmail, "email", "email@example.com", "Candidate email")
	runCommand.Flags().StringVar(&runPhone, "phone", "555-0100", "Candidate phone")
	runCommand.Flags().StringVarP(&runTemplate, "template", "t", "templates/one_page_resume.tex", "Path to LaTeX template")
	runCommand.Flags().IntVar(&runMaxBullets, "max-bullets", 25, "Maximum bullets allowed")
	runCommand.Flags().IntVar(&runMaxLines, "max-lines", 35, "Maximum lines allowed")
	runCommand.Flags().BoolVar(&runUseBrowser, "use-browser", false, "Use headless browser for SPA sites (requires Chrome)")
	runCommand.Flags().BoolVarP(&runVerbose, "verbose", "v", false, "Print detailed debug information")

	// API key can be passed as a flag, or read from env var GEMINI_API_KEY
	runCommand.Flags().StringVar(&runAPIKey, "api-key", "", "Gemini API Key (optional, defaults to GEMINI_API_KEY env var)")

	// Note: --job is no longer required; we validate mutual exclusivity in runPipelineCmd
	// Note: --company-seed is now optional; auto-discovery via Google Search if not provided
	if err := runCommand.MarkFlagRequired("experience"); err != nil {
		panic(fmt.Sprintf("failed to mark experience flag as required: %v", err))
	}
	if err := runCommand.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(runCommand)
}

func runPipelineCmd(_ *cobra.Command, _ []string) error {
	// Validate mutual exclusivity of --job and --job-url
	if runJob == "" && runJobURL == "" {
		return fmt.Errorf("either --job or --job-url must be provided")
	}
	if runJob != "" && runJobURL != "" {
		return fmt.Errorf("--job and --job-url are mutually exclusive; provide only one")
	}

	// API Key handling
	if runAPIKey == "" {
		runAPIKey = os.Getenv("GEMINI_API_KEY")
	}
	if runAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable or --api-key flag is required")
	}

	// If --job-url is provided, ingest the job posting first
	jobPath := runJob
	if runJobURL != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Ingesting job posting from URL: %s\n", runJobURL)

		ctx := context.Background()
		cleanedText, metadata, err := ingestion.IngestFromURL(ctx, runJobURL, runAPIKey, runUseBrowser, runVerbose)
		if err != nil {
			return fmt.Errorf("failed to ingest job from URL: %w", err)
		}

		// Write the ingested job to the output directory
		if err := ingestion.WriteOutput(runOut, cleanedText, metadata); err != nil {
			return fmt.Errorf("failed to write ingested job: %w", err)
		}

		// Update jobPath to point to the ingested file
		jobPath = filepath.Join(runOut, "job_posting.cleaned.txt")
		_, _ = fmt.Fprintf(os.Stdout, "Job posting ingested to: %s\n", jobPath)
	}

	opts := pipeline.RunOptions{
		JobPath:        jobPath,
		ExperiencePath: runExperience,
		CompanySeedURL: runCompanySeed,
		OutputDir:      runOut,
		CandidateName:  runName,
		CandidateEmail: runEmail,
		CandidatePhone: runPhone,
		TemplatePath:   runTemplate,
		MaxBullets:     runMaxBullets,
		MaxLines:       runMaxLines,
		APIKey:         runAPIKey,
	}

	// Create a context (could be cancellable if we wanted to add signal handling)
	ctx := context.Background()

	return pipeline.RunPipeline(ctx, opts)
}
