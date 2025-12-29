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
// Research Session Methods
// -----------------------------------------------------------------------------

// CreateResearchSession creates a new research session
func (db *DB) CreateResearchSession(ctx context.Context, input *ResearchSessionInput) (*ResearchSession, error) {
	pagesLimit := input.PagesLimit
	if pagesLimit <= 0 {
		pagesLimit = DefaultPagesLimit
	}

	var session ResearchSession
	err := db.pool.QueryRow(ctx,
		`INSERT INTO research_sessions (company_id, run_id, company_name, domain, pages_limit, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, company_id, run_id, company_name, domain, status, error_message,
		           pages_crawled, pages_limit, corpus_text, created_at, started_at, completed_at`,
		input.CompanyID, input.RunID, input.CompanyName, nullIfEmpty(input.Domain),
		pagesLimit, ResearchStatusPending,
	).Scan(&session.ID, &session.CompanyID, &session.RunID, &session.CompanyName,
		&session.Domain, &session.Status, &session.ErrorMessage, &session.PagesCrawled,
		&session.PagesLimit, &session.CorpusText, &session.CreatedAt, &session.StartedAt,
		&session.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create research session: %w", err)
	}

	return &session, nil
}

// GetResearchSessionByID retrieves a research session by ID
func (db *DB) GetResearchSessionByID(ctx context.Context, id uuid.UUID) (*ResearchSession, error) {
	var session ResearchSession
	err := db.pool.QueryRow(ctx,
		`SELECT id, company_id, run_id, company_name, domain, status, error_message,
		        pages_crawled, pages_limit, corpus_text, created_at, started_at, completed_at
		 FROM research_sessions WHERE id = $1`,
		id,
	).Scan(&session.ID, &session.CompanyID, &session.RunID, &session.CompanyName,
		&session.Domain, &session.Status, &session.ErrorMessage, &session.PagesCrawled,
		&session.PagesLimit, &session.CorpusText, &session.CreatedAt, &session.StartedAt,
		&session.CompletedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get research session: %w", err)
	}

	return &session, nil
}

// GetRecentResearchSession finds a recent completed session for a company (within maxAge)
func (db *DB) GetRecentResearchSession(ctx context.Context, companyID uuid.UUID, maxAge time.Duration) (*ResearchSession, error) {
	cutoff := time.Now().Add(-maxAge)

	var session ResearchSession
	err := db.pool.QueryRow(ctx,
		`SELECT id, company_id, run_id, company_name, domain, status, error_message,
		        pages_crawled, pages_limit, corpus_text, created_at, started_at, completed_at
		 FROM research_sessions
		 WHERE company_id = $1 AND status = $2 AND completed_at > $3
		 ORDER BY completed_at DESC LIMIT 1`,
		companyID, ResearchStatusCompleted, cutoff,
	).Scan(&session.ID, &session.CompanyID, &session.RunID, &session.CompanyName,
		&session.Domain, &session.Status, &session.ErrorMessage, &session.PagesCrawled,
		&session.PagesLimit, &session.CorpusText, &session.CreatedAt, &session.StartedAt,
		&session.CompletedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get recent research session: %w", err)
	}

	return &session, nil
}

// UpdateResearchSessionStatus updates the status of a research session
func (db *DB) UpdateResearchSessionStatus(ctx context.Context, id uuid.UUID, status string, errorMsg string) error {
	var err error
	now := time.Now()

	switch status {
	case ResearchStatusInProgress:
		_, err = db.pool.Exec(ctx,
			`UPDATE research_sessions SET status = $1, started_at = $2 WHERE id = $3`,
			status, now, id)
	case ResearchStatusCompleted:
		_, err = db.pool.Exec(ctx,
			`UPDATE research_sessions SET status = $1, completed_at = $2 WHERE id = $3`,
			status, now, id)
	case ResearchStatusFailed:
		_, err = db.pool.Exec(ctx,
			`UPDATE research_sessions SET status = $1, error_message = $2, completed_at = $3 WHERE id = $4`,
			status, nullIfEmpty(errorMsg), now, id)
	default:
		_, err = db.pool.Exec(ctx,
			`UPDATE research_sessions SET status = $1 WHERE id = $2`,
			status, id)
	}

	if err != nil {
		return fmt.Errorf("failed to update research session status: %w", err)
	}
	return nil
}

