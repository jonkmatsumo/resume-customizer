//go:build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// strPtr returns a pointer to the string
func strPtr(s string) *string {
	return &s
}

// cleanupCompany removes a company and all its related data
func cleanupCompany(t *testing.T, db *DB, companyID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	_, _ = db.pool.Exec(ctx, "DELETE FROM companies WHERE id = $1", companyID)
}

// =============================================================================
// Company Profile Integration Tests
// =============================================================================

func TestIntegration_CreateCompanyProfile(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create test company first
	company, err := db.FindOrCreateCompany(ctx, "Profile Test Corp")
	if err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}
	defer cleanupCompany(t, db, company.ID)

	t.Run("creates profile with all fields", func(t *testing.T) {
		input := &ProfileCreateInput{
			CompanyID:     company.ID,
			Tone:          "direct and technical",
			DomainContext: "FinTech, consumer finance",
			SourceCorpus:  "Sample corpus text for testing",
			StyleRules:    []string{"Use active voice", "Be concise"},
			TabooPhrases: []TabooPhraseInput{
				{Phrase: "synergy", Reason: "overused"},
				{Phrase: "leverage"},
			},
			Values: []string{"Innovation", "Customer Focus"},
			EvidenceURLs: []ProfileSourceInput{
				{URL: "https://example.com/about", SourceType: SourceTypeAbout},
				{URL: "https://example.com/values", SourceType: SourceTypeValues},
			},
		}

		profile, err := db.CreateCompanyProfile(ctx, input)
		if err != nil {
			t.Fatalf("CreateCompanyProfile failed: %v", err)
		}

		if profile.ID == uuid.Nil {
			t.Error("Profile ID should not be nil")
		}
		if profile.Tone != "direct and technical" {
			t.Errorf("Tone = %q, want 'direct and technical'", profile.Tone)
		}
		if profile.Version != 1 {
			t.Errorf("Version = %d, want 1", profile.Version)
		}
		if len(profile.StyleRules) != 2 {
			t.Errorf("StyleRules count = %d, want 2", len(profile.StyleRules))
		}
		if len(profile.TabooPhrases) != 2 {
			t.Errorf("TabooPhrases count = %d, want 2", len(profile.TabooPhrases))
		}
		if len(profile.Values) != 2 {
			t.Errorf("Values count = %d, want 2", len(profile.Values))
		}
		if len(profile.EvidenceURLs) != 2 {
			t.Errorf("EvidenceURLs count = %d, want 2", len(profile.EvidenceURLs))
		}
	})

	t.Run("updates existing profile", func(t *testing.T) {
		// Get current profile
		original, _ := db.GetCompanyProfileByCompanyID(ctx, company.ID)
		originalVersion := original.Version

		// Update with new data
		input := &ProfileCreateInput{
			CompanyID:     company.ID,
			Tone:          "warm and inclusive",
			DomainContext: "FinTech",
			StyleRules:    []string{"New rule 1", "New rule 2", "New rule 3"},
			Values:        []string{"Updated Value"},
		}

		updated, err := db.CreateCompanyProfile(ctx, input)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if updated.Version != originalVersion+1 {
			t.Errorf("Version = %d, want %d", updated.Version, originalVersion+1)
		}
		if updated.Tone != "warm and inclusive" {
			t.Errorf("Tone not updated")
		}
		if len(updated.StyleRules) != 3 {
			t.Errorf("StyleRules not replaced, got %d", len(updated.StyleRules))
		}
	})
}

