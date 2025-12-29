package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// -----------------------------------------------------------------------------
// Run Ranked Stories Methods
// -----------------------------------------------------------------------------

// SaveRunRankedStories saves ranked stories for a pipeline run
func (db *DB) SaveRunRankedStories(ctx context.Context, runID uuid.UUID, stories []RunRankedStoryInput) ([]RunRankedStory, error) {
	// Delete existing ranked stories for this run (upsert behavior)
	_, err := db.pool.Exec(ctx, "DELETE FROM run_ranked_stories WHERE run_id = $1", runID)
	if err != nil {
		return nil, fmt.Errorf("failed to clear existing ranked stories: %w", err)
	}

	var result []RunRankedStory
	for _, input := range stories {
		var rs RunRankedStory
		var matchedSkillsJSON []byte
		if len(input.MatchedSkills) > 0 {
			matchedSkillsJSON, _ = json.Marshal(input.MatchedSkills)
		}

		err := db.pool.QueryRow(ctx,
			`INSERT INTO run_ranked_stories (run_id, story_id, story_id_text, relevance_score,
			                                  skill_overlap, keyword_overlap, evidence_strength,
			                                  heuristic_score, llm_score, llm_reasoning,
			                                  matched_skills, notes, ordinal)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			 RETURNING id, run_id, story_id, story_id_text, relevance_score, skill_overlap,
			           keyword_overlap, evidence_strength, heuristic_score, llm_score,
			           llm_reasoning, notes, ordinal, created_at`,
			runID, input.StoryID, input.StoryIDText, input.RelevanceScore,
			input.SkillOverlap, input.KeywordOverlap, input.EvidenceStrength,
			input.HeuristicScore, input.LLMScore, nullIfEmpty(input.LLMReasoning),
			matchedSkillsJSON, nullIfEmpty(input.Notes), input.Ordinal,
		).Scan(&rs.ID, &rs.RunID, &rs.StoryID, &rs.StoryIDText, &rs.RelevanceScore,
			&rs.SkillOverlap, &rs.KeywordOverlap, &rs.EvidenceStrength,
			&rs.HeuristicScore, &rs.LLMScore, &rs.LLMReasoning, &rs.Notes,
			&rs.Ordinal, &rs.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to save ranked story: %w", err)
		}
		rs.MatchedSkills = input.MatchedSkills
		result = append(result, rs)
	}

	return result, nil
}

