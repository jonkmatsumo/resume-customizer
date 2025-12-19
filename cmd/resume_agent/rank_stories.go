// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/experience"
	"github.com/jonathan/resume-customizer/internal/ranking"
	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/spf13/cobra"
)

var rankStoriesCmd = &cobra.Command{
	Use:   "rank-stories",
	Short: "Rank experience stories against a job profile",
	Long:  "Deterministically ranks experience stories from an experience bank against a job profile, producing a RankedStories JSON sorted by relevance score.",
	RunE:  runRankStories,
}

var (
	rankStoriesJobProfile string
	rankStoriesExperience string
	rankStoriesOutput     string
)

func init() {
	rankStoriesCmd.Flags().StringVarP(&rankStoriesJobProfile, "job-profile", "j", "", "Path to input JobProfile JSON file (required)")
	rankStoriesCmd.Flags().StringVarP(&rankStoriesExperience, "experience", "e", "", "Path to input ExperienceBank JSON file (required)")
	rankStoriesCmd.Flags().StringVarP(&rankStoriesOutput, "out", "o", "", "Path to output RankedStories JSON file (required)")

	if err := rankStoriesCmd.MarkFlagRequired("job-profile"); err != nil {
		panic(fmt.Sprintf("failed to mark job-profile flag as required: %v", err))
	}
	if err := rankStoriesCmd.MarkFlagRequired("experience"); err != nil {
		panic(fmt.Sprintf("failed to mark experience flag as required: %v", err))
	}
	if err := rankStoriesCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(rankStoriesCmd)
}

func runRankStories(_ *cobra.Command, _ []string) error {
	// 1. Load JobProfile
	jobProfileContent, err := os.ReadFile(rankStoriesJobProfile)
	if err != nil {
		return fmt.Errorf("failed to read job profile file %s: %w", rankStoriesJobProfile, err)
	}

	var jobProfile types.JobProfile
	if err := json.Unmarshal(jobProfileContent, &jobProfile); err != nil {
		return fmt.Errorf("failed to unmarshal job profile JSON: %w", err)
	}

	// 2. Load ExperienceBank
	experienceBank, err := experience.LoadExperienceBank(rankStoriesExperience)
	if err != nil {
		return fmt.Errorf("failed to load experience bank: %w", err)
	}

	// 3. Rank stories
	rankedStories, err := ranking.RankStories(&jobProfile, experienceBank)
	if err != nil {
		return fmt.Errorf("failed to rank stories: %w", err)
	}

	// 4. Marshal to JSON with indentation
	jsonOutput, err := json.MarshalIndent(rankedStories, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ranked stories to JSON: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(rankStoriesOutput)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
		}
	}

	// 5. Write to output file
	if err := os.WriteFile(rankStoriesOutput, jsonOutput, 0644); err != nil {
		return fmt.Errorf("failed to write ranked stories to output file %s: %w", rankStoriesOutput, err)
	}

	// 6. Validate output against schema (optional - non-fatal)
	schemaPath := schemas.ResolveSchemaPath("schemas/ranked_stories.schema.json")
	if schemaPath != "" {
		if err := schemas.ValidateJSON(schemaPath, rankStoriesOutput); err != nil {
			// Output validation is a safety check, not a requirement
			// Log warning for any validation error but don't fail the command
			_, _ = fmt.Fprintf(os.Stderr, "Warning: Output validation failed: %v\n", err)
		}
	}
	// If schema path not found or validation fails, skip validation (non-fatal)

	_, _ = fmt.Fprintf(os.Stdout, "Successfully ranked %d stories to %s\n", len(rankedStories.Ranked), rankStoriesOutput)

	return nil
}
