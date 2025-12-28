package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/pipeline"
	"github.com/spf13/cobra"
)

var runCommand = &cobra.Command{
	Use:   "run",
	Short: "Run the full resume generation pipeline end-to-end",
	Long: `Orchestrates the entire resume generation process: ingestion -> parsing -> planning -> selection -> crawling -> voice -> rewriting -> rendering -> validation -> repair.

Configuration can be loaded from a JSON file using --config. Command-line arguments override config file values.`,
	RunE: runPipelineCmd,
}

var (
	runConfigPath  string
	runJob         string
	runJobURL      string
	runCompanySeed string
	runName        string
	runEmail       string
	runPhone       string
	runTemplate    string
	runMaxBullets  int
	runMaxLines    int
	runAPIKey      string
	runUseBrowser  bool
	runVerbose     bool
	runDatabaseURL string
)

func init() {
	// Config file flag (processed first)
	runCommand.Flags().StringVar(&runConfigPath, "config", "", "Path to config.json file (values can be overridden by other flags)")

	runCommand.Flags().StringVarP(&runJob, "job", "j", "", "Path to job posting text file (mutually exclusive with --job-url)")
	runCommand.Flags().StringVar(&runJobURL, "job-url", "", "URL to fetch job posting from (mutually exclusive with --job)")
	runCommand.Flags().StringVarP(&runCompanySeed, "company-seed", "c", "", "Company seed URL (optional, auto-discovered if not provided)")
	runCommand.Flags().StringVarP(&runName, "name", "n", "", "Candidate name")
	runCommand.Flags().StringVar(&runEmail, "email", "", "Candidate email")
	runCommand.Flags().StringVar(&runPhone, "phone", "", "Candidate phone")
	runCommand.Flags().StringVarP(&runTemplate, "template", "t", "", "Path to LaTeX template")
	runCommand.Flags().IntVar(&runMaxBullets, "max-bullets", 0, "Maximum bullets allowed")
	runCommand.Flags().IntVar(&runMaxLines, "max-lines", 0, "Maximum lines allowed")
	runCommand.Flags().BoolVar(&runUseBrowser, "use-browser", false, "Use headless browser for SPA sites (requires Chrome)")
	runCommand.Flags().BoolVarP(&runVerbose, "verbose", "v", false, "Print detailed debug information")

	// API key can be passed as a flag, or read from env var GEMINI_API_KEY
	runCommand.Flags().StringVar(&runAPIKey, "api-key", "", "Gemini API Key (optional, defaults to GEMINI_API_KEY env var)")

	// Database URL for artifact persistence
	runCommand.Flags().StringVar(&runDatabaseURL, "db-url", "", "PostgreSQL connection URL (optional, defaults to DATABASE_URL env var)")

	// Note: --job is no longer required; we validate after merging config

	rootCmd.AddCommand(runCommand)
}

func runPipelineCmd(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Step 1: Load config file if provided
	var cfg config.Config
	if runConfigPath != "" {
		loadedCfg, err := config.LoadConfig(runConfigPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Validate loaded config
		if err := loadedCfg.Validate(); err != nil {
			return err
		}

		cfg = *loadedCfg
		if runVerbose {
			_, _ = fmt.Fprintf(os.Stdout, "Loaded config from: %s\n", runConfigPath)
		}
	}

	// Step 2: Apply CLI overrides (command-line args take priority)
	// Only override if the flag was explicitly set
	if cmd.Flags().Changed("job") {
		cfg.Job = runJob
	}
	if cmd.Flags().Changed("job-url") {
		cfg.JobURL = runJobURL
	}
	if cmd.Flags().Changed("company-seed") {
		cfg.CompanySeed = runCompanySeed
	}
	if cmd.Flags().Changed("name") {
		cfg.Name = runName
	}
	if cmd.Flags().Changed("email") {
		cfg.Email = runEmail
	}
	if cmd.Flags().Changed("phone") {
		cfg.Phone = runPhone
	}
	if cmd.Flags().Changed("template") {
		cfg.Template = runTemplate
	}
	if cmd.Flags().Changed("max-bullets") {
		cfg.MaxBullets = runMaxBullets
	}
	if cmd.Flags().Changed("max-lines") {
		cfg.MaxLines = runMaxLines
	}
	if cmd.Flags().Changed("api-key") {
		cfg.APIKey = runAPIKey
	}
	if cmd.Flags().Changed("use-browser") {
		cfg.UseBrowser = runUseBrowser
	}
	if cmd.Flags().Changed("verbose") {
		cfg.Verbose = runVerbose
	}
	if cmd.Flags().Changed("db-url") {
		cfg.DatabaseURL = runDatabaseURL
	}

	// Step 3: Apply defaults for unset values
	defaults := config.Config{
		Name:       "Candidate Name",
		Email:      "email@example.com",
		Phone:      "555-0100",
		Template:   "templates/one_page_resume.tex",
		MaxBullets: 25,
		MaxLines:   35,
	}
	cfg = cfg.MergeWithDefaults(defaults)

	// Step 4: Validate required fields
	if cfg.Job == "" && cfg.JobURL == "" {
		return fmt.Errorf("either --job or --job-url must be provided (via flag or config)")
	}
	if cfg.Job != "" && cfg.JobURL != "" {
		return fmt.Errorf("--job and --job-url are mutually exclusive; provide only one")
	}
	if cfg.UserID == "" {
		return fmt.Errorf("--user-id is required (via config file)")
	}

	// Step 5: API Key handling
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("GEMINI_API_KEY")
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable or --api-key flag is required")
	}

	// Step 6: Database URL handling (required for fetching user data)
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	}
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable or --db-url flag is required")
	}

	// Connect to DB to fetch experience data
	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	uid, err := uuid.Parse(cfg.UserID)
	if err != nil {
		return fmt.Errorf("invalid user_id format: %w", err)
	}

	expBank, err := database.GetExperienceBank(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to fetch experience bank for user %s: %w", uid, err)
	}

	opts := pipeline.RunOptions{
		JobPath:        cfg.Job,
		JobURL:         cfg.JobURL,
		ExperienceData: expBank,
		CompanySeedURL: cfg.CompanySeed,
		CandidateName:  cfg.Name,
		CandidateEmail: cfg.Email,
		CandidatePhone: cfg.Phone,
		TemplatePath:   cfg.Template,
		MaxBullets:     cfg.MaxBullets,
		MaxLines:       cfg.MaxLines,
		APIKey:         cfg.APIKey,
		UseBrowser:     cfg.UseBrowser,
		Verbose:        cfg.Verbose,
		DatabaseURL:    cfg.DatabaseURL,
	}

	return pipeline.RunPipeline(ctx, opts)
}
