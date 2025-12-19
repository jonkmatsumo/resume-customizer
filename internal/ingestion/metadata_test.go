package ingestion

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadata_JSONMarshaling(t *testing.T) {
	metadata := &Metadata{
		URL:       "https://example.com/job",
		Timestamp: "2024-01-01T00:00:00Z",
		Hash:      "abcd1234",
	}

	// Test marshaling
	jsonBytes, err := metadata.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	// Test that it's valid JSON
	var unmarshaled Metadata
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, metadata.URL, unmarshaled.URL)
	assert.Equal(t, metadata.Timestamp, unmarshaled.Timestamp)
	assert.Equal(t, metadata.Hash, unmarshaled.Hash)
}

func TestMetadata_JSONUnmarshaling(t *testing.T) {
	jsonStr := `{
  "url": "https://example.com/job",
  "timestamp": "2024-01-01T00:00:00Z",
  "hash": "abcd1234"
}`

	var metadata Metadata
	err := json.Unmarshal([]byte(jsonStr), &metadata)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/job", metadata.URL)
	assert.Equal(t, "2024-01-01T00:00:00Z", metadata.Timestamp)
	assert.Equal(t, "abcd1234", metadata.Hash)
}

func TestComputeHash(t *testing.T) {
	content1 := "test content"
	content2 := "different content"

	hash1 := computeHash(content1)
	hash2 := computeHash(content2)

	// Hash should be 64 hex characters (SHA256)
	assert.Len(t, hash1, 64)
	assert.Len(t, hash2, 64)

	// Different content should produce different hashes
	assert.NotEqual(t, hash1, hash2)

	// Same content should produce same hash
	hash1Again := computeHash(content1)
	assert.Equal(t, hash1, hash1Again)
}

func TestNewMetadata(t *testing.T) {
	content := "test content"
	url := "https://example.com/job"

	metadata := NewMetadata(content, url)

	assert.Equal(t, url, metadata.URL)
	assert.NotEmpty(t, metadata.Timestamp)
	assert.Len(t, metadata.Hash, 64) // SHA256 hex length

	// Verify timestamp is valid RFC3339
	_, err := time.Parse(time.RFC3339, metadata.Timestamp)
	assert.NoError(t, err)

	// Verify hash is computed from content
	expectedHash := computeHash(content)
	assert.Equal(t, expectedHash, metadata.Hash)
}

func TestNewMetadata_EmptyURL(t *testing.T) {
	content := "test content"
	metadata := NewMetadata(content, "")

	assert.Empty(t, metadata.URL)
	assert.NotEmpty(t, metadata.Timestamp)
	assert.NotEmpty(t, metadata.Hash)
}
