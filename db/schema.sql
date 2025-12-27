-- Resume Customizer Database Schema
-- This file is loaded automatically by PostgreSQL on first container start

-- Pipeline runs table: tracks each execution of the resume pipeline
CREATE TABLE pipeline_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company TEXT,
    role_title TEXT,
    job_url TEXT,
    status TEXT DEFAULT 'running',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Artifacts table: stores all intermediate outputs from each pipeline step
CREATE TABLE artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    step TEXT NOT NULL,  -- 'job_posting', 'job_profile', 'ranked_stories', etc.
    content JSONB,       -- for structured JSON data
    text_content TEXT,   -- for .txt/.tex files
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(run_id, step)
);

-- Indexes for common query patterns
CREATE INDEX idx_runs_company ON pipeline_runs(company);
CREATE INDEX idx_runs_created ON pipeline_runs(created_at DESC);
CREATE INDEX idx_artifacts_step ON artifacts(step);
CREATE INDEX idx_artifacts_run ON artifacts(run_id);

-- Comments for documentation
COMMENT ON TABLE pipeline_runs IS 'Tracks individual pipeline executions';
COMMENT ON TABLE artifacts IS 'Stores intermediate artifacts from each pipeline step';
COMMENT ON COLUMN artifacts.step IS 'Step name: job_posting, job_profile, ranked_stories, resume_plan, selected_bullets, company_profile, rewritten_bullets, violations, resume_tex';
