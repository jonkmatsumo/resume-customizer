package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// -----------------------------------------------------------------------------
// Skill Methods
// -----------------------------------------------------------------------------

// FindOrCreateSkill finds an existing skill or creates a new one
func (db *DB) FindOrCreateSkill(ctx context.Context, skillName string) (*Skill, error) {
	normalized := NormalizeSkillName(skillName)
	if normalized == "" {
		return nil, fmt.Errorf("skill name cannot be empty")
	}

	category := DetectSkillCategory(normalized)

	var skill Skill
	err := db.pool.QueryRow(ctx,
		`INSERT INTO skills (name, name_normalized, category)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (name_normalized) DO UPDATE SET name = skills.name
		 RETURNING id, name, name_normalized, category, created_at`,
		skillName, normalized, category,
	).Scan(&skill.ID, &skill.Name, &skill.NameNormalized, &skill.Category, &skill.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create skill: %w", err)
	}

	return &skill, nil
}

// GetSkillByName retrieves a skill by its normalized name
func (db *DB) GetSkillByName(ctx context.Context, name string) (*Skill, error) {
	normalized := NormalizeSkillName(name)

	var skill Skill
	err := db.pool.QueryRow(ctx,
		`SELECT id, name, name_normalized, category, created_at
		 FROM skills WHERE name_normalized = $1`,
		normalized,
	).Scan(&skill.ID, &skill.Name, &skill.NameNormalized, &skill.Category, &skill.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}

	return &skill, nil
}

// GetSkillByID retrieves a skill by its ID
func (db *DB) GetSkillByID(ctx context.Context, id uuid.UUID) (*Skill, error) {
	var skill Skill
	err := db.pool.QueryRow(ctx,
		`SELECT id, name, name_normalized, category, created_at
		 FROM skills WHERE id = $1`,
		id,
	).Scan(&skill.ID, &skill.Name, &skill.NameNormalized, &skill.Category, &skill.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}

	return &skill, nil
}

// ListSkills retrieves all skills, optionally filtered by category
func (db *DB) ListSkills(ctx context.Context, category string) ([]Skill, error) {
	var rows pgx.Rows
	var err error

	if category == "" {
		rows, err = db.pool.Query(ctx,
			`SELECT id, name, name_normalized, category, created_at
			 FROM skills ORDER BY name_normalized`)
	} else {
		rows, err = db.pool.Query(ctx,
			`SELECT id, name, name_normalized, category, created_at
			 FROM skills WHERE category = $1 ORDER BY name_normalized`,
			category)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list skills: %w", err)
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var s Skill
		if err := rows.Scan(&s.ID, &s.Name, &s.NameNormalized, &s.Category, &s.CreatedAt); err != nil {
			return nil, err
		}
		skills = append(skills, s)
	}
	return skills, nil
}

// -----------------------------------------------------------------------------
// Story Methods
// -----------------------------------------------------------------------------

