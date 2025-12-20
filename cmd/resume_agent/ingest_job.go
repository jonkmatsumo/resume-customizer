// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"fmt"
	"os"

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
	textFile   string
	urlStr     string
	outDir     string
	verbose    bool
	useBrowser bool
)

func init() {
	ingestJobCmd.Flags().StringVarP(&textFile, "text-file", "t", "", "Path to text file containing job posting")
	ingestJobCmd.Flags().StringVarP(&urlStr, "url", "u", "", "URL to fetch job posting from")
	ingestJobCmd.Flags().StringVarP(&outDir, "out", "o", "", "Output directory (required)")
	ingestJobCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Print detailed debug information")
	ingestJobCmd.Flags().BoolVar(&useBrowser, "use-browser", false, "Use headless browser for SPA sites (requires Chrome)")

	ingestJobCmd.Flags().StringVar(&runAPIKey, "api-key", "", "Gemini API Key (optional, defaults to GEMINI_API_KEY env var) - required for HTML extraction")

	if err := ingestJobCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(ingestJobCmd)
}

func runIngestJob(cmd *cobra.Command, _ []string) error {
	// Validate mutually exclusive flags
	if textFile == "" && urlStr == "" {
		return fmt.Errorf("either --text-file or --url must be provided")
	}
	if textFile != "" && urlStr != "" {
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
	if textFile != "" {
		cleanedText, metadata, err = ingestion.IngestFromFile(ctx, textFile, apiKey)
		if err != nil {
			return fmt.Errorf("failed to ingest from file: %w", err)
		}
	} else {
		// URL ingestion with platform detection, LLM extraction, and optional browser fallback
		cleanedText, metadata, err = ingestion.IngestFromURL(ctx, urlStr, apiKey, useBrowser, verbose)
		if err != nil {
			return fmt.Errorf("failed to ingest from URL: %w", err)
		}
	}

	// Write output files
	if err := ingestion.WriteOutput(outDir, cleanedText, metadata); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully ingested job posting\n")
	_, _ = fmt.Fprintf(os.Stdout, "Cleaned text: %s/job_posting.cleaned.txt\n", outDir)
	_, _ = fmt.Fprintf(os.Stdout, "Metadata: %s/job_posting.meta.json\n", outDir)

	return nil
}
