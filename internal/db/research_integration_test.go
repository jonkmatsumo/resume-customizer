//go:build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Research Session Integration Tests
// =============================================================================

func TestIntegration_ResearchSession_CRUD(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create test company
	company, err := db.FindOrCreateCompany(ctx, "Research Test Company")
	if err != nil {
		t.Fatalf("Failed to create test company: %v", err)
	}
	defer cleanupTestCompany(t, db, company.ID)

	t.Run("create research session", func(t *testing.T) {
		input := &ResearchSessionInput{
			CompanyID:   &company.ID,
			CompanyName: company.Name,
			Domain:      "researchtestcompany.com",
			PagesLimit:  10,
		}

		session, err := db.CreateResearchSession(ctx, input)
		if err != nil {
			t.Fatalf("CreateResearchSession failed: %v", err)
		}

		if session.ID == uuid.Nil {
			t.Error("Session ID should not be nil")
		}
		if session.Status != ResearchStatusPending {
			t.Errorf("Status = %q, want 'pending'", session.Status)
		}
		if session.PagesLimit != 10 {
			t.Errorf("PagesLimit = %d, want 10", session.PagesLimit)
		}
		if session.PagesCrawled != 0 {
			t.Errorf("PagesCrawled = %d, want 0", session.PagesCrawled)
		}

		// Cleanup
		_ = db.DeleteResearchSession(ctx, session.ID)
	})

	t.Run("get research session by ID", func(t *testing.T) {
		input := &ResearchSessionInput{
			CompanyID:   &company.ID,
			CompanyName: company.Name,
		}
		created, _ := db.CreateResearchSession(ctx, input)
		defer func() { _ = db.DeleteResearchSession(ctx, created.ID) }()

		session, err := db.GetResearchSessionByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetResearchSessionByID failed: %v", err)
		}
		if session == nil {
			t.Fatal("Session not found")
		}
		if session.ID != created.ID {
			t.Error("ID mismatch")
		}
	})

	t.Run("update session status to in_progress", func(t *testing.T) {
		input := &ResearchSessionInput{
			CompanyID:   &company.ID,
			CompanyName: company.Name,
		}
		session, _ := db.CreateResearchSession(ctx, input)
		defer func() { _ = db.DeleteResearchSession(ctx, session.ID) }()

		err := db.UpdateResearchSessionStatus(ctx, session.ID, ResearchStatusInProgress, "")
		if err != nil {
			t.Fatalf("UpdateResearchSessionStatus failed: %v", err)
		}

		updated, _ := db.GetResearchSessionByID(ctx, session.ID)
		if updated.Status != ResearchStatusInProgress {
			t.Errorf("Status = %q, want 'in_progress'", updated.Status)
		}
		if updated.StartedAt == nil {
			t.Error("StartedAt should be set")
		}
	})

	t.Run("update session status to completed", func(t *testing.T) {
		input := &ResearchSessionInput{
			CompanyID:   &company.ID,
			CompanyName: company.Name,
		}
		session, _ := db.CreateResearchSession(ctx, input)
		defer func() { _ = db.DeleteResearchSession(ctx, session.ID) }()

		_ = db.UpdateResearchSessionStatus(ctx, session.ID, ResearchStatusInProgress, "")
		err := db.UpdateResearchSessionStatus(ctx, session.ID, ResearchStatusCompleted, "")
		if err != nil {
			t.Fatalf("UpdateResearchSessionStatus failed: %v", err)
		}

		updated, _ := db.GetResearchSessionByID(ctx, session.ID)
		if updated.Status != ResearchStatusCompleted {
			t.Errorf("Status = %q, want 'completed'", updated.Status)
		}
		if updated.CompletedAt == nil {
			t.Error("CompletedAt should be set")
		}
	})

	t.Run("update session status to failed with error", func(t *testing.T) {
		input := &ResearchSessionInput{
			CompanyID:   &company.ID,
			CompanyName: company.Name,
		}
		session, _ := db.CreateResearchSession(ctx, input)
		defer func() { _ = db.DeleteResearchSession(ctx, session.ID) }()

		err := db.UpdateResearchSessionStatus(ctx, session.ID, ResearchStatusFailed, "Connection timeout")
		if err != nil {
			t.Fatalf("UpdateResearchSessionStatus failed: %v", err)
		}

		updated, _ := db.GetResearchSessionByID(ctx, session.ID)
		if updated.Status != ResearchStatusFailed {
			t.Errorf("Status = %q, want 'failed'", updated.Status)
		}
		if updated.ErrorMessage == nil || *updated.ErrorMessage != "Connection timeout" {
			t.Error("ErrorMessage not set correctly")
		}
	})

	t.Run("update session progress", func(t *testing.T) {
		input := &ResearchSessionInput{
			CompanyID:   &company.ID,
			CompanyName: company.Name,
		}
		session, _ := db.CreateResearchSession(ctx, input)
		defer func() { _ = db.DeleteResearchSession(ctx, session.ID) }()

		corpus := "This is the aggregated corpus text from crawled pages."
		err := db.UpdateResearchSessionProgress(ctx, session.ID, 3, corpus)
		if err != nil {
			t.Fatalf("UpdateResearchSessionProgress failed: %v", err)
		}

		updated, _ := db.GetResearchSessionByID(ctx, session.ID)
		if updated.PagesCrawled != 3 {
			t.Errorf("PagesCrawled = %d, want 3", updated.PagesCrawled)
		}
		if updated.CorpusText == nil || *updated.CorpusText != corpus {
			t.Error("CorpusText not set correctly")
		}
	})

	t.Run("get recent research session", func(t *testing.T) {
		input := &ResearchSessionInput{
			CompanyID:   &company.ID,
			CompanyName: company.Name,
		}
		session, _ := db.CreateResearchSession(ctx, input)
		defer func() { _ = db.DeleteResearchSession(ctx, session.ID) }()

		// Complete the session
		_ = db.UpdateResearchSessionStatus(ctx, session.ID, ResearchStatusCompleted, "")

		// Find recent session (within 1 hour)
		recent, err := db.GetRecentResearchSession(ctx, company.ID, time.Hour)
		if err != nil {
			t.Fatalf("GetRecentResearchSession failed: %v", err)
		}
		if recent == nil {
			t.Fatal("Should find recent session")
		}
		if recent.ID != session.ID {
			t.Error("Should find the just-completed session")
		}
	})

	t.Run("no recent session for old data", func(t *testing.T) {
		// With 0 duration, nothing should match
		recent, err := db.GetRecentResearchSession(ctx, company.ID, 0)
		if err != nil {
			t.Fatalf("GetRecentResearchSession failed: %v", err)
		}
		if recent != nil {
			t.Error("Should not find recent session with 0 duration")
		}
	})

	t.Run("list sessions by company", func(t *testing.T) {
		input1 := &ResearchSessionInput{CompanyID: &company.ID, CompanyName: company.Name}
		input2 := &ResearchSessionInput{CompanyID: &company.ID, CompanyName: company.Name}
		session1, _ := db.CreateResearchSession(ctx, input1)
		session2, _ := db.CreateResearchSession(ctx, input2)
		defer func() {
			_ = db.DeleteResearchSession(ctx, session1.ID)
			_ = db.DeleteResearchSession(ctx, session2.ID)
		}()

		sessions, err := db.ListResearchSessionsByCompany(ctx, company.ID)
		if err != nil {
			t.Fatalf("ListResearchSessionsByCompany failed: %v", err)
		}
		if len(sessions) < 2 {
			t.Errorf("Should have at least 2 sessions, got %d", len(sessions))
		}
	})

	t.Run("session not found returns nil", func(t *testing.T) {
		session, err := db.GetResearchSessionByID(ctx, uuid.New())
		if err != nil {
			t.Fatalf("GetResearchSessionByID failed: %v", err)
		}
		if session != nil {
			t.Error("Should return nil for nonexistent session")
		}
	})
}

