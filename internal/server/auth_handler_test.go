package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/jonathan/resume-customizer/internal/server/middleware"
	"github.com/stretchr/testify/assert"
)

// setupTestAuthHandler creates an AuthHandler with test services.
func setupTestAuthHandler(_ *testing.T) *AuthHandler {
	passwordConfig := &config.PasswordConfig{
		BcryptCost: 10, // Lower cost for faster tests
		Pepper:     "",
	}
	jwtConfig := &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing-minimum-32-bytes",
		ExpirationHours: 24,
	}

	userSvc := NewUserService(nil, passwordConfig) // nil DB for unit tests - will fail on actual service calls
	jwtSvc := NewJWTService(jwtConfig)
	return NewAuthHandler(userSvc, jwtSvc)
}

// setUserIDInContext sets the user ID in the request context for testing.
func setUserIDInContext(r *http.Request, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey(), userID)
	return r.WithContext(ctx)
}

func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	handler := setupTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestAuthHandler_Register_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		reqBody     map[string]string
		description string
	}{
		{
			name:        "missing name",
			reqBody:     map[string]string{"email": "test@example.com", "password": "password123"},
			description: "should return 400 when name is missing",
		},
		{
			name:        "empty name",
			reqBody:     map[string]string{"name": "", "email": "test@example.com", "password": "password123"},
			description: "should return 400 when name is empty",
		},
		{
			name:        "invalid email",
			reqBody:     map[string]string{"name": "Test User", "email": "invalid-email", "password": "password123"},
			description: "should return 400 when email is invalid",
		},
		{
			name:        "missing email",
			reqBody:     map[string]string{"name": "Test User", "password": "password123"},
			description: "should return 400 when email is missing",
		},
		{
			name:        "password too short",
			reqBody:     map[string]string{"name": "Test User", "email": "test@example.com", "password": "short"},
			description: "should return 400 when password is too short",
		},
		{
			name:        "missing password",
			reqBody:     map[string]string{"name": "Test User", "email": "test@example.com"},
			description: "should return 400 when password is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupTestAuthHandler(t)

			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Register(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code, tt.description)
			assert.Contains(t, w.Body.String(), "validation error", tt.description)
		})
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	handler := setupTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestAuthHandler_Login_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		reqBody     map[string]string
		description string
	}{
		{
			name:        "missing email",
			reqBody:     map[string]string{"password": "password123"},
			description: "should return 400 when email is missing",
		},
		{
			name:        "invalid email format",
			reqBody:     map[string]string{"email": "invalid-email", "password": "password123"},
			description: "should return 400 when email format is invalid",
		},
		{
			name:        "missing password",
			reqBody:     map[string]string{"email": "test@example.com"},
			description: "should return 400 when password is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupTestAuthHandler(t)

			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Login(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code, tt.description)
			assert.Contains(t, w.Body.String(), "validation error", tt.description)
		})
	}
}

func TestAuthHandler_UpdatePassword_MissingUserID(t *testing.T) {
	handler := setupTestAuthHandler(t)

	reqBody := map[string]string{
		"current_password": "oldpassword",
		"new_password":     "newpassword123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/users/me/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No user ID in context
	w := httptest.NewRecorder()

	handler.UpdatePassword(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestAuthHandler_UpdatePassword_InvalidJSON(t *testing.T) {
	handler := setupTestAuthHandler(t)
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodPut, "/users/me/password", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req = setUserIDInContext(req, userID)
	w := httptest.NewRecorder()

	handler.UpdatePassword(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestAuthHandler_UpdatePassword_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		reqBody     map[string]string
		description string
	}{
		{
			name:        "missing current password",
			reqBody:     map[string]string{"new_password": "newpassword123"},
			description: "should return 400 when current password is missing",
		},
		{
			name:        "missing new password",
			reqBody:     map[string]string{"current_password": "oldpassword"},
			description: "should return 400 when new password is missing",
		},
		{
			name:        "new password too short",
			reqBody:     map[string]string{"current_password": "oldpassword", "new_password": "short"},
			description: "should return 400 when new password is too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupTestAuthHandler(t)
			userID := uuid.New()

			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPut, "/users/me/password", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = setUserIDInContext(req, userID)
			w := httptest.NewRecorder()

			handler.UpdatePassword(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code, tt.description)
			assert.Contains(t, w.Body.String(), "validation error", tt.description)
		})
	}
}
