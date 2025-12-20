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

func TestSource_JSONMarshaling(t *testing.T) {
	source := Source{
		URL:       "https://example.com/page",
		Timestamp: "2024-01-01T00:00:00Z",
		Hash:      "abc123",
	}

	jsonBytes, err := json.MarshalIndent(source, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"url": "https://example.com/page"`)
	assert.Contains(t, string(jsonBytes), `"timestamp": "2024-01-01T00:00:00Z"`)
	assert.Contains(t, string(jsonBytes), `"hash": "abc123"`)

	var unmarshaledSource Source
	err = json.Unmarshal(jsonBytes, &unmarshaledSource)
	require.NoError(t, err)
	assert.Equal(t, source, unmarshaledSource)
}

func TestCompanyCorpus_JSONMarshaling(t *testing.T) {
	corpus := CompanyCorpus{
		Corpus: "Page 1 content\n\n---\n\nPage 2 content",
		Sources: []Source{
			{
				URL:       "https://example.com/page1",
				Timestamp: "2024-01-01T00:00:00Z",
				Hash:      "hash1",
			},
			{
				URL:       "https://example.com/page2",
				Timestamp: "2024-01-01T00:00:01Z",
				Hash:      "hash2",
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(corpus, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"corpus":`)
	assert.Contains(t, string(jsonBytes), `"sources":`)
	assert.Contains(t, string(jsonBytes), `"url": "https://example.com/page1"`)
	assert.Contains(t, string(jsonBytes), `"url": "https://example.com/page2"`)

	var unmarshaledCorpus CompanyCorpus
	err = json.Unmarshal(jsonBytes, &unmarshaledCorpus)
	require.NoError(t, err)
	assert.Equal(t, corpus, unmarshaledCorpus)
}

func TestCompanyCorpus_EmptySources(t *testing.T) {
	corpus := CompanyCorpus{
		Corpus:  "Some content",
		Sources: []Source{},
	}

	jsonBytes, err := json.MarshalIndent(corpus, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"sources": []`)

	var unmarshaledCorpus CompanyCorpus
	err = json.Unmarshal(jsonBytes, &unmarshaledCorpus)
	require.NoError(t, err)
	assert.Equal(t, corpus, unmarshaledCorpus)
}
