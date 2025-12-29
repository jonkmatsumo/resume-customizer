package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// -----------------------------------------------------------------------------
// Job Posting Methods
// -----------------------------------------------------------------------------

// GetJobPostingByURL retrieves a job posting by its URL
func (db *DB) GetJobPostingByURL(ctx context.Context, url string) (*JobPosting, error) {
	var p JobPosting
	var adminInfoJSON, linksJSON []byte

	err := db.pool.QueryRow(ctx,
		`SELECT id, company_id, url, role_title, platform, raw_html, cleaned_text,
		        content_hash, about_company, admin_info, extracted_links,
		        http_status, fetch_status, error_message, fetched_at, expires_at,
		        last_accessed_at, created_at, updated_at
		 FROM job_postings WHERE url = $1`,
		url,
	).Scan(&p.ID, &p.CompanyID, &p.URL, &p.RoleTitle, &p.Platform, &p.RawHTML,
		&p.CleanedText, &p.ContentHash, &p.AboutCompany, &adminInfoJSON, &linksJSON,
		&p.HTTPStatus, &p.FetchStatus, &p.ErrorMessage, &p.FetchedAt, &p.ExpiresAt,
		&p.LastAccessed, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get job posting: %w", err)
	}

	// Parse JSONB fields
	if adminInfoJSON != nil {
		_ = json.Unmarshal(adminInfoJSON, &p.AdminInfo)
	}
	if linksJSON != nil {
		_ = json.Unmarshal(linksJSON, &p.ExtractedLinks)
	}

	return &p, nil
}

// GetJobPostingByID retrieves a job posting by its ID
func (db *DB) GetJobPostingByID(ctx context.Context, id uuid.UUID) (*JobPosting, error) {
	var p JobPosting
	var adminInfoJSON, linksJSON []byte

	err := db.pool.QueryRow(ctx,
		`SELECT id, company_id, url, role_title, platform, raw_html, cleaned_text,
		        content_hash, about_company, admin_info, extracted_links,
		        http_status, fetch_status, error_message, fetched_at, expires_at,
		        last_accessed_at, created_at, updated_at
		 FROM job_postings WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.CompanyID, &p.URL, &p.RoleTitle, &p.Platform, &p.RawHTML,
		&p.CleanedText, &p.ContentHash, &p.AboutCompany, &adminInfoJSON, &linksJSON,
		&p.HTTPStatus, &p.FetchStatus, &p.ErrorMessage, &p.FetchedAt, &p.ExpiresAt,
		&p.LastAccessed, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get job posting: %w", err)
	}

	if adminInfoJSON != nil {
		_ = json.Unmarshal(adminInfoJSON, &p.AdminInfo)
	}
	if linksJSON != nil {
		_ = json.Unmarshal(linksJSON, &p.ExtractedLinks)
	}

	return &p, nil
}

// GetFreshJobPosting retrieves a posting only if it's not expired
func (db *DB) GetFreshJobPosting(ctx context.Context, url string) (*JobPosting, error) {
	posting, err := db.GetJobPostingByURL(ctx, url)
	if err != nil {
		return nil, err
	}
	if posting == nil {
		return nil, nil
	}

	if posting.IsExpired() {
		return nil, nil // Expired, should re-fetch
	}

	// Update last accessed
	_, _ = db.pool.Exec(ctx,
		"UPDATE job_postings SET last_accessed_at = NOW() WHERE id = $1",
		posting.ID)

	return posting, nil
}

// UpsertJobPosting creates or updates a job posting
func (db *DB) UpsertJobPosting(ctx context.Context, input *JobPostingCreateInput) (*JobPosting, error) {
	var p JobPosting

	// Prepare JSONB fields
	var adminInfoJSON, linksJSON []byte
	var err error
	if input.AdminInfo != nil {
		adminInfoJSON, err = json.Marshal(input.AdminInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal admin info: %w", err)
		}
	}
	if len(input.Links) > 0 {
		linksJSON, err = json.Marshal(input.Links)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal links: %w", err)
		}
	}

	// Compute content hash
	contentHash := HashJobContent(input.CleanedText)

	// Set expiry
	expiresAt := time.Now().Add(DefaultJobPostingCacheTTL)

	err = db.pool.QueryRow(ctx,
		`INSERT INTO job_postings (company_id, url, role_title, platform, raw_html, 
		                           cleaned_text, content_hash, about_company, admin_info,
		                           extracted_links, http_status, fetch_status, fetched_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'success', NOW(), $12)
		 ON CONFLICT (url) DO UPDATE SET
		     company_id = COALESCE($1, job_postings.company_id),
		     role_title = $3,
		     platform = $4,
		     raw_html = $5,
		     cleaned_text = $6,
		     content_hash = $7,
		     about_company = $8,
		     admin_info = $9,
		     extracted_links = $10,
		     http_status = $11,
		     fetch_status = 'success',
		     error_message = NULL,
		     fetched_at = NOW(),
		     expires_at = $12,
		     updated_at = NOW()
		 RETURNING id, company_id, url, role_title, platform, content_hash, fetch_status,
		           fetched_at, expires_at, created_at, updated_at`,
		input.CompanyID, input.URL, input.RoleTitle, input.Platform, input.RawHTML,
		input.CleanedText, contentHash, input.AboutCompany, adminInfoJSON, linksJSON,
		input.HTTPStatus, expiresAt,
	).Scan(&p.ID, &p.CompanyID, &p.URL, &p.RoleTitle, &p.Platform, &p.ContentHash,
		&p.FetchStatus, &p.FetchedAt, &p.ExpiresAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert job posting: %w", err)
	}

	return &p, nil
}

// RecordFailedJobFetch records a failed fetch attempt
func (db *DB) RecordFailedJobFetch(ctx context.Context, url string, httpStatus *int, errorMsg string) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO job_postings (url, http_status, fetch_status, error_message, fetched_at, expires_at)
		 VALUES ($1, $2, 'error', $3, NOW(), NOW() + INTERVAL '1 hour')
		 ON CONFLICT (url) DO UPDATE SET
		     http_status = $2,
		     fetch_status = 'error',
		     error_message = $3,
		     fetched_at = NOW(),
		     updated_at = NOW()`,
		url, httpStatus, errorMsg,
	)
	if err != nil {
		return fmt.Errorf("failed to record failed job fetch: %w", err)
	}
	return nil
}

