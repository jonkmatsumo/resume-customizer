// Package db provides PostgreSQL database access for artifact storage.
package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jonathan/resume-customizer/internal/types"
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
func (db *DB) SaveArtifact(ctx context.Context, runID uuid.UUID, step, category string, content any) error {
	jsonBytes, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal artifact: %w", err)
	}

	_, err = db.pool.Exec(ctx,
		`INSERT INTO artifacts (run_id, step, category, content)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (run_id, step) DO UPDATE SET category = $3, content = $4, created_at = NOW()`,
		runID, step, category, jsonBytes,
	)
	if err != nil {
		return fmt.Errorf("failed to save artifact %s: %w", step, err)
	}
	return nil
}

// SaveTextArtifact stores a text artifact (like .tex or .txt files) for a pipeline run
func (db *DB) SaveTextArtifact(ctx context.Context, runID uuid.UUID, step, category, text string) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO artifacts (run_id, step, category, text_content)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (run_id, step) DO UPDATE SET category = $3, text_content = $4, created_at = NOW()`,
		runID, step, category, text,
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

// Artifact represents an artifact record
type Artifact struct {
	ID          uuid.UUID `json:"id"`
	RunID       uuid.UUID `json:"run_id"`
	Step        string    `json:"step"`
	Category    string    `json:"category"`
	Content     any       `json:"content,omitempty"`
	TextContent string    `json:"text_content,omitempty"`
}

// GetArtifactByID retrieves an artifact by its UUID
func (db *DB) GetArtifactByID(ctx context.Context, artifactID uuid.UUID) (*Artifact, error) {
	var artifact Artifact
	var contentBytes []byte
	var textContent *string
	var category *string

	err := db.pool.QueryRow(ctx,
		`SELECT id, run_id, step, category, content, text_content
		 FROM artifacts WHERE id = $1`,
		artifactID,
	).Scan(&artifact.ID, &artifact.RunID, &artifact.Step, &category, &contentBytes, &textContent)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	if category != nil {
		artifact.Category = *category
	}
	if textContent != nil {
		artifact.TextContent = *textContent
	}
	if len(contentBytes) > 0 {
		var content any
		if err := json.Unmarshal(contentBytes, &content); err == nil {
			artifact.Content = content
		}
	}

	return &artifact, nil
}

// RunFilters holds optional filters for listing runs
type RunFilters struct {
	Company string
	Status  string
	Limit   int
}

// ListRunsFiltered retrieves runs with optional filters
func (db *DB) ListRunsFiltered(ctx context.Context, filters RunFilters) ([]Run, error) {
	if filters.Limit == 0 {
		filters.Limit = 50
	}

	query := `SELECT id, company, role_title, job_url, status, created_at, completed_at
		FROM pipeline_runs WHERE 1=1`
	args := []any{}
	argNum := 1

	if filters.Company != "" {
		query += fmt.Sprintf(" AND company ILIKE $%d", argNum)
		args = append(args, "%"+filters.Company+"%")
		argNum++
	}
	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, filters.Status)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argNum)
	args = append(args, filters.Limit)

	rows, err := db.pool.Query(ctx, query, args...)
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

// DeleteRun deletes a pipeline run and all its artifacts (via cascade)
func (db *DB) DeleteRun(ctx context.Context, runID uuid.UUID) error {
	result, err := db.pool.Exec(ctx, `DELETE FROM pipeline_runs WHERE id = $1`, runID)
	if err != nil {
		return fmt.Errorf("failed to delete run: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("run not found: %s", runID)
	}
	return nil
}

// ---------------------------------------------------------------------
// User Profile Methods
// ---------------------------------------------------------------------

// CreateUser creates a new user
func (db *DB) CreateUser(ctx context.Context, name, email, phone string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.pool.QueryRow(ctx,
		`INSERT INTO users (name, email, phone)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		name, email, phone,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create user: %w", err)
	}
	return id, nil
}

