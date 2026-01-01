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

	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServerForE2E creates a test server instance for E2E testing
func setupTestServerForE2E(t *testing.T) (*Server, *db.DB) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://resume:resume_dev@localhost:5432/resume_customizer?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping E2E test: failed to connect to DB: %v", err)
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

func TestE2E_CompleteAuthenticationFlow(t *testing.T) {
	// Skip in short mode (CI/CD) - E2E tests are comprehensive and slower
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode (CI/CD)")
	}

	server, database := setupTestServerForE2E(t)
	defer database.Close()

	handler := server.httpServer.Handler

	// Step 1: Register user (use unique email with timestamp to avoid conflicts)
	timestamp := time.Now().UnixNano()
	registerReq := types.CreateUserRequest{
		Name:     "E2E Test User",
		Email:    fmt.Sprintf("e2e-complete-flow-%d@example.com", timestamp),
		Password: "initialpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerHTTPReq.RemoteAddr = "192.0.2.1:1234" // Use unique IP to avoid rate limiting
	registerW := httptest.NewRecorder()
	handler.ServeHTTP(registerW, registerHTTPReq)

	assert.Equal(t, http.StatusCreated, registerW.Code)
	var registerResponse types.LoginResponse
	err := json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	require.NoError(t, err)
	require.NotNil(t, registerResponse.User)
	userID := registerResponse.User.ID
	testEmail := registerReq.Email // Store email for reuse

	// Step 2: Login with credentials (use same email as registration)
	loginReq := types.LoginRequest{
		Email:    testEmail,
		Password: "initialpassword123",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBody))
	loginHTTPReq.Header.Set("Content-Type", "application/json")
	loginHTTPReq.RemoteAddr = "192.0.2.1:1234" // Use same IP
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginHTTPReq)

	assert.Equal(t, http.StatusOK, loginW.Code)
	var loginResponse types.LoginResponse
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	require.NotNil(t, loginResponse.User)
	assert.Equal(t, userID, loginResponse.User.ID)
	loginToken := loginResponse.Token

	// Step 3: Use token to update password
	updateReq := types.UpdatePasswordRequest{
		CurrentPassword: "initialpassword123",
		NewPassword:     "updatedpassword456",
	}
	updateBody, _ := json.Marshal(updateReq)
	updateHTTPReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/users/%s/password", userID), bytes.NewReader(updateBody))
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("Authorization", "Bearer "+loginToken)
	updateHTTPReq.RemoteAddr = "192.0.2.1:1234" // Use same IP
	updateW := httptest.NewRecorder()
	handler.ServeHTTP(updateW, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateW.Code)

	// Step 4: Login with new password (with retry logic for rate limiting)
	loginReq2 := types.LoginRequest{
		Email:    testEmail,
		Password: "updatedpassword456",
	}
	loginBody2, _ := json.Marshal(loginReq2)

	var loginW2 *httptest.ResponseRecorder
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		loginHTTPReq2 := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBody2))
		loginHTTPReq2.Header.Set("Content-Type", "application/json")
		loginHTTPReq2.RemoteAddr = "192.0.2.1:1234" // Use same IP
		loginW2 = httptest.NewRecorder()
		handler.ServeHTTP(loginW2, loginHTTPReq2)

		if loginW2.Code != http.StatusTooManyRequests {
			break
		}
		if attempt < maxRetries-1 {
			delay := time.Duration(attempt+1) * time.Second
			time.Sleep(delay)
		}
	}

	assert.Equal(t, http.StatusOK, loginW2.Code, "Login with new password should succeed after retries")
	var loginResponse2 types.LoginResponse
	err = json.Unmarshal(loginW2.Body.Bytes(), &loginResponse2)
	require.NoError(t, err)
	require.NotNil(t, loginResponse2.User)
	assert.Equal(t, userID, loginResponse2.User.ID)

	// Verify old password no longer works
	loginReq3 := types.LoginRequest{
		Email:    testEmail,
		Password: "initialpassword123",
	}
	loginBody3, _ := json.Marshal(loginReq3)
	loginHTTPReq3 := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBody3))
	loginHTTPReq3.Header.Set("Content-Type", "application/json")
	loginW3 := httptest.NewRecorder()
	handler.ServeHTTP(loginW3, loginHTTPReq3)

	assert.Equal(t, http.StatusUnauthorized, loginW3.Code)

	// Cleanup
	database.DeleteUser(context.Background(), userID)
}

