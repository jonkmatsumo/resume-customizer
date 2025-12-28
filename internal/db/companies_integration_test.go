//go:build integration

package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// These tests require a running PostgreSQL database.
// Set TEST_DATABASE_URL environment variable to run them.
// Example: TEST_DATABASE_URL=postgres://user:pass@localhost:5432/resume_customizer_test

func getTestDB(t *testing.T) *DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	db, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Clean up test data before each test
	ctx := context.Background()
	_, _ = db.pool.Exec(ctx, "DELETE FROM crawled_pages WHERE url LIKE '%test.example.com%'")
	_, _ = db.pool.Exec(ctx, "DELETE FROM company_domains WHERE domain LIKE '%test.example.com%'")
	_, _ = db.pool.Exec(ctx, "DELETE FROM companies WHERE name_normalized LIKE 'testcompany%'")

	return db
}

func TestIntegration_FindOrCreateCompany(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create a company
	company, err := db.FindOrCreateCompany(ctx, "Test Company Alpha")
	if err != nil {
		t.Fatalf("FindOrCreateCompany failed: %v", err)
	}
	if company == nil {
		t.Fatal("Expected company, got nil")
	}
	if company.Name != "Test Company Alpha" {
		t.Errorf("Expected name 'Test Company Alpha', got %q", company.Name)
	}
	if company.NameNormalized != "testcompanyalpha" {
		t.Errorf("Expected normalized name 'testcompanyalpha', got %q", company.NameNormalized)
	}

	// Finding same company again should return the same record
	company2, err := db.FindOrCreateCompany(ctx, "test company alpha")
	if err != nil {
		t.Fatalf("FindOrCreateCompany (second call) failed: %v", err)
	}
	if company2.ID != company.ID {
		t.Errorf("Expected same company ID, got different: %s vs %s", company.ID, company2.ID)
	}

	// Different company name should create new record
	company3, err := db.FindOrCreateCompany(ctx, "Test Company Beta")
	if err != nil {
		t.Fatalf("FindOrCreateCompany (different company) failed: %v", err)
	}
	if company3.ID == company.ID {
		t.Errorf("Expected different company ID for different company")
	}
}

func TestIntegration_GetCompanyByID(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create a company
	company, err := db.FindOrCreateCompany(ctx, "Test Company Gamma")
	if err != nil {
		t.Fatalf("FindOrCreateCompany failed: %v", err)
	}

	// Retrieve by ID
	retrieved, err := db.GetCompanyByID(ctx, company.ID)
	if err != nil {
		t.Fatalf("GetCompanyByID failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected company, got nil")
	}
	if retrieved.ID != company.ID {
		t.Errorf("Expected ID %s, got %s", company.ID, retrieved.ID)
	}

	// Non-existent ID should return nil
	nonExistent, err := db.GetCompanyByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetCompanyByID (non-existent) failed: %v", err)
	}
	if nonExistent != nil {
		t.Errorf("Expected nil for non-existent company, got %+v", nonExistent)
	}
}

func TestIntegration_CompanyDomains(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create a company
	company, err := db.FindOrCreateCompany(ctx, "Test Company Delta")
	if err != nil {
		t.Fatalf("FindOrCreateCompany failed: %v", err)
	}

	// Set primary domain
	err = db.UpdateCompanyDomain(ctx, company.ID, "https://www.test.example.com/")
	if err != nil {
		t.Fatalf("UpdateCompanyDomain failed: %v", err)
	}

	// Verify domain was normalized and saved
	updated, err := db.GetCompanyByID(ctx, company.ID)
	if err != nil {
		t.Fatalf("GetCompanyByID failed: %v", err)
	}
	if updated.Domain == nil || *updated.Domain != "test.example.com" {
		domain := "<nil>"
		if updated.Domain != nil {
			domain = *updated.Domain
		}
		t.Errorf("Expected domain 'test.example.com', got %q", domain)
	}

	// Add additional domains
	err = db.AddCompanyDomain(ctx, company.ID, "blog.test.example.com", DomainTypeTechBlog)
	if err != nil {
		t.Fatalf("AddCompanyDomain failed: %v", err)
	}

	// List domains
	domains, err := db.ListCompanyDomains(ctx, company.ID)
	if err != nil {
		t.Fatalf("ListCompanyDomains failed: %v", err)
	}
	if len(domains) != 1 {
		t.Errorf("Expected 1 domain in list, got %d", len(domains))
	}

	// Find by domain
	found, err := db.GetCompanyByDomain(ctx, "blog.test.example.com")
	if err != nil {
		t.Fatalf("GetCompanyByDomain failed: %v", err)
	}
	if found == nil || found.ID != company.ID {
		t.Errorf("Expected to find company by domain")
	}
}

