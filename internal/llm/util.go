// Package llm - util.go provides shared utilities for LLM response processing.
package llm

import "strings"

// CleanJSONBlock removes markdown code block wrappers and preamble text from JSON responses.
// LLMs often wrap JSON in ```json ... ``` blocks or include conversational text before the JSON.
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

	// Handle preamble or trailing text around JSON
	// Look for the first '{' or '[' which starts JSON
	if jsonStart := strings.Index(text, "{"); jsonStart >= 0 {
		// Find the matching closing brace
		extracted := extractJSONObject(text[jsonStart:])
		if extracted != "" {
			return extracted
		}
	}
	// Try to find JSON array
	if jsonStart := strings.Index(text, "["); jsonStart >= 0 {
		extracted := extractJSONArray(text[jsonStart:])
		if extracted != "" {
			return extracted
		}
	}

	return text
}

// extractJSONObject attempts to extract a complete JSON object from text starting with '{'
func extractJSONObject(text string) string {
	if !strings.HasPrefix(text, "{") {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i, char := range text {
		if escaped {
			escaped = false
			continue
		}

		if char == '\\' && inString {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch char {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[:i+1]
			}
		}
	}

	// No matching closing brace found, return original
	return text
}

// extractJSONArray attempts to extract a complete JSON array from text starting with '['
func extractJSONArray(text string) string {
	if !strings.HasPrefix(text, "[") {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i, char := range text {
		if escaped {
			escaped = false
			continue
		}

		if char == '\\' && inString {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch char {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return text[:i+1]
			}
		}
	}

	// No matching closing bracket found, return original
	return text
}
