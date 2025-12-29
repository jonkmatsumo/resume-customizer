// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/ranking"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/spf13/cobra"
)

var rankStoriesCmd = &cobra.Command{
	Use:   "rank-stories",
	Short: "Rank experience stories against a job profile",
	Long: `Ranks experience stories from an experience bank against a job profile from the database,
producing ranked stories sorted by relevance score and saving to the database.

When an API key is provided (via --api-key or GEMINI_API_KEY env var), 
uses hybrid scoring: 50% heuristic + 50% LLM-judged relevance.
Falls back to heuristic-only scoring if LLM is unavailable.`,
	RunE: runRankStories,
}

var (
	rankStoriesUserID      string
	rankStoriesRunID       string
	rankStoriesDatabaseURL string
	rankStoriesAPIKey      string
)

func init() {
	rankStoriesCmd.Flags().StringVarP(&rankStoriesUserID, "user-id", "u", "", "User ID (required)")
	rankStoriesCmd.Flags().StringVar(&rankStoriesRunID, "run-id", "", "Run ID to load job profile from database (required)")
	rankStoriesCmd.Flags().StringVar(&rankStoriesDatabaseURL, "db-url", "", "Database URL (required)")
	rankStoriesCmd.Flags().StringVar(&rankStoriesAPIKey, "api-key", "", "Optional Gemini API key for LLM-enhanced ranking (env: GEMINI_API_KEY)")

	if err := rankStoriesCmd.MarkFlagRequired("user-id"); err != nil {
		panic(fmt.Sprintf("failed to mark user-id flag as required: %v", err))
	}
	if err := rankStoriesCmd.MarkFlagRequired("run-id"); err != nil {
		panic(fmt.Sprintf("failed to mark run-id flag as required: %v", err))
	}
	if err := rankStoriesCmd.MarkFlagRequired("db-url"); err != nil {
		panic(fmt.Sprintf("failed to mark db-url flag as required: %v", err))
	}

	rootCmd.AddCommand(rankStoriesCmd)
}

func runRankStories(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Connect to database
	if rankStoriesDatabaseURL == "" {
		rankStoriesDatabaseURL = os.Getenv("DATABASE_URL")
	}
	if rankStoriesDatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL not set (set DATABASE_URL environment variable or use --db-url flag)")
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

	// Load ExperienceBank from DB
	experienceBank, err := database.GetExperienceBank(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to load experience bank from DB: %w", err)
	}

	// Parse run ID
	runID, err := uuid.Parse(rankStoriesRunID)
	if err != nil {
		return fmt.Errorf("invalid run-id: %w", err)
	}

	// Load job profile from database
	jobProfile, err := database.GetJobProfileByRunID(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load job profile from database: %w", err)
	}
	if jobProfile == nil {
		return fmt.Errorf("job profile not found for run %s", runID)
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

	// Save to database
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

	return nil
}
