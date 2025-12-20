// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/crawling"
	"github.com/spf13/cobra"
)

var crawlBrandCmd = &cobra.Command{
	Use:   "crawl-brand",
	Short: "Crawl a company website and build a text corpus",
	Long:  "Crawls a company website starting from a seed URL, classifies links using LLM, and builds a text corpus for brand voice analysis.",
	RunE:  runCrawlBrand,
}

var (
	crawlBrandSeedURL string
	crawlBrandMaxPages int
	crawlBrandOutputDir string
	crawlBrandAPIKey   string
)

func init() {
	crawlBrandCmd.Flags().StringVarP(&crawlBrandSeedURL, "seed-url", "u", "", "Company homepage URL (required)")
	crawlBrandCmd.Flags().IntVar(&crawlBrandMaxPages, "max-pages", 10, "Maximum pages to crawl (default: 10, max: 15)")
	crawlBrandCmd.Flags().StringVarP(&crawlBrandOutputDir, "out", "o", "", "Output directory (required)")
	crawlBrandCmd.Flags().StringVar(&crawlBrandAPIKey, "api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")

	if err := crawlBrandCmd.MarkFlagRequired("seed-url"); err != nil {
		panic(fmt.Sprintf("failed to mark seed-url flag as required: %v", err))
	}
	if err := crawlBrandCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(crawlBrandCmd)
}

func runCrawlBrand(_ *cobra.Command, _ []string) error {
	// Get API key from flag or environment
	apiKey := crawlBrandAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("API key required: set --api-key flag or GEMINI_API_KEY environment variable")
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(crawlBrandOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", crawlBrandOutputDir, err)
	}

	// Crawl corpus
	ctx := context.Background()
	corpus, err := crawling.CrawlBrandCorpus(ctx, crawlBrandSeedURL, crawlBrandMaxPages, apiKey)
	if err != nil {
		return fmt.Errorf("failed to crawl brand corpus: %w", err)
	}

	// Write corpus text file
	corpusPath := filepath.Join(crawlBrandOutputDir, "company_corpus.txt")
	if err := os.WriteFile(corpusPath, []byte(corpus.Corpus), 0644); err != nil {
		return fmt.Errorf("failed to write corpus file %s: %w", corpusPath, err)
	}

	// Write sources JSON file
	sourcesPath := filepath.Join(crawlBrandOutputDir, "company_corpus.sources.json")
	sourcesJSON, err := json.MarshalIndent(corpus.Sources, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sources to JSON: %w", err)
	}
	if err := os.WriteFile(sourcesPath, sourcesJSON, 0644); err != nil {
		return fmt.Errorf("failed to write sources file %s: %w", sourcesPath, err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully crawled %d pages\n", len(corpus.Sources))
	_, _ = fmt.Fprintf(os.Stdout, "Corpus: %s\n", corpusPath)
	_, _ = fmt.Fprintf(os.Stdout, "Sources: %s\n", sourcesPath)

	return nil
}

