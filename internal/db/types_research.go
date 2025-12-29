package db

import (
	"time"

	"github.com/google/uuid"
)

// ResearchSessionStatus constants
const (
	ResearchStatusPending    = "pending"
	ResearchStatusInProgress = "in_progress"
	ResearchStatusCompleted  = "completed"
	ResearchStatusFailed     = "failed"
)

// FrontierURLStatus constants
const (
	FrontierStatusPending = "pending"
	FrontierStatusFetched = "fetched"
	FrontierStatusSkipped = "skipped"
	FrontierStatusFailed  = "failed"
)

// DefaultPagesLimit is the default max pages to crawl per session
const DefaultPagesLimit = 5

// Note: PageType constants are defined in types_companies.go
// (PageTypeValues, PageTypeCulture, PageTypeEngineering, PageTypeAbout,
// PageTypeCareers, PageTypePress, PageTypeOther)

// ResearchSession represents a research session for a company
type ResearchSession struct {
	ID           uuid.UUID  `json:"id"`
	CompanyID    *uuid.UUID `json:"company_id,omitempty"`
	RunID        *uuid.UUID `json:"run_id,omitempty"`
	CompanyName  string     `json:"company_name"`
	Domain       *string    `json:"domain,omitempty"`
	Status       string     `json:"status"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	PagesCrawled int        `json:"pages_crawled"`
	PagesLimit   int        `json:"pages_limit"`
	CorpusText   *string    `json:"corpus_text,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`

	// Denormalized data loaded via joins
	FrontierURLs []FrontierURL         `json:"frontier_urls,omitempty"`
	BrandSignals []ResearchBrandSignal `json:"brand_signals,omitempty"`
}

// FrontierURL represents a URL in the research frontier queue
type FrontierURL struct {
	ID            uuid.UUID  `json:"id"`
	SessionID     uuid.UUID  `json:"session_id"`
	URL           string     `json:"url"`
	Priority      float64    `json:"priority"`
	PageType      *string    `json:"page_type,omitempty"`
	Reason        *string    `json:"reason,omitempty"`
	Status        string     `json:"status"`
	SkipReason    *string    `json:"skip_reason,omitempty"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	CrawledPageID *uuid.UUID `json:"crawled_page_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	FetchedAt     *time.Time `json:"fetched_at,omitempty"`
}

// ResearchBrandSignal represents extracted brand information from a page
type ResearchBrandSignal struct {
	ID          uuid.UUID `json:"id"`
	SessionID   uuid.UUID `json:"session_id"`
	URL         string    `json:"url"`
	SignalType  *string   `json:"signal_type,omitempty"`
	KeyPoints   []string  `json:"key_points,omitempty"`
	ValuesFound []string  `json:"values_found,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ResearchSessionInput is used when creating a new research session
type ResearchSessionInput struct {
	CompanyID   *uuid.UUID
	RunID       *uuid.UUID
	CompanyName string
	Domain      string
	PagesLimit  int
}

// FrontierURLInput is used when adding URLs to the frontier
type FrontierURLInput struct {
	URL      string
	Priority float64
	PageType string
	Reason   string
}

// ResearchBrandSignalInput is used when saving brand signals
type ResearchBrandSignalInput struct {
	URL         string
	SignalType  string
	KeyPoints   []string
	ValuesFound []string
}

// ValidResearchStatus checks if a research status value is valid
func ValidResearchStatus(status string) bool {
	switch status {
	case ResearchStatusPending, ResearchStatusInProgress, ResearchStatusCompleted, ResearchStatusFailed:
		return true
	default:
		return false
	}
}

// ValidFrontierStatus checks if a frontier URL status value is valid
func ValidFrontierStatus(status string) bool {
	switch status {
	case FrontierStatusPending, FrontierStatusFetched, FrontierStatusSkipped, FrontierStatusFailed:
		return true
	default:
		return false
	}
}

// ValidPageType checks if a page type value is valid
func ValidPageType(pageType string) bool {
	switch pageType {
	case PageTypeValues, PageTypeCulture, PageTypeEngineering, PageTypeAbout,
		PageTypeCareers, PageTypePress, PageTypeOther, "":
		return true
	default:
		return false
	}
}