func TestIntegration_GetCompanyProfile(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create test company and profile
	company, _ := db.FindOrCreateCompany(ctx, "Get Profile Test Corp")
	defer cleanupCompany(t, db, company.ID)

	input := &ProfileCreateInput{
		CompanyID:    company.ID,
		Tone:         "professional",
		StyleRules:   []string{"Be clear"},
		TabooPhrases: []TabooPhraseInput{{Phrase: "test"}},
		Values:       []string{"Excellence"},
	}
	_, _ = db.CreateCompanyProfile(ctx, input)

	t.Run("get by company ID", func(t *testing.T) {
		profile, err := db.GetCompanyProfileByCompanyID(ctx, company.ID)
		if err != nil {
			t.Fatalf("GetCompanyProfileByCompanyID failed: %v", err)
		}
		if profile == nil {
			t.Fatal("Profile not found")
		}
		if profile.Tone != "professional" {
			t.Errorf("Tone = %q, want 'professional'", profile.Tone)
		}
	})

	t.Run("get by profile ID", func(t *testing.T) {
		byCompany, _ := db.GetCompanyProfileByCompanyID(ctx, company.ID)

		profile, err := db.GetCompanyProfileByID(ctx, byCompany.ID)
		if err != nil {
			t.Fatalf("GetCompanyProfileByID failed: %v", err)
		}
		if profile == nil {
			t.Fatal("Profile not found")
		}
		if profile.ID != byCompany.ID {
			t.Error("IDs don't match")
		}
	})

	t.Run("get nonexistent returns nil", func(t *testing.T) {
		profile, err := db.GetCompanyProfileByCompanyID(ctx, uuid.New())
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if profile != nil {
			t.Error("Expected nil for nonexistent profile")
		}
	})

	t.Run("get with company details", func(t *testing.T) {
		profile, err := db.GetProfileWithCompany(ctx, company.ID)
		if err != nil {
			t.Fatalf("GetProfileWithCompany failed: %v", err)
		}
		if profile.Company == nil {
			t.Error("Company should be loaded")
		}
		if profile.Company.Name != "Get Profile Test Corp" {
			t.Errorf("Company name = %q", profile.Company.Name)
		}
	})
}

func TestIntegration_GetFreshCompanyProfile(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, _ := db.FindOrCreateCompany(ctx, "Fresh Profile Test Corp")
	defer cleanupCompany(t, db, company.ID)

	input := &ProfileCreateInput{
		CompanyID: company.ID,
		Tone:      "fresh",
	}
	_, _ = db.CreateCompanyProfile(ctx, input)

	t.Run("returns fresh profile", func(t *testing.T) {
		profile, err := db.GetFreshCompanyProfile(ctx, company.ID, 24*time.Hour)
		if err != nil {
			t.Fatalf("GetFreshCompanyProfile failed: %v", err)
		}
		if profile == nil {
			t.Error("Fresh profile should be returned")
		}
	})

	t.Run("returns nil for stale profile", func(t *testing.T) {
		// Make profile stale by setting last_verified_at to past
		_, _ = db.pool.Exec(ctx,
			"UPDATE company_profiles SET last_verified_at = NOW() - INTERVAL '2 days' WHERE company_id = $1",
			company.ID)

		profile, err := db.GetFreshCompanyProfile(ctx, company.ID, 24*time.Hour)
		if err != nil {
			t.Fatalf("GetFreshCompanyProfile failed: %v", err)
		}
		if profile != nil {
			t.Error("Stale profile should not be returned")
		}
	})
}

func TestIntegration_UpdateProfileVerification(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, _ := db.FindOrCreateCompany(ctx, "Verify Profile Test Corp")
	defer cleanupCompany(t, db, company.ID)

	input := &ProfileCreateInput{
		CompanyID: company.ID,
		Tone:      "test",
	}
	profile, _ := db.CreateCompanyProfile(ctx, input)

	// Make profile stale
	_, _ = db.pool.Exec(ctx,
		"UPDATE company_profiles SET last_verified_at = NOW() - INTERVAL '2 days' WHERE id = $1",
		profile.ID)

	// Update verification
	err := db.UpdateProfileVerification(ctx, profile.ID)
	if err != nil {
		t.Fatalf("UpdateProfileVerification failed: %v", err)
	}

	// Should now be fresh
	fresh, _ := db.GetFreshCompanyProfile(ctx, company.ID, 24*time.Hour)
	if fresh == nil {
		t.Error("Profile should be fresh after verification update")
	}
}

func TestIntegration_DeleteCompanyProfile(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, _ := db.FindOrCreateCompany(ctx, "Delete Profile Test Corp")
	defer cleanupCompany(t, db, company.ID)

	input := &ProfileCreateInput{
		CompanyID:  company.ID,
		Tone:       "delete me",
		StyleRules: []string{"rule 1", "rule 2"},
		Values:     []string{"value 1"},
	}
	profile, _ := db.CreateCompanyProfile(ctx, input)

	// Delete
	err := db.DeleteCompanyProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("DeleteCompanyProfile failed: %v", err)
	}

	// Verify deleted
	deleted, _ := db.GetCompanyProfileByID(ctx, profile.ID)
	if deleted != nil {
		t.Error("Profile should be deleted")
	}

	// Verify cascaded deletions
	var count int
	_ = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM company_style_rules WHERE profile_id = $1", profile.ID).Scan(&count)
	if count != 0 {
		t.Errorf("Style rules should be cascaded deleted, found %d", count)
	}
}