// CreateStory creates a new story with all its bullets
func (db *DB) CreateStory(ctx context.Context, input *StoryCreateInput) (*Story, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil && rErr != pgx.ErrTxClosed {
			// Log rollback error but don't overwrite main error
			_ = rErr
		}
	}()

	// Insert story
	var story Story
	err = tx.QueryRow(ctx,
		`INSERT INTO stories (story_id, user_id, job_id, title, description)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (story_id) DO UPDATE SET
		     job_id = $3,
		     title = $4,
		     description = $5,
		     updated_at = NOW()
		 RETURNING id, story_id, user_id, job_id, title, description, created_at, updated_at`,
		input.StoryID, input.UserID, input.JobID,
		nullIfEmpty(input.Title), nullIfEmpty(input.Description),
	).Scan(&story.ID, &story.StoryID, &story.UserID, &story.JobID,
		&story.Title, &story.Description, &story.CreatedAt, &story.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create story: %w", err)
	}

	// Delete existing bullets for upsert
	_, _ = tx.Exec(ctx, "DELETE FROM bullets WHERE story_id = $1", story.ID)

	// Insert bullets
	for i, bulletInput := range input.Bullets {
		var bullet Bullet
		ordinal := bulletInput.Ordinal
		if ordinal == 0 {
			ordinal = i + 1
		}

		// Normalize evidence strength
		evidenceStrength := bulletInput.EvidenceStrength
		if evidenceStrength == "" {
			evidenceStrength = EvidenceStrengthMedium
		}

		err = tx.QueryRow(ctx,
			`INSERT INTO bullets (bullet_id, story_id, job_id, text, metrics, length_chars, 
			                      evidence_strength, risk_flags, ordinal)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 RETURNING id, bullet_id, story_id, job_id, text, metrics, length_chars,
			           evidence_strength, risk_flags, ordinal, created_at, updated_at`,
			bulletInput.BulletID, story.ID, input.JobID, bulletInput.Text,
			nullIfEmpty(bulletInput.Metrics), len(bulletInput.Text),
			evidenceStrength, StringArray(bulletInput.RiskFlags), ordinal,
		).Scan(&bullet.ID, &bullet.BulletID, &bullet.StoryID, &bullet.JobID,
			&bullet.Text, &bullet.Metrics, &bullet.LengthChars,
			&bullet.EvidenceStrength, &bullet.RiskFlags, &bullet.Ordinal,
			&bullet.CreatedAt, &bullet.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to create bullet: %w", err)
		}

		// Link skills to bullet
		for _, skillName := range bulletInput.Skills {
			normalized := NormalizeSkillName(skillName)
			if normalized == "" {
				continue
			}

			category := DetectSkillCategory(normalized)

			// Find or create skill
			var skillID uuid.UUID
			err = tx.QueryRow(ctx,
				`INSERT INTO skills (name, name_normalized, category)
				 VALUES ($1, $2, $3)
				 ON CONFLICT (name_normalized) DO UPDATE SET name = skills.name
				 RETURNING id`,
				skillName, normalized, category,
			).Scan(&skillID)
			if err != nil {
				return nil, fmt.Errorf("failed to create skill %s: %w", skillName, err)
			}

			// Link bullet to skill
			_, err = tx.Exec(ctx,
				`INSERT INTO bullet_skills (bullet_id, skill_id)
				 VALUES ($1, $2)
				 ON CONFLICT DO NOTHING`,
				bullet.ID, skillID,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to link skill: %w", err)
			}

			bullet.Skills = append(bullet.Skills, skillName)
		}

		story.Bullets = append(story.Bullets, bullet)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &story, nil
}

// GetStoryByID retrieves a story by its UUID
func (db *DB) GetStoryByID(ctx context.Context, id uuid.UUID) (*Story, error) {
	var story Story
	err := db.pool.QueryRow(ctx,
		`SELECT id, story_id, user_id, job_id, title, description, created_at, updated_at
		 FROM stories WHERE id = $1`,
		id,
	).Scan(&story.ID, &story.StoryID, &story.UserID, &story.JobID,
		&story.Title, &story.Description, &story.CreatedAt, &story.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get story: %w", err)
	}

	// Load bullets
	if err := db.loadStoryBullets(ctx, &story); err != nil {
		return nil, err
	}

	return &story, nil
}

// GetStoryByStoryID retrieves a story by its human-readable ID
func (db *DB) GetStoryByStoryID(ctx context.Context, storyID string) (*Story, error) {
	var story Story
	err := db.pool.QueryRow(ctx,
		`SELECT id, story_id, user_id, job_id, title, description, created_at, updated_at
		 FROM stories WHERE story_id = $1`,
		storyID,
	).Scan(&story.ID, &story.StoryID, &story.UserID, &story.JobID,
		&story.Title, &story.Description, &story.CreatedAt, &story.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get story: %w", err)
	}

	if err := db.loadStoryBullets(ctx, &story); err != nil {
		return nil, err
	}

	return &story, nil
}

// ListStoriesByUser retrieves all stories for a user
func (db *DB) ListStoriesByUser(ctx context.Context, userID uuid.UUID) ([]Story, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT s.id, s.story_id, s.user_id, s.job_id, s.title, s.description, 
		        s.created_at, s.updated_at
		 FROM stories s
		 WHERE s.user_id = $1
		 ORDER BY s.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list stories: %w", err)
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var s Story
		if err := rows.Scan(&s.ID, &s.StoryID, &s.UserID, &s.JobID,
			&s.Title, &s.Description, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		// Load bullets for each story
		if err := db.loadStoryBullets(ctx, &s); err != nil {
			return nil, err
		}
		stories = append(stories, s)
	}
	return stories, nil
}

