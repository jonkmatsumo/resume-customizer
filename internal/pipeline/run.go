// Package pipeline provides the high-level orchestration for the resume generation process.
package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// RunOptions holds configuration for running the pipeline
type RunOptions struct {
	JobPath        string
	JobURL         string
	ExperiencePath string
	CompanySeedURL string
	OutputDir      string
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

// RunPipeline orchestrates the full resume generation pipeline
func RunPipeline(ctx context.Context, opts RunOptions) error {
	// 1. Ensure output directory exists
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

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
		// Write the ingested job to the output directory so subsequent runs/debug can use it
		if err := ingestion.WriteOutput(opts.OutputDir, cleanedText, jobMetadata); err != nil {
			return fmt.Errorf("failed to write ingested job: %w", err)
		}
		if opts.Verbose {
			fmt.Printf("[VERBOSE] Ingested job saved to %s\n", filepath.Join(opts.OutputDir, "job_posting.cleaned.txt"))
		}
	} else {
		fmt.Printf("Step 1/12: Ingesting job posting from file: %s...\n", opts.JobPath)
		cleanedText, jobMetadata, err = ingestion.IngestFromFile(ctx, opts.JobPath, opts.APIKey)
		if err != nil {
			return fmt.Errorf("job ingestion from file failed: %w", err)
		}
	}

	// Save job metadata
	jobMetaPath := filepath.Join(opts.OutputDir, "job_metadata.json")
	if err := saveJSON(jobMetaPath, jobMetadata); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved job metadata to %s\n", jobMetaPath)
	}

	fmt.Printf("Step 2/12: Parsing job profile...\n")
	jobProfile, err := parsing.ParseJobProfile(ctx, cleanedText, opts.APIKey)
	if err != nil {
		return fmt.Errorf("job parsing failed: %w", err)
	}
	jobProfilePath := filepath.Join(opts.OutputDir, "job_profile.json")
	if err := saveJSON(jobProfilePath, jobProfile); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved job profile to %s\n", jobProfilePath)
		printer.PrintJobProfile(jobProfile)
	}

	// Save to database if connected
	if database != nil {
		runID, err = database.CreateRun(ctx, jobProfile.Company, jobProfile.RoleTitle, opts.JobURL)
		if err != nil {
			fmt.Printf("Warning: Failed to create database run: %v\n", err)
		} else {
			if opts.Verbose {
				fmt.Printf("[VERBOSE] Created database run: %s\n", runID)
			}
			// Save initial artifacts
			_ = database.SaveTextArtifact(ctx, runID, db.StepJobPosting, cleanedText)
			_ = database.SaveArtifact(ctx, runID, db.StepJobMetadata, jobMetadata)
			_ = database.SaveArtifact(ctx, runID, db.StepJobProfile, jobProfile)
		}
	}

	fmt.Printf("Step 2a/12: Extracting education requirements...\n")
	eduReq, err := parsing.ExtractEducationRequirements(ctx, cleanedText, opts.APIKey)
	if err != nil {
		fmt.Printf("Warning: Failed to extract education requirements: %v\n", err)
	} else {
		jobProfile.EducationRequirements = eduReq
		eduReqPath := filepath.Join(opts.OutputDir, "education_requirements.json")
		if err := saveJSON(eduReqPath, eduReq); err != nil {
			return err
		}
		if opts.Verbose {
			fmt.Printf("[VERBOSE] Saved education requirements to %s\n", eduReqPath)
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
		result, err := runExperienceBranch(gCtx, opts, jobProfile, cleanedText, printer)
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
		result, err := runResearchBranch(gCtx, opts, jobProfile, jobMetadata, printer)
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
	rewrittenBullets, err := rewriting.RewriteBullets(ctx, experienceResult.SelectedBullets, jobProfile, researchResult.CompanyProfile, opts.APIKey)
	if err != nil {
		return fmt.Errorf("rewriting bullets failed: %w", err)
	}
	rewrittenBulletsPath := filepath.Join(opts.OutputDir, "rewritten_bullets.json")
	if err := saveJSON(rewrittenBulletsPath, rewrittenBullets); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved rewritten bullets to %s\n", rewrittenBulletsPath)
		printer.PrintRewrittenBullets(rewrittenBullets)
	}

	fmt.Printf("Step 10/12: Rendering LaTeX resume...\n")
	latex, err := rendering.RenderLaTeX(experienceResult.ResumePlan, rewrittenBullets, opts.TemplatePath, opts.CandidateName, opts.CandidateEmail, opts.CandidatePhone, experienceResult.ExperienceBank, experienceResult.SelectedEducation)
	if err != nil {
		return fmt.Errorf("rendering latex failed: %w", err)
	}
	latexPath := filepath.Join(opts.OutputDir, "resume.tex")
	if err := os.WriteFile(latexPath, []byte(latex), 0644); err != nil {
		return fmt.Errorf("failed to write resume.tex: %w", err)
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved LaTeX resume to %s\n", latexPath)
	}

	fmt.Printf("Step 11/12: Validating LaTeX constraints...\n")
	violations, err := validation.ValidateConstraints(latexPath, researchResult.CompanyProfile, 1, 100) // Default max 1 page, 100 chars per line (approx)
	if err != nil {
		return fmt.Errorf("validating latex failed: %w", err)
	}
	violationsPath := filepath.Join(opts.OutputDir, "violations.json")
	if err := saveJSON(violationsPath, violations); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved violations to %s\n", violationsPath)
		printer.PrintViolations(violations)
	}

	if violations != nil && len(violations.Violations) > 0 {
		fmt.Printf("Step 12/12: Violations found (%d), entering repair loop...\n", len(violations.Violations))

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
			100, // max chars per line
			5,   // max iterations
			opts.APIKey,
		)
		if err != nil {
			return fmt.Errorf("repair loop failed: %w", err)
		}

		// Save final artifacts
		finalPlanPath := filepath.Join(opts.OutputDir, "resume_plan_final.json")
		if err := saveJSON(finalPlanPath, finalPlan); err != nil {
			return err
		}
		if opts.Verbose {
			fmt.Printf("[VERBOSE] Saved final resume plan to %s\n", finalPlanPath)
		}

		finalBulletsPath := filepath.Join(opts.OutputDir, "rewritten_bullets_final.json")
		if err := saveJSON(finalBulletsPath, finalBullets); err != nil {
			return err
		}
		if opts.Verbose {
			fmt.Printf("[VERBOSE] Saved final rewritten bullets to %s\n", finalBulletsPath)
		}

		finalViolationsPath := filepath.Join(opts.OutputDir, "violations_final.json")
		if err := saveJSON(finalViolationsPath, finalViolations); err != nil {
			return err
		}
		if opts.Verbose {
			fmt.Printf("[VERBOSE] Saved final violations to %s\n", finalViolationsPath)
		}

		if err := os.WriteFile(filepath.Join(opts.OutputDir, "resume.tex"), []byte(finalLaTeX), 0644); err != nil {
			return fmt.Errorf("failed to overwrite resume.tex with final version: %w", err)
		}
		if opts.Verbose {
			fmt.Printf("[VERBOSE] Overwrote resume.tex with final version\n")
		}

		if finalViolations != nil && len(finalViolations.Violations) > 0 {
			fmt.Printf("⚠️ Warning: Repair loop finished after %d iterations but %d violations remain.\n", iterations, len(finalViolations.Violations))
		} else {
			fmt.Printf("✅ Successfully repaired all violations in %d iterations!\n", iterations)
		}

	} else {
		fmt.Printf("Step 12/12: Validation passed! No repairs needed.\n")
	}

	fmt.Printf("Done! Generated resume at %s\n", latexPath)
	return nil
}

