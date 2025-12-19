package parsing

import "fmt"

// APICallError represents an error from the Gemini API
type APICallError struct {
	Message string
	Cause   error
}

func (e *APICallError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("API call failed: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("API call failed: %s", e.Message)
}

func (e *APICallError) Unwrap() error {
	return e.Cause
}

// ParseError represents an error parsing the API response
type ParseError struct {
	Message string
	Cause   error
}

func (e *ParseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("parse error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("parse error: %s", e.Message)
}

func (e *ParseError) Unwrap() error {
	return e.Cause
}

// ValidationError represents an error during post-processing validation
type ValidationError struct {
	Message string
	Field   string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error in %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}
