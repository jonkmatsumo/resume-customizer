// Package types provides type definitions for structured data used throughout the resume-customizer system.
package types

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// CreateUserRequest represents the request to create a new user with password authentication.
type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=1"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Phone    string `json:"phone,omitempty"`
}

// LoginRequest represents the login request.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// User represents a user profile for API responses (avoids import cycle with db package).
type User struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone,omitempty"`
	PasswordSet bool      `json:"password_set"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LoginResponse represents the login/register response with user data and authentication token.
type LoginResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

// UpdatePasswordRequest represents a password update request.
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// Validate validates the CreateUserRequest using the validator.
func (r *CreateUserRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

// Validate validates the LoginRequest using the validator.
func (r *LoginRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

// Validate validates the UpdatePasswordRequest using the validator.
func (r *UpdatePasswordRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}
