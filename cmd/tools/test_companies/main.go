// Command test_companies is a manual integration test for the companies and crawled pages tables.
// It verifies that the database schema and repository methods work correctly.
//
// Usage:
//
//	go run cmd/tools/test_companies/main.go
//
// Requires DATABASE_URL environment variable to be set.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/fetch"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "ERROR: DATABASE_URL environment variable not set")
		os.Exit(1)
	}

	database, err := db.New(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	ctx := context.Background()

	fmt.Println("=== Phase 1 Integration Test ===")
	fmt.Println()

	// Test 1: Create a company
	fmt.Println("Test 1: Creating company...")
	company, err := database.FindOrCreateCompany(ctx, "Test Integration Company")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: FindOrCreateCompany: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Created company: %s (ID: %s)\n", company.Name, company.ID)
	fmt.Printf("  Normalized name: %s\n", company.NameNormalized)

	// Test 2: Verify deduplication
	fmt.Println("\nTest 2: Testing deduplication...")
	company2, err := database.FindOrCreateCompany(ctx, "test integration company") // lowercase
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: FindOrCreateCompany (dedup): %v\n", err)
		os.Exit(1)
	}
	if company2.ID != company.ID {
		fmt.Fprintf(os.Stderr, "FAIL: Expected same company ID for normalized match\n")
		fmt.Fprintf(os.Stderr, "  Got: %s vs %s\n", company2.ID, company.ID)
		os.Exit(1)
	}
	fmt.Println("  Deduplication works correctly")

	// Test 3: Add domain
	fmt.Println("\nTest 3: Adding company domain...")
	err = database.UpdateCompanyDomain(ctx, company.ID, "https://www.test-integration.example.com/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: UpdateCompanyDomain: %v\n", err)
		os.Exit(1)
	}
	updated, _ := database.GetCompanyByID(ctx, company.ID)
	if updated.Domain == nil || *updated.Domain != "test-integration.example.com" {
		fmt.Fprintf(os.Stderr, "FAIL: Domain not normalized correctly\n")
		os.Exit(1)
	}
	fmt.Printf("  Domain set: %s\n", *updated.Domain)

	// Test 4: Add additional domain
	fmt.Println("\nTest 4: Adding additional domain...")
	err = database.AddCompanyDomain(ctx, company.ID, "blog.test-integration.example.com", db.DomainTypeTechBlog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: AddCompanyDomain: %v\n", err)
		os.Exit(1)
	}
	domains, _ := database.ListCompanyDomains(ctx, company.ID)
	fmt.Printf("  Company has %d additional domains\n", len(domains))

	// Test 5: Find by domain
	fmt.Println("\nTest 5: Finding company by domain...")
	found, err := database.GetCompanyByDomain(ctx, "blog.test-integration.example.com")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: GetCompanyByDomain: %v\n", err)
		os.Exit(1)
	}
	if found == nil || found.ID != company.ID {
		fmt.Fprintf(os.Stderr, "FAIL: Company not found by domain\n")
		os.Exit(1)
	}
	fmt.Printf("  Found company: %s\n", found.Name)

	// Test 6: Cache a page
	fmt.Println("\nTest 6: Caching a page...")
	rawHTML := "<html><body><h1>Test Page</h1><p>This is test content.</p></body></html>"
	parsedText := "Test Page\nThis is test content."
	statusCode := 200
	testURL := fmt.Sprintf("https://test-integration.example.com/page-%d", time.Now().Unix())

	page := &db.CrawledPage{
		CompanyID:   &company.ID,
		URL:         testURL,
		RawHTML:     &rawHTML,
		ParsedText:  &parsedText,
		HTTPStatus:  &statusCode,
		FetchStatus: db.FetchStatusSuccess,
	}
	err = database.UpsertCrawledPage(ctx, page)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: UpsertCrawledPage: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Cached page: %s (ID: %s)\n", testURL, page.ID)
	if page.ContentHash != nil {
		fmt.Printf("  Content hash: %s...\n", (*page.ContentHash)[:16])
	}

	// Test 7: Retrieve fresh page
	fmt.Println("\nTest 7: Retrieving fresh page...")
	fresh, err := database.GetFreshCrawledPage(ctx, testURL, 7*24*time.Hour)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: GetFreshCrawledPage: %v\n", err)
		os.Exit(1)
	}
	if fresh == nil {
		fmt.Fprintf(os.Stderr, "FAIL: Expected fresh page, got nil\n")
		os.Exit(1)
	}
	fmt.Println("  Fresh page retrieved successfully")

	// Test 8: Record failed fetch
	fmt.Println("\nTest 8: Recording failed fetch...")
	failedURL := fmt.Sprintf("https://test-integration.example.com/notfound-%d", time.Now().Unix())
	err = database.RecordFailedFetch(ctx, failedURL, 404, "Page not found")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: RecordFailedFetch: %v\n", err)
		os.Exit(1)
	}
	failedPage, _ := database.GetCrawledPageByURL(ctx, failedURL)
	if !failedPage.IsPermanentFailure {
		fmt.Fprintf(os.Stderr, "FAIL: Expected permanent failure for 404\n")
		os.Exit(1)
	}
	fmt.Println("  404 recorded as permanent failure")

	// Test 9: ShouldSkipURL
	fmt.Println("\nTest 9: Testing ShouldSkipURL...")
	shouldSkip, reason, err := database.ShouldSkipURL(ctx, failedURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: ShouldSkipURL: %v\n", err)
		os.Exit(1)
	}
	if !shouldSkip {
		fmt.Fprintf(os.Stderr, "FAIL: Expected URL to be skipped\n")
		os.Exit(1)
	}
	fmt.Printf("  URL correctly skipped: %s\n", reason)

	// Test 10: Cached fetcher
	fmt.Println("\nTest 10: Testing CachedFetcher...")
	fetcher := fetch.NewCachedFetcher(database, nil)
	result, err := fetcher.Fetch(ctx, failedURL)
	if err == nil {
		fmt.Fprintf(os.Stderr, "FAIL: Expected error for skipped URL\n")
		os.Exit(1)
	}
	if result != nil {
		fmt.Fprintf(os.Stderr, "FAIL: Expected nil result for skipped URL\n")
		os.Exit(1)
	}
	fmt.Println("  CachedFetcher correctly skips failed URLs")

	// Clean up test data
	fmt.Println("\nCleaning up test data...")
	_, _ = database.Pool().Exec(ctx, "DELETE FROM crawled_pages WHERE url LIKE '%test-integration.example.com%'")
	_, _ = database.Pool().Exec(ctx, "DELETE FROM company_domains WHERE domain LIKE '%test-integration.example.com%'")
	_, _ = database.Pool().Exec(ctx, "DELETE FROM companies WHERE name_normalized = 'testintegrationcompany'")

	fmt.Println("\n=== All Tests Passed ===")
}
