// Package main provides the entry point for the Resume Customizer HTTP API server.
package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "resume_agent",
	Short: "Resume Customizer HTTP API Server",
	Long:  "Resume Customizer generates strictly formatted, one-page LaTeX resumes tailored to job postings and company brand voice via REST API.",
}

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
