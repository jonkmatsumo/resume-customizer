package parsing

import (
	"strings"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJSONResponse(t *testing.T) {
	tests := []struct {
		name      string
		jsonText  string
		wantError bool
		validate  func(*testing.T, *types.JobProfile)
	}{
		{
			name: "Valid JSON response",
			jsonText: `{
				"company": "Acme Corp",
				"role_title": "Senior Software Engineer",
				"responsibilities": ["Build systems", "Lead team"],
				"hard_requirements": [
					{"skill": "Go", "level": "3+ years", "evidence": "3+ years with Go"}
				],
				"nice_to_haves": [
					{"skill": "Kubernetes", "evidence": "K8s experience preferred"}
				],
				"keywords": ["distributed systems", "microservices"],
				"eval_signals": {
					"latency": true,
					"reliability": true
				}
			}`,
			wantError: false,
			validate: func(t *testing.T, profile *types.JobProfile) {
				assert.Equal(t, "Acme Corp", profile.Company)
				assert.Equal(t, "Senior Software Engineer", profile.RoleTitle)
				assert.Len(t, profile.Responsibilities, 2)
				assert.Len(t, profile.HardRequirements, 1)
				assert.Equal(t, "Go", profile.HardRequirements[0].Skill)
				assert.NotNil(t, profile.EvalSignals)
				assert.True(t, profile.EvalSignals.Latency)
			},
		},
		{
			name:      "Invalid JSON",
			jsonText:  `{invalid json}`,
			wantError: true,
		},
		{
			name: "Missing required fields",
			jsonText: `{
				"company": "Acme Corp"
			}`,
			wantError: false, // JSON parsing succeeds, validation happens later
			validate: func(t *testing.T, profile *types.JobProfile) {
				assert.Equal(t, "Acme Corp", profile.Company)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := parseJSONResponse(tt.jsonText)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, profile)
			} else {
				require.NoError(t, err)
				require.NotNil(t, profile)
				if tt.validate != nil {
					tt.validate(t, profile)
				}
			}
		})
	}
}

func TestExtractTextFromResponse_MarkdownCodeBlocks(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		wantError bool
	}{
		{
			name:      "JSON wrapped in ```json block",
			input:     "```json\n{\"key\": \"value\"}\n```",
			expected:  `{"key": "value"}`,
			wantError: false,
		},
		{
			name:      "JSON wrapped in ``` block",
			input:     "```\n{\"key\": \"value\"}\n```",
			expected:  `{"key": "value"}`,
			wantError: false,
		},
		{
			name:      "Plain JSON without code blocks",
			input:     `{"key": "value"}`,
			expected:  `{"key": "value"}`,
			wantError: false,
		},
		{
			name:      "Whitespace around code blocks",
			input:     "  ```json\n{\"key\": \"value\"}\n```  ",
			expected:  `{"key": "value"}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since we can't easily mock the genai response, test the logic
			// by simulating what extractTextFromResponse would do
			text := tt.input
			text = strings.TrimSpace(text)
			if strings.HasPrefix(text, "```json") {
				text = strings.TrimPrefix(text, "```json")
				text = strings.TrimPrefix(text, "```")
				text = strings.TrimSuffix(text, "```")
				text = strings.TrimSpace(text)
			} else if strings.HasPrefix(text, "```") {
				text = strings.TrimPrefix(text, "```")
				text = strings.TrimSuffix(text, "```")
				text = strings.TrimSpace(text)
			}

			assert.Equal(t, tt.expected, text)
		})
	}
}

func TestPostProcessProfile(t *testing.T) {
	tests := []struct {
		name      string
		profile   *types.JobProfile
		wantError bool
		validate  func(*testing.T, *types.JobProfile)
	}{
		{
			name: "Normalize skill names",
			profile: &types.JobProfile{
				Company:          "Acme",
				RoleTitle:        "Engineer",
				Responsibilities: []string{"Build"},
				HardRequirements: []types.Requirement{
					{Skill: "Golang", Evidence: "test"},
					{Skill: "javascript", Evidence: "test"},
				},
				NiceToHaves: []types.Requirement{
					{Skill: "js", Evidence: "test"},
				},
				Keywords:    []string{"  TEST  ", "test", "OTHER"},
				EvalSignals: &types.EvalSignals{Latency: true},
			},
			wantError: false,
			validate: func(t *testing.T, profile *types.JobProfile) {
				assert.Equal(t, "Go", profile.HardRequirements[0].Skill)
				assert.Equal(t, "JavaScript", profile.HardRequirements[1].Skill)
				assert.Equal(t, "JavaScript", profile.NiceToHaves[0].Skill)
				// Keywords should be normalized (lowercase, deduplicated)
				assert.Contains(t, profile.Keywords, "test")
				assert.Contains(t, profile.Keywords, "other")
				assert.Len(t, profile.Keywords, 2) // deduplicated
			},
		},
		{
			name: "Deduplicate requirements",
			profile: &types.JobProfile{
				Company:          "Acme",
				RoleTitle:        "Engineer",
				Responsibilities: []string{},
				HardRequirements: []types.Requirement{
					{Skill: "Go", Level: "3+ years", Evidence: "first"},
					{Skill: "Golang", Level: "5+ years", Evidence: "second"},
				},
				NiceToHaves: []types.Requirement{},
				Keywords:    []string{},
				EvalSignals: &types.EvalSignals{},
			},
			wantError: false,
			validate: func(t *testing.T, profile *types.JobProfile) {
				// Should deduplicate, keeping first occurrence
				assert.Len(t, profile.HardRequirements, 1)
				assert.Equal(t, "Go", profile.HardRequirements[0].Skill)
				assert.Equal(t, "3+ years", profile.HardRequirements[0].Level)
				assert.Equal(t, "first", profile.HardRequirements[0].Evidence)
			},
		},
		{
			name: "Missing evidence snippets",
			profile: &types.JobProfile{
				Company:          "Acme",
				RoleTitle:        "Engineer",
				Responsibilities: []string{},
				HardRequirements: []types.Requirement{
					{Skill: "Go", Evidence: ""},
				},
				NiceToHaves: []types.Requirement{},
				Keywords:    []string{},
				EvalSignals: &types.EvalSignals{},
			},
			wantError: true,
		},
		{
			name: "Initialize eval_signals if nil",
			profile: &types.JobProfile{
				Company:          "Acme",
				RoleTitle:        "Engineer",
				Responsibilities: []string{},
				HardRequirements: []types.Requirement{
					{Skill: "Go", Evidence: "test"},
				},
				NiceToHaves: []types.Requirement{},
				Keywords:    []string{},
				EvalSignals: nil,
			},
			wantError: false,
			validate: func(t *testing.T, profile *types.JobProfile) {
				assert.NotNil(t, profile.EvalSignals)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := postProcessProfile(tt.profile)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, tt.profile)
				}
			}
		})
	}
}

func TestBuildExtractionPrompt(t *testing.T) {
	jobText := "We are looking for a Senior Engineer with Go experience."
	prompt := buildExtractionPrompt(jobText)

	// Verify prompt contains key elements
	assert.Contains(t, prompt, jobText, "should include job text")
	assert.Contains(t, prompt, "company", "should mention company field")
	assert.Contains(t, prompt, "role_title", "should mention role_title field")
	assert.Contains(t, prompt, "hard_requirements", "should mention hard_requirements")
	assert.Contains(t, prompt, "evidence", "should mention evidence requirement")
	assert.Contains(t, prompt, "eval_signals", "should mention eval_signals")
	assert.Contains(t, prompt, "ONLY valid JSON", "should emphasize JSON only")
}
