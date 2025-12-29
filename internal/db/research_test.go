package db

import (
	"testing"

	"github.com/google/uuid"
)

// =============================================================================
// Research Session Status Constants Tests
// =============================================================================

func TestResearchStatusConstants(t *testing.T) {
	statuses := []string{
		ResearchStatusPending,
		ResearchStatusInProgress,
		ResearchStatusCompleted,
		ResearchStatusFailed,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("Research status constant should not be empty")
		}
	}

	// Verify expected values
	if ResearchStatusPending != "pending" {
		t.Errorf("ResearchStatusPending = %q, want 'pending'", ResearchStatusPending)
	}
	if ResearchStatusInProgress != "in_progress" {
		t.Errorf("ResearchStatusInProgress = %q, want 'in_progress'", ResearchStatusInProgress)
	}
	if ResearchStatusCompleted != "completed" {
		t.Errorf("ResearchStatusCompleted = %q, want 'completed'", ResearchStatusCompleted)
	}
	if ResearchStatusFailed != "failed" {
		t.Errorf("ResearchStatusFailed = %q, want 'failed'", ResearchStatusFailed)
	}
}

func TestFrontierStatusConstants(t *testing.T) {
	statuses := []string{
		FrontierStatusPending,
		FrontierStatusFetched,
		FrontierStatusSkipped,
		FrontierStatusFailed,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("Frontier status constant should not be empty")
		}
	}

	if FrontierStatusPending != "pending" {
		t.Errorf("FrontierStatusPending = %q, want 'pending'", FrontierStatusPending)
	}
	if FrontierStatusFetched != "fetched" {
		t.Errorf("FrontierStatusFetched = %q, want 'fetched'", FrontierStatusFetched)
	}
}

func TestPageTypeConstants(t *testing.T) {
	pageTypes := []string{
		PageTypeValues,
		PageTypeCulture,
		PageTypeEngineering,
		PageTypeAbout,
		PageTypeCareers,
		PageTypePress,
		PageTypeOther,
	}

	for _, pt := range pageTypes {
		if pt == "" {
			t.Error("Page type constant should not be empty")
		}
	}

	if PageTypeValues != "values" {
		t.Errorf("PageTypeValues = %q, want 'values'", PageTypeValues)
	}
	if PageTypeCulture != "culture" {
		t.Errorf("PageTypeCulture = %q, want 'culture'", PageTypeCulture)
	}
}

func TestDefaultPagesLimit(t *testing.T) {
	if DefaultPagesLimit != 5 {
		t.Errorf("DefaultPagesLimit = %d, want 5", DefaultPagesLimit)
	}
}

// =============================================================================
// Validation Function Tests
// =============================================================================

func TestValidResearchStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"pending", true},
		{"in_progress", true},
		{"completed", true},
		{"failed", true},
		{"unknown", false},
		{"", false},
		{"PENDING", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ValidResearchStatus(tt.input)
			if result != tt.expected {
				t.Errorf("ValidResearchStatus(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidFrontierStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"pending", true},
		{"fetched", true},
		{"skipped", true},
		{"failed", true},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ValidFrontierStatus(tt.input)
			if result != tt.expected {
				t.Errorf("ValidFrontierStatus(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidPageType(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"values", true},
		{"culture", true},
		{"engineering", true},
		{"about", true},
		{"careers", true},
		{"press", true},
		{"other", true},
		{"", true}, // empty is valid (optional)
		{"blog", false},
		{"homepage", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ValidPageType(tt.input)
			if result != tt.expected {
				t.Errorf("ValidPageType(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Type Tests
// =============================================================================

func TestResearchSessionType(t *testing.T) {
	companyID := uuid.New()
	domain := "example.com"
	session := ResearchSession{
		ID:           uuid.New(),
		CompanyID:    &companyID,
		CompanyName:  "Test Company",
		Domain:       &domain,
		Status:       ResearchStatusPending,
		PagesCrawled: 0,
		PagesLimit:   5,
	}

	if session.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if session.CompanyName != "Test Company" {
		t.Errorf("CompanyName = %q", session.CompanyName)
	}
	if session.Status != ResearchStatusPending {
		t.Errorf("Status = %q", session.Status)
	}
	if session.PagesLimit != 5 {
		t.Errorf("PagesLimit = %d", session.PagesLimit)
	}
}

func TestFrontierURLType(t *testing.T) {
	pageType := PageTypeCulture
	reason := "Contains culture info"
	fu := FrontierURL{
		ID:        uuid.New(),
		SessionID: uuid.New(),
		URL:       "https://example.com/culture",
		Priority:  0.9,
		PageType:  &pageType,
		Reason:    &reason,
		Status:    FrontierStatusPending,
	}

	if fu.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if fu.URL != "https://example.com/culture" {
		t.Errorf("URL = %q", fu.URL)
	}
	if fu.Priority != 0.9 {
		t.Errorf("Priority = %f", fu.Priority)
	}
	if *fu.PageType != PageTypeCulture {
		t.Errorf("PageType = %q", *fu.PageType)
	}
}

func TestResearchBrandSignalType(t *testing.T) {
	signalType := PageTypeValues
	rbs := ResearchBrandSignal{
		ID:          uuid.New(),
		SessionID:   uuid.New(),
		URL:         "https://example.com/values",
		SignalType:  &signalType,
		KeyPoints:   []string{"Innovation", "Customer focus"},
		ValuesFound: []string{"integrity", "excellence"},
	}

	if rbs.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if rbs.URL != "https://example.com/values" {
		t.Errorf("URL = %q", rbs.URL)
	}
	if len(rbs.KeyPoints) != 2 {
		t.Errorf("KeyPoints count = %d", len(rbs.KeyPoints))
	}
	if len(rbs.ValuesFound) != 2 {
		t.Errorf("ValuesFound count = %d", len(rbs.ValuesFound))
	}
}

// =============================================================================
// Input Type Tests
// =============================================================================

func TestResearchSessionInput(t *testing.T) {
	companyID := uuid.New()
	input := ResearchSessionInput{
		CompanyID:   &companyID,
		CompanyName: "Test Corp",
		Domain:      "testcorp.com",
		PagesLimit:  10,
	}

	if input.CompanyName != "Test Corp" {
		t.Errorf("CompanyName = %q", input.CompanyName)
	}
	if input.Domain != "testcorp.com" {
		t.Errorf("Domain = %q", input.Domain)
	}
	if input.PagesLimit != 10 {
		t.Errorf("PagesLimit = %d", input.PagesLimit)
	}
}

func TestFrontierURLInput(t *testing.T) {
	input := FrontierURLInput{
		URL:      "https://example.com/engineering",
		Priority: 0.8,
		PageType: PageTypeEngineering,
		Reason:   "Engineering blog",
	}

	if input.URL != "https://example.com/engineering" {
		t.Errorf("URL = %q", input.URL)
	}
	if input.Priority != 0.8 {
		t.Errorf("Priority = %f", input.Priority)
	}
	if input.PageType != PageTypeEngineering {
		t.Errorf("PageType = %q", input.PageType)
	}
}

func TestResearchBrandSignalInput(t *testing.T) {
	input := ResearchBrandSignalInput{
		URL:         "https://example.com/values",
		SignalType:  PageTypeValues,
		KeyPoints:   []string{"Customer first", "Innovation"},
		ValuesFound: []string{"customer_focus", "innovation"},
	}

	if input.URL != "https://example.com/values" {
		t.Errorf("URL = %q", input.URL)
	}
	if input.SignalType != PageTypeValues {
		t.Errorf("SignalType = %q", input.SignalType)
	}
	if len(input.KeyPoints) != 2 {
		t.Errorf("KeyPoints count = %d", len(input.KeyPoints))
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestResearchSessionNilOptionalFields(t *testing.T) {
	session := ResearchSession{
		ID:          uuid.New(),
		CompanyName: "Test Company",
		Status:      ResearchStatusPending,
		PagesLimit:  DefaultPagesLimit,
		// All optional fields nil
	}

	if session.CompanyID != nil {
		t.Error("CompanyID should be nil")
	}
	if session.RunID != nil {
		t.Error("RunID should be nil")
	}
	if session.Domain != nil {
		t.Error("Domain should be nil")
	}
	if session.ErrorMessage != nil {
		t.Error("ErrorMessage should be nil")
	}
	if session.CorpusText != nil {
		t.Error("CorpusText should be nil")
	}
}

func TestFrontierURLNilOptionalFields(t *testing.T) {
	fu := FrontierURL{
		ID:        uuid.New(),
		SessionID: uuid.New(),
		URL:       "https://example.com",
		Priority:  0.5,
		Status:    FrontierStatusPending,
		// All optional fields nil
	}

	if fu.PageType != nil {
		t.Error("PageType should be nil")
	}
	if fu.Reason != nil {
		t.Error("Reason should be nil")
	}
	if fu.SkipReason != nil {
		t.Error("SkipReason should be nil")
	}
	if fu.CrawledPageID != nil {
		t.Error("CrawledPageID should be nil")
	}
}

func TestResearchSessionWithEmptyFrontierAndSignals(t *testing.T) {
	session := ResearchSession{
		ID:           uuid.New(),
		CompanyName:  "Test",
		Status:       ResearchStatusPending,
		PagesLimit:   5,
		FrontierURLs: []FrontierURL{},
		BrandSignals: []ResearchBrandSignal{},
	}

	if session.FrontierURLs == nil {
		t.Error("FrontierURLs should be empty slice, not nil")
	}
	if len(session.FrontierURLs) != 0 {
		t.Errorf("FrontierURLs should be empty, got %d", len(session.FrontierURLs))
	}
	if session.BrandSignals == nil {
		t.Error("BrandSignals should be empty slice, not nil")
	}
}