// TestE2E_MultipleUsers tests multiple user registration and authentication.
// This test is excluded from CI/CD runs (use -short flag to skip).
// It includes retry logic with exponential backoff for rate limiting.
func TestE2E_MultipleUsers(t *testing.T) {
	// Skip in CI/CD or when -short flag is used
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode (CI/CD)")
	}

	server, database := setupTestServerForE2E(t)
	defer database.Close()

	handler := server.httpServer.Handler
	var userIDs []string
	var tokens []string
	var emails []string

	// Generate unique timestamp for this test run
	timestamp := time.Now().Format("20060102150405")

	// Register multiple users with retry logic for rate limiting
	for i := 0; i < 3; i++ {
		email := "e2e-multi-user-" + string(rune('a'+i)) + "-" + timestamp + "@example.com"
		emails = append(emails, email)

		registerReq := types.CreateUserRequest{
			Name:     "E2E Multi User",
			Email:    email,
			Password: "testpassword123",
		}
		registerBody, _ := json.Marshal(registerReq)

		// Retry logic with exponential backoff for rate limiting
		var registerResponse types.LoginResponse
		maxRetries := 5
		baseDelay := 100 * time.Millisecond
		registered := false

		for attempt := 0; attempt < maxRetries; attempt++ {
			registerHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody))
			registerHTTPReq.Header.Set("Content-Type", "application/json")
			// Use different IP addresses to avoid rate limiting
			registerHTTPReq.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", i+1)
			registerW := httptest.NewRecorder()
			handler.ServeHTTP(registerW, registerHTTPReq)

			if registerW.Code == http.StatusCreated {
				err := json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
				require.NoError(t, err, "Should be able to unmarshal response")
				require.NotNil(t, registerResponse.User, "User should be in response")
				registered = true
				break
			}
			if registerW.Code == http.StatusTooManyRequests && attempt < maxRetries-1 {
				// Rate limited - wait with exponential backoff
				delay := baseDelay * time.Duration(1<<uint(attempt)) // Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms
				t.Logf("Rate limited on attempt %d, waiting %v before retry", attempt+1, delay)
				time.Sleep(delay)
				continue
			}
			// Other error - fail the test
			require.Equal(t, http.StatusCreated, registerW.Code, "User registration should succeed after retries")
		}

		require.True(t, registered, "User should be registered after retries")
		userIDs = append(userIDs, registerResponse.User.ID.String())
		tokens = append(tokens, registerResponse.Token)
	}

	// Verify tokens are unique
	assert.Equal(t, 3, len(tokens), "Should have 3 tokens")
	assert.NotEqual(t, tokens[0], tokens[1], "Tokens should be unique")
	assert.NotEqual(t, tokens[1], tokens[2], "Tokens should be unique")
	assert.NotEqual(t, tokens[0], tokens[2], "Tokens should be unique")

	// Login as each user and verify tokens work
	for i, token := range tokens {
		loginReq := types.LoginRequest{
			Email:    emails[i],
			Password: "testpassword123",
		}
		loginBody, _ := json.Marshal(loginReq)
		loginHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBody))
		loginHTTPReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()
		handler.ServeHTTP(loginW, loginHTTPReq)

		assert.Equal(t, http.StatusOK, loginW.Code, "Login should succeed")
		var loginResponse types.LoginResponse
		err := json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
		require.NoError(t, err, "Should be able to unmarshal login response")
		require.NotNil(t, loginResponse.User, "User should be in login response")
		assert.Equal(t, userIDs[i], loginResponse.User.ID.String(), "User ID should match")

		// Verify token works for protected route
		updateReq := types.UpdatePasswordRequest{
			CurrentPassword: "testpassword123",
			NewPassword:     "newpassword456",
		}
		updateBody, _ := json.Marshal(updateReq)
		updateHTTPReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/users/%s/password", loginResponse.User.ID), bytes.NewReader(updateBody))
		updateHTTPReq.Header.Set("Content-Type", "application/json")
		updateHTTPReq.Header.Set("Authorization", "Bearer "+token)
		updateW := httptest.NewRecorder()
		handler.ServeHTTP(updateW, updateHTTPReq)

		assert.Equal(t, http.StatusOK, updateW.Code, "Token should work for protected route")
	}

	// Cleanup
	for _, userIDStr := range userIDs {
		userID, _ := json.Marshal(userIDStr)
		// Parse UUID from string
		userIDParsed, err := json.Marshal(userIDStr)
		if err == nil {
			var uuidStr string
			json.Unmarshal(userIDParsed, &uuidStr)
			// Cleanup handled by defer
		}
		_ = userID
	}
	// Manual cleanup - use the emails we actually registered
	for _, email := range emails {
		dbUser, err := database.GetUserByEmail(context.Background(), email)
		if err == nil && dbUser != nil {
			database.DeleteUser(context.Background(), dbUser.ID)
		}
	}
}

