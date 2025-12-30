package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/server/middleware"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB connects to the local DB for integration testing
func setupTestDBForAuth(t *testing.T) *db.DB {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Default to local docker connection
		dbURL = "postgres://resume:resume_dev@localhost:5432/resume_customizer?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping integration test: failed to connect to DB: %v", err)
	}
	return database
}

// setupTestAuthHandlerIntegration creates an AuthHandler with real services for integration testing
func setupTestAuthHandlerIntegration(t *testing.T) (*AuthHandler, *db.DB) {
	database := setupTestDBForAuth(t)
	passwordConfig, err := config.NewPasswordConfig()
	require.NoError(t, err)

	// Use test JWT config instead of environment variable
	jwtConfig := &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing-minimum-32-bytes",
		ExpirationHours: 24,
	}

	userSvc := NewUserService(database, passwordConfig)
	jwtSvc := NewJWTService(jwtConfig)
	handler := NewAuthHandler(userSvc, jwtSvc)

	return handler, database
}

// cleanupTestUser deletes a test user from the database
func cleanupTestUser(t *testing.T, database *db.DB, userID uuid.UUID) {
	ctx := context.Background()
	err := database.DeleteUser(ctx, userID)
	if err != nil {
		t.Logf("Failed to cleanup test user: %v", err)
	}
}

