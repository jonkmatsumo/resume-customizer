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

func TestViolation_JSONMarshaling(t *testing.T) {
	lineNum := 10
	charCount := 95
	violation := Violation{
		Type:             "line_too_long",
		Severity:         "error",
		Details:          "Line exceeds maximum character count",
		AffectedSections: []string{"experience"},
		LineNumber:       &lineNum,
		CharCount:        &charCount,
	}

	jsonBytes, err := json.MarshalIndent(violation, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"type": "line_too_long"`)
	assert.Contains(t, string(jsonBytes), `"severity": "error"`)
	assert.Contains(t, string(jsonBytes), `"details": "Line exceeds maximum character count"`)
	assert.Contains(t, string(jsonBytes), `"affected_sections": [`)
	assert.Contains(t, string(jsonBytes), `"line_number": 10`)
	assert.Contains(t, string(jsonBytes), `"char_count": 95`)

	var unmarshaled Violation
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, violation.Type, unmarshaled.Type)
	assert.Equal(t, violation.Severity, unmarshaled.Severity)
	assert.Equal(t, violation.Details, unmarshaled.Details)
	assert.Equal(t, lineNum, *unmarshaled.LineNumber)
	assert.Equal(t, charCount, *unmarshaled.CharCount)
}

func TestViolation_OptionalFields(t *testing.T) {
	violation := Violation{
		Type:     "page_overflow",
		Severity: "error",
		Details:  "Resume exceeds maximum page count",
	}

	jsonBytes, err := json.Marshal(violation)
	require.NoError(t, err)

	var unmarshaled Violation
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Nil(t, unmarshaled.LineNumber)
	assert.Nil(t, unmarshaled.CharCount)
	assert.Empty(t, unmarshaled.AffectedSections)
}

func TestViolations_JSONMarshaling(t *testing.T) {
	lineNum := 5
	violations := Violations{
		Violations: []Violation{
			{
				Type:     "page_overflow",
				Severity: "error",
				Details:  "Resume has 2 pages, maximum is 1",
			},
			{
				Type:       "line_too_long",
				Severity:   "warning",
				Details:    "Line exceeds character limit",
				LineNumber: &lineNum,
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(violations, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"violations": [`)
	assert.Contains(t, string(jsonBytes), `"page_overflow"`)
	assert.Contains(t, string(jsonBytes), `"line_too_long"`)

	var unmarshaled Violations
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Len(t, unmarshaled.Violations, 2)
}