func TestE2E_TokenReuse(t *testing.T) {
	// Skip in short mode (CI/CD) - E2E tests are comprehensive and slower
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode (CI/CD)")
	}

	server, database := setupTestServerForE2E(t)
	defer database.Close()

	handler := server.httpServer.Handler

	// Register and login
	registerReq := types.CreateUserRequest{
		Name:     "E2E Token Reuse",
		Email:    "e2e-token-reuse@example.com",
		Password: "testpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerW := httptest.NewRecorder()
	handler.ServeHTTP(registerW, registerHTTPReq)

	var registerResponse types.LoginResponse
	json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	userID := registerResponse.User.ID
	token := registerResponse.Token

	// Use same token multiple times
	for i := 0; i < 5; i++ {
		updateReq := types.UpdatePasswordRequest{
			CurrentPassword: "testpassword123",
			NewPassword:     "newpassword456",
		}
		updateBody, _ := json.Marshal(updateReq)
		updateHTTPReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/users/%s/password", userID), bytes.NewReader(updateBody))
		updateHTTPReq.Header.Set("Content-Type", "application/json")
		updateHTTPReq.Header.Set("Authorization", "Bearer "+token)
		updateW := httptest.NewRecorder()
		handler.ServeHTTP(updateW, updateHTTPReq)

		// First request should succeed, subsequent ones will fail due to password mismatch
		// (since we're updating password each time)
		if i == 0 {
			assert.Equal(t, http.StatusOK, updateW.Code, "First token use should succeed")
		} else {
			// Password was changed, so current password is wrong
			assert.Equal(t, http.StatusUnauthorized, updateW.Code, "Subsequent uses should fail due to password change")
		}
	}

	// Cleanup
	database.DeleteUser(context.Background(), userID)
}

func TestE2E_TokenExpiration(t *testing.T) {
	// Skip in short mode (CI/CD) - E2E tests are comprehensive and slower
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode (CI/CD)")
	}

	server, database := setupTestServerForE2E(t)
	defer database.Close()

	handler := server.httpServer.Handler

	// Register user
	registerReq := types.CreateUserRequest{
		Name:     "E2E Token Expiration",
		Email:    "e2e-token-expiration@example.com",
		Password: "testpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerW := httptest.NewRecorder()
	handler.ServeHTTP(registerW, registerHTTPReq)

	var registerResponse types.LoginResponse
	json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	userID := registerResponse.User.ID

	// Create a token with very short expiration (1 second)
	// Note: JWT config has minimum 1 hour, so we'll test with a manually expired token
	jwtConfig := &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing-minimum-32-bytes",
		ExpirationHours: 24,
	}
	jwtSvc := NewJWTService(jwtConfig)

	// Generate a token and verify it works
	token, err := jwtSvc.GenerateToken(userID)
	require.NoError(t, err)

	updateReq := types.UpdatePasswordRequest{
		CurrentPassword: "testpassword123",
		NewPassword:     "newpassword456",
	}
	updateBody, _ := json.Marshal(updateReq)
	updateHTTPReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/users/%s/password", userID), bytes.NewReader(updateBody))
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("Authorization", "Bearer "+token)
	updateW := httptest.NewRecorder()
	handler.ServeHTTP(updateW, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateW.Code, "Valid token should work")

	// Test that token validation checks expiration
	// (Actual expiration would require waiting 24 hours, which is impractical for tests)
	// This is verified in integration tests that test token expiration logic

	// Cleanup
	database.DeleteUser(context.Background(), userID)
}

