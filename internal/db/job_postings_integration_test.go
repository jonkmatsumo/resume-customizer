//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// =============================================================================
// Job Posting Integration Tests
// =============================================================================

func TestIntegration_JobPosting_CRUD(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create test company
	company, err := db.FindOrCreateCompany(ctx, "Job Posting Test Corp")
	if err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}
	defer cleanupCompany(t, db, company.ID)

	t.Run("create job posting", func(t *testing.T) {
		input := &JobPostingCreateInput{
			URL:         "https://boards.greenhouse.io/testcorp/jobs/" + uuid.New().String(),
			CompanyID:   &company.ID,
			RoleTitle:   "Senior Software Engineer",
			Platform:    PlatformGreenhouse,
			RawHTML:     "<html><body>Job posting</body></html>",
			CleanedText: "Senior Software Engineer at Test Corp...",
			HTTPStatus:  200,
		}

		posting, err := db.UpsertJobPosting(ctx, input)
		if err != nil {
			t.Fatalf("UpsertJobPosting failed: %v", err)
		}

		if posting.ID == uuid.Nil {
			t.Error("Posting ID should not be nil")
		}
		if posting.FetchStatus != "success" {
			t.Errorf("FetchStatus = %q, want 'success'", posting.FetchStatus)
		}
		if posting.ContentHash == nil || *posting.ContentHash == "" {
			t.Error("ContentHash should be set")
		}
	})

	t.Run("get posting by URL", func(t *testing.T) {
		url := "https://boards.greenhouse.io/testcorp/jobs/" + uuid.New().String()
		input := &JobPostingCreateInput{
			URL:         url,
			CompanyID:   &company.ID,
			RoleTitle:   "Backend Engineer",
			Platform:    PlatformGreenhouse,
			CleanedText: "Backend Engineer...",
			HTTPStatus:  200,
		}
		_, err := db.UpsertJobPosting(ctx, input)
		if err != nil {
			t.Fatalf("UpsertJobPosting failed: %v", err)
		}

		posting, err := db.GetJobPostingByURL(ctx, url)
		if err != nil {
			t.Fatalf("GetJobPostingByURL failed: %v", err)
		}
		if posting == nil {
			t.Fatal("Posting not found")
		}
		if *posting.RoleTitle != "Backend Engineer" {
			t.Errorf("RoleTitle = %q, want 'Backend Engineer'", *posting.RoleTitle)
		}
	})

	t.Run("get fresh posting", func(t *testing.T) {
		url := "https://boards.greenhouse.io/testcorp/jobs/" + uuid.New().String()
		input := &JobPostingCreateInput{
			URL:         url,
			CleanedText: "Fresh posting test",
			HTTPStatus:  200,
		}
		_, err := db.UpsertJobPosting(ctx, input)
		if err != nil {
			t.Fatalf("UpsertJobPosting failed: %v", err)
		}

		fresh, err := db.GetFreshJobPosting(ctx, url)
		if err != nil {
			t.Fatalf("GetFreshJobPosting failed: %v", err)
		}
		if fresh == nil {
			t.Error("Fresh posting should be returned")
		}
	})

	t.Run("expired posting returns nil", func(t *testing.T) {
		url := "https://boards.greenhouse.io/testcorp/jobs/" + uuid.New().String()
		input := &JobPostingCreateInput{
			URL:         url,
			CleanedText: "Expired posting test",
			HTTPStatus:  200,
		}
		_, err := db.UpsertJobPosting(ctx, input)
		if err != nil {
			t.Fatalf("UpsertJobPosting failed: %v", err)
		}

		// Make it expired
		_, err = db.pool.Exec(ctx,
			"UPDATE job_postings SET expires_at = NOW() - INTERVAL '1 hour' WHERE url = $1",
			url)
		if err != nil {
			t.Fatalf("Failed to expire posting: %v", err)
		}

		fresh, err := db.GetFreshJobPosting(ctx, url)
		if err != nil {
			t.Fatalf("GetFreshJobPosting failed: %v", err)
		}
		if fresh != nil {
			t.Error("Expired posting should not be returned")
		}
	})

	t.Run("update existing posting", func(t *testing.T) {
		url := "https://boards.greenhouse.io/testcorp/jobs/" + uuid.New().String()
		input := &JobPostingCreateInput{
			URL:         url,
			RoleTitle:   "Original Title",
			CleanedText: "Original content",
			HTTPStatus:  200,
		}
		original, err := db.UpsertJobPosting(ctx, input)
		if err != nil {
			t.Fatalf("UpsertJobPosting failed: %v", err)
		}

		// Update
		input.RoleTitle = "Updated Title"
		input.CleanedText = "Updated content"
		updated, err := db.UpsertJobPosting(ctx, input)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if updated.ID != original.ID {
			t.Error("Should update same record, not create new")
		}
		if *updated.RoleTitle != "Updated Title" {
			t.Errorf("RoleTitle not updated")
		}
	})

	t.Run("record failed fetch", func(t *testing.T) {
		url := "https://boards.greenhouse.io/testcorp/jobs/notfound-" + uuid.New().String()
		status := 404
		err := db.RecordFailedJobFetch(ctx, url, &status, "Not found")
		if err != nil {
			t.Fatalf("RecordFailedJobFetch failed: %v", err)
		}

		posting, err := db.GetJobPostingByURL(ctx, url)
		if err != nil {
			t.Fatalf("GetJobPostingByURL failed: %v", err)
		}
		if posting == nil {
			t.Fatal("Failed posting should still be recorded")
		}
		if posting.FetchStatus != "error" {
			t.Errorf("FetchStatus = %q, want 'error'", posting.FetchStatus)
		}
	})

	t.Run("list postings by company", func(t *testing.T) {
		postings, err := db.ListJobPostingsByCompany(ctx, company.ID)
		if err != nil {
			t.Fatalf("ListJobPostingsByCompany failed: %v", err)
		}
		if len(postings) == 0 {
			t.Error("Should have at least one posting")
		}
	})

	t.Run("get posting by ID", func(t *testing.T) {
		url := "https://boards.greenhouse.io/testcorp/jobs/" + uuid.New().String()
		input := &JobPostingCreateInput{
			URL:         url,
			RoleTitle:   "Get By ID Test",
			CleanedText: "Test content",
			HTTPStatus:  200,
		}
		created, err := db.UpsertJobPosting(ctx, input)
		if err != nil {
			t.Fatalf("UpsertJobPosting failed: %v", err)
		}

		posting, err := db.GetJobPostingByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetJobPostingByID failed: %v", err)
		}
		if posting == nil {
			t.Fatal("Posting not found")
		}
		if posting.ID != created.ID {
			t.Error("IDs don't match")
		}
	})

	t.Run("get nonexistent posting returns nil", func(t *testing.T) {
		posting, err := db.GetJobPostingByURL(ctx, "https://nonexistent.com/job")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if posting != nil {
			t.Error("Expected nil for nonexistent posting")
		}
	})
}

