package main

import (
	"fmt"
	"os"

	"github.com/jonathan/resume-customizer/internal/server"
	"github.com/spf13/cobra"
)

var (
	servePort int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the REST API server",
	Long:  `Start an HTTP server that exposes REST endpoints for running the resume pipeline.`,
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}

func runServe(_ *cobra.Command, _ []string) error {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Get API key from environment
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable is required")
	}

	cfg := server.Config{
		Port:        servePort,
		DatabaseURL: databaseURL,
		APIKey:      apiKey,
	}

	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	return srv.Start()
}
