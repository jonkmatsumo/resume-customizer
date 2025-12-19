// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/experience"
	"github.com/spf13/cobra"
)

var loadExperienceCmd = &cobra.Command{
	Use:   "load-experience",
	Short: "Load and normalize an experience bank file",
	Long:  "Loads an experience bank JSON file, normalizes it (skills, length_chars, evidence_strength), and writes the normalized output to a file.",
	RunE:  runLoadExperience,
}

var (
	loadInputFile  string
	loadOutputFile string
)

func init() {
	loadExperienceCmd.Flags().StringVarP(&loadInputFile, "in", "i", "", "Path to input experience bank JSON file (required)")
	loadExperienceCmd.Flags().StringVarP(&loadOutputFile, "out", "o", "", "Path to output normalized experience bank JSON file (required)")

	if err := loadExperienceCmd.MarkFlagRequired("in"); err != nil {
		panic(fmt.Sprintf("failed to mark in flag as required: %v", err))
	}
	if err := loadExperienceCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(loadExperienceCmd)
}

func runLoadExperience(_ *cobra.Command, _ []string) error {
	// Load experience bank
	bank, err := experience.LoadExperienceBank(loadInputFile)
	if err != nil {
		return fmt.Errorf("failed to load experience bank: %w", err)
	}

	// Normalize
	if err := experience.NormalizeExperienceBank(bank); err != nil {
		return fmt.Errorf("failed to normalize experience bank: %w", err)
	}

	// Marshal to JSON with indentation
	jsonBytes, err := json.MarshalIndent(bank, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal normalized experience bank: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(loadOutputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to output file
	if err := os.WriteFile(loadOutputFile, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully loaded and normalized experience bank\n")
	_, _ = fmt.Fprintf(os.Stdout, "Output: %s\n", loadOutputFile)

	return nil
}
