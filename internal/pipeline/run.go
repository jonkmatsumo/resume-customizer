// Package pipeline provides the high-level orchestration for the resume generation process.
package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/experience"
	"github.com/jonathan/resume-customizer/internal/fetch"
	"github.com/jonathan/resume-customizer/internal/ingestion"
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
}

// RunPipeline orchestrates the full resume generation pipeline
func RunPipeline(ctx context.Context, opts RunOptions) error {
	// 1. Ensure output directory exists
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
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

	fmt.Printf("Step 3/12: Loading and normalizing experience bank from %s...\n", opts.ExperiencePath)
	experienceBank, err := experience.LoadExperienceBank(opts.ExperiencePath)
	if err != nil {
		return fmt.Errorf("loading experience bank failed: %w", err)
	}
	if err := experience.NormalizeExperienceBank(experienceBank); err != nil {
		return fmt.Errorf("normalizing experience bank failed: %w", err)
	}
	expBankPath := filepath.Join(opts.OutputDir, "experience_bank_normalized.json")
	if err := saveJSON(expBankPath, experienceBank); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved normalized experience bank to %s\n", expBankPath)
	}

	fmt.Printf("Step 4/12: Ranking stories...\n")
	rankedStories, err := ranking.RankStories(jobProfile, experienceBank)
	if err != nil {
		return fmt.Errorf("ranking stories failed: %w", err)
	}
	rankedStoriesPath := filepath.Join(opts.OutputDir, "ranked_stories.json")
	if err := saveJSON(rankedStoriesPath, rankedStories); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved ranked stories to %s\n", rankedStoriesPath)
	}

	fmt.Printf("Step 4a/12: Scoring education relevance...\n")
	var selectedEducation []types.Education
	eduScores, err := ranking.ScoreEducation(ctx, experienceBank.Education, jobProfile.EducationRequirements, cleanedText, opts.APIKey)
	if err != nil {
		fmt.Printf("Warning: Education scoring failed: %v. Including all education.\n", err)
		selectedEducation = experienceBank.Education
	} else {
		eduScoresPath := filepath.Join(opts.OutputDir, "education_scores.json")
		if err := saveJSON(eduScoresPath, eduScores); err != nil {
			return err
		}
		if opts.Verbose {
			fmt.Printf("[VERBOSE] Saved education scores to %s\n", eduScoresPath)
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

	fmt.Printf("Step 5/12: Selecting optimum resume plan...\n")
	spaceBudget := &types.SpaceBudget{
		MaxBullets: opts.MaxBullets,
		MaxLines:   opts.MaxLines,
	}
	resumePlan, err := selection.SelectPlan(rankedStories, jobProfile, experienceBank, spaceBudget)
	if err != nil {
		return fmt.Errorf("selecting plan failed: %w", err)
	}
	planPath := filepath.Join(opts.OutputDir, "resume_plan.json")
	if err := saveJSON(planPath, resumePlan); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved resume plan to %s\n", planPath)
	}

	fmt.Printf("Step 6/12: Materializing selected bullets...\n")
	selectedBullets, err := selection.MaterializeBullets(resumePlan, experienceBank)
	if err != nil {
		return fmt.Errorf("materializing bullets failed: %w", err)
	}
	bulletsPath := filepath.Join(opts.OutputDir, "selected_bullets.json")
	if err := saveJSON(bulletsPath, selectedBullets); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved selected bullets to %s\n", bulletsPath)
	}

	fmt.Printf("Step 7/12: Researching company voice...\\n")

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
		fmt.Printf("Debug: Google Search API keys not found in environment (GOOGLE_SEARCH_API_KEY: %t, GOOGLE_SEARCH_CX: %t)\n", googleKey != "", googleCX != "")
	}

	if googleKey != "" && googleCX != "" {
		if opts.Verbose {
			fmt.Printf("[VERBOSE] Using Google Search for discovery...\n")
		}
		researcher, err := research.NewResearcher(ctx, googleKey, googleCX)
		if err == nil {
			// 1. Discover website if not provided
			companyWebsite := opts.CompanySeedURL
			if companyWebsite == "" && companyName != "" {
				website, err := researcher.DiscoverCompanyWebsite(ctx, jobProfile)
				if err != nil {
					fmt.Printf("Warning: Failed to discover company website: %v\n", err)
				} else if website != "" {
					fmt.Printf("Discovered company website: %s\n", website)
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
					fmt.Printf("Warning: Failed to find voice seeds: %v\n", err)
				} else if len(discoveredSeeds) > 0 {
					fmt.Printf("Discovered %d additional voice seeds\n", len(discoveredSeeds))
					seeds = append(seeds, discoveredSeeds...)
				}
			}
		} else {
			fmt.Printf("Warning: Failed to initialize researcher: %v\n", err)
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
		return fmt.Errorf("no company seed URL provided and discovery failed. Set GOOGLE_SEARCH_API_KEY and GOOGLE_SEARCH_CX env vars for auto-discovery, or provide --company-seed")
	}

	fmt.Printf("Researching company voice with LLM-guided crawling (seeds: %v)...\n", seeds)

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
		return fmt.Errorf("research failed: %w", err)
	}

	// Build corpus from research session
	companyCorpus := &types.CompanyCorpus{
		Corpus:  researchSession.Corpus,
		Sources: researchSession.ToSources(),
	}

	// Save sources
	sourcesPath := filepath.Join(opts.OutputDir, "sources.json")
	if err := saveJSON(sourcesPath, companyCorpus.Sources); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved research sources to %s\n", sourcesPath)
	}

	// Save corpus text for debug
	corpusPath := filepath.Join(opts.OutputDir, "company_corpus.txt")
	if err := os.WriteFile(corpusPath, []byte(companyCorpus.Corpus), 0644); err != nil {
		return fmt.Errorf("failed to save company corpus text: %w", err)
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved company corpus text to %s\n", corpusPath)
	}

	// Save research session for debug
	sessionPath := filepath.Join(opts.OutputDir, "research_session.json")
	if err := saveJSON(sessionPath, researchSession); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved research session to %s\n", sessionPath)
	}

	fmt.Printf("Step 8/12: Summarizing company voice...\n")
	companyProfile, err := voice.SummarizeVoice(ctx, companyCorpus.Corpus, companyCorpus.Sources, opts.APIKey)
	if err != nil {
		return fmt.Errorf("summarizing voice failed: %w", err)
	}
	voiceProfilePath := filepath.Join(opts.OutputDir, "company_profile.json")
	if err := saveJSON(voiceProfilePath, companyProfile); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved company profile to %s\n", voiceProfilePath)
	}

	fmt.Printf("Step 9/12: Rewriting bullets to match voice...\n")
	rewrittenBullets, err := rewriting.RewriteBullets(ctx, selectedBullets, jobProfile, companyProfile, opts.APIKey)
	if err != nil {
		return fmt.Errorf("rewriting bullets failed: %w", err)
	}
	rewrittenBulletsPath := filepath.Join(opts.OutputDir, "rewritten_bullets.json")
	if err := saveJSON(rewrittenBulletsPath, rewrittenBullets); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved rewritten bullets to %s\n", rewrittenBulletsPath)
	}

	fmt.Printf("Step 10/12: Rendering LaTeX resume...\n")
	latex, err := rendering.RenderLaTeX(resumePlan, rewrittenBullets, opts.TemplatePath, opts.CandidateName, opts.CandidateEmail, opts.CandidatePhone, experienceBank, selectedEducation)
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
	violations, err := validation.ValidateConstraints(latexPath, companyProfile, 1, 100) // Default max 1 page, 100 chars per line (approx)
	if err != nil {
		return fmt.Errorf("validating latex failed: %w", err)
	}
	violationsPath := filepath.Join(opts.OutputDir, "violations.json")
	if err := saveJSON(violationsPath, violations); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Printf("[VERBOSE] Saved violations to %s\n", violationsPath)
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
			resumePlan,
			rewrittenBullets,
			violations,
			rankedStories,
			jobProfile,
			companyProfile,
			experienceBank,
			opts.TemplatePath,
			candidateInfo,
			selectedEducation,
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
