//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// These tests require a running PostgreSQL database.
// Set TEST_DATABASE_URL environment variable to run them.
// Example: TEST_DATABASE_URL=postgres://user:pass@localhost:5432/resume_customizer_test

func getExperienceBankTestDB(t *testing.T) *DB {
	t.Helper()

	db := getTestDB(t) // Use existing helper

	// Clean up test data before each test
	ctx := context.Background()
	_, _ = db.pool.Exec(ctx, "DELETE FROM bullet_skills")
	_, _ = db.pool.Exec(ctx, "DELETE FROM bullets")
	_, _ = db.pool.Exec(ctx, "DELETE FROM stories WHERE story_id LIKE 'test-%'")
	_, _ = db.pool.Exec(ctx, "DELETE FROM skills WHERE name_normalized LIKE 'test%'")
	_, _ = db.pool.Exec(ctx, "DELETE FROM education_highlights")

	return db
}

// =============================================================================
// Skill Integration Tests
// =============================================================================

func TestIntegration_Skill_CRUD(t *testing.T) {
	db := getExperienceBankTestDB(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("find or create skill", func(t *testing.T) {
		skill, err := db.FindOrCreateSkill(ctx, "Go")
		if err != nil {
			t.Fatalf("FindOrCreateSkill failed: %v", err)
		}

		if skill.ID == uuid.Nil {
			t.Error("Skill ID should not be nil")
		}
		if skill.NameNormalized != "go" {
			t.Errorf("NameNormalized = %q, want 'go'", skill.NameNormalized)
		}
		if skill.Category == nil || *skill.Category != SkillCategoryProgramming {
			t.Errorf("Category should be 'programming'")
		}
	})

	t.Run("skill deduplication", func(t *testing.T) {
		skill1, _ := db.FindOrCreateSkill(ctx, "Python")
		skill2, _ := db.FindOrCreateSkill(ctx, "python")
		skill3, _ := db.FindOrCreateSkill(ctx, "PYTHON")

		if skill1.ID != skill2.ID || skill2.ID != skill3.ID {
			t.Error("Same skill (different case) should return same ID")
		}
	})

	t.Run("skill synonyms", func(t *testing.T) {
		golang, _ := db.FindOrCreateSkill(ctx, "Golang")
		goSkill, _ := db.FindOrCreateSkill(ctx, "Go")

		if golang.ID != goSkill.ID {
			t.Error("Golang and Go should be the same skill")
		}
	})

	t.Run("get skill by name", func(t *testing.T) {
		_, _ = db.FindOrCreateSkill(ctx, "Rust")
		skill, err := db.GetSkillByName(ctx, "rust")
		if err != nil {
			t.Fatalf("GetSkillByName failed: %v", err)
		}
		if skill == nil {
			t.Fatal("Skill not found")
		}
		if skill.NameNormalized != "rust" {
			t.Errorf("NameNormalized = %q", skill.NameNormalized)
		}
	})

	t.Run("get skill by ID", func(t *testing.T) {
		created, _ := db.FindOrCreateSkill(ctx, "Scala")
		skill, err := db.GetSkillByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetSkillByID failed: %v", err)
		}
		if skill == nil {
			t.Fatal("Skill not found")
		}
		if skill.Name != "Scala" {
			t.Errorf("Name = %q, want 'Scala'", skill.Name)
		}
	})

	t.Run("list skills", func(t *testing.T) {
		skills, err := db.ListSkills(ctx, "")
		if err != nil {
			t.Fatalf("ListSkills failed: %v", err)
		}
		if len(skills) == 0 {
			t.Error("Should have at least one skill")
		}
	})

	t.Run("list skills by category", func(t *testing.T) {
		_, _ = db.FindOrCreateSkill(ctx, "Java")
		skills, err := db.ListSkills(ctx, SkillCategoryProgramming)
		if err != nil {
			t.Fatalf("ListSkills failed: %v", err)
		}
		if len(skills) == 0 {
			t.Error("Should have programming skills")
		}
		for _, s := range skills {
			if s.Category == nil || *s.Category != SkillCategoryProgramming {
				t.Errorf("Expected programming category, got %v", s.Category)
			}
		}
	})

	t.Run("skill not found returns nil", func(t *testing.T) {
		skill, err := db.GetSkillByName(ctx, "nonexistent-skill-xyz")
		if err != nil {
			t.Fatalf("GetSkillByName failed: %v", err)
		}
		if skill != nil {
			t.Error("Should return nil for nonexistent skill")
		}
	})
}

