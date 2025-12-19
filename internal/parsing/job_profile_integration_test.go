//go:build integration
// +build integration

package parsing

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJobProfile_Integration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	tests := []struct {
		name     string
		fixture  string
		validate func(*testing.T, interface{})
	}{
		{
			name:    "Markdown format",
			fixture: "testdata/parsing/sample_job_markdown.txt",
			validate: func(t *testing.T, profile interface{}) {
				// Basic validation - profile should be non-nil and have required fields
				// Schema validation happens separately
			},
		},
		{
			name:    "Plain text format",
			fixture: "testdata/parsing/sample_job_plain.txt",
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read fixture
			fixturePath := filepath.Join("..", "..", tt.fixture)
			content, err := os.ReadFile(fixturePath)
			require.NoError(t, err, "should read fixture file")

			// Parse job profile
			profile, err := ParseJobProfile(ctx, string(content), apiKey)
			require.NoError(t, err, "should parse job profile successfully")
			require.NotNil(t, profile, "profile should not be nil")

			// Validate required fields
			assert.NotEmpty(t, profile.Company, "company should be set")
			assert.NotEmpty(t, profile.RoleTitle, "role_title should be set")
			assert.NotEmpty(t, profile.Responsibilities, "responsibilities should not be empty")
			assert.NotEmpty(t, profile.HardRequirements, "hard_requirements should not be empty")
			assert.NotEmpty(t, profile.Keywords, "keywords should not be empty")
			assert.NotNil(t, profile.EvalSignals, "eval_signals should be set")

			// Validate evidence snippets
			for i, req := range profile.HardRequirements {
				assert.NotEmpty(t, req.Evidence, "hard_requirements[%d].evidence should not be empty", i)
				assert.NotEmpty(t, req.Skill, "hard_requirements[%d].skill should not be empty", i)
			}

			for i, req := range profile.NiceToHaves {
				if req.Skill != "" {
					assert.NotEmpty(t, req.Evidence, "nice_to_haves[%d].evidence should not be empty", i)
				}
			}

			// Validate schema by writing to temp file and validating
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "job_profile.json")

			// We'll marshal and validate - but we need json package
			// For now, just check basic structure
			// Full schema validation will be done via CLI tests
		})
	}
}

func TestParseJobProfile_SchemaValidation(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	fixturePath := filepath.Join("..", "..", "testdata", "parsing", "sample_job_markdown.txt")
	content, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	ctx := context.Background()
	profile, err := ParseJobProfile(ctx, string(content), apiKey)
	require.NoError(t, err)

	// Validate against schema using the schemas package
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "job_profile.json")

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(profile, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(outputPath, jsonBytes, 0644)
	require.NoError(t, err)

	// Validate against schema
	schemaPath := filepath.Join("..", "..", "schemas", "job_profile.schema.json")
	err = schemas.ValidateJSON(schemaPath, outputPath)
	assert.NoError(t, err, "generated profile should validate against schema")
}

func TestParseJobProfile_SkillNormalization(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	// Use a job posting that mentions "Golang" to test normalization
	jobText := `Senior Software Engineer

Requirements:
- 3+ years experience with Golang
- Experience with JavaScript
- Kubernetes (K8s) preferred`

	ctx := context.Background()
	profile, err := ParseJobProfile(ctx, jobText, apiKey)
	require.NoError(t, err)

	// Check that "Golang" was normalized to "Go"
	foundGo := false
	for _, req := range profile.HardRequirements {
		if req.Skill == "Go" {
			foundGo = true
			break
		}
	}
	assert.True(t, foundGo, "Golang should be normalized to Go")
}