// GetRunRankedStories retrieves ranked stories for a pipeline run
func (db *DB) GetRunRankedStories(ctx context.Context, runID uuid.UUID) ([]RunRankedStory, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, run_id, story_id, story_id_text, relevance_score, skill_overlap,
		        keyword_overlap, evidence_strength, heuristic_score, llm_score,
		        llm_reasoning, matched_skills, notes, ordinal, created_at
		 FROM run_ranked_stories
		 WHERE run_id = $1
		 ORDER BY ordinal`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ranked stories: %w", err)
	}
	defer rows.Close()

	var stories []RunRankedStory
	for rows.Next() {
		var rs RunRankedStory
		var matchedSkillsJSON []byte
		if err := rows.Scan(&rs.ID, &rs.RunID, &rs.StoryID, &rs.StoryIDText,
			&rs.RelevanceScore, &rs.SkillOverlap, &rs.KeywordOverlap,
			&rs.EvidenceStrength, &rs.HeuristicScore, &rs.LLMScore,
			&rs.LLMReasoning, &matchedSkillsJSON, &rs.Notes, &rs.Ordinal,
			&rs.CreatedAt); err != nil {
			return nil, err
		}
		if matchedSkillsJSON != nil {
			_ = json.Unmarshal(matchedSkillsJSON, &rs.MatchedSkills)
		}
		stories = append(stories, rs)
	}
	return stories, nil
}

// -----------------------------------------------------------------------------
// Run Resume Plan Methods
// -----------------------------------------------------------------------------

// SaveRunResumePlan saves or updates a resume plan for a pipeline run
func (db *DB) SaveRunResumePlan(ctx context.Context, runID uuid.UUID, input *RunResumePlanInput) (*RunResumePlan, error) {
	var plan RunResumePlan
	var sectionBudgetsJSON, topSkillsJSON []byte
	var err error

	if len(input.SectionBudgets) > 0 {
		sectionBudgetsJSON, err = json.Marshal(input.SectionBudgets)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal section budgets: %w", err)
		}
	}
	if len(input.TopSkillsCovered) > 0 {
		topSkillsJSON, err = json.Marshal(input.TopSkillsCovered)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal top skills: %w", err)
		}
	}

	err = db.pool.QueryRow(ctx,
		`INSERT INTO run_resume_plans (run_id, max_bullets, max_lines, skill_match_ratio,
		                               section_budgets, top_skills_covered, coverage_score)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (run_id) DO UPDATE SET
		     max_bullets = $2,
		     max_lines = $3,
		     skill_match_ratio = $4,
		     section_budgets = $5,
		     top_skills_covered = $6,
		     coverage_score = $7
		 RETURNING id, run_id, max_bullets, max_lines, skill_match_ratio, coverage_score, created_at`,
		runID, input.MaxBullets, input.MaxLines, input.SkillMatchRatio,
		sectionBudgetsJSON, topSkillsJSON, input.CoverageScore,
	).Scan(&plan.ID, &plan.RunID, &plan.MaxBullets, &plan.MaxLines,
		&plan.SkillMatchRatio, &plan.CoverageScore, &plan.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to save resume plan: %w", err)
	}

	plan.SectionBudgets = input.SectionBudgets
	plan.TopSkillsCovered = input.TopSkillsCovered

	return &plan, nil
}

// GetRunResumePlan retrieves the resume plan for a pipeline run
func (db *DB) GetRunResumePlan(ctx context.Context, runID uuid.UUID) (*RunResumePlan, error) {
	var plan RunResumePlan
	var sectionBudgetsJSON, topSkillsJSON []byte

	err := db.pool.QueryRow(ctx,
		`SELECT id, run_id, max_bullets, max_lines, skill_match_ratio,
		        section_budgets, top_skills_covered, coverage_score, created_at
		 FROM run_resume_plans
		 WHERE run_id = $1`,
		runID,
	).Scan(&plan.ID, &plan.RunID, &plan.MaxBullets, &plan.MaxLines,
		&plan.SkillMatchRatio, &sectionBudgetsJSON, &topSkillsJSON,
		&plan.CoverageScore, &plan.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get resume plan: %w", err)
	}

	if sectionBudgetsJSON != nil {
		_ = json.Unmarshal(sectionBudgetsJSON, &plan.SectionBudgets)
	}
	if topSkillsJSON != nil {
		_ = json.Unmarshal(topSkillsJSON, &plan.TopSkillsCovered)
	}

	return &plan, nil
}

// -----------------------------------------------------------------------------
// Run Selected Bullets Methods
// -----------------------------------------------------------------------------

// SaveRunSelectedBullets saves selected bullets for a pipeline run
func (db *DB) SaveRunSelectedBullets(ctx context.Context, runID uuid.UUID, planID *uuid.UUID, bullets []RunSelectedBulletInput) ([]RunSelectedBullet, error) {
	// Delete existing selected bullets for this run (upsert behavior)
	_, err := db.pool.Exec(ctx, "DELETE FROM run_selected_bullets WHERE run_id = $1", runID)
	if err != nil {
		return nil, fmt.Errorf("failed to clear existing selected bullets: %w", err)
	}

	var result []RunSelectedBullet
	for _, input := range bullets {
		var sb RunSelectedBullet
		var skillsJSON []byte
		if len(input.Skills) > 0 {
			skillsJSON, _ = json.Marshal(input.Skills)
		}

		err := db.pool.QueryRow(ctx,
			`INSERT INTO run_selected_bullets (run_id, plan_id, bullet_id, bullet_id_text,
			                                    story_id, story_id_text, text, skills, metrics,
			                                    length_chars, section, ordinal)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			 RETURNING id, run_id, plan_id, bullet_id, bullet_id_text, story_id, story_id_text,
			           text, metrics, length_chars, section, ordinal, created_at`,
			runID, planID, input.BulletID, input.BulletIDText,
			input.StoryID, input.StoryIDText, input.Text, skillsJSON,
			nullIfEmpty(input.Metrics), input.LengthChars, input.Section, input.Ordinal,
		).Scan(&sb.ID, &sb.RunID, &sb.PlanID, &sb.BulletID, &sb.BulletIDText,
			&sb.StoryID, &sb.StoryIDText, &sb.Text, &sb.Metrics, &sb.LengthChars,
			&sb.Section, &sb.Ordinal, &sb.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to save selected bullet: %w", err)
		}
		sb.Skills = input.Skills
		result = append(result, sb)
	}

	return result, nil
}

// GetRunSelectedBullets retrieves selected bullets for a pipeline run
func (db *DB) GetRunSelectedBullets(ctx context.Context, runID uuid.UUID) ([]RunSelectedBullet, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, run_id, plan_id, bullet_id, bullet_id_text, story_id, story_id_text,
		        text, skills, metrics, length_chars, section, ordinal, created_at
		 FROM run_selected_bullets
		 WHERE run_id = $1
		 ORDER BY ordinal`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get selected bullets: %w", err)
	}
	defer rows.Close()

	var bullets []RunSelectedBullet
	for rows.Next() {
		var sb RunSelectedBullet
		var skillsJSON []byte
		if err := rows.Scan(&sb.ID, &sb.RunID, &sb.PlanID, &sb.BulletID, &sb.BulletIDText,
			&sb.StoryID, &sb.StoryIDText, &sb.Text, &skillsJSON, &sb.Metrics,
			&sb.LengthChars, &sb.Section, &sb.Ordinal, &sb.CreatedAt); err != nil {
			return nil, err
		}
		if skillsJSON != nil {
			_ = json.Unmarshal(skillsJSON, &sb.Skills)
		}
		bullets = append(bullets, sb)
	}
	return bullets, nil
}