// =============================================================================
// Job Profile Integration Tests
// =============================================================================

func TestIntegration_JobProfile_CRUD(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create test company and posting
	company, err := db.FindOrCreateCompany(ctx, "Job Profile Test Corp")
	if err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}
	defer cleanupCompany(t, db, company.ID)

	postingInput := &JobPostingCreateInput{
		URL:         "https://boards.greenhouse.io/profiletest/jobs/" + uuid.New().String(),
		CompanyID:   &company.ID,
		RoleTitle:   "Staff Engineer",
		CleanedText: "Staff Engineer position...",
		HTTPStatus:  200,
	}
	posting, err := db.UpsertJobPosting(ctx, postingInput)
	if err != nil {
		t.Fatalf("UpsertJobPosting failed: %v", err)
	}

	t.Run("create job profile with all fields", func(t *testing.T) {
		input := &JobProfileCreateInput{
			PostingID:   posting.ID,
			CompanyName: "Job Profile Test Corp",
			RoleTitle:   "Staff Engineer",
			EvalLatency: true,
			EvalScale:   true,
			EvalSignalsRaw: map[string]interface{}{
				"latency": true,
				"scale":   true,
			},
			EducationMinDegree:       DegreeBachelor,
			EducationPreferredFields: []string{"Computer Science", "Engineering"},
			EducationIsRequired:      false,
			EducationEvidence:        "BS in Computer Science or related field preferred",
			Responsibilities: []string{
				"Design and implement distributed systems",
				"Mentor junior engineers",
				"Lead technical initiatives",
			},
			HardRequirements: []RequirementInput{
				{Skill: "Go", Level: "5+ years", Evidence: "Expert in Go"},
				{Skill: "Distributed Systems", Level: "proficient"},
			},
			NiceToHaves: []RequirementInput{
				{Skill: "Kubernetes"},
				{Skill: "gRPC"},
			},
			Keywords:      []string{"Go", "Distributed Systems", "Kubernetes", "gRPC"},
			ParserVersion: "1.0",
		}

		profile, err := db.CreateJobProfile(ctx, input)
		if err != nil {
			t.Fatalf("CreateJobProfile failed: %v", err)
		}

		if profile.ID == uuid.Nil {
			t.Error("Profile ID should not be nil")
		}
		if len(profile.Responsibilities) != 3 {
			t.Errorf("Responsibilities count = %d, want 3", len(profile.Responsibilities))
		}
		if len(profile.HardRequirements) != 2 {
			t.Errorf("HardRequirements count = %d, want 2", len(profile.HardRequirements))
		}
		if len(profile.NiceToHaves) != 2 {
			t.Errorf("NiceToHaves count = %d, want 2", len(profile.NiceToHaves))
		}
		if len(profile.Keywords) != 4 {
			t.Errorf("Keywords count = %d, want 4", len(profile.Keywords))
		}
	})

	t.Run("get profile by posting ID", func(t *testing.T) {
		profile, err := db.GetJobProfileByPostingID(ctx, posting.ID)
		if err != nil {
			t.Fatalf("GetJobProfileByPostingID failed: %v", err)
		}
		if profile == nil {
			t.Fatal("Profile not found")
		}
		if profile.CompanyName != "Job Profile Test Corp" {
			t.Errorf("CompanyName = %q", profile.CompanyName)
		}
	})

	t.Run("get profile by ID", func(t *testing.T) {
		byPosting, err := db.GetJobProfileByPostingID(ctx, posting.ID)
		if err != nil {
			t.Fatalf("GetJobProfileByPostingID failed: %v", err)
		}

		profile, err := db.GetJobProfileByID(ctx, byPosting.ID)
		if err != nil {
			t.Fatalf("GetJobProfileByID failed: %v", err)
		}
		if profile == nil {
			t.Fatal("Profile not found")
		}
		if profile.ID != byPosting.ID {
			t.Error("IDs don't match")
		}
	})

	t.Run("update existing profile", func(t *testing.T) {
		original, err := db.GetJobProfileByPostingID(ctx, posting.ID)
		if err != nil {
			t.Fatalf("GetJobProfileByPostingID failed: %v", err)
		}

		input := &JobProfileCreateInput{
			PostingID:   posting.ID,
			CompanyName: "Job Profile Test Corp",
			RoleTitle:   "Staff Engineer (Updated)",
			EvalLatency: false,
			Responsibilities: []string{
				"New responsibility 1",
				"New responsibility 2",
			},
		}

		updated, err := db.CreateJobProfile(ctx, input)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if updated.ID != original.ID {
			t.Error("Should update same record")
		}
		if updated.RoleTitle != "Staff Engineer (Updated)" {
			t.Error("RoleTitle not updated")
		}
		if len(updated.Responsibilities) != 2 {
			t.Errorf("Responsibilities should be replaced, got %d", len(updated.Responsibilities))
		}
	})

	t.Run("get nonexistent profile returns nil", func(t *testing.T) {
		profile, err := db.GetJobProfileByPostingID(ctx, uuid.New())
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if profile != nil {
			t.Error("Expected nil for nonexistent profile")
		}
	})
}

