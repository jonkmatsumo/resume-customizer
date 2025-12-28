package db

import (
	"time"

	"github.com/google/uuid"
)

// DefaultProfileCacheTTL is how long before a profile is considered stale
const DefaultProfileCacheTTL = 30 * 24 * time.Hour // 30 days

// CompanyProfile represents a summarized company voice/style
type CompanyProfile struct {
	ID             uuid.UUID  `json:"id"`
	CompanyID      uuid.UUID  `json:"company_id"`
	Company        *Company   `json:"company,omitempty"` // joined
	Tone           string     `json:"tone"`
	DomainContext  *string    `json:"domain_context,omitempty"`
	SourceCorpus   *string    `json:"-"` // Don't serialize (large)
	Version        int        `json:"version"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Denormalized for convenience (loaded via separate queries)
	StyleRules   []string `json:"style_rules,omitempty"`
	TabooPhrases []string `json:"taboo_phrases,omitempty"`
	Values       []string `json:"values,omitempty"`
	EvidenceURLs []string `json:"evidence_urls,omitempty"`
}

// CompanyStyleRule represents a writing style rule
type CompanyStyleRule struct {
	ID        uuid.UUID `json:"id"`
	ProfileID uuid.UUID `json:"profile_id"`
	RuleText  string    `json:"rule_text"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
}

// CompanyTabooPhrase represents a phrase to avoid
type CompanyTabooPhrase struct {
	ID        uuid.UUID `json:"id"`
	ProfileID uuid.UUID `json:"profile_id"`
	Phrase    string    `json:"phrase"`
	Reason    *string   `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// CompanyValue represents a core company value
type CompanyValue struct {
	ID        uuid.UUID `json:"id"`
	ProfileID uuid.UUID `json:"profile_id"`
	ValueText string    `json:"value_text"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
}

// CompanyProfileSource links a profile to its evidence URLs
type CompanyProfileSource struct {
	ID            uuid.UUID  `json:"id"`
	ProfileID     uuid.UUID  `json:"profile_id"`
	CrawledPageID *uuid.UUID `json:"crawled_page_id,omitempty"`
	URL           string     `json:"url"`
	SourceType    *string    `json:"source_type,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// BrandSignal represents extracted signals from a crawled page
type BrandSignal struct {
	ID              uuid.UUID `json:"id"`
	CrawledPageID   uuid.UUID `json:"crawled_page_id"`
	URL             string    `json:"url,omitempty"` // from joined crawled_pages
	SignalType      *string   `json:"signal_type,omitempty"`
	KeyPoints       []string  `json:"key_points,omitempty"`
	ExtractedValues []string  `json:"extracted_values,omitempty"`
	RawExcerpt      *string   `json:"raw_excerpt,omitempty"`
	ConfidenceScore *float64  `json:"confidence_score,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// SignalType constants for brand signals
const (
	SignalTypeCulture     = "culture"
	SignalTypeValues      = "values"
	SignalTypeEngineering = "engineering"
	SignalTypeMission     = "mission"
	SignalTypeProduct     = "product"
	SignalTypeTeam        = "team"
)

// SourceType constants for profile sources
const (
	SourceTypeValues      = "values"
	SourceTypeCulture     = "culture"
	SourceTypeAbout       = "about"
	SourceTypeCareers     = "careers"
	SourceTypeEngineering = "engineering"
	SourceTypeBlog        = "blog"
)

// ProfileCreateInput is used when creating a new company profile
type ProfileCreateInput struct {
	CompanyID     uuid.UUID
	Tone          string
	DomainContext string
	SourceCorpus  string
	StyleRules    []string
	TabooPhrases  []TabooPhraseInput
	Values        []string
	EvidenceURLs  []ProfileSourceInput
}

// TabooPhraseInput is used when adding a taboo phrase
type TabooPhraseInput struct {
	Phrase string
	Reason string
}

// ProfileSourceInput is used when adding a profile source
type ProfileSourceInput struct {
	URL           string
	CrawledPageID *uuid.UUID
	SourceType    string
}

// IsStale returns true if the profile hasn't been verified recently
func (p *CompanyProfile) IsStale(maxAge time.Duration) bool {
	if p.LastVerifiedAt == nil {
		return true
	}
	return time.Since(*p.LastVerifiedAt) > maxAge
}

// NeedsUpdate returns true if the profile version is outdated
func (p *CompanyProfile) NeedsUpdate(currentVersion int) bool {
	return p.Version < currentVersion
}
