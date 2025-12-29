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
	"github.com/jonathan/resume-customizer/internal/repair"
	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/spf13/cobra"
)

var repairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair LaTeX resume violations automatically",
	Long:  "Automatically repairs violations in a LaTeX resume by proposing and applying repair actions iteratively until violations are resolved or max iterations reached.",
	RunE:  runRepair,
}

var (
	repairPlanFile           string
	repairBulletsFile        string
	repairViolationsFile     string
	repairRankedFile         string
	repairJobProfileFile     string
	repairCompanyProfileFile string
	repairRunID              string
	repairUserID             string
	repairDatabaseURL        string
	repairTemplateFile       string
	repairName               string
	repairEmail              string
	repairPhone              string
	repairMaxPages           int
	repairMaxChars           int
	repairMaxIterations      int
	repairAPIKey             string
	repairOutputDir          string
)

func init() {
	repairCmd.Flags().StringVarP(&repairPlanFile, "plan", "p", "", "Path to ResumePlan JSON file (deprecated: use --run-id)")
	repairCmd.Flags().StringVarP(&repairBulletsFile, "bullets", "b", "", "Path to RewrittenBullets JSON file (deprecated: use --run-id)")
	repairCmd.Flags().StringVarP(&repairViolationsFile, "violations", "v", "", "Path to Violations JSON file (deprecated: use --run-id)")
	repairCmd.Flags().StringVarP(&repairRankedFile, "ranked", "r", "", "Path to RankedStories JSON file (deprecated: use --run-id)")
	repairCmd.Flags().StringVarP(&repairJobProfileFile, "job-profile", "j", "", "Path to JobProfile JSON file (deprecated: use --run-id)")
	repairCmd.Flags().StringVarP(&repairCompanyProfileFile, "company-profile", "c", "", "Path to CompanyProfile JSON file (deprecated: use --run-id)")
	repairCmd.Flags().StringVar(&repairRunID, "run-id", "", "Run ID to load data from database (required if not using file flags)")
	repairCmd.Flags().StringVarP(&repairUserID, "user-id", "u", "", "User ID (required)")
	repairCmd.Flags().StringVar(&repairDatabaseURL, "db-url", "", "Database URL (required with --run-id)")
	repairCmd.Flags().StringVarP(&repairTemplateFile, "template", "t", "templates/one_page_resume.tex", "Path to LaTeX template file")
	repairCmd.Flags().StringVarP(&repairName, "name", "n", "", "Candidate name (required)")
	repairCmd.Flags().StringVar(&repairEmail, "email", "", "Candidate email (required)")
	repairCmd.Flags().StringVar(&repairPhone, "phone", "", "Candidate phone (optional)")
	repairCmd.Flags().IntVar(&repairMaxPages, "max-pages", 1, "Maximum page count")
	repairCmd.Flags().IntVar(&repairMaxChars, "max-chars", 90, "Maximum characters per line")
	repairCmd.Flags().IntVar(&repairMaxIterations, "max-iterations", 5, "Maximum repair iterations")
	repairCmd.Flags().StringVar(&repairAPIKey, "api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
	repairCmd.Flags().StringVarP(&repairOutputDir, "out", "o", "", "Output directory (deprecated: use --run-id)")

	rootCmd.AddCommand(repairCmd)
}

func runRepair(_ *cobra.Command, _ []string) error {
	// Determine mode: database or file
	useDatabase := repairRunID != ""
	useFiles := repairPlanFile != "" || repairBulletsFile != "" || repairViolationsFile != ""

	if useDatabase && useFiles {
		return fmt.Errorf("cannot use --run-id with file flags (--plan/--bullets/--violations/etc)")
	}
	if !useDatabase && !useFiles {
		return fmt.Errorf("must provide either --run-id or file flags (--plan/--bullets/--violations/etc)")
	}

	ctx := context.Background()

	// Load data
	var plan *types.ResumePlan
	var bullets *types.RewrittenBullets
	var violations *types.Violations
	var rankedStories *types.RankedStories
	var jobProfile *types.JobProfile
	var companyProfile *types.CompanyProfile
	var runID uuid.UUID
	var database *db.DB

	if useFiles {
		// File mode (deprecated)
		fmt.Fprintf(os.Stderr, "Warning: File-based mode is deprecated. Use --run-id instead.\n")

		// Load ResumePlan
		planContent, err := os.ReadFile(repairPlanFile)
		if err != nil {
			return fmt.Errorf("failed to read plan file: %w", err)
		}
		var p types.ResumePlan
		if err := json.Unmarshal(planContent, &p); err != nil {
			return fmt.Errorf("failed to unmarshal plan JSON: %w", err)
		}
		plan = &p

		// Load RewrittenBullets
		bulletsContent, err := os.ReadFile(repairBulletsFile)
		if err != nil {
			return fmt.Errorf("failed to read bullets file: %w", err)
		}
		var b types.RewrittenBullets
		if err := json.Unmarshal(bulletsContent, &b); err != nil {
			return fmt.Errorf("failed to unmarshal bullets JSON: %w", err)
		}
		bullets = &b

		// Load Violations
		violationsContent, err := os.ReadFile(repairViolationsFile)
		if err != nil {
			return fmt.Errorf("failed to read violations file: %w", err)
		}
		var v types.Violations
		if err := json.Unmarshal(violationsContent, &v); err != nil {
			return fmt.Errorf("failed to unmarshal violations JSON: %w", err)
		}
		violations = &v

		// Load RankedStories
		rankedContent, err := os.ReadFile(repairRankedFile)
		if err != nil {
			return fmt.Errorf("failed to read ranked file: %w", err)
		}
		var rs types.RankedStories
		if err := json.Unmarshal(rankedContent, &rs); err != nil {
			return fmt.Errorf("failed to unmarshal ranked stories JSON: %w", err)
		}
		rankedStories = &rs

		// Load JobProfile
		jobProfileContent, err := os.ReadFile(repairJobProfileFile)
		if err != nil {
			return fmt.Errorf("failed to read job profile file: %w", err)
		}
		var jp types.JobProfile
		if err := json.Unmarshal(jobProfileContent, &jp); err != nil {
			return fmt.Errorf("failed to unmarshal job profile JSON: %w", err)
		}
		jobProfile = &jp

		// Load CompanyProfile
		companyProfileContent, err := os.ReadFile(repairCompanyProfileFile)
		if err != nil {
			return fmt.Errorf("failed to read company profile file: %w", err)
		}
		var cp types.CompanyProfile
		if err := json.Unmarshal(companyProfileContent, &cp); err != nil {
			return fmt.Errorf("failed to unmarshal company profile JSON: %w", err)
		}
		companyProfile = &cp
	} else {
		// Database mode
		var err error
		runID, err = uuid.Parse(repairRunID)
		if err != nil {
			return fmt.Errorf("invalid run-id: %w", err)
		}

		// Connect to database
		if repairDatabaseURL == "" {
			repairDatabaseURL = os.Getenv("DATABASE_URL")
		}
		if repairDatabaseURL == "" {
			return fmt.Errorf("DATABASE_URL required when using --run-id")
		}

		database, err = db.Connect(ctx, repairDatabaseURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer database.Close()

		plan, err = database.GetResumePlanByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load resume plan from database: %w", err)
		}
		if plan == nil {
			return fmt.Errorf("resume plan not found for run %s", runID)
		}

		bullets, err = database.GetRewrittenBulletsByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load rewritten bullets from database: %w", err)
		}
		if bullets == nil {
			return fmt.Errorf("rewritten bullets not found for run %s", runID)
		}

		violations, err = database.GetViolationsByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load violations from database: %w", err)
		}
		if violations == nil {
			return fmt.Errorf("violations not found for run %s", runID)
		}

		rankedStories, err = database.GetRankedStoriesByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load ranked stories from database: %w", err)
		}
		if rankedStories == nil {
			return fmt.Errorf("ranked stories not found for run %s", runID)
		}

		jobProfile, err = database.GetJobProfileByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load job profile from database: %w", err)
		}
		if jobProfile == nil {
			return fmt.Errorf("job profile not found for run %s", runID)
		}

		companyProfile, err = database.GetCompanyProfileByRunID(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to load company profile from database: %w", err)
		}
		if companyProfile == nil {
			return fmt.Errorf("company profile not found for run %s", runID)
		}
	}

	// Load ExperienceBank from DB (always from database)
	if !useDatabase {
		// Connect to database for experience bank
		if repairDatabaseURL == "" {
			repairDatabaseURL = os.Getenv("DATABASE_URL")
		}
		if repairDatabaseURL == "" {
			return fmt.Errorf("DATABASE_URL not set and --db-url not provided (required for DB access)")
		}

		var err error
		database, err = db.Connect(ctx, repairDatabaseURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer database.Close()
	}

	uid, err := uuid.Parse(repairUserID)
	if err != nil {
		return fmt.Errorf("invalid user-id: %w", err)
	}

	experienceBank, err := database.GetExperienceBank(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to load experience bank from DB: %w", err)
	}

	// Get API key
	apiKey := repairAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required (set GEMINI_API_KEY environment variable or use --api-key flag)")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(repairOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build candidate info
	candidateInfo := repair.CandidateInfo{
		Name:  repairName,
		Email: repairEmail,
		Phone: repairPhone,
	}

	// Run repair loop
	var selectedEducation []types.Education
	if experienceBank != nil {
		selectedEducation = experienceBank.Education
	}
	finalPlan, finalBullets, finalLaTeX, finalViolations, iterations, err := repair.RunRepairLoop(
		ctx,
		plan,
		bullets,
		violations,
		rankedStories,
		jobProfile,
		companyProfile,
		experienceBank,
		repairTemplateFile,
		candidateInfo,
		selectedEducation,
		repairMaxPages,
		repairMaxChars,
		repairMaxIterations,
		apiKey,
	)
	if err != nil {
		return fmt.Errorf("repair loop failed: %w", err)
	}

	if useFiles {
		// File mode: write outputs to files
		planPath := filepath.Join(repairOutputDir, "resume_plan.json")
		planBytes, err := json.MarshalIndent(finalPlan, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal final plan: %w", err)
		}
		if err := os.WriteFile(planPath, planBytes, 0644); err != nil {
			return fmt.Errorf("failed to write final plan: %w", err)
		}

		bulletsPath := filepath.Join(repairOutputDir, "rewritten_bullets.json")
		bulletsBytes, err := json.MarshalIndent(finalBullets, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal final bullets: %w", err)
		}
		if err := os.WriteFile(bulletsPath, bulletsBytes, 0644); err != nil {
			return fmt.Errorf("failed to write final bullets: %w", err)
		}

		latexPath := filepath.Join(repairOutputDir, "resume.tex")
		if err := os.WriteFile(latexPath, []byte(finalLaTeX), 0644); err != nil {
			return fmt.Errorf("failed to write final LaTeX: %w", err)
		}

		violationsPath := filepath.Join(repairOutputDir, "violations.json")
		violationsBytes, err := json.MarshalIndent(finalViolations, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal final violations: %w", err)
		}
		if err := os.WriteFile(violationsPath, violationsBytes, 0644); err != nil {
			return fmt.Errorf("failed to write final violations: %w", err)
		}

		// Validate outputs against schemas (non-fatal)
		schemaPaths := map[string]string{
			planPath:       "schemas/resume_plan.schema.json",
			bulletsPath:    "schemas/bullets.schema.json",
			violationsPath: "schemas/violations.schema.json",
		}

		for outputPath, schemaRelPath := range schemaPaths {
			schemaPath := schemas.ResolveSchemaPath(schemaRelPath)
			if schemaPath != "" {
				if err := schemas.ValidateJSON(schemaPath, outputPath); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "Warning: Output %s does not validate against schema: %v\n", outputPath, err)
				}
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "Repair loop completed after %d iteration(s)\n", iterations)
		if len(finalViolations.Violations) == 0 {
			_, _ = fmt.Fprintf(os.Stdout, "✅ All violations resolved\n")
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "⚠️  %d violation(s) remaining\n", len(finalViolations.Violations))
		}
		_, _ = fmt.Fprintf(os.Stdout, "Output directory: %s\n", repairOutputDir)
	} else {
		// Database mode: save to database
		// Update resume plan
		resumePlan, err := database.GetRunResumePlan(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to get resume plan: %w", err)
		}
		if resumePlan != nil {
			input := &db.RunResumePlanInput{
				MaxBullets:       finalPlan.SpaceBudget.MaxBullets,
				MaxLines:         finalPlan.SpaceBudget.MaxLines,
				SkillMatchRatio:  finalPlan.SpaceBudget.SkillMatchRatio,
				SectionBudgets:   finalPlan.SpaceBudget.Sections,
				TopSkillsCovered: finalPlan.Coverage.TopSkillsCovered,
				CoverageScore:    finalPlan.Coverage.CoverageScore,
			}
			_, err = database.SaveRunResumePlan(ctx, runID, input)
			if err != nil {
				return fmt.Errorf("failed to update resume plan: %w", err)
			}
		}

		// Save artifacts
		if err := database.SaveArtifact(ctx, runID, db.StepResumePlan, db.CategoryExperience, finalPlan); err != nil {
			return fmt.Errorf("failed to save final plan artifact: %w", err)
		}

		if err := database.SaveArtifact(ctx, runID, db.StepRewrittenBullets, db.CategoryRewriting, finalBullets); err != nil {
			return fmt.Errorf("failed to save final bullets artifact: %w", err)
		}

		if err := database.SaveTextArtifact(ctx, runID, db.StepResumeTex, db.CategoryValidation, finalLaTeX); err != nil {
			return fmt.Errorf("failed to save final LaTeX artifact: %w", err)
		}

		if err := database.SaveArtifact(ctx, runID, db.StepViolations, db.CategoryValidation, finalViolations); err != nil {
			return fmt.Errorf("failed to save final violations artifact: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Repair loop completed after %d iteration(s)\n", iterations)
		if len(finalViolations.Violations) == 0 {
			_, _ = fmt.Fprintf(os.Stdout, "✅ All violations resolved\n")
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "⚠️  %d violation(s) remaining\n", len(finalViolations.Violations))
		}
		_, _ = fmt.Fprintf(os.Stdout, "Saved to database (run: %s)\n", runID)

		// Optionally write to files if --out provided (for debugging)
		if repairOutputDir != "" {
			fmt.Fprintf(os.Stderr, "Warning: Writing files for debugging (deprecated). Use database artifacts instead.\n")
			planPath := filepath.Join(repairOutputDir, "resume_plan.json")
			planBytes, _ := json.MarshalIndent(finalPlan, "", "  ")
			_ = os.WriteFile(planPath, planBytes, 0644)

			bulletsPath := filepath.Join(repairOutputDir, "rewritten_bullets.json")
			bulletsBytes, _ := json.MarshalIndent(finalBullets, "", "  ")
			_ = os.WriteFile(bulletsPath, bulletsBytes, 0644)

			latexPath := filepath.Join(repairOutputDir, "resume.tex")
			_ = os.WriteFile(latexPath, []byte(finalLaTeX), 0644)

			violationsPath := filepath.Join(repairOutputDir, "violations.json")
			violationsBytes, _ := json.MarshalIndent(finalViolations, "", "  ")
			_ = os.WriteFile(violationsPath, violationsBytes, 0644)

			_, _ = fmt.Fprintf(os.Stdout, "Debug files written to: %s\n", repairOutputDir)
		}
	}

	return nil
}
