// Package selection provides functionality to select optimal stories and bullets for a resume plan.
package selection

import "fmt"

// Error represents an error that occurs during story/bullet selection
type Error struct {
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}
