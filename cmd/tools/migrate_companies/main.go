// Command migrate_companies extracts company names from existing job_profile artifacts
// and populates the new companies table.
//
// This is a one-time migration script for Phase 1 of the database normalization project.
//
// Usage:
//
//	go run cmd/tools/migrate_companies/main.go
//
// Requires DATABASE_URL environment variable to be set.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jonathan/resume-customizer/internal/db"
)


func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "ERROR: DATABASE_URL environment variable not set")
		os.Exit(1)
	}

	ctx := context.Background()

	// Connect to database
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	database, err := db.New(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to create db instance: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	fmt.Println("=== Company Migration Script ===")
	fmt.Println()

	// Query distinct company names from job_profile artifacts
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT content->>'company' AS company_name
		FROM artifacts
		WHERE step = 'job_profile'
		  AND content->>'company' IS NOT NULL
		  AND content->>'company' != ''
		ORDER BY company_name
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to query artifacts: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	var companyNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to scan row: %v\n", err)
			os.Exit(1)
		}
		companyNames = append(companyNames, name)
	}

	if len(companyNames) == 0 {
		fmt.Println("No companies found in job_profile artifacts.")
		fmt.Println("This is expected if no pipeline runs have been executed yet.")
		return
	}

	fmt.Printf("Found %d distinct companies in job_profile artifacts:\n\n", len(companyNames))

	created := 0
	existing := 0
	failed := 0

	for _, name := range companyNames {
		company, err := database.FindOrCreateCompany(ctx, name)
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", name, err)
			failed++
			continue
		}

		// Check if it was just created or already existed
		normalized := db.NormalizeName(name)
		existingCompany, _ := database.GetCompanyByNormalizedName(ctx, normalized)
		if existingCompany != nil && existingCompany.ID == company.ID {
			// Check if created just now by comparing timestamps
			if company.CreatedAt.Equal(company.UpdatedAt) {
				fmt.Printf("  ✓ Created: %s (normalized: %s)\n", name, company.NameNormalized)
				created++
			} else {
				fmt.Printf("  • Existing: %s (ID: %s)\n", name, company.ID)
				existing++
			}
		}
	}

	fmt.Println()
	fmt.Println("=== Migration Summary ===")
	fmt.Printf("  Created: %d\n", created)
	fmt.Printf("  Existing: %d\n", existing)
	fmt.Printf("  Failed: %d\n", failed)
	fmt.Printf("  Total: %d\n", len(companyNames))

	// Also migrate company domains from company_profile artifacts if available
	fmt.Println()
	fmt.Println("Checking for company profiles with domains...")

	domainRows, err := pool.Query(ctx, `
		SELECT 
			p.content->>'company' AS company_name,
			p.content AS profile_json
		FROM artifacts p
		WHERE p.step = 'company_profile'
		  AND p.content->>'company' IS NOT NULL
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Failed to query company_profile artifacts: %v\n", err)
		return
	}
	defer domainRows.Close()

	domainsAdded := 0
	for domainRows.Next() {
		var companyName string
		var profileJSON []byte
		if err := domainRows.Scan(&companyName, &profileJSON); err != nil {
			continue
		}

		// Parse the profile to extract website
		var profile struct {
			Website string `json:"website"`
		}
		if err := json.Unmarshal(profileJSON, &profile); err != nil {
			continue
		}

		if profile.Website == "" {
			continue
		}

		// Find the company
		normalized := db.NormalizeName(companyName)
		company, err := database.GetCompanyByNormalizedName(ctx, normalized)
		if err != nil || company == nil {
			continue
		}

		// Extract and add domain
		domain, err := db.ExtractDomain(profile.Website)
		if err != nil || domain == "" {
			continue
		}

		if company.Domain == nil || *company.Domain == "" {
			err = database.UpdateCompanyDomain(ctx, company.ID, domain)
			if err == nil {
				fmt.Printf("  ✓ Added domain %s to %s\n", domain, companyName)
				domainsAdded++
			}
		}
	}

	if domainsAdded > 0 {
		fmt.Printf("\nDomains added: %d\n", domainsAdded)
	} else {
		fmt.Println("No new domains to add.")
	}

	fmt.Println()
	fmt.Println("=== Migration Complete ===")
}

// Helper to get normalized name (exposed for use in this script)
func init() {
	// Verify db.NormalizeName exists
	_ = db.NormalizeName("test")
	_ = uuid.Nil
}