// GetUser retrieves a user by ID
func (db *DB) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := db.pool.QueryRow(ctx,
		`SELECT id, name, email, phone, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

// UpdateUser updates a user profile
func (db *DB) UpdateUser(ctx context.Context, u *User) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE users SET name = $1, email = $2, phone = $3 WHERE id = $4`,
		u.Name, u.Email, u.Phone, u.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// DeleteUser deletes a user (cascades to jobs/education)
func (db *DB) DeleteUser(ctx context.Context, id uuid.UUID) error {
	cmd, err := db.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

// ---------------------------------------------------------------------
// Job Methods
// ---------------------------------------------------------------------

// CreateJob creates a new job entry
func (db *DB) CreateJob(ctx context.Context, job *Job) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.pool.QueryRow(ctx,
		`INSERT INTO jobs (user_id, company, role_title, location, employment_type, start_date, end_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		job.UserID, job.Company, job.RoleTitle, job.Location, job.EmploymentType, job.StartDate, job.EndDate,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create job: %w", err)
	}
	return id, nil
}

// ListJobs retrieves all jobs for a user
func (db *DB) ListJobs(ctx context.Context, userID uuid.UUID) ([]Job, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, company, role_title, location, employment_type, start_date, end_date, created_at
		 FROM jobs WHERE user_id = $1 ORDER BY start_date DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.UserID, &j.Company, &j.RoleTitle, &j.Location, &j.EmploymentType, &j.StartDate, &j.EndDate, &j.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// UpdateJob updates a job entry
func (db *DB) UpdateJob(ctx context.Context, job *Job) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE jobs SET company = $1, role_title = $2, location = $3, employment_type = $4, start_date = $5, end_date = $6
		 WHERE id = $7`,
		job.Company, job.RoleTitle, job.Location, job.EmploymentType, job.StartDate, job.EndDate, job.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}
	return nil
}

// DeleteJob deletes a job entry
func (db *DB) DeleteJob(ctx context.Context, id uuid.UUID) error {
	cmd, err := db.pool.Exec(ctx, `DELETE FROM jobs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("job not found: %s", id)
	}
	return nil
}

// ---------------------------------------------------------------------
// Experience Methods
// ---------------------------------------------------------------------

// CreateExperience creates a new experience bullet
func (db *DB) CreateExperience(ctx context.Context, exp *Experience) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.pool.QueryRow(ctx,
		`INSERT INTO experiences (job_id, bullet_text, skills, evidence_strength, risk_flags)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		exp.JobID, exp.BulletText, exp.Skills, exp.EvidenceStrength, exp.RiskFlags,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create experience: %w", err)
	}
	return id, nil
}

// ListExperiences retrieves all bullets for a job
func (db *DB) ListExperiences(ctx context.Context, jobID uuid.UUID) ([]Experience, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, job_id, bullet_text, skills, evidence_strength, risk_flags, created_at
		 FROM experiences WHERE job_id = $1 ORDER BY created_at ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list experiences: %w", err)
	}
	defer rows.Close()

	var experiences []Experience
	for rows.Next() {
		var e Experience
		if err := rows.Scan(&e.ID, &e.JobID, &e.BulletText, &e.Skills, &e.EvidenceStrength, &e.RiskFlags, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan experience: %w", err)
		}
		experiences = append(experiences, e)
	}
	return experiences, nil
}

// UpdateExperience updates an experience bullet
func (db *DB) UpdateExperience(ctx context.Context, exp *Experience) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE experiences SET bullet_text = $1, skills = $2, evidence_strength = $3, risk_flags = $4
		 WHERE id = $5`,
		exp.BulletText, exp.Skills, exp.EvidenceStrength, exp.RiskFlags, exp.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update experience: %w", err)
	}
	return nil
}

// DeleteExperience deletes an experience bullet
func (db *DB) DeleteExperience(ctx context.Context, id uuid.UUID) error {
	cmd, err := db.pool.Exec(ctx, `DELETE FROM experiences WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete experience: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("experience not found: %s", id)
	}
	return nil
}

// ---------------------------------------------------------------------
// Education Methods
// ---------------------------------------------------------------------

// CreateEducation creates a new education entry
func (db *DB) CreateEducation(ctx context.Context, edu *Education) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.pool.QueryRow(ctx,
		`INSERT INTO education (user_id, school, degree_type, field, gpa, location, start_date, end_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		edu.UserID, edu.School, edu.DegreeType, edu.Field, edu.GPA, edu.Location, edu.StartDate, edu.EndDate,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create education: %w", err)
	}
	return id, nil
}

// ListEducation retrieves all education entries for a user
func (db *DB) ListEducation(ctx context.Context, userID uuid.UUID) ([]Education, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, school, degree_type, field, gpa, location, start_date, end_date, created_at
		 FROM education WHERE user_id = $1 ORDER BY start_date DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list education: %w", err)
	}
	defer rows.Close()

	var education []Education
	for rows.Next() {
		var e Education
		if err := rows.Scan(&e.ID, &e.UserID, &e.School, &e.DegreeType, &e.Field, &e.GPA, &e.Location, &e.StartDate, &e.EndDate, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan education: %w", err)
		}
		education = append(education, e)
	}
	return education, nil
}

// UpdateEducation updates an education entry
func (db *DB) UpdateEducation(ctx context.Context, edu *Education) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE education SET school = $1, degree_type = $2, field = $3, gpa = $4, location = $5, start_date = $6, end_date = $7
		 WHERE id = $8`,
		edu.School, edu.DegreeType, edu.Field, edu.GPA, edu.Location, edu.StartDate, edu.EndDate, edu.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update education: %w", err)
	}
	return nil
}