// ListStoriesByJob retrieves all stories for a job
func (db *DB) ListStoriesByJob(ctx context.Context, jobID uuid.UUID) ([]Story, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, story_id, user_id, job_id, title, description, created_at, updated_at
		 FROM stories WHERE job_id = $1 ORDER BY created_at DESC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list stories: %w", err)
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var s Story
		if err := rows.Scan(&s.ID, &s.StoryID, &s.UserID, &s.JobID,
			&s.Title, &s.Description, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		if err := db.loadStoryBullets(ctx, &s); err != nil {
			return nil, err
		}
		stories = append(stories, s)
	}
	return stories, nil
}

// DeleteStory removes a story and all its bullets (cascades)
func (db *DB) DeleteStory(ctx context.Context, id uuid.UUID) error {
	_, err := db.pool.Exec(ctx, "DELETE FROM stories WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete story: %w", err)
	}
	return nil
}

// loadStoryBullets loads bullets for a story
func (db *DB) loadStoryBullets(ctx context.Context, story *Story) error {
	rows, err := db.pool.Query(ctx,
		`SELECT id, bullet_id, story_id, job_id, text, metrics, length_chars,
		        evidence_strength, risk_flags, ordinal, created_at, updated_at
		 FROM bullets
		 WHERE story_id = $1
		 ORDER BY ordinal`,
		story.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to load bullets: %w", err)
	}
	defer rows.Close()

	story.Bullets = nil // Reset
	for rows.Next() {
		var b Bullet
		if err := rows.Scan(&b.ID, &b.BulletID, &b.StoryID, &b.JobID, &b.Text, &b.Metrics,
			&b.LengthChars, &b.EvidenceStrength, &b.RiskFlags, &b.Ordinal,
			&b.CreatedAt, &b.UpdatedAt); err != nil {
			return err
		}
		// Load skills
		skills, _ := db.GetBulletSkills(ctx, b.ID)
		b.Skills = skills
		story.Bullets = append(story.Bullets, b)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Bullet Methods
// -----------------------------------------------------------------------------

// GetBulletByID retrieves a bullet by its UUID
func (db *DB) GetBulletByID(ctx context.Context, id uuid.UUID) (*Bullet, error) {
	var b Bullet
	err := db.pool.QueryRow(ctx,
		`SELECT id, bullet_id, story_id, job_id, text, metrics, length_chars,
		        evidence_strength, risk_flags, ordinal, created_at, updated_at
		 FROM bullets WHERE id = $1`,
		id,
	).Scan(&b.ID, &b.BulletID, &b.StoryID, &b.JobID, &b.Text, &b.Metrics,
		&b.LengthChars, &b.EvidenceStrength, &b.RiskFlags, &b.Ordinal,
		&b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bullet: %w", err)
	}

	skills, _ := db.GetBulletSkills(ctx, b.ID)
	b.Skills = skills

	return &b, nil
}

// GetBulletByBulletID retrieves a bullet by its stable identifier
func (db *DB) GetBulletByBulletID(ctx context.Context, bulletID string) (*Bullet, error) {
	var b Bullet
	err := db.pool.QueryRow(ctx,
		`SELECT id, bullet_id, story_id, job_id, text, metrics, length_chars,
		        evidence_strength, risk_flags, ordinal, created_at, updated_at
		 FROM bullets WHERE bullet_id = $1`,
		bulletID,
	).Scan(&b.ID, &b.BulletID, &b.StoryID, &b.JobID, &b.Text, &b.Metrics,
		&b.LengthChars, &b.EvidenceStrength, &b.RiskFlags, &b.Ordinal,
		&b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bullet: %w", err)
	}

	skills, _ := db.GetBulletSkills(ctx, b.ID)
	b.Skills = skills

	return &b, nil
}

// GetBulletSkills retrieves all skill names for a bullet
func (db *DB) GetBulletSkills(ctx context.Context, bulletID uuid.UUID) ([]string, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT s.name FROM skills s
		 JOIN bullet_skills bs ON bs.skill_id = s.id
		 WHERE bs.bullet_id = $1
		 ORDER BY s.name`,
		bulletID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get bullet skills: %w", err)
	}
	defer rows.Close()

	var skills []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		skills = append(skills, name)
	}
	return skills, nil
}

// -----------------------------------------------------------------------------
// Query Methods
// -----------------------------------------------------------------------------

// FindBulletsBySkill finds all bullets that use a specific skill
func (db *DB) FindBulletsBySkill(ctx context.Context, skillName string) ([]Bullet, error) {
	normalized := NormalizeSkillName(skillName)

	rows, err := db.pool.Query(ctx,
		`SELECT DISTINCT b.id, b.bullet_id, b.story_id, b.job_id, b.text, b.metrics,
		        b.length_chars, b.evidence_strength, b.risk_flags, b.ordinal,
		        b.created_at, b.updated_at
		 FROM bullets b
		 JOIN bullet_skills bs ON bs.bullet_id = b.id
		 JOIN skills s ON s.id = bs.skill_id
		 WHERE s.name_normalized = $1
		 ORDER BY b.created_at DESC`,
		normalized,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find bullets by skill: %w", err)
	}
	defer rows.Close()

	var bullets []Bullet
	for rows.Next() {
		var b Bullet
		if err := rows.Scan(&b.ID, &b.BulletID, &b.StoryID, &b.JobID, &b.Text, &b.Metrics,
			&b.LengthChars, &b.EvidenceStrength, &b.RiskFlags, &b.Ordinal,
			&b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		skills, _ := db.GetBulletSkills(ctx, b.ID)
		b.Skills = skills
		bullets = append(bullets, b)
	}
	return bullets, nil
}

// FindBulletsByEvidenceStrength finds bullets by evidence strength
func (db *DB) FindBulletsByEvidenceStrength(ctx context.Context, strength string) ([]Bullet, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, bullet_id, story_id, job_id, text, metrics, length_chars,
		        evidence_strength, risk_flags, ordinal, created_at, updated_at
		 FROM bullets
		 WHERE evidence_strength = $1
		 ORDER BY created_at DESC`,
		strength,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find bullets: %w", err)
	}
	defer rows.Close()

	var bullets []Bullet
	for rows.Next() {
		var b Bullet
		if err := rows.Scan(&b.ID, &b.BulletID, &b.StoryID, &b.JobID, &b.Text, &b.Metrics,
			&b.LengthChars, &b.EvidenceStrength, &b.RiskFlags, &b.Ordinal,
			&b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		bullets = append(bullets, b)
	}
	return bullets, nil
}

// GetSkillUsageCount returns how many bullets use each skill
func (db *DB) GetSkillUsageCount(ctx context.Context) (map[string]int, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT s.name, COUNT(bs.bullet_id) as count
		 FROM skills s
		 LEFT JOIN bullet_skills bs ON bs.skill_id = s.id
		 GROUP BY s.id, s.name
		 ORDER BY count DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get skill usage: %w", err)
	}
	defer rows.Close()

	usage := make(map[string]int)
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		usage[name] = count
	}
	return usage, nil
}