func TestIntegration_CrawledPage_CRUD(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	testURL := "https://test.example.com/page-" + uuid.New().String()
	rawHTML := "<html><body>Hello World</body></html>"
	parsedText := "Hello World"

	page := &CrawledPage{
		URL:         testURL,
		RawHTML:     &rawHTML,
		ParsedText:  &parsedText,
		HTTPStatus:  intPtr(200),
		FetchStatus: FetchStatusSuccess,
	}

	// Upsert new page
	err := db.UpsertCrawledPage(ctx, page)
	if err != nil {
		t.Fatalf("UpsertCrawledPage failed: %v", err)
	}
	if page.ID == uuid.Nil {
		t.Error("Expected page ID to be set after upsert")
	}

	// Retrieve page
	retrieved, err := db.GetCrawledPageByURL(ctx, testURL)
	if err != nil {
		t.Fatalf("GetCrawledPageByURL failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected page, got nil")
	}
	if retrieved.URL != testURL {
		t.Errorf("Expected URL %q, got %q", testURL, retrieved.URL)
	}
	if retrieved.ContentHash == nil || *retrieved.ContentHash == "" {
		t.Error("Expected content hash to be computed")
	}
	if retrieved.FetchStatus != FetchStatusSuccess {
		t.Errorf("Expected status 'success', got %q", retrieved.FetchStatus)
	}

	// Update page
	updatedHTML := "<html><body>Updated content</body></html>"
	page.RawHTML = &updatedHTML
	err = db.UpsertCrawledPage(ctx, page)
	if err != nil {
		t.Fatalf("UpsertCrawledPage (update) failed: %v", err)
	}

	// Verify update
	updated, err := db.GetCrawledPageByURL(ctx, testURL)
	if err != nil {
		t.Fatalf("GetCrawledPageByURL after update failed: %v", err)
	}
	if retrieved.ContentHash != nil && updated.ContentHash != nil && *retrieved.ContentHash == *updated.ContentHash {
		t.Error("Expected content hash to change after update")
	}
}

func TestIntegration_GetFreshCrawledPage(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	testURL := "https://test.example.com/fresh-" + uuid.New().String()
	rawHTML := "<html><body>Fresh content</body></html>"
	parsedText := "Fresh content"

	page := &CrawledPage{
		URL:         testURL,
		RawHTML:     &rawHTML,
		ParsedText:  &parsedText,
		HTTPStatus:  intPtr(200),
		FetchStatus: FetchStatusSuccess,
	}
	err := db.UpsertCrawledPage(ctx, page)
	if err != nil {
		t.Fatalf("UpsertCrawledPage failed: %v", err)
	}

	// Should be fresh with 7 day TTL
	fresh, err := db.GetFreshCrawledPage(ctx, testURL, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("GetFreshCrawledPage failed: %v", err)
	}
	if fresh == nil {
		t.Error("Expected fresh page, got nil")
	}

	// Should not be fresh with 0 TTL
	stale, err := db.GetFreshCrawledPage(ctx, testURL, 0)
	if err != nil {
		t.Fatalf("GetFreshCrawledPage (0 TTL) failed: %v", err)
	}
	if stale != nil {
		t.Error("Expected nil for expired page, got page")
	}
}

func TestIntegration_RecordFailedFetch(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	testURL := "https://test.example.com/notfound-" + uuid.New().String()

	// Record 404 error (permanent)
	err := db.RecordFailedFetch(ctx, testURL, 404, "Page not found")
	if err != nil {
		t.Fatalf("RecordFailedFetch failed: %v", err)
	}

	// Should be marked as permanent failure
	page, err := db.GetCrawledPageByURL(ctx, testURL)
	if err != nil {
		t.Fatalf("GetCrawledPageByURL failed: %v", err)
	}
	if page == nil {
		t.Fatal("Expected page record, got nil")
	}
	if !page.IsPermanentFailure {
		t.Error("Expected is_permanent_failure to be true for 404")
	}
	if page.FetchStatus != FetchStatusNotFound {
		t.Errorf("Expected fetch_status 'not_found', got %q", page.FetchStatus)
	}
	if page.RetryAfter != nil {
		t.Error("Expected retry_after to be nil for permanent failure")
	}
}

