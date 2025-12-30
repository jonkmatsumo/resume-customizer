// Package server provides the HTTP REST API for the resume customizer.
package server

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// ErrEmailAlreadyExists indicates email is already registered
type ErrEmailAlreadyExists struct {
	Email string
}

func (e *ErrEmailAlreadyExists) Error() string {
	return fmt.Sprintf("email already registered: %s", e.Email)
}

// ErrInvalidCredentials indicates invalid login credentials
type ErrInvalidCredentials struct{}

func (e *ErrInvalidCredentials) Error() string {
	return "invalid email or password"
}

// ErrUserNotFound indicates user was not found
type ErrUserNotFound struct {
	UserID uuid.UUID
}

func (e *ErrUserNotFound) Error() string {
	return fmt.Sprintf("user not found: %s", e.UserID)
}

// ErrPasswordMismatch indicates current password is incorrect
type ErrPasswordMismatch struct{}

func (e *ErrPasswordMismatch) Error() string {
	return "current password is incorrect"
}

// ErrValidation indicates request validation failure
type ErrValidation struct {
	Field   string
	Message string
}

func (e *ErrValidation) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// HTTPStatus returns the appropriate HTTP status code for an error
func HTTPStatus(err error) int {
	switch err.(type) {
	case *ErrEmailAlreadyExists:
		return http.StatusConflict
	case *ErrInvalidCredentials, *ErrPasswordMismatch:
		return http.StatusUnauthorized
	case *ErrUserNotFound:
		return http.StatusNotFound
	case *ErrValidation:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
