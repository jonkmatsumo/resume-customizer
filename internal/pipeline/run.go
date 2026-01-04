// Package pipeline provides the high-level orchestration for the resume generation process.
package pipeline

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/experience"
	"github.com/jonathan/resume-customizer/internal/fetch"
	"github.com/jonathan/resume-customizer/internal/ingestion"
	"github.com/jonathan/resume-customizer/internal/observability"
	"github.com/jonathan/resume-customizer/internal/parsing"
	"github.com/jonathan/resume-customizer/internal/ranking"
	"github.com/jonathan/resume-customizer/internal/rendering"
	"github.com/jonathan/resume-customizer/internal/repair"
	"github.com/jonathan/resume-customizer/internal/research"
	"github.com/jonathan/resume-customizer/internal/rewriting"
	"github.com/jonathan/resume-customizer/internal/selection"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/jonathan/resume-customizer/internal/validation"
	"github.com/jonathan/resume-customizer/internal/voice"
)

// ProgressEvent represents a progress update during pipeline execution
type ProgressEvent struct {
	Step     string `json:"step"`
	Category string `json:"category"`
	Message  string `json:"message"`
	RunID    string `json:"run_id,omitempty"`
	Content  any    `json:"content,omitempty"`
}

// ProgressCallback is called when pipeline progress occurs
type ProgressCallback func(event ProgressEvent)

// RunOptions holds configuration for running the pipeline
type RunOptions struct {
	JobPath        string
	JobURL         string
	ExperienceData *types.ExperienceBank // Required: Direct data injection
	CompanySeedURL string
	CandidateName  string
	CandidateEmail string
	CandidatePhone string
	TemplatePath   string
	MaxBullets     int
	MaxLines       int
	APIKey         string
	UseBrowser     bool
	Verbose        bool
	DatabaseURL    string
	OnProgress     ProgressCallback
	ExistingRunID  *uuid.UUID // Optional: Use existing run ID instead of creating new one
	RunStartedSent bool       // Flag to indicate run_started event was already sent
}

// ExperienceBranchResult holds the outputs from the experience processing branch
type ExperienceBranchResult struct {
	SelectedBullets   *types.SelectedBullets
	RankedStories     *types.RankedStories
	ExperienceBank    *types.ExperienceBank
	SelectedEducation []types.Education
	ResumePlan        *types.ResumePlan
}

// ResearchBranchResult holds the outputs from the research/voice branch
type ResearchBranchResult struct {
	CompanyProfile *types.CompanyProfile
	CompanyCorpus  *types.CompanyCorpus
}

// logPrefix is used to distinguish concurrent log output
type logPrefix string

const (
	prefixExperience logPrefix = "[Experience] "
	prefixResearch   logPrefix = "[Research]   "
)

// stepNameMap maps pipeline step constants to step registry names
var stepNameMap = map[string]string{
	db.StepJobPosting:       "ingest_job",
	db.StepJobProfile:       "parse_job",
	db.StepEducationReq:     "extract_education",
	db.StepExperienceBank:   "load_experience",
	db.StepRankedStories:    "rank_stories",
	db.StepEducationScores:  "score_education",
	db.StepResumePlan:       "select_plan",
	db.StepSelectedBullets:  "materialize_bullets",
	db.StepSources:          "research_company",
	db.StepCompanyProfile:   "summarize_voice",
	db.StepRewrittenBullets: "rewrite_bullets",
	db.StepResumeTex:        "render_latex",
	db.StepViolations:       "validate_latex",
}

// stepCategoryMap maps pipeline step constants to step categories
var stepCategoryMap = map[string]string{
	db.StepJobPosting:       db.StepCategoryIngestion,
	db.StepJobProfile:       db.StepCategoryIngestion,
	db.StepEducationReq:     db.StepCategoryIngestion,
	db.StepExperienceBank:   db.StepCategoryExperience,
	db.StepRankedStories:    db.StepCategoryExperience,
	db.StepEducationScores:  db.StepCategoryExperience,
	db.StepResumePlan:       db.StepCategoryExperience,
	db.StepSelectedBullets:  db.StepCategoryExperience,
	db.StepSources:          db.StepCategoryResearch,
	db.StepCompanyProfile:   db.StepCategoryResearch,
	db.StepRewrittenBullets: db.StepCategoryRewriting,
	db.StepResumeTex:        db.StepCategoryValidation,
	db.StepViolations:       db.StepCategoryValidation,
}

