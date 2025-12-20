// Package main implements the resume_agent CLI tool for schema-first resume generation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/experience"
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
	repairExperienceFile     string
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
	repairCmd.Flags().StringVarP(&repairPlanFile, "plan", "p", "", "Path to ResumePlan JSON file (required)")
	repairCmd.Flags().StringVarP(&repairBulletsFile, "bullets", "b", "", "Path to RewrittenBullets JSON file (required)")
	repairCmd.Flags().StringVarP(&repairViolationsFile, "violations", "v", "", "Path to Violations JSON file (required)")
	repairCmd.Flags().StringVarP(&repairRankedFile, "ranked", "r", "", "Path to RankedStories JSON file (required)")
	repairCmd.Flags().StringVarP(&repairJobProfileFile, "job-profile", "j", "", "Path to JobProfile JSON file (required)")
	repairCmd.Flags().StringVarP(&repairCompanyProfileFile, "company-profile", "c", "", "Path to CompanyProfile JSON file (required)")
	repairCmd.Flags().StringVarP(&repairExperienceFile, "experience", "e", "", "Path to ExperienceBank JSON file (required)")
	repairCmd.Flags().StringVarP(&repairTemplateFile, "template", "t", "templates/one_page_resume.tex", "Path to LaTeX template file")
	repairCmd.Flags().StringVarP(&repairName, "name", "n", "", "Candidate name (required)")
	repairCmd.Flags().StringVar(&repairEmail, "email", "", "Candidate email (required)")
	repairCmd.Flags().StringVar(&repairPhone, "phone", "", "Candidate phone (optional)")
	repairCmd.Flags().IntVar(&repairMaxPages, "max-pages", 1, "Maximum page count")
	repairCmd.Flags().IntVar(&repairMaxChars, "max-chars", 90, "Maximum characters per line")
	repairCmd.Flags().IntVar(&repairMaxIterations, "max-iterations", 5, "Maximum repair iterations")
	repairCmd.Flags().StringVar(&repairAPIKey, "api-key", "", "Gemini API key (overrides GEMINI_API_KEY env var)")
	repairCmd.Flags().StringVarP(&repairOutputDir, "out", "o", "", "Output directory (required)")

	if err := repairCmd.MarkFlagRequired("plan"); err != nil {
		panic(fmt.Sprintf("failed to mark plan flag as required: %v", err))
	}
	if err := repairCmd.MarkFlagRequired("bullets"); err != nil {
		panic(fmt.Sprintf("failed to mark bullets flag as required: %v", err))
	}
	if err := repairCmd.MarkFlagRequired("violations"); err != nil {
		panic(fmt.Sprintf("failed to mark violations flag as required: %v", err))
	}
	if err := repairCmd.MarkFlagRequired("ranked"); err != nil {
		panic(fmt.Sprintf("failed to mark ranked flag as required: %v", err))
	}
	if err := repairCmd.MarkFlagRequired("job-profile"); err != nil {
		panic(fmt.Sprintf("failed to mark job-profile flag as required: %v", err))
	}
	if err := repairCmd.MarkFlagRequired("company-profile"); err != nil {
		panic(fmt.Sprintf("failed to mark company-profile flag as required: %v", err))
	}
	if err := repairCmd.MarkFlagRequired("experience"); err != nil {
		panic(fmt.Sprintf("failed to mark experience flag as required: %v", err))
	}
	if err := repairCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Sprintf("failed to mark name flag as required: %v", err))
	}
	if err := repairCmd.MarkFlagRequired("email"); err != nil {
		panic(fmt.Sprintf("failed to mark email flag as required: %v", err))
	}
	if err := repairCmd.MarkFlagRequired("out"); err != nil {
		panic(fmt.Sprintf("failed to mark out flag as required: %v", err))
	}

	rootCmd.AddCommand(repairCmd)
}

func runRepair(_ *cobra.Command, _ []string) error {
	// Load ResumePlan
	planContent, err := os.ReadFile(repairPlanFile)
	if err != nil {
		return fmt.Errorf("failed to read plan file: %w", err)
	}
	var plan types.ResumePlan
	if err := json.Unmarshal(planContent, &plan); err != nil {
		return fmt.Errorf("failed to unmarshal plan JSON: %w", err)
	}

	// Load RewrittenBullets
	bulletsContent, err := os.ReadFile(repairBulletsFile)
	if err != nil {
		return fmt.Errorf("failed to read bullets file: %w", err)
	}
	var bullets types.RewrittenBullets
	if err := json.Unmarshal(bulletsContent, &bullets); err != nil {
		return fmt.Errorf("failed to unmarshal bullets JSON: %w", err)
	}

	// Load Violations
	violationsContent, err := os.ReadFile(repairViolationsFile)
	if err != nil {
		return fmt.Errorf("failed to read violations file: %w", err)
	}
	var violations types.Violations
	if err := json.Unmarshal(violationsContent, &violations); err != nil {
		return fmt.Errorf("failed to unmarshal violations JSON: %w", err)
	}

	// Load RankedStories
	rankedContent, err := os.ReadFile(repairRankedFile)
	if err != nil {
		return fmt.Errorf("failed to read ranked file: %w", err)
	}
	var rankedStories types.RankedStories
	if err := json.Unmarshal(rankedContent, &rankedStories); err != nil {
		return fmt.Errorf("failed to unmarshal ranked stories JSON: %w", err)
	}

	// Load JobProfile
	jobProfileContent, err := os.ReadFile(repairJobProfileFile)
	if err != nil {
		return fmt.Errorf("failed to read job profile file: %w", err)
	}
	var jobProfile types.JobProfile
	if err := json.Unmarshal(jobProfileContent, &jobProfile); err != nil {
		return fmt.Errorf("failed to unmarshal job profile JSON: %w", err)
	}

	// Load CompanyProfile
	companyProfileContent, err := os.ReadFile(repairCompanyProfileFile)
	if err != nil {
		return fmt.Errorf("failed to read company profile file: %w", err)
	}
	var companyProfile types.CompanyProfile
	if err := json.Unmarshal(companyProfileContent, &companyProfile); err != nil {
		return fmt.Errorf("failed to unmarshal company profile JSON: %w", err)
	}

	// Load ExperienceBank
	experienceBank, err := experience.LoadExperienceBank(repairExperienceFile)
	if err != nil {
		return fmt.Errorf("failed to load experience bank: %w", err)
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
	ctx := context.Background()
	finalPlan, finalBullets, finalLaTeX, finalViolations, iterations, err := repair.RunRepairLoop(
		ctx,
		&plan,
		&bullets,
		&violations,
		&rankedStories,
		&jobProfile,
		&companyProfile,
		experienceBank,
		repairTemplateFile,
		candidateInfo,
		repairMaxPages,
		repairMaxChars,
		repairMaxIterations,
		apiKey,
	)
	if err != nil {
		return fmt.Errorf("repair loop failed: %w", err)
	}

	// Write outputs
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
				// Log warning but don't fail
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Output %s does not validate against schema: %v\n", outputPath, err)
			}
		}
	}

	// Print summary
	_, _ = fmt.Fprintf(os.Stdout, "Repair loop completed after %d iteration(s)\n", iterations)
	if len(finalViolations.Violations) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "✅ All violations resolved\n")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "⚠️  %d violation(s) remaining\n", len(finalViolations.Violations))
	}
	_, _ = fmt.Fprintf(os.Stdout, "Output directory: %s\n", repairOutputDir)

	return nil
}
