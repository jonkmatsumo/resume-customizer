-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    phone TEXT,
    password_hash TEXT NOT NULL DEFAULT '',
    password_set BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Migration: Add password authentication fields (if table already exists)
-- Run these ALTER TABLE statements if the table was created before this migration:
-- ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';
-- ALTER TABLE users ADD COLUMN IF NOT EXISTS password_set BOOLEAN DEFAULT FALSE;
-- ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();
-- UPDATE users SET password_set = FALSE WHERE password_hash = '';

-- Jobs table (employment history)
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    company TEXT NOT NULL,
    role_title TEXT NOT NULL,
    location TEXT,
    employment_type TEXT DEFAULT 'full-time',
    start_date DATE,
    end_date DATE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Experiences table (bullet points within jobs)
CREATE TABLE experiences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    bullet_text TEXT NOT NULL,
    skills JSONB DEFAULT '[]',
    evidence_strength TEXT DEFAULT 'medium',
    risk_flags JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Education table
CREATE TABLE education (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    school TEXT NOT NULL,
    degree_type TEXT,
    field TEXT,
    gpa TEXT,
    location TEXT,
    start_date DATE,
    end_date DATE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_jobs_user ON jobs(user_id);
CREATE INDEX idx_experiences_job ON experiences(job_id);
CREATE INDEX idx_education_user ON education(user_id);
CREATE INDEX idx_experiences_skills ON experiences USING GIN (skills);
