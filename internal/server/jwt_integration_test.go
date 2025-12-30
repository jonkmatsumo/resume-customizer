package server

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_JWTService_GenerateAndValidate(t *testing.T) {
	// Set up test environment variables
	originalSecret := os.Getenv("JWT_SECRET")
	originalExpiration := os.Getenv("JWT_EXPIRATION_HOURS")
	defer func() {
		if originalSecret != "" {
			os.Setenv("JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
		if originalExpiration != "" {
			os.Setenv("JWT_EXPIRATION_HOURS", originalExpiration)
		} else {
			os.Unsetenv("JWT_EXPIRATION_HOURS")
		}
	}()

	os.Setenv("JWT_SECRET", "integration-test-secret-key-minimum-32-bytes-long")
	os.Unsetenv("JWT_EXPIRATION_HOURS") // Use default

	cfg, err := config.NewJWTConfig()
	require.NoError(t, err)

	service := NewJWTService(cfg)
	userID := uuid.New()

	// Generate token
	token, err := service.GenerateToken(userID)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Validate token immediately (should succeed)
	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)

	// Test with different user IDs
	userID2 := uuid.New()
	token2, err := service.GenerateToken(userID2)
	require.NoError(t, err)

	claims2, err := service.ValidateToken(token2)
	require.NoError(t, err)
	assert.Equal(t, userID2, claims2.UserID)
	assert.NotEqual(t, userID, claims2.UserID)
}

func TestIntegration_JWTService_Expiration(t *testing.T) {
	// Set up test environment variables
	originalSecret := os.Getenv("JWT_SECRET")
	originalExpiration := os.Getenv("JWT_EXPIRATION_HOURS")
	defer func() {
		if originalSecret != "" {
			os.Setenv("JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
		if originalExpiration != "" {
			os.Setenv("JWT_EXPIRATION_HOURS", originalExpiration)
		} else {
			os.Unsetenv("JWT_EXPIRATION_HOURS")
		}
	}()

	os.Setenv("JWT_SECRET", "integration-test-secret-key-minimum-32-bytes-long")

	// Create service with 1 minute expiration for testing
	cfg := &config.JWTConfig{
		Secret:          "integration-test-secret-key-minimum-32-bytes-long",
		ExpirationHours: 0, // We'll manually create token with 1 second expiration
	}
	service := NewJWTService(cfg)

	userID := uuid.New()

	// Generate token with 1 second expiration manually
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
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	require.NoError(t, err)

	// Token should be valid before expiration
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

func TestIntegration_JWTService_EnvironmentVariables(t *testing.T) {
	// Save original values
	originalSecret := os.Getenv("JWT_SECRET")
	originalExpiration := os.Getenv("JWT_EXPIRATION_HOURS")
	defer func() {
		if originalSecret != "" {
			os.Setenv("JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
		if originalExpiration != "" {
			os.Setenv("JWT_EXPIRATION_HOURS", originalExpiration)
		} else {
			os.Unsetenv("JWT_EXPIRATION_HOURS")
		}
	}()

	// Test with JWT_SECRET from environment
	os.Setenv("JWT_SECRET", "env-test-secret-key-minimum-32-bytes-long")
	os.Setenv("JWT_EXPIRATION_HOURS", "12")

	cfg, err := config.NewJWTConfig()
	require.NoError(t, err)
	assert.Equal(t, "env-test-secret-key-minimum-32-bytes-long", cfg.Secret)
	assert.Equal(t, 12, cfg.ExpirationHours)

	service := NewJWTService(cfg)
	userID := uuid.New()

	token, err := service.GenerateToken(userID)
	require.NoError(t, err)

	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)

	// Test default values when env vars not set
	os.Unsetenv("JWT_EXPIRATION_HOURS")
	os.Setenv("JWT_SECRET", "default-test-secret-key-minimum-32-bytes-long")

	cfg2, err := config.NewJWTConfig()
	require.NoError(t, err)
	assert.Equal(t, 24, cfg2.ExpirationHours, "should use default 24 hours")

	// Test error when JWT_SECRET missing
	os.Unsetenv("JWT_SECRET")
	cfg3, err := config.NewJWTConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg3)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestIntegration_JWTService_RealWorldScenario(t *testing.T) {
	// Set up test environment variables
	originalSecret := os.Getenv("JWT_SECRET")
	originalExpiration := os.Getenv("JWT_EXPIRATION_HOURS")
	defer func() {
		if originalSecret != "" {
			os.Setenv("JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
		if originalExpiration != "" {
			os.Setenv("JWT_EXPIRATION_HOURS", originalExpiration)
		} else {
			os.Unsetenv("JWT_EXPIRATION_HOURS")
		}
	}()

	os.Setenv("JWT_SECRET", "realworld-test-secret-key-minimum-32-bytes-long")
	os.Setenv("JWT_EXPIRATION_HOURS", "24")

	cfg, err := config.NewJWTConfig()
	require.NoError(t, err)

	service := NewJWTService(cfg)
	userID := uuid.New()

	// Simulate login flow: generate token after authentication
	token, err := service.GenerateToken(userID)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Simulate request flow: validate token in middleware
	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)

	// Test token can be used multiple times (until expired)
	claims2, err := service.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims2.UserID)

	claims3, err := service.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims3.UserID)
}