// DeleteEducation deletes an education entry
func (db *DB) DeleteEducation(ctx context.Context, id uuid.UUID) error {
	cmd, err := db.pool.Exec(ctx, `DELETE FROM education WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete education: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("education not found: %s", id)
	}
	return nil
}

// ArtifactSummary is a lightweight view of an artifact for listing
type ArtifactSummary struct {
	ID        uuid.UUID `json:"id"`
	Step      string    `json:"step"`
	Category  string    `json:"category"`
	CreatedAt string    `json:"created_at"`
	HasJSON   bool      `json:"has_json"`
	HasText   bool      `json:"has_text"`
}

// ArtifactFilters holds optional filters for listing artifacts
type ArtifactFilters struct {
	RunID    uuid.UUID
	Step     string
	Category string
}

// ListArtifacts retrieves artifacts with optional filters
func (db *DB) ListArtifacts(ctx context.Context, filters ArtifactFilters) ([]ArtifactSummary, error) {
	query := `SELECT id, step, COALESCE(category, ''), created_at, 
		      content IS NOT NULL as has_json, text_content IS NOT NULL as has_text
		FROM artifacts WHERE 1=1`
	args := []any{}
	argNum := 1

	if filters.RunID != uuid.Nil {
		query += fmt.Sprintf(" AND run_id = $%d", argNum)
		args = append(args, filters.RunID)
		argNum++
	}
	if filters.Step != "" {
		query += fmt.Sprintf(" AND step = $%d", argNum)
		args = append(args, filters.Step)
		argNum++
	}
	if filters.Category != "" {
		query += fmt.Sprintf(" AND category = $%d", argNum)
		args = append(args, filters.Category)
	}

	query += " ORDER BY created_at ASC"

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []ArtifactSummary
	for rows.Next() {
		var a ArtifactSummary
		var createdAt any
		if err := rows.Scan(&a.ID, &a.Step, &a.Category, &createdAt, &a.HasJSON, &a.HasText); err != nil {
			return nil, fmt.Errorf("failed to scan artifact: %w", err)
		}
		if t, ok := createdAt.(interface{ String() string }); ok {
			a.CreatedAt = t.String()
		}
		artifacts = append(artifacts, a)
	}
	return artifacts, nil
}

// GetExperienceBank assembles a full ExperienceBank for a user from the database
func (db *DB) GetExperienceBank(ctx context.Context, userID uuid.UUID) (*types.ExperienceBank, error) {
	// 1. Fetch Jobs
	jobs, err := db.ListJobs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	// 2. Build Stories from Jobs
	var stories []types.Story
	for _, job := range jobs {
		// Fetch experiences for this job
		exps, err := db.ListExperiences(ctx, job.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list experiences for job %s: %w", job.ID, err)
		}

		// Map Bullets
		var bullets []types.Bullet
		for _, exp := range exps {
			bullets = append(bullets, types.Bullet{
				ID:               exp.ID.String(),
				Text:             exp.BulletText,
				Skills:           exp.Skills,
				EvidenceStrength: exp.EvidenceStrength,
				RiskFlags:        exp.RiskFlags,
				LengthChars:      len(exp.BulletText),
			})
		}

		// Helper to format dates
		formatDate := func(d *Date) string {
			if d == nil {
				return ""
			}
			return d.Format("2006-01")
		}

		stories = append(stories, types.Story{
			ID:        job.ID.String(),
			Company:   job.Company,
			Role:      job.RoleTitle,
			StartDate: formatDate(job.StartDate),
			EndDate:   formatDate(job.EndDate),
			Bullets:   bullets,
		})
	}

	// 3. Fetch Education
	dbEdu, err := db.ListEducation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list education: %w", err)
	}

	// 4. Map Education
	var education []types.Education
	for _, edu := range dbEdu {
		formatDate := func(d *Date) string {
			if d == nil {
				return ""
			}
			return d.Format("2006-01")
		}

		education = append(education, types.Education{
			ID:        edu.ID.String(),
			School:    edu.School,
			Degree:    edu.DegreeType,
			Field:     edu.Field,
			GPA:       edu.GPA,
			StartDate: formatDate(edu.StartDate),
			EndDate:   formatDate(edu.EndDate),
		})
	}

	return &types.ExperienceBank{
		Stories:   stories,
		Education: education,
	}, nil
}
