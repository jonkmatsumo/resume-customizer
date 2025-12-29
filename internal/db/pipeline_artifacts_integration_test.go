//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// =============================================================================
// Run Ranked Stories Integration Tests
// =============================================================================

func TestIntegration_RunRankedStories_CRUD(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create test run
	runID := createTestRun(t, db, ctx)
	defer cleanupTestRun(t, db, runID)

	t.Run("save ranked stories", func(t *testing.T) {
		llmScore := 0.9
		input := []RunRankedStoryInput{
			{
				StoryIDText:      "amazon-analytics",
				RelevanceScore:   0.85,
				SkillOverlap:     0.7,
				KeywordOverlap:   0.6,
				EvidenceStrength: 0.9,
				HeuristicScore:   0.75,
				LLMScore:         &llmScore,
				LLMReasoning:     "Strong match",
				MatchedSkills:    []string{"Go", "AWS"},
				Notes:            "Top candidate",
				Ordinal:          1,
			},
			{
				StoryIDText:      "google-ml",
				RelevanceScore:   0.75,
				SkillOverlap:     0.6,
				KeywordOverlap:   0.5,
				EvidenceStrength: 0.8,
				HeuristicScore:   0.65,
				MatchedSkills:    []string{"Python", "TensorFlow"},
				Ordinal:          2,
			},
		}

		stories, err := db.SaveRunRankedStories(ctx, runID, input)
		if err != nil {
			t.Fatalf("SaveRunRankedStories failed: %v", err)
		}

		if len(stories) != 2 {
			t.Errorf("Stories count = %d, want 2", len(stories))
		}
		if stories[0].StoryIDText != "amazon-analytics" {
			t.Errorf("First story ID = %q", stories[0].StoryIDText)
		}
		if len(stories[0].MatchedSkills) != 2 {
			t.Errorf("First story matched skills = %d", len(stories[0].MatchedSkills))
		}
	})

	t.Run("get ranked stories", func(t *testing.T) {
		stories, err := db.GetRunRankedStories(ctx, runID)
		if err != nil {
			t.Fatalf("GetRunRankedStories failed: %v", err)
		}

		if len(stories) != 2 {
			t.Errorf("Stories count = %d, want 2", len(stories))
		}
		if stories[0].Ordinal != 1 {
			t.Error("Stories should be ordered by ordinal")
		}
	})

	t.Run("upsert ranked stories", func(t *testing.T) {
		// Save new stories - should replace existing
		input := []RunRankedStoryInput{
			{
				StoryIDText:    "new-story",
				RelevanceScore: 0.95,
				Ordinal:        1,
			},
		}

		stories, err := db.SaveRunRankedStories(ctx, runID, input)
		if err != nil {
			t.Fatalf("SaveRunRankedStories failed: %v", err)
		}

		if len(stories) != 1 {
			t.Errorf("Stories count = %d, want 1 (replaced)", len(stories))
		}
	})
}

// =============================================================================
// Run Resume Plan Integration Tests
// =============================================================================

func TestIntegration_RunResumePlan_CRUD(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	runID := createTestRun(t, db, ctx)
	defer cleanupTestRun(t, db, runID)

	t.Run("save resume plan", func(t *testing.T) {
		input := &RunResumePlanInput{
			MaxBullets:       8,
			MaxLines:         40,
			SkillMatchRatio:  0.7,
			SectionBudgets:   map[string]int{"experience": 30, "projects": 10},
			TopSkillsCovered: []string{"Go", "Python", "AWS"},
			CoverageScore:    0.85,
		}

		plan, err := db.SaveRunResumePlan(ctx, runID, input)
		if err != nil {
			t.Fatalf("SaveRunResumePlan failed: %v", err)
		}

		if plan.ID == uuid.Nil {
			t.Error("Plan ID should not be nil")
		}
		if plan.MaxBullets != 8 {
			t.Errorf("MaxBullets = %d, want 8", plan.MaxBullets)
		}
		if plan.SectionBudgets["experience"] != 30 {
			t.Errorf("SectionBudgets[experience] = %d", plan.SectionBudgets["experience"])
		}
	})

	t.Run("get resume plan", func(t *testing.T) {
		plan, err := db.GetRunResumePlan(ctx, runID)
		if err != nil {
			t.Fatalf("GetRunResumePlan failed: %v", err)
		}

		if plan == nil {
			t.Fatal("Plan not found")
		}
		if plan.MaxBullets != 8 {
			t.Errorf("MaxBullets = %d", plan.MaxBullets)
		}
		if len(plan.TopSkillsCovered) != 3 {
			t.Errorf("TopSkillsCovered = %d", len(plan.TopSkillsCovered))
		}
	})

	t.Run("upsert resume plan", func(t *testing.T) {
		input := &RunResumePlanInput{
			MaxBullets:      10, // Changed
			MaxLines:        50, // Changed
			SkillMatchRatio: 0.8,
			CoverageScore:   0.9,
		}

		plan, err := db.SaveRunResumePlan(ctx, runID, input)
		if err != nil {
			t.Fatalf("SaveRunResumePlan failed: %v", err)
		}

		if plan.MaxBullets != 10 {
			t.Errorf("MaxBullets should be updated to 10, got %d", plan.MaxBullets)
		}
	})

	t.Run("plan not found returns nil", func(t *testing.T) {
		plan, err := db.GetRunResumePlan(ctx, uuid.New())
		if err != nil {
			t.Fatalf("GetRunResumePlan failed: %v", err)
		}
		if plan != nil {
			t.Error("Should return nil for nonexistent run")
		}
	})
}