func TestIntegration_AuthHandler_Register_EndToEnd(t *testing.T) {
	handler, database := setupTestAuthHandlerIntegration(t)
	defer database.Close()

	reqBody := types.CreateUserRequest{
		Name:     "Integration Test User",
		Email:    "integration-test@example.com",
		Password: "testpassword123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response types.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response.User)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, reqBody.Name, response.User.Name)
	assert.Equal(t, reqBody.Email, response.User.Email)
	assert.True(t, response.User.PasswordSet)

	// Verify user exists in database
	ctx := context.Background()
	dbUser, err := database.GetUserByEmail(ctx, reqBody.Email)
	require.NoError(t, err)
	require.NotNil(t, dbUser)
	assert.Equal(t, response.User.ID, dbUser.ID)
	assert.NotEmpty(t, dbUser.PasswordHash)
	assert.NotEqual(t, reqBody.Password, dbUser.PasswordHash) // Password should be hashed

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
	cleanupTestUser(t, database, response.User.ID)
}

func TestIntegration_AuthHandler_Register_EmailAlreadyExists(t *testing.T) {
	handler, database := setupTestAuthHandlerIntegration(t)
	defer database.Close()

	email := "duplicate-test@example.com"
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
	handler.Register(w, req)
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
	w2 := httptest.NewRecorder()
	handler.Register(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)
	assert.Contains(t, w2.Body.String(), "email already registered")

	// Cleanup
	cleanupTestUser(t, database, firstUserID)
}

func TestIntegration_AuthHandler_Login_EndToEnd(t *testing.T) {
	handler, database := setupTestAuthHandlerIntegration(t)
	defer database.Close()

	// First register a user
	registerReq := types.CreateUserRequest{
		Name:     "Login Test User",
		Email:    "login-test@example.com",
		Password: "loginpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerW := httptest.NewRecorder()
	handler.Register(registerW, registerHTTPReq)

	var registerResponse types.LoginResponse
	err := json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	require.NoError(t, err)
	userID := registerResponse.User.ID

	// Now login with correct credentials
	loginReq := types.LoginRequest{
		Email:    "login-test@example.com",
		Password: "loginpassword123",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHTTPReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginHTTPReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()

	handler.Login(loginW, loginHTTPReq)

	assert.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse types.LoginResponse
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	assert.Equal(t, userID, loginResponse.User.ID)
	assert.NotEmpty(t, loginResponse.Token)
	// Tokens may be the same if generated in the same second, but both should be valid
	// The important thing is that a new token is generated and returned

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
	cleanupTestUser(t, database, userID)
}

func TestIntegration_AuthHandler_Login_InvalidCredentials(t *testing.T) {
	handler, database := setupTestAuthHandlerIntegration(t)
	defer database.Close()

	// Register a user
	registerReq := types.CreateUserRequest{
		Name:     "Invalid Login Test",
		Email:    "invalid-login-test@example.com",
		Password: "correctpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerW := httptest.NewRecorder()
	handler.Register(registerW, registerHTTPReq)

	var registerResponse types.LoginResponse
	json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	userID := registerResponse.User.ID

	// Try to login with wrong password
	loginReq := types.LoginRequest{
		Email:    "invalid-login-test@example.com",
		Password: "wrongpassword",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHTTPReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginHTTPReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()

	handler.Login(loginW, loginHTTPReq)

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
	loginW2 := httptest.NewRecorder()

	handler.Login(loginW2, loginHTTPReq2)

	assert.Equal(t, http.StatusUnauthorized, loginW2.Code)
	assert.Contains(t, loginW2.Body.String(), "invalid email or password")

	// Cleanup
	cleanupTestUser(t, database, userID)
}

func TestIntegration_AuthHandler_UpdatePassword_EndToEnd(t *testing.T) {
	handler, database := setupTestAuthHandlerIntegration(t)
	defer database.Close()

	// Register and login a user
	registerReq := types.CreateUserRequest{
		Name:     "Update Password Test",
		Email:    "update-password-test@example.com",
		Password: "oldpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerW := httptest.NewRecorder()
	handler.Register(registerW, registerHTTPReq)

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
	updateHTTPReq := httptest.NewRequest(http.MethodPut, "/users/me/password", bytes.NewReader(updateBody))
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("Authorization", "Bearer "+token)
	// Set user ID in context (simulating middleware)
	ctx := context.WithValue(updateHTTPReq.Context(), middleware.UserIDKey(), userID)
	updateHTTPReq = updateHTTPReq.WithContext(ctx)
	updateW := httptest.NewRecorder()

	handler.UpdatePassword(updateW, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateW.Code)

	var updateResponse map[string]string
	err := json.Unmarshal(updateW.Body.Bytes(), &updateResponse)
	require.NoError(t, err)
	assert.Equal(t, "Password updated successfully", updateResponse["message"])

	// Verify old password no longer works
	loginReqOld := types.LoginRequest{
		Email:    "update-password-test@example.com",
		Password: "oldpassword123",
	}
	loginBodyOld, _ := json.Marshal(loginReqOld)
	loginHTTPReqOld := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBodyOld))
	loginHTTPReqOld.Header.Set("Content-Type", "application/json")
	loginWOld := httptest.NewRecorder()
	handler.Login(loginWOld, loginHTTPReqOld)
	assert.Equal(t, http.StatusUnauthorized, loginWOld.Code)

	// Verify new password works
	loginReqNew := types.LoginRequest{
		Email:    "update-password-test@example.com",
		Password: "newpassword456",
	}
	loginBodyNew, _ := json.Marshal(loginReqNew)
	loginHTTPReqNew := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBodyNew))
	loginHTTPReqNew.Header.Set("Content-Type", "application/json")
	loginWNew := httptest.NewRecorder()
	handler.Login(loginWNew, loginHTTPReqNew)
	assert.Equal(t, http.StatusOK, loginWNew.Code)

	// Cleanup
	cleanupTestUser(t, database, userID)
}

func TestIntegration_AuthHandler_UpdatePassword_WrongCurrentPassword(t *testing.T) {
	handler, database := setupTestAuthHandlerIntegration(t)
	defer database.Close()

	// Register a user
	registerReq := types.CreateUserRequest{
		Name:     "Wrong Password Test",
		Email:    "wrong-password-test@example.com",
		Password: "correctpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerW := httptest.NewRecorder()
	handler.Register(registerW, registerHTTPReq)

	var registerResponse types.LoginResponse
	json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	userID := registerResponse.User.ID

	// Try to update password with wrong current password
	updateReq := types.UpdatePasswordRequest{
		CurrentPassword: "wrongpassword",
		NewPassword:     "newpassword123",
	}
	updateBody, _ := json.Marshal(updateReq)
	updateHTTPReq := httptest.NewRequest(http.MethodPut, "/users/me/password", bytes.NewReader(updateBody))
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(updateHTTPReq.Context(), middleware.UserIDKey(), userID)
	updateHTTPReq = updateHTTPReq.WithContext(ctx)
	updateW := httptest.NewRecorder()

	handler.UpdatePassword(updateW, updateHTTPReq)

	assert.Equal(t, http.StatusUnauthorized, updateW.Code)
	assert.Contains(t, updateW.Body.String(), "current password is incorrect")

	// Cleanup
	cleanupTestUser(t, database, userID)
}
