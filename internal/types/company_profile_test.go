// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompanyProfile_JSONMarshaling(t *testing.T) {
	profile := CompanyProfile{
		Company:       "Acme Corp",
		Tone:          "direct, metric-driven",
		StyleRules:    []string{"Lead with quantified impact", "Avoid marketing jargon", "Use active voice"},
		TabooPhrases:  []string{"synergy", "ninja", "rockstar"},
		DomainContext: "B2B SaaS, infrastructure",
		Values:        []string{"Ownership", "Customer obsession"},
		EvidenceURLs:  []string{"https://company.com/values", "https://company.com/culture"},
	}

	jsonBytes, err := json.MarshalIndent(profile, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"company": "Acme Corp"`)
	assert.Contains(t, string(jsonBytes), `"tone": "direct, metric-driven"`)
	assert.Contains(t, string(jsonBytes), `"style_rules"`)
	assert.Contains(t, string(jsonBytes), `"Lead with quantified impact"`)
	assert.Contains(t, string(jsonBytes), `"taboo_phrases"`)
	assert.Contains(t, string(jsonBytes), `"synergy"`)
	assert.Contains(t, string(jsonBytes), `"domain_context": "B2B SaaS, infrastructure"`)
	assert.Contains(t, string(jsonBytes), `"values"`)
	assert.Contains(t, string(jsonBytes), `"Ownership"`)
	assert.Contains(t, string(jsonBytes), `"evidence_urls"`)
	assert.Contains(t, string(jsonBytes), `"https://company.com/values"`)

	var unmarshaledProfile CompanyProfile
	err = json.Unmarshal(jsonBytes, &unmarshaledProfile)
	require.NoError(t, err)
	assert.Equal(t, profile, unmarshaledProfile)
}

func TestCompanyProfile_EmptyArrays(t *testing.T) {
	profile := CompanyProfile{
		Company:       "Test Company",
		Tone:          "professional",
		StyleRules:    []string{},
		TabooPhrases:  []string{},
		DomainContext: "Technology",
		Values:        []string{},
		EvidenceURLs:  []string{},
	}

	jsonBytes, err := json.Marshal(profile)
	require.NoError(t, err)

	var unmarshaledProfile CompanyProfile
	err = json.Unmarshal(jsonBytes, &unmarshaledProfile)
	require.NoError(t, err)
	assert.Equal(t, profile, unmarshaledProfile)
	assert.NotNil(t, unmarshaledProfile.StyleRules)
	assert.NotNil(t, unmarshaledProfile.TabooPhrases)
	assert.NotNil(t, unmarshaledProfile.Values)
	assert.NotNil(t, unmarshaledProfile.EvidenceURLs)
}

func TestCompanyProfile_RequiredFields(t *testing.T) {
	// Test that all required fields from schema are present
	profile := CompanyProfile{
		Company:       "Required",
		Tone:          "Required",
		StyleRules:    []string{"Required"},
		TabooPhrases:  []string{"Required"},
		DomainContext: "Required",
		Values:        []string{"Required"},
		EvidenceURLs:  []string{"https://example.com"},
	}

	jsonBytes, err := json.Marshal(profile)
	require.NoError(t, err)

	var unmarshaledProfile CompanyProfile
	err = json.Unmarshal(jsonBytes, &unmarshaledProfile)
	require.NoError(t, err)

	assert.NotEmpty(t, unmarshaledProfile.Company)
	assert.NotEmpty(t, unmarshaledProfile.Tone)
	assert.NotEmpty(t, unmarshaledProfile.StyleRules)
	assert.NotEmpty(t, unmarshaledProfile.TabooPhrases)
	assert.NotEmpty(t, unmarshaledProfile.DomainContext)
	assert.NotEmpty(t, unmarshaledProfile.Values)
	assert.NotEmpty(t, unmarshaledProfile.EvidenceURLs)
}
