// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/jonathan/resume-customizer/internal/skills"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/spf13/cobra"
)

var buildSkillTargetsCmd = &cobra.Command{
	Use:   "build-skill-targets",
	Short: "Build weighted skill targets from a JobProfile",
	Long:  "Builds a weighted list of target skills from a JobProfile JSON file, applying deterministic weighting rules and producing a SkillTargets JSON that validates against the schema.",
	RunE:  runBuildSkillTargets,
}

var (
	buildSkillTargetsJobProfile string
	buildSkillTargetsOutput     string
)

func init() {
	buildSkillTargetsCmd.Flags().StringVarP(&buildSkillTargetsJobProfile, "job-profile", "j", "", "Path to input JobProfile JSON file (required)")
	buildSkillTargetsCmd.Flags().StringVarP(&buildSkillTargetsOutput, "out", "o", "", "Path to output SkillTargets JSON file (required)")

	if err := buildSkillTargetsCmd.MarkFlagRequired("job-profile"); err != nil {
		panic(fmt.Sprintf("failed to mark job-profile flag as required: %v", err))
	}
	if err := buildSkillTargetsCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(buildSkillTargetsCmd)
}

func runBuildSkillTargets(_ *cobra.Command, _ []string) error {
	// 1. Load JobProfile from JSON file
	jobProfileContent, err := os.ReadFile(buildSkillTargetsJobProfile)
	if err != nil {
		return fmt.Errorf("failed to read job profile file %s: %w", buildSkillTargetsJobProfile, err)
	}

	var jobProfile types.JobProfile
	if err := json.Unmarshal(jobProfileContent, &jobProfile); err != nil {
		return fmt.Errorf("failed to unmarshal job profile JSON: %w", err)
	}

	// 2. Validate input JobProfile against schema (optional but recommended)
	schemaPath := schemas.ResolveSchemaPath("schemas/job_profile.schema.json")
	if schemaPath != "" {
		if err := schemas.ValidateJSON(schemaPath, buildSkillTargetsJobProfile); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: Input job profile failed schema validation: %v\n", err)
		}
	}

	// 3. Build skill targets
	targets, err := skills.BuildSkillTargets(&jobProfile)
	if err != nil {
		return fmt.Errorf("failed to build skill targets: %w", err)
	}

	// 4. Marshal to JSON with indentation
	jsonOutput, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skill targets to JSON: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(buildSkillTargetsOutput)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
		}
	}

	// 5. Write to output file
	if err := os.WriteFile(buildSkillTargetsOutput, jsonOutput, 0644); err != nil {
		return fmt.Errorf("failed to write skill targets to output file %s: %w", buildSkillTargetsOutput, err)
	}

	// 6. Validate output against schema (if schema file exists)
	outputSchemaPath := schemas.ResolveSchemaPath("schemas/skill_targets.schema.json")
	if outputSchemaPath != "" {
		if err := schemas.ValidateJSON(outputSchemaPath, buildSkillTargetsOutput); err != nil {
			// Distinguish between validation errors (data doesn't match schema) and schema load errors
			var validationErr *schemas.ValidationError
			var schemaLoadErr *schemas.SchemaLoadError
			if errors.As(err, &validationErr) {
				// Actual validation failure - return error
				return fmt.Errorf("generated skill targets are invalid: %w", err)
			} else if errors.As(err, &schemaLoadErr) {
				// Schema loading issue - log warning and continue
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema (schema loading failed): %v\n", err)
			} else {
				// Other errors - log warning and continue
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not validate output against schema: %v\n", err)
			}
		}
	}
	// If schema path not found, skip validation (non-fatal)

	_, _ = fmt.Fprintf(os.Stdout, "Successfully built skill targets to %s\n", buildSkillTargetsOutput)

	return nil
}
