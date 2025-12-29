-- Run Steps Schema
-- Depends on: resumes.sql (pipeline_runs)

-- =============================================================================
-- RUN STEPS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS run_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    step VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,
    artifact_id UUID,  -- References to various artifact tables (flexible)
    error_message TEXT,
    parameters JSONB,  -- Step execution parameters
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(run_id, step)
);

-- =============================================================================
-- RUN CHECKPOINTS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS run_checkpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    step VARCHAR(100) NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    artifacts JSONB,  -- Map of step -> artifact_id
    metadata JSONB,   -- Additional checkpoint metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(run_id, step)
);

-- =============================================================================
-- INDEXES
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_run_steps_run_id ON run_steps(run_id);
CREATE INDEX IF NOT EXISTS idx_run_steps_status ON run_steps(status);
CREATE INDEX IF NOT EXISTS idx_run_steps_category ON run_steps(category);
CREATE INDEX IF NOT EXISTS idx_run_steps_step ON run_steps(step);

CREATE INDEX IF NOT EXISTS idx_run_checkpoints_run_id ON run_checkpoints(run_id);
CREATE INDEX IF NOT EXISTS idx_run_checkpoints_step ON run_checkpoints(step);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE run_steps IS 'Tracks execution status of individual pipeline steps';
COMMENT ON TABLE run_checkpoints IS 'Stores checkpoint state after each completed step';

COMMENT ON COLUMN run_steps.status IS 'pending, in_progress, completed, failed, skipped, blocked';
COMMENT ON COLUMN run_steps.category IS 'ingestion, experience, research, rewriting, validation';
COMMENT ON COLUMN run_steps.artifact_id IS 'Flexible reference to various artifact tables';
COMMENT ON COLUMN run_checkpoints.artifacts IS 'JSONB map of step name to artifact_id';

