package main

import (
	"context"
	"fmt"
	"os"

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
)

func init() {
	runCommand.Flags().StringVarP(&runJob, "job", "j", "", "Path to job posting text file (required)")
	runCommand.Flags().StringVarP(&runExperience, "experience", "e", "", "Path to experience bank JSON file (required)")
	runCommand.Flags().StringVarP(&runCompanySeed, "company-seed", "c", "", "Company seed URL (required)")
	runCommand.Flags().StringVarP(&runOut, "out", "o", "", "Output directory (required)")
	runCommand.Flags().StringVarP(&runName, "name", "n", "Candidate Name", "Candidate name")
	runCommand.Flags().StringVar(&runEmail, "email", "email@example.com", "Candidate email")
	runCommand.Flags().StringVar(&runPhone, "phone", "555-0100", "Candidate phone")
	runCommand.Flags().StringVarP(&runTemplate, "template", "t", "templates/one_page_resume.tex", "Path to LaTeX template")
	runCommand.Flags().IntVar(&runMaxBullets, "max-bullets", 25, "Maximum bullets allowed")
	runCommand.Flags().IntVar(&runMaxLines, "max-lines", 35, "Maximum lines allowed")

	// API key can be passed as a flag, or read from env var GEMINI_API_KEY
	runCommand.Flags().StringVar(&runAPIKey, "api-key", "", "Gemini API Key (optional, defaults to GEMINI_API_KEY env var)")

	if err := runCommand.MarkFlagRequired("job"); err != nil {
		panic(fmt.Sprintf("failed to mark job flag as required: %v", err))
	}
	if err := runCommand.MarkFlagRequired("experience"); err != nil {
		panic(fmt.Sprintf("failed to mark experience flag as required: %v", err))
	}
	if err := runCommand.MarkFlagRequired("company-seed"); err != nil {
		panic(fmt.Sprintf("failed to mark company-seed flag as required: %v", err))
	}
	if err := runCommand.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(runCommand)
}

func runPipelineCmd(_ *cobra.Command, _ []string) error {
	// API Key handling
	if runAPIKey == "" {
		runAPIKey = os.Getenv("GEMINI_API_KEY")
	}
	if runAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable or --api-key flag is required")
	}

	opts := pipeline.RunOptions{
		JobPath:        runJob,
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
