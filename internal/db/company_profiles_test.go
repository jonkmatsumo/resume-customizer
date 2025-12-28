package db

import (
	"testing"
	"time"
)

// =============================================================================
// CompanyProfile Method Tests
// =============================================================================

func TestCompanyProfile_IsStale(t *testing.T) {
	now := time.Now()
	past := now.Add(-48 * time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name           string
		lastVerifiedAt *time.Time
		maxAge         time.Duration
		expected       bool
	}{
		{"nil last verified", nil, 24 * time.Hour, true},
		{"verified recently", &now, 24 * time.Hour, false},
		{"verified 2 days ago, 1 day max", &past, 24 * time.Hour, true},
		{"verified 2 days ago, 7 day max", &past, 7 * 24 * time.Hour, false},
		{"verified in future", &future, 24 * time.Hour, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &CompanyProfile{LastVerifiedAt: tt.lastVerifiedAt}
			result := p.IsStale(tt.maxAge)
			if result != tt.expected {
				t.Errorf("IsStale() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCompanyProfile_NeedsUpdate(t *testing.T) {
	tests := []struct {
		name           string
		version        int
		currentVersion int
		expected       bool
	}{
		{"same version", 1, 1, false},
		{"older version", 1, 2, true},
		{"newer version", 2, 1, false},
		{"zero version", 0, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &CompanyProfile{Version: tt.version}
			result := p.NeedsUpdate(tt.currentVersion)
			if result != tt.expected {
				t.Errorf("NeedsUpdate(%d) = %v, want %v", tt.currentVersion, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Signal Type Constant Tests
// =============================================================================

func TestSignalTypeConstants(t *testing.T) {
	// Verify constants are defined correctly
	signalTypes := []string{
		SignalTypeCulture,
		SignalTypeValues,
		SignalTypeEngineering,
		SignalTypeMission,
		SignalTypeProduct,
		SignalTypeTeam,
	}

	for _, st := range signalTypes {
		if st == "" {
			t.Errorf("Signal type constant should not be empty")
		}
	}
}

func TestSourceTypeConstants(t *testing.T) {
	sourceTypes := []string{
		SourceTypeValues,
		SourceTypeCulture,
		SourceTypeAbout,
		SourceTypeCareers,
		SourceTypeEngineering,
		SourceTypeBlog,
	}

	for _, st := range sourceTypes {
		if st == "" {
			t.Errorf("Source type constant should not be empty")
		}
	}
}

// =============================================================================
// Input Validation Tests
// =============================================================================

func TestProfileCreateInput_Validation(t *testing.T) {
	t.Run("empty tone is allowed", func(t *testing.T) {
		input := ProfileCreateInput{
			Tone: "",
		}
		// We don't validate at struct level, but DB will enforce NOT NULL
		if input.Tone != "" {
			t.Error("Expected empty tone")
		}
	})

	t.Run("nil slices are safe", func(t *testing.T) {
		input := ProfileCreateInput{
			StyleRules:   nil,
			TabooPhrases: nil,
			Values:       nil,
			EvidenceURLs: nil,
		}
		if len(input.StyleRules) != 0 {
			t.Error("Expected empty slice")
		}
	})
}

func TestTabooPhraseInput(t *testing.T) {
	t.Run("with reason", func(t *testing.T) {
		input := TabooPhraseInput{
			Phrase: "synergy",
			Reason: "overused corporate jargon",
		}
		if input.Phrase != "synergy" {
			t.Errorf("Phrase = %q, want 'synergy'", input.Phrase)
		}
		if input.Reason != "overused corporate jargon" {
			t.Errorf("Reason not set correctly")
		}
	})

	t.Run("without reason", func(t *testing.T) {
		input := TabooPhraseInput{
			Phrase: "rockstar",
		}
		if input.Reason != "" {
			t.Error("Reason should be empty")
		}
	})
}

// =============================================================================
// Default Constant Tests
// =============================================================================

func TestDefaultProfileCacheTTL(t *testing.T) {
	expected := 30 * 24 * time.Hour
	if DefaultProfileCacheTTL != expected {
		t.Errorf("DefaultProfileCacheTTL = %v, want %v", DefaultProfileCacheTTL, expected)
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestNullIfEmpty(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantVal string
	}{
		{"empty string", "", true, ""},
		{"non-empty string", "test", false, "test"},
		{"whitespace only", "   ", false, "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullIfEmpty(tt.input)
			if tt.wantNil {
				if result != nil {
					t.Errorf("nullIfEmpty(%q) = %v, want nil", tt.input, *result)
				}
			} else {
				if result == nil {
					t.Errorf("nullIfEmpty(%q) = nil, want %q", tt.input, tt.wantVal)
				} else if *result != tt.wantVal {
					t.Errorf("nullIfEmpty(%q) = %q, want %q", tt.input, *result, tt.wantVal)
				}
			}
		})
	}
}