// ListSkillsByUserID retrieves all unique skills used by bullets in stories belonging to a user
func (db *DB) ListSkillsByUserID(ctx context.Context, userID uuid.UUID) ([]Skill, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT DISTINCT s.id, s.name, s.name_normalized, s.category, s.created_at
		 FROM skills s
		 JOIN bullet_skills bs ON bs.skill_id = s.id
		 JOIN bullets b ON b.id = bs.bullet_id
		 JOIN stories st ON st.id = b.story_id
		 WHERE st.user_id = $1
		 ORDER BY s.name_normalized`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list skills by user: %w", err)
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var s Skill
		if err := rows.Scan(&s.ID, &s.Name, &s.NameNormalized, &s.Category, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan skill: %w", err)
		}
		skills = append(skills, s)
	}
	return skills, nil
}

// GetBulletsByStoryID retrieves all bullets for a story
func (db *DB) GetBulletsByStoryID(ctx context.Context, storyID uuid.UUID) ([]Bullet, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, bullet_id, story_id, job_id, text, metrics, length_chars,
		        evidence_strength, risk_flags, ordinal, created_at, updated_at
		 FROM bullets
		 WHERE story_id = $1
		 ORDER BY ordinal`,
		storyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get bullets by story: %w", err)
	}
	defer rows.Close()

	var bullets []Bullet
	for rows.Next() {
		var b Bullet
		if err := rows.Scan(&b.ID, &b.BulletID, &b.StoryID, &b.JobID, &b.Text, &b.Metrics,
			&b.LengthChars, &b.EvidenceStrength, &b.RiskFlags, &b.Ordinal,
			&b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan bullet: %w", err)
		}
		// Load skills for each bullet
		skills, _ := db.GetBulletSkills(ctx, b.ID)
		b.Skills = skills
		bullets = append(bullets, b)
	}
	return bullets, nil
}

