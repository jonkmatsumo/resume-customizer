// Package llm - util.go provides shared utilities for LLM response processing.
package llm

import "strings"

// CleanJSONBlock removes markdown code block wrappers from JSON responses.
// LLMs often wrap JSON in ```json ... ``` blocks even when instructed not to.
func CleanJSONBlock(text string) string {
	text = strings.TrimSpace(text)

	// Handle ```json ... ``` blocks
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		if idx := strings.LastIndex(text, "```"); idx >= 0 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
		return text
	}

	// Handle generic ``` ... ``` blocks
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		// Skip potential language identifier on first line
		if idx := strings.Index(text, "\n"); idx >= 0 {
			firstLine := text[:idx]
			// If first line looks like a language identifier (no spaces, short), skip it
			if len(firstLine) < 20 && !strings.Contains(firstLine, " ") && !strings.Contains(firstLine, "{") {
				text = text[idx+1:]
			}
		}
		if idx := strings.LastIndex(text, "```"); idx >= 0 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
		return text
	}

	return text
}