// =============================================================================
// Query Integration Tests
// =============================================================================

func TestIntegration_JobQueries(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Setup: Create company, posting, and profile with specific skills
	company, err := db.FindOrCreateCompany(ctx, "Query Test Corp")
	if err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}
	defer cleanupCompany(t, db, company.ID)

	postingInput := &JobPostingCreateInput{
		URL:         "https://boards.greenhouse.io/querytest/jobs/" + uuid.New().String(),
		CompanyID:   &company.ID,
		RoleTitle:   "Python Developer",
		CleanedText: "Python Developer position...",
		HTTPStatus:  200,
	}
	posting, err := db.UpsertJobPosting(ctx, postingInput)
	if err != nil {
		t.Fatalf("UpsertJobPosting failed: %v", err)
	}

	profileInput := &JobProfileCreateInput{
		PostingID:   posting.ID,
		CompanyName: "Query Test Corp",
		RoleTitle:   "Python Developer",
		HardRequirements: []RequirementInput{
			{Skill: "Python", Level: "5+ years"},
			{Skill: "Django", Level: "3+ years"},
		},
		NiceToHaves: []RequirementInput{
			{Skill: "FastAPI"},
		},
		Keywords: []string{"Python", "Django", "FastAPI", "REST API"},
	}
	_, err = db.CreateJobProfile(ctx, profileInput)
	if err != nil {
		t.Fatalf("CreateJobProfile failed: %v", err)
	}

	t.Run("find jobs by skill", func(t *testing.T) {
		profiles, err := db.FindJobsBySkill(ctx, "Python", RequirementTypeHard)
		if err != nil {
			t.Fatalf("FindJobsBySkill failed: %v", err)
		}
		if len(profiles) == 0 {
			t.Error("Should find at least one job")
		}

		found := false
		for _, p := range profiles {
			if p.RoleTitle == "Python Developer" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Should find Python Developer job")
		}
	})

	t.Run("find jobs by skill - nice to have", func(t *testing.T) {
		profiles, err := db.FindJobsBySkill(ctx, "FastAPI", RequirementTypeNiceToHave)
		if err != nil {
			t.Fatalf("FindJobsBySkill failed: %v", err)
		}
		if len(profiles) == 0 {
			t.Error("Should find job with FastAPI as nice-to-have")
		}
	})

	t.Run("find jobs by skill - any type", func(t *testing.T) {
		profiles, err := db.FindJobsBySkill(ctx, "Python", "")
		if err != nil {
			t.Fatalf("FindJobsBySkill failed: %v", err)
		}
		if len(profiles) == 0 {
			t.Error("Should find job")
		}
	})

	t.Run("find jobs by keyword", func(t *testing.T) {
		profiles, err := db.FindJobsByKeyword(ctx, "REST API")
		if err != nil {
			t.Fatalf("FindJobsByKeyword failed: %v", err)
		}
		if len(profiles) == 0 {
			t.Error("Should find job with REST API keyword")
		}
	})

	t.Run("find jobs by keyword - case insensitive", func(t *testing.T) {
		profiles, err := db.FindJobsByKeyword(ctx, "rest api")
		if err != nil {
			t.Fatalf("FindJobsByKeyword failed: %v", err)
		}
		if len(profiles) == 0 {
			t.Error("Should find job (case insensitive)")
		}
	})

	t.Run("find jobs by nonexistent skill", func(t *testing.T) {
		profiles, err := db.FindJobsBySkill(ctx, "NonexistentSkill123", RequirementTypeHard)
		if err != nil {
			t.Fatalf("FindJobsBySkill failed: %v", err)
		}
		if len(profiles) != 0 {
			t.Error("Should not find any jobs")
		}
	})

	t.Run("find jobs by nonexistent keyword", func(t *testing.T) {
		profiles, err := db.FindJobsByKeyword(ctx, "nonexistentkeyword123")
		if err != nil {
			t.Fatalf("FindJobsByKeyword failed: %v", err)
		}
		if len(profiles) != 0 {
			t.Error("Should not find any jobs")
		}
	})
}

