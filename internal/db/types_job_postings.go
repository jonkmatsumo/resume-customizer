package db

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DefaultJobPostingCacheTTL is how long before a job posting is considered stale
const DefaultJobPostingCacheTTL = 24 * time.Hour

// Platform constants for job boards
const (
	PlatformGreenhouse = "greenhouse"
	PlatformLever      = "lever"
	PlatformLinkedIn   = "linkedin"
	PlatformWorkday    = "workday"
	PlatformAshby      = "ashby"
	PlatformUnknown    = "unknown"
)

// RequirementType constants
const (
	RequirementTypeHard       = "hard"
	RequirementTypeNiceToHave = "nice_to_have"
)

// EducationDegree constants
const (
	DegreeNone      = "none"
	DegreeAssociate = "associate"
	DegreeBachelor  = "bachelor"
	DegreeMaster    = "master"
	DegreePhD       = "phd"
)

// JobPosting represents a raw job posting fetched from a job board
type JobPosting struct {
	ID             uuid.UUID  `json:"id"`
	CompanyID      *uuid.UUID `json:"company_id,omitempty"`
	Company        *Company   `json:"company,omitempty"` // joined
	URL            string     `json:"url"`
	RoleTitle      *string    `json:"role_title,omitempty"`
	Platform       *string    `json:"platform,omitempty"`
	RawHTML        *string    `json:"-"` // Don't serialize (large)
	CleanedText    *string    `json:"cleaned_text,omitempty"`
	ContentHash    *string    `json:"content_hash,omitempty"`
	AboutCompany   *string    `json:"about_company,omitempty"`
	AdminInfo      *AdminInfo `json:"admin_info,omitempty"`
	ExtractedLinks []string   `json:"extracted_links,omitempty"`

	// Caching
	HTTPStatus   *int       `json:"http_status,omitempty"`
	FetchStatus  string     `json:"fetch_status"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	FetchedAt    time.Time  `json:"fetched_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	LastAccessed time.Time  `json:"last_accessed_at"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AdminInfo contains structured administrative info about a job
type AdminInfo struct {
	Salary         *string `json:"salary,omitempty"`
	SalaryMin      *int    `json:"salary_min,omitempty"`
	SalaryMax      *int    `json:"salary_max,omitempty"`
	Location       *string `json:"location,omitempty"`
	RemotePolicy   *string `json:"remote_policy,omitempty"`   // 'remote', 'hybrid', 'onsite'
	EmploymentType *string `json:"employment_type,omitempty"` // 'full_time', 'contract', etc.
}

// JobProfile represents a parsed/structured job profile
type JobProfile struct {
	ID        uuid.UUID   `json:"id"`
	PostingID uuid.UUID   `json:"posting_id"`
	Posting   *JobPosting `json:"posting,omitempty"` // joined

	// Denormalized
	CompanyName string `json:"company_name"`
	RoleTitle   string `json:"role_title"`

	// Evaluation signals
	EvalLatency       bool                   `json:"eval_latency"`
	EvalReliability   bool                   `json:"eval_reliability"`
	EvalOwnership     bool                   `json:"eval_ownership"`
	EvalScale         bool                   `json:"eval_scale"`
	EvalCollaboration bool                   `json:"eval_collaboration"`
	EvalSignalsRaw    map[string]interface{} `json:"eval_signals_raw,omitempty"`

	// Education
	EducationMinDegree       *string  `json:"education_min_degree,omitempty"`
	EducationPreferredFields []string `json:"education_preferred_fields,omitempty"`
	EducationIsRequired      bool     `json:"education_is_required"`
	EducationEvidence        *string  `json:"education_evidence,omitempty"`

	// Parsing metadata
	ParsedAt      time.Time `json:"parsed_at"`
	ParserVersion *string   `json:"parser_version,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Denormalized relations (loaded via separate queries)
	Responsibilities []string         `json:"responsibilities,omitempty"`
	HardRequirements []JobRequirement `json:"hard_requirements,omitempty"`
	NiceToHaves      []JobRequirement `json:"nice_to_haves,omitempty"`
	Keywords         []string         `json:"keywords,omitempty"`
}

// JobResponsibility represents a responsibility from a job posting
type JobResponsibility struct {
	ID           uuid.UUID `json:"id"`
	JobProfileID uuid.UUID `json:"job_profile_id"`
	Text         string    `json:"text"`
	Ordinal      int       `json:"ordinal"`
	CreatedAt    time.Time `json:"created_at"`
}

// JobRequirement represents a skill requirement
type JobRequirement struct {
	ID              uuid.UUID `json:"id"`
	JobProfileID    uuid.UUID `json:"job_profile_id"`
	RequirementType string    `json:"requirement_type"` // 'hard' or 'nice_to_have'
	Skill           string    `json:"skill"`
	Level           *string   `json:"level,omitempty"`
	Evidence        *string   `json:"evidence,omitempty"`
	Ordinal         int       `json:"ordinal"`
	CreatedAt       time.Time `json:"created_at"`
}

// JobKeyword represents an extracted keyword
type JobKeyword struct {
	ID                uuid.UUID `json:"id"`
	JobProfileID      uuid.UUID `json:"job_profile_id"`
	Keyword           string    `json:"keyword"`
	KeywordNormalized string    `json:"keyword_normalized"`
	Source            *string   `json:"source,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// JobPostingCreateInput is used when creating a new job posting
type JobPostingCreateInput struct {
	URL          string
	CompanyID    *uuid.UUID
	RoleTitle    string
	Platform     string
	RawHTML      string
	CleanedText  string
	AboutCompany string
	AdminInfo    *AdminInfo
	Links        []string
	HTTPStatus   int
}

// JobProfileCreateInput is used when creating a new job profile
type JobProfileCreateInput struct {
	PostingID                uuid.UUID
	CompanyName              string
	RoleTitle                string
	EvalLatency              bool
	EvalReliability          bool
	EvalOwnership            bool
	EvalScale                bool
	EvalCollaboration        bool
	EvalSignalsRaw           map[string]interface{}
	EducationMinDegree       string
	EducationPreferredFields []string
	EducationIsRequired      bool
	EducationEvidence        string
	Responsibilities         []string
	HardRequirements         []RequirementInput
	NiceToHaves              []RequirementInput
	Keywords                 []string
	ParserVersion            string
}

// RequirementInput is used when adding a requirement
type RequirementInput struct {
	Skill    string
	Level    string
	Evidence string
}

// IsFresh returns true if the posting hasn't expired
func (p *JobPosting) IsFresh() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().Before(*p.ExpiresAt)
}

// IsExpired returns true if the posting has expired
func (p *JobPosting) IsExpired() bool {
	return !p.IsFresh()
}

// HashJobContent generates a SHA-256 hash of the cleaned text
func HashJobContent(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// NormalizeKeyword normalizes a keyword for matching
func NormalizeKeyword(keyword string) string {
	return strings.ToLower(strings.TrimSpace(keyword))
}

// DetectPlatform attempts to detect the job board platform from a URL
func DetectPlatform(url string) string {
	urlLower := strings.ToLower(url)
	switch {
	case strings.Contains(urlLower, "greenhouse.io") || strings.Contains(urlLower, "boards.greenhouse"):
		return PlatformGreenhouse
	case strings.Contains(urlLower, "lever.co"):
		return PlatformLever
	case strings.Contains(urlLower, "linkedin.com"):
		return PlatformLinkedIn
	case strings.Contains(urlLower, "myworkday") || strings.Contains(urlLower, "workday.com"):
		return PlatformWorkday
	case strings.Contains(urlLower, "ashbyhq.com"):
		return PlatformAshby
	default:
		return PlatformUnknown
	}
}
