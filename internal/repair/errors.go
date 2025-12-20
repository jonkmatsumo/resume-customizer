// Package repair provides functionality to automatically fix violations in LaTeX resumes.
package repair

import "fmt"

// Error represents a general repair error
type Error struct {
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("repair error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("repair error: %s", e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// ProposeError represents an error during repair proposal (LLM call failure)
type ProposeError struct {
	Message string
	Cause   error
}

func (e *ProposeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("repair proposal error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("repair proposal error: %s", e.Message)
}

func (e *ProposeError) Unwrap() error {
	return e.Cause
}

// ApplyError represents an error during repair application (invalid action, missing IDs, etc.)
type ApplyError struct {
	Message string
	Cause   error
}

func (e *ApplyError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("repair apply error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("repair apply error: %s", e.Message)
}

func (e *ApplyError) Unwrap() error {
	return e.Cause
}
