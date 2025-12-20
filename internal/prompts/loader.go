// Package prompts provides a loader for externalized LLM prompt templates.
// Prompts are stored as JSON files and embedded at compile time.
package prompts

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed *.json
var promptFiles embed.FS

// cache stores parsed prompt files to avoid repeated JSON parsing
var (
	cache   = make(map[string]map[string]string)
	cacheMu sync.RWMutex
)

// Get retrieves a prompt by filename and key.
// The filename should not include the path (e.g., "parsing.json").
// Returns an error if the file or key is not found.
func Get(filename, key string) (string, error) {
	prompts, err := loadFile(filename)
	if err != nil {
		return "", err
	}

	prompt, exists := prompts[key]
	if !exists {
		return "", fmt.Errorf("prompt key %q not found in %s", key, filename)
	}

	return prompt, nil
}

// MustGet retrieves a prompt by filename and key, panicking if not found.
// Use this for prompts that are required at initialization time.
func MustGet(filename, key string) string {
	prompt, err := Get(filename, key)
	if err != nil {
		panic(fmt.Sprintf("failed to load prompt: %v", err))
	}
	return prompt
}

// Format replaces template placeholders in the form {{.Key}} with values from data.
// This is a simple template system for prompt customization.
func Format(template string, data map[string]string) string {
	result := template
	for key, value := range data {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// loadFile loads and caches a prompt file.
func loadFile(filename string) (map[string]string, error) {
	// Check cache first
	cacheMu.RLock()
	if prompts, exists := cache[filename]; exists {
		cacheMu.RUnlock()
		return prompts, nil
	}
	cacheMu.RUnlock()

	// Load from embedded filesystem
	data, err := promptFiles.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file %s: %w", filename, err)
	}

	var prompts map[string]string
	if err := json.Unmarshal(data, &prompts); err != nil {
		return nil, fmt.Errorf("failed to parse prompt file %s: %w", filename, err)
	}

	// Cache the result
	cacheMu.Lock()
	cache[filename] = prompts
	cacheMu.Unlock()

	return prompts, nil
}

// ClearCache clears the prompt cache. Useful for testing.
func ClearCache() {
	cacheMu.Lock()
	cache = make(map[string]map[string]string)
	cacheMu.Unlock()
}

// List returns all available prompt keys in a file.
func List(filename string) ([]string, error) {
	prompts, err := loadFile(filename)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(prompts))
	for key := range prompts {
		keys = append(keys, key)
	}
	return keys, nil
}