// -----------------------------------------------------------------------------
// Run Rewritten Bullets Methods
// -----------------------------------------------------------------------------

// SaveRunRewrittenBullets saves rewritten bullets for a pipeline run
func (db *DB) SaveRunRewrittenBullets(ctx context.Context, runID uuid.UUID, bullets []RunRewrittenBulletInput) ([]RunRewrittenBullet, error) {
	// Delete existing rewritten bullets for this run (upsert behavior)
	_, err := db.pool.Exec(ctx, "DELETE FROM run_rewritten_bullets WHERE run_id = $1", runID)
	if err != nil {
		return nil, fmt.Errorf("failed to clear existing rewritten bullets: %w", err)
	}

	var result []RunRewrittenBullet
	for _, input := range bullets {
		var rb RunRewrittenBullet

		err := db.pool.QueryRow(ctx,
			`INSERT INTO run_rewritten_bullets (run_id, selected_bullet_id, original_bullet_id_text,
			                                     final_text, length_chars, estimated_lines,
			                                     style_strong_verb, style_quantified, style_no_taboo,
			                                     style_target_length, ordinal)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			 RETURNING id, run_id, selected_bullet_id, original_bullet_id_text, final_text,
			           length_chars, estimated_lines, style_strong_verb, style_quantified,
			           style_no_taboo, style_target_length, ordinal, created_at`,
			runID, input.SelectedBulletID, input.OriginalBulletIDText,
			input.FinalText, input.LengthChars, input.EstimatedLines,
			input.StyleStrongVerb, input.StyleQuantified, input.StyleNoTaboo,
			input.StyleTargetLength, input.Ordinal,
		).Scan(&rb.ID, &rb.RunID, &rb.SelectedBulletID, &rb.OriginalBulletIDText,
			&rb.FinalText, &rb.LengthChars, &rb.EstimatedLines, &rb.StyleStrongVerb,
			&rb.StyleQuantified, &rb.StyleNoTaboo, &rb.StyleTargetLength, &rb.Ordinal,
			&rb.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to save rewritten bullet: %w", err)
		}
		result = append(result, rb)
	}

	return result, nil
}

