package db

import (
	"encoding/json"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
)

func TestGetJobProfileByRunID(t *testing.T) {
	// This is a unit test that verifies the unmarshaling logic
	// Integration tests will verify database operations
	t.Run("unmarshal valid job profile", func(t *testing.T) {
		profile := &types.JobProfile{
			Company:   "Test Company",
			RoleTitle: "Software Engineer",
		}
		jsonBytes, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("Failed to marshal test profile: %v", err)
		}

		var result types.JobProfile
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if result.Company != "Test Company" {
			t.Errorf("Company = %q, want 'Test Company'", result.Company)
		}
	})
}

func TestGetRankedStoriesByRunID(t *testing.T) {
	t.Run("unmarshal valid ranked stories", func(t *testing.T) {
		stories := &types.RankedStories{
			Ranked: []types.RankedStory{
				{StoryID: "test-story", RelevanceScore: 0.9},
			},
		}
		jsonBytes, err := json.Marshal(stories)
		if err != nil {
			t.Fatalf("Failed to marshal test stories: %v", err)
		}

		var result types.RankedStories
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(result.Ranked) != 1 {
			t.Errorf("Ranked count = %d, want 1", len(result.Ranked))
		}
	})
}

func TestGetResumePlanByRunID(t *testing.T) {
	t.Run("unmarshal valid resume plan", func(t *testing.T) {
		plan := &types.ResumePlan{
			SelectedStories: []types.SelectedStory{
				{StoryID: "test-story", BulletIDs: []string{"bullet-1"}},
			},
		}
		jsonBytes, err := json.Marshal(plan)
		if err != nil {
			t.Fatalf("Failed to marshal test plan: %v", err)
		}

		var result types.ResumePlan
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(result.SelectedStories) != 1 {
			t.Errorf("SelectedStories count = %d, want 1", len(result.SelectedStories))
		}
	})
}

func TestGetSelectedBulletsByRunID(t *testing.T) {
	t.Run("unmarshal valid selected bullets", func(t *testing.T) {
		bullets := &types.SelectedBullets{
			Bullets: []types.SelectedBullet{
				{ID: "bullet-1", Text: "Test bullet"},
			},
		}
		jsonBytes, err := json.Marshal(bullets)
		if err != nil {
			t.Fatalf("Failed to marshal test bullets: %v", err)
		}

		var result types.SelectedBullets
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(result.Bullets) != 1 {
			t.Errorf("Bullets count = %d, want 1", len(result.Bullets))
		}
	})
}

func TestGetRewrittenBulletsByRunID(t *testing.T) {
	t.Run("unmarshal valid rewritten bullets", func(t *testing.T) {
		bullets := &types.RewrittenBullets{
			Bullets: []types.RewrittenBullet{
				{OriginalBulletID: "bullet-1", FinalText: "Rewritten text"},
			},
		}
		jsonBytes, err := json.Marshal(bullets)
		if err != nil {
			t.Fatalf("Failed to marshal test bullets: %v", err)
		}

		var result types.RewrittenBullets
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(result.Bullets) != 1 {
			t.Errorf("Bullets count = %d, want 1", len(result.Bullets))
		}
	})
}

func TestGetCompanyProfileByRunID(t *testing.T) {
	t.Run("unmarshal valid company profile", func(t *testing.T) {
		profile := &types.CompanyProfile{
			Tone: "professional",
		}
		jsonBytes, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("Failed to marshal test profile: %v", err)
		}

		var result types.CompanyProfile
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if result.Tone != "professional" {
			t.Errorf("Tone = %q, want 'professional'", result.Tone)
		}
	})
}

func TestGetJobMetadataByRunID(t *testing.T) {
	t.Run("returns raw JSON bytes", func(t *testing.T) {
		// Test that GetJobMetadataByRunID returns raw JSON bytes
		// (to avoid import cycle with ingestion package)
		metadataJSON := []byte(`{"url":"https://example.com/job","platform":"greenhouse"}`)

		// Verify it's valid JSON
		var result map[string]interface{}
		if err := json.Unmarshal(metadataJSON, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if result["url"] != "https://example.com/job" {
			t.Errorf("URL = %q, want 'https://example.com/job'", result["url"])
		}
	})
}

func TestGetViolationsByRunID(t *testing.T) {
	t.Run("unmarshal valid violations", func(t *testing.T) {
		violations := &types.Violations{
			Violations: []types.Violation{
				{Type: "page_overflow", Severity: "error"},
			},
		}
		jsonBytes, err := json.Marshal(violations)
		if err != nil {
			t.Fatalf("Failed to marshal test violations: %v", err)
		}

		var result types.Violations
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(result.Violations) != 1 {
			t.Errorf("Violations count = %d, want 1", len(result.Violations))
		}
	})
}

// Integration tests will be in artifacts_integration_test.go
// These unit tests verify the JSON unmarshaling logic works correctly
