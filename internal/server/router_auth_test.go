package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServerForRouter creates a test server instance for router testing
func setupTestServerForRouter(t *testing.T) (*Server, *db.DB) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://resume:resume_dev@localhost:5432/resume_customizer?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping router test: failed to connect to DB: %v", err)
	}

	// Set JWT_SECRET environment variable for server creation
	originalJWTSecret := os.Getenv("JWT_SECRET")
	os.Setenv("JWT_SECRET", "test-secret-key-for-jwt-signing-minimum-32-bytes")
	defer func() {
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	// Create server with test config
	server, err := New(Config{
		Port:        8080,
		DatabaseURL: dbURL,
		APIKey:      "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	return server, database
}

func TestPublicRoutes_Register(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	reqBody := types.CreateUserRequest{
		Name:     "Router Test User",
		Email:    "router-test-register@example.com",
		Password: "testpassword123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Use the server's handler which includes all middleware
	handler := server.httpServer.Handler
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var response types.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response.User)
	assert.NotEmpty(t, response.Token)

	// Cleanup
	if response.User != nil {
		database.DeleteUser(context.Background(), response.User.ID)
	}
}

func TestPublicRoutes_Login(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	// First register a user
	registerReq := types.CreateUserRequest{
		Name:     "Router Test Login",
		Email:    "router-test-login@example.com",
		Password: "testpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(registerW, registerHTTPReq)

	var registerResponse types.LoginResponse
	err := json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	require.NoError(t, err)
	require.NotNil(t, registerResponse.User)
	userID := registerResponse.User.ID

	// Now test login
	loginReq := types.LoginRequest{
		Email:    "router-test-login@example.com",
		Password: "testpassword123",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBody))
	loginHTTPReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()

	handler := server.httpServer.Handler
	handler.ServeHTTP(loginW, loginHTTPReq)

	assert.Equal(t, http.StatusOK, loginW.Code)
	var loginResponse types.LoginResponse
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	require.NotNil(t, loginResponse.User)
	assert.Equal(t, userID, loginResponse.User.ID)
	assert.NotEmpty(t, loginResponse.Token)

	// Cleanup
	database.DeleteUser(context.Background(), userID)
}

func TestProtectedRoute_UpdatePassword_WithValidToken(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	// Register and login to get a token
	registerReq := types.CreateUserRequest{
		Name:     "Router Test Protected",
		Email:    "router-test-protected@example.com",
		Password: "oldpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(registerW, registerHTTPReq)

	var registerResponse types.LoginResponse
	json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	userID := registerResponse.User.ID
	token := registerResponse.Token

	// Update password with valid token
	updateReq := types.UpdatePasswordRequest{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword456",
	}
	updateBody, _ := json.Marshal(updateReq)
	updateHTTPReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/users/%s/password", userID), bytes.NewReader(updateBody))
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("Authorization", "Bearer "+token)
	updateW := httptest.NewRecorder()

	handler := server.httpServer.Handler
	handler.ServeHTTP(updateW, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateW.Code)
	var updateResponse map[string]string
	err := json.Unmarshal(updateW.Body.Bytes(), &updateResponse)
	require.NoError(t, err)
	assert.Equal(t, "Password updated successfully", updateResponse["message"])

	// Cleanup
	database.DeleteUser(context.Background(), userID)
}

func TestProtectedRoute_UpdatePassword_WithoutToken(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	// Use a dummy UUID for unauthorized tests (request will fail at auth middleware)
	dummyUserID := uuid.New()

	updateReq := types.UpdatePasswordRequest{
		CurrentPassword: "oldpassword",
		NewPassword:     "newpassword123",
	}
	updateBody, _ := json.Marshal(updateReq)
	updateHTTPReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/users/%s/password", dummyUserID), bytes.NewReader(updateBody))
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	// No Authorization header
	updateW := httptest.NewRecorder()

	handler := server.httpServer.Handler
	handler.ServeHTTP(updateW, updateHTTPReq)

	assert.Equal(t, http.StatusUnauthorized, updateW.Code)
	assert.Contains(t, updateW.Body.String(), "Unauthorized")
}

func TestProtectedRoute_UpdatePassword_WithInvalidToken(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	// Use a dummy UUID for unauthorized tests (request will fail at auth middleware)
	dummyUserID := uuid.New()

	updateReq := types.UpdatePasswordRequest{
		CurrentPassword: "oldpassword",
		NewPassword:     "newpassword123",
	}
	updateBody, _ := json.Marshal(updateReq)

	tests := []struct {
		name        string
		token       string
		description string
	}{
		{
			name:        "malformed token",
			token:       "invalid.token.here",
			description: "should return 401 for malformed token",
		},
		{
			name:        "wrong signature",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzNCIsImV4cCI6OTk5OTk5OTk5OX0.wrong-signature",
			description: "should return 401 for wrong signature",
		},
		{
			name:        "empty token",
			token:       "",
			description: "should return 401 for empty token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateHTTPReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/users/%s/password", dummyUserID), bytes.NewReader(updateBody))
			updateHTTPReq.Header.Set("Content-Type", "application/json")
			updateHTTPReq.Header.Set("Authorization", "Bearer "+tt.token)
			updateW := httptest.NewRecorder()

			handler := server.httpServer.Handler
			handler.ServeHTTP(updateW, updateHTTPReq)

			assert.Equal(t, http.StatusUnauthorized, updateW.Code, tt.description)
			assert.Contains(t, updateW.Body.String(), "Unauthorized", tt.description)
		})
	}
}