// runExperienceBranch executes Steps 3-6: Loading, ranking, selecting, and materializing experience
func runExperienceBranch(ctx context.Context, opts RunOptions, jobProfile *types.JobProfile, cleanedText string, printer *observability.Printer) (*ExperienceBranchResult, error) {
	prefix := prefixExperience

	fmt.Printf("%sStep 3/12: Loading and normalizing experience bank from %s...\n", prefix, opts.ExperiencePath)
	experienceBank, err := experience.LoadExperienceBank(opts.ExperiencePath)
	if err != nil {
		return nil, fmt.Errorf("loading experience bank failed: %w", err)
	}
	if err := experience.NormalizeExperienceBank(experienceBank); err != nil {
		return nil, fmt.Errorf("normalizing experience bank failed: %w", err)
	}
	expBankPath := filepath.Join(opts.OutputDir, "experience_bank_normalized.json")
	if err := saveJSON(expBankPath, experienceBank); err != nil {
		return nil, err
	}
	if opts.Verbose {
		fmt.Printf("%s[VERBOSE] Saved normalized experience bank to %s\n", prefix, expBankPath)
	}

	fmt.Printf("%sStep 4/12: Ranking stories...\n", prefix)
	rankedStories, err := ranking.RankStories(jobProfile, experienceBank)
	if err != nil {
		return nil, fmt.Errorf("ranking stories failed: %w", err)
	}
	rankedStoriesPath := filepath.Join(opts.OutputDir, "ranked_stories.json")
	if err := saveJSON(rankedStoriesPath, rankedStories); err != nil {
		return nil, err
	}
	if opts.Verbose {
		fmt.Printf("%s[VERBOSE] Saved ranked stories to %s\n", prefix, rankedStoriesPath)
		printer.PrintRankedStories(rankedStories)
	}

	fmt.Printf("%sStep 4a/12: Scoring education relevance...\n", prefix)
	var selectedEducation []types.Education
	eduScores, err := ranking.ScoreEducation(ctx, experienceBank.Education, jobProfile.EducationRequirements, cleanedText, opts.APIKey)
	if err != nil {
		fmt.Printf("%sWarning: Education scoring failed: %v. Including all education.\n", prefix, err)
		selectedEducation = experienceBank.Education
	} else {
		eduScoresPath := filepath.Join(opts.OutputDir, "education_scores.json")
		if err := saveJSON(eduScoresPath, eduScores); err != nil {
			return nil, err
		}
		if opts.Verbose {
			fmt.Printf("%s[VERBOSE] Saved education scores to %s\n", prefix, eduScoresPath)
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
	spaceBudget := &types.SpaceBudget{
		MaxBullets: opts.MaxBullets,
		MaxLines:   opts.MaxLines,
	}
	resumePlan, err := selection.SelectPlan(rankedStories, jobProfile, experienceBank, spaceBudget)
	if err != nil {
		return nil, fmt.Errorf("selecting plan failed: %w", err)
	}
	planPath := filepath.Join(opts.OutputDir, "resume_plan.json")
	if err := saveJSON(planPath, resumePlan); err != nil {
		return nil, err
	}
	if opts.Verbose {
		fmt.Printf("%s[VERBOSE] Saved resume plan to %s\n", prefix, planPath)
	}

	fmt.Printf("%sStep 6/12: Materializing selected bullets...\n", prefix)
	selectedBullets, err := selection.MaterializeBullets(resumePlan, experienceBank)
	if err != nil {
		return nil, fmt.Errorf("materializing bullets failed: %w", err)
	}
	bulletsPath := filepath.Join(opts.OutputDir, "selected_bullets.json")
	if err := saveJSON(bulletsPath, selectedBullets); err != nil {
		return nil, err
	}
	if opts.Verbose {
		fmt.Printf("%s[VERBOSE] Saved selected bullets to %s\n", prefix, bulletsPath)
		printer.PrintSelectedBullets(selectedBullets)
	}

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
func runResearchBranch(ctx context.Context, opts RunOptions, jobProfile *types.JobProfile, jobMetadata *ingestion.Metadata, printer *observability.Printer) (*ResearchBranchResult, error) {
	prefix := prefixResearch

	fmt.Printf("%sStep 7/12: Researching company voice...\n", prefix)

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

	// Fallback/Augment with provided seed if not already in list
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
		return nil, fmt.Errorf("research failed: %w", err)
	}

	// Build corpus from research session
	companyCorpus := &types.CompanyCorpus{
		Corpus:  researchSession.Corpus,
		Sources: researchSession.ToSources(),
	}

	// Save sources
	sourcesPath := filepath.Join(opts.OutputDir, "sources.json")
	if err := saveJSON(sourcesPath, companyCorpus.Sources); err != nil {
		return nil, err
	}
	if opts.Verbose {
		fmt.Printf("%s[VERBOSE] Saved research sources to %s\n", prefix, sourcesPath)
	}

	// Save corpus text for debug
	corpusPath := filepath.Join(opts.OutputDir, "company_corpus.txt")
	if err := os.WriteFile(corpusPath, []byte(companyCorpus.Corpus), 0644); err != nil {
		return nil, fmt.Errorf("failed to save company corpus text: %w", err)
	}
	if opts.Verbose {
		fmt.Printf("%s[VERBOSE] Saved company corpus text to %s\n", prefix, corpusPath)
	}

	// Save research session for debug
	sessionPath := filepath.Join(opts.OutputDir, "research_session.json")
	if err := saveJSON(sessionPath, researchSession); err != nil {
		return nil, err
	}
	if opts.Verbose {
		fmt.Printf("%s[VERBOSE] Saved research session to %s\n", prefix, sessionPath)
	}

	fmt.Printf("%sStep 8/12: Summarizing company voice...\n", prefix)
	companyProfile, err := voice.SummarizeVoice(ctx, companyCorpus.Corpus, companyCorpus.Sources, opts.APIKey)
	if err != nil {
		return nil, fmt.Errorf("summarizing voice failed: %w", err)
	}
	voiceProfilePath := filepath.Join(opts.OutputDir, "company_profile.json")
	if err := saveJSON(voiceProfilePath, companyProfile); err != nil {
		return nil, err
	}
	if opts.Verbose {
		fmt.Printf("%s[VERBOSE] Saved company profile to %s\n", prefix, voiceProfilePath)
		printer.PrintCompanyProfile(companyProfile)
	}

	fmt.Printf("%s✅ Research branch complete.\n", prefix)

	return &ResearchBranchResult{
		CompanyProfile: companyProfile,
		CompanyCorpus:  companyCorpus,
	}, nil
}

func saveJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal json for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write json file %s: %w", path, err)
	}
	return nil
}