// emitProgress calls the progress callback if configured
func emitProgress(opts *RunOptions, step, category, message string, content any) {
	if opts.OnProgress != nil {
		opts.OnProgress(ProgressEvent{
			Step:     step,
			Category: category,
			Message:  message,
			Content:  content,
		})
	}
}

// emitRunStarted emits the run_started event with the run ID as the first streamed event
func emitRunStarted(opts *RunOptions, runID uuid.UUID) {
	if opts.OnProgress != nil {
		opts.OnProgress(ProgressEvent{
			Step:     db.StepRunStarted,
			Category: db.CategoryLifecycle,
			Message:  "Pipeline run started",
			RunID:    runID.String(),
		})
	}
}

// startStep creates or updates a run step to "in_progress" status
// stepConstant can be either a db.Step* constant or a direct step name (e.g., "repair_violations")
func startStep(ctx context.Context, database *db.DB, runID uuid.UUID, stepConstant string) error {
	if database == nil || runID == uuid.Nil {
		return nil // Skip if no database connection
	}

	// Check if stepConstant is a mapped constant or a direct step name
	stepName, ok := stepNameMap[stepConstant]
	if !ok {
		// If not in map, assume it's already a step name (e.g., "repair_violations")
		stepName = stepConstant
	}

	category := stepCategoryMap[stepConstant]
	if category == "" {
		// Default category based on step name if not in map
		if stepName == "repair_violations" {
			category = db.StepCategoryValidation
		} else {
			category = db.StepCategoryIngestion // Default category
		}
	}

	// Check if step already exists
	existingStep, err := database.GetRunStep(ctx, runID, stepName)
	if err != nil {
		return fmt.Errorf("failed to get existing step: %w", err)
	}

	if existingStep == nil {
		// Create new step
		stepInput := &db.RunStepInput{
			Step:     stepName,
			Category: category,
			Status:   db.StepStatusInProgress,
		}
		_, err = database.CreateRunStep(ctx, runID, stepInput)
		if err != nil {
			return fmt.Errorf("failed to create step: %w", err)
		}
	} else {
		// Update existing step to in_progress
		err = database.UpdateRunStepStatus(ctx, runID, stepName, db.StepStatusInProgress, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to update step status: %w", err)
		}
	}

	return nil
}

// completeStep updates a run step to "completed" status
// stepConstant can be either a db.Step* constant or a direct step name (e.g., "repair_violations")
func completeStep(ctx context.Context, database *db.DB, runID uuid.UUID, stepConstant string, artifactID *uuid.UUID) error {
	if database == nil || runID == uuid.Nil {
		return nil // Skip if no database connection
	}

	// Check if stepConstant is a mapped constant or a direct step name
	stepName, ok := stepNameMap[stepConstant]
	if !ok {
		// If not in map, assume it's already a step name (e.g., "repair_violations")
		stepName = stepConstant
	}

	err := database.UpdateRunStepStatus(ctx, runID, stepName, db.StepStatusCompleted, nil, artifactID)
	if err != nil {
		return fmt.Errorf("failed to complete step: %w", err)
	}

	return nil
}

// failStep updates a run step to "failed" status with an error message
// stepConstant can be either a db.Step* constant or a direct step name (e.g., "repair_violations")
func failStep(ctx context.Context, database *db.DB, runID uuid.UUID, stepConstant string, err error) error {
	if database == nil || runID == uuid.Nil {
		return nil // Skip if no database connection
	}

	// Check if stepConstant is a mapped constant or a direct step name
	stepName, ok := stepNameMap[stepConstant]
	if !ok {
		// If not in map, assume it's already a step name (e.g., "repair_violations")
		stepName = stepConstant
	}

	errMsg := err.Error()
	err = database.UpdateRunStepStatus(ctx, runID, stepName, db.StepStatusFailed, &errMsg, nil)
	if err != nil {
		return fmt.Errorf("failed to fail step: %w", err)
	}

	return nil
}

// countBullets returns the total number of bullets in an experience bank
func countBullets(bank *types.ExperienceBank) int {
	count := 0
	for _, story := range bank.Stories {
		count += len(story.Bullets)
	}
	return count
}

