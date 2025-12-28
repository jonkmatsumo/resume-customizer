-- Companies & Crawled Pages Schema

-- =============================================================================
-- COMPANIES TABLE
-- =============================================================================

-- Canonical company records (deduplicated by normalized name)
CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    name_normalized TEXT NOT NULL,  -- lowercase, alphanumeric only, for matching
    domain TEXT,                    -- primary domain (e.g., 'affirm.com')
    industry TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(name_normalized)
);

-- All domains associated with a company (tech blog, investor relations, etc.)
CREATE TABLE IF NOT EXISTS company_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    domain TEXT NOT NULL,
    domain_type TEXT DEFAULT 'primary',  -- 'primary', 'tech_blog', 'investor_relations'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(domain)
);

-- =============================================================================
-- CRAWLED PAGES TABLE
-- =============================================================================

-- Cached crawled pages with TTL for cache invalidation
CREATE TABLE IF NOT EXISTS crawled_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    url TEXT NOT NULL UNIQUE,
    page_type TEXT,                 -- 'values', 'culture', 'about', 'careers', 'engineering', 'other'
    raw_html TEXT,                  -- original HTML (can be large)
    parsed_text TEXT,               -- cleaned/extracted text
    content_hash TEXT,              -- SHA-256 for change detection
    http_status INTEGER,
    -- Error tracking fields
    fetch_status TEXT DEFAULT 'success',  -- 'success', 'error', 'not_found', 'timeout', 'blocked'
    error_message TEXT,             -- error details if fetch failed
    is_permanent_failure BOOLEAN DEFAULT FALSE,  -- true = never retry (404, 410, etc.)
    retry_count INTEGER DEFAULT 0,  -- number of failed attempts
    retry_after TIMESTAMPTZ,        -- don't retry before this time
    -- Timestamps
    fetched_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,         -- TTL for cache invalidation
    last_accessed_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- INDEXES
-- =============================================================================

-- Companies lookups
CREATE INDEX IF NOT EXISTS idx_companies_name ON companies(name_normalized);
CREATE INDEX IF NOT EXISTS idx_companies_domain ON companies(domain);

-- Company domains lookups
CREATE INDEX IF NOT EXISTS idx_company_domains_company ON company_domains(company_id);
CREATE INDEX IF NOT EXISTS idx_company_domains_domain ON company_domains(domain);

-- Crawled pages lookups
CREATE INDEX IF NOT EXISTS idx_crawled_pages_company ON crawled_pages(company_id);
CREATE INDEX IF NOT EXISTS idx_crawled_pages_type ON crawled_pages(page_type);
CREATE INDEX IF NOT EXISTS idx_crawled_pages_expires ON crawled_pages(expires_at);
CREATE INDEX IF NOT EXISTS idx_crawled_pages_hash ON crawled_pages(content_hash);
CREATE INDEX IF NOT EXISTS idx_crawled_pages_fetched ON crawled_pages(fetched_at DESC);
CREATE INDEX IF NOT EXISTS idx_crawled_pages_status ON crawled_pages(fetch_status);
CREATE INDEX IF NOT EXISTS idx_crawled_pages_permanent_fail ON crawled_pages(is_permanent_failure) WHERE is_permanent_failure = TRUE;

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE companies IS 'Canonical company records, deduplicated by normalized name';
COMMENT ON TABLE company_domains IS 'All domains associated with a company';
COMMENT ON TABLE crawled_pages IS 'Cached web pages with TTL for reuse';

COMMENT ON COLUMN companies.name_normalized IS 'Lowercase alphanumeric name for deduplication';
COMMENT ON COLUMN crawled_pages.content_hash IS 'SHA-256 hash for detecting page changes';
COMMENT ON COLUMN crawled_pages.expires_at IS 'When cache should be considered stale';
COMMENT ON COLUMN crawled_pages.fetch_status IS 'success, error, not_found, timeout, blocked';
COMMENT ON COLUMN crawled_pages.is_permanent_failure IS 'True for 404/410 - never retry these URLs';
COMMENT ON COLUMN crawled_pages.retry_after IS 'Retry backoff (1min→5min→25min→2hr cap) - do not retry before this time';
COMMENT ON COLUMN crawled_pages.retry_count IS 'Number of failed attempts, used for backoff calculation';

