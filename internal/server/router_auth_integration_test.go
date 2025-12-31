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
	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServerForRouterIntegration creates a test server instance for integration testing
func setupTestServerForRouterIntegration(t *testing.T) (*Server, *db.DB) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://resume:resume_dev@localhost:5432/resume_customizer?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping integration test: failed to connect to DB: %v", err)
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

func TestIntegration_Register_EndToEnd(t *testing.T) {
	server, database := setupTestServerForRouterIntegration(t)
	defer database.Close()

	reqBody := types.CreateUserRequest{
		Name:     "Integration Router Test",
		Email:    "integration-router-test@example.com",
		Password: "testpassword123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Use unique IP address to avoid rate limiting interference between tests
	req.RemoteAddr = "192.0.2.20:1234"
	w := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response types.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response.User)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, reqBody.Name, response.User.Name)
	assert.Equal(t, reqBody.Email, response.User.Email)

	// Verify user exists in database
	dbUser, err := database.GetUserByEmail(context.Background(), reqBody.Email)
	require.NoError(t, err)
	require.NotNil(t, dbUser)
	assert.Equal(t, response.User.ID, dbUser.ID)
	assert.NotEmpty(t, dbUser.PasswordHash)

	// Verify token is valid
	jwtConfig := &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing-minimum-32-bytes",
		ExpirationHours: 24,
	}
	jwtSvc := NewJWTService(jwtConfig)
	claims, err := jwtSvc.ValidateToken(response.Token)
	require.NoError(t, err)
	assert.Equal(t, response.User.ID, claims.GetUserID())

	// Cleanup
	database.DeleteUser(context.Background(), response.User.ID)
}

func TestIntegration_Login_EndToEnd(t *testing.T) {
	server, database := setupTestServerForRouterIntegration(t)
	defer database.Close()

	// Register a user first
	registerReq := types.CreateUserRequest{
		Name:     "Integration Login Test",
		Email:    "integration-login-test@example.com",
		Password: "loginpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	// Use unique IP address to avoid rate limiting interference between tests
	registerHTTPReq.RemoteAddr = "192.0.2.21:1234"
	registerW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(registerW, registerHTTPReq)

	var registerResponse types.LoginResponse
	json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	userID := registerResponse.User.ID

	// Now login
	loginReq := types.LoginRequest{
		Email:    "integration-login-test@example.com",
		Password: "loginpassword123",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHTTPReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginHTTPReq.Header.Set("Content-Type", "application/json")
	// Use unique IP address to avoid rate limiting interference between tests
	loginHTTPReq.RemoteAddr = "192.0.2.22:1234"
	loginW := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(loginW, loginHTTPReq)

	assert.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse types.LoginResponse
	err := json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	assert.Equal(t, userID, loginResponse.User.ID)
	assert.NotEmpty(t, loginResponse.Token)

	// Verify token is valid
	jwtConfig := &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing-minimum-32-bytes",
		ExpirationHours: 24,
	}
	jwtSvc := NewJWTService(jwtConfig)
	claims, err := jwtSvc.ValidateToken(loginResponse.Token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.GetUserID())

	// Cleanup
	database.DeleteUser(context.Background(), userID)
}

func TestIntegration_UpdatePassword_EndToEnd(t *testing.T) {
	server, database := setupTestServerForRouterIntegration(t)
	defer database.Close()

	// Register and login a user
	registerReq := types.CreateUserRequest{
		Name:     "Integration Update Password",
		Email:    "integration-update-password@example.com",
		Password: "oldpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	// Use unique IP address to avoid rate limiting interference between tests
	registerHTTPReq.RemoteAddr = "192.0.2.21:1234"
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
	// Use unique IP address to avoid rate limiting interference between tests
	updateHTTPReq.RemoteAddr = "192.0.2.23:1234"
	updateW := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(updateW, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateW.Code)

	var updateResponse map[string]string
	err := json.Unmarshal(updateW.Body.Bytes(), &updateResponse)
	require.NoError(t, err)
	assert.Equal(t, "Password updated successfully", updateResponse["message"])

	// Verify old password no longer works
	loginReqOld := types.LoginRequest{
		Email:    "integration-update-password@example.com",
		Password: "oldpassword123",
	}
	loginBodyOld, _ := json.Marshal(loginReqOld)
	loginHTTPReqOld := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBodyOld))
	loginHTTPReqOld.Header.Set("Content-Type", "application/json")
	// Use unique IP address to avoid rate limiting interference between tests
	loginHTTPReqOld.RemoteAddr = "192.0.2.28:1234"
	loginWOld := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(loginWOld, loginHTTPReqOld)
	assert.Equal(t, http.StatusUnauthorized, loginWOld.Code)

	// Verify new password works
	loginReqNew := types.LoginRequest{
		Email:    "integration-update-password@example.com",
		Password: "newpassword456",
	}
	loginBodyNew, _ := json.Marshal(loginReqNew)
	loginHTTPReqNew := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBodyNew))
	loginHTTPReqNew.Header.Set("Content-Type", "application/json")
	// Use unique IP address to avoid rate limiting interference between tests
	loginHTTPReqNew.RemoteAddr = "192.0.2.29:1234"
	loginWNew := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(loginWNew, loginHTTPReqNew)
	assert.Equal(t, http.StatusOK, loginWNew.Code)

	// Cleanup
	database.DeleteUser(context.Background(), userID)
}