// =============================================================================
// Research Frontier Integration Tests
// =============================================================================

func TestIntegration_ResearchFrontier_CRUD(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, _ := db.FindOrCreateCompany(ctx, "Frontier Test Company")
	defer cleanupTestCompany(t, db, company.ID)

	session, _ := db.CreateResearchSession(ctx, &ResearchSessionInput{
		CompanyID:   &company.ID,
		CompanyName: company.Name,
	})
	defer func() { _ = db.DeleteResearchSession(ctx, session.ID) }()

	t.Run("add frontier URLs", func(t *testing.T) {
		input := []FrontierURLInput{
			{URL: "https://example.com/values", Priority: 0.9, PageType: PageTypeValues, Reason: "Values page"},
			{URL: "https://example.com/culture", Priority: 0.8, PageType: PageTypeCulture, Reason: "Culture page"},
			{URL: "https://example.com/about", Priority: 0.7, PageType: PageTypeAbout, Reason: "About page"},
		}

		urls, err := db.AddFrontierURLs(ctx, session.ID, input)
		if err != nil {
			t.Fatalf("AddFrontierURLs failed: %v", err)
		}

		if len(urls) != 3 {
			t.Errorf("URLs count = %d, want 3", len(urls))
		}
		if urls[0].Status != FrontierStatusPending {
			t.Errorf("Status = %q, want 'pending'", urls[0].Status)
		}
	})

	t.Run("get pending frontier URLs ordered by priority", func(t *testing.T) {
		urls, err := db.GetPendingFrontierURLs(ctx, session.ID)
		if err != nil {
			t.Fatalf("GetPendingFrontierURLs failed: %v", err)
		}

		if len(urls) != 3 {
			t.Errorf("URLs count = %d, want 3", len(urls))
		}
		// Should be sorted by priority DESC
		if urls[0].Priority < urls[1].Priority {
			t.Error("URLs should be sorted by priority DESC")
		}
	})

	t.Run("mark frontier URL as fetched", func(t *testing.T) {
		urls, _ := db.GetPendingFrontierURLs(ctx, session.ID)
		if len(urls) == 0 {
			t.Skip("No pending URLs")
		}

		err := db.MarkFrontierURLFetched(ctx, urls[0].ID, nil)
		if err != nil {
			t.Fatalf("MarkFrontierURLFetched failed: %v", err)
		}

		// Verify
		allURLs, _ := db.GetAllFrontierURLs(ctx, session.ID)
		var fetched *FrontierURL
		for _, u := range allURLs {
			if u.ID == urls[0].ID {
				fetched = &u
				break
			}
		}
		if fetched == nil {
			t.Fatal("Fetched URL not found")
		}
		if fetched.Status != FrontierStatusFetched {
			t.Errorf("Status = %q, want 'fetched'", fetched.Status)
		}
		if fetched.FetchedAt == nil {
			t.Error("FetchedAt should be set")
		}
	})

	t.Run("mark frontier URL as skipped", func(t *testing.T) {
		urls, _ := db.GetPendingFrontierURLs(ctx, session.ID)
		if len(urls) == 0 {
			t.Skip("No pending URLs")
		}

		err := db.MarkFrontierURLSkipped(ctx, urls[0].ID, "Third-party site")
		if err != nil {
			t.Fatalf("MarkFrontierURLSkipped failed: %v", err)
		}

		// Verify
		allURLs, _ := db.GetAllFrontierURLs(ctx, session.ID)
		var skipped *FrontierURL
		for _, u := range allURLs {
			if u.ID == urls[0].ID {
				skipped = &u
				break
			}
		}
		if skipped == nil {
			t.Fatal("Skipped URL not found")
		}
		if skipped.Status != FrontierStatusSkipped {
			t.Errorf("Status = %q, want 'skipped'", skipped.Status)
		}
		if skipped.SkipReason == nil || *skipped.SkipReason != "Third-party site" {
			t.Error("SkipReason not set correctly")
		}
	})

	t.Run("mark frontier URL as failed", func(t *testing.T) {
		urls, _ := db.GetPendingFrontierURLs(ctx, session.ID)
		if len(urls) == 0 {
			t.Skip("No pending URLs")
		}

		err := db.MarkFrontierURLFailed(ctx, urls[0].ID, "Connection timeout")
		if err != nil {
			t.Fatalf("MarkFrontierURLFailed failed: %v", err)
		}

		allURLs, _ := db.GetAllFrontierURLs(ctx, session.ID)
		var failed *FrontierURL
		for _, u := range allURLs {
			if u.ID == urls[0].ID {
				failed = &u
				break
			}
		}
		if failed == nil {
			t.Fatal("Failed URL not found")
		}
		if failed.Status != FrontierStatusFailed {
			t.Errorf("Status = %q, want 'failed'", failed.Status)
		}
	})

	t.Run("count frontier URLs by status", func(t *testing.T) {
		counts, err := db.CountFrontierURLsByStatus(ctx, session.ID)
		if err != nil {
			t.Fatalf("CountFrontierURLsByStatus failed: %v", err)
		}

		total := 0
		for _, c := range counts {
			total += c
		}
		if total != 3 {
			t.Errorf("Total URLs = %d, want 3", total)
		}
	})
}

