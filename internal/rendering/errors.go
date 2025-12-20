// Package rendering provides functionality to render LaTeX resumes from templates.
package rendering

import "fmt"

// TemplateError represents an error parsing or executing a LaTeX template
type TemplateError struct {
	Message string
	Cause   error
}

func (e *TemplateError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("template error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("template error: %s", e.Message)
}

func (e *TemplateError) Unwrap() error {
	return e.Cause
}

// RenderError represents a general rendering failure
type RenderError struct {
	Message string
	Cause   error
}

func (e *RenderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("render error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("render error: %s", e.Message)
}

func (e *RenderError) Unwrap() error {
	return e.Cause
}
