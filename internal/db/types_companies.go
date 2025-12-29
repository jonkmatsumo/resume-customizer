package db

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Company represents a canonical company record
type Company struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	NameNormalized string    `json:"name_normalized"`
	Domain         *string   `json:"domain,omitempty"`
	Industry       *string   `json:"industry,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CompanyDomain represents a domain associated with a company
type CompanyDomain struct {
	ID         uuid.UUID `json:"id"`
	CompanyID  uuid.UUID `json:"company_id"`
	Domain     string    `json:"domain"`
	DomainType string    `json:"domain_type"` // 'primary', 'tech_blog', 'investor_relations'
	CreatedAt  time.Time `json:"created_at"`
}

// CrawledPage represents a cached web page
type CrawledPage struct {
	ID          uuid.UUID  `json:"id"`
	CompanyID   *uuid.UUID `json:"company_id,omitempty"`
	URL         string     `json:"url"`
	PageType    *string    `json:"page_type,omitempty"`
	RawHTML     *string    `json:"-"` // Don't serialize (large)
	ParsedText  *string    `json:"parsed_text,omitempty"`
	ContentHash *string    `json:"content_hash,omitempty"`
	HTTPStatus  *int       `json:"http_status,omitempty"`
	// Error tracking
	FetchStatus        string     `json:"fetch_status"` // 'success', 'error', 'not_found', 'timeout', 'blocked'
	ErrorMessage       *string    `json:"error_message,omitempty"`
	IsPermanentFailure bool       `json:"is_permanent_failure"`
	RetryCount         int        `json:"retry_count"`
	RetryAfter         *time.Time `json:"retry_after,omitempty"`
	// Timestamps
	FetchedAt      time.Time  `json:"fetched_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	LastAccessedAt time.Time  `json:"last_accessed_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PageType constants for crawled pages
const (
	PageTypeValues      = "values"
	PageTypeCulture     = "culture"
	PageTypeAbout       = "about"
	PageTypeCareers     = "careers"
	PageTypeEngineering = "engineering"
	PageTypePress       = "press"
	PageTypeOther       = "other"
)

// FetchStatus constants for crawled pages
const (
	FetchStatusSuccess  = "success"   // Page fetched successfully
	FetchStatusError    = "error"     // Generic error (may retry)
	FetchStatusNotFound = "not_found" // 404/410 - permanent failure
	FetchStatusTimeout  = "timeout"   // Request timed out (may retry)
	FetchStatusBlocked  = "blocked"   // 403/429 - blocked by server
)

// DomainType constants for company domains
const (
	DomainTypePrimary           = "primary"
	DomainTypeTechBlog          = "tech_blog"
	DomainTypeInvestorRelations = "investor_relations"
)

// DefaultPageCacheTTL is the default time-to-live for cached pages (7 days)
const DefaultPageCacheTTL = 7 * 24 * time.Hour

// Retry backoff constants for transient failures
// Schedule: 1 min → 5 min → 25 min → 2 hours (capped)
const (
	RetryInitialBackoff = 1 * time.Minute // First retry after 1 minute
	RetryBackoffFactor  = 5               // Multiply by 5 each retry
	RetryMaxBackoff     = 2 * time.Hour   // Cap at 2 hours
	RetryMaxAttempts    = 4               // Give up after ~2 hours total
)

// IsPermanentHTTPStatus returns true for status codes that indicate permanent failure
func IsPermanentHTTPStatus(status int) bool {
	switch status {
	case 404, 410, 451: // Not Found, Gone, Unavailable for Legal Reasons
		return true
	default:
		return false
	}
}

// FetchStatusFromHTTP determines fetch status from HTTP status code
func FetchStatusFromHTTP(status int) string {
	switch {
	case status >= 200 && status < 300:
		return FetchStatusSuccess
	case status == 404 || status == 410:
		return FetchStatusNotFound
	case status == 403 || status == 429:
		return FetchStatusBlocked
	default:
		return FetchStatusError
	}
}

// NormalizeName converts a company name to a normalized form for matching
// Example: "Affirm, Inc." -> "affirminc"
func NormalizeName(name string) string {
	// Lowercase
	normalized := strings.ToLower(name)
	// Remove non-alphanumeric characters
	reg := regexp.MustCompile(`[^a-z0-9]`)
	normalized = reg.ReplaceAllString(normalized, "")
	return normalized
}

// HashContent computes SHA-256 hash of content for change detection
func HashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// IsExpired returns true if the page cache has expired
func (p *CrawledPage) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false // No expiry set, never expires
	}
	return time.Now().After(*p.ExpiresAt)
}

// IsFresh returns true if the page was fetched within maxAge
func (p *CrawledPage) IsFresh(maxAge time.Duration) bool {
	return time.Since(p.FetchedAt) < maxAge
}