// =============================================================================
// Story Integration Tests
// =============================================================================

func TestIntegration_Story_CRUD(t *testing.T) {
	db := getExperienceBankTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create test user and job
	user := createTestUserForExperience(t, db, ctx)
	defer cleanupTestUser(t, db, user.ID)

	job := createTestJobForExperience(t, db, ctx, user.ID)

	t.Run("create story with bullets", func(t *testing.T) {
		input := &StoryCreateInput{
			StoryID: "test-story-" + uuid.New().String()[:8],
			UserID:  user.ID,
			JobID:   job.ID,
			Title:   "Analytics Platform",
			Bullets: []BulletCreateInput{
				{
					BulletID:         "test-bullet-001",
					Text:             "Built distributed analytics system processing 1M requests/day",
					Metrics:          "1M requests/day",
					EvidenceStrength: EvidenceStrengthHigh,
					Skills:           []string{"Go", "Distributed Systems", "PostgreSQL"},
				},
				{
					BulletID:         "test-bullet-002",
					Text:             "Reduced latency by 40%",
					EvidenceStrength: EvidenceStrengthHigh,
					Skills:           []string{"Go", "Performance"},
				},
			},
		}

		story, err := db.CreateStory(ctx, input)
		if err != nil {
			t.Fatalf("CreateStory failed: %v", err)
		}

		if story.ID == uuid.Nil {
			t.Error("Story ID should not be nil")
		}
		if len(story.Bullets) != 2 {
			t.Errorf("Bullets count = %d, want 2", len(story.Bullets))
		}
		if len(story.Bullets[0].Skills) != 3 {
			t.Errorf("First bullet skills count = %d, want 3", len(story.Bullets[0].Skills))
		}
	})

	t.Run("get story by story_id", func(t *testing.T) {
		storyID := "test-get-story-" + uuid.New().String()[:8]
		input := &StoryCreateInput{
			StoryID: storyID,
			UserID:  user.ID,
			JobID:   job.ID,
			Bullets: []BulletCreateInput{
				{
					BulletID:         "test-bullet-get",
					Text:             "Test bullet",
					EvidenceStrength: EvidenceStrengthMedium,
					Skills:           []string{"Go"},
				},
			},
		}
		_, _ = db.CreateStory(ctx, input)

		story, err := db.GetStoryByStoryID(ctx, storyID)
		if err != nil {
			t.Fatalf("GetStoryByStoryID failed: %v", err)
		}
		if story == nil {
			t.Fatal("Story not found")
		}
		if story.StoryID != storyID {
			t.Errorf("StoryID = %q, want %q", story.StoryID, storyID)
		}
		if len(story.Bullets) != 1 {
			t.Errorf("Bullets count = %d, want 1", len(story.Bullets))
		}
	})

	t.Run("get story by ID", func(t *testing.T) {
		storyID := "test-get-by-id-" + uuid.New().String()[:8]
		input := &StoryCreateInput{
			StoryID: storyID,
			UserID:  user.ID,
			JobID:   job.ID,
			Bullets: []BulletCreateInput{
				{
					BulletID:         "test-bullet-byid",
					Text:             "Test bullet",
					EvidenceStrength: EvidenceStrengthMedium,
					Skills:           []string{"Python"},
				},
			},
		}
		created, _ := db.CreateStory(ctx, input)

		story, err := db.GetStoryByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetStoryByID failed: %v", err)
		}
		if story == nil {
			t.Fatal("Story not found")
		}
		if story.ID != created.ID {
			t.Errorf("ID mismatch")
		}
	})

	t.Run("list stories by user", func(t *testing.T) {
		stories, err := db.ListStoriesByUser(ctx, user.ID)
		if err != nil {
			t.Fatalf("ListStoriesByUser failed: %v", err)
		}
		if len(stories) == 0 {
			t.Error("Should have at least one story")
		}
	})

	t.Run("list stories by job", func(t *testing.T) {
		stories, err := db.ListStoriesByJob(ctx, job.ID)
		if err != nil {
			t.Fatalf("ListStoriesByJob failed: %v", err)
		}
		if len(stories) == 0 {
			t.Error("Should have at least one story")
		}
	})

	t.Run("update story (upsert)", func(t *testing.T) {
		storyID := "test-upsert-" + uuid.New().String()[:8]
		input := &StoryCreateInput{
			StoryID: storyID,
			UserID:  user.ID,
			JobID:   job.ID,
			Title:   "Original Title",
			Bullets: []BulletCreateInput{
				{
					BulletID:         "test-bullet-original",
					Text:             "Original bullet",
					EvidenceStrength: EvidenceStrengthMedium,
					Skills:           []string{"Go"},
				},
			},
		}
		original, _ := db.CreateStory(ctx, input)

		// Update
		input.Title = "Updated Title"
		input.Bullets = []BulletCreateInput{
			{
				BulletID:         "test-bullet-new",
				Text:             "New bullet",
				EvidenceStrength: EvidenceStrengthHigh,
				Skills:           []string{"Python"},
			},
		}
		updated, err := db.CreateStory(ctx, input)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if updated.ID != original.ID {
			t.Error("Should update same story")
		}
		if *updated.Title != "Updated Title" {
			t.Error("Title not updated")
		}
		if len(updated.Bullets) != 1 {
			t.Errorf("Bullets should be replaced, got %d", len(updated.Bullets))
		}
		if updated.Bullets[0].BulletID != "test-bullet-new" {
			t.Error("Bullet not replaced")
		}
	})

	t.Run("delete story cascades to bullets", func(t *testing.T) {
		storyID := "test-delete-" + uuid.New().String()[:8]
		input := &StoryCreateInput{
			StoryID: storyID,
			UserID:  user.ID,
			JobID:   job.ID,
			Bullets: []BulletCreateInput{
				{
					BulletID:         "test-bullet-delete",
					Text:             "To be deleted",
					EvidenceStrength: EvidenceStrengthMedium,
					Skills:           []string{"Go"},
				},
			},
		}
		story, _ := db.CreateStory(ctx, input)
		bulletID := story.Bullets[0].ID

		err := db.DeleteStory(ctx, story.ID)
		if err != nil {
			t.Fatalf("DeleteStory failed: %v", err)
		}

		// Verify story is deleted
		deletedStory, _ := db.GetStoryByID(ctx, story.ID)
		if deletedStory != nil {
			t.Error("Story should be deleted")
		}

		// Verify bullet is deleted
		bullet, _ := db.GetBulletByID(ctx, bulletID)
		if bullet != nil {
			t.Error("Bullet should be cascade deleted")
		}
	})

	t.Run("story not found returns nil", func(t *testing.T) {
		story, err := db.GetStoryByStoryID(ctx, "nonexistent-story-xyz")
		if err != nil {
			t.Fatalf("GetStoryByStoryID failed: %v", err)
		}
		if story != nil {
			t.Error("Should return nil for nonexistent story")
		}
	})
}