// RunPipeline orchestrates the full resume generation pipeline
func RunPipeline(ctx context.Context, opts RunOptions) error {

	// Initialize observability printer for verbose output
	printer := observability.NewPrinter(os.Stdout)

	// Initialize database connection if configured
	var database *db.DB
	var runID uuid.UUID
	if opts.DatabaseURL != "" {
		var err error
		database, err = db.Connect(ctx, opts.DatabaseURL)
		if err != nil {
			fmt.Printf("Warning: Failed to connect to database: %v\n", err)
			fmt.Printf("Continuing without database persistence...\n")
		} else {
			defer database.Close()
			if opts.Verbose {
				fmt.Printf("[VERBOSE] Connected to database\n")
			}
		}
	}

	// Step 1: Ingest job posting (from URL or File)
	var cleanedText string
	var jobMetadata *ingestion.Metadata
	var err error

	if opts.JobURL != "" {
		fmt.Printf("Step 1/12: Ingesting job posting from URL: %s...\n", opts.JobURL)
		cleanedText, jobMetadata, err = ingestion.IngestFromURL(ctx, opts.JobURL, opts.APIKey, opts.UseBrowser, opts.Verbose)
		if err != nil {
			return fmt.Errorf("job ingestion from URL failed: %w", err)
		}
	} else {
		fmt.Printf("Step 1/12: Ingesting job posting from file: %s...\n", opts.JobPath)
		cleanedText, jobMetadata, err = ingestion.IngestFromFile(ctx, opts.JobPath, opts.APIKey)
		if err != nil {
			return fmt.Errorf("job ingestion from file failed: %w", err)
		}
	}

	emitProgress(&opts, db.StepJobPosting, db.CategoryIngestion,
		fmt.Sprintf("Ingested and cleaned job posting from %s", opts.JobURL), nil)

	fmt.Printf("Step 2/12: Parsing job profile...\n")
	jobProfile, err := parsing.ParseJobProfile(ctx, cleanedText, opts.APIKey)
	if err != nil {
		if database != nil && runID != uuid.Nil {
			_ = failStep(ctx, database, runID, db.StepJobProfile, err)
		}
		return fmt.Errorf("job parsing failed: %w", err)
	}
	if opts.Verbose {
		printer.PrintJobProfile(jobProfile)
	}
	emitProgress(&opts, db.StepJobProfile, db.CategoryIngestion,
		fmt.Sprintf("Parsed job profile: %s at %s", jobProfile.RoleTitle, jobProfile.Company), jobProfile)

	// Save to database if connected
	if database != nil {
		if opts.ExistingRunID != nil {
			// Use existing run ID and update company/role
			runID = *opts.ExistingRunID
			if err := database.UpdateRunCompanyAndRole(ctx, runID, jobProfile.Company, jobProfile.RoleTitle); err != nil {
				fmt.Printf("Warning: Failed to update run company/role: %v\n", err)
			}
			if opts.Verbose {
				fmt.Printf("[VERBOSE] Using existing database run: %s\n", runID)
			}
			// Don't emit run_started again - it was already sent by handleRunStream
		} else if !opts.RunStartedSent {
			// Create new run
			runID, err = database.CreateRun(ctx, jobProfile.Company, jobProfile.RoleTitle, opts.JobURL)
			if err != nil {
				fmt.Printf("Warning: Failed to create database run: %v\n", err)
			} else {
				if opts.Verbose {
					fmt.Printf("[VERBOSE] Created database run: %s\n", runID)
				}
				// Emit run_started as the first event with the run ID
				emitRunStarted(&opts, runID)
			}
		}

		if runID != uuid.Nil {
			// Track job posting step (already completed, but we track it now that we have runID)
			_ = startStep(ctx, database, runID, db.StepJobPosting)
			_ = completeStep(ctx, database, runID, db.StepJobPosting, nil)
			// Save initial artifacts
			_ = database.SaveTextArtifact(ctx, runID, db.StepJobPosting, db.CategoryIngestion, cleanedText)
			_ = database.SaveArtifact(ctx, runID, db.StepJobMetadata, db.CategoryIngestion, jobMetadata)
			// Track job profile step
			_ = startStep(ctx, database, runID, db.StepJobProfile)
			_ = database.SaveArtifact(ctx, runID, db.StepJobProfile, db.CategoryIngestion, jobProfile)
			_ = completeStep(ctx, database, runID, db.StepJobProfile, nil)
		}
	}

	fmt.Printf("Step 2a/12: Extracting education requirements...\n")
	if err := startStep(ctx, database, runID, db.StepEducationReq); err != nil {
		fmt.Printf("Warning: Failed to start step tracking: %v\n", err)
	}

	eduReq, err := parsing.ExtractEducationRequirements(ctx, cleanedText, opts.APIKey)
	if err != nil {
		fmt.Printf("Warning: Failed to extract education requirements: %v\n", err)
		_ = failStep(ctx, database, runID, db.StepEducationReq, err)
	} else {
		jobProfile.EducationRequirements = eduReq
		// Save to database
		if database != nil && runID != uuid.Nil {
			_ = database.SaveArtifact(ctx, runID, db.StepEducationReq, db.CategoryIngestion, eduReq)
			_ = completeStep(ctx, database, runID, db.StepEducationReq, nil)
		}
	}

	// =========================================================================
	// PARALLEL EXECUTION: Experience Branch + Research Branch
	// =========================================================================
	fmt.Printf("\n🚀 Starting parallel execution of Experience and Research branches...\n\n")

	g, gCtx := errgroup.WithContext(ctx)

	var experienceResult *ExperienceBranchResult
	var researchResult *ResearchBranchResult
	var expMu, resMu sync.Mutex // Protect result assignments

	// Experience Branch (Steps 3-6)
	g.Go(func() error {
		result, err := runExperienceBranch(gCtx, opts, jobProfile, cleanedText, printer, database, runID)
		if err != nil {
			return fmt.Errorf("experience branch failed: %w", err)
		}
		expMu.Lock()
		experienceResult = result
		expMu.Unlock()
		return nil
	})

	// Research Branch (Steps 7-8)
	g.Go(func() error {
		result, err := runResearchBranch(gCtx, opts, jobProfile, jobMetadata, printer, database, runID)
		if err != nil {
			return fmt.Errorf("research branch failed: %w", err)
		}
		resMu.Lock()
		researchResult = result
		resMu.Unlock()
		return nil
	})

	// Wait for both branches to complete
	if err := g.Wait(); err != nil {
		return err
	}

	fmt.Printf("\n✅ Both branches completed. Continuing with rewriting...\n\n")
	// =========================================================================

	// Step 9: Rewrite bullets (requires both branches)
	fmt.Printf("Step 9/12: Rewriting bullets to match voice...\n")
	if err := startStep(ctx, database, runID, db.StepRewrittenBullets); err != nil {
		fmt.Printf("Warning: Failed to start step tracking: %v\n", err)
	}

	rewrittenBullets, err := rewriting.RewriteBullets(ctx, experienceResult.SelectedBullets, jobProfile, researchResult.CompanyProfile, opts.APIKey)
	if err != nil {
		_ = failStep(ctx, database, runID, db.StepRewrittenBullets, err)
		return fmt.Errorf("rewriting bullets failed: %w", err)
	}
	if opts.Verbose {
		printer.PrintRewrittenBullets(rewrittenBullets)
	}
	emitProgress(&opts, db.StepRewrittenBullets, db.CategoryRewriting,
		fmt.Sprintf("Rewritten %d bullets", len(rewrittenBullets.Bullets)), nil)

	fmt.Printf("Step 10/12: Rendering LaTeX resume...\n")
	if err := startStep(ctx, database, runID, db.StepResumeTex); err != nil {
		fmt.Printf("Warning: Failed to start step tracking: %v\n", err)
	}

	latex, lineMap, err := rendering.RenderLaTeX(experienceResult.ResumePlan, rewrittenBullets, opts.TemplatePath, opts.CandidateName, opts.CandidateEmail, opts.CandidatePhone, experienceResult.ExperienceBank, experienceResult.SelectedEducation)
	if err != nil {
		_ = failStep(ctx, database, runID, db.StepResumeTex, err)
		return fmt.Errorf("rendering latex failed: %w", err)
	}
	emitProgress(&opts, db.StepResumeTex, db.CategoryValidation, "Rendered LaTeX resume", nil)

	fmt.Printf("Step 11/12: Validating LaTeX constraints...\n")
	if err := startStep(ctx, database, runID, db.StepViolations); err != nil {
		fmt.Printf("Warning: Failed to start step tracking: %v\n", err)
	}

	// Create validation options with line-to-bullet mapping
	var validationOpts *validation.Options
	if lineMap != nil {
		// Compute forbidden phrase mapping from rewritten bullets
		forbiddenPhraseMap := rewriting.CheckForbiddenPhrasesInBullets(rewrittenBullets, researchResult.CompanyProfile)

		validationOpts = &validation.Options{
			LineToBulletMap:    lineMap.LineToBullet,
			Bullets:            rewrittenBullets,
			Plan:               experienceResult.ResumePlan,
			ForbiddenPhraseMap: forbiddenPhraseMap,
		}
	}

	violations, err := validation.ValidateFromContent(latex, researchResult.CompanyProfile, 1, 200, validationOpts) // Default max 1 page, 200 chars per line (2 lines)
	if err != nil {
		_ = failStep(ctx, database, runID, db.StepViolations, err)
		return fmt.Errorf("validating latex failed: %w", err)
	}
	if opts.Verbose {
		printer.PrintViolations(violations)
	}
	// Save rewriting artifacts to database
	if database != nil && runID != uuid.Nil {
		_ = database.SaveArtifact(ctx, runID, db.StepRewrittenBullets, db.CategoryRewriting, rewrittenBullets)
		_ = completeStep(ctx, database, runID, db.StepRewrittenBullets, nil)
		_ = database.SaveTextArtifact(ctx, runID, db.StepResumeTex, db.CategoryValidation, latex)
		_ = completeStep(ctx, database, runID, db.StepResumeTex, nil)
		_ = database.SaveArtifact(ctx, runID, db.StepViolations, db.CategoryValidation, violations)
		_ = completeStep(ctx, database, runID, db.StepViolations, nil)
	}

	if violations != nil && len(violations.Violations) > 0 {
		fmt.Printf("Step 12/12: Violations found (%d), entering repair loop...\n", len(violations.Violations))

		// Track repair_violations step
		if database != nil && runID != uuid.Nil {
			_ = startStep(ctx, database, runID, "repair_violations")
		}

		candidateInfo := repair.CandidateInfo{
			Name:  opts.CandidateName,
			Email: opts.CandidateEmail,
			Phone: opts.CandidatePhone,
		}

		finalPlan, finalBullets, finalLaTeX, finalViolations, iterations, err := repair.RunRepairLoop(
			ctx,
			experienceResult.ResumePlan,
			rewrittenBullets,
			violations,
			experienceResult.RankedStories,
			jobProfile,
			researchResult.CompanyProfile,
			experienceResult.ExperienceBank,
			opts.TemplatePath,
			candidateInfo,
			experienceResult.SelectedEducation,
			1,   // max pages
			200, // max chars per line (2 lines)
			5,   // max iterations
			opts.APIKey,
		)
		if err != nil {
			if database != nil && runID != uuid.Nil {
				_ = failStep(ctx, database, runID, "repair_violations", err)
			}
			return fmt.Errorf("repair loop failed: %w", err)
		}

		// Update database with final artifacts (overwrite previous)
		if database != nil && runID != uuid.Nil {
			_ = database.SaveArtifact(ctx, runID, db.StepResumePlan, db.CategoryExperience, finalPlan)
			_ = database.SaveArtifact(ctx, runID, db.StepRewrittenBullets, db.CategoryRewriting, finalBullets)
			_ = database.SaveTextArtifact(ctx, runID, db.StepResumeTex, db.CategoryValidation, finalLaTeX)
			_ = database.SaveArtifact(ctx, runID, db.StepViolations, db.CategoryValidation, finalViolations)
			_ = completeStep(ctx, database, runID, "repair_violations", nil)
		}

		if finalViolations != nil && len(finalViolations.Violations) > 0 {
			fmt.Printf("⚠️ Warning: Repair loop finished after %d iterations but %d violations remain.\n", iterations, len(finalViolations.Violations))
		} else {
			fmt.Printf("✅ Successfully repaired all violations in %d iterations!\n", iterations)
		}

	} else {
		fmt.Printf("Step 12/12: Validation passed! No repairs needed.\n")
	}

	// Mark run as completed
	if database != nil && runID != uuid.Nil {
		_ = database.CompleteRun(ctx, runID, "completed")
	}

	fmt.Printf("Done! Resume stored in database.\n")
	return nil
}

