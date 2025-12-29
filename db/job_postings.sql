-- Job Postings & Job Profiles Schema

-- =============================================================================
-- JOB POSTINGS TABLE
-- =============================================================================

-- Raw job posting data (fetched from job boards)
CREATE TABLE IF NOT EXISTS job_postings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    url TEXT NOT NULL UNIQUE,
    role_title TEXT,
    platform TEXT,                  -- 'greenhouse', 'lever', 'linkedin', 'workday', etc.
    raw_html TEXT,
    cleaned_text TEXT,
    content_hash TEXT,              -- SHA-256 for change detection
    about_company TEXT,             -- extracted company blurb from posting
    admin_info JSONB,               -- salary, location, remote policy, etc.
    extracted_links JSONB,          -- array of URLs found in posting
    
    -- Caching and freshness
    http_status INTEGER,
    fetch_status TEXT NOT NULL DEFAULT 'pending', -- 'success', 'error', 'not_found'
    error_message TEXT,
    fetched_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,         -- when to re-fetch (default: 24 hours)
    last_accessed_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- JOB PROFILES TABLE
-- =============================================================================

-- Parsed/structured job profile (LLM-extracted from posting)
CREATE TABLE IF NOT EXISTS job_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    posting_id UUID REFERENCES job_postings(id) ON DELETE CASCADE UNIQUE,
    
    -- Denormalized for convenience
    company_name TEXT NOT NULL,
    role_title TEXT NOT NULL,
    
    -- Evaluation signals (booleans for quick filtering)
    eval_latency BOOLEAN DEFAULT false,
    eval_reliability BOOLEAN DEFAULT false,
    eval_ownership BOOLEAN DEFAULT false,
    eval_scale BOOLEAN DEFAULT false,
    eval_collaboration BOOLEAN DEFAULT false,
    eval_signals_raw JSONB,         -- original LLM output for reference
    
    -- Education requirements
    education_min_degree TEXT,      -- 'none', 'associate', 'bachelor', 'master', 'phd'
    education_preferred_fields JSONB, -- array of strings
    education_is_required BOOLEAN DEFAULT false,
    education_evidence TEXT,        -- quote from posting
    
    -- Parsing metadata
    parsed_at TIMESTAMPTZ DEFAULT NOW(),
    parser_version TEXT,            -- for re-parsing if schema changes
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- JOB RESPONSIBILITIES TABLE
-- =============================================================================

-- Responsibilities listed in job posting
CREATE TABLE IF NOT EXISTS job_responsibilities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_profile_id UUID NOT NULL REFERENCES job_profiles(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    ordinal INTEGER,                -- ordering (1, 2, 3...)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- JOB REQUIREMENTS TABLE
-- =============================================================================

-- Skill requirements (hard requirements and nice-to-haves)
CREATE TABLE IF NOT EXISTS job_requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_profile_id UUID NOT NULL REFERENCES job_profiles(id) ON DELETE CASCADE,
    requirement_type TEXT NOT NULL, -- 'hard' or 'nice_to_have'
    skill TEXT NOT NULL,
    level TEXT,                     -- '3+ years', 'proficient', 'familiar', 'expert'
    evidence TEXT,                  -- quote from posting
    ordinal INTEGER,                -- ordering within type
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- JOB KEYWORDS TABLE
-- =============================================================================

-- Keywords extracted from posting (for matching)
CREATE TABLE IF NOT EXISTS job_keywords (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_profile_id UUID NOT NULL REFERENCES job_profiles(id) ON DELETE CASCADE,
    keyword TEXT NOT NULL,
    keyword_normalized TEXT NOT NULL, -- lowercase for matching
    source TEXT,                    -- where in posting it was found
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- INDEXES
-- =============================================================================

-- Job postings lookups
CREATE INDEX IF NOT EXISTS idx_job_postings_company ON job_postings(company_id);
CREATE INDEX IF NOT EXISTS idx_job_postings_url ON job_postings(url);
CREATE INDEX IF NOT EXISTS idx_job_postings_hash ON job_postings(content_hash);
CREATE INDEX IF NOT EXISTS idx_job_postings_platform ON job_postings(platform);
CREATE INDEX IF NOT EXISTS idx_job_postings_expires ON job_postings(expires_at);
CREATE INDEX IF NOT EXISTS idx_job_postings_status ON job_postings(fetch_status);

-- Job profiles lookups
CREATE INDEX IF NOT EXISTS idx_job_profiles_posting ON job_profiles(posting_id);
CREATE INDEX IF NOT EXISTS idx_job_profiles_company ON job_profiles(company_name);
CREATE INDEX IF NOT EXISTS idx_job_profiles_role ON job_profiles(role_title);

-- Job responsibilities lookups
CREATE INDEX IF NOT EXISTS idx_job_responsibilities_profile ON job_responsibilities(job_profile_id);

-- Job requirements lookups
CREATE INDEX IF NOT EXISTS idx_job_requirements_profile ON job_requirements(job_profile_id);
CREATE INDEX IF NOT EXISTS idx_job_requirements_type ON job_requirements(requirement_type);
CREATE INDEX IF NOT EXISTS idx_job_requirements_skill ON job_requirements(skill);

-- Job keywords lookups
CREATE INDEX IF NOT EXISTS idx_job_keywords_profile ON job_keywords(job_profile_id);
CREATE INDEX IF NOT EXISTS idx_job_keywords_normalized ON job_keywords(keyword_normalized);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE job_postings IS 'Raw job posting data fetched from job boards';
COMMENT ON TABLE job_profiles IS 'Parsed/structured job profile extracted from posting';
COMMENT ON TABLE job_responsibilities IS 'Responsibilities listed in job posting';
COMMENT ON TABLE job_requirements IS 'Skill requirements (hard and nice-to-have)';
COMMENT ON TABLE job_keywords IS 'Keywords extracted from posting for matching';

COMMENT ON COLUMN job_postings.platform IS 'Job board platform (greenhouse, lever, linkedin, etc.)';
COMMENT ON COLUMN job_postings.content_hash IS 'SHA-256 hash for detecting changes to posting';
COMMENT ON COLUMN job_postings.admin_info IS 'Structured data: salary, location, remote policy';
COMMENT ON COLUMN job_profiles.eval_signals_raw IS 'Original LLM output for evaluation signals';
COMMENT ON COLUMN job_profiles.parser_version IS 'Version of parser used, for re-parsing on schema changes';
COMMENT ON COLUMN job_requirements.requirement_type IS 'Either "hard" or "nice_to_have"';
COMMENT ON COLUMN job_keywords.keyword_normalized IS 'Lowercase keyword for case-insensitive matching';