// =============================================================================
// Bullet Query Tests
// =============================================================================

func TestIntegration_BulletQueries(t *testing.T) {
	db := getExperienceBankTestDB(t)
	defer db.Close()
	ctx := context.Background()

	user := createTestUserForExperience(t, db, ctx)
	defer cleanupTestUser(t, db, user.ID)

	job := createTestJobForExperience(t, db, ctx, user.ID)

	// Create story with specific skills for testing
	input := &StoryCreateInput{
		StoryID: "test-query-" + uuid.New().String()[:8],
		UserID:  user.ID,
		JobID:   job.ID,
		Bullets: []BulletCreateInput{
			{
				BulletID:         "test-query-bullet-1",
				Text:             "Built Python data pipeline",
				EvidenceStrength: EvidenceStrengthHigh,
				Skills:           []string{"Python", "Data Engineering"},
			},
			{
				BulletID:         "test-query-bullet-2",
				Text:             "Developed Go microservices",
				EvidenceStrength: EvidenceStrengthMedium,
				Skills:           []string{"Go", "Microservices"},
			},
		},
	}
	_, _ = db.CreateStory(ctx, input)

	t.Run("find bullets by skill", func(t *testing.T) {
		bullets, err := db.FindBulletsBySkill(ctx, "Python")
		if err != nil {
			t.Fatalf("FindBulletsBySkill failed: %v", err)
		}
		if len(bullets) == 0 {
			t.Error("Should find bullets with Python skill")
		}

		// Verify the bullet has Python skill
		found := false
		for _, b := range bullets {
			for _, s := range b.Skills {
				if s == "Python" {
					found = true
					break
				}
			}
		}
		if !found {
			t.Error("Found bullet should have Python skill")
		}
	})

	t.Run("find bullets by skill - case insensitive", func(t *testing.T) {
		bullets, err := db.FindBulletsBySkill(ctx, "python")
		if err != nil {
			t.Fatalf("FindBulletsBySkill failed: %v", err)
		}
		if len(bullets) == 0 {
			t.Error("Should find bullets (case insensitive)")
		}
	})

	t.Run("find bullets by evidence strength", func(t *testing.T) {
		bullets, err := db.FindBulletsByEvidenceStrength(ctx, EvidenceStrengthHigh)
		if err != nil {
			t.Fatalf("FindBulletsByEvidenceStrength failed: %v", err)
		}
		if len(bullets) == 0 {
			t.Error("Should find high-evidence bullets")
		}
		for _, b := range bullets {
			if b.EvidenceStrength != EvidenceStrengthHigh {
				t.Errorf("Expected high evidence, got %q", b.EvidenceStrength)
			}
		}
	})

	t.Run("get skill usage count", func(t *testing.T) {
		usage, err := db.GetSkillUsageCount(ctx)
		if err != nil {
			t.Fatalf("GetSkillUsageCount failed: %v", err)
		}
		if len(usage) == 0 {
			t.Error("Should have skill usage data")
		}

		// Python should have at least 1 usage
		if usage["Python"] < 1 {
			t.Errorf("Python usage = %d, want >= 1", usage["Python"])
		}
	})

	t.Run("get bullet by bullet_id", func(t *testing.T) {
		bullet, err := db.GetBulletByBulletID(ctx, "test-query-bullet-1")
		if err != nil {
			t.Fatalf("GetBulletByBulletID failed: %v", err)
		}
		if bullet == nil {
			t.Fatal("Bullet not found")
		}
		if bullet.BulletID != "test-query-bullet-1" {
			t.Errorf("BulletID = %q", bullet.BulletID)
		}
	})

	t.Run("get bullet skills", func(t *testing.T) {
		bullet, _ := db.GetBulletByBulletID(ctx, "test-query-bullet-1")
		skills, err := db.GetBulletSkills(ctx, bullet.ID)
		if err != nil {
			t.Fatalf("GetBulletSkills failed: %v", err)
		}
		if len(skills) != 2 {
			t.Errorf("Skills count = %d, want 2", len(skills))
		}
	})
}

