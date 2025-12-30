package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServerForGetMe creates a test server with a real database connection
// For unit tests, we'll use integration-style tests with real DB
func setupTestServerForGetMe(t *testing.T) (*Server, *db.DB) {
	return setupTestServerForRouter(t)
}

func TestHandleGetMe_Success(t *testing.T) {
	server, database := setupTestServerForGetMe(t)
	defer database.Close()

	// Create a test user
	ctx := context.Background()
	userID, err := database.CreateUser(ctx, "Unit Test User", "unit-test-"+uuid.New().String()+"@example.com", "555-1234")
	require.NoError(t, err)
	defer database.DeleteUser(ctx, userID)

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req = setUserIDInContext(req, userID)
	w := httptest.NewRecorder()

	server.handleGetMe(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var user db.User
	err = json.Unmarshal(w.Body.Bytes(), &user)
	require.NoError(t, err)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "Unit Test User", user.Name)
	assert.Equal(t, "555-1234", user.Phone)
}

func TestHandleGetMe_UserNotFound(t *testing.T) {
	server, database := setupTestServerForGetMe(t)
	defer database.Close()

	// Use a non-existent user ID
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req = setUserIDInContext(req, userID)
	w := httptest.NewRecorder()

	server.handleGetMe(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errorResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"], "User not found")
}

func TestHandleGetMe_MissingUserIDInContext(t *testing.T) {
	server, database := setupTestServerForGetMe(t)
	defer database.Close()

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	// Don't set user ID in context - simulate missing authentication
	w := httptest.NewRecorder()

	server.handleGetMe(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var errorResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	assert.Equal(t, "Unauthorized", errorResp["error"])
}

// Integration tests using real database
func TestGetMe_Integration(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	// Register user with password (this sets the password and returns a token)
	registerReq := types.CreateUserRequest{
		Name:     "GetMe Test User",
		Email:    "getme-" + uuid.New().String() + "@example.com",
		Password: "testpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerHTTPReq.RemoteAddr = "192.0.2.1:1234"
	registerW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(registerW, registerHTTPReq)

	require.Equal(t, http.StatusCreated, registerW.Code)
	var registerResponse types.LoginResponse
	err := json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	require.NoError(t, err)
	require.NotNil(t, registerResponse.User)
	userID := registerResponse.User.ID
	token := registerResponse.Token
	defer database.DeleteUser(context.Background(), userID)

	// Make request with JWT token
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "192.0.2.1:5678"
	w := httptest.NewRecorder()

	handler := server.httpServer.Handler
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var user db.User
	err = json.Unmarshal(w.Body.Bytes(), &user)
	require.NoError(t, err)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "GetMe Test User", user.Name)
	assert.True(t, user.PasswordSet)
}

func TestGetMe_Integration_InvalidToken(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	req.RemoteAddr = "192.0.2.2:1234"
	w := httptest.NewRecorder()

	handler := server.httpServer.Handler
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetMe_Integration_MissingToken(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req.RemoteAddr = "192.0.2.3:1234"
	w := httptest.NewRecorder()

	handler := server.httpServer.Handler
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetMe_Integration_ExpiredToken(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	// Create a test user
	ctx := context.Background()
	userID, err := database.CreateUser(ctx, "Expired Token Test", "expired-"+uuid.New().String()+"@example.com", "555-9999")
	require.NoError(t, err)
	defer database.DeleteUser(ctx, userID)

	// Create JWT service with very short expiration (1 hour, but we'll use a manually expired token)
	// For this test, we'll use an invalid/expired token string
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzNDU2NzgtMTIzNC0xMjM0LTEyMzQtMTIzNDU2Nzg5MDEyIiwiZXhwIjoxfQ.invalid")
	req.RemoteAddr = "192.0.2.4:1234"
	w := httptest.NewRecorder()

	handler := server.httpServer.Handler
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetMe_Integration_UserDeletedAfterTokenIssued(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	// Register user with password to get a token
	registerReq := types.CreateUserRequest{
		Name:     "Deleted User Test",
		Email:    "deleted-" + uuid.New().String() + "@example.com",
		Password: "testpassword123",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerHTTPReq.RemoteAddr = "192.0.2.5:1234"
	registerW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(registerW, registerHTTPReq)

	require.Equal(t, http.StatusCreated, registerW.Code)
	var registerResponse types.LoginResponse
	err := json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	require.NoError(t, err)
	require.NotNil(t, registerResponse.User)
	userID := registerResponse.User.ID
	token := registerResponse.Token

	// Delete the user
	ctx := context.Background()
	err = database.DeleteUser(ctx, userID)
	require.NoError(t, err)

	// Try to get user with valid token but deleted user
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "192.0.2.5:5678"
	w := httptest.NewRecorder()

	handler := server.httpServer.Handler
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errorResp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"], "User not found")
}

func TestGetMe_Integration_ResponseFormat(t *testing.T) {
	server, database := setupTestServerForRouter(t)
	defer database.Close()

	// Register user with password to get a token
	registerReq := types.CreateUserRequest{
		Name:     "Format Test User",
		Email:    "format-" + uuid.New().String() + "@example.com",
		Password: "testpassword123",
		Phone:    "555-FORMAT",
	}
	registerBody, _ := json.Marshal(registerReq)
	registerHTTPReq := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody))
	registerHTTPReq.Header.Set("Content-Type", "application/json")
	registerHTTPReq.RemoteAddr = "192.0.2.6:1234"
	registerW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(registerW, registerHTTPReq)

	require.Equal(t, http.StatusCreated, registerW.Code)
	var registerResponse types.LoginResponse
	err := json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
	require.NoError(t, err)
	require.NotNil(t, registerResponse.User)
	userID := registerResponse.User.ID
	token := registerResponse.Token
	defer database.DeleteUser(context.Background(), userID)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "192.0.2.6:5678"
	w := httptest.NewRecorder()

	handler := server.httpServer.Handler
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Verify response format matches User schema
	var user db.User
	err = json.Unmarshal(w.Body.Bytes(), &user)
	require.NoError(t, err)

	// Verify all expected fields are present
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "Format Test User", user.Name)
	assert.NotEmpty(t, user.Email)
	assert.Equal(t, "555-FORMAT", user.Phone)
	// Password hash should never be in response
	assert.Empty(t, user.PasswordHash)
	// PasswordSet should be present and true
	assert.True(t, user.PasswordSet)
}
