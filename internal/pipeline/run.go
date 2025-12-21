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

	fmt.Printf("Step 1/12: Ingesting job posting from %s...\n", opts.JobPath)
	cleanedText, jobMetadata, err := ingestion.IngestFromFile(ctx, opts.JobPath, opts.APIKey)
	if err != nil {
		return fmt.Errorf("job ingestion failed: %w", err)
	}
	// Save job metadata
	if err := saveJSON(filepath.Join(opts.OutputDir, "job_metadata.json"), jobMetadata); err != nil {
		return err
	}

	fmt.Printf("Step 2/12: Parsing job profile...\n")
	jobProfile, err := parsing.ParseJobProfile(ctx, cleanedText, opts.APIKey)
	if err != nil {
		return fmt.Errorf("job parsing failed: %w", err)
	}
	if err := saveJSON(filepath.Join(opts.OutputDir, "job_profile.json"), jobProfile); err != nil {
		return err
	}

	fmt.Printf("Step 3/12: Loading and normalizing experience bank from %s...\n", opts.ExperiencePath)
	experienceBank, err := experience.LoadExperienceBank(opts.ExperiencePath)
	if err != nil {
		return fmt.Errorf("loading experience bank failed: %w", err)
	}
	if err := experience.NormalizeExperienceBank(experienceBank); err != nil {
		return fmt.Errorf("normalizing experience bank failed: %w", err)
	}
	if err := saveJSON(filepath.Join(opts.OutputDir, "experience_bank_normalized.json"), experienceBank); err != nil {
		return err
	}

	fmt.Printf("Step 4/12: Ranking stories...\n")
	rankedStories, err := ranking.RankStories(jobProfile, experienceBank)
	if err != nil {
		return fmt.Errorf("ranking stories failed: %w", err)
	}
	if err := saveJSON(filepath.Join(opts.OutputDir, "ranked_stories.json"), rankedStories); err != nil {
		return err
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
	if err := saveJSON(filepath.Join(opts.OutputDir, "resume_plan.json"), resumePlan); err != nil {
		return err
	}

	fmt.Printf("Step 6/12: Materializing selected bullets...\n")
	selectedBullets, err := selection.MaterializeBullets(resumePlan, experienceBank)
	if err != nil {
		return fmt.Errorf("materializing bullets failed: %w", err)
	}
	if err := saveJSON(filepath.Join(opts.OutputDir, "selected_bullets.json"), selectedBullets); err != nil {
		return err
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
		fmt.Printf("Using Google Search for discovery...\n")
		researcher, err := research.NewResearcher(ctx, googleKey, googleCX)
		if err == nil {
			// 1. Discover website if not provided
			companyWebsite := opts.CompanySeedURL
			if companyWebsite == "" && companyName != "" {
				website, err := researcher.DiscoverCompanyWebsite(ctx, jobProfile)
				if err != nil {
					fmt.Printf("Warning: Failed to discover company website: %v\n", err)
				} else {
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
				} else {
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
	if err := saveJSON(filepath.Join(opts.OutputDir, "sources.json"), companyCorpus.Sources); err != nil {
		return err
	}
	// Save corpus text for debug
	if err := os.WriteFile(filepath.Join(opts.OutputDir, "company_corpus.txt"), []byte(companyCorpus.Corpus), 0644); err != nil {
		return fmt.Errorf("failed to save company corpus text: %w", err)
	}
	// Save research session for debug
	if err := saveJSON(filepath.Join(opts.OutputDir, "research_session.json"), researchSession); err != nil {
		return err
	}

	fmt.Printf("Step 8/12: Summarizing company voice...\n")
	companyProfile, err := voice.SummarizeVoice(ctx, companyCorpus.Corpus, companyCorpus.Sources, opts.APIKey)
	if err != nil {
		return fmt.Errorf("summarizing voice failed: %w", err)
	}
	if err := saveJSON(filepath.Join(opts.OutputDir, "company_profile.json"), companyProfile); err != nil {
		return err
	}

	fmt.Printf("Step 9/12: Rewriting bullets to match voice...\n")
	rewrittenBullets, err := rewriting.RewriteBullets(ctx, selectedBullets, jobProfile, companyProfile, opts.APIKey)
	if err != nil {
		return fmt.Errorf("rewriting bullets failed: %w", err)
	}
	if err := saveJSON(filepath.Join(opts.OutputDir, "rewritten_bullets.json"), rewrittenBullets); err != nil {
		return err
	}

	fmt.Printf("Step 10/12: Rendering LaTeX resume...\n")
	latex, err := rendering.RenderLaTeX(resumePlan, rewrittenBullets, opts.TemplatePath, opts.CandidateName, opts.CandidateEmail, opts.CandidatePhone, experienceBank)
	if err != nil {
		return fmt.Errorf("rendering latex failed: %w", err)
	}
	latexPath := filepath.Join(opts.OutputDir, "resume.tex")
	if err := os.WriteFile(latexPath, []byte(latex), 0644); err != nil {
		return fmt.Errorf("failed to write resume.tex: %w", err)
	}

	fmt.Printf("Step 11/12: Validating LaTeX constraints...\n")
	violations, err := validation.ValidateConstraints(latexPath, companyProfile, 1, 120) // Default max 1 page, 120 chars per line (approx)
	if err != nil {
		return fmt.Errorf("validating latex failed: %w", err)
	}
	if err := saveJSON(filepath.Join(opts.OutputDir, "violations.json"), violations); err != nil {
		return err
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
			1,   // max pages
			120, // max chars per line
			5,   // max iterations
			opts.APIKey,
		)
		if err != nil {
			return fmt.Errorf("repair loop failed: %w", err)
		}

		// Save final artifacts
		if err := saveJSON(filepath.Join(opts.OutputDir, "resume_plan_final.json"), finalPlan); err != nil {
			return err
		}
		if err := saveJSON(filepath.Join(opts.OutputDir, "rewritten_bullets_final.json"), finalBullets); err != nil {
			return err
		}
		if err := saveJSON(filepath.Join(opts.OutputDir, "violations_final.json"), finalViolations); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(opts.OutputDir, "resume.tex"), []byte(finalLaTeX), 0644); err != nil {
			return fmt.Errorf("failed to overwrite resume.tex with final version: %w", err)
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
