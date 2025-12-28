// Package db provides PostgreSQL database access for artifact storage.
package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a PostgreSQL connection pool
type DB struct {
	pool *pgxpool.Pool
}

// Connect establishes a connection pool to the database
func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close closes the connection pool
func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// CreateRun creates a new pipeline run record and returns its ID
func (db *DB) CreateRun(ctx context.Context, company, roleTitle, jobURL string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.pool.QueryRow(ctx,
		`INSERT INTO pipeline_runs (company, role_title, job_url, status)
		 VALUES ($1, $2, $3, 'running')
		 RETURNING id`,
		company, roleTitle, jobURL,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create run: %w", err)
	}
	return id, nil
}

// CompleteRun marks a pipeline run as completed
func (db *DB) CompleteRun(ctx context.Context, runID uuid.UUID, status string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE pipeline_runs SET status = $1, completed_at = NOW() WHERE id = $2`,
		status, runID,
	)
	if err != nil {
		return fmt.Errorf("failed to complete run: %w", err)
	}
	return nil
}

// SaveArtifact stores a JSON artifact for a pipeline run
func (db *DB) SaveArtifact(ctx context.Context, runID uuid.UUID, step string, content any) error {
	jsonBytes, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal artifact: %w", err)
	}

	_, err = db.pool.Exec(ctx,
		`INSERT INTO artifacts (run_id, step, content)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (run_id, step) DO UPDATE SET content = $3, created_at = NOW()`,
		runID, step, jsonBytes,
	)
	if err != nil {
		return fmt.Errorf("failed to save artifact %s: %w", step, err)
	}
	return nil
}

// SaveTextArtifact stores a text artifact (like .tex or .txt files) for a pipeline run
func (db *DB) SaveTextArtifact(ctx context.Context, runID uuid.UUID, step string, text string) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO artifacts (run_id, step, text_content)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (run_id, step) DO UPDATE SET text_content = $3, created_at = NOW()`,
		runID, step, text,
	)
	if err != nil {
		return fmt.Errorf("failed to save text artifact %s: %w", step, err)
	}
	return nil
}

// GetArtifact retrieves a JSON artifact by run ID and step
func (db *DB) GetArtifact(ctx context.Context, runID uuid.UUID, step string) ([]byte, error) {
	var content []byte
	err := db.pool.QueryRow(ctx,
		`SELECT content FROM artifacts WHERE run_id = $1 AND step = $2`,
		runID, step,
	).Scan(&content)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get artifact %s: %w", step, err)
	}
	return content, nil
}

// GetTextArtifact retrieves a text artifact by run ID and step
func (db *DB) GetTextArtifact(ctx context.Context, runID uuid.UUID, step string) (string, error) {
	var text string
	err := db.pool.QueryRow(ctx,
		`SELECT text_content FROM artifacts WHERE run_id = $1 AND step = $2`,
		runID, step,
	).Scan(&text)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get text artifact %s: %w", step, err)
	}
	return text, nil
}

// GetRun retrieves a pipeline run by ID
func (db *DB) GetRun(ctx context.Context, runID uuid.UUID) (*Run, error) {
	var run Run
	err := db.pool.QueryRow(ctx,
		`SELECT id, company, role_title, job_url, status, created_at, completed_at
		 FROM pipeline_runs WHERE id = $1`,
		runID,
	).Scan(&run.ID, &run.Company, &run.RoleTitle, &run.JobURL, &run.Status, &run.CreatedAt, &run.CompletedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get run: %w", err)
	}
	return &run, nil
}

// ListRuns retrieves recent pipeline runs
func (db *DB) ListRuns(ctx context.Context, limit int) ([]Run, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, company, role_title, job_url, status, created_at, completed_at
		 FROM pipeline_runs ORDER BY created_at DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var run Run
		if err := rows.Scan(&run.ID, &run.Company, &run.RoleTitle, &run.JobURL, &run.Status, &run.CreatedAt, &run.CompletedAt); err != nil {
			return nil, fmt.Errorf("failed to scan run: %w", err)
		}
		runs = append(runs, run)
	}
	return runs, nil
}
