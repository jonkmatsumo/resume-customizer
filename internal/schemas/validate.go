// Package schemas provides JSON Schema validation functionality for structured data artifacts.
package schemas

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// ResolveSchemaPath attempts to find a schema file by trying multiple common path resolutions.
// It tries paths relative to the current working directory, then paths relative to likely repo root locations.
// Returns the first path that exists, or empty string if none found.
// This is useful when CLI commands may run from different working directory contexts (e.g., tests).
func ResolveSchemaPath(relativePath string) string {
	// Try paths in order:
	// 1. Relative to current working directory
	// 2. One level up (../schemas/...)
	// 3. Two levels up (../../schemas/...)
	candidates := []string{
		relativePath,
		filepath.Join("..", relativePath),
		filepath.Join("..", "..", relativePath),
	}

	for _, candidate := range candidates {
		if absPath, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	return ""
}

// ValidationError represents a schema validation error with field paths
type ValidationError struct {
	Errors []FieldError
}

// FieldError represents a single validation error at a specific field
type FieldError struct {
	Field   string
	Message string
}

// SchemaLoadError represents errors loading or parsing the schema itself
type SchemaLoadError struct {
	Path    string
	Message string
	Cause   error
}

func (e *SchemaLoadError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("failed to load schema %s: %s: %v", e.Path, e.Message, e.Cause)
	}
	return fmt.Sprintf("failed to load schema %s: %s", e.Path, e.Message)
}

func (e *SchemaLoadError) Unwrap() error {
	return e.Cause
}

func (ve *ValidationError) Error() string {
	var sb strings.Builder
	sb.WriteString("validation failed:\n")
	for i, err := range ve.Errors {
		sb.WriteString(fmt.Sprintf("  %d. %s: %s\n", i+1, err.Field, err.Message))
	}
	return sb.String()
}

// ValidateJSON validates a JSON file against a JSON Schema file
func ValidateJSON(schemaPath, jsonPath string) error {
	// Resolve absolute paths to handle relative paths correctly
	schemaAbsPath, err := filepath.Abs(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to resolve schema path: %w", err)
	}

	jsonAbsPath, err := filepath.Abs(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to resolve JSON path: %w", err)
	}

	// Check if files exist
	if _, err := os.Stat(schemaAbsPath); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s", schemaAbsPath)
	}

	if _, err := os.Stat(jsonAbsPath); os.IsNotExist(err) {
		return fmt.Errorf("JSON file not found: %s", jsonAbsPath)
	}

	// Load schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaAbsPath)

	// Load JSON document
	documentLoader := gojsonschema.NewReferenceLoader("file://" + jsonAbsPath)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		// Check if it's a schema loading error (e.g., can't resolve $ref, invalid schema syntax)
		// vs a document loading error
		return &SchemaLoadError{
			Path:    schemaAbsPath,
			Message: "schema validation failed during load",
			Cause:   err,
		}
	}

	if result.Valid() {
		return nil
	}

	// Build structured error
	validationErr := &ValidationError{
		Errors: make([]FieldError, 0, len(result.Errors())),
	}

	for _, desc := range result.Errors() {
		field := desc.Field()
		if field == "" {
			field = "(root)"
		}
		validationErr.Errors = append(validationErr.Errors, FieldError{
			Field:   field,
			Message: desc.Description(),
		})
	}

	return validationErr
}

// ValidateJSONString validates JSON string content against schema string content
func ValidateJSONString(schemaContent, jsonContent string) error {
	schemaLoader := gojsonschema.NewStringLoader(schemaContent)
	documentLoader := gojsonschema.NewStringLoader(jsonContent)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return &SchemaLoadError{
			Path:    "(string schema)",
			Message: "schema validation failed during load",
			Cause:   err,
		}
	}

	if result.Valid() {
		return nil
	}

	// Build structured error
	validationErr := &ValidationError{
		Errors: make([]FieldError, 0, len(result.Errors())),
	}

	for _, desc := range result.Errors() {
		field := desc.Field()
		if field == "" {
			field = "(root)"
		}
		validationErr.Errors = append(validationErr.Errors, FieldError{
			Field:   field,
			Message: desc.Description(),
		})
	}

	return validationErr
}
