//go:build integration
// +build integration

package voice

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeVoice_RealAPI(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	// Sample corpus text
	corpusText := `Our company values ownership and customer obsession above all else. 
We communicate in a direct, metric-driven manner. We avoid marketing jargon and buzzwords.
Our style emphasizes leading with quantified impact. We use active voice in all communications.
We operate in the B2B SaaS infrastructure domain.`

	sources := []types.Source{
		{URL: "https://example.com/values", Timestamp: "2023-10-27T10:00:00Z", Hash: "hash1"},
		{URL: "https://example.com/culture", Timestamp: "2023-10-27T10:05:00Z", Hash: "hash2"},
	}

	ctx := context.Background()
	profile, err := SummarizeVoice(ctx, corpusText, sources, apiKey)
	require.NoError(t, err)
	require.NotNil(t, profile)

	// Verify all required fields are present
	assert.NotEmpty(t, profile.Company)
	assert.NotEmpty(t, profile.Tone)
	assert.NotEmpty(t, profile.StyleRules)
	assert.NotEmpty(t, profile.TabooPhrases)
	assert.NotEmpty(t, profile.DomainContext)
	assert.NotEmpty(t, profile.Values)
	assert.NotEmpty(t, profile.EvidenceURLs)

	// Verify evidence URLs are populated from sources
	assert.Len(t, profile.EvidenceURLs, 2)
	assert.Contains(t, profile.EvidenceURLs, "https://example.com/values")
	assert.Contains(t, profile.EvidenceURLs, "https://example.com/culture")

	// Verify style rules are actionable
	for _, rule := range profile.StyleRules {
		assert.NotEmpty(t, rule)
	}
}

func TestSummarizeVoice_SchemaValidation(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	corpusText := `We value transparency and innovation. Our communication style is collaborative and values-driven.
We focus on the fintech domain, building financial infrastructure.`

	sources := []types.Source{
		{URL: "https://example.com/about", Timestamp: "2023-10-27T10:00:00Z", Hash: "hash1"},
	}

	ctx := context.Background()
	profile, err := SummarizeVoice(ctx, corpusText, sources, apiKey)
	require.NoError(t, err)

	// Marshal to JSON
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "company_profile.json")

	jsonBytes, err := json.MarshalIndent(profile, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(outputPath, jsonBytes, 0644)
	require.NoError(t, err)

	// Validate against schema
	schemaPath := filepath.Join("..", "..", "schemas", "company_profile.schema.json")
	err = schemas.ValidateJSON(schemaPath, outputPath)
	assert.NoError(t, err, "generated profile should validate against schema")
}

func TestSummarizeVoice_Stability(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	corpusText := `Our company emphasizes ownership and direct communication. We avoid hype and focus on metrics.
Domain: B2B SaaS infrastructure. Values: ownership, customer obsession.`

	sources := []types.Source{
		{URL: "https://example.com/values", Timestamp: "2023-10-27T10:00:00Z", Hash: "hash1"},
	}

	ctx := context.Background()

	// Run twice with same input
	profile1, err := SummarizeVoice(ctx, corpusText, sources, apiKey)
	require.NoError(t, err)

	profile2, err := SummarizeVoice(ctx, corpusText, sources, apiKey)
	require.NoError(t, err)

	// With low temperature, results should be similar (not necessarily identical due to LLM variance)
	// Check that key fields are present in both
	assert.NotEmpty(t, profile1.Company)
	assert.NotEmpty(t, profile2.Company)
	assert.NotEmpty(t, profile1.Tone)
	assert.NotEmpty(t, profile2.Tone)
	assert.NotEmpty(t, profile1.Values)
	assert.NotEmpty(t, profile2.Values)

	// Evidence URLs should be identical (populated deterministically from sources)
	assert.Equal(t, profile1.EvidenceURLs, profile2.EvidenceURLs)
}

func TestSummarizeVoice_MissingAPIKey(t *testing.T) {
	corpusText := "Test corpus"
	sources := []types.Source{}

	ctx := context.Background()
	_, err := SummarizeVoice(ctx, corpusText, sources, "")
	assert.Error(t, err)
	var apiErr *APICallError
	assert.ErrorAs(t, err, &apiErr)
	assert.Contains(t, err.Error(), "API key is required")
}
