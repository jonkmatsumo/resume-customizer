// Package experience provides functionality to load and normalize experience bank files.
package experience

import "fmt"

// LoadError represents an error during file I/O or JSON parsing
type LoadError struct {
	Message string
	Cause   error
}

func (e *LoadError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("load error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("load error: %s", e.Message)
}

func (e *LoadError) Unwrap() error {
	return e.Cause
}

// NormalizationError represents an error during normalization
type NormalizationError struct {
	Message string
	Cause   error
}

func (e *NormalizationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("normalization error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("normalization error: %s", e.Message)
}

func (e *NormalizationError) Unwrap() error {
	return e.Cause
}