// =============================================================================
// Run Selected Bullets Integration Tests
// =============================================================================

func TestIntegration_RunSelectedBullets_CRUD(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	runID := createTestRun(t, db, ctx)
	defer cleanupTestRun(t, db, runID)

	// Create a plan first
	planInput := &RunResumePlanInput{
		MaxBullets:    8,
		MaxLines:      40,
		CoverageScore: 0.85,
	}
	plan, _ := db.SaveRunResumePlan(ctx, runID, planInput)

	t.Run("save selected bullets", func(t *testing.T) {
		input := []RunSelectedBulletInput{
			{
				BulletIDText: "bullet_001",
				StoryIDText:  "amazon-analytics",
				Text:         "Built distributed analytics platform",
				Skills:       []string{"Go", "AWS"},
				Metrics:      "1M requests/day",
				LengthChars:  35,
				Section:      SectionExperience,
				Ordinal:      1,
			},
			{
				BulletIDText: "bullet_002",
				StoryIDText:  "amazon-analytics",
				Text:         "Reduced latency by 40%",
				Skills:       []string{"Go", "Performance"},
				LengthChars:  22,
				Section:      SectionExperience,
				Ordinal:      2,
			},
		}

		bullets, err := db.SaveRunSelectedBullets(ctx, runID, &plan.ID, input)
		if err != nil {
			t.Fatalf("SaveRunSelectedBullets failed: %v", err)
		}

		if len(bullets) != 2 {
			t.Errorf("Bullets count = %d, want 2", len(bullets))
		}
		if bullets[0].BulletIDText != "bullet_001" {
			t.Errorf("First bullet ID = %q", bullets[0].BulletIDText)
		}
		if bullets[0].PlanID == nil || *bullets[0].PlanID != plan.ID {
			t.Error("Bullet should be linked to plan")
		}
	})

	t.Run("get selected bullets", func(t *testing.T) {
		bullets, err := db.GetRunSelectedBullets(ctx, runID)
		if err != nil {
			t.Fatalf("GetRunSelectedBullets failed: %v", err)
		}

		if len(bullets) != 2 {
			t.Errorf("Bullets count = %d, want 2", len(bullets))
		}
		if bullets[0].Ordinal != 1 {
			t.Error("Bullets should be ordered by ordinal")
		}
		if len(bullets[0].Skills) != 2 {
			t.Errorf("First bullet skills = %d", len(bullets[0].Skills))
		}
	})

	t.Run("upsert selected bullets", func(t *testing.T) {
		input := []RunSelectedBulletInput{
			{
				BulletIDText: "bullet_new",
				StoryIDText:  "new-story",
				Text:         "New bullet",
				LengthChars:  10,
				Section:      SectionProjects,
				Ordinal:      1,
			},
		}

		bullets, err := db.SaveRunSelectedBullets(ctx, runID, nil, input)
		if err != nil {
			t.Fatalf("SaveRunSelectedBullets failed: %v", err)
		}

		if len(bullets) != 1 {
			t.Errorf("Bullets count = %d, want 1 (replaced)", len(bullets))
		}
	})
}

// =============================================================================
// Run Rewritten Bullets Integration Tests
// =============================================================================

