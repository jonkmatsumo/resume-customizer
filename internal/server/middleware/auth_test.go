package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTokenValidator is a test implementation of TokenValidator for unit tests.
type testTokenValidator struct {
	validTokens map[string]uuid.UUID
}

func newTestTokenValidator() *testTokenValidator {
	return &testTokenValidator{
		validTokens: make(map[string]uuid.UUID),
	}
}

func (v *testTokenValidator) addValidToken(token string, userID uuid.UUID) {
	v.validTokens[token] = userID
}

func (v *testTokenValidator) ValidateToken(tokenString string) (UserIDGetter, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token string is empty")
	}
	userID, ok := v.validTokens[tokenString]
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}
	return &testClaims{userID: userID}, nil
}

type testClaims struct {
	userID uuid.UUID
}

func (c *testClaims) GetUserID() uuid.UUID {
	return c.userID
}

func setupTestJWTService(_ *testing.T) TokenValidator {
	return newTestTokenValidator()
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtService := setupTestJWTService(t).(*testTokenValidator)
	userID := uuid.New()

	// Create valid token for test
	token := "valid-test-token-123"
	jwtService.addValidToken(token, userID)

	// Create handler that checks context
	handlerCalled := false
	var contextUserID uuid.UUID
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		extractedUserID, err := GetUserID(r)
		require.NoError(t, err)
		contextUserID = extractedUserID
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware
	middleware := AuthMiddleware(jwtService)
	wrappedHandler := middleware(handler)

	// Create request with Authorization header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(w, req)

	// Verify
	assert.True(t, handlerCalled, "handler should be called")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, userID, contextUserID)
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	jwtService := setupTestJWTService(t)

	handlerCalled := false
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
	})

	middleware := AuthMiddleware(jwtService)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Authorization header
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.False(t, handlerCalled, "handler should not be called")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	jwtService := setupTestJWTService(t)

	tests := []struct {
		name        string
		authHeader  string
		description string
	}{
		{
			name:        "missing Bearer prefix",
			authHeader:  "token123",
			description: "should reject token without Bearer prefix",
		},
		{
			name:        "empty token",
			authHeader:  "Bearer ",
			description: "should reject empty token",
		},
		{
			name:        "only Bearer",
			authHeader:  "Bearer",
			description: "should reject Bearer without token",
		},
		{
			name:        "multiple spaces",
			authHeader:  "Bearer  token123",
			description: "should handle multiple spaces",
		},
		{
			name:        "lowercase bearer",
			authHeader:  "bearer token123",
			description: "should handle case-insensitive Bearer",
		},
		{
			name:        "mixed case bearer",
			authHeader:  "BeArEr token123",
			description: "should handle mixed case Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				handlerCalled = true
			})

			middleware := AuthMiddleware(jwtService)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tt.authHeader)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			// For valid Bearer format (case-insensitive), token will be invalid but format is OK
			if strings.EqualFold(strings.Fields(tt.authHeader)[0], "Bearer") && len(strings.Fields(tt.authHeader)) == 2 && strings.TrimSpace(strings.Fields(tt.authHeader)[1]) != "" {
				// Format is valid, but token is invalid - will fail validation
				assert.False(t, handlerCalled, "handler should not be called for invalid token")
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			} else {
				// Format is invalid
				assert.False(t, handlerCalled, "handler should not be called")
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			}
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwtService := setupTestJWTService(t)

	tests := []struct {
		name        string
		token       string
		description string
	}{
		{
			name:        "wrong signature",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzIn0.invalid",
			description: "should reject token with wrong signature",
		},
		{
			name:        "malformed token",
			token:       "not.a.valid.jwt.token",
			description: "should reject malformed token",
		},
		{
			name:        "empty token string",
			token:       "",
			description: "should reject empty token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				handlerCalled = true
			})

			middleware := AuthMiddleware(jwtService)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			assert.False(t, handlerCalled, "handler should not be called")
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Body.String(), "Unauthorized")
		})
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	jwtService := setupTestJWTService(t)

	// For this test, we'll use an invalid token to simulate expired/invalid scenarios
	// Testing actual expiration would require time manipulation which is better suited for integration tests
	handlerCalled := false
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
	})

	middleware := AuthMiddleware(jwtService)
	wrappedHandler := middleware(handler)

	// Use a token signed with different secret (simulates various invalid scenarios including expired)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.expired.token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.False(t, handlerCalled, "handler should not be called for invalid token")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ContextInjection(t *testing.T) {
	jwtService := setupTestJWTService(t).(*testTokenValidator)
	userID := uuid.New()

	token := "test-token-for-context-injection"
	jwtService.addValidToken(token, userID)

	var extractedUserID uuid.UUID
	var err error
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extractedUserID, err = GetUserID(r)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(jwtService)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, userID, extractedUserID)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUserID_Success(t *testing.T) {
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, userIDKey, userID)
	req = req.WithContext(ctx)

	extractedUserID, err := GetUserID(req)
	require.NoError(t, err)
	assert.Equal(t, userID, extractedUserID)
}

func TestGetUserID_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No user ID in context

	userID, err := GetUserID(req)
	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, userID)
	assert.Contains(t, err.Error(), "user ID not found")
}

func TestGetUserID_InvalidType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := req.Context()
	// Set wrong type in context
	ctx = context.WithValue(ctx, userIDKey, "not-a-uuid")
	req = req.WithContext(ctx)

	userID, err := GetUserID(req)
	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, userID)
}