// =============================================================================
// Education Highlight Tests
// =============================================================================

func TestIntegration_EducationHighlights(t *testing.T) {
	db := getExperienceBankTestDB(t)
	defer db.Close()
	ctx := context.Background()

	user := createTestUserForExperience(t, db, ctx)
	defer cleanupTestUser(t, db, user.ID)

	edu := createTestEducationForExperience(t, db, ctx, user.ID)

	t.Run("add education highlights", func(t *testing.T) {
		h1, err := db.AddEducationHighlight(ctx, edu.ID, "Dean's List", 1)
		if err != nil {
			t.Fatalf("AddEducationHighlight failed: %v", err)
		}

		h2, err := db.AddEducationHighlight(ctx, edu.ID, "Research Assistant", 2)
		if err != nil {
			t.Fatalf("AddEducationHighlight failed: %v", err)
		}

		if h1.ID == uuid.Nil || h2.ID == uuid.Nil {
			t.Error("Highlight IDs should not be nil")
		}
		if h1.Text != "Dean's List" {
			t.Errorf("First highlight text = %q", h1.Text)
		}
		if h1.Ordinal != 1 {
			t.Errorf("First highlight ordinal = %d, want 1", h1.Ordinal)
		}
	})

	t.Run("get education highlights", func(t *testing.T) {
		highlights, err := db.GetEducationHighlights(ctx, edu.ID)
		if err != nil {
			t.Fatalf("GetEducationHighlights failed: %v", err)
		}
		if len(highlights) != 2 {
			t.Errorf("Highlights count = %d, want 2", len(highlights))
		}
		if highlights[0].Ordinal != 1 {
			t.Error("First highlight should have ordinal 1")
		}
		if highlights[1].Ordinal != 2 {
			t.Error("Second highlight should have ordinal 2")
		}
	})

	t.Run("delete education highlights", func(t *testing.T) {
		err := db.DeleteEducationHighlights(ctx, edu.ID)
		if err != nil {
			t.Fatalf("DeleteEducationHighlights failed: %v", err)
		}

		highlights, _ := db.GetEducationHighlights(ctx, edu.ID)
		if len(highlights) != 0 {
			t.Errorf("Highlights should be deleted, got %d", len(highlights))
		}
	})
}