// =============================================================================
// Cascade Delete Tests
// =============================================================================

func TestIntegration_JobProfile_CascadeDelete(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, err := db.FindOrCreateCompany(ctx, "Cascade Delete Test Corp")
	if err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}
	defer cleanupCompany(t, db, company.ID)

	postingInput := &JobPostingCreateInput{
		URL:         "https://boards.greenhouse.io/cascadetest/jobs/" + uuid.New().String(),
		CompanyID:   &company.ID,
		CleanedText: "Test",
		HTTPStatus:  200,
	}
	posting, err := db.UpsertJobPosting(ctx, postingInput)
	if err != nil {
		t.Fatalf("UpsertJobPosting failed: %v", err)
	}

	profileInput := &JobProfileCreateInput{
		PostingID:        posting.ID,
		CompanyName:      "Test",
		RoleTitle:        "Test",
		Responsibilities: []string{"Resp 1", "Resp 2"},
		HardRequirements: []RequirementInput{{Skill: "Go"}},
		Keywords:         []string{"Go", "Test"},
	}
	profile, err := db.CreateJobProfile(ctx, profileInput)
	if err != nil {
		t.Fatalf("CreateJobProfile failed: %v", err)
	}

	// Delete profile
	err = db.DeleteJobProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("DeleteJobProfile failed: %v", err)
	}

	// Verify cascaded deletions
	var respCount, reqCount, kwCount int
	err = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM job_responsibilities WHERE job_profile_id = $1", profile.ID).Scan(&respCount)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	err = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM job_requirements WHERE job_profile_id = $1", profile.ID).Scan(&reqCount)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	err = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM job_keywords WHERE job_profile_id = $1", profile.ID).Scan(&kwCount)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if respCount != 0 {
		t.Errorf("Responsibilities should be deleted, found %d", respCount)
	}
	if reqCount != 0 {
		t.Errorf("Requirements should be deleted, found %d", reqCount)
	}
	if kwCount != 0 {
		t.Errorf("Keywords should be deleted, found %d", kwCount)
	}
}