// =============================================================================
// Research Brand Signals Integration Tests
// =============================================================================

func TestIntegration_ResearchBrandSignals_CRUD(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, _ := db.FindOrCreateCompany(ctx, "Brand Signal Test Company")
	defer cleanupTestCompany(t, db, company.ID)

	session, _ := db.CreateResearchSession(ctx, &ResearchSessionInput{
		CompanyID:   &company.ID,
		CompanyName: company.Name,
	})
	defer func() { _ = db.DeleteResearchSession(ctx, session.ID) }()

	t.Run("save brand signals", func(t *testing.T) {
		input := []ResearchBrandSignalInput{
			{
				URL:         "https://example.com/values",
				SignalType:  PageTypeValues,
				KeyPoints:   []string{"Customer obsession", "Innovation"},
				ValuesFound: []string{"customer_focus", "innovation"},
			},
			{
				URL:         "https://example.com/culture",
				SignalType:  PageTypeCulture,
				KeyPoints:   []string{"Collaboration", "Ownership"},
				ValuesFound: []string{"teamwork", "ownership"},
			},
		}

		signals, err := db.SaveResearchBrandSignals(ctx, session.ID, input)
		if err != nil {
			t.Fatalf("SaveResearchBrandSignals failed: %v", err)
		}

		if len(signals) != 2 {
			t.Errorf("Signals count = %d, want 2", len(signals))
		}
		if len(signals[0].KeyPoints) != 2 {
			t.Errorf("KeyPoints count = %d, want 2", len(signals[0].KeyPoints))
		}
	})

	t.Run("get brand signals for session", func(t *testing.T) {
		signals, err := db.GetResearchBrandSignals(ctx, session.ID)
		if err != nil {
			t.Fatalf("GetResearchBrandSignals failed: %v", err)
		}

		if len(signals) != 2 {
			t.Errorf("Signals count = %d, want 2", len(signals))
		}
	})

	t.Run("get brand signals by company", func(t *testing.T) {
		signals, err := db.GetResearchBrandSignalsByCompany(ctx, company.ID)
		if err != nil {
			t.Fatalf("GetResearchBrandSignalsByCompany failed: %v", err)
		}

		if len(signals) == 0 {
			t.Error("Should find signals for company")
		}
	})
}

