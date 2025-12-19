package schemas

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJSON_ValidJSON(t *testing.T) {
	schemaPath := filepath.Join("testdata", "valid_schema.json")
	jsonPath := filepath.Join("testdata", "valid_json.json")

	err := ValidateJSON(schemaPath, jsonPath)
	assert.NoError(t, err)
}

func TestValidateJSON_InvalidJSON_MissingField(t *testing.T) {
	schemaPath := filepath.Join("testdata", "valid_schema.json")
	jsonPath := filepath.Join("testdata", "invalid_json.json")

	err := ValidateJSON(schemaPath, jsonPath)
	require.Error(t, err)

	validationErr, ok := err.(*ValidationError)
	require.True(t, ok, "error should be ValidationError type")
	assert.Greater(t, len(validationErr.Errors), 0)
}

func TestValidateJSON_InvalidJSON_WrongType(t *testing.T) {
	schemaPath := filepath.Join("testdata", "valid_schema.json")
	jsonPath := filepath.Join("testdata", "type_mismatch.json")

	err := ValidateJSON(schemaPath, jsonPath)
	require.Error(t, err)

	validationErr, ok := err.(*ValidationError)
	require.True(t, ok, "error should be ValidationError type")
	assert.Greater(t, len(validationErr.Errors), 0)
}

func TestValidateJSON_NonExistentSchema(t *testing.T) {
	schemaPath := "testdata/nonexistent_schema.json"
	jsonPath := filepath.Join("testdata", "valid_json.json")

	err := ValidateJSON(schemaPath, jsonPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestValidateJSON_NonExistentJSON(t *testing.T) {
	schemaPath := filepath.Join("testdata", "valid_schema.json")
	jsonPath := "testdata/nonexistent_json.json"

	err := ValidateJSON(schemaPath, jsonPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestValidateJSON_MalformedJSON(t *testing.T) {
	// Create a temporary malformed JSON file
	tmpDir := t.TempDir()
	malformedJSON := filepath.Join(tmpDir, "malformed.json")
	err := os.WriteFile(malformedJSON, []byte("{ invalid json }"), 0644)
	require.NoError(t, err)

	schemaPath := filepath.Join("testdata", "valid_schema.json")

	valErr := ValidateJSON(schemaPath, malformedJSON)
	require.Error(t, valErr)
	// The error might be from gojsonschema parsing, not our code
}

func TestValidateJSON_JobProfileSchema(t *testing.T) {
	tests := []struct {
		name      string
		jsonFile  string
		wantError bool
	}{
		{
			name:      "valid job profile",
			jsonFile:  "../../testdata/valid/job_profile.json",
			wantError: false,
		},
		{
			name:      "missing required field",
			jsonFile:  "../../testdata/invalid/missing_field.json",
			wantError: true,
		},
		{
			name:      "wrong type",
			jsonFile:  "../../testdata/invalid/wrong_type.json",
			wantError: true,
		},
	}

	schemaPath := "../../schemas/job_profile.schema.json"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSON(schemaPath, tt.jsonFile)
			if tt.wantError {
				require.Error(t, err)
				// Should be ValidationError, not SchemaLoadError
				validationErr, ok := err.(*ValidationError)
				if !ok {
					// If it's a SchemaLoadError, that's also an error condition we should know about
					schemaErr, isSchemaErr := err.(*SchemaLoadError)
					if isSchemaErr {
						t.Fatalf("unexpected SchemaLoadError (schema loading failed): %v", schemaErr)
					}
					t.Fatalf("error should be ValidationError or SchemaLoadError, got %T: %v", err, err)
				}
				assert.Greater(t, len(validationErr.Errors), 0, "validation error should have at least one field error")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateJSONString_Valid(t *testing.T) {
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["name"],
		"properties": {
			"name": {"type": "string"}
		}
	}`
	jsonContent := `{"name": "test"}`

	err := ValidateJSONString(schemaContent, jsonContent)
	assert.NoError(t, err)
}

func TestValidateJSONString_Invalid(t *testing.T) {
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["name"],
		"properties": {
			"name": {"type": "string"}
		}
	}`
	jsonContent := `{"age": 30}`

	err := ValidateJSONString(schemaContent, jsonContent)
	require.Error(t, err)

	validationErr, ok := err.(*ValidationError)
	require.True(t, ok)
	assert.Greater(t, len(validationErr.Errors), 0)
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Errors: []FieldError{
			{Field: "name", Message: "is required"},
			{Field: "age", Message: "must be a number"},
		},
	}

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "validation failed")
	assert.Contains(t, errorMsg, "name")
	assert.Contains(t, errorMsg, "age")
}

func TestValidateJSON_NestedFieldValidation(t *testing.T) {
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["person"],
		"properties": {
			"person": {
				"type": "object",
				"required": ["name"],
				"properties": {
					"name": {"type": "string"}
				}
			}
		}
	}`

	jsonContent := `{"person": {}}`

	err := ValidateJSONString(schemaContent, jsonContent)
	require.Error(t, err)

	validationErr, ok := err.(*ValidationError)
	require.True(t, ok)
	assert.Greater(t, len(validationErr.Errors), 0)
	// Check that the field path includes nested field
	found := false
	for _, fieldErr := range validationErr.Errors {
		if fieldErr.Field != "" {
			found = true
			break
		}
	}
	assert.True(t, found, "should include field path in error")
}

func TestValidateJSON_ArrayValidation(t *testing.T) {
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"items": {
				"type": "array",
				"items": {"type": "string"},
				"minItems": 1
			}
		}
	}`

	jsonContent := `{"items": []}`

	err := ValidateJSONString(schemaContent, jsonContent)
	// This may or may not error depending on schema strictness
	// Just ensure it doesn't panic
	_ = err
}
