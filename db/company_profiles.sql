-- Company Profiles & Brand Signals Schema

-- =============================================================================
-- COMPANY PROFILES TABLE
-- =============================================================================

-- Summarized company voice/style (one active profile per company)
CREATE TABLE IF NOT EXISTS company_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Core voice attributes
    tone TEXT NOT NULL,                 -- e.g., 'direct and technical', 'warm and inclusive'
    domain_context TEXT,                -- e.g., 'FinTech, consumer finance'
    source_corpus TEXT,                 -- aggregated corpus text used for summarization
    
    -- Versioning and freshness
    version INTEGER DEFAULT 1,          -- increment on regeneration
    last_verified_at TIMESTAMPTZ,       -- when we last confirmed sources still exist
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- One active profile per company
    UNIQUE(company_id)
);

-- =============================================================================
-- STYLE RULES TABLE
-- =============================================================================

-- Style rules extracted from company profile
CREATE TABLE IF NOT EXISTS company_style_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES company_profiles(id) ON DELETE CASCADE,
    rule_text TEXT NOT NULL,
    priority INTEGER DEFAULT 0,         -- for ordering (higher = more important)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- TABOO PHRASES TABLE
-- =============================================================================

-- Phrases to avoid when writing for this company
CREATE TABLE IF NOT EXISTS company_taboo_phrases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES company_profiles(id) ON DELETE CASCADE,
    phrase TEXT NOT NULL,
    reason TEXT,                        -- why it's taboo
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- COMPANY VALUES TABLE
-- =============================================================================

-- Company values extracted from crawled pages
CREATE TABLE IF NOT EXISTS company_values (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES company_profiles(id) ON DELETE CASCADE,
    value_text TEXT NOT NULL,
    priority INTEGER DEFAULT 0,         -- for ordering
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- PROFILE SOURCES TABLE
-- =============================================================================

-- Evidence URLs linking profile to crawled pages
CREATE TABLE IF NOT EXISTS company_profile_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES company_profiles(id) ON DELETE CASCADE,
    crawled_page_id UUID REFERENCES crawled_pages(id) ON DELETE SET NULL,
    url TEXT NOT NULL,                  -- original URL (may not be in crawled_pages)
    source_type TEXT,                   -- 'values', 'culture', 'about', 'engineering', etc.
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- BRAND SIGNALS TABLE
-- =============================================================================

-- Brand signals extracted from individual crawled pages (intermediate data)
CREATE TABLE IF NOT EXISTS brand_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crawled_page_id UUID NOT NULL REFERENCES crawled_pages(id) ON DELETE CASCADE,
    signal_type TEXT,                   -- 'culture', 'values', 'engineering', 'mission', etc.
    key_points JSONB,                   -- array of extracted points
    extracted_values JSONB,             -- array of values found
    raw_excerpt TEXT,                   -- relevant text excerpt from page
    confidence_score NUMERIC(3,2),      -- 0.00 to 1.00
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- INDEXES
-- =============================================================================

-- Company profiles lookups
CREATE INDEX IF NOT EXISTS idx_company_profiles_company ON company_profiles(company_id);
CREATE INDEX IF NOT EXISTS idx_company_profiles_updated ON company_profiles(updated_at DESC);

-- Style rules lookups
CREATE INDEX IF NOT EXISTS idx_style_rules_profile ON company_style_rules(profile_id);
CREATE INDEX IF NOT EXISTS idx_style_rules_priority ON company_style_rules(profile_id, priority DESC);

-- Taboo phrases lookups
CREATE INDEX IF NOT EXISTS idx_taboo_phrases_profile ON company_taboo_phrases(profile_id);
CREATE INDEX IF NOT EXISTS idx_taboo_phrases_phrase ON company_taboo_phrases(phrase);

-- Company values lookups
CREATE INDEX IF NOT EXISTS idx_company_values_profile ON company_values(profile_id);
CREATE INDEX IF NOT EXISTS idx_company_values_priority ON company_values(profile_id, priority DESC);

-- Profile sources lookups
CREATE INDEX IF NOT EXISTS idx_profile_sources_profile ON company_profile_sources(profile_id);
CREATE INDEX IF NOT EXISTS idx_profile_sources_page ON company_profile_sources(crawled_page_id);

-- Brand signals lookups
CREATE INDEX IF NOT EXISTS idx_brand_signals_page ON brand_signals(crawled_page_id);
CREATE INDEX IF NOT EXISTS idx_brand_signals_type ON brand_signals(signal_type);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE company_profiles IS 'Summarized company voice/style, one per company';
COMMENT ON TABLE company_style_rules IS 'Writing style rules extracted from company content';
COMMENT ON TABLE company_taboo_phrases IS 'Phrases to avoid when writing for this company';
COMMENT ON TABLE company_values IS 'Core values extracted from company content';
COMMENT ON TABLE company_profile_sources IS 'URLs used as evidence for profile generation';
COMMENT ON TABLE brand_signals IS 'Raw brand signals extracted from individual pages';

COMMENT ON COLUMN company_profiles.tone IS 'Overall writing tone (e.g., direct, warm, technical)';
COMMENT ON COLUMN company_profiles.domain_context IS 'Industry/domain context for the company';
COMMENT ON COLUMN company_profiles.source_corpus IS 'Aggregated text corpus used for LLM summarization';
COMMENT ON COLUMN company_profiles.version IS 'Incremented when profile is regenerated';
COMMENT ON COLUMN company_profiles.last_verified_at IS 'When source URLs were last verified as accessible';
COMMENT ON COLUMN brand_signals.confidence_score IS 'How confident the extraction was (0.00-1.00)';

