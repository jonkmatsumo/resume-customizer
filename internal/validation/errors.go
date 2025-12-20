// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import "fmt"

// Error represents a general validation error
type Error struct {
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("validation error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// CompilationError represents a LaTeX compilation failure
type CompilationError struct {
	Message   string
	LogOutput string
	Cause     error
}

func (e *CompilationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("LaTeX compilation error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("LaTeX compilation error: %s", e.Message)
}

func (e *CompilationError) Unwrap() error {
	return e.Cause
}

// FileReadError represents an error reading a file
type FileReadError struct {
	Message string
	Cause   error
}

func (e *FileReadError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("file read error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("file read error: %s", e.Message)
}

func (e *FileReadError) Unwrap() error {
	return e.Cause
}
