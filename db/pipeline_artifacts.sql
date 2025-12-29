-- Pipeline Run Artifacts Schema
-- Phase 5 of the database normalization project
-- Depends on: resumes.sql, experience_bank.sql, job_postings.sql, company_profiles.sql

-- =============================================================================
-- PIPELINE RUNS UPDATES
-- =============================================================================

-- Add foreign key references to normalized entities (if not exists)
DO $$ 
BEGIN
    -- Add user_id if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'pipeline_runs' AND column_name = 'user_id') THEN
        ALTER TABLE pipeline_runs ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    
    -- Add job_posting_id if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'pipeline_runs' AND column_name = 'job_posting_id') THEN
        ALTER TABLE pipeline_runs ADD COLUMN job_posting_id UUID REFERENCES job_postings(id) ON DELETE SET NULL;
    END IF;
    
    -- Add job_profile_id if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'pipeline_runs' AND column_name = 'job_profile_id') THEN
        ALTER TABLE pipeline_runs ADD COLUMN job_profile_id UUID REFERENCES job_profiles(id) ON DELETE SET NULL;
    END IF;
    
    -- Add company_profile_id if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'pipeline_runs' AND column_name = 'company_profile_id') THEN
        ALTER TABLE pipeline_runs ADD COLUMN company_profile_id UUID REFERENCES company_profiles(id) ON DELETE SET NULL;
    END IF;
END $$;

-- =============================================================================
-- RUN RANKED STORIES TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS run_ranked_stories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    story_id UUID REFERENCES stories(id) ON DELETE SET NULL,
    story_id_text TEXT NOT NULL,           -- original story_id string for reference
    
    -- Scores
    relevance_score NUMERIC(5,4),
    skill_overlap NUMERIC(5,4),
    keyword_overlap NUMERIC(5,4),
    evidence_strength NUMERIC(5,4),
    heuristic_score NUMERIC(5,4),
    llm_score NUMERIC(5,4),
    
    -- Metadata
    llm_reasoning TEXT,
    matched_skills JSONB,                  -- keep as JSONB for flexibility
    notes TEXT,
    ordinal INTEGER,                       -- rank order (1 = top ranked)
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- RUN RESUME PLANS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS run_resume_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE UNIQUE,
    
    -- Space budget
    max_bullets INTEGER,
    max_lines INTEGER,
    skill_match_ratio NUMERIC(3,2),
    section_budgets JSONB,                 -- map of section -> lines
    
    -- Coverage
    top_skills_covered JSONB,              -- array of skill names
    coverage_score NUMERIC(5,4),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- RUN SELECTED BULLETS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS run_selected_bullets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    plan_id UUID REFERENCES run_resume_plans(id) ON DELETE CASCADE,
    bullet_id UUID REFERENCES bullets(id) ON DELETE SET NULL,
    bullet_id_text TEXT NOT NULL,          -- original bullet_id string
    story_id UUID REFERENCES stories(id) ON DELETE SET NULL,
    story_id_text TEXT NOT NULL,           -- original story_id string
    
    -- Content snapshot (for audit)
    text TEXT NOT NULL,
    skills JSONB,
    metrics TEXT,
    length_chars INTEGER,
    
    -- Position
    section TEXT,                          -- 'experience', 'projects', etc.
    ordinal INTEGER,                       -- order within run
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- RUN REWRITTEN BULLETS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS run_rewritten_bullets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    selected_bullet_id UUID REFERENCES run_selected_bullets(id) ON DELETE CASCADE,
    original_bullet_id_text TEXT NOT NULL, -- original bullet_id string
    
    -- Content
    final_text TEXT NOT NULL,
    length_chars INTEGER,
    estimated_lines INTEGER,
    
    -- Style checks
    style_strong_verb BOOLEAN DEFAULT false,
    style_quantified BOOLEAN DEFAULT false,
    style_no_taboo BOOLEAN DEFAULT false,
    style_target_length BOOLEAN DEFAULT false,
    
    -- Position
    ordinal INTEGER,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- RUN VIOLATIONS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS run_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    
    -- Violation details
    violation_type TEXT NOT NULL,          -- 'page_overflow', 'line_too_long', etc.
    severity TEXT NOT NULL,                -- 'error', 'warning'
    details TEXT,
    line_number INTEGER,
    char_count INTEGER,
    affected_sections JSONB,               -- array of section names
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- INDEXES
-- =============================================================================

-- Pipeline runs indexes (new columns)
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_user ON pipeline_runs(user_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_job_posting ON pipeline_runs(job_posting_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_job_profile ON pipeline_runs(job_profile_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_company_profile ON pipeline_runs(company_profile_id);

-- Run ranked stories
CREATE INDEX IF NOT EXISTS idx_run_ranked_run ON run_ranked_stories(run_id);
CREATE INDEX IF NOT EXISTS idx_run_ranked_story ON run_ranked_stories(story_id);
CREATE INDEX IF NOT EXISTS idx_run_ranked_ordinal ON run_ranked_stories(run_id, ordinal);

-- Run resume plans
CREATE INDEX IF NOT EXISTS idx_run_plans_run ON run_resume_plans(run_id);

-- Run selected bullets
CREATE INDEX IF NOT EXISTS idx_run_selected_run ON run_selected_bullets(run_id);
CREATE INDEX IF NOT EXISTS idx_run_selected_plan ON run_selected_bullets(plan_id);
CREATE INDEX IF NOT EXISTS idx_run_selected_bullet ON run_selected_bullets(bullet_id);
CREATE INDEX IF NOT EXISTS idx_run_selected_story ON run_selected_bullets(story_id);

-- Run rewritten bullets
CREATE INDEX IF NOT EXISTS idx_run_rewritten_run ON run_rewritten_bullets(run_id);
CREATE INDEX IF NOT EXISTS idx_run_rewritten_selected ON run_rewritten_bullets(selected_bullet_id);

-- Run violations
CREATE INDEX IF NOT EXISTS idx_run_violations_run ON run_violations(run_id);
CREATE INDEX IF NOT EXISTS idx_run_violations_type ON run_violations(violation_type);
CREATE INDEX IF NOT EXISTS idx_run_violations_severity ON run_violations(severity);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE run_ranked_stories IS 'Ranked experience stories for each pipeline run';
COMMENT ON TABLE run_resume_plans IS 'Resume plan configuration for each pipeline run';
COMMENT ON TABLE run_selected_bullets IS 'Selected bullets for each pipeline run';
COMMENT ON TABLE run_rewritten_bullets IS 'Rewritten bullets for each pipeline run';
COMMENT ON TABLE run_violations IS 'Validation violations for each pipeline run';

COMMENT ON COLUMN run_ranked_stories.story_id_text IS 'Original story_id string from experience bank';
COMMENT ON COLUMN run_ranked_stories.ordinal IS 'Rank order (1 = highest ranked)';
COMMENT ON COLUMN run_resume_plans.section_budgets IS 'JSONB map of section name to line budget';
COMMENT ON COLUMN run_selected_bullets.bullet_id_text IS 'Original bullet_id string from experience bank';
COMMENT ON COLUMN run_rewritten_bullets.original_bullet_id_text IS 'Original bullet_id that was rewritten';
COMMENT ON COLUMN run_violations.severity IS 'error or warning';

