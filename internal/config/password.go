// Package config provides password configuration and hashing functionality.
package config

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

// PasswordConfig holds configuration for password hashing and verification.
type PasswordConfig struct {
	BcryptCost int
	Pepper     string // optional global secret for additional security
}

// NewPasswordConfig creates a new password configuration from environment variables.
// It reads BCRYPT_COST (default: 12) and optionally PASSWORD_PEPPER.
func NewPasswordConfig() (*PasswordConfig, error) {
	costStr := os.Getenv("BCRYPT_COST")
	if costStr == "" {
		costStr = "12" // default
	}

	cost, err := strconv.Atoi(costStr)
	if err != nil {
		return nil, fmt.Errorf("invalid BCRYPT_COST: %v", err)
	}

	config := &PasswordConfig{
		BcryptCost: cost,
		Pepper:     os.Getenv("PASSWORD_PEPPER"), // empty if not set
	}

	if err := config.normalize(); err != nil {
		return nil, err
	}

	return config, nil
}

// normalize validates the configuration.
func (c *PasswordConfig) normalize() error {
	if c.BcryptCost < 10 || c.BcryptCost > 14 {
		return fmt.Errorf("bcrypt cost out of range: %d (must be 10-14)", c.BcryptCost)
	}
	return nil
}

// HashPassword hashes a password using bcrypt (with optional pepper).
func (c *PasswordConfig) HashPassword(pw string) (string, error) {
	password := pw
	if c.Pepper != "" {
		password = pw + c.Pepper
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), c.BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword verifies a password against a stored hash (with optional pepper).
func (c *PasswordConfig) VerifyPassword(pw, storedHash string) bool {
	password := pw
	if c.Pepper != "" {
		password = pw + c.Pepper
	}

	err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	return err == nil
}

