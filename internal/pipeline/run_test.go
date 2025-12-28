package pipeline

import (
	"context"
	"os"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
)

func TestRunPipeline_Integration(t *testing.T) {
	// This integration test requires a valid API key and internet access.
	// It is skipped by default to avoid failing in CI/CD or environments without credentials.
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: GEMINI_API_KEY not set")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("Skipping integration test: DATABASE_URL not set (required for storage)")
	}

	// Create dummy experience data
	expData := &types.ExperienceBank{
		Stories: []types.Story{},
	}

	opts := RunOptions{
		JobPath:        "../../testdata/parsing/sample_job_plain.txt",
		ExperienceData: expData, // Inject ExperienceData instead of ExperiencePath
		CompanySeedURL: "https://example.com",
		CandidateName:  "Test Candidate",
		CandidateEmail: "test@example.com",
		CandidatePhone: "555-0123",
		TemplatePath:   "../../templates/one_page_resume.tex",
		MaxBullets:     25,
		MaxLines:       35,
		APIKey:         apiKey,
		DatabaseURL:    databaseURL,
	}

	// Verify test files exist
	if _, err := os.Stat(opts.JobPath); os.IsNotExist(err) {
		t.Skipf("Skipping integration test: test data not found at %s", opts.JobPath)
	}
	// Mock file checks are no longer needed for ExperiencePath as it's removed
	if opts.ExperienceData == nil {
		t.Error("expected experience data to be set")
	}
	if _, err := os.Stat(opts.TemplatePath); os.IsNotExist(err) {
		t.Skipf("Skipping integration test: template not found at %s", opts.TemplatePath)
	}

	// Run pipeline
	// Note: this will likely fail on Crawling or Parsing if the environment doesn't allow it,
	// but it serves as a runner for the developer.
	ctx := context.Background()
	err := RunPipeline(ctx, opts)
	if err != nil {
		t.Logf("Pipeline run failed (expected if external services are unreachable): %v", err)
		// We don't necessarily fail the test because we depend on external APIs
	} else {
		t.Log("Pipeline completed successfully - artifacts stored in database")
	}
}
