// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/ranking"
	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/spf13/cobra"
)

var rankStoriesCmd = &cobra.Command{
	Use:   "rank-stories",
	Short: "Rank experience stories against a job profile",
	Long: `Ranks experience stories from an experience bank against a job profile, 
producing a RankedStories JSON sorted by relevance score.

When an API key is provided (via --api-key or GEMINI_API_KEY env var), 
uses hybrid scoring: 50% heuristic + 50% LLM-judged relevance.
Falls back to heuristic-only scoring if LLM is unavailable.`,
	RunE: runRankStories,
}

var (
	rankStoriesJobProfile  string
	rankStoriesUserID      string
	rankStoriesRunID       string
	rankStoriesDatabaseURL string
	rankStoriesOutput      string
	rankStoriesAPIKey      string
)

func init() {
	rankStoriesCmd.Flags().StringVarP(&rankStoriesJobProfile, "job-profile", "j", "", "Path to input JobProfile JSON file (deprecated: use --run-id)")
	rankStoriesCmd.Flags().StringVarP(&rankStoriesUserID, "user-id", "u", "", "User ID (required)")
	rankStoriesCmd.Flags().StringVar(&rankStoriesRunID, "run-id", "", "Run ID to load job profile from database (required if not using --job-profile)")
	rankStoriesCmd.Flags().StringVar(&rankStoriesDatabaseURL, "db-url", "", "Database URL (required with --run-id)")
	rankStoriesCmd.Flags().StringVarP(&rankStoriesOutput, "out", "o", "", "Path to output RankedStories JSON file (deprecated: use --run-id)")
	rankStoriesCmd.Flags().StringVar(&rankStoriesAPIKey, "api-key", "", "Optional Gemini API key for LLM-enhanced ranking (env: GEMINI_API_KEY)")

	rootCmd.AddCommand(rankStoriesCmd)
}

func runRankStories(_ *cobra.Command, _ []string) error {
	// Determine mode: database or file
	useDatabase := rankStoriesRunID != ""
	useFiles := rankStoriesJobProfile != "" || rankStoriesOutput != ""

	if useDatabase && useFiles {
		return fmt.Errorf("cannot use --run-id with --job-profile/--out flags")
	}
	if !useDatabase && !useFiles {
		return fmt.Errorf("must provide either --run-id or --job-profile/--out flags")
	}

	ctx := context.Background()

	// Connect to database (required for both modes - experience bank is always from DB)
	if rankStoriesDatabaseURL == "" {
		rankStoriesDatabaseURL = os.Getenv("DATABASE_URL")
	}
	if rankStoriesDatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL not set and --db-url not provided")
	}

	database, err := db.Connect(ctx, rankStoriesDatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	// Parse user ID
	uid, err := uuid.Parse(rankStoriesUserID)
	if err != nil {
		return fmt.Errorf("invalid user-id: %w", err)
	}

	// Load ExperienceBank from DB (always from database)
	experienceBank, err := database.GetExperienceBank(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to load experience bank from DB: %w", err)
	}

	// Load job profile
	var jobProfile *types.JobProfile
	var runID uuid.UUID

	if useFiles {
		// File mode (deprecated)
		fmt.Fprintf(os.Stderr, "Warning: File-based mode is deprecated. Use --run-id instead.\n")

		jobProfileContent, err := os.ReadFile(rankStoriesJobProfile)
		if err != nil {
			return fmt.Errorf("failed to read job profile file %s: %w", rankStoriesJobProfile, err)
		}

		var profile types.JobProfile
		if err := json.Unmarshal(jobProfileContent, &profile); err != nil {
			return fmt.Errorf("failed to unmarshal job profile JSON: %w", err)
		}
		jobProfile = &profile
	} else {
		// Database mode
		runID, err = uuid.Parse(rankStoriesRunID)
		if err != nil {
			return fmt.Errorf("invalid run-id: %w", err)
		}

		jobProfile, err = database.GetJobProfileByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load job profile from database: %w", err)
		}
		if jobProfile == nil {
			return fmt.Errorf("job profile not found for run %s", runID)
		}
	}

	// Determine API key (flag takes precedence, then env var)
	apiKey := rankStoriesAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	// Rank stories (with or without LLM)
	var rankedStories *types.RankedStories
	if apiKey != "" {
		_, _ = fmt.Fprintf(os.Stderr, "Using LLM-enhanced ranking mode\n")
		rankedStories, err = ranking.RankStoriesWithLLM(ctx, jobProfile, experienceBank, apiKey)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "Using heuristic-only ranking mode (no API key provided)\n")
		rankedStories, err = ranking.RankStories(jobProfile, experienceBank)
	}
	if err != nil {
		return fmt.Errorf("failed to rank stories: %w", err)
	}

	if useFiles {
		// File mode: write to file
		jsonOutput, err := json.MarshalIndent(rankedStories, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal ranked stories to JSON: %w", err)
		}

		outputDir := filepath.Dir(rankStoriesOutput)
		if outputDir != "" && outputDir != "." {
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
			}
		}

		if err := os.WriteFile(rankStoriesOutput, jsonOutput, 0644); err != nil {
			return fmt.Errorf("failed to write ranked stories to output file %s: %w", rankStoriesOutput, err)
		}

		// Validate output against schema (optional - non-fatal)
		schemaPath := schemas.ResolveSchemaPath("schemas/ranked_stories.schema.json")
		if schemaPath != "" {
			if err := schemas.ValidateJSON(schemaPath, rankStoriesOutput); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Output validation failed: %v\n", err)
			}
		}

		llmCount := 0
		for _, story := range rankedStories.Ranked {
			if story.LLMScore != nil {
				llmCount++
			}
		}

		if llmCount > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "Successfully ranked %d stories (%d with LLM scoring) to %s\n", len(rankedStories.Ranked), llmCount, rankStoriesOutput)
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "Successfully ranked %d stories to %s\n", len(rankedStories.Ranked), rankStoriesOutput)
		}
	} else {
		// Database mode: save to database
		// Convert to RunRankedStoryInput format
		var inputs []db.RunRankedStoryInput
		for i, story := range rankedStories.Ranked {
			input := db.RunRankedStoryInput{
				StoryIDText:      story.StoryID,
				RelevanceScore:   story.RelevanceScore,
				SkillOverlap:     story.SkillOverlap,
				KeywordOverlap:   story.KeywordOverlap,
				EvidenceStrength: story.EvidenceStrength,
				HeuristicScore:   story.HeuristicScore,
				LLMScore:         story.LLMScore,
				LLMReasoning:     story.LLMReasoning,
				MatchedSkills:    story.MatchedSkills,
				Notes:            story.Notes,
				Ordinal:          i + 1,
			}
			inputs = append(inputs, input)
		}

		_, err = database.SaveRunRankedStories(ctx, runID, inputs)
		if err != nil {
			return fmt.Errorf("failed to save ranked stories to database: %w", err)
		}

		// Also save as artifact
		if err := database.SaveArtifact(ctx, runID, db.StepRankedStories, db.CategoryExperience, rankedStories); err != nil {
			return fmt.Errorf("failed to save ranked stories artifact: %w", err)
		}

		llmCount := 0
		for _, story := range rankedStories.Ranked {
			if story.LLMScore != nil {
				llmCount++
			}
		}

		if llmCount > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "Successfully ranked %d stories (%d with LLM scoring) and saved to database (run: %s)\n", len(rankedStories.Ranked), llmCount, runID)
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "Successfully ranked %d stories and saved to database (run: %s)\n", len(rankedStories.Ranked), runID)
		}
	}

	return nil
}