func TestProtectedRoute_UpdatePassword_WithWrongBearerFormat(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	// Register a user first to get a user ID
	registerReq := types.CreateUserRequest{
		Name:     "Bearer Format Test",
		Email:    fmt.Sprintf("bearer-format-test-%d@example.com", time.Now().UnixNano()),
		Password: "testpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(registerW, registerHTTPReq)

	var registerResponse types.LoginResponse
	err := json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	require.NoError(t, err)
	require.NotNil(t, registerResponse.User)
	userID := registerResponse.User.ID

	updateReq := types.UpdatePasswordRequest{
		CurrentPassword: "oldpassword",
		NewPassword:     "newpassword123",
	}
	updateBody, _ := json.Marshal(updateReq)

	tests := []struct {
		name        string
		authHeader  string
		description string
	}{
		{
			name:        "missing Bearer prefix",
			authHeader:  "some-token-here",
			description: "should return 401 when Bearer prefix is missing",
		},
		{
			name:        "lowercase bearer",
			authHeader:  "bearer some-token",
			description: "should accept lowercase bearer (case-insensitive)",
		},
		{
			name:        "extra spaces",
			authHeader:  "Bearer  some-token",
			description: "should handle extra spaces",
		},
		{
			name:        "empty token after Bearer",
			authHeader:  "Bearer ",
			description: "should return 401 for empty token after Bearer",
		},
		{
			name:        "missing Authorization header",
			authHeader:  "",
			description: "should return 401 when Authorization header is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateHTTPReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/users/%s/password", userID), bytes.NewReader(updateBody))
			updateHTTPReq.Header.Set("Content-Type", "application/json")
			if tt.authHeader != "" {
				updateHTTPReq.Header.Set("Authorization", tt.authHeader)
			}
			updateW := httptest.NewRecorder()

			handler := server.httpServer.Handler
			handler.ServeHTTP(updateW, updateHTTPReq)

			// Note: lowercase "bearer" should work (case-insensitive), but others should fail
			if tt.name == "lowercase bearer" {
				// This should still fail because token is invalid, but format is accepted
				assert.Equal(t, http.StatusUnauthorized, updateW.Code, tt.description)
			} else {
				assert.Equal(t, http.StatusUnauthorized, updateW.Code, tt.description)
				assert.Contains(t, updateW.Body.String(), "Unauthorized", tt.description)
			}
		})
	}

	// Cleanup
	database.DeleteUser(context.Background(), userID)
}

func TestCORS_AuthorizationHeader(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	// Test preflight OPTIONS request
	req := httptest.NewRequest(http.MethodOptions, "/v1/auth/register", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	w := httptest.NewRecorder()

	handler := server.httpServer.Handler
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
}