func TestIntegration_ListStaleProfiles(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create two companies with profiles
	company1, _ := db.FindOrCreateCompany(ctx, "Stale List Corp 1")
	company2, _ := db.FindOrCreateCompany(ctx, "Stale List Corp 2")
	defer cleanupCompany(t, db, company1.ID)
	defer cleanupCompany(t, db, company2.ID)

	_, _ = db.CreateCompanyProfile(ctx, &ProfileCreateInput{CompanyID: company1.ID, Tone: "fresh"})
	_, _ = db.CreateCompanyProfile(ctx, &ProfileCreateInput{CompanyID: company2.ID, Tone: "stale"})

	// Make company2's profile stale
	_, _ = db.pool.Exec(ctx,
		`UPDATE company_profiles SET last_verified_at = NOW() - INTERVAL '7 days' 
		 WHERE company_id = $1`,
		company2.ID)

	stale, err := db.ListStaleProfiles(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("ListStaleProfiles failed: %v", err)
	}

	// Should find at least company2's profile
	found := false
	for _, p := range stale {
		if p.CompanyID == company2.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Stale profile should be in list")
	}
}

// =============================================================================
// Brand Signal Integration Tests
// =============================================================================

func TestIntegration_BrandSignals(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create test company and crawled page
	company, _ := db.FindOrCreateCompany(ctx, "Brand Signal Test Corp")
	defer cleanupCompany(t, db, company.ID)

	html := "<html><body>Values page</body></html>"
	text := "Our values are..."
	status := 200
	page := &CrawledPage{
		CompanyID:   &company.ID,
		URL:         "https://brandsignal.test/values-" + uuid.New().String(),
		RawHTML:     &html,
		ParsedText:  &text,
		HTTPStatus:  &status,
		FetchStatus: FetchStatusSuccess,
	}
	_ = db.UpsertCrawledPage(ctx, page)

	t.Run("create brand signal", func(t *testing.T) {
		confidence := 0.85
		signalType := SignalTypeValues
		signal := &BrandSignal{
			CrawledPageID:   page.ID,
			SignalType:      &signalType,
			KeyPoints:       []string{"Innovation", "Customer first"},
			ExtractedValues: []string{"Innovation", "Integrity"},
			RawExcerpt:      strPtr("We believe in innovation and customer first..."),
			ConfidenceScore: &confidence,
		}

		err := db.CreateBrandSignal(ctx, signal)
		if err != nil {
			t.Fatalf("CreateBrandSignal failed: %v", err)
		}

		if signal.ID == uuid.Nil {
			t.Error("Signal ID should be set")
		}
	})

	t.Run("get signals by page", func(t *testing.T) {
		signals, err := db.GetBrandSignalsByPage(ctx, page.ID)
		if err != nil {
			t.Fatalf("GetBrandSignalsByPage failed: %v", err)
		}
		if len(signals) != 1 {
			t.Errorf("Expected 1 signal, got %d", len(signals))
		}
		if len(signals[0].KeyPoints) != 2 {
			t.Errorf("Expected 2 key points, got %d", len(signals[0].KeyPoints))
		}
	})

	t.Run("get signals by company", func(t *testing.T) {
		signals, err := db.GetBrandSignalsByCompany(ctx, company.ID)
		if err != nil {
			t.Fatalf("GetBrandSignalsByCompany failed: %v", err)
		}
		if len(signals) != 1 {
			t.Errorf("Expected 1 signal, got %d", len(signals))
		}
		if signals[0].URL != page.URL {
			t.Error("URL should be joined from crawled_pages")
		}
	})

	t.Run("delete signals for page", func(t *testing.T) {
		err := db.DeleteBrandSignalsForPage(ctx, page.ID)
		if err != nil {
			t.Fatalf("DeleteBrandSignalsForPage failed: %v", err)
		}

		signals, _ := db.GetBrandSignalsByPage(ctx, page.ID)
		if len(signals) != 0 {
			t.Error("Signals should be deleted")
		}
	})
}

