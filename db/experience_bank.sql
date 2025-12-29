-- Experience Bank Normalization Schema
-- Extends users.sql with skills normalization and story groupings

-- =============================================================================
-- SKILLS TABLE (Normalized skill catalog)
-- =============================================================================

CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    name_normalized TEXT NOT NULL UNIQUE,  -- lowercase, for matching
    category TEXT,                         -- 'programming', 'framework', 'database', 'soft_skill', 'tool', etc.
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- STORIES TABLE (Group experiences by project/initiative)
-- =============================================================================

CREATE TABLE IF NOT EXISTS stories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    story_id TEXT NOT NULL UNIQUE,         -- human-readable ID like 'amazon-partner-analytics'
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    title TEXT,                            -- optional title for the story
    description TEXT,                      -- optional description
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- BULLETS TABLE (Individual bullet points)
-- =============================================================================

CREATE TABLE IF NOT EXISTS bullets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bullet_id TEXT NOT NULL UNIQUE,        -- stable identifier like 'bullet_001'
    story_id UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    job_id UUID REFERENCES jobs(id) ON DELETE SET NULL,  -- denormalized for convenience
    
    -- Content
    text TEXT NOT NULL,
    metrics TEXT,                          -- quantified metrics (optional)
    length_chars INTEGER,                  -- character length of text
    
    -- Quality indicators
    evidence_strength TEXT DEFAULT 'medium', -- 'high', 'medium', 'low'
    risk_flags JSONB DEFAULT '[]',         -- array of risk flag strings
    
    -- Ordering
    ordinal INTEGER,                       -- order within story
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- BULLET SKILLS (Many-to-many: bullets <-> skills)
-- =============================================================================

CREATE TABLE IF NOT EXISTS bullet_skills (
    bullet_id UUID NOT NULL REFERENCES bullets(id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (bullet_id, skill_id)
);

-- =============================================================================
-- EDUCATION HIGHLIGHTS (Many-to-one: highlights -> education)
-- =============================================================================

CREATE TABLE IF NOT EXISTS education_highlights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    education_id UUID NOT NULL REFERENCES education(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    ordinal INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- INDEXES
-- =============================================================================

-- Skills lookups
CREATE INDEX IF NOT EXISTS idx_skills_normalized ON skills(name_normalized);
CREATE INDEX IF NOT EXISTS idx_skills_category ON skills(category);

-- Stories lookups
CREATE INDEX IF NOT EXISTS idx_stories_user ON stories(user_id);
CREATE INDEX IF NOT EXISTS idx_stories_job ON stories(job_id);
CREATE INDEX IF NOT EXISTS idx_stories_story_id ON stories(story_id);

-- Bullets lookups
CREATE INDEX IF NOT EXISTS idx_bullets_story ON bullets(story_id);
CREATE INDEX IF NOT EXISTS idx_bullets_job ON bullets(job_id);
CREATE INDEX IF NOT EXISTS idx_bullets_bullet_id ON bullets(bullet_id);
CREATE INDEX IF NOT EXISTS idx_bullets_evidence ON bullets(evidence_strength);

-- Bullet skills lookups
CREATE INDEX IF NOT EXISTS idx_bullet_skills_skill ON bullet_skills(skill_id);
CREATE INDEX IF NOT EXISTS idx_bullet_skills_bullet ON bullet_skills(bullet_id);

-- Education highlights lookups
CREATE INDEX IF NOT EXISTS idx_education_highlights_education ON education_highlights(education_id);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE skills IS 'Normalized skill catalog for experience matching';
COMMENT ON TABLE stories IS 'Groups of related experience bullets (projects/initiatives)';
COMMENT ON TABLE bullets IS 'Individual experience bullet points linked to stories';
COMMENT ON TABLE bullet_skills IS 'Many-to-many relationship between bullets and skills';
COMMENT ON TABLE education_highlights IS 'Notable achievements for education entries';

COMMENT ON COLUMN skills.name_normalized IS 'Lowercase skill name for case-insensitive matching';
COMMENT ON COLUMN skills.category IS 'Skill category: programming, framework, database, tool, soft_skill';
COMMENT ON COLUMN stories.story_id IS 'Human-readable stable identifier (e.g., amazon-partner-analytics)';
COMMENT ON COLUMN bullets.bullet_id IS 'Stable identifier matching experience_bank.json format';
COMMENT ON COLUMN bullets.evidence_strength IS 'Quality indicator: high, medium, low';
COMMENT ON COLUMN bullets.risk_flags IS 'JSONB array of risk flags (needs_citation, unclear_metric)';

