// Package main provides a manual integration test for Phase 2 company profiles.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jonathan/resume-customizer/internal/db"
)

func main() {
	ctx := context.Background()

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://resume:resume@localhost:5432/resume_customizer"
	}

	database, err := db.New(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer database.Close()

	fmt.Println("=== Phase 2 Integration Test: Company Profiles ===")
	fmt.Println()

	// Test 1: Create company
	fmt.Println("1. Creating test company...")
	company, err := database.FindOrCreateCompany(ctx, "Profile Test Inc")
	if err != nil {
		log.Fatalf("Failed to create company: %v", err)
	}
	fmt.Printf("   ✓ Company: %s (ID: %s)\n", company.Name, company.ID)

	// Test 2: Create company profile
	fmt.Println("\n2. Creating company profile...")
	input := &db.ProfileCreateInput{
		CompanyID:     company.ID,
		Tone:          "direct, technical, and inclusive",
		DomainContext: "FinTech, consumer finance, mobile payments",
		SourceCorpus:  "Sample corpus text used for summarization...",
		StyleRules: []string{
			"Use active voice",
			"Lead with impact metrics",
			"Avoid jargon unless industry-specific",
		},
		TabooPhrases: []db.TabooPhraseInput{
			{Phrase: "synergy", Reason: "overused corporate buzzword"},
			{Phrase: "rockstar", Reason: "too casual"},
			{Phrase: "guru", Reason: "pretentious"},
		},
		Values: []string{
			"People Come First",
			"No Shortcuts",
			"Simpler is Better",
		},
		EvidenceURLs: []db.ProfileSourceInput{
			{URL: "https://example.com/about", SourceType: db.SourceTypeAbout},
			{URL: "https://example.com/values", SourceType: db.SourceTypeValues},
			{URL: "https://example.com/engineering", SourceType: db.SourceTypeEngineering},
		},
	}

	profile, err := database.CreateCompanyProfile(ctx, input)
	if err != nil {
		log.Fatalf("Failed to create profile: %v", err)
	}
	fmt.Printf("   ✓ Profile created: ID=%s, Version=%d\n", profile.ID, profile.Version)
	fmt.Printf("   ✓ Tone: %s\n", profile.Tone)
	fmt.Printf("   ✓ Style Rules: %d\n", len(profile.StyleRules))
	fmt.Printf("   ✓ Taboo Phrases: %d\n", len(profile.TabooPhrases))
	fmt.Printf("   ✓ Values: %d\n", len(profile.Values))
	fmt.Printf("   ✓ Evidence URLs: %d\n", len(profile.EvidenceURLs))

	// Test 3: Retrieve profile
	fmt.Println("\n3. Retrieving profile by company ID...")
	retrieved, err := database.GetCompanyProfileByCompanyID(ctx, company.ID)
	if err != nil {
		log.Fatalf("Failed to get profile: %v", err)
	}
	if retrieved != nil {
		fmt.Printf("   ✓ Retrieved profile: Tone=%q, Version=%d\n", retrieved.Tone, retrieved.Version)
	}

	// Test 4: Get fresh profile
	fmt.Println("\n4. Getting fresh profile (should return cached)...")
	fresh, err := database.GetFreshCompanyProfile(ctx, company.ID, 24*time.Hour)
	if err != nil {
		log.Fatalf("Failed to get fresh profile: %v", err)
	}
	if fresh != nil {
		fmt.Println("   ✓ Fresh profile returned from cache")
	}

	// Test 5: Update profile (should increment version)
	fmt.Println("\n5. Updating profile (should increment version)...")
	input.Tone = "warm, inclusive, and customer-focused"
	input.StyleRules = []string{"New rule 1", "New rule 2"}
	updated, err := database.CreateCompanyProfile(ctx, input)
	if err != nil {
		log.Fatalf("Failed to update profile: %v", err)
	}
	fmt.Printf("   ✓ Updated profile: Version=%d (was %d)\n", updated.Version, profile.Version)

	// Test 6: Create crawled page and brand signal
	fmt.Println("\n6. Creating crawled page with brand signal...")
	html := "<html><body>Our company values are...</body></html>"
	text := "Our company values are innovation, integrity, and customer focus."
	status := 200
	page := &db.CrawledPage{
		CompanyID:   &company.ID,
		URL:         "https://example.com/test-values-page",
		RawHTML:     &html,
		ParsedText:  &text,
		HTTPStatus:  &status,
		FetchStatus: db.FetchStatusSuccess,
	}
	err = database.UpsertCrawledPage(ctx, page)
	if err != nil {
		log.Fatalf("Failed to create page: %v", err)
	}
	fmt.Printf("   ✓ Crawled page: %s\n", page.URL)

	confidence := 0.92
	signalType := db.SignalTypeValues
	signal := &db.BrandSignal{
		CrawledPageID:   page.ID,
		SignalType:      &signalType,
		KeyPoints:       []string{"Innovation", "Integrity", "Customer focus"},
		ExtractedValues: []string{"innovation", "integrity", "customer focus"},
		RawExcerpt:      &text,
		ConfidenceScore: &confidence,
	}
	err = database.CreateBrandSignal(ctx, signal)
	if err != nil {
		log.Fatalf("Failed to create signal: %v", err)
	}
	fmt.Printf("   ✓ Brand signal: Type=%s, Confidence=%.2f\n", *signal.SignalType, *signal.ConfidenceScore)

	// Test 7: Get brand signals by company
	fmt.Println("\n7. Getting brand signals for company...")
	signals, err := database.GetBrandSignalsByCompany(ctx, company.ID)
	if err != nil {
		log.Fatalf("Failed to get signals: %v", err)
	}
	fmt.Printf("   ✓ Found %d brand signals\n", len(signals))
	for _, s := range signals {
		sigType := "unknown"
		if s.SignalType != nil {
			sigType = *s.SignalType
		}
		fmt.Printf("     - Type: %s, Points: %v\n", sigType, s.KeyPoints)
	}

	// Test 8: Get profile with company details
	fmt.Println("\n8. Getting profile with company details...")
	withCompany, err := database.GetProfileWithCompany(ctx, company.ID)
	if err != nil {
		log.Fatalf("Failed to get profile with company: %v", err)
	}
	if withCompany.Company != nil {
		fmt.Printf("   ✓ Profile for company: %s\n", withCompany.Company.Name)
	}

	// Test 9: List stale profiles
	fmt.Println("\n9. Testing stale profile detection...")
	// Make profile stale
	_, _ = database.Pool().Exec(ctx,
		"UPDATE company_profiles SET last_verified_at = NOW() - INTERVAL '60 days' WHERE company_id = $1",
		company.ID)

	stale, err := database.ListStaleProfiles(ctx, 30*24*time.Hour)
	if err != nil {
		log.Fatalf("Failed to list stale profiles: %v", err)
	}
	foundStale := false
	for _, p := range stale {
		if p.CompanyID == company.ID {
			foundStale = true
			break
		}
	}
	if foundStale {
		fmt.Println("   ✓ Correctly detected stale profile")
	} else {
		fmt.Println("   ✗ Failed to detect stale profile")
	}

	// Test 10: Update verification
	fmt.Println("\n10. Updating profile verification...")
	err = database.UpdateProfileVerification(ctx, updated.ID)
	if err != nil {
		log.Fatalf("Failed to update verification: %v", err)
	}
	fmt.Println("   ✓ Verification timestamp updated")

	// Verify no longer stale
	stale, _ = database.ListStaleProfiles(ctx, 30*24*time.Hour)
	stillStale := false
	for _, p := range stale {
		if p.CompanyID == company.ID {
			stillStale = true
			break
		}
	}
	if !stillStale {
		fmt.Println("   ✓ Profile no longer detected as stale")
	}

	fmt.Println("\n=== All Phase 2 Tests Passed ===")
}
