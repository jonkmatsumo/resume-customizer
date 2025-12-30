package server

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestJWTService(_ *testing.T, expirationHours int) *JWTService {
	cfg := &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing-minimum-32-bytes",
		ExpirationHours: expirationHours,
	}
	return NewJWTService(cfg)
}

func TestJWTService_GenerateToken(t *testing.T) {
	service := setupTestJWTService(t, 24)
	userID := uuid.New()

	token, err := service.GenerateToken(userID)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Test token format is valid JWT (three parts separated by dots)
	parts := strings.Split(token, ".")
	assert.Equal(t, 3, len(parts), "JWT should have 3 parts separated by dots")
	assert.NotEmpty(t, parts[0], "Header should not be empty")
	assert.NotEmpty(t, parts[1], "Payload should not be empty")
	assert.NotEmpty(t, parts[2], "Signature should not be empty")
}

func TestJWTService_GenerateToken_ContainsUserID(t *testing.T) {
	service := setupTestJWTService(t, 24)
	userID := uuid.New()

	token, err := service.GenerateToken(userID)
	require.NoError(t, err)

	// Validate token and check claims
	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestJWTService_GenerateToken_DifferentUserIDs(t *testing.T) {
	service := setupTestJWTService(t, 24)
	userID1 := uuid.New()
	userID2 := uuid.New()

	token1, err1 := service.GenerateToken(userID1)
	require.NoError(t, err1)

	token2, err2 := service.GenerateToken(userID2)
	require.NoError(t, err2)

	// Tokens should be different
	assert.NotEqual(t, token1, token2)

	// Validate both tokens
	claims1, err := service.ValidateToken(token1)
	require.NoError(t, err)
	assert.Equal(t, userID1, claims1.UserID)

	claims2, err := service.ValidateToken(token2)
	require.NoError(t, err)
	assert.Equal(t, userID2, claims2.UserID)
}

func TestJWTService_GenerateToken_UniqueTokens(t *testing.T) {
	service := setupTestJWTService(t, 24)
	userID := uuid.New()

	// Generate two tokens for the same user
	token1, err1 := service.GenerateToken(userID)
	require.NoError(t, err1)

	// Delay to ensure different issued at time (at least 1 second)
	time.Sleep(1100 * time.Millisecond)

	token2, err2 := service.GenerateToken(userID)
	require.NoError(t, err2)

	// Tokens should be different due to issued at time
	assert.NotEqual(t, token1, token2, "tokens generated at different times should be different")

	// Both should be valid and contain same user ID
	claims1, err := service.ValidateToken(token1)
	require.NoError(t, err)
	assert.Equal(t, userID, claims1.UserID)

	claims2, err := service.ValidateToken(token2)
	require.NoError(t, err)
	assert.Equal(t, userID, claims2.UserID)
}

func TestJWTService_ValidateToken_Success(t *testing.T) {
	service := setupTestJWTService(t, 24)
	userID := uuid.New()

	token, err := service.GenerateToken(userID)
	require.NoError(t, err)

	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}

func TestJWTService_ValidateToken_InvalidSignature(t *testing.T) {
	service1 := setupTestJWTService(t, 24)
	service2 := setupTestJWTService(t, 24)
	// Create service2 with different secret
	service2.config.Secret = "different-secret-key-for-jwt-signing-minimum-32-bytes"

	userID := uuid.New()
	token, err := service1.GenerateToken(userID)
	require.NoError(t, err)

	// Try to validate with different secret
	claims, err := service2.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "signature")
}

func TestJWTService_ValidateToken_MalformedToken(t *testing.T) {
	service := setupTestJWTService(t, 24)

	tests := []struct {
		name        string
		token       string
		description string
	}{
		{
			name:        "empty token",
			token:       "",
			description: "should error on empty token",
		},
		{
			name:        "invalid format - one part",
			token:       "invalid",
			description: "should error on token with one part",
		},
		{
			name:        "invalid format - two parts",
			token:       "invalid.token",
			description: "should error on token with two parts",
		},
		{
			name:        "invalid format - four parts",
			token:       "invalid.token.format.extra",
			description: "should error on token with four parts",
		},
		{
			name:        "invalid base64",
			token:       "invalid.base64.signature",
			description: "should error on invalid base64 encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateToken(tt.token)
			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestJWTService_TokenExpiration(t *testing.T) {
	// Test with very short expiration (1 second)
	service := setupTestJWTService(t, 24)
	userID := uuid.New()

	// Generate token with custom expiration (1 second) by manually creating claims
	now := time.Now()
	expiresAt := now.Add(1 * time.Second)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(service.config.Secret))
	require.NoError(t, err)

	// Token should be valid immediately
	validClaims, err := service.ValidateToken(tokenString)
	require.NoError(t, err)
	assert.Equal(t, userID, validClaims.UserID)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Token should be invalid after expiration
	expiredClaims, err := service.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, expiredClaims)
	assert.Contains(t, err.Error(), "expired")
}

func TestJWTService_TokenExpiration_DifferentHours(t *testing.T) {
	tests := []struct {
		name            string
		expirationHours int
		description     string
	}{
		{
			name:            "1 hour expiration",
			expirationHours: 1,
			description:     "should work with 1 hour expiration",
		},
		{
			name:            "12 hours expiration",
			expirationHours: 12,
			description:     "should work with 12 hours expiration",
		},
		{
			name:            "24 hours expiration",
			expirationHours: 24,
			description:     "should work with 24 hours expiration",
		},
		{
			name:            "48 hours expiration",
			expirationHours: 48,
			description:     "should work with 48 hours expiration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := setupTestJWTService(t, tt.expirationHours)
			userID := uuid.New()

			token, err := service.GenerateToken(userID)
			require.NoError(t, err)

			claims, err := service.ValidateToken(token)
			require.NoError(t, err)
			assert.Equal(t, userID, claims.UserID)
			assert.NotNil(t, claims.ExpiresAt)
		})
	}
}

func TestJWTService_ErrorHandling(t *testing.T) {
	service := setupTestJWTService(t, 24)

	// Test empty token
	claims, err := service.ValidateToken("")
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "empty")
}