// ListJobPostingsByCompany retrieves all postings for a company
func (db *DB) ListJobPostingsByCompany(ctx context.Context, companyID uuid.UUID) ([]JobPosting, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, company_id, url, role_title, platform, content_hash, fetch_status,
		        fetched_at, expires_at, created_at, updated_at
		 FROM job_postings 
		 WHERE company_id = $1
		 ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list job postings: %w", err)
	}
	defer rows.Close()

	var postings []JobPosting
	for rows.Next() {
		var p JobPosting
		if err := rows.Scan(&p.ID, &p.CompanyID, &p.URL, &p.RoleTitle, &p.Platform,
			&p.ContentHash, &p.FetchStatus, &p.FetchedAt, &p.ExpiresAt,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		postings = append(postings, p)
	}
	return postings, nil
}

// ListJobPostingsOptions contains filters for listing job postings
type ListJobPostingsOptions struct {
	Platform  *string    // Filter by platform (greenhouse, lever, etc.)
	CompanyID *uuid.UUID // Filter by company
	Limit     int        // Pagination limit
	Offset    int        // Pagination offset
}

// ListJobPostings lists job postings with optional filters and pagination
func (db *DB) ListJobPostings(ctx context.Context, opts ListJobPostingsOptions) ([]JobPosting, int, error) {
	// Build WHERE clause dynamically
	var conditions []string
	var args []interface{}
	argIndex := 1

	if opts.Platform != nil && *opts.Platform != "" {
		conditions = append(conditions, fmt.Sprintf("platform = $%d", argIndex))
		args = append(args, *opts.Platform)
		argIndex++
	}

	if opts.CompanyID != nil {
		conditions = append(conditions, fmt.Sprintf("company_id = $%d", argIndex))
		args = append(args, *opts.CompanyID)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM job_postings %s", whereClause)
	var total int
	err := db.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count job postings: %w", err)
	}

	// Get postings
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(
		`SELECT id, company_id, url, role_title, platform, cleaned_text,
		        content_hash, about_company, admin_info, extracted_links,
		        http_status, fetch_status, error_message, fetched_at, expires_at,
		        last_accessed_at, created_at, updated_at
		 FROM job_postings %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1,
	)

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list job postings: %w", err)
	}
	defer rows.Close()

	var postings []JobPosting
	for rows.Next() {
		var p JobPosting
		var adminInfoJSON, linksJSON []byte
		var companyID *uuid.UUID

		err := rows.Scan(
			&p.ID, &companyID, &p.URL, &p.RoleTitle, &p.Platform,
			&p.CleanedText, &p.ContentHash, &p.AboutCompany, &adminInfoJSON, &linksJSON,
			&p.HTTPStatus, &p.FetchStatus, &p.ErrorMessage, &p.FetchedAt, &p.ExpiresAt,
			&p.LastAccessed, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		p.CompanyID = companyID

		// Parse JSONB fields
		if adminInfoJSON != nil {
			p.AdminInfo = &AdminInfo{}
			_ = json.Unmarshal(adminInfoJSON, p.AdminInfo)
		}
		if linksJSON != nil {
			_ = json.Unmarshal(linksJSON, &p.ExtractedLinks)
		}

		postings = append(postings, p)
	}

	return postings, total, nil
}

// -----------------------------------------------------------------------------
// Job Profile Methods
// -----------------------------------------------------------------------------

// GetJobProfileByPostingID retrieves the profile for a posting
func (db *DB) GetJobProfileByPostingID(ctx context.Context, postingID uuid.UUID) (*JobProfile, error) {
	var p JobProfile
	var evalSignalsJSON, eduFieldsJSON []byte

	err := db.pool.QueryRow(ctx,
		`SELECT id, posting_id, company_name, role_title,
		        eval_latency, eval_reliability, eval_ownership, eval_scale, eval_collaboration,
		        eval_signals_raw, education_min_degree, education_preferred_fields,
		        education_is_required, education_evidence, parsed_at, parser_version,
		        created_at, updated_at
		 FROM job_profiles WHERE posting_id = $1`,
		postingID,
	).Scan(&p.ID, &p.PostingID, &p.CompanyName, &p.RoleTitle,
		&p.EvalLatency, &p.EvalReliability, &p.EvalOwnership, &p.EvalScale, &p.EvalCollaboration,
		&evalSignalsJSON, &p.EducationMinDegree, &eduFieldsJSON,
		&p.EducationIsRequired, &p.EducationEvidence, &p.ParsedAt, &p.ParserVersion,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get job profile: %w", err)
	}

	// Parse JSONB fields
	if evalSignalsJSON != nil {
		_ = json.Unmarshal(evalSignalsJSON, &p.EvalSignalsRaw)
	}
	if eduFieldsJSON != nil {
		_ = json.Unmarshal(eduFieldsJSON, &p.EducationPreferredFields)
	}

	// Load relations
	if err := db.loadJobProfileRelations(ctx, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// GetJobProfileByID retrieves a profile by its ID
func (db *DB) GetJobProfileByID(ctx context.Context, id uuid.UUID) (*JobProfile, error) {
	var p JobProfile
	var evalSignalsJSON, eduFieldsJSON []byte

	err := db.pool.QueryRow(ctx,
		`SELECT id, posting_id, company_name, role_title,
		        eval_latency, eval_reliability, eval_ownership, eval_scale, eval_collaboration,
		        eval_signals_raw, education_min_degree, education_preferred_fields,
		        education_is_required, education_evidence, parsed_at, parser_version,
		        created_at, updated_at
		 FROM job_profiles WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.PostingID, &p.CompanyName, &p.RoleTitle,
		&p.EvalLatency, &p.EvalReliability, &p.EvalOwnership, &p.EvalScale, &p.EvalCollaboration,
		&evalSignalsJSON, &p.EducationMinDegree, &eduFieldsJSON,
		&p.EducationIsRequired, &p.EducationEvidence, &p.ParsedAt, &p.ParserVersion,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get job profile: %w", err)
	}

	if evalSignalsJSON != nil {
		_ = json.Unmarshal(evalSignalsJSON, &p.EvalSignalsRaw)
	}
	if eduFieldsJSON != nil {
		_ = json.Unmarshal(eduFieldsJSON, &p.EducationPreferredFields)
	}

	if err := db.loadJobProfileRelations(ctx, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// CreateJobProfile creates a new job profile with all related data
func (db *DB) CreateJobProfile(ctx context.Context, input *JobProfileCreateInput) (*JobProfile, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil && rErr != pgx.ErrTxClosed {
			// Log rollback error but don't override the original error
			fmt.Printf("Rollback error: %v\n", rErr)
		}
	}()

	// Prepare JSONB fields
	var evalSignalsJSON, eduFieldsJSON []byte
	if input.EvalSignalsRaw != nil {
		evalSignalsJSON, _ = json.Marshal(input.EvalSignalsRaw)
	}
	if len(input.EducationPreferredFields) > 0 {
		eduFieldsJSON, _ = json.Marshal(input.EducationPreferredFields)
	}

	// Insert or update profile
	var p JobProfile
	err = tx.QueryRow(ctx,
		`INSERT INTO job_profiles (posting_id, company_name, role_title,
		                           eval_latency, eval_reliability, eval_ownership, eval_scale, eval_collaboration,
		                           eval_signals_raw, education_min_degree, education_preferred_fields,
		                           education_is_required, education_evidence, parser_version, parsed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		 ON CONFLICT (posting_id) DO UPDATE SET
		     company_name = $2,
		     role_title = $3,
		     eval_latency = $4,
		     eval_reliability = $5,
		     eval_ownership = $6,
		     eval_scale = $7,
		     eval_collaboration = $8,
		     eval_signals_raw = $9,
		     education_min_degree = $10,
		     education_preferred_fields = $11,
		     education_is_required = $12,
		     education_evidence = $13,
		     parser_version = $14,
		     parsed_at = NOW(),
		     updated_at = NOW()
		 RETURNING id, posting_id, company_name, role_title, parsed_at, created_at, updated_at`,
		input.PostingID, input.CompanyName, input.RoleTitle,
		input.EvalLatency, input.EvalReliability, input.EvalOwnership, input.EvalScale, input.EvalCollaboration,
		evalSignalsJSON, nullIfEmpty(input.EducationMinDegree), eduFieldsJSON,
		input.EducationIsRequired, nullIfEmpty(input.EducationEvidence), nullIfEmpty(input.ParserVersion),
	).Scan(&p.ID, &p.PostingID, &p.CompanyName, &p.RoleTitle, &p.ParsedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create job profile: %w", err)
	}

	// Clear existing related data (for upsert)
	_, _ = tx.Exec(ctx, "DELETE FROM job_responsibilities WHERE job_profile_id = $1", p.ID)
	_, _ = tx.Exec(ctx, "DELETE FROM job_requirements WHERE job_profile_id = $1", p.ID)
	_, _ = tx.Exec(ctx, "DELETE FROM job_keywords WHERE job_profile_id = $1", p.ID)

	// Insert responsibilities
	for i, resp := range input.Responsibilities {
		_, err = tx.Exec(ctx,
			`INSERT INTO job_responsibilities (job_profile_id, text, ordinal)
			 VALUES ($1, $2, $3)`,
			p.ID, resp, i+1,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert responsibility: %w", err)
		}
	}

	// Insert hard requirements
	for i, req := range input.HardRequirements {
		_, err = tx.Exec(ctx,
			`INSERT INTO job_requirements (job_profile_id, requirement_type, skill, level, evidence, ordinal)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			p.ID, RequirementTypeHard, req.Skill, nullIfEmpty(req.Level), nullIfEmpty(req.Evidence), i+1,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert hard requirement: %w", err)
		}
	}

	// Insert nice-to-haves
	for i, req := range input.NiceToHaves {
		_, err = tx.Exec(ctx,
			`INSERT INTO job_requirements (job_profile_id, requirement_type, skill, level, evidence, ordinal)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			p.ID, RequirementTypeNiceToHave, req.Skill, nullIfEmpty(req.Level), nullIfEmpty(req.Evidence), i+1,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert nice-to-have: %w", err)
		}
	}

	// Insert keywords
	for _, kw := range input.Keywords {
		_, err = tx.Exec(ctx,
			`INSERT INTO job_keywords (job_profile_id, keyword, keyword_normalized)
			 VALUES ($1, $2, $3)`,
			p.ID, kw, NormalizeKeyword(kw),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert keyword: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load relations for return
	if err := db.loadJobProfileRelations(ctx, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// loadJobProfileRelations loads responsibilities, requirements, and keywords
func (db *DB) loadJobProfileRelations(ctx context.Context, p *JobProfile) error {
	// Load responsibilities
	rows, err := db.pool.Query(ctx,
		`SELECT text FROM job_responsibilities 
		 WHERE job_profile_id = $1 ORDER BY ordinal`,
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to load responsibilities: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return err
		}
		p.Responsibilities = append(p.Responsibilities, text)
	}

	// Load hard requirements
	rows, err = db.pool.Query(ctx,
		`SELECT id, job_profile_id, requirement_type, skill, level, evidence, ordinal, created_at
		 FROM job_requirements 
		 WHERE job_profile_id = $1 AND requirement_type = $2
		 ORDER BY ordinal`,
		p.ID, RequirementTypeHard,
	)
	if err != nil {
		return fmt.Errorf("failed to load hard requirements: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var r JobRequirement
		if err := rows.Scan(&r.ID, &r.JobProfileID, &r.RequirementType, &r.Skill,
			&r.Level, &r.Evidence, &r.Ordinal, &r.CreatedAt); err != nil {
			return err
		}
		p.HardRequirements = append(p.HardRequirements, r)
	}

	// Load nice-to-haves
	rows, err = db.pool.Query(ctx,
		`SELECT id, job_profile_id, requirement_type, skill, level, evidence, ordinal, created_at
		 FROM job_requirements 
		 WHERE job_profile_id = $1 AND requirement_type = $2
		 ORDER BY ordinal`,
		p.ID, RequirementTypeNiceToHave,
	)
	if err != nil {
		return fmt.Errorf("failed to load nice-to-haves: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var r JobRequirement
		if err := rows.Scan(&r.ID, &r.JobProfileID, &r.RequirementType, &r.Skill,
			&r.Level, &r.Evidence, &r.Ordinal, &r.CreatedAt); err != nil {
			return err
		}
		p.NiceToHaves = append(p.NiceToHaves, r)
	}

	// Load keywords
	rows, err = db.pool.Query(ctx,
		`SELECT keyword FROM job_keywords WHERE job_profile_id = $1`,
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to load keywords: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var kw string
		if err := rows.Scan(&kw); err != nil {
			return err
		}
		p.Keywords = append(p.Keywords, kw)
	}

	return nil
}

// DeleteJobProfile removes a profile and all related data (cascades)
func (db *DB) DeleteJobProfile(ctx context.Context, id uuid.UUID) error {
	_, err := db.pool.Exec(ctx, "DELETE FROM job_profiles WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete job profile: %w", err)
	}
	return nil
}

// GetRequirementsByProfileID retrieves all requirements (hard + nice-to-have) for a job profile
func (db *DB) GetRequirementsByProfileID(ctx context.Context, profileID uuid.UUID) ([]JobRequirement, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, job_profile_id, requirement_type, skill, level, evidence, ordinal, created_at
		 FROM job_requirements
		 WHERE job_profile_id = $1
		 ORDER BY requirement_type, ordinal`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get requirements: %w", err)
	}
	defer rows.Close()

	var requirements []JobRequirement
	for rows.Next() {
		var r JobRequirement
		if err := rows.Scan(&r.ID, &r.JobProfileID, &r.RequirementType, &r.Skill,
			&r.Level, &r.Evidence, &r.Ordinal, &r.CreatedAt); err != nil {
			return nil, err
		}
		requirements = append(requirements, r)
	}

	return requirements, nil
}

// GetResponsibilitiesByProfileID retrieves all responsibilities for a job profile
func (db *DB) GetResponsibilitiesByProfileID(ctx context.Context, profileID uuid.UUID) ([]JobResponsibility, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, job_profile_id, text, ordinal, created_at
		 FROM job_responsibilities
		 WHERE job_profile_id = $1
		 ORDER BY ordinal`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get responsibilities: %w", err)
	}
	defer rows.Close()

	var responsibilities []JobResponsibility
	for rows.Next() {
		var r JobResponsibility
		if err := rows.Scan(&r.ID, &r.JobProfileID, &r.Text, &r.Ordinal, &r.CreatedAt); err != nil {
			return nil, err
		}
		responsibilities = append(responsibilities, r)
	}

	return responsibilities, nil
}

// GetKeywordsByProfileID retrieves all keywords for a job profile
func (db *DB) GetKeywordsByProfileID(ctx context.Context, profileID uuid.UUID) ([]JobKeyword, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, job_profile_id, keyword, keyword_normalized, source, created_at
		 FROM job_keywords
		 WHERE job_profile_id = $1
		 ORDER BY keyword`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get keywords: %w", err)
	}
	defer rows.Close()

	var keywords []JobKeyword
	for rows.Next() {
		var k JobKeyword
		if err := rows.Scan(&k.ID, &k.JobProfileID, &k.Keyword, &k.KeywordNormalized,
			&k.Source, &k.CreatedAt); err != nil {
			return nil, err
		}
		keywords = append(keywords, k)
	}

	return keywords, nil
}

// -----------------------------------------------------------------------------
// Query Methods
// -----------------------------------------------------------------------------

// FindJobsBySkill finds all jobs requiring a specific skill
func (db *DB) FindJobsBySkill(ctx context.Context, skill string, requirementType string) ([]JobProfile, error) {
	skillPattern := "%" + strings.ToLower(skill) + "%"

	rows, err := db.pool.Query(ctx,
		`SELECT DISTINCT jp.id, jp.posting_id, jp.company_name, jp.role_title,
		        jp.parsed_at, jp.created_at
		 FROM job_profiles jp
		 JOIN job_requirements jr ON jr.job_profile_id = jp.id
		 WHERE LOWER(jr.skill) LIKE $1
		   AND ($2 = '' OR jr.requirement_type = $2)
		 ORDER BY jp.created_at DESC`,
		skillPattern, requirementType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find jobs by skill: %w", err)
	}
	defer rows.Close()

	var profiles []JobProfile
	for rows.Next() {
		var p JobProfile
		if err := rows.Scan(&p.ID, &p.PostingID, &p.CompanyName, &p.RoleTitle,
			&p.ParsedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// FindJobsByKeyword finds all jobs with a specific keyword
func (db *DB) FindJobsByKeyword(ctx context.Context, keyword string) ([]JobProfile, error) {
	normalized := NormalizeKeyword(keyword)

	rows, err := db.pool.Query(ctx,
		`SELECT DISTINCT jp.id, jp.posting_id, jp.company_name, jp.role_title,
		        jp.parsed_at, jp.created_at
		 FROM job_profiles jp
		 JOIN job_keywords jk ON jk.job_profile_id = jp.id
		 WHERE jk.keyword_normalized = $1
		 ORDER BY jp.created_at DESC`,
		normalized,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find jobs by keyword: %w", err)
	}
	defer rows.Close()

	var profiles []JobProfile
	for rows.Next() {
		var p JobProfile
		if err := rows.Scan(&p.ID, &p.PostingID, &p.CompanyName, &p.RoleTitle,
			&p.ParsedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}
