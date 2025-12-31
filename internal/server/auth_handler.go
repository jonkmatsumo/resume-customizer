// Package server provides the HTTP REST API for the resume customizer.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/types"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	userService *UserService
	jwtService  *JWTService
	validator   *validator.Validate
}

// NewAuthHandler creates a new AuthHandler with the given dependencies.
func NewAuthHandler(userService *UserService, jwtService *JWTService) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		jwtService:  jwtService,
		validator:   validator.New(),
	}
}

// Register handles user registration requests.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req types.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		validationErrors := extractValidationErrors(err)
		http.Error(w, validationErrors, http.StatusBadRequest)
		return
	}

	user, err := h.userService.Register(r.Context(), &req)
	if err != nil {
		status := HTTPStatus(err)
		http.Error(w, err.Error(), status)
		return
	}

	token, err := h.jwtService.GenerateToken(user.ID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := types.LoginResponse{
		User:  user,
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log error but response already sent
		return
	}
}

// Login handles user login requests.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req types.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		validationErrors := extractValidationErrors(err)
		http.Error(w, validationErrors, http.StatusBadRequest)
		return
	}

	user, err := h.userService.Login(r.Context(), &req)
	if err != nil {
		status := HTTPStatus(err)
		http.Error(w, err.Error(), status)
		return
	}

	token, err := h.jwtService.GenerateToken(user.ID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := types.LoginResponse{
		User:  user,
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log error but response already sent
		return
	}
}

// UpdatePasswordWithUserID handles password update requests with an explicit user ID.
func (h *AuthHandler) UpdatePasswordWithUserID(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	var req types.UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		validationErrors := extractValidationErrors(err)
		http.Error(w, validationErrors, http.StatusBadRequest)
		return
	}

	if err := h.userService.UpdatePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		status := HTTPStatus(err)
		http.Error(w, err.Error(), status)
		return
	}

	response := map[string]string{
		"message": "Password updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log error but response already sent
		return
	}
}

// extractValidationErrors extracts validation error messages from validator errors.
func extractValidationErrors(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		if len(validationErrors) > 0 {
			// Return first validation error for simplicity
			ve := validationErrors[0]
			return fmt.Sprintf("validation error: %s - %s", ve.Field(), ve.Tag())
		}
	}
	return "validation error: invalid request"
}
