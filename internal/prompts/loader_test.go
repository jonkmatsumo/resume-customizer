package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet_ValidPrompt(t *testing.T) {
	ClearCache()

	prompt, err := Get("parsing.json", "extract-job-profile")
	require.NoError(t, err)
	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "Extract structured information")
}

func TestGet_InvalidFile(t *testing.T) {
	ClearCache()

	_, err := Get("nonexistent.json", "some-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read prompt file")
}

func TestGet_InvalidKey(t *testing.T) {
	ClearCache()

	_, err := Get("parsing.json", "nonexistent-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMustGet_Panics(t *testing.T) {
	ClearCache()

	assert.Panics(t, func() {
		MustGet("nonexistent.json", "some-key")
	})
}

func TestMustGet_ValidPrompt(t *testing.T) {
	ClearCache()

	assert.NotPanics(t, func() {
		prompt := MustGet("parsing.json", "extract-job-profile")
		assert.NotEmpty(t, prompt)
	})
}

func TestFormat(t *testing.T) {
	template := "Hello {{.Name}}, welcome to {{.Company}}!"
	data := map[string]string{
		"Name":    "Alice",
		"Company": "Acme Corp",
	}

	result := Format(template, data)
	assert.Equal(t, "Hello Alice, welcome to Acme Corp!", result)
}

func TestFormat_NoPlaceholders(t *testing.T) {
	template := "No placeholders here"
	data := map[string]string{"Key": "Value"}

	result := Format(template, data)
	assert.Equal(t, template, result)
}

func TestFormat_EmptyData(t *testing.T) {
	template := "Hello {{.Name}}"
	data := map[string]string{}

	result := Format(template, data)
	assert.Equal(t, template, result) // Placeholder remains
}

func TestList(t *testing.T) {
	ClearCache()

	keys, err := List("parsing.json")
	require.NoError(t, err)
	assert.Contains(t, keys, "extract-job-profile")
}

func TestCaching(t *testing.T) {
	ClearCache()

	// First call loads from file
	prompt1, err := Get("parsing.json", "extract-job-profile")
	require.NoError(t, err)

	// Second call should use cache
	prompt2, err := Get("parsing.json", "extract-job-profile")
	require.NoError(t, err)

	assert.Equal(t, prompt1, prompt2)
}
