// Package fetch provides generic URL fetching with optional caching.
package fetch

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
)

// CachedFetcher wraps URL fetching with database-backed caching.
type CachedFetcher struct {
	db        *db.DB
	options   *Options
	cacheTTL  time.Duration
	skipCache bool // For testing or forcing fresh fetches
}

// CachedFetcherConfig holds configuration for the cached fetcher.
type CachedFetcherConfig struct {
	CacheTTL  time.Duration
	SkipCache bool
	Options   *Options
}

// DefaultCachedFetcherConfig returns sensible defaults.
func DefaultCachedFetcherConfig() *CachedFetcherConfig {
	return &CachedFetcherConfig{
		CacheTTL:  db.DefaultPageCacheTTL, // 7 days
		SkipCache: false,
		Options:   DefaultOptions(),
	}
}

// NewCachedFetcher creates a new cached fetcher.
func NewCachedFetcher(database *db.DB, config *CachedFetcherConfig) *CachedFetcher {
	if config == nil {
		config = DefaultCachedFetcherConfig()
	}
	if config.Options == nil {
		config.Options = DefaultOptions()
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = db.DefaultPageCacheTTL
	}
	return &CachedFetcher{
		db:        database,
		options:   config.Options,
		cacheTTL:  config.CacheTTL,
		skipCache: config.SkipCache,
	}
}

// CachedResult extends Result with cache metadata.
type CachedResult struct {
	*Result
	FromCache bool      // Whether this result came from cache
	PageID    uuid.UUID // Database ID of the cached page
}

// Fetch retrieves a URL, using cache if available and fresh.
// Returns cached content if within TTL, otherwise fetches fresh content and caches it.
func (f *CachedFetcher) Fetch(ctx context.Context, urlStr string) (*CachedResult, error) {
	return f.FetchWithCompany(ctx, urlStr, nil, nil)
}

// FetchWithCompany retrieves a URL with optional company association.
// This allows the cached page to be linked to a company for later retrieval.
func (f *CachedFetcher) FetchWithCompany(ctx context.Context, urlStr string, companyID *uuid.UUID, pageType *string) (*CachedResult, error) {
	// Step 1: Check if URL should be skipped (permanent failure or backoff)
	if !f.skipCache && f.db != nil {
		shouldSkip, reason, err := f.db.ShouldSkipURL(ctx, urlStr)
		if err != nil {
			return nil, fmt.Errorf("failed to check skip status: %w", err)
		}
		if shouldSkip {
			return nil, &Error{
				URL:       urlStr,
				Message:   fmt.Sprintf("URL skipped: %s", reason),
				Retryable: false,
			}
		}
	}

	// Step 2: Try to get fresh cached page
	if !f.skipCache && f.db != nil {
		cached, err := f.db.GetFreshCrawledPage(ctx, urlStr, f.cacheTTL)
		if err != nil {
			return nil, fmt.Errorf("failed to check cache: %w", err)
		}
		if cached != nil {
			// Return cached content
			return &CachedResult{
				Result: &Result{
					URL:        cached.URL,
					HTML:       derefString(cached.RawHTML),
					Text:       derefString(cached.ParsedText),
					StatusCode: derefInt(cached.HTTPStatus),
				},
				FromCache: true,
				PageID:    cached.ID,
			}, nil
		}
	}

	// Step 3: Fetch fresh content
	result, err := URL(ctx, urlStr, f.options)
	if err != nil {
		// Record failure in database
		if f.db != nil {
			statusCode := 0
			errMsg := err.Error()
			if result != nil {
				statusCode = result.StatusCode
			}
			_ = f.db.RecordFailedFetch(ctx, urlStr, statusCode, errMsg)
		}
		return nil, err
	}

	// Step 4: Extract text from HTML
	text, _ := ExtractMainText(result.HTML, DefaultTextSelectors())
	result.Text = text

	// Step 5: Store in cache
	if f.db != nil {
		page := &db.CrawledPage{
			CompanyID:   companyID,
			URL:         urlStr,
			PageType:    pageType,
			RawHTML:     &result.HTML,
			ParsedText:  &result.Text,
			HTTPStatus:  &result.StatusCode,
			FetchStatus: db.FetchStatusSuccess,
		}
		if err := f.db.UpsertCrawledPage(ctx, page); err != nil {
			// Log but don't fail - the fetch succeeded
			// In production, this should log the error
			_ = err
		} else {
			return &CachedResult{
				Result:    result,
				FromCache: false,
				PageID:    page.ID,
			}, nil
		}
	}

	return &CachedResult{
		Result:    result,
		FromCache: false,
	}, nil
}

// FetchMultiple fetches multiple URLs concurrently with caching.
// Returns results in the same order as input URLs. Failed fetches are nil in the result slice.
func (f *CachedFetcher) FetchMultiple(ctx context.Context, urls []string) ([]*CachedResult, []error) {
	results := make([]*CachedResult, len(urls))
	errors := make([]error, len(urls))

	// Use a simple sequential approach for now
	// Can be parallelized with goroutines if needed
	for i, url := range urls {
		result, err := f.Fetch(ctx, url)
		if err != nil {
			errors[i] = err
		} else {
			results[i] = result
		}
	}

	return results, errors
}

// InvalidateCache marks a cached page as stale, forcing a re-fetch on next request.
func (f *CachedFetcher) InvalidateCache(ctx context.Context, urlStr string) error {
	if f.db == nil {
		return nil
	}

	page, err := f.db.GetCrawledPageByURL(ctx, urlStr)
	if err != nil || page == nil {
		return err
	}

	// Set expires_at to past to force re-fetch
	past := time.Now().Add(-time.Hour)
	page.ExpiresAt = &past
	return f.db.UpsertCrawledPage(ctx, page)
}

// Helper functions

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