// =============================================================================
// Import Tests
// =============================================================================

func TestIntegration_ImportExperienceBank(t *testing.T) {
	db := getExperienceBankTestDB(t)
	defer db.Close()
	ctx := context.Background()

	user := createTestUserForExperience(t, db, ctx)
	defer cleanupTestUser(t, db, user.ID)

	input := &ExperienceBankImportInput{
		UserID: user.ID,
		Stories: []StoryImportInput{
			{
				ID:        "test-import-story-001",
				Company:   "Test Company",
				Role:      "Software Engineer",
				StartDate: "2020-01",
				EndDate:   "2023-06",
				Bullets: []BulletImportInput{
					{
						ID:               "test-import-bullet-001",
						Text:             "Built distributed system",
						Skills:           []string{"Go", "Distributed Systems"},
						LengthChars:      25,
						EvidenceStrength: "high",
						RiskFlags:        []string{},
					},
					{
						ID:               "test-import-bullet-002",
						Text:             "Improved latency by 40%",
						Skills:           []string{"Go", "Performance"},
						Metrics:          "40% improvement",
						LengthChars:      24,
						EvidenceStrength: "high",
						RiskFlags:        []string{},
					},
				},
			},
		},
		Education: []EducationImportInput{
			{
				ID:         "test-import-edu-001",
				School:     "Test University",
				Degree:     "bachelor",
				Field:      "Computer Science",
				Highlights: []string{"Magna Cum Laude", "Dean's List"},
			},
		},
	}

	t.Run("import experience bank", func(t *testing.T) {
		err := db.ImportExperienceBank(ctx, input)
		if err != nil {
			t.Fatalf("ImportExperienceBank failed: %v", err)
		}
	})

	t.Run("verify story was created", func(t *testing.T) {
		story, err := db.GetStoryByStoryID(ctx, "test-import-story-001")
		if err != nil {
			t.Fatalf("GetStoryByStoryID failed: %v", err)
		}
		if story == nil {
			t.Fatal("Story not imported")
		}
		if len(story.Bullets) != 2 {
			t.Errorf("Bullets count = %d, want 2", len(story.Bullets))
		}
	})

	t.Run("verify skills were created", func(t *testing.T) {
		skills, _ := db.ListSkills(ctx, "")
		foundGo := false
		for _, s := range skills {
			if s.NameNormalized == "go" {
				foundGo = true
				break
			}
		}
		if !foundGo {
			t.Error("Go skill should have been created")
		}
	})

	t.Run("verify education highlights were created", func(t *testing.T) {
		// Find the education by querying
		var eduID uuid.UUID
		err := db.pool.QueryRow(ctx,
			`SELECT id FROM education WHERE user_id = $1 AND school = $2`,
			user.ID, "Test University",
		).Scan(&eduID)
		if err != nil {
			t.Fatalf("Failed to find education: %v", err)
		}

		highlights, err := db.GetEducationHighlights(ctx, eduID)
		if err != nil {
			t.Fatalf("GetEducationHighlights failed: %v", err)
		}
		if len(highlights) != 2 {
			t.Errorf("Highlights count = %d, want 2", len(highlights))
		}
	})

	t.Run("re-import updates existing data", func(t *testing.T) {
		// Modify input and re-import
		input.Stories[0].Bullets = []BulletImportInput{
			{
				ID:               "test-import-bullet-new",
				Text:             "New bullet after re-import",
				Skills:           []string{"Python"},
				LengthChars:      30,
				EvidenceStrength: "medium",
				RiskFlags:        []string{},
			},
		}

		err := db.ImportExperienceBank(ctx, input)
		if err != nil {
			t.Fatalf("Re-import failed: %v", err)
		}

		story, _ := db.GetStoryByStoryID(ctx, "test-import-story-001")
		if story == nil {
			t.Fatal("Story not found after re-import")
		}
		if len(story.Bullets) != 1 {
			t.Errorf("Bullets should be replaced, got %d", len(story.Bullets))
		}
		if story.Bullets[0].BulletID != "test-import-bullet-new" {
			t.Error("Bullet should be the new one")
		}
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func createTestUserForExperience(t *testing.T, db *DB, ctx context.Context) *User {
	t.Helper()
	var user User
	err := db.pool.QueryRow(ctx,
		`INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at`,
		"Test User", "exp-test-"+uuid.New().String()[:8]+"@example.com",
	).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return &user
}

func cleanupTestUser(t *testing.T, db *DB, userID uuid.UUID) {
	t.Helper()
	_, _ = db.pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
}

func createTestJobForExperience(t *testing.T, db *DB, ctx context.Context, userID uuid.UUID) *Job {
	t.Helper()
	var job Job
	err := db.pool.QueryRow(ctx,
		`INSERT INTO jobs (user_id, company, role_title) VALUES ($1, $2, $3) 
		 RETURNING id, user_id, company, role_title, created_at`,
		userID, "Test Company", "Test Role",
	).Scan(&job.ID, &job.UserID, &job.Company, &job.RoleTitle, &job.CreatedAt)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}
	return &job
}

func createTestEducationForExperience(t *testing.T, db *DB, ctx context.Context, userID uuid.UUID) *Education {
	t.Helper()
	var edu Education
	err := db.pool.QueryRow(ctx,
		`INSERT INTO education (user_id, school, degree_type, field) VALUES ($1, $2, $3, $4) 
		 RETURNING id, user_id, school, degree_type, field, created_at`,
		userID, "Test University", "bachelor", "Computer Science",
	).Scan(&edu.ID, &edu.UserID, &edu.School, &edu.DegreeType, &edu.Field, &edu.CreatedAt)
	if err != nil {
		t.Fatalf("Failed to create test education: %v", err)
	}
	return &edu
}