// GetRunRewrittenBullets retrieves rewritten bullets for a pipeline run
func (db *DB) GetRunRewrittenBullets(ctx context.Context, runID uuid.UUID) ([]RunRewrittenBullet, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, run_id, selected_bullet_id, original_bullet_id_text, final_text,
		        length_chars, estimated_lines, style_strong_verb, style_quantified,
		        style_no_taboo, style_target_length, ordinal, created_at
		 FROM run_rewritten_bullets
		 WHERE run_id = $1
		 ORDER BY ordinal`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get rewritten bullets: %w", err)
	}
	defer rows.Close()

	var bullets []RunRewrittenBullet
	for rows.Next() {
		var rb RunRewrittenBullet
		if err := rows.Scan(&rb.ID, &rb.RunID, &rb.SelectedBulletID, &rb.OriginalBulletIDText,
			&rb.FinalText, &rb.LengthChars, &rb.EstimatedLines, &rb.StyleStrongVerb,
			&rb.StyleQuantified, &rb.StyleNoTaboo, &rb.StyleTargetLength, &rb.Ordinal,
			&rb.CreatedAt); err != nil {
			return nil, err
		}
		bullets = append(bullets, rb)
	}
	return bullets, nil
}

// -----------------------------------------------------------------------------
// Run Violations Methods
// -----------------------------------------------------------------------------

// SaveRunViolations saves violations for a pipeline run
func (db *DB) SaveRunViolations(ctx context.Context, runID uuid.UUID, violations []RunViolationInput) ([]RunViolation, error) {
	// Delete existing violations for this run (upsert behavior)
	_, err := db.pool.Exec(ctx, "DELETE FROM run_violations WHERE run_id = $1", runID)
	if err != nil {
		return nil, fmt.Errorf("failed to clear existing violations: %w", err)
	}

	var result []RunViolation
	for _, input := range violations {
		var rv RunViolation
		var affectedSectionsJSON []byte
		if len(input.AffectedSections) > 0 {
			affectedSectionsJSON, _ = json.Marshal(input.AffectedSections)
		}

		err := db.pool.QueryRow(ctx,
			`INSERT INTO run_violations (run_id, violation_type, severity, details,
			                              line_number, char_count, affected_sections)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id, run_id, violation_type, severity, details, line_number,
			           char_count, created_at`,
			runID, input.ViolationType, input.Severity, nullIfEmpty(input.Details),
			input.LineNumber, input.CharCount, affectedSectionsJSON,
		).Scan(&rv.ID, &rv.RunID, &rv.ViolationType, &rv.Severity, &rv.Details,
			&rv.LineNumber, &rv.CharCount, &rv.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to save violation: %w", err)
		}
		rv.AffectedSections = input.AffectedSections
		result = append(result, rv)
	}

	return result, nil
}

// GetRunViolations retrieves violations for a pipeline run
func (db *DB) GetRunViolations(ctx context.Context, runID uuid.UUID) ([]RunViolation, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, run_id, violation_type, severity, details, line_number,
		        char_count, affected_sections, created_at
		 FROM run_violations
		 WHERE run_id = $1
		 ORDER BY created_at`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get violations: %w", err)
	}
	defer rows.Close()

	var violations []RunViolation
	for rows.Next() {
		var rv RunViolation
		var affectedSectionsJSON []byte
		if err := rows.Scan(&rv.ID, &rv.RunID, &rv.ViolationType, &rv.Severity,
			&rv.Details, &rv.LineNumber, &rv.CharCount, &affectedSectionsJSON,
			&rv.CreatedAt); err != nil {
			return nil, err
		}
		if affectedSectionsJSON != nil {
			_ = json.Unmarshal(affectedSectionsJSON, &rv.AffectedSections)
		}
		violations = append(violations, rv)
	}
	return violations, nil
}