// GetBulletsBySkillIDAndUserID retrieves all bullets that use a specific skill, scoped to a user
func (db *DB) GetBulletsBySkillIDAndUserID(ctx context.Context, skillID uuid.UUID, userID uuid.UUID) ([]Bullet, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT DISTINCT b.id, b.bullet_id, b.story_id, b.job_id, b.text, b.metrics,
		        b.length_chars, b.evidence_strength, b.risk_flags, b.ordinal,
		        b.created_at, b.updated_at
		 FROM bullets b
		 JOIN bullet_skills bs ON bs.bullet_id = b.id
		 JOIN stories st ON st.id = b.story_id
		 WHERE bs.skill_id = $1 AND st.user_id = $2
		 ORDER BY b.created_at DESC`,
		skillID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get bullets by skill and user: %w", err)
	}
	defer rows.Close()

	var bullets []Bullet
	for rows.Next() {
		var b Bullet
		if err := rows.Scan(&b.ID, &b.BulletID, &b.StoryID, &b.JobID, &b.Text, &b.Metrics,
			&b.LengthChars, &b.EvidenceStrength, &b.RiskFlags, &b.Ordinal,
			&b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan bullet: %w", err)
		}
		// Load skills for each bullet
		skills, _ := db.GetBulletSkills(ctx, b.ID)
		b.Skills = skills
		bullets = append(bullets, b)
	}
	return bullets, nil
}

// -----------------------------------------------------------------------------
// Education Highlight Methods
// -----------------------------------------------------------------------------

// AddEducationHighlight adds a highlight to an education entry
func (db *DB) AddEducationHighlight(ctx context.Context, educationID uuid.UUID, text string, ordinal int) (*EducationHighlight, error) {
	var h EducationHighlight
	err := db.pool.QueryRow(ctx,
		`INSERT INTO education_highlights (education_id, text, ordinal)
		 VALUES ($1, $2, $3)
		 RETURNING id, education_id, text, ordinal, created_at`,
		educationID, text, ordinal,
	).Scan(&h.ID, &h.EducationID, &h.Text, &h.Ordinal, &h.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to add education highlight: %w", err)
	}
	return &h, nil
}

// GetEducationHighlights retrieves all highlights for an education entry
func (db *DB) GetEducationHighlights(ctx context.Context, educationID uuid.UUID) ([]EducationHighlight, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, education_id, text, ordinal, created_at
		 FROM education_highlights
		 WHERE education_id = $1
		 ORDER BY ordinal`,
		educationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get highlights: %w", err)
	}
	defer rows.Close()

	var highlights []EducationHighlight
	for rows.Next() {
		var h EducationHighlight
		if err := rows.Scan(&h.ID, &h.EducationID, &h.Text, &h.Ordinal, &h.CreatedAt); err != nil {
			return nil, err
		}
		highlights = append(highlights, h)
	}
	return highlights, nil
}