func TestIntegration_JobPosting_CascadeToProfile(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, err := db.FindOrCreateCompany(ctx, "Posting Cascade Test Corp")
	if err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}
	defer cleanupCompany(t, db, company.ID)

	postingInput := &JobPostingCreateInput{
		URL:         "https://boards.greenhouse.io/postingcascade/jobs/" + uuid.New().String(),
		CompanyID:   &company.ID,
		CleanedText: "Test",
		HTTPStatus:  200,
	}
	posting, err := db.UpsertJobPosting(ctx, postingInput)
	if err != nil {
		t.Fatalf("UpsertJobPosting failed: %v", err)
	}

	profileInput := &JobProfileCreateInput{
		PostingID:        posting.ID,
		CompanyName:      "Test",
		RoleTitle:        "Test",
		Responsibilities: []string{"Resp 1"},
		HardRequirements: []RequirementInput{{Skill: "Go"}},
		Keywords:         []string{"Go"},
	}
	profile, err := db.CreateJobProfile(ctx, profileInput)
	if err != nil {
		t.Fatalf("CreateJobProfile failed: %v", err)
	}

	// Delete posting (should cascade to profile)
	_, err = db.pool.Exec(ctx, "DELETE FROM job_postings WHERE id = $1", posting.ID)
	if err != nil {
		t.Fatalf("Delete posting failed: %v", err)
	}

	// Verify profile is deleted
	deletedProfile, err := db.GetJobProfileByID(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetJobProfileByID failed: %v", err)
	}
	if deletedProfile != nil {
		t.Error("Profile should be cascade deleted when posting is deleted")
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestIntegration_JobProfile_EmptyArrays(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, err := db.FindOrCreateCompany(ctx, "Empty Arrays Test Corp")
	if err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}
	defer cleanupCompany(t, db, company.ID)

	postingInput := &JobPostingCreateInput{
		URL:         "https://boards.greenhouse.io/emptytest/jobs/" + uuid.New().String(),
		CompanyID:   &company.ID,
		CleanedText: "Test",
		HTTPStatus:  200,
	}
	posting, err := db.UpsertJobPosting(ctx, postingInput)
	if err != nil {
		t.Fatalf("UpsertJobPosting failed: %v", err)
	}

	input := &JobProfileCreateInput{
		PostingID:        posting.ID,
		CompanyName:      "Empty Arrays Test Corp",
		RoleTitle:        "Minimal Role",
		Responsibilities: []string{},
		HardRequirements: []RequirementInput{},
		NiceToHaves:      []RequirementInput{},
		Keywords:         []string{},
	}

	profile, err := db.CreateJobProfile(ctx, input)
	if err != nil {
		t.Fatalf("CreateJobProfile failed: %v", err)
	}

	if len(profile.Responsibilities) != 0 {
		t.Error("Responsibilities should be empty")
	}
	if len(profile.HardRequirements) != 0 {
		t.Error("HardRequirements should be empty")
	}
	if len(profile.NiceToHaves) != 0 {
		t.Error("NiceToHaves should be empty")
	}
	if len(profile.Keywords) != 0 {
		t.Error("Keywords should be empty")
	}
}

func TestIntegration_JobPosting_WithAdminInfo(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, err := db.FindOrCreateCompany(ctx, "Admin Info Test Corp")
	if err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}
	defer cleanupCompany(t, db, company.ID)

	salary := "$150,000 - $200,000"
	location := "San Francisco, CA"
	remote := "hybrid"

	input := &JobPostingCreateInput{
		URL:         "https://boards.greenhouse.io/admininfo/jobs/" + uuid.New().String(),
		CompanyID:   &company.ID,
		RoleTitle:   "Engineer with Admin Info",
		CleanedText: "Test content",
		HTTPStatus:  200,
		AdminInfo: &AdminInfo{
			Salary:       &salary,
			Location:     &location,
			RemotePolicy: &remote,
		},
		Links: []string{"https://example.com/link1", "https://example.com/link2"},
	}

	posting, err := db.UpsertJobPosting(ctx, input)
	if err != nil {
		t.Fatalf("UpsertJobPosting failed: %v", err)
	}

	// Retrieve and verify
	retrieved, err := db.GetJobPostingByURL(ctx, input.URL)
	if err != nil {
		t.Fatalf("GetJobPostingByURL failed: %v", err)
	}

	if retrieved.ID != posting.ID {
		t.Error("IDs don't match")
	}
	if retrieved.AdminInfo == nil {
		t.Fatal("AdminInfo should not be nil")
	}
	if retrieved.AdminInfo.Salary == nil || *retrieved.AdminInfo.Salary != salary {
		t.Error("Salary not preserved")
	}
	if len(retrieved.ExtractedLinks) != 2 {
		t.Errorf("ExtractedLinks count = %d, want 2", len(retrieved.ExtractedLinks))
	}
}