func TestIntegration_UpdatePassword_Unauthorized(t *testing.T) {
	server, database := setupTestServerForRouterIntegration(t)
	defer database.Close()

	// Use a dummy UUID for unauthorized tests (request will fail at auth middleware)
	dummyUserID := uuid.New()

	updateReq := types.UpdatePasswordRequest{
		CurrentPassword: "oldpassword",
		NewPassword:     "newpassword123",
	}
	updateBody, _ := json.Marshal(updateReq)

	// Test without token
	updateHTTPReq1 := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/users/%s/password", dummyUserID), bytes.NewReader(updateBody))
	updateHTTPReq1.Header.Set("Content-Type", "application/json")
	// Use unique IP address to avoid rate limiting interference between tests
	updateHTTPReq1.RemoteAddr = "192.0.2.30:1234"
	updateW1 := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(updateW1, updateHTTPReq1)
	assert.Equal(t, http.StatusUnauthorized, updateW1.Code)

	// Test with invalid token
	updateHTTPReq2 := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/users/%s/password", dummyUserID), bytes.NewReader(updateBody))
	updateHTTPReq2.Header.Set("Content-Type", "application/json")
	updateHTTPReq2.Header.Set("Authorization", "Bearer invalid-token")
	// Use unique IP address to avoid rate limiting interference between tests
	updateHTTPReq2.RemoteAddr = "192.0.2.31:1234"
	updateW2 := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(updateW2, updateHTTPReq2)
	assert.Equal(t, http.StatusUnauthorized, updateW2.Code)
}

func TestIntegration_Register_EmailAlreadyExists(t *testing.T) {
	server, database := setupTestServerForRouterIntegration(t)
	defer database.Close()

	email := "integration-duplicate-test@example.com"
	reqBody := types.CreateUserRequest{
		Name:     "First User",
		Email:    email,
		Password: "password123",
	}

	// Register first user
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var firstResponse types.LoginResponse
	json.Unmarshal(w.Body.Bytes(), &firstResponse)
	firstUserID := firstResponse.User.ID

	// Try to register again with same email
	reqBody2 := types.CreateUserRequest{
		Name:     "Second User",
		Email:    email,
		Password: "password456",
	}
	body2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	// Use unique IP address to avoid rate limiting interference between tests
	req2.RemoteAddr = "192.0.2.24:1234"
	w2 := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)
	assert.Contains(t, w2.Body.String(), "email already registered")

	// Cleanup
	database.DeleteUser(context.Background(), firstUserID)
}

func TestIntegration_Login_InvalidCredentials(t *testing.T) {
	server, database := setupTestServerForRouterIntegration(t)
	defer database.Close()

	// Register a user
	registerReq := types.CreateUserRequest{
		Name:     "Invalid Credentials Test",
		Email:    "invalid-credentials-test@example.com",
		Password: "correctpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	// Use unique IP address to avoid rate limiting interference between tests
	registerHTTPReq.RemoteAddr = "192.0.2.21:1234"
	registerW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(registerW, registerHTTPReq)

	var registerResponse types.LoginResponse
	json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	userID := registerResponse.User.ID

	// Try to login with wrong password
	loginReq := types.LoginRequest{
		Email:    "invalid-credentials-test@example.com",
		Password: "wrongpassword",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHTTPReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginHTTPReq.Header.Set("Content-Type", "application/json")
	// Use unique IP address to avoid rate limiting interference between tests
	loginHTTPReq.RemoteAddr = "192.0.2.22:1234"
	loginW := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(loginW, loginHTTPReq)

	assert.Equal(t, http.StatusUnauthorized, loginW.Code)
	assert.Contains(t, loginW.Body.String(), "invalid email or password")

	// Try to login with non-existent email
	loginReq2 := types.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "anypassword",
	}
	loginBody2, _ := json.Marshal(loginReq2)
	loginHTTPReq2 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody2))
	loginHTTPReq2.Header.Set("Content-Type", "application/json")
	// Use unique IP address to avoid rate limiting interference between tests
	loginHTTPReq2.RemoteAddr = "192.0.2.25:1234"
	loginW2 := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(loginW2, loginHTTPReq2)

	assert.Equal(t, http.StatusUnauthorized, loginW2.Code)
	assert.Contains(t, loginW2.Body.String(), "invalid email or password")

	// Cleanup
	database.DeleteUser(context.Background(), userID)
}
