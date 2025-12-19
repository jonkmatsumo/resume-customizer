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
	textFile string
	urlStr   string
	outDir   string
)

func init() {
	ingestJobCmd.Flags().StringVarP(&textFile, "text-file", "t", "", "Path to text file containing job posting")
	ingestJobCmd.Flags().StringVarP(&urlStr, "url", "u", "", "URL to fetch job posting from")
	ingestJobCmd.Flags().StringVarP(&outDir, "out", "o", "", "Output directory (required)")

	ingestJobCmd.MarkFlagRequired("out")

	rootCmd.AddCommand(ingestJobCmd)
}

func runIngestJob(cmd *cobra.Command, args []string) error {
	// Validate mutually exclusive flags
	if textFile == "" && urlStr == "" {
		return fmt.Errorf("either --text-file or --url must be provided")
	}
	if textFile != "" && urlStr != "" {
		return fmt.Errorf("--text-file and --url are mutually exclusive; provide only one")
	}

	var cleanedText string
	var metadata *ingestion.Metadata
	var err error

	// Ingest from either text file or URL
	if textFile != "" {
		cleanedText, metadata, err = ingestion.IngestFromFile(textFile)
		if err != nil {
			return fmt.Errorf("failed to ingest from file: %w", err)
		}
	} else {
		cleanedText, metadata, err = ingestion.IngestFromURL(urlStr)
		if err != nil {
			return fmt.Errorf("failed to ingest from URL: %w", err)
		}
	}

	// Write output files
	if err := ingestion.WriteOutput(outDir, cleanedText, metadata); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Successfully ingested job posting\n")
	fmt.Fprintf(os.Stdout, "Cleaned text: %s/job_posting.cleaned.txt\n", outDir)
	fmt.Fprintf(os.Stdout, "Metadata: %s/job_posting.meta.json\n", outDir)

	return nil
}
