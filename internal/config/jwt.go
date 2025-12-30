// Package config provides JWT configuration functionality.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// JWTConfig holds configuration for JWT token generation and validation.
type JWTConfig struct {
	Secret          string
	ExpirationHours int
}

// NewJWTConfig creates a new JWT configuration from environment variables.
// It reads JWT_SECRET (required) and JWT_EXPIRATION_HOURS (default: 24).
func NewJWTConfig() (*JWTConfig, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required but not set")
	}

	expirationStr := os.Getenv("JWT_EXPIRATION_HOURS")
	if expirationStr == "" {
		expirationStr = "24" // default
	}

	expirationHours, err := strconv.Atoi(expirationStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRATION_HOURS: %v", err)
	}

	config := &JWTConfig{
		Secret:          secret,
		ExpirationHours: expirationHours,
	}

	if err := config.normalize(); err != nil {
		return nil, err
	}

	return config, nil
}

// normalize validates the configuration.
func (c *JWTConfig) normalize() error {
	if c.Secret == "" {
		return fmt.Errorf("JWT_SECRET cannot be empty")
	}
	if c.ExpirationHours < 1 {
		return fmt.Errorf("JWT_EXPIRATION_HOURS must be at least 1 hour, got: %d", c.ExpirationHours)
	}
	return nil
}