func TestIntegration_MultipleBrandSignals(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, _ := db.FindOrCreateCompany(ctx, "Multi Signal Test Corp")
	defer cleanupCompany(t, db, company.ID)

	// Create multiple pages with signals
	pageURLs := []string{
		"https://multisignal.test/values-" + uuid.New().String(),
		"https://multisignal.test/culture-" + uuid.New().String(),
	}

	for i, url := range pageURLs {
		html := "<html><body>Page content</body></html>"
		status := 200
		page := &CrawledPage{
			CompanyID:   &company.ID,
			URL:         url,
			RawHTML:     &html,
			HTTPStatus:  &status,
			FetchStatus: FetchStatusSuccess,
		}
		_ = db.UpsertCrawledPage(ctx, page)

		signalType := SignalTypeValues
		if i == 1 {
			signalType = SignalTypeCulture
		}

		_ = db.CreateBrandSignal(ctx, &BrandSignal{
			CrawledPageID: page.ID,
			SignalType:    &signalType,
			KeyPoints:     []string{"Point 1", "Point 2"},
		})
	}

	signals, err := db.GetBrandSignalsByCompany(ctx, company.ID)
	if err != nil {
		t.Fatalf("GetBrandSignalsByCompany failed: %v", err)
	}
	if len(signals) != 2 {
		t.Errorf("Expected 2 signals, got %d", len(signals))
	}
}

// =============================================================================
// Profile Source Integration Tests
// =============================================================================

func TestIntegration_ProfileSources(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, _ := db.FindOrCreateCompany(ctx, "Profile Source Test Corp")
	defer cleanupCompany(t, db, company.ID)

	// Create a crawled page first
	html := "<html></html>"
	status := 200
	page := &CrawledPage{
		CompanyID:   &company.ID,
		URL:         "https://sourcestest.com/about-" + uuid.New().String(),
		RawHTML:     &html,
		HTTPStatus:  &status,
		FetchStatus: FetchStatusSuccess,
	}
	_ = db.UpsertCrawledPage(ctx, page)

	// Create profile with source linked to crawled page
	input := &ProfileCreateInput{
		CompanyID: company.ID,
		Tone:      "test",
		EvidenceURLs: []ProfileSourceInput{
			{URL: page.URL, CrawledPageID: &page.ID, SourceType: SourceTypeAbout},
			{URL: "https://external.com/values", SourceType: SourceTypeValues}, // No crawled page
		},
	}

	profile, err := db.CreateCompanyProfile(ctx, input)
	if err != nil {
		t.Fatalf("CreateCompanyProfile failed: %v", err)
	}

	if len(profile.EvidenceURLs) != 2 {
		t.Errorf("Expected 2 evidence URLs, got %d", len(profile.EvidenceURLs))
	}

	// Verify sources are linked correctly
	var count int
	err = db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM company_profile_sources 
		 WHERE profile_id = $1 AND crawled_page_id IS NOT NULL`,
		profile.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 source linked to crawled page, got %d", count)
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestIntegration_ProfileWithEmptyArrays(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, _ := db.FindOrCreateCompany(ctx, "Empty Arrays Test Corp")
	defer cleanupCompany(t, db, company.ID)

	input := &ProfileCreateInput{
		CompanyID:    company.ID,
		Tone:         "minimal",
		StyleRules:   []string{},
		TabooPhrases: []TabooPhraseInput{},
		Values:       []string{},
		EvidenceURLs: []ProfileSourceInput{},
	}

	profile, err := db.CreateCompanyProfile(ctx, input)
	if err != nil {
		t.Fatalf("CreateCompanyProfile failed: %v", err)
	}

	if len(profile.StyleRules) != 0 {
		t.Error("StyleRules should be empty")
	}
	if len(profile.TabooPhrases) != 0 {
		t.Error("TabooPhrases should be empty")
	}
}

func TestIntegration_ProfileStyleRuleOrdering(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, _ := db.FindOrCreateCompany(ctx, "Rule Ordering Test Corp")
	defer cleanupCompany(t, db, company.ID)

	input := &ProfileCreateInput{
		CompanyID:  company.ID,
		Tone:       "test",
		StyleRules: []string{"First rule", "Second rule", "Third rule"},
	}

	profile, _ := db.CreateCompanyProfile(ctx, input)

	// First rule should have highest priority and come first
	if profile.StyleRules[0] != "First rule" {
		t.Errorf("First rule should be first, got %q", profile.StyleRules[0])
	}
	if profile.StyleRules[2] != "Third rule" {
		t.Errorf("Third rule should be last, got %q", profile.StyleRules[2])
	}
}