func TestIntegration_RunRewrittenBullets_CRUD(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	runID := createTestRun(t, db, ctx)
	defer cleanupTestRun(t, db, runID)

	t.Run("save rewritten bullets", func(t *testing.T) {
		input := []RunRewrittenBulletInput{
			{
				OriginalBulletIDText: "bullet_001",
				FinalText:            "Architected distributed analytics platform processing 1M daily requests",
				LengthChars:          70,
				EstimatedLines:       1,
				StyleStrongVerb:      true,
				StyleQuantified:      true,
				StyleNoTaboo:         true,
				StyleTargetLength:    true,
				Ordinal:              1,
			},
			{
				OriginalBulletIDText: "bullet_002",
				FinalText:            "Optimized system performance achieving 40% latency reduction",
				LengthChars:          60,
				EstimatedLines:       1,
				StyleStrongVerb:      true,
				StyleQuantified:      true,
				StyleNoTaboo:         true,
				StyleTargetLength:    true,
				Ordinal:              2,
			},
		}

		bullets, err := db.SaveRunRewrittenBullets(ctx, runID, input)
		if err != nil {
			t.Fatalf("SaveRunRewrittenBullets failed: %v", err)
		}

		if len(bullets) != 2 {
			t.Errorf("Bullets count = %d, want 2", len(bullets))
		}
		if !bullets[0].StyleStrongVerb {
			t.Error("StyleStrongVerb should be true")
		}
	})

	t.Run("get rewritten bullets", func(t *testing.T) {
		bullets, err := db.GetRunRewrittenBullets(ctx, runID)
		if err != nil {
			t.Fatalf("GetRunRewrittenBullets failed: %v", err)
		}

		if len(bullets) != 2 {
			t.Errorf("Bullets count = %d, want 2", len(bullets))
		}
		if bullets[0].Ordinal != 1 {
			t.Error("Bullets should be ordered by ordinal")
		}
	})
}

// =============================================================================
// Run Violations Integration Tests
// =============================================================================

func TestIntegration_RunViolations_CRUD(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	runID := createTestRun(t, db, ctx)
	defer cleanupTestRun(t, db, runID)

	t.Run("save violations", func(t *testing.T) {
		lineNum := 42
		charCount := 150
		input := []RunViolationInput{
			{
				ViolationType:    ViolationLineTooLong,
				Severity:         SeverityError,
				Details:          "Line exceeds 120 characters",
				LineNumber:       &lineNum,
				CharCount:        &charCount,
				AffectedSections: []string{"experience"},
			},
			{
				ViolationType:    ViolationPageOverflow,
				Severity:         SeverityWarning,
				Details:          "Resume may exceed one page",
				AffectedSections: []string{"experience", "projects"},
			},
		}

		violations, err := db.SaveRunViolations(ctx, runID, input)
		if err != nil {
			t.Fatalf("SaveRunViolations failed: %v", err)
		}

		if len(violations) != 2 {
			t.Errorf("Violations count = %d, want 2", len(violations))
		}
		if violations[0].ViolationType != ViolationLineTooLong {
			t.Errorf("First violation type = %q", violations[0].ViolationType)
		}
	})

	t.Run("get violations", func(t *testing.T) {
		violations, err := db.GetRunViolations(ctx, runID)
		if err != nil {
			t.Fatalf("GetRunViolations failed: %v", err)
		}

		if len(violations) != 2 {
			t.Errorf("Violations count = %d, want 2", len(violations))
		}
	})

	t.Run("get violations by type", func(t *testing.T) {
		violations, err := db.GetRunViolationsByType(ctx, ViolationLineTooLong)
		if err != nil {
			t.Fatalf("GetRunViolationsByType failed: %v", err)
		}

		if len(violations) == 0 {
			t.Error("Should find at least one line too long violation")
		}
		for _, v := range violations {
			if v.ViolationType != ViolationLineTooLong {
				t.Errorf("Unexpected violation type: %q", v.ViolationType)
			}
		}
	})
}

// =============================================================================
// Query Method Integration Tests
// =============================================================================

func TestIntegration_QueryMethods(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create multiple runs with data
	runID1 := createTestRunWithJobURL(t, db, ctx, "https://example.com/job/123")
	defer cleanupTestRun(t, db, runID1)

	runID2 := createTestRunWithJobURL(t, db, ctx, "https://example.com/job/123")
	defer cleanupTestRun(t, db, runID2)

	// Save selected bullets for both runs
	bullets1 := []RunSelectedBulletInput{
		{BulletIDText: "common_bullet", StoryIDText: "story1", Text: "Test", LengthChars: 4, Ordinal: 1},
	}
	bullets2 := []RunSelectedBulletInput{
		{BulletIDText: "common_bullet", StoryIDText: "story1", Text: "Test", LengthChars: 4, Ordinal: 1},
		{BulletIDText: "unique_bullet", StoryIDText: "story2", Text: "Test2", LengthChars: 5, Ordinal: 2},
	}
	_, _ = db.SaveRunSelectedBullets(ctx, runID1, nil, bullets1)
	_, _ = db.SaveRunSelectedBullets(ctx, runID2, nil, bullets2)

	// Save ranked stories for both runs
	stories1 := []RunRankedStoryInput{
		{StoryIDText: "top_story", RelevanceScore: 0.9, Ordinal: 1},
	}
	stories2 := []RunRankedStoryInput{
		{StoryIDText: "top_story", RelevanceScore: 0.85, Ordinal: 1},
	}
	_, _ = db.SaveRunRankedStories(ctx, runID1, stories1)
	_, _ = db.SaveRunRankedStories(ctx, runID2, stories2)

	t.Run("get most selected bullets", func(t *testing.T) {
		usage, err := db.GetMostSelectedBullets(ctx, 10)
		if err != nil {
			t.Fatalf("GetMostSelectedBullets failed: %v", err)
		}

		if usage["common_bullet"] != 2 {
			t.Errorf("common_bullet selected %d times, want 2", usage["common_bullet"])
		}
		if usage["unique_bullet"] != 1 {
			t.Errorf("unique_bullet selected %d times, want 1", usage["unique_bullet"])
		}
	})

	t.Run("get top ranked stories for job", func(t *testing.T) {
		stories, err := db.GetTopRankedStoriesForJob(ctx, "https://example.com/job/123", 10)
		if err != nil {
			t.Fatalf("GetTopRankedStoriesForJob failed: %v", err)
		}

		if len(stories) == 0 {
			t.Error("Should find ranked stories for job")
		}

		// Should be sorted by relevance_score DESC
		if len(stories) >= 2 {
			if *stories[0].RelevanceScore < *stories[1].RelevanceScore {
				t.Error("Stories should be sorted by relevance_score DESC")
			}
		}
	})
}

