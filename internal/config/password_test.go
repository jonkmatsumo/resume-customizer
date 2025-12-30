package config

import (
	"os"
	"testing"
)

func TestNewPasswordConfig(t *testing.T) {
	tests := []struct {
		name        string
		bcryptCost  string
		pepper      string
		wantCost    int
		wantErr     bool
		description string
	}{
		{
			name:        "default cost",
			bcryptCost:  "",
			pepper:      "",
			wantCost:    12,
			wantErr:     false,
			description: "should use default cost of 12 when BCRYPT_COST is not set",
		},
		{
			name:        "valid cost",
			bcryptCost:  "12",
			pepper:      "",
			wantCost:    12,
			wantErr:     false,
			description: "should accept valid cost",
		},
		{
			name:        "cost too low",
			bcryptCost:  "9",
			pepper:      "",
			wantCost:    0,
			wantErr:     true,
			description: "should reject cost below 10",
		},
		{
			name:        "cost too high",
			bcryptCost:  "15",
			pepper:      "",
			wantCost:    0,
			wantErr:     true,
			description: "should reject cost above 14",
		},
		{
			name:        "invalid cost",
			bcryptCost:  "invalid",
			pepper:      "",
			wantCost:    0,
			wantErr:     true,
			description: "should reject non-numeric cost",
		},
		{
			name:        "with pepper",
			bcryptCost:  "12",
			pepper:      "test-pepper",
			wantCost:    12,
			wantErr:     false,
			description: "should accept optional pepper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			originalCost := os.Getenv("BCRYPT_COST")
			originalPepper := os.Getenv("PASSWORD_PEPPER")
			defer func() {
				os.Setenv("BCRYPT_COST", originalCost)
				os.Setenv("PASSWORD_PEPPER", originalPepper)
			}()

			// Set test values
			if tt.bcryptCost != "" {
				os.Setenv("BCRYPT_COST", tt.bcryptCost)
			} else {
				os.Unsetenv("BCRYPT_COST")
			}
			if tt.pepper != "" {
				os.Setenv("PASSWORD_PEPPER", tt.pepper)
			} else {
				os.Unsetenv("PASSWORD_PEPPER")
			}

			config, err := NewPasswordConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPasswordConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if config.BcryptCost != tt.wantCost {
					t.Errorf("NewPasswordConfig() BcryptCost = %v, want %v", config.BcryptCost, tt.wantCost)
				}
				if tt.pepper != "" && config.Pepper != tt.pepper {
					t.Errorf("NewPasswordConfig() Pepper = %v, want %v", config.Pepper, tt.pepper)
				}
			}
		})
	}
}

func TestPasswordConfig_HashPassword(t *testing.T) {
	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	password := "test-password-123"
	hash, err := config.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Error("HashPassword() returned empty hash")
	}

	// Hash should be different each time (bcrypt includes salt)
	hash2, err := config.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == hash2 {
		t.Error("HashPassword() should produce different hashes for same password (salt)")
	}
}

func TestPasswordConfig_VerifyPassword(t *testing.T) {
	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	password := "test-password-123"
	hash, err := config.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// Correct password should verify
	if !config.VerifyPassword(password, hash) {
		t.Error("VerifyPassword() should return true for correct password")
	}

	// Wrong password should not verify
	if config.VerifyPassword("wrong-password", hash) {
		t.Error("VerifyPassword() should return false for incorrect password")
	}
}

func TestPasswordConfig_VerifyPassword_WithPepper(t *testing.T) {
	// Save original pepper
	originalPepper := os.Getenv("PASSWORD_PEPPER")
	defer os.Setenv("PASSWORD_PEPPER", originalPepper)

	// Set pepper
	os.Setenv("PASSWORD_PEPPER", "test-pepper-123")

	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	if config.Pepper != "test-pepper-123" {
		t.Fatalf("Config pepper = %v, want test-pepper-123", config.Pepper)
	}

	password := "test-password-123"
	hash, err := config.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// Correct password should verify
	if !config.VerifyPassword(password, hash) {
		t.Error("VerifyPassword() should return true for correct password with pepper")
	}

	// Wrong password should not verify
	if config.VerifyPassword("wrong-password", hash) {
		t.Error("VerifyPassword() should return false for incorrect password with pepper")
	}

	// Password without pepper should not verify
	os.Unsetenv("PASSWORD_PEPPER")
	configNoPepper, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config without pepper: %v", err)
	}

	if configNoPepper.VerifyPassword(password, hash) {
		t.Error("VerifyPassword() should return false when pepper is removed")
	}
}