// DeleteEducationHighlights removes all highlights for an education entry
func (db *DB) DeleteEducationHighlights(ctx context.Context, educationID uuid.UUID) error {
	_, err := db.pool.Exec(ctx, "DELETE FROM education_highlights WHERE education_id = $1", educationID)
	if err != nil {
		return fmt.Errorf("failed to delete highlights: %w", err)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Import Methods
// -----------------------------------------------------------------------------

// ImportExperienceBank imports a complete experience bank from JSON structure
func (db *DB) ImportExperienceBank(ctx context.Context, input *ExperienceBankImportInput) error {
	for _, storyInput := range input.Stories {
		// Find or create job
		job, err := db.findOrCreateJobForStory(ctx, input.UserID, storyInput.Company, storyInput.Role,
			storyInput.StartDate, storyInput.EndDate)
		if err != nil {
			return fmt.Errorf("failed to create job for story %s: %w", storyInput.ID, err)
		}

		// Create story
		bullets := make([]BulletCreateInput, len(storyInput.Bullets))
		for i, b := range storyInput.Bullets {
			bullets[i] = BulletCreateInput{
				BulletID:         b.ID,
				Text:             b.Text,
				Metrics:          b.Metrics,
				EvidenceStrength: b.EvidenceStrength,
				RiskFlags:        b.RiskFlags,
				Skills:           b.Skills,
				Ordinal:          i + 1,
			}
		}

		storyCreateInput := &StoryCreateInput{
			StoryID: storyInput.ID,
			UserID:  input.UserID,
			JobID:   job.ID,
			Bullets: bullets,
		}

		_, err = db.CreateStory(ctx, storyCreateInput)
		if err != nil {
			return fmt.Errorf("failed to create story %s: %w", storyInput.ID, err)
		}
	}

	// Import education
	for _, eduInput := range input.Education {
		edu, err := db.findOrCreateEducationForImport(ctx, input.UserID, eduInput)
		if err != nil {
			return fmt.Errorf("failed to create education %s: %w", eduInput.ID, err)
		}

		// Clear existing highlights and add new ones
		_ = db.DeleteEducationHighlights(ctx, edu.ID)

		// Add highlights
		for i, highlight := range eduInput.Highlights {
			_, err = db.AddEducationHighlight(ctx, edu.ID, highlight, i+1)
			if err != nil {
				return fmt.Errorf("failed to add education highlight: %w", err)
			}
		}
	}

	return nil
}

// findOrCreateJobForStory creates a job entry if it doesn't exist
func (db *DB) findOrCreateJobForStory(ctx context.Context, userID uuid.UUID, company, role, startDate, endDate string) (*Job, error) {
	// Try to find existing job
	var job Job
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, company, role_title, start_date, end_date, created_at
		 FROM jobs WHERE user_id = $1 AND company = $2 AND role_title = $3`,
		userID, company, role,
	).Scan(&job.ID, &job.UserID, &job.Company, &job.RoleTitle, &job.StartDate, &job.EndDate, &job.CreatedAt)

	if err == nil {
		return &job, nil
	}
	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to find job: %w", err)
	}

	// Parse dates
	start, _ := time.Parse("2006-01", startDate)
	var end *time.Time
	if endDate != "present" && endDate != "" {
		e, _ := time.Parse("2006-01", endDate)
		end = &e
	}

	// Create new job
	err = db.pool.QueryRow(ctx,
		`INSERT INTO jobs (user_id, company, role_title, start_date, end_date)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, company, role_title, start_date, end_date, created_at`,
		userID, company, role, start, end,
	).Scan(&job.ID, &job.UserID, &job.Company, &job.RoleTitle, &job.StartDate, &job.EndDate, &job.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return &job, nil
}

// findOrCreateEducationForImport creates an education entry if it doesn't exist
func (db *DB) findOrCreateEducationForImport(ctx context.Context, userID uuid.UUID, input EducationImportInput) (*Education, error) {
	// Try to find existing education
	var edu Education
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, school, degree_type, field, gpa, created_at
		 FROM education WHERE user_id = $1 AND school = $2 AND field = $3`,
		userID, input.School, input.Field,
	).Scan(&edu.ID, &edu.UserID, &edu.School, &edu.DegreeType, &edu.Field, &edu.GPA, &edu.CreatedAt)

	if err == nil {
		return &edu, nil
	}
	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to find education: %w", err)
	}

	// Create new education
	err = db.pool.QueryRow(ctx,
		`INSERT INTO education (user_id, school, degree_type, field, gpa)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, school, degree_type, field, gpa, created_at`,
		userID, input.School, input.Degree, input.Field, nullIfEmpty(input.GPA),
	).Scan(&edu.ID, &edu.UserID, &edu.School, &edu.DegreeType, &edu.Field, &edu.GPA, &edu.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create education: %w", err)
	}

	return &edu, nil
}