// =============================================================================
// Cascade Delete Tests
// =============================================================================

func TestIntegration_CascadeDelete(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	runID := createTestRun(t, db, ctx)

	// Create all artifacts
	_, _ = db.SaveRunRankedStories(ctx, runID, []RunRankedStoryInput{
		{StoryIDText: "test", RelevanceScore: 0.5, Ordinal: 1},
	})
	plan, _ := db.SaveRunResumePlan(ctx, runID, &RunResumePlanInput{MaxBullets: 5, MaxLines: 20})
	_, _ = db.SaveRunSelectedBullets(ctx, runID, &plan.ID, []RunSelectedBulletInput{
		{BulletIDText: "b1", StoryIDText: "s1", Text: "T", LengthChars: 1, Ordinal: 1},
	})
	_, _ = db.SaveRunRewrittenBullets(ctx, runID, []RunRewrittenBulletInput{
		{OriginalBulletIDText: "b1", FinalText: "T", LengthChars: 1, Ordinal: 1},
	})
	_, _ = db.SaveRunViolations(ctx, runID, []RunViolationInput{
		{ViolationType: ViolationPageOverflow, Severity: SeverityWarning},
	})

	// Verify data exists
	stories, _ := db.GetRunRankedStories(ctx, runID)
	if len(stories) == 0 {
		t.Fatal("Stories should exist before delete")
	}

	// Delete the run
	_, err := db.pool.Exec(ctx, "DELETE FROM pipeline_runs WHERE id = $1", runID)
	if err != nil {
		t.Fatalf("Failed to delete run: %v", err)
	}

	// Verify all artifacts are deleted
	stories, _ = db.GetRunRankedStories(ctx, runID)
	if len(stories) != 0 {
		t.Error("Ranked stories should be cascade deleted")
	}

	resumePlan, _ := db.GetRunResumePlan(ctx, runID)
	if resumePlan != nil {
		t.Error("Resume plan should be cascade deleted")
	}

	selectedBullets, _ := db.GetRunSelectedBullets(ctx, runID)
	if len(selectedBullets) != 0 {
		t.Error("Selected bullets should be cascade deleted")
	}

	rewrittenBullets, _ := db.GetRunRewrittenBullets(ctx, runID)
	if len(rewrittenBullets) != 0 {
		t.Error("Rewritten bullets should be cascade deleted")
	}

	violations, _ := db.GetRunViolations(ctx, runID)
	if len(violations) != 0 {
		t.Error("Violations should be cascade deleted")
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func createTestRun(t *testing.T, db *DB, ctx context.Context) uuid.UUID {
	t.Helper()
	runID, err := db.CreateRun(ctx, "Test Company", "Test Role", "https://test.example.com/job/"+uuid.New().String()[:8])
	if err != nil {
		t.Fatalf("Failed to create test run: %v", err)
	}
	return runID
}

func createTestRunWithJobURL(t *testing.T, db *DB, ctx context.Context, jobURL string) uuid.UUID {
	t.Helper()
	runID, err := db.CreateRun(ctx, "Test Company", "Test Role", jobURL)
	if err != nil {
		t.Fatalf("Failed to create test run: %v", err)
	}
	return runID
}

func cleanupTestRun(t *testing.T, db *DB, runID uuid.UUID) {
	t.Helper()
	_, _ = db.pool.Exec(context.Background(), "DELETE FROM pipeline_runs WHERE id = $1", runID)
}
