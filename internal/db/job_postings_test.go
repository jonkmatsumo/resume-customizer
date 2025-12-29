package db

import (
	"testing"
	"time"
)

// =============================================================================
// JobPosting Method Tests
// =============================================================================

func TestJobPosting_IsFresh(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		expected  bool
	}{
		{"nil expires_at", nil, false},
		{"expired", &past, false},
		{"not expired", &future, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &JobPosting{ExpiresAt: tt.expiresAt}
			result := p.IsFresh()
			if result != tt.expected {
				t.Errorf("IsFresh() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestJobPosting_IsExpired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		expected  bool
	}{
		{"nil expires_at", nil, true},
		{"expired", &past, true},
		{"not expired", &future, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &JobPosting{ExpiresAt: tt.expiresAt}
			result := p.IsExpired()
			if result != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestHashJobContent(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string // SHA-256 of the text
	}{
		{"empty", "", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"hello", "hello", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashJobContent(tt.text)
			if result != tt.expected {
				t.Errorf("HashJobContent(%q) = %q, want %q", tt.text, result, tt.expected)
			}
		})
	}

	// Same input should give same hash
	hash1 := HashJobContent("test content")
	hash2 := HashJobContent("test content")
	if hash1 != hash2 {
		t.Error("Same content should produce same hash")
	}
}

func TestNormalizeKeyword(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Python", "python"},
		{"  golang  ", "golang"},
		{"Machine Learning", "machine learning"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeKeyword(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeKeyword(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectPlatform(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://boards.greenhouse.io/company/jobs/123", PlatformGreenhouse},
		{"https://company.greenhouse.io/jobs/123", PlatformGreenhouse},
		{"https://jobs.lever.co/company/123", PlatformLever},
		{"https://www.linkedin.com/jobs/view/123", PlatformLinkedIn},
		{"https://company.myworkdayjobs.com/en-US/job/123", PlatformWorkday},
		{"https://jobs.ashbyhq.com/company/123", PlatformAshby},
		{"https://company.com/careers/job-123", PlatformUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := DetectPlatform(tt.url)
			if result != tt.expected {
				t.Errorf("DetectPlatform(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Constant Tests
// =============================================================================

func TestPlatformConstants(t *testing.T) {
	platforms := []string{
		PlatformGreenhouse,
		PlatformLever,
		PlatformLinkedIn,
		PlatformWorkday,
		PlatformAshby,
		PlatformUnknown,
	}

	for _, p := range platforms {
		if p == "" {
			t.Errorf("Platform constant should not be empty")
		}
	}
}

func TestRequirementTypeConstants(t *testing.T) {
	if RequirementTypeHard != "hard" {
		t.Errorf("RequirementTypeHard = %q, want 'hard'", RequirementTypeHard)
	}
	if RequirementTypeNiceToHave != "nice_to_have" {
		t.Errorf("RequirementTypeNiceToHave = %q, want 'nice_to_have'", RequirementTypeNiceToHave)
	}
}

func TestEducationDegreeConstants(t *testing.T) {
	degrees := []string{DegreeNone, DegreeAssociate, DegreeBachelor, DegreeMaster, DegreePhD}
	for _, d := range degrees {
		if d == "" {
			t.Errorf("Degree constant should not be empty")
		}
	}
}

func TestDefaultJobPostingCacheTTL(t *testing.T) {
	expected := 24 * time.Hour
	if DefaultJobPostingCacheTTL != expected {
		t.Errorf("DefaultJobPostingCacheTTL = %v, want %v", DefaultJobPostingCacheTTL, expected)
	}
}

// =============================================================================
// Input Type Tests
// =============================================================================

func TestJobPostingCreateInput(t *testing.T) {
	input := JobPostingCreateInput{
		URL:         "https://example.com/job",
		RoleTitle:   "Software Engineer",
		Platform:    PlatformGreenhouse,
		CleanedText: "Job description",
		HTTPStatus:  200,
	}

	if input.URL != "https://example.com/job" {
		t.Error("URL not set correctly")
	}
	if input.CompanyID != nil {
		t.Error("CompanyID should be nil by default")
	}
}

func TestJobProfileCreateInput(t *testing.T) {
	input := JobProfileCreateInput{
		CompanyName: "Test Corp",
		RoleTitle:   "Engineer",
		EvalLatency: true,
		Responsibilities: []string{
			"Build systems",
			"Write tests",
		},
		HardRequirements: []RequirementInput{
			{Skill: "Go", Level: "5+ years"},
		},
	}

	if len(input.Responsibilities) != 2 {
		t.Errorf("Responsibilities count = %d, want 2", len(input.Responsibilities))
	}
	if len(input.HardRequirements) != 1 {
		t.Errorf("HardRequirements count = %d, want 1", len(input.HardRequirements))
	}
}

func TestRequirementInput(t *testing.T) {
	input := RequirementInput{
		Skill:    "Python",
		Level:    "expert",
		Evidence: "Strong Python skills required",
	}

	if input.Skill != "Python" {
		t.Errorf("Skill = %q, want 'Python'", input.Skill)
	}
	if input.Level != "expert" {
		t.Errorf("Level = %q, want 'expert'", input.Level)
	}
}

// =============================================================================
// AdminInfo Tests
// =============================================================================

func TestAdminInfo(t *testing.T) {
	salary := "$150,000 - $200,000"
	location := "San Francisco, CA"
	remote := "hybrid"

	info := AdminInfo{
		Salary:       &salary,
		Location:     &location,
		RemotePolicy: &remote,
	}

	if info.Salary == nil || *info.Salary != salary {
		t.Error("Salary not set correctly")
	}
	if info.Location == nil || *info.Location != location {
		t.Error("Location not set correctly")
	}
	if info.RemotePolicy == nil || *info.RemotePolicy != remote {
		t.Error("RemotePolicy not set correctly")
	}
	if info.SalaryMin != nil {
		t.Error("SalaryMin should be nil")
	}
}
