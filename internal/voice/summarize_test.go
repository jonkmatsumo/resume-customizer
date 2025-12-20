// Package voice provides functionality to extract brand voice and style rules from company corpus text.
package voice

import (
	"testing"

	"github.com/google/generative-ai-go/genai"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildExtractionPrompt(t *testing.T) {
	corpusText := "We value ownership and customer obsession. Our tone is direct and metric-driven."
	sourceURLs := []string{"https://example.com/values", "https://example.com/culture"}

	prompt := buildExtractionPrompt(corpusText, sourceURLs)

	assert.Contains(t, prompt, corpusText)
	assert.Contains(t, prompt, "https://example.com/values")
	assert.Contains(t, prompt, "https://example.com/culture")
	assert.Contains(t, prompt, "company")
	assert.Contains(t, prompt, "tone")
	assert.Contains(t, prompt, "style_rules")
	assert.Contains(t, prompt, "taboo_phrases")
	assert.Contains(t, prompt, "domain_context")
	assert.Contains(t, prompt, "values")
	assert.Contains(t, prompt, "actionable")
}

func TestBuildExtractionPrompt_NoSources(t *testing.T) {
	corpusText := "Test corpus text"
	sourceURLs := []string{}

	prompt := buildExtractionPrompt(corpusText, sourceURLs)

	assert.Contains(t, prompt, corpusText)
	assert.NotContains(t, prompt, "Sources (for context):")
}

func TestExtractTextFromResponse_ValidResponse(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []genai.Part{
						genai.Text(`{"company": "Test", "tone": "direct"}`),
					},
				},
			},
		},
	}

	text, err := extractTextFromResponse(resp)
	require.NoError(t, err)
	assert.Contains(t, text, "company")
	assert.Contains(t, text, "Test")
}

func TestExtractTextFromResponse_WithMarkdownCodeBlocks(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []genai.Part{
						genai.Text("```json\n{\"company\": \"Test\"}\n```"),
					},
				},
			},
		},
	}

	text, err := extractTextFromResponse(resp)
	require.NoError(t, err)
	assert.Contains(t, text, "company")
	assert.Contains(t, text, "Test")
	assert.NotContains(t, text, "```")
}

func TestExtractTextFromResponse_WithJSONCodeBlock(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []genai.Part{
						genai.Text("```json\n{\"company\": \"Test\"}\n```"),
					},
				},
			},
		},
	}

	text, err := extractTextFromResponse(resp)
	require.NoError(t, err)
	assert.Contains(t, text, "company")
	assert.NotContains(t, text, "```json")
}

func TestExtractTextFromResponse_NoCandidates(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{},
	}

	_, err := extractTextFromResponse(resp)
	assert.Error(t, err)
	var parseErr *ParseError
	assert.ErrorAs(t, err, &parseErr)
	assert.Contains(t, err.Error(), "no candidates")
}

func TestExtractTextFromResponse_NoContent(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: nil,
			},
		},
	}

	_, err := extractTextFromResponse(resp)
	assert.Error(t, err)
	var parseErr *ParseError
	assert.ErrorAs(t, err, &parseErr)
	assert.Contains(t, err.Error(), "no content")
}

func TestParseJSONResponse_ValidJSON(t *testing.T) {
	jsonText := `{
		"company": "Acme Corp",
		"tone": "direct, metric-driven",
		"style_rules": ["Lead with metrics"],
		"taboo_phrases": ["synergy"],
		"domain_context": "B2B SaaS",
		"values": ["Ownership"]
	}`

	profile, err := parseJSONResponse(jsonText)
	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", profile.Company)
	assert.Equal(t, "direct, metric-driven", profile.Tone)
	assert.Len(t, profile.StyleRules, 1)
	assert.Len(t, profile.TabooPhrases, 1)
	assert.Equal(t, "B2B SaaS", profile.DomainContext)
	assert.Len(t, profile.Values, 1)
}

