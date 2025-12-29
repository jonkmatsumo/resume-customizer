-- Research Sessions Schema
-- Phase 6 of the database normalization project
-- Depends on: companies.sql, resumes.sql (pipeline_runs), companies.sql (crawled_pages)

-- =============================================================================
-- RESEARCH SESSIONS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS research_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    run_id UUID REFERENCES pipeline_runs(id) ON DELETE SET NULL,
    
    -- Company info (denormalized for convenience)
    company_name TEXT NOT NULL,
    domain TEXT,
    
    -- Status tracking
    status TEXT DEFAULT 'pending',    -- 'pending', 'in_progress', 'completed', 'failed'
    error_message TEXT,               -- Error details if failed
    
    -- Progress
    pages_crawled INTEGER DEFAULT 0,
    pages_limit INTEGER DEFAULT 5,
    
    -- Corpus output
    corpus_text TEXT,                 -- Aggregated crawled text
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

-- =============================================================================
-- RESEARCH FRONTIER TABLE (URLs to crawl)
-- =============================================================================

CREATE TABLE IF NOT EXISTS research_frontier (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES research_sessions(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    
    -- Prioritization
    priority NUMERIC(3,2),            -- 0.00-1.00, higher = more relevant
    page_type TEXT,                   -- 'values', 'culture', 'engineering', 'about', 'careers', 'press', 'other'
    reason TEXT,                      -- Why this URL is relevant
    
    -- Status tracking
    status TEXT DEFAULT 'pending',    -- 'pending', 'fetched', 'skipped', 'failed'
    skip_reason TEXT,                 -- If skipped, why
    error_message TEXT,               -- If failed, error details
    
    -- Crawl result link (if fetched)
    crawled_page_id UUID REFERENCES crawled_pages(id) ON DELETE SET NULL,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    fetched_at TIMESTAMPTZ
);

-- =============================================================================
-- RESEARCH BRAND SIGNALS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS research_brand_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES research_sessions(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    signal_type TEXT,                 -- 'values', 'culture', 'engineering', 'press'
    key_points JSONB,                 -- Array of extracted points
    values_found JSONB,               -- Array of inferred values
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- INDEXES
-- =============================================================================

-- Research sessions
CREATE INDEX IF NOT EXISTS idx_research_sessions_company ON research_sessions(company_id);
CREATE INDEX IF NOT EXISTS idx_research_sessions_run ON research_sessions(run_id);
CREATE INDEX IF NOT EXISTS idx_research_sessions_status ON research_sessions(status);
CREATE INDEX IF NOT EXISTS idx_research_sessions_created ON research_sessions(created_at DESC);

-- Research frontier
CREATE INDEX IF NOT EXISTS idx_research_frontier_session ON research_frontier(session_id);
CREATE INDEX IF NOT EXISTS idx_research_frontier_status ON research_frontier(status);
CREATE INDEX IF NOT EXISTS idx_research_frontier_priority ON research_frontier(session_id, priority DESC);

-- Research brand signals
CREATE INDEX IF NOT EXISTS idx_research_brand_signals_session ON research_brand_signals(session_id);
CREATE INDEX IF NOT EXISTS idx_research_brand_signals_type ON research_brand_signals(signal_type);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE research_sessions IS 'Tracks research sessions for company brand discovery';
COMMENT ON TABLE research_frontier IS 'URL queue for research crawling with priority';
COMMENT ON TABLE research_brand_signals IS 'Extracted brand signals from crawled pages';

COMMENT ON COLUMN research_sessions.status IS 'pending, in_progress, completed, failed';
COMMENT ON COLUMN research_frontier.priority IS 'Crawl priority: 0.00-1.00, higher = more relevant';
COMMENT ON COLUMN research_frontier.page_type IS 'values, culture, engineering, about, careers, press, other';
COMMENT ON COLUMN research_frontier.status IS 'pending, fetched, skipped, failed';