// UpdateResearchSessionProgress updates the pages crawled count and corpus
func (db *DB) UpdateResearchSessionProgress(ctx context.Context, id uuid.UUID, pagesCrawled int, corpus string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE research_sessions SET pages_crawled = $1, corpus_text = $2 WHERE id = $3`,
		pagesCrawled, nullIfEmpty(corpus), id)
	if err != nil {
		return fmt.Errorf("failed to update research session progress: %w", err)
	}
	return nil
}

// ListResearchSessionsByCompany retrieves all research sessions for a company
func (db *DB) ListResearchSessionsByCompany(ctx context.Context, companyID uuid.UUID) ([]ResearchSession, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, company_id, run_id, company_name, domain, status, error_message,
		        pages_crawled, pages_limit, corpus_text, created_at, started_at, completed_at
		 FROM research_sessions
		 WHERE company_id = $1
		 ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list research sessions: %w", err)
	}
	defer rows.Close()

	var sessions []ResearchSession
	for rows.Next() {
		var s ResearchSession
		if err := rows.Scan(&s.ID, &s.CompanyID, &s.RunID, &s.CompanyName, &s.Domain,
			&s.Status, &s.ErrorMessage, &s.PagesCrawled, &s.PagesLimit, &s.CorpusText,
			&s.CreatedAt, &s.StartedAt, &s.CompletedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// DeleteResearchSession deletes a research session and cascades to frontier/signals
func (db *DB) DeleteResearchSession(ctx context.Context, id uuid.UUID) error {
	_, err := db.pool.Exec(ctx, "DELETE FROM research_sessions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete research session: %w", err)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Research Frontier Methods
// -----------------------------------------------------------------------------

// AddFrontierURLs adds URLs to the research frontier
func (db *DB) AddFrontierURLs(ctx context.Context, sessionID uuid.UUID, urls []FrontierURLInput) ([]FrontierURL, error) {
	var result []FrontierURL

	for _, input := range urls {
		var fu FrontierURL
		err := db.pool.QueryRow(ctx,
			`INSERT INTO research_frontier (session_id, url, priority, page_type, reason, status)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id, session_id, url, priority, page_type, reason, status, skip_reason,
			           error_message, crawled_page_id, created_at, fetched_at`,
			sessionID, input.URL, input.Priority, nullIfEmpty(input.PageType),
			nullIfEmpty(input.Reason), FrontierStatusPending,
		).Scan(&fu.ID, &fu.SessionID, &fu.URL, &fu.Priority, &fu.PageType, &fu.Reason,
			&fu.Status, &fu.SkipReason, &fu.ErrorMessage, &fu.CrawledPageID, &fu.CreatedAt,
			&fu.FetchedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to add frontier URL: %w", err)
		}
		result = append(result, fu)
	}

	return result, nil
}

// GetPendingFrontierURLs retrieves pending URLs for a session ordered by priority
func (db *DB) GetPendingFrontierURLs(ctx context.Context, sessionID uuid.UUID) ([]FrontierURL, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, session_id, url, priority, page_type, reason, status, skip_reason,
		        error_message, crawled_page_id, created_at, fetched_at
		 FROM research_frontier
		 WHERE session_id = $1 AND status = $2
		 ORDER BY priority DESC`,
		sessionID, FrontierStatusPending,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending frontier URLs: %w", err)
	}
	defer rows.Close()

	var urls []FrontierURL
	for rows.Next() {
		var fu FrontierURL
		if err := rows.Scan(&fu.ID, &fu.SessionID, &fu.URL, &fu.Priority, &fu.PageType,
			&fu.Reason, &fu.Status, &fu.SkipReason, &fu.ErrorMessage, &fu.CrawledPageID,
			&fu.CreatedAt, &fu.FetchedAt); err != nil {
			return nil, err
		}
		urls = append(urls, fu)
	}
	return urls, nil
}

