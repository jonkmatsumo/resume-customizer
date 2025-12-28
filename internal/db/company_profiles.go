package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// -----------------------------------------------------------------------------
// Company Profile Methods
// -----------------------------------------------------------------------------

// GetCompanyProfileByCompanyID retrieves the profile for a company
func (db *DB) GetCompanyProfileByCompanyID(ctx context.Context, companyID uuid.UUID) (*CompanyProfile, error) {
	var p CompanyProfile
	err := db.pool.QueryRow(ctx,
		`SELECT id, company_id, tone, domain_context, source_corpus, version, 
		        last_verified_at, created_at, updated_at
		 FROM company_profiles WHERE company_id = $1`,
		companyID,
	).Scan(&p.ID, &p.CompanyID, &p.Tone, &p.DomainContext, &p.SourceCorpus,
		&p.Version, &p.LastVerifiedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get company profile: %w", err)
	}

	// Load related data
	if err := db.loadProfileRelations(ctx, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// GetCompanyProfileByID retrieves a profile by its UUID
func (db *DB) GetCompanyProfileByID(ctx context.Context, id uuid.UUID) (*CompanyProfile, error) {
	var p CompanyProfile
	err := db.pool.QueryRow(ctx,
		`SELECT id, company_id, tone, domain_context, source_corpus, version, 
		        last_verified_at, created_at, updated_at
		 FROM company_profiles WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.CompanyID, &p.Tone, &p.DomainContext, &p.SourceCorpus,
		&p.Version, &p.LastVerifiedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get company profile: %w", err)
	}

	if err := db.loadProfileRelations(ctx, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// GetFreshCompanyProfile retrieves a profile only if it's not stale
func (db *DB) GetFreshCompanyProfile(ctx context.Context, companyID uuid.UUID, maxAge time.Duration) (*CompanyProfile, error) {
	profile, err := db.GetCompanyProfileByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, nil
	}

	// Check if profile is stale
	if profile.IsStale(maxAge) {
		return nil, nil // Stale, should regenerate
	}

	return profile, nil
}

// CreateCompanyProfile creates a new company profile with all related data
func (db *DB) CreateCompanyProfile(ctx context.Context, input *ProfileCreateInput) (*CompanyProfile, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert profile
	var p CompanyProfile
	now := time.Now()
	err = tx.QueryRow(ctx,
		`INSERT INTO company_profiles (company_id, tone, domain_context, source_corpus, last_verified_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (company_id) DO UPDATE SET
		     tone = $2,
		     domain_context = $3,
		     source_corpus = $4,
		     version = company_profiles.version + 1,
		     last_verified_at = $5,
		     updated_at = NOW()
		 RETURNING id, company_id, tone, domain_context, source_corpus, version, 
		           last_verified_at, created_at, updated_at`,
		input.CompanyID, input.Tone, nullIfEmpty(input.DomainContext), nullIfEmpty(input.SourceCorpus), now,
	).Scan(&p.ID, &p.CompanyID, &p.Tone, &p.DomainContext, &p.SourceCorpus,
		&p.Version, &p.LastVerifiedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create company profile: %w", err)
	}

	// Clear existing related data if updating
	if p.Version > 1 {
		_, _ = tx.Exec(ctx, "DELETE FROM company_style_rules WHERE profile_id = $1", p.ID)
		_, _ = tx.Exec(ctx, "DELETE FROM company_taboo_phrases WHERE profile_id = $1", p.ID)
		_, _ = tx.Exec(ctx, "DELETE FROM company_values WHERE profile_id = $1", p.ID)
		_, _ = tx.Exec(ctx, "DELETE FROM company_profile_sources WHERE profile_id = $1", p.ID)
	}

	// Insert style rules
	for i, rule := range input.StyleRules {
		_, err = tx.Exec(ctx,
			`INSERT INTO company_style_rules (profile_id, rule_text, priority)
			 VALUES ($1, $2, $3)`,
			p.ID, rule, len(input.StyleRules)-i, // Higher priority for first rules
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert style rule: %w", err)
		}
	}

	// Insert taboo phrases
	for _, taboo := range input.TabooPhrases {
		var reason *string
		if taboo.Reason != "" {
			reason = &taboo.Reason
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO company_taboo_phrases (profile_id, phrase, reason)
			 VALUES ($1, $2, $3)`,
			p.ID, taboo.Phrase, reason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert taboo phrase: %w", err)
		}
	}

	// Insert values
	for i, value := range input.Values {
		_, err = tx.Exec(ctx,
			`INSERT INTO company_values (profile_id, value_text, priority)
			 VALUES ($1, $2, $3)`,
			p.ID, value, len(input.Values)-i,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert company value: %w", err)
		}
	}

	// Insert evidence URLs
	for _, source := range input.EvidenceURLs {
		var sourceType *string
		if source.SourceType != "" {
			sourceType = &source.SourceType
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO company_profile_sources (profile_id, crawled_page_id, url, source_type)
			 VALUES ($1, $2, $3, $4)`,
			p.ID, source.CrawledPageID, source.URL, sourceType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert profile source: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load relations for return
	if err := db.loadProfileRelations(ctx, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// UpdateProfileVerification updates the last_verified_at timestamp
func (db *DB) UpdateProfileVerification(ctx context.Context, profileID uuid.UUID) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE company_profiles SET last_verified_at = NOW(), updated_at = NOW() WHERE id = $1`,
		profileID,
	)
	if err != nil {
		return fmt.Errorf("failed to update profile verification: %w", err)
	}
	return nil
}

// DeleteCompanyProfile removes a profile and all related data (cascades)
func (db *DB) DeleteCompanyProfile(ctx context.Context, profileID uuid.UUID) error {
	_, err := db.pool.Exec(ctx, "DELETE FROM company_profiles WHERE id = $1", profileID)
	if err != nil {
		return fmt.Errorf("failed to delete company profile: %w", err)
	}
	return nil
}

// loadProfileRelations loads style rules, taboo phrases, values, and sources
func (db *DB) loadProfileRelations(ctx context.Context, p *CompanyProfile) error {
	// Load style rules
	rows, err := db.pool.Query(ctx,
		`SELECT rule_text FROM company_style_rules 
		 WHERE profile_id = $1 ORDER BY priority DESC`,
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to load style rules: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rule string
		if err := rows.Scan(&rule); err != nil {
			return err
		}
		p.StyleRules = append(p.StyleRules, rule)
	}

	// Load taboo phrases
	rows, err = db.pool.Query(ctx,
		`SELECT phrase FROM company_taboo_phrases WHERE profile_id = $1`,
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to load taboo phrases: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var phrase string
		if err := rows.Scan(&phrase); err != nil {
			return err
		}
		p.TabooPhrases = append(p.TabooPhrases, phrase)
	}

	// Load values
	rows, err = db.pool.Query(ctx,
		`SELECT value_text FROM company_values 
		 WHERE profile_id = $1 ORDER BY priority DESC`,
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to load company values: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return err
		}
		p.Values = append(p.Values, value)
	}

	// Load evidence URLs
	rows, err = db.pool.Query(ctx,
		`SELECT url FROM company_profile_sources WHERE profile_id = $1`,
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to load profile sources: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return err
		}
		p.EvidenceURLs = append(p.EvidenceURLs, url)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Brand Signal Methods
// -----------------------------------------------------------------------------

// CreateBrandSignal stores a brand signal extracted from a crawled page
func (db *DB) CreateBrandSignal(ctx context.Context, signal *BrandSignal) error {
	keyPointsJSON, err := json.Marshal(signal.KeyPoints)
	if err != nil {
		return fmt.Errorf("failed to marshal key points: %w", err)
	}

	valuesJSON, err := json.Marshal(signal.ExtractedValues)
	if err != nil {
		return fmt.Errorf("failed to marshal values: %w", err)
	}

	err = db.pool.QueryRow(ctx,
		`INSERT INTO brand_signals (crawled_page_id, signal_type, key_points, extracted_values, 
		                            raw_excerpt, confidence_score)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		signal.CrawledPageID, signal.SignalType, keyPointsJSON, valuesJSON,
		signal.RawExcerpt, signal.ConfidenceScore,
	).Scan(&signal.ID, &signal.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create brand signal: %w", err)
	}
	return nil
}

// GetBrandSignalsByPage retrieves all signals for a crawled page
func (db *DB) GetBrandSignalsByPage(ctx context.Context, crawledPageID uuid.UUID) ([]BrandSignal, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT bs.id, bs.crawled_page_id, cp.url, bs.signal_type, bs.key_points, 
		        bs.extracted_values, bs.raw_excerpt, bs.confidence_score, bs.created_at
		 FROM brand_signals bs
		 JOIN crawled_pages cp ON bs.crawled_page_id = cp.id
		 WHERE bs.crawled_page_id = $1
		 ORDER BY bs.created_at`,
		crawledPageID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get brand signals: %w", err)
	}
	defer rows.Close()

	var signals []BrandSignal
	for rows.Next() {
		var s BrandSignal
		var keyPointsJSON, valuesJSON []byte
		if err := rows.Scan(&s.ID, &s.CrawledPageID, &s.URL, &s.SignalType, &keyPointsJSON,
			&valuesJSON, &s.RawExcerpt, &s.ConfidenceScore, &s.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(keyPointsJSON, &s.KeyPoints)
		_ = json.Unmarshal(valuesJSON, &s.ExtractedValues)
		signals = append(signals, s)
	}
	return signals, nil
}

// GetBrandSignalsByCompany retrieves all signals for a company (via crawled pages)
func (db *DB) GetBrandSignalsByCompany(ctx context.Context, companyID uuid.UUID) ([]BrandSignal, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT bs.id, bs.crawled_page_id, cp.url, bs.signal_type, bs.key_points, 
		        bs.extracted_values, bs.raw_excerpt, bs.confidence_score, bs.created_at
		 FROM brand_signals bs
		 JOIN crawled_pages cp ON bs.crawled_page_id = cp.id
		 WHERE cp.company_id = $1
		 ORDER BY bs.confidence_score DESC NULLS LAST, bs.created_at`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get brand signals for company: %w", err)
	}
	defer rows.Close()

	var signals []BrandSignal
	for rows.Next() {
		var s BrandSignal
		var keyPointsJSON, valuesJSON []byte
		if err := rows.Scan(&s.ID, &s.CrawledPageID, &s.URL, &s.SignalType, &keyPointsJSON,
			&valuesJSON, &s.RawExcerpt, &s.ConfidenceScore, &s.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(keyPointsJSON, &s.KeyPoints)
		_ = json.Unmarshal(valuesJSON, &s.ExtractedValues)
		signals = append(signals, s)
	}
	return signals, nil
}

// DeleteBrandSignalsForPage removes all signals for a crawled page
func (db *DB) DeleteBrandSignalsForPage(ctx context.Context, crawledPageID uuid.UUID) error {
	_, err := db.pool.Exec(ctx, "DELETE FROM brand_signals WHERE crawled_page_id = $1", crawledPageID)
	if err != nil {
		return fmt.Errorf("failed to delete brand signals: %w", err)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Helper Methods
// -----------------------------------------------------------------------------

// GetProfileWithCompany retrieves a profile with company details
func (db *DB) GetProfileWithCompany(ctx context.Context, companyID uuid.UUID) (*CompanyProfile, error) {
	profile, err := db.GetCompanyProfileByCompanyID(ctx, companyID)
	if err != nil || profile == nil {
		return profile, err
	}

	company, err := db.GetCompanyByID(ctx, companyID)
	if err != nil {
		return nil, err
	}
	profile.Company = company

	return profile, nil
}

// ListStaleProfiles returns profiles that haven't been verified recently
func (db *DB) ListStaleProfiles(ctx context.Context, maxAge time.Duration) ([]CompanyProfile, error) {
	cutoff := time.Now().Add(-maxAge)

	rows, err := db.pool.Query(ctx,
		`SELECT id, company_id, tone, domain_context, version, 
		        last_verified_at, created_at, updated_at
		 FROM company_profiles 
		 WHERE last_verified_at IS NULL OR last_verified_at < $1
		 ORDER BY last_verified_at NULLS FIRST`,
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list stale profiles: %w", err)
	}
	defer rows.Close()

	var profiles []CompanyProfile
	for rows.Next() {
		var p CompanyProfile
		if err := rows.Scan(&p.ID, &p.CompanyID, &p.Tone, &p.DomainContext,
			&p.Version, &p.LastVerifiedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// nullIfEmpty returns nil if the string is empty, otherwise a pointer to the string
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
