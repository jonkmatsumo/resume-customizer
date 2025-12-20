package main

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

// TestMain runs before all tests and loads .env if available
func TestMain(m *testing.M) {
	// Try to load .env file - ignore error if it doesn't exist (CI environment)
	_ = godotenv.Load()

	// Run tests
	os.Exit(m.Run())
}
