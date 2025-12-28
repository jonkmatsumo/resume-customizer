package llm

import (
	"testing"
)

func TestCleanJSONBlock_MarkdownCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "json code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "generic code block",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "code block with language",
			input:    "```javascript\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "plain JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanJSONBlock(tt.input)
			if result != tt.expected {
				t.Errorf("CleanJSONBlock() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanJSONBlock_PreambleText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "preamble before JSON object",
			input:    "As requested, here is the JSON:\n{\"company\": \"Acme\"}",
			expected: `{"company": "Acme"}`,
		},
		{
			name:     "conversational preamble",
			input:    "Based on the company information provided, I've analyzed the brand voice. Here's the structured output:\n\n{\"company\": \"Test\", \"tone\": \"professional\"}",
			expected: `{"company": "Test", "tone": "professional"}`,
		},
		{
			name:     "preamble with multiple sentences",
			input:    "I analyzed the text. The company values innovation. Here is the result: {\"values\": [\"innovation\"]}",
			expected: `{"values": ["innovation"]}`,
		},
		{
			name:     "preamble before JSON array",
			input:    "Here are the items:\n[\"item1\", \"item2\"]",
			expected: `["item1", "item2"]`,
		},
		{
			name:     "JSON with trailing text",
			input:    "{\"key\": \"value\"}\n\nLet me know if you need anything else!",
			expected: `{"key": "value"}`,
		},
		{
			name:     "nested objects",
			input:    "Output:\n{\"outer\": {\"inner\": \"value\"}}",
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:     "JSON with escaped quotes",
			input:    "Result: {\"message\": \"He said \\\"hello\\\"\"}",
			expected: `{"message": "He said \"hello\""}`,
		},
		{
			name:     "deeply nested",
			input:    "Here: {\"a\": {\"b\": {\"c\": {\"d\": \"deep\"}}}}",
			expected: `{"a": {"b": {"c": {"d": "deep"}}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanJSONBlock(tt.input)
			if result != tt.expected {
				t.Errorf("CleanJSONBlock() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractJSONObject(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple object",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "nested objects",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:     "object with array",
			input:    `{"items": [1, 2, 3]}`,
			expected: `{"items": [1, 2, 3]}`,
		},
		{
			name:     "object with trailing text",
			input:    `{"key": "value"} and some more text`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "string with braces inside",
			input:    `{"template": "Hello {name}!"}`,
			expected: `{"template": "Hello {name}!"}`,
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "not starting with brace",
			input:    "not json",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONObject(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSONObject() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractJSONArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple array",
			input:    `["a", "b", "c"]`,
			expected: `["a", "b", "c"]`,
		},
		{
			name:     "nested arrays",
			input:    `[[1, 2], [3, 4]]`,
			expected: `[[1, 2], [3, 4]]`,
		},
		{
			name:     "array of objects",
			input:    `[{"id": 1}, {"id": 2}]`,
			expected: `[{"id": 1}, {"id": 2}]`,
		},
		{
			name:     "array with trailing text",
			input:    `[1, 2, 3] extra stuff`,
			expected: `[1, 2, 3]`,
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "not starting with bracket",
			input:    "not array",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONArray(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSONArray() = %q, want %q", result, tt.expected)
			}
		})
	}
}
