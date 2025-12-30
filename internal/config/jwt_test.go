package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTConfig_DefaultValues(t *testing.T) {
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

	// Set required secret
	os.Setenv("JWT_SECRET", "test-secret-key")
	os.Unsetenv("JWT_EXPIRATION_HOURS")

	cfg, err := NewJWTConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "test-secret-key", cfg.Secret)
	assert.Equal(t, 24, cfg.ExpirationHours, "should use default expiration of 24 hours")
}

func TestNewJWTConfig_CustomExpiration(t *testing.T) {
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

	tests := []struct {
		name          string
		expiration    string
		expectedHours int
		wantErr       bool
		description   string
	}{
		{
			name:          "custom expiration 12 hours",
			expiration:    "12",
			expectedHours: 12,
			wantErr:       false,
			description:   "should accept custom expiration of 12 hours",
		},
		{
			name:          "custom expiration 48 hours",
			expiration:    "48",
			expectedHours: 48,
			wantErr:       false,
			description:   "should accept custom expiration of 48 hours",
		},
		{
			name:          "minimum expiration 1 hour",
			expiration:    "1",
			expectedHours: 1,
			wantErr:       false,
			description:   "should accept minimum expiration of 1 hour",
		},
		{
			name:          "large expiration",
			expiration:    "168", // 1 week
			expectedHours: 168,
			wantErr:       false,
			description:   "should accept large expiration values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("JWT_SECRET", "test-secret-key")
			os.Setenv("JWT_EXPIRATION_HOURS", tt.expiration)

			cfg, err := NewJWTConfig()
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				assert.Equal(t, "test-secret-key", cfg.Secret)
				assert.Equal(t, tt.expectedHours, cfg.ExpirationHours, tt.description)
			}
		})
	}
}

func TestNewJWTConfig_MissingSecret(t *testing.T) {
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

	tests := []struct {
		name        string
		secret      string
		wantErr     bool
		description string
	}{
		{
			name:        "empty secret",
			secret:      "",
			wantErr:     true,
			description: "should error when JWT_SECRET is empty",
		},
		{
			name:        "secret not set",
			secret:      "", // unset
			wantErr:     true,
			description: "should error when JWT_SECRET is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.secret == "" {
				os.Unsetenv("JWT_SECRET")
			} else {
				os.Setenv("JWT_SECRET", tt.secret)
			}

			cfg, err := NewJWTConfig()
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, cfg)
				assert.Contains(t, err.Error(), "JWT_SECRET")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestNewJWTConfig_InvalidExpiration(t *testing.T) {
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

	tests := []struct {
		name        string
		expiration  string
		wantErr     bool
		description string
	}{
		{
			name:        "non-numeric expiration",
			expiration:  "invalid",
			wantErr:     true,
			description: "should error when JWT_EXPIRATION_HOURS is non-numeric",
		},
		{
			name:        "zero expiration",
			expiration:  "0",
			wantErr:     true,
			description: "should error when JWT_EXPIRATION_HOURS is zero",
		},
		{
			name:        "negative expiration",
			expiration:  "-1",
			wantErr:     true,
			description: "should error when JWT_EXPIRATION_HOURS is negative",
		},
		{
			name:        "float expiration",
			expiration:  "12.5",
			wantErr:     true,
			description: "should error when JWT_EXPIRATION_HOURS is a float",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("JWT_SECRET", "test-secret-key")
			os.Setenv("JWT_EXPIRATION_HOURS", tt.expiration)

			cfg, err := NewJWTConfig()
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, cfg)
				assert.Contains(t, err.Error(), "JWT_EXPIRATION_HOURS")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestNewJWTConfig_EnvironmentVariableHandling(t *testing.T) {
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

	// Test that environment variables are read correctly
	os.Setenv("JWT_SECRET", "my-secret-key-123")
	os.Setenv("JWT_EXPIRATION_HOURS", "36")

	cfg, err := NewJWTConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "my-secret-key-123", cfg.Secret)
	assert.Equal(t, 36, cfg.ExpirationHours)
}
