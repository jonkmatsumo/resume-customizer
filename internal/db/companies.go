package db

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// -----------------------------------------------------------------------------
// Company Methods
// -----------------------------------------------------------------------------

// FindOrCreateCompany finds an existing company by name or creates a new one
func (db *DB) FindOrCreateCompany(ctx context.Context, name string) (*Company, error) {
	normalized := NormalizeName(name)
	if normalized == "" {
		return nil, fmt.Errorf("company name cannot be empty")
	}

	// Try to find existing
	company, err := db.GetCompanyByNormalizedName(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if company != nil {
		return company, nil
	}

	// Create new
	var c Company
	err = db.pool.QueryRow(ctx,
		`INSERT INTO companies (name, name_normalized)
		 VALUES ($1, $2)
		 ON CONFLICT (name_normalized) DO UPDATE SET updated_at = NOW()
		 RETURNING id, name, name_normalized, domain, industry, created_at, updated_at`,
		name, normalized,
	).Scan(&c.ID, &c.Name, &c.NameNormalized, &c.Domain, &c.Industry, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}

	return &c, nil
}

// GetCompanyByNormalizedName retrieves a company by its normalized name
func (db *DB) GetCompanyByNormalizedName(ctx context.Context, normalized string) (*Company, error) {
	var c Company
	err := db.pool.QueryRow(ctx,
		`SELECT id, name, name_normalized, domain, industry, created_at, updated_at
		 FROM companies WHERE name_normalized = $1`,
		normalized,
	).Scan(&c.ID, &c.Name, &c.NameNormalized, &c.Domain, &c.Industry, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	return &c, nil
}

// GetCompanyByID retrieves a company by its UUID
func (db *DB) GetCompanyByID(ctx context.Context, id uuid.UUID) (*Company, error) {
	var c Company
	err := db.pool.QueryRow(ctx,
		`SELECT id, name, name_normalized, domain, industry, created_at, updated_at
		 FROM companies WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.Name, &c.NameNormalized, &c.Domain, &c.Industry, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	return &c, nil
}

// GetCompanyByDomain finds a company by one of its domains
func (db *DB) GetCompanyByDomain(ctx context.Context, domain string) (*Company, error) {
	domain = normalizeDomain(domain)

	var c Company
	err := db.pool.QueryRow(ctx,
		`SELECT c.id, c.name, c.name_normalized, c.domain, c.industry, c.created_at, c.updated_at
		 FROM companies c
		 LEFT JOIN company_domains cd ON cd.company_id = c.id
		 WHERE c.domain = $1 OR cd.domain = $1
		 LIMIT 1`,
		domain,
	).Scan(&c.ID, &c.Name, &c.NameNormalized, &c.Domain, &c.Industry, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get company by domain: %w", err)
	}
	return &c, nil
}

// UpdateCompanyDomain sets the primary domain for a company
func (db *DB) UpdateCompanyDomain(ctx context.Context, companyID uuid.UUID, domain string) error {
	domain = normalizeDomain(domain)
	_, err := db.pool.Exec(ctx,
		`UPDATE companies SET domain = $1, updated_at = NOW() WHERE id = $2`,
		domain, companyID,
	)
	if err != nil {
		return fmt.Errorf("failed to update company domain: %w", err)
	}
	return nil
}

// AddCompanyDomain adds an additional domain for a company
func (db *DB) AddCompanyDomain(ctx context.Context, companyID uuid.UUID, domain, domainType string) error {
	domain = normalizeDomain(domain)
	if domainType == "" {
		domainType = DomainTypePrimary
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO company_domains (company_id, domain, domain_type)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (domain) DO UPDATE SET domain_type = $3`,
		companyID, domain, domainType,
	)
	if err != nil {
		return fmt.Errorf("failed to add company domain: %w", err)
	}
	return nil
}

// ListCompanyDomains returns all domains associated with a company
func (db *DB) ListCompanyDomains(ctx context.Context, companyID uuid.UUID) ([]CompanyDomain, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, company_id, domain, domain_type, created_at
		 FROM company_domains WHERE company_id = $1 ORDER BY created_at`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list company domains: %w", err)
	}
	defer rows.Close()

	var domains []CompanyDomain
	for rows.Next() {
		var d CompanyDomain
		if err := rows.Scan(&d.ID, &d.CompanyID, &d.Domain, &d.DomainType, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan company domain: %w", err)
		}
		domains = append(domains, d)
	}
	return domains, nil
}

// -----------------------------------------------------------------------------
// Crawled Page Methods
// -----------------------------------------------------------------------------

// GetCrawledPageByURL retrieves a cached page by URL
func (db *DB) GetCrawledPageByURL(ctx context.Context, pageURL string) (*CrawledPage, error) {
	var p CrawledPage
	err := db.pool.QueryRow(ctx,
		`SELECT id, company_id, url, page_type, raw_html, parsed_text, content_hash, 
		        http_status, fetch_status, error_message, is_permanent_failure, retry_count, retry_after,
		        fetched_at, expires_at, last_accessed_at, created_at, updated_at
		 FROM crawled_pages WHERE url = $1`,
		pageURL,
	).Scan(&p.ID, &p.CompanyID, &p.URL, &p.PageType, &p.RawHTML, &p.ParsedText, &p.ContentHash,
		&p.HTTPStatus, &p.FetchStatus, &p.ErrorMessage, &p.IsPermanentFailure, &p.RetryCount, &p.RetryAfter,
		&p.FetchedAt, &p.ExpiresAt, &p.LastAccessedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get crawled page: %w", err)
	}
	return &p, nil
}

// GetFreshCrawledPage retrieves a page only if it's not stale and was successful
func (db *DB) GetFreshCrawledPage(ctx context.Context, pageURL string, maxAge time.Duration) (*CrawledPage, error) {
	page, err := db.GetCrawledPageByURL(ctx, pageURL)
	if err != nil {
		return nil, err
	}
	if page == nil {
		return nil, nil
	}

	// Check freshness
	if !page.IsFresh(maxAge) {
		return nil, nil // Stale, should re-fetch
	}

	// Only return successful pages from cache
	if page.FetchStatus != FetchStatusSuccess {
		return nil, nil
	}

	// Update last accessed time
	_ = db.TouchCrawledPage(ctx, page.ID)

	return page, nil
}

// ShouldSkipURL checks if a URL should be skipped due to previous permanent failure
func (db *DB) ShouldSkipURL(ctx context.Context, pageURL string) (bool, string, error) {
	page, err := db.GetCrawledPageByURL(ctx, pageURL)
	if err != nil {
		return false, "", err
	}
	if page == nil {
		return false, "", nil // Never tried, don't skip
	}

	// Skip permanently failed pages forever
	if page.IsPermanentFailure {
		reason := "permanent failure"
		if page.ErrorMessage != nil {
			reason = *page.ErrorMessage
		}
		return true, reason, nil
	}

	// Skip pages with retry_after in the future
	if page.RetryAfter != nil && time.Now().Before(*page.RetryAfter) {
		return true, "retry backoff", nil
	}

	return false, "", nil
}

// UpsertCrawledPage inserts or updates a crawled page (for successful fetches)
func (db *DB) UpsertCrawledPage(ctx context.Context, page *CrawledPage) error {
	// Compute content hash if we have HTML
	var contentHash *string
	if page.RawHTML != nil {
		hash := HashContent(*page.RawHTML)
		contentHash = &hash
	}

	// Set default TTL if not provided
	expiresAt := page.ExpiresAt
	if expiresAt == nil {
		t := time.Now().Add(DefaultPageCacheTTL)
		expiresAt = &t
	}

	// Default to success status
	fetchStatus := page.FetchStatus
	if fetchStatus == "" {
		fetchStatus = FetchStatusSuccess
	}

	err := db.pool.QueryRow(ctx,
		`INSERT INTO crawled_pages (company_id, url, page_type, raw_html, parsed_text, content_hash, 
		                            http_status, fetch_status, error_message, is_permanent_failure, 
		                            retry_count, fetched_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0, NOW(), $11)
		 ON CONFLICT (url) DO UPDATE SET
		     company_id = COALESCE($1, crawled_pages.company_id),
		     page_type = COALESCE($3, crawled_pages.page_type),
		     raw_html = $4,
		     parsed_text = $5,
		     content_hash = $6,
		     http_status = $7,
		     fetch_status = $8,
		     error_message = $9,
		     is_permanent_failure = $10,
		     retry_count = 0,
		     retry_after = NULL,
		     fetched_at = NOW(),
		     expires_at = $11,
		     updated_at = NOW()
		 RETURNING id, fetched_at, created_at, updated_at`,
		page.CompanyID, page.URL, page.PageType, page.RawHTML, page.ParsedText, contentHash,
		page.HTTPStatus, fetchStatus, page.ErrorMessage, page.IsPermanentFailure, expiresAt,
	).Scan(&page.ID, &page.FetchedAt, &page.CreatedAt, &page.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert crawled page: %w", err)
	}
	return nil
}

// RecordFailedFetch records a failed fetch attempt with exponential backoff
func (db *DB) RecordFailedFetch(ctx context.Context, pageURL string, httpStatus int, errorMsg string) error {
	fetchStatus := FetchStatusFromHTTP(httpStatus)
	isPermanent := IsPermanentHTTPStatus(httpStatus)

	// Calculate retry backoff: 1 min * 5^retry_count, capped at 2 hours
	// Schedule: 1 min → 5 min → 25 min → 2 hours
	// For permanent failures, set retry_after to NULL (never retry)
	_, err := db.pool.Exec(ctx,
		`INSERT INTO crawled_pages (url, http_status, fetch_status, error_message, is_permanent_failure, retry_count, retry_after, fetched_at)
		 VALUES ($1, $2, $3, $4, $5, 1, 
		         CASE WHEN $5 THEN NULL ELSE NOW() + INTERVAL '1 minute' END,
		         NOW())
		 ON CONFLICT (url) DO UPDATE SET
		     http_status = $2,
		     fetch_status = $3,
		     error_message = $4,
		     is_permanent_failure = $5 OR crawled_pages.is_permanent_failure,
		     retry_count = crawled_pages.retry_count + 1,
		     retry_after = CASE 
		         WHEN $5 OR crawled_pages.is_permanent_failure THEN NULL
		         ELSE NOW() + LEAST(
		             INTERVAL '1 minute' * POWER(5, LEAST(crawled_pages.retry_count, 3)),
		             INTERVAL '2 hours'
		         )
		     END,
		     fetched_at = NOW(),
		     updated_at = NOW()`,
		pageURL, httpStatus, fetchStatus, errorMsg, isPermanent,
	)
	if err != nil {
		return fmt.Errorf("failed to record failed fetch: %w", err)
	}
	return nil
}

// TouchCrawledPage updates the last_accessed_at timestamp
func (db *DB) TouchCrawledPage(ctx context.Context, id uuid.UUID) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE crawled_pages SET last_accessed_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to touch crawled page: %w", err)
	}
	return nil
}

// ListFreshPagesByCompany retrieves all non-expired pages for a company
func (db *DB) ListFreshPagesByCompany(ctx context.Context, companyID uuid.UUID, maxAge time.Duration) ([]CrawledPage, error) {
	cutoff := time.Now().Add(-maxAge)

	rows, err := db.pool.Query(ctx,
		`SELECT id, company_id, url, page_type, parsed_text, content_hash, 
		        http_status, fetch_status, error_message, is_permanent_failure, retry_count, retry_after,
		        fetched_at, expires_at, last_accessed_at, created_at, updated_at
		 FROM crawled_pages 
		 WHERE company_id = $1 AND fetched_at > $2 AND fetch_status = $3
		 ORDER BY fetched_at DESC`,
		companyID, cutoff, FetchStatusSuccess,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pages: %w", err)
	}
	defer rows.Close()

	var pages []CrawledPage
	for rows.Next() {
		var p CrawledPage
		// Note: raw_html intentionally omitted (large field)
		if err := rows.Scan(&p.ID, &p.CompanyID, &p.URL, &p.PageType, &p.ParsedText, &p.ContentHash,
			&p.HTTPStatus, &p.FetchStatus, &p.ErrorMessage, &p.IsPermanentFailure, &p.RetryCount, &p.RetryAfter,
			&p.FetchedAt, &p.ExpiresAt, &p.LastAccessedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan page: %w", err)
		}
		pages = append(pages, p)
	}
	return pages, nil
}

// DeleteExpiredPages removes pages that have passed their expires_at
func (db *DB) DeleteExpiredPages(ctx context.Context) (int64, error) {
	result, err := db.pool.Exec(ctx,
		`DELETE FROM crawled_pages WHERE expires_at < NOW()`,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired pages: %w", err)
	}
	return result.RowsAffected(), nil
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// normalizeDomain cleans up a domain string
func normalizeDomain(domain string) string {
	domain = strings.ToLower(domain)
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "www.")
	domain = strings.TrimSuffix(domain, "/")
	return domain
}

// ExtractDomain extracts the domain from a full URL
func ExtractDomain(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return normalizeDomain(parsed.Host), nil
}

