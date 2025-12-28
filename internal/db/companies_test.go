package db

import (
	"testing"
	"time"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Affirm", "affirm"},
		{"Affirm, Inc.", "affirminc"},
		{"Google LLC", "googlellc"},
		{"NVIDIA Corporation", "nvidiacorporation"},
		{"Meta Platforms, Inc.", "metaplatformsinc"},
		{"open AI", "openai"},
		{"100 Thieves", "100thieves"},
		{"  Spaces Around  ", "spacesaround"},
		{"Stripe Inc", "stripeinc"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHashContent(t *testing.T) {
	// Same input should produce same hash
	hash1 := HashContent("hello world")
	hash2 := HashContent("hello world")
	if hash1 != hash2 {
		t.Errorf("Same content produced different hashes: %s vs %s", hash1, hash2)
	}

	// Different input should produce different hash
	hash3 := HashContent("different content")
	if hash1 == hash3 {
		t.Errorf("Different content produced same hash: %s", hash1)
	}

	// Hash should be 64 characters (SHA-256 hex)
	if len(hash1) != 64 {
		t.Errorf("Hash length is %d, expected 64", len(hash1))
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"https://www.affirm.com/careers", "affirm.com", false},
		{"http://engineering.affirm.com/blog", "engineering.affirm.com", false},
		{"https://stripe.com", "stripe.com", false},
		{"https://www.google.com/search?q=test", "google.com", false},
		{"ftp://files.example.com", "files.example.com", false},
		// Invalid URLs
		{"://invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ExtractDomain(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("ExtractDomain(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ExtractDomain(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("ExtractDomain(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsPermanentHTTPStatus(t *testing.T) {
	tests := []struct {
		status   int
		expected bool
	}{
		{200, false},
		{301, false},
		{302, false},
		{400, false},
		{403, false},
		{404, true},  // Not Found - permanent
		{410, true},  // Gone - permanent
		{429, false}, // Too Many Requests - retry
		{451, true},  // Unavailable for Legal Reasons - permanent
		{500, false}, // Server error - retry
		{503, false}, // Service Unavailable - retry
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.status)), func(t *testing.T) {
			result := IsPermanentHTTPStatus(tt.status)
			if result != tt.expected {
				t.Errorf("IsPermanentHTTPStatus(%d) = %v, expected %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestFetchStatusFromHTTP(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{200, FetchStatusSuccess},
		{201, FetchStatusSuccess},
		{204, FetchStatusSuccess},
		{301, FetchStatusError},
		{302, FetchStatusError},
		{400, FetchStatusError},
		{403, FetchStatusBlocked},
		{404, FetchStatusNotFound},
		{410, FetchStatusNotFound},
		{429, FetchStatusBlocked},
		{500, FetchStatusError},
		{502, FetchStatusError},
		{503, FetchStatusError},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.status)), func(t *testing.T) {
			result := FetchStatusFromHTTP(tt.status)
			if result != tt.expected {
				t.Errorf("FetchStatusFromHTTP(%d) = %q, expected %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestCrawledPage_IsFresh(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		fetchedAt time.Time
		maxAge    time.Duration
		expected  bool
	}{
		{"Fresh page", now.Add(-1 * time.Hour), 7 * 24 * time.Hour, true},
		{"Just expired", now.Add(-8 * 24 * time.Hour), 7 * 24 * time.Hour, false},
		{"Very old", now.Add(-30 * 24 * time.Hour), 7 * 24 * time.Hour, false},
		{"Just fetched", now, 7 * 24 * time.Hour, true},
		{"Short TTL fresh", now.Add(-30 * time.Minute), 1 * time.Hour, true},
		{"Short TTL expired", now.Add(-2 * time.Hour), 1 * time.Hour, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &CrawledPage{FetchedAt: tt.fetchedAt}
			result := page.IsFresh(tt.maxAge)
			if result != tt.expected {
				t.Errorf("IsFresh(%v) = %v, expected %v", tt.maxAge, result, tt.expected)
			}
		})
	}
}

func TestCrawledPage_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt *time.Time
		expected  bool
	}{
		{"No expiry set", nil, false},
		{"Expires in future", timePtr(now.Add(1 * time.Hour)), false},
		{"Expired", timePtr(now.Add(-1 * time.Hour)), true},
		{"Expires exactly now", timePtr(now), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &CrawledPage{ExpiresAt: tt.expiresAt}
			result := page.IsExpired()
			if result != tt.expected {
				t.Errorf("IsExpired() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://www.affirm.com/", "affirm.com"},
		{"http://www.google.com", "google.com"},
		{"www.stripe.com/", "stripe.com"},
		{"EXAMPLE.COM", "example.com"},
		{"engineering.meta.com", "engineering.meta.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeDomain(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeDomain(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function for creating time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}