// -----------------------------------------------------------------------------
// Query Methods
// -----------------------------------------------------------------------------

// GetRunViolationsByType retrieves violations by type across all runs
func (db *DB) GetRunViolationsByType(ctx context.Context, violationType string) ([]RunViolation, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, run_id, violation_type, severity, details, line_number,
		        char_count, affected_sections, created_at
		 FROM run_violations
		 WHERE violation_type = $1
		 ORDER BY created_at DESC`,
		violationType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get violations by type: %w", err)
	}
	defer rows.Close()

	var violations []RunViolation
	for rows.Next() {
		var rv RunViolation
		var affectedSectionsJSON []byte
		if err := rows.Scan(&rv.ID, &rv.RunID, &rv.ViolationType, &rv.Severity,
			&rv.Details, &rv.LineNumber, &rv.CharCount, &affectedSectionsJSON,
			&rv.CreatedAt); err != nil {
			return nil, err
		}
		if affectedSectionsJSON != nil {
			_ = json.Unmarshal(affectedSectionsJSON, &rv.AffectedSections)
		}
		violations = append(violations, rv)
	}
	return violations, nil
}

// GetMostSelectedBullets returns the most frequently selected bullets across runs
func (db *DB) GetMostSelectedBullets(ctx context.Context, limit int) (map[string]int, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT bullet_id_text, COUNT(*) as count
		 FROM run_selected_bullets
		 GROUP BY bullet_id_text
		 ORDER BY count DESC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get most selected bullets: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var bulletID string
		var count int
		if err := rows.Scan(&bulletID, &count); err != nil {
			return nil, err
		}
		result[bulletID] = count
	}
	return result, nil
}

// GetTopRankedStoriesForJob returns the top-ranked stories for runs targeting a specific job URL
func (db *DB) GetTopRankedStoriesForJob(ctx context.Context, jobURL string, limit int) ([]RunRankedStory, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT rs.id, rs.run_id, rs.story_id, rs.story_id_text, rs.relevance_score,
		        rs.skill_overlap, rs.keyword_overlap, rs.evidence_strength,
		        rs.heuristic_score, rs.llm_score, rs.llm_reasoning, rs.matched_skills,
		        rs.notes, rs.ordinal, rs.created_at
		 FROM run_ranked_stories rs
		 JOIN pipeline_runs pr ON rs.run_id = pr.id
		 WHERE pr.job_url = $1
		 ORDER BY rs.relevance_score DESC NULLS LAST
		 LIMIT $2`,
		jobURL, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get top ranked stories: %w", err)
	}
	defer rows.Close()

	var stories []RunRankedStory
	for rows.Next() {
		var rs RunRankedStory
		var matchedSkillsJSON []byte
		if err := rows.Scan(&rs.ID, &rs.RunID, &rs.StoryID, &rs.StoryIDText,
			&rs.RelevanceScore, &rs.SkillOverlap, &rs.KeywordOverlap,
			&rs.EvidenceStrength, &rs.HeuristicScore, &rs.LLMScore,
			&rs.LLMReasoning, &matchedSkillsJSON, &rs.Notes, &rs.Ordinal,
			&rs.CreatedAt); err != nil {
			return nil, err
		}
		if matchedSkillsJSON != nil {
			_ = json.Unmarshal(matchedSkillsJSON, &rs.MatchedSkills)
		}
		stories = append(stories, rs)
	}
	return stories, nil
}

// CountRunsByViolationStatus counts runs with/without violations
func (db *DB) CountRunsByViolationStatus(ctx context.Context) (withViolations int, withoutViolations int, err error) {
	err = db.pool.QueryRow(ctx,
		`SELECT 
		    COUNT(DISTINCT run_id) as with_violations,
		    (SELECT COUNT(*) FROM pipeline_runs) - COUNT(DISTINCT run_id) as without_violations
		 FROM run_violations`,
	).Scan(&withViolations, &withoutViolations)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count runs by violation status: %w", err)
	}
	return
}
