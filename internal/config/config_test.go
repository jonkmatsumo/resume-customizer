package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_ValidJSON(t *testing.T) {
	// Create temp config file
	content := `{
		"user_id": "550e8400-e29b-41d4-a716-446655440000",
		"job_url": "https://example.com/job",
		"name": "Test User",
		"max_bullets": 20,
		"verbose": true
	}`

	tmpFile := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(tmpFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", cfg.UserID)
	assert.Equal(t, "https://example.com/job", cfg.JobURL)
	assert.Equal(t, "Test User", cfg.Name)
	assert.Equal(t, 20, cfg.MaxBullets)
	assert.True(t, cfg.Verbose)
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	content := `{ invalid json }`

	tmpFile := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(tmpFile)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to parse config JSON")
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path/config.json")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadConfig_EmptyPath(t *testing.T) {
	cfg, err := LoadConfig("")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "config path is empty")
}

func TestValidate_MutuallyExclusive(t *testing.T) {
	cfg := &Config{
		Job:    "job.txt",
		JobURL: "https://example.com/job",
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestValidate_NegativeValues(t *testing.T) {
	cfg := &Config{
		MaxBullets: -1,
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_bullets")
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Name:       "Test User",
		MaxBullets: 25,
		MaxLines:   35,
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestMergeWithDefaults(t *testing.T) {
	defaults := Config{
		Name:       "Default Name",
		Email:      "default@example.com",
		Template:   "default.tex",
		MaxBullets: 25,
		MaxLines:   35,
	}

	partial := Config{
		Name:   "Custom Name",
		UserID: "custom-user-id",
	}

	merged := partial.MergeWithDefaults(defaults)

	// Custom values should be preserved
	assert.Equal(t, "Custom Name", merged.Name)
	assert.Equal(t, "custom-user-id", merged.UserID)

	// Default values should fill in empty fields
	assert.Equal(t, "default@example.com", merged.Email)
	assert.Equal(t, "default.tex", merged.Template)
	assert.Equal(t, 25, merged.MaxBullets)
	assert.Equal(t, 35, merged.MaxLines)
}

func TestMergeWithDefaults_EmptyDefaults(t *testing.T) {
	cfg := Config{
		Name:   "Test",
		UserID: "test-user",
	}

	merged := cfg.MergeWithDefaults(Config{})

	assert.Equal(t, "Test", merged.Name)
	assert.Equal(t, "test-user", merged.UserID)
}