func TestParseJSONResponse_InvalidJSON(t *testing.T) {
	jsonText := "not valid json"

	_, err := parseJSONResponse(jsonText)
	assert.Error(t, err)
	var parseErr *ParseError
	assert.ErrorAs(t, err, &parseErr)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestPostProcessProfile_ValidProfile(t *testing.T) {
	profile := &types.CompanyProfile{
		Company:       "Test Company",
		Tone:          "professional",
		StyleRules:    []string{"Use active voice"},
		TabooPhrases:  []string{"synergy"},
		DomainContext: "Technology",
		Values:        []string{"Ownership"},
		EvidenceURLs:  []string{}, // Will be populated
	}

	sources := []types.Source{
		{URL: "https://example.com/values"},
		{URL: "https://example.com/culture"},
	}

	err := postProcessProfile(profile, sources)
	require.NoError(t, err)
	assert.Len(t, profile.EvidenceURLs, 2)
	assert.Contains(t, profile.EvidenceURLs, "https://example.com/values")
	assert.Contains(t, profile.EvidenceURLs, "https://example.com/culture")
}

func TestPostProcessProfile_MissingCompany(t *testing.T) {
	profile := &types.CompanyProfile{
		Company:       "",
		Tone:          "professional",
		StyleRules:    []string{"Use active voice"},
		TabooPhrases:  []string{"synergy"},
		DomainContext: "Technology",
		Values:        []string{"Ownership"},
	}

	err := postProcessProfile(profile, []types.Source{})
	assert.Error(t, err)
	var validationErr *ValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "company", validationErr.Field)
}

func TestPostProcessProfile_MissingTone(t *testing.T) {
	profile := &types.CompanyProfile{
		Company:       "Test",
		Tone:          "",
		StyleRules:    []string{"Use active voice"},
		TabooPhrases:  []string{"synergy"},
		DomainContext: "Technology",
		Values:        []string{"Ownership"},
	}

	err := postProcessProfile(profile, []types.Source{})
	assert.Error(t, err)
	var validationErr *ValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "tone", validationErr.Field)
}

func TestPostProcessProfile_EmptyStyleRules(t *testing.T) {
	profile := &types.CompanyProfile{
		Company:       "Test",
		Tone:          "professional",
		StyleRules:    []string{},
		TabooPhrases:  []string{"synergy"},
		DomainContext: "Technology",
		Values:        []string{"Ownership"},
	}

	err := postProcessProfile(profile, []types.Source{})
	assert.Error(t, err)
	var validationErr *ValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "style_rules", validationErr.Field)
}

func TestPostProcessProfile_EmptyTabooPhrases(t *testing.T) {
	profile := &types.CompanyProfile{
		Company:       "Test",
		Tone:          "professional",
		StyleRules:    []string{"Use active voice"},
		TabooPhrases:  []string{},
		DomainContext: "Technology",
		Values:        []string{"Ownership"},
	}

	err := postProcessProfile(profile, []types.Source{})
	assert.Error(t, err)
	var validationErr *ValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "taboo_phrases", validationErr.Field)
}

func TestPostProcessProfile_MissingDomainContext(t *testing.T) {
	profile := &types.CompanyProfile{
		Company:       "Test",
		Tone:          "professional",
		StyleRules:    []string{"Use active voice"},
		TabooPhrases:  []string{"synergy"},
		DomainContext: "",
		Values:        []string{"Ownership"},
	}

	err := postProcessProfile(profile, []types.Source{})
	assert.Error(t, err)
	var validationErr *ValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "domain_context", validationErr.Field)
}

func TestPostProcessProfile_EmptyValues(t *testing.T) {
	profile := &types.CompanyProfile{
		Company:       "Test",
		Tone:          "professional",
		StyleRules:    []string{"Use active voice"},
		TabooPhrases:  []string{"synergy"},
		DomainContext: "Technology",
		Values:        []string{},
	}

	err := postProcessProfile(profile, []types.Source{})
	assert.Error(t, err)
	var validationErr *ValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "values", validationErr.Field)
}