// runExperienceBranch executes Steps 3-6: Loading, ranking, selecting, and materializing experience
func runExperienceBranch(ctx context.Context, opts RunOptions, jobProfile *types.JobProfile, cleanedText string, printer *observability.Printer, database *db.DB, runID uuid.UUID) (*ExperienceBranchResult, error) {
	prefix := prefixExperience

	fmt.Printf("%sStep 3/12: Loading and normalizing experience bank...\n", prefix)

	if err := startStep(ctx, database, runID, db.StepExperienceBank); err != nil {
		fmt.Printf("%sWarning: Failed to start step tracking: %v\n", prefix, err)
	}

	// Determine experience data source
	if opts.ExperienceData == nil {
		err := fmt.Errorf("experience data is missing (legacy file path support removed)")
		_ = failStep(ctx, database, runID, db.StepExperienceBank, err)
		return nil, err
	}

	fmt.Printf("%sUsing provided experience data (from DB)...\n", prefix)
	experienceBank := opts.ExperienceData

	if err := experience.NormalizeExperienceBank(experienceBank); err != nil {
		_ = failStep(ctx, database, runID, db.StepExperienceBank, err)
		return nil, fmt.Errorf("normalizing experience bank failed: %w", err)
	}
	emitProgress(&opts, db.StepExperienceBank, db.CategoryExperience,
		fmt.Sprintf("Loaded %d stories with %d total bullets", len(experienceBank.Stories), countBullets(experienceBank)), nil)
	// Save to database
	if database != nil && runID != uuid.Nil {
		_ = database.SaveArtifact(ctx, runID, db.StepExperienceBank, db.CategoryExperience, experienceBank)
		_ = completeStep(ctx, database, runID, db.StepExperienceBank, nil)
	}

	fmt.Printf("%sStep 4/12: Ranking stories...\n", prefix)
	if err := startStep(ctx, database, runID, db.StepRankedStories); err != nil {
		fmt.Printf("%sWarning: Failed to start step tracking: %v\n", prefix, err)
	}

	rankedStories, err := ranking.RankStories(jobProfile, experienceBank)
	if err != nil {
		_ = failStep(ctx, database, runID, db.StepRankedStories, err)
		return nil, fmt.Errorf("ranking stories failed: %w", err)
	}
	if opts.Verbose {
		printer.PrintRankedStories(rankedStories)
	}
	// Save to database
	if database != nil && runID != uuid.Nil {
		_ = database.SaveArtifact(ctx, runID, db.StepRankedStories, db.CategoryExperience, rankedStories)
		_ = completeStep(ctx, database, runID, db.StepRankedStories, nil)
	}
	emitProgress(&opts, db.StepRankedStories, db.CategoryExperience, "Ranked stories by relevance", rankedStories)

	fmt.Printf("%sStep 4a/12: Scoring education relevance...\n", prefix)
	if err := startStep(ctx, database, runID, db.StepEducationScores); err != nil {
		fmt.Printf("%sWarning: Failed to start step tracking: %v\n", prefix, err)
	}

	var selectedEducation []types.Education
	eduScores, err := ranking.ScoreEducation(ctx, experienceBank.Education, jobProfile.EducationRequirements, cleanedText, opts.APIKey)
	if err != nil {
		fmt.Printf("%sWarning: Education scoring failed: %v. Including all education.\n", prefix, err)
		selectedEducation = experienceBank.Education
		_ = failStep(ctx, database, runID, db.StepEducationScores, err)
	} else {
		// Save to database
		if database != nil && runID != uuid.Nil {
			_ = database.SaveArtifact(ctx, runID, db.StepEducationScores, db.CategoryExperience, eduScores)
			_ = completeStep(ctx, database, runID, db.StepEducationScores, nil)
		}
		// Filter based on Included flag
		for _, score := range eduScores {
			if score.Included {
				for _, edu := range experienceBank.Education {
					if edu.ID == score.EducationID {
						selectedEducation = append(selectedEducation, edu)
					}
				}
			}
		}
	}

	fmt.Printf("%sStep 5/12: Selecting optimum resume plan...\n", prefix)
	if err := startStep(ctx, database, runID, db.StepResumePlan); err != nil {
		fmt.Printf("%sWarning: Failed to start step tracking: %v\n", prefix, err)
	}

	spaceBudget := &types.SpaceBudget{
		MaxBullets: opts.MaxBullets,
		MaxLines:   opts.MaxLines,
	}
	resumePlan, err := selection.SelectPlan(rankedStories, jobProfile, experienceBank, spaceBudget)
	if err != nil {
		_ = failStep(ctx, database, runID, db.StepResumePlan, err)
		return nil, fmt.Errorf("selecting plan failed: %w", err)
	}
	// Save to database
	if database != nil && runID != uuid.Nil {
		_ = database.SaveArtifact(ctx, runID, db.StepResumePlan, db.CategoryExperience, resumePlan)
		_ = completeStep(ctx, database, runID, db.StepResumePlan, nil)
	}

	fmt.Printf("%sStep 6/12: Materializing selected bullets...\n", prefix)
	if err := startStep(ctx, database, runID, db.StepSelectedBullets); err != nil {
		fmt.Printf("%sWarning: Failed to start step tracking: %v\n", prefix, err)
	}

	selectedBullets, err := selection.MaterializeBullets(resumePlan, experienceBank)
	if err != nil {
		_ = failStep(ctx, database, runID, db.StepSelectedBullets, err)
		return nil, fmt.Errorf("materializing bullets failed: %w", err)
	}
	if opts.Verbose {
		printer.PrintSelectedBullets(selectedBullets)
	}
	// Save to database
	if database != nil && runID != uuid.Nil {
		_ = database.SaveArtifact(ctx, runID, db.StepSelectedBullets, db.CategoryExperience, selectedBullets)
		_ = completeStep(ctx, database, runID, db.StepSelectedBullets, nil)
	}
	emitProgress(&opts, db.StepSelectedBullets, db.CategoryExperience,
		fmt.Sprintf("Selected %d bullets for resume", len(selectedBullets.Bullets)), selectedBullets)

	fmt.Printf("%s✅ Experience branch complete.\n", prefix)

	return &ExperienceBranchResult{
		SelectedBullets:   selectedBullets,
		RankedStories:     rankedStories,
		ExperienceBank:    experienceBank,
		SelectedEducation: selectedEducation,
		ResumePlan:        resumePlan,
	}, nil
}