// =============================================================================
// Full Session Loading Integration Tests
// =============================================================================

func TestIntegration_GetResearchSessionWithDetails(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, _ := db.FindOrCreateCompany(ctx, "Full Session Test Company")
	defer cleanupTestCompany(t, db, company.ID)

	session, _ := db.CreateResearchSession(ctx, &ResearchSessionInput{
		CompanyID:   &company.ID,
		CompanyName: company.Name,
	})
	defer func() { _ = db.DeleteResearchSession(ctx, session.ID) }()

	// Add frontier URLs
	_, _ = db.AddFrontierURLs(ctx, session.ID, []FrontierURLInput{
		{URL: "https://example.com/1", Priority: 0.9},
		{URL: "https://example.com/2", Priority: 0.8},
	})

	// Add brand signals
	_, _ = db.SaveResearchBrandSignals(ctx, session.ID, []ResearchBrandSignalInput{
		{URL: "https://example.com/1", SignalType: PageTypeValues, KeyPoints: []string{"Point 1"}},
	})

	t.Run("load session with details", func(t *testing.T) {
		fullSession, err := db.GetResearchSessionWithDetails(ctx, session.ID)
		if err != nil {
			t.Fatalf("GetResearchSessionWithDetails failed: %v", err)
		}

		if fullSession == nil {
			t.Fatal("Session not found")
		}
		if len(fullSession.FrontierURLs) != 2 {
			t.Errorf("FrontierURLs count = %d, want 2", len(fullSession.FrontierURLs))
		}
		if len(fullSession.BrandSignals) != 1 {
			t.Errorf("BrandSignals count = %d, want 1", len(fullSession.BrandSignals))
		}
	})
}

// =============================================================================
// Cascade Delete Tests
// =============================================================================

func TestIntegration_ResearchSession_CascadeDelete(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	company, _ := db.FindOrCreateCompany(ctx, "Cascade Test Company")
	defer cleanupTestCompany(t, db, company.ID)

	session, _ := db.CreateResearchSession(ctx, &ResearchSessionInput{
		CompanyID:   &company.ID,
		CompanyName: company.Name,
	})

	// Add frontier URLs
	urls, _ := db.AddFrontierURLs(ctx, session.ID, []FrontierURLInput{
		{URL: "https://example.com/1", Priority: 0.9},
	})
	urlID := urls[0].ID

	// Add brand signals
	signals, _ := db.SaveResearchBrandSignals(ctx, session.ID, []ResearchBrandSignalInput{
		{URL: "https://example.com/1", KeyPoints: []string{"Test"}},
	})
	signalID := signals[0].ID

	// Delete session
	err := db.DeleteResearchSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("DeleteResearchSession failed: %v", err)
	}

	// Verify session is deleted
	deletedSession, _ := db.GetResearchSessionByID(ctx, session.ID)
	if deletedSession != nil {
		t.Error("Session should be deleted")
	}

	// Verify frontier URLs are cascade deleted
	var urlCount int
	_ = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM research_frontier WHERE id = $1", urlID).Scan(&urlCount)
	if urlCount != 0 {
		t.Error("Frontier URLs should be cascade deleted")
	}

	// Verify brand signals are cascade deleted
	var signalCount int
	_ = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM research_brand_signals WHERE id = $1", signalID).Scan(&signalCount)
	if signalCount != 0 {
		t.Error("Brand signals should be cascade deleted")
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func cleanupTestCompany(t *testing.T, db *DB, companyID uuid.UUID) {
	t.Helper()
	_, _ = db.pool.Exec(context.Background(), "DELETE FROM companies WHERE id = $1", companyID)
}
