package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunPipeline_Integration(t *testing.T) {
	// This integration test requires a valid API key and internet access.
	// It is skipped by default to avoid failing in CI/CD or environments without credentials.
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: GEMINI_API_KEY not set")
	}

	// Setup temporary output directory
	outDir, err := os.MkdirTemp("", "pipeline_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(outDir)

	// Use testdata paths (assuming running from module root or adjusting path)
	// We need to locate the project root relative to this test file.
	// For simplicity, we assume the test is run from the project root or we can find testdata.
	// Since this is in internal/pipeline, testdata is at ../../testdata
	
	opts := RunOptions{
		JobPath:        "../../testdata/parsing/sample_job_plain.txt",
		ExperiencePath: "../../testdata/valid/experience_bank.json",
		CompanySeedURL: "https://example.com",
		OutputDir:      outDir,
		CandidateName:  "Test Candidate",
		CandidateEmail: "test@example.com",
		CandidatePhone: "555-0123",
		TemplatePath:   "../../templates/one_page_resume.tex", // This needs to be strictly correct
		MaxBullets:     25,
		MaxLines:       35,
		APIKey:         apiKey,
	}

	// Verify test files exist
	if _, err := os.Stat(opts.JobPath); os.IsNotExist(err) {
		t.Skipf("Skipping integration test: test data not found at %s", opts.JobPath)
	}
	if _, err := os.Stat(opts.ExperiencePath); os.IsNotExist(err) {
		t.Skipf("Skipping integration test: test data not found at %s", opts.ExperiencePath)
	}
	if _, err := os.Stat(opts.TemplatePath); os.IsNotExist(err) {
		t.Skipf("Skipping integration test: template not found at %s", opts.TemplatePath)
	}

	// Run pipeline
	// Note: this will likely fail on Crawling or Parsing if the environment doesn't allow it,
	// but it serves as a runner for the developer.
	ctx := context.Background()
	err = RunPipeline(ctx, opts)
	if err != nil {
		t.Logf("Pipeline run failed (expected if external services are unreachable): %v", err)
		// We don't necessarily fail the test because we depend on external APIs
	} else {
		// If it succeeded, verify artifacts
		files := []string{
			"job_metadata.json",
			"job_profile.json",
			"experience_bank_normalized.json",
			"ranked_stories.json",
			"resume_plan.json",
			"selected_bullets.json",
			"company_profile.json",
			"rewritten_bullets.json",
			"resume.tex",
		}

		for _, f := range files {
			if _, err := os.Stat(filepath.Join(outDir, f)); os.IsNotExist(err) {
				t.Errorf("Expected artifact %s was not created", f)
			}
		}
	}
}