// GetAllFrontierURLs retrieves all URLs for a session
func (db *DB) GetAllFrontierURLs(ctx context.Context, sessionID uuid.UUID) ([]FrontierURL, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, session_id, url, priority, page_type, reason, status, skip_reason,
		        error_message, crawled_page_id, created_at, fetched_at
		 FROM research_frontier
		 WHERE session_id = $1
		 ORDER BY priority DESC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get frontier URLs: %w", err)
	}
	defer rows.Close()

	var urls []FrontierURL
	for rows.Next() {
		var fu FrontierURL
		if err := rows.Scan(&fu.ID, &fu.SessionID, &fu.URL, &fu.Priority, &fu.PageType,
			&fu.Reason, &fu.Status, &fu.SkipReason, &fu.ErrorMessage, &fu.CrawledPageID,
			&fu.CreatedAt, &fu.FetchedAt); err != nil {
			return nil, err
		}
		urls = append(urls, fu)
	}
	return urls, nil
}

// MarkFrontierURLFetched marks a frontier URL as fetched
func (db *DB) MarkFrontierURLFetched(ctx context.Context, id uuid.UUID, crawledPageID *uuid.UUID) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE research_frontier SET status = $1, fetched_at = $2, crawled_page_id = $3 WHERE id = $4`,
		FrontierStatusFetched, time.Now(), crawledPageID, id)
	if err != nil {
		return fmt.Errorf("failed to mark frontier URL as fetched: %w", err)
	}
	return nil
}

// MarkFrontierURLSkipped marks a frontier URL as skipped
func (db *DB) MarkFrontierURLSkipped(ctx context.Context, id uuid.UUID, reason string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE research_frontier SET status = $1, skip_reason = $2 WHERE id = $3`,
		FrontierStatusSkipped, reason, id)
	if err != nil {
		return fmt.Errorf("failed to mark frontier URL as skipped: %w", err)
	}
	return nil
}

// MarkFrontierURLFailed marks a frontier URL as failed
func (db *DB) MarkFrontierURLFailed(ctx context.Context, id uuid.UUID, errorMsg string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE research_frontier SET status = $1, error_message = $2 WHERE id = $3`,
		FrontierStatusFailed, errorMsg, id)
	if err != nil {
		return fmt.Errorf("failed to mark frontier URL as failed: %w", err)
	}
	return nil
}

// CountFrontierURLsByStatus counts URLs by status for a session
func (db *DB) CountFrontierURLsByStatus(ctx context.Context, sessionID uuid.UUID) (map[string]int, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT status, COUNT(*) as count
		 FROM research_frontier
		 WHERE session_id = $1
		 GROUP BY status`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to count frontier URLs: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, nil
}

// -----------------------------------------------------------------------------
// Research Brand Signals Methods
// -----------------------------------------------------------------------------