func TestIntegration_RecordFailedFetch_WithBackoff(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	testURL := "https://test.example.com/error-" + uuid.New().String()

	// Record 500 error (transient)
	err := db.RecordFailedFetch(ctx, testURL, 500, "Internal server error")
	if err != nil {
		t.Fatalf("RecordFailedFetch failed: %v", err)
	}

	// Should have retry_after set
	page, err := db.GetCrawledPageByURL(ctx, testURL)
	if err != nil {
		t.Fatalf("GetCrawledPageByURL failed: %v", err)
	}
	if page == nil {
		t.Fatal("Expected page record, got nil")
	}
	if page.IsPermanentFailure {
		t.Error("Expected is_permanent_failure to be false for 500")
	}
	if page.RetryAfter == nil {
		t.Error("Expected retry_after to be set for transient failure")
	}
	if page.RetryCount != 1 {
		t.Errorf("Expected retry_count 1, got %d", page.RetryCount)
	}

	// Record another failure - should increase backoff
	err = db.RecordFailedFetch(ctx, testURL, 500, "Still broken")
	if err != nil {
		t.Fatalf("RecordFailedFetch (second) failed: %v", err)
	}

	page2, err := db.GetCrawledPageByURL(ctx, testURL)
	if err != nil {
		t.Fatalf("GetCrawledPageByURL failed: %v", err)
	}
	if page2.RetryCount != 2 {
		t.Errorf("Expected retry_count 2, got %d", page2.RetryCount)
	}
	// Backoff should have increased
	if page2.RetryAfter != nil && page.RetryAfter != nil && !page2.RetryAfter.After(*page.RetryAfter) {
		t.Error("Expected retry_after to increase with exponential backoff")
	}
}

func TestIntegration_ShouldSkipURL(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Unknown URL should not be skipped
	unknownURL := "https://test.example.com/unknown-" + uuid.New().String()
	skip, _, err := db.ShouldSkipURL(ctx, unknownURL)
	if err != nil {
		t.Fatalf("ShouldSkipURL failed: %v", err)
	}
	if skip {
		t.Error("Unknown URL should not be skipped")
	}

	// Permanent failure should be skipped
	permanentURL := "https://test.example.com/gone-" + uuid.New().String()
	err = db.RecordFailedFetch(ctx, permanentURL, 404, "Not found")
	if err != nil {
		t.Fatalf("RecordFailedFetch failed: %v", err)
	}

	skip, reason, err = db.ShouldSkipURL(ctx, permanentURL)
	if err != nil {
		t.Fatalf("ShouldSkipURL (permanent) failed: %v", err)
	}
	if !skip {
		t.Error("Permanent failure should be skipped")
	}
	if reason != "Not found" {
		t.Errorf("Expected reason 'Not found', got %q", reason)
	}

	// Transient failure within backoff should be skipped
	transientURL := "https://test.example.com/temp-" + uuid.New().String()
	err = db.RecordFailedFetch(ctx, transientURL, 500, "Server error")
	if err != nil {
		t.Fatalf("RecordFailedFetch failed: %v", err)
	}

	skip, reason, err = db.ShouldSkipURL(ctx, transientURL)
	if err != nil {
		t.Fatalf("ShouldSkipURL (transient) failed: %v", err)
	}
	if !skip {
		t.Error("Transient failure within backoff should be skipped")
	}
	if reason != "retry backoff" {
		t.Errorf("Expected reason 'retry backoff', got %q", reason)
	}
}

func TestIntegration_ListFreshPagesByCompany(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create company
	company, err := db.FindOrCreateCompany(ctx, "Test Company Epsilon")
	if err != nil {
		t.Fatalf("FindOrCreateCompany failed: %v", err)
	}

	// Add some pages
	for i := 0; i < 3; i++ {
		testURL := "https://test.example.com/list-" + uuid.New().String()
		rawHTML := "<html><body>Page content</body></html>"
		page := &CrawledPage{
			CompanyID:   &company.ID,
			URL:         testURL,
			RawHTML:     &rawHTML,
			HTTPStatus:  intPtr(200),
			FetchStatus: FetchStatusSuccess,
		}
		err := db.UpsertCrawledPage(ctx, page)
		if err != nil {
			t.Fatalf("UpsertCrawledPage failed: %v", err)
		}
	}

	// List pages
	pages, err := db.ListFreshPagesByCompany(ctx, company.ID, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("ListFreshPagesByCompany failed: %v", err)
	}
	if len(pages) < 3 {
		t.Errorf("Expected at least 3 pages, got %d", len(pages))
	}
}

// Helper for creating int pointers
func intPtr(i int) *int {
	return &i
}