// runResearchBranch executes Steps 7-8: Company research and voice summarization
func runResearchBranch(ctx context.Context, opts RunOptions, jobProfile *types.JobProfile, jobMetadata *ingestion.Metadata, printer *observability.Printer, database *db.DB, runID uuid.UUID) (*ResearchBranchResult, error) {
	prefix := prefixResearch

	fmt.Printf("%sStep 7/12: Researching company voice...\n", prefix)

	if err := startStep(ctx, database, runID, db.StepSources); err != nil {
		fmt.Printf("%sWarning: Failed to start step tracking: %v\n", prefix, err)
	}

	// Determine seeds and company info for research
	var seeds []string
	if jobMetadata != nil && len(jobMetadata.ExtractedLinks) > 0 {
		seeds = append(seeds, jobMetadata.ExtractedLinks...)
	}

	// Build initial corpus from "About Company" section if available
	initialCorpus := ""
	if jobMetadata != nil && jobMetadata.AboutCompany != "" {
		initialCorpus = "## About the Company\n" + jobMetadata.AboutCompany + "\n\n"
	}

	companyName := jobProfile.Company
	if companyName == "" && jobMetadata != nil && jobMetadata.Company != "" {
		companyName = jobMetadata.Company
	}
	if companyName == "" && jobMetadata != nil && jobMetadata.URL != "" {
		companyName = fetch.ExtractCompanyFromURL(jobMetadata.URL)
	}
	companyDomain := ""

	// If Google Search API keys are present, try discovery
	googleKey := os.Getenv("GOOGLE_SEARCH_API_KEY")
	googleCX := os.Getenv("GOOGLE_SEARCH_CX")

	if googleKey == "" || googleCX == "" {
		fmt.Printf("%sDebug: Google Search API keys not found in environment (GOOGLE_SEARCH_API_KEY: %t, GOOGLE_SEARCH_CX: %t)\n", prefix, googleKey != "", googleCX != "")
	}

	if googleKey != "" && googleCX != "" {
		if opts.Verbose {
			fmt.Printf("%s[VERBOSE] Using Google Search for discovery...\n", prefix)
		}
		researcher, err := research.NewResearcher(ctx, googleKey, googleCX)
		if err == nil {
			// 1. Discover website if not provided
			companyWebsite := opts.CompanySeedURL
			if companyWebsite == "" && companyName != "" {
				website, err := researcher.DiscoverCompanyWebsite(ctx, jobProfile)
				if err != nil {
					fmt.Printf("%sWarning: Failed to discover company website: %v\n", prefix, err)
				} else if website != "" {
					fmt.Printf("%sDiscovered company website: %s\n", prefix, website)
					companyWebsite = website
				}
			}

			// Extract domain for research
			if companyWebsite != "" {
				companyDomain = research.ExtractDomain(companyWebsite)
				seeds = append(seeds, companyWebsite)
			}

			// 2. Find voice seeds (About, Culture, Values pages)
			if companyWebsite != "" || companyName != "" {
				discoveredSeeds, err := researcher.FindVoiceSeeds(ctx, companyName, companyWebsite)
				if err != nil {
					fmt.Printf("%sWarning: Failed to find voice seeds: %v\n", prefix, err)
				} else if len(discoveredSeeds) > 0 {
					fmt.Printf("%sDiscovered %d additional voice seeds\n", prefix, len(discoveredSeeds))
					seeds = append(seeds, discoveredSeeds...)
				}
			}
		} else {
			fmt.Printf("%sWarning: Failed to initialize researcher: %v\n", prefix, err)
		}
	}

	// Add user-provided company seed if set (not already in seeds)
	if opts.CompanySeedURL != "" {
		found := false
		for _, s := range seeds {
			if s == opts.CompanySeedURL {
				found = true
				break
			}
		}
		if !found {
			seeds = append(seeds, opts.CompanySeedURL)
		}
		// Ensure domain is set
		if companyDomain == "" {
			companyDomain = research.ExtractDomain(opts.CompanySeedURL)
		}
	}

	if len(seeds) == 0 {
		return nil, fmt.Errorf("no company seed URL provided and discovery failed. Set GOOGLE_SEARCH_API_KEY and GOOGLE_SEARCH_CX env vars for auto-discovery, or provide --company-seed")
	}

	fmt.Printf("%sResearching company voice with LLM-guided crawling (seeds: %v)...\n", prefix, seeds)

	// Use research module for smarter LLM-filtered crawling
	researchSession, err := research.RunResearch(ctx, research.RunResearchOptions{
		SeedURLs:      seeds,
		Company:       companyName,
		Domain:        companyDomain,
		InitialCorpus: initialCorpus,
		MaxPages:      5,
		APIKey:        opts.APIKey,
		Verbose:       opts.Verbose,
		UseBrowser:    opts.UseBrowser,
	})
	if err != nil {
		_ = failStep(ctx, database, runID, db.StepSources, err)
		return nil, fmt.Errorf("research failed: %w", err)
	}

	// Build corpus from research session
	companyCorpus := &types.CompanyCorpus{
		Corpus:  researchSession.Corpus,
		Sources: researchSession.ToSources(),
	}

	// Save to database
	if database != nil && runID != uuid.Nil {
		_ = database.SaveArtifact(ctx, runID, db.StepSources, db.CategoryResearch, companyCorpus.Sources)
		_ = database.SaveTextArtifact(ctx, runID, db.StepCompanyCorpus, db.CategoryResearch, companyCorpus.Corpus)
		_ = database.SaveArtifact(ctx, runID, db.StepResearchSession, db.CategoryResearch, researchSession)
		_ = completeStep(ctx, database, runID, db.StepSources, nil)
	}

	fmt.Printf("%sStep 8/12: Summarizing company voice...\n", prefix)
	if err := startStep(ctx, database, runID, db.StepCompanyProfile); err != nil {
		fmt.Printf("%sWarning: Failed to start step tracking: %v\n", prefix, err)
	}

	companyProfile, err := voice.SummarizeVoice(ctx, companyCorpus.Corpus, companyCorpus.Sources, opts.APIKey)
	if err != nil {
		_ = failStep(ctx, database, runID, db.StepCompanyProfile, err)
		return nil, fmt.Errorf("summarizing voice failed: %w", err)
	}
	if opts.Verbose {
		printer.PrintCompanyProfile(companyProfile)
	}
	// Save to database
	if database != nil && runID != uuid.Nil {
		_ = database.SaveArtifact(ctx, runID, db.StepCompanyProfile, db.CategoryResearch, companyProfile)
		_ = completeStep(ctx, database, runID, db.StepCompanyProfile, nil)
	}
	emitProgress(&opts, db.StepCompanyProfile, db.CategoryResearch,
		fmt.Sprintf("Analyzed company voice: %s", companyProfile.Company), companyProfile)

	fmt.Printf("%s✅ Research branch complete.\n", prefix)

	return &ResearchBranchResult{
		CompanyProfile: companyProfile,
		CompanyCorpus:  companyCorpus,
	}, nil
}
