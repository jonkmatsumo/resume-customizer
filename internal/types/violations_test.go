// Package types provides type definitions for structured data used throughout the resume-customizer system.
package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViolation_JSONMarshal_WithNewFields(t *testing.T) {
	bulletID := "bullet_123"
	storyID := "story_456"
	bulletText := "Built system with $1M budget"
	lineNum := 42
	charCount := 100

	violation := Violation{
		Type:       "line_too_long",
		Severity:   "error",
		Details:    "Line exceeds maximum character count",
		LineNumber: &lineNum,
		CharCount:  &charCount,
		BulletID:   &bulletID,
		StoryID:    &storyID,
		BulletText: &bulletText,
	}

	data, err := json.Marshal(violation)
	require.NoError(t, err)

	var unmarshaled Violation
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, violation.Type, unmarshaled.Type)
	assert.Equal(t, violation.Severity, unmarshaled.Severity)
	assert.Equal(t, violation.Details, unmarshaled.Details)
	assert.Equal(t, violation.LineNumber, unmarshaled.LineNumber)
	assert.Equal(t, violation.CharCount, unmarshaled.CharCount)
	assert.Equal(t, violation.BulletID, unmarshaled.BulletID)
	assert.Equal(t, violation.StoryID, unmarshaled.StoryID)
	assert.Equal(t, violation.BulletText, unmarshaled.BulletText)
}

func TestViolation_JSONMarshal_BackwardCompatibility(t *testing.T) {
	// Test that old format (without new fields) still works
	lineNum := 42
	charCount := 100

	violation := Violation{
		Type:       "line_too_long",
		Severity:   "error",
		Details:    "Line exceeds maximum character count",
		LineNumber: &lineNum,
		CharCount:  &charCount,
		// BulletID, StoryID, BulletText are nil (old format)
	}

	data, err := json.Marshal(violation)
	require.NoError(t, err)

	var unmarshaled Violation
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, violation.Type, unmarshaled.Type)
	assert.Equal(t, violation.Severity, unmarshaled.Severity)
	assert.Equal(t, violation.Details, unmarshaled.Details)
	assert.Equal(t, violation.LineNumber, unmarshaled.LineNumber)
	assert.Equal(t, violation.CharCount, unmarshaled.CharCount)
	assert.Nil(t, unmarshaled.BulletID)
	assert.Nil(t, unmarshaled.StoryID)
	assert.Nil(t, unmarshaled.BulletText)
}

func TestViolation_JSONUnmarshal_OldFormat(t *testing.T) {
	// Test unmarshaling JSON without new fields (backward compatibility)
	jsonStr := `{
		"type": "line_too_long",
		"severity": "error",
		"details": "Line exceeds maximum character count",
		"line_number": 42,
		"char_count": 100
	}`

	var violation Violation
	err := json.Unmarshal([]byte(jsonStr), &violation)
	require.NoError(t, err)

	assert.Equal(t, "line_too_long", violation.Type)
	assert.Equal(t, "error", violation.Severity)
	assert.Equal(t, "Line exceeds maximum character count", violation.Details)
	assert.NotNil(t, violation.LineNumber)
	assert.Equal(t, 42, *violation.LineNumber)
	assert.NotNil(t, violation.CharCount)
	assert.Equal(t, 100, *violation.CharCount)
	assert.Nil(t, violation.BulletID)
	assert.Nil(t, violation.StoryID)
	assert.Nil(t, violation.BulletText)
}

func TestViolation_JSONUnmarshal_NewFormat(t *testing.T) {
	// Test unmarshaling JSON with new fields
	jsonStr := `{
		"type": "line_too_long",
		"severity": "error",
		"details": "Line exceeds maximum character count",
		"line_number": 42,
		"char_count": 100,
		"bullet_id": "bullet_123",
		"story_id": "story_456",
		"bullet_text": "Built system with $1M budget"
	}`

	var violation Violation
	err := json.Unmarshal([]byte(jsonStr), &violation)
	require.NoError(t, err)

	assert.Equal(t, "line_too_long", violation.Type)
	assert.NotNil(t, violation.BulletID)
	assert.Equal(t, "bullet_123", *violation.BulletID)
	assert.NotNil(t, violation.StoryID)
	assert.Equal(t, "story_456", *violation.StoryID)
	assert.NotNil(t, violation.BulletText)
	assert.Equal(t, "Built system with $1M budget", *violation.BulletText)
}

func TestViolation_NilPointerHandling(t *testing.T) {
	// Test that nil pointers are handled correctly
	violation := Violation{
		Type:     "page_overflow",
		Severity: "error",
		Details:  "Resume has 2 pages, maximum allowed is 1",
		// All optional fields are nil
	}

	data, err := json.Marshal(violation)
	require.NoError(t, err)

	// Should not include nil fields in JSON (due to omitempty)
	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "bullet_id")
	assert.NotContains(t, jsonStr, "story_id")
	assert.NotContains(t, jsonStr, "bullet_text")
	assert.NotContains(t, jsonStr, "line_number")
	assert.NotContains(t, jsonStr, "char_count")
}