// SaveResearchBrandSignals saves brand signals for a session
func (db *DB) SaveResearchBrandSignals(ctx context.Context, sessionID uuid.UUID, signals []ResearchBrandSignalInput) ([]ResearchBrandSignal, error) {
	var result []ResearchBrandSignal

	for _, input := range signals {
		var rbs ResearchBrandSignal
		var keyPointsJSON, valuesJSON []byte
		var err error

		if len(input.KeyPoints) > 0 {
			keyPointsJSON, err = json.Marshal(input.KeyPoints)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal key points: %w", err)
			}
		}
		if len(input.ValuesFound) > 0 {
			valuesJSON, err = json.Marshal(input.ValuesFound)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal values: %w", err)
			}
		}

		err = db.pool.QueryRow(ctx,
			`INSERT INTO research_brand_signals (session_id, url, signal_type, key_points, values_found)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id, session_id, url, signal_type, created_at`,
			sessionID, input.URL, nullIfEmpty(input.SignalType), keyPointsJSON, valuesJSON,
		).Scan(&rbs.ID, &rbs.SessionID, &rbs.URL, &rbs.SignalType, &rbs.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to save brand signal: %w", err)
		}
		rbs.KeyPoints = input.KeyPoints
		rbs.ValuesFound = input.ValuesFound
		result = append(result, rbs)
	}

	return result, nil
}

// GetResearchBrandSignals retrieves brand signals for a session
func (db *DB) GetResearchBrandSignals(ctx context.Context, sessionID uuid.UUID) ([]ResearchBrandSignal, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, session_id, url, signal_type, key_points, values_found, created_at
		 FROM research_brand_signals
		 WHERE session_id = $1
		 ORDER BY created_at`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get brand signals: %w", err)
	}
	defer rows.Close()

	var signals []ResearchBrandSignal
	for rows.Next() {
		var rbs ResearchBrandSignal
		var keyPointsJSON, valuesJSON []byte
		if err := rows.Scan(&rbs.ID, &rbs.SessionID, &rbs.URL, &rbs.SignalType,
			&keyPointsJSON, &valuesJSON, &rbs.CreatedAt); err != nil {
			return nil, err
		}
		if keyPointsJSON != nil {
			_ = json.Unmarshal(keyPointsJSON, &rbs.KeyPoints)
		}
		if valuesJSON != nil {
			_ = json.Unmarshal(valuesJSON, &rbs.ValuesFound)
		}
		signals = append(signals, rbs)
	}
	return signals, nil
}

// GetResearchBrandSignalsByCompany retrieves brand signals for all sessions of a company
func (db *DB) GetResearchBrandSignalsByCompany(ctx context.Context, companyID uuid.UUID) ([]ResearchBrandSignal, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT rbs.id, rbs.session_id, rbs.url, rbs.signal_type, rbs.key_points, 
		        rbs.values_found, rbs.created_at
		 FROM research_brand_signals rbs
		 JOIN research_sessions rs ON rbs.session_id = rs.id
		 WHERE rs.company_id = $1
		 ORDER BY rbs.created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get brand signals by company: %w", err)
	}
	defer rows.Close()

	var signals []ResearchBrandSignal
	for rows.Next() {
		var rbs ResearchBrandSignal
		var keyPointsJSON, valuesJSON []byte
		if err := rows.Scan(&rbs.ID, &rbs.SessionID, &rbs.URL, &rbs.SignalType,
			&keyPointsJSON, &valuesJSON, &rbs.CreatedAt); err != nil {
			return nil, err
		}
		if keyPointsJSON != nil {
			_ = json.Unmarshal(keyPointsJSON, &rbs.KeyPoints)
		}
		if valuesJSON != nil {
			_ = json.Unmarshal(valuesJSON, &rbs.ValuesFound)
		}
		signals = append(signals, rbs)
	}
	return signals, nil
}

// -----------------------------------------------------------------------------
// Full Session Loading
// -----------------------------------------------------------------------------

// GetResearchSessionWithDetails loads a session with its frontier and signals
func (db *DB) GetResearchSessionWithDetails(ctx context.Context, id uuid.UUID) (*ResearchSession, error) {
	session, err := db.GetResearchSessionByID(ctx, id)
	if err != nil || session == nil {
		return session, err
	}

	session.FrontierURLs, err = db.GetAllFrontierURLs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load frontier URLs: %w", err)
	}

	session.BrandSignals, err = db.GetResearchBrandSignals(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load brand signals: %w", err)
	}

	return session, nil
}