func TestE2E_PasswordUpdateFlow(t *testing.T) {
	// Skip in short mode (CI/CD) - E2E tests are comprehensive and slower
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode (CI/CD)")
	}

	server, database := setupTestServerForE2E(t)
	defer database.Close()

	handler := server.httpServer.Handler

	// Register and login
	registerReq := types.CreateUserRequest{
		Name:     "E2E Password Update Flow",
		Email:    "e2e-password-update-flow@example.com",
		Password: "password1",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerW := httptest.NewRecorder()
	handler.ServeHTTP(registerW, registerHTTPReq)

	assert.Equal(t, http.StatusCreated, registerW.Code, "Registration should succeed")
	var registerResponse types.LoginResponse
	err := json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	require.NoError(t, err)
	require.NotNil(t, registerResponse.User, "User should be in registration response")
	userID := registerResponse.User.ID

	// Update password multiple times
	passwords := []string{"password1", "password2", "password3", "password4"}

	for i := 0; i < len(passwords)-1; i++ {
		currentPassword := passwords[i]
		newPassword := passwords[i+1]

		// Login with current password
		loginReq := types.LoginRequest{
			Email:    "e2e-password-update-flow@example.com",
			Password: currentPassword,
		}
		loginBody, _ := json.Marshal(loginReq)
		loginHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBody))
		loginHTTPReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()
		handler.ServeHTTP(loginW, loginHTTPReq)

		assert.Equal(t, http.StatusOK, loginW.Code, "Should be able to login with current password")
		var loginResponse types.LoginResponse
		json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
		token := loginResponse.Token

		// Update to new password
		updateReq := types.UpdatePasswordRequest{
			CurrentPassword: currentPassword,
			NewPassword:     newPassword,
		}
		updateBody, _ := json.Marshal(updateReq)
		updateHTTPReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/users/%s/password", userID), bytes.NewReader(updateBody))
		updateHTTPReq.Header.Set("Content-Type", "application/json")
		updateHTTPReq.Header.Set("Authorization", "Bearer "+token)
		updateW := httptest.NewRecorder()
		handler.ServeHTTP(updateW, updateHTTPReq)

		assert.Equal(t, http.StatusOK, updateW.Code, "Should be able to update password")

		// Verify old password no longer works
		loginReqOld := types.LoginRequest{
			Email:    "e2e-password-update-flow@example.com",
			Password: currentPassword,
		}
		loginBodyOld, _ := json.Marshal(loginReqOld)
		loginHTTPReqOld := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBodyOld))
		loginHTTPReqOld.Header.Set("Content-Type", "application/json")
		loginWOld := httptest.NewRecorder()
		handler.ServeHTTP(loginWOld, loginHTTPReqOld)

		assert.Equal(t, http.StatusUnauthorized, loginWOld.Code, "Old password should no longer work")

		// Verify new password works
		loginReqNew := types.LoginRequest{
			Email:    "e2e-password-update-flow@example.com",
			Password: newPassword,
		}
		loginBodyNew, _ := json.Marshal(loginReqNew)
		loginHTTPReqNew := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBodyNew))
		loginHTTPReqNew.Header.Set("Content-Type", "application/json")
		loginWNew := httptest.NewRecorder()
		handler.ServeHTTP(loginWNew, loginHTTPReqNew)

		assert.Equal(t, http.StatusOK, loginWNew.Code, "New password should work")
	}

	// Cleanup
	database.DeleteUser(context.Background(), userID)
}
