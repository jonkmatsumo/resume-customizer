package fetch

import (
	"testing"
)

func TestDerefString(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{"nil pointer", nil, ""},
		{"empty string", strPtr(""), ""},
		{"non-empty string", strPtr("hello"), "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := derefString(tt.input)
			if result != tt.expected {
				t.Errorf("derefString(%v) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDerefInt(t *testing.T) {
	tests := []struct {
		name     string
		input    *int
		expected int
	}{
		{"nil pointer", nil, 0},
		{"zero value", intPtr(0), 0},
		{"positive value", intPtr(200), 200},
		{"negative value", intPtr(-1), -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := derefInt(tt.input)
			if result != tt.expected {
				t.Errorf("derefInt(%v) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDefaultCachedFetcherConfig(t *testing.T) {
	config := DefaultCachedFetcherConfig()

	if config == nil {
		t.Fatal("DefaultCachedFetcherConfig returned nil")
	}

	if config.CacheTTL == 0 {
		t.Error("Expected non-zero CacheTTL")
	}

	if config.SkipCache != false {
		t.Error("Expected SkipCache to be false by default")
	}

	if config.Options == nil {
		t.Error("Expected Options to be non-nil")
	}
}

func TestNewCachedFetcher_NilConfig(t *testing.T) {
	fetcher := NewCachedFetcher(nil, nil)

	if fetcher == nil {
		t.Fatal("NewCachedFetcher returned nil")
	}

	if fetcher.cacheTTL == 0 {
		t.Error("Expected non-zero cacheTTL")
	}

	if fetcher.options == nil {
		t.Error("Expected non-nil options")
	}
}

func TestNewCachedFetcher_EmptyConfig(t *testing.T) {
	config := &CachedFetcherConfig{}
	fetcher := NewCachedFetcher(nil, config)

	if fetcher == nil {
		t.Fatal("NewCachedFetcher returned nil")
	}

	// Should use defaults for zero values
	if fetcher.cacheTTL == 0 {
		t.Error("Expected non-zero cacheTTL even with empty config")
	}

	if fetcher.options == nil {
		t.Error("Expected non-nil options even with empty config")
	}
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

