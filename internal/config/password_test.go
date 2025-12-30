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
		// Boundary value tests
		{
			name:        "boundary cost 10",
			bcryptCost:  "10",
			pepper:      "",
			wantCost:    10,
			wantErr:     false,
			description: "should accept minimum valid cost 10",
		},
		{
			name:        "boundary cost 11",
			bcryptCost:  "11",
			pepper:      "",
			wantCost:    11,
			wantErr:     false,
			description: "should accept cost 11",
		},
		{
			name:        "boundary cost 13",
			bcryptCost:  "13",
			pepper:      "",
			wantCost:    13,
			wantErr:     false,
			description: "should accept cost 13",
		},
		{
			name:        "boundary cost 14",
			bcryptCost:  "14",
			pepper:      "",
			wantCost:    14,
			wantErr:     false,
			description: "should accept maximum valid cost 14",
		},
		{
			name:        "boundary cost 9 rejected",
			bcryptCost:  "9",
			pepper:      "",
			wantCost:    0,
			wantErr:     true,
			description: "should reject cost 9 (below minimum)",
		},
		{
			name:        "boundary cost 15 rejected",
			bcryptCost:  "15",
			pepper:      "",
			wantCost:    0,
			wantErr:     true,
			description: "should reject cost 15 (above maximum)",
		},
		{
			name:        "negative cost",
			bcryptCost:  "-5",
			pepper:      "",
			wantCost:    0,
			wantErr:     true,
			description: "should reject negative cost",
		},
		{
			name:        "zero cost",
			bcryptCost:  "0",
			pepper:      "",
			wantCost:    0,
			wantErr:     true,
			description: "should reject zero cost",
		},
		{
			name:        "float cost",
			bcryptCost:  "12.5",
			pepper:      "",
			wantCost:    0,
			wantErr:     true,
			description: "should reject float cost",
		},
		{
			name:        "empty string cost",
			bcryptCost:  "",
			pepper:      "",
			wantCost:    12,
			wantErr:     false,
			description: "should use default when cost is empty string",
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

// Test password edge cases
func TestPasswordConfig_EmptyPassword(t *testing.T) {
	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Empty password should hash successfully
	hash, err := config.HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword() with empty password should not error: %v", err)
	}

	if hash == "" {
		t.Error("HashPassword() should return a hash even for empty password")
	}

	// Empty password should verify
	if !config.VerifyPassword("", hash) {
		t.Error("VerifyPassword() should return true for empty password with correct hash")
	}

	// Non-empty password should not verify against empty password hash
	if config.VerifyPassword("not-empty", hash) {
		t.Error("VerifyPassword() should return false for non-empty password against empty password hash")
	}
}

func TestPasswordConfig_VeryLongPassword(t *testing.T) {
	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Test password approaching 72-byte limit (bcrypt's maximum)
	// 72 bytes = 72 ASCII characters
	longPassword := string(make([]byte, 70)) // 70 bytes
	for i := range longPassword {
		longPassword = longPassword[:i] + "a" + longPassword[i+1:]
	}

	hash, err := config.HashPassword(longPassword)
	if err != nil {
		t.Fatalf("HashPassword() with long password should not error: %v", err)
	}

	if !config.VerifyPassword(longPassword, hash) {
		t.Error("VerifyPassword() should work with long passwords")
	}
}

func TestPasswordConfig_PasswordExceeding72Bytes(t *testing.T) {
	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create password that exceeds 72 bytes
	veryLongPassword := string(make([]byte, 100))
	for i := range veryLongPassword {
		veryLongPassword = veryLongPassword[:i] + "a" + veryLongPassword[i+1:]
	}

	// Bcrypt errors when password exceeds 72 bytes (does not truncate)
	hash, err := config.HashPassword(veryLongPassword)
	if err == nil {
		t.Error("HashPassword() should error when password exceeds 72 bytes")
	}

	if hash != "" {
		t.Error("HashPassword() should return empty hash when password exceeds 72 bytes")
	}
}

// Test pepper-specific scenarios
func TestPasswordConfig_PepperLengthLimits(t *testing.T) {
	originalPepper := os.Getenv("PASSWORD_PEPPER")
	defer os.Setenv("PASSWORD_PEPPER", originalPepper)

	tests := []struct {
		name     string
		pepper   string
		password string
		wantErr  bool
	}{
		{
			name:     "32 byte pepper (recommended)",
			pepper:   "erop9LTNyViL9dRhkFvfVpvT4zasc/DGTkKIikjV3YE=", // 32 bytes base64
			password: "test123",
			wantErr:  false,
		},
		{
			name:     "64 byte pepper exceeds limit",
			pepper:   "erop9LTNyViL9dRhkFvfVpvT4zasc/DGTkKIikjV3YE=erop9LTNyViL9dRhkFvfVpvT4zasc/DGTkKIikjV3YE=",
			password: "test123",
			wantErr:  true, // 64 bytes + 7 bytes = 71 bytes, but base64 is longer
		},
		{
			name: "pepper at 72-byte limit",
			pepper: func() string {
				s := make([]byte, 63)
				for i := range s {
					s[i] = 'a'
				}
				return string(s)
			}(), // 63 bytes + 9 byte password = 72 bytes exactly
			password: "test12345",
			wantErr:  false,
		},
		{
			name: "pepper exceeding 72-byte limit",
			pepper: func() string {
				s := make([]byte, 64)
				for i := range s {
					s[i] = 'a'
				}
				return string(s)
			}(), // 64 bytes + 9 byte password = 73 bytes (exceeds limit)
			password: "test12345",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("PASSWORD_PEPPER", tt.pepper)
			config, err := NewPasswordConfig()
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			hash, err := config.HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !config.VerifyPassword(tt.password, hash) {
					t.Error("VerifyPassword() should work with pepper")
				}
			}
		})
	}
}

func TestPasswordConfig_PepperRotation(t *testing.T) {
	originalPepper := os.Getenv("PASSWORD_PEPPER")
	defer os.Setenv("PASSWORD_PEPPER", originalPepper)

	password := "test-password-123"
	oldPepper := "old-pepper-123"
	newPepper := "new-pepper-456"

	// Hash with old pepper
	os.Setenv("PASSWORD_PEPPER", oldPepper)
	configOld, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config with old pepper: %v", err)
	}

	oldHash, err := configOld.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() with old pepper failed: %v", err)
	}

	// Verify with old pepper (should work)
	if !configOld.VerifyPassword(password, oldHash) {
		t.Error("VerifyPassword() should work with old pepper")
	}

	// Switch to new pepper
	os.Setenv("PASSWORD_PEPPER", newPepper)
	configNew, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config with new pepper: %v", err)
	}

	// Old hash should NOT verify with new pepper
	if configNew.VerifyPassword(password, oldHash) {
		t.Error("VerifyPassword() should fail when pepper changes")
	}

	// New hash with new pepper should verify
	newHash, err := configNew.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() with new pepper failed: %v", err)
	}

	if !configNew.VerifyPassword(password, newHash) {
		t.Error("VerifyPassword() should work with new pepper")
	}
}

func TestPasswordConfig_DifferentPepperValues(t *testing.T) {
	originalPepper := os.Getenv("PASSWORD_PEPPER")
	defer os.Setenv("PASSWORD_PEPPER", originalPepper)

	password := "test-password"

	pepper1 := "pepper-one"
	pepper2 := "pepper-two"

	// Hash with pepper1
	os.Setenv("PASSWORD_PEPPER", pepper1)
	config1, _ := NewPasswordConfig()
	hash1, _ := config1.HashPassword(password)

	// Hash with pepper2
	os.Setenv("PASSWORD_PEPPER", pepper2)
	config2, _ := NewPasswordConfig()
	hash2, _ := config2.HashPassword(password)

	// Hashes should be different
	if hash1 == hash2 {
		t.Error("Hashes with different peppers should be different")
	}

	// Each hash should only verify with its own pepper
	if !config1.VerifyPassword(password, hash1) {
		t.Error("Hash1 should verify with pepper1")
	}

	if config1.VerifyPassword(password, hash2) {
		t.Error("Hash2 should not verify with pepper1")
	}

	if !config2.VerifyPassword(password, hash2) {
		t.Error("Hash2 should verify with pepper2")
	}

	if config2.VerifyPassword(password, hash1) {
		t.Error("Hash1 should not verify with pepper2")
	}
}

// Test error handling
func TestPasswordConfig_ErrorScenarios(t *testing.T) {
	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Test empty hash verification
	if config.VerifyPassword("any-password", "") {
		t.Error("VerifyPassword() should return false for empty hash")
	}

	// Test malformed hash strings
	malformedHashes := []string{
		"not-a-hash",
		"$2a$12$invalid",
		"$2a$12$tooshort",
		"$2a$12$" + string(make([]byte, 100)), // Too long
		"invalid$format",
	}

	for _, malformed := range malformedHashes {
		if config.VerifyPassword("test", malformed) {
			t.Errorf("VerifyPassword() should return false for malformed hash: %s", malformed)
		}
	}
}

func TestPasswordConfig_InvalidCostErrors(t *testing.T) {
	originalCost := os.Getenv("BCRYPT_COST")
	defer os.Setenv("BCRYPT_COST", originalCost)

	invalidCosts := []string{
		"9",    // Below minimum
		"15",   // Above maximum
		"-1",   // Negative
		"0",    // Zero
		"abc",  // Non-numeric
		"12.5", // Float
		"",     // Empty (should use default, not error)
	}

	for _, cost := range invalidCosts {
		t.Run("cost_"+cost, func(t *testing.T) {
			if cost == "" {
				os.Unsetenv("BCRYPT_COST")
			} else {
				os.Setenv("BCRYPT_COST", cost)
			}

			config, err := NewPasswordConfig()
			if cost == "" {
				// Empty should use default, not error
				if err != nil {
					t.Errorf("NewPasswordConfig() with empty cost should use default, got error: %v", err)
				}
				if config == nil || config.BcryptCost != 12 {
					t.Errorf("NewPasswordConfig() with empty cost should default to 12, got: %v", config)
				}
			} else {
				// All other invalid costs should error
				if err == nil {
					t.Errorf("NewPasswordConfig() with invalid cost %s should error, got: %v", cost, config)
				}
			}
		})
	}
}

// Test security properties
func TestPasswordConfig_SaltUniqueness(t *testing.T) {
	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	password := "test-password-123"
	hashes := make(map[string]bool)
	const iterations = 100

	// Generate many hashes of the same password
	for i := 0; i < iterations; i++ {
		hash, err := config.HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword() failed on iteration %d: %v", i, err)
		}

		// Each hash should be unique (due to salt)
		if hashes[hash] {
			t.Errorf("Duplicate hash found at iteration %d - salt is not unique", i)
		}
		hashes[hash] = true

		// All hashes should verify the same password
		if !config.VerifyPassword(password, hash) {
			t.Errorf("Hash at iteration %d does not verify correctly", i)
		}
	}

	// All hashes should be different
	if len(hashes) != iterations {
		t.Errorf("Expected %d unique hashes, got %d", iterations, len(hashes))
	}
}

func TestPasswordConfig_HashUniqueness(t *testing.T) {
	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	passwords := []string{
		"password1",
		"password2",
		"password3",
		"different",
		"test",
		"",
	}

	hashes := make(map[string]string)

	// Hash each password multiple times
	for _, password := range passwords {
		for i := 0; i < 10; i++ {
			hash, err := config.HashPassword(password)
			if err != nil {
				t.Fatalf("HashPassword() failed for password %s: %v", password, err)
			}

			// Verify this hash works for this password
			if !config.VerifyPassword(password, hash) {
				t.Errorf("Hash does not verify for password: %s", password)
			}

			// Store first hash for each password
			if i == 0 {
				hashes[password] = hash
			}
		}
	}

	// Verify different passwords produce different hashes (very high probability)
	// Note: There's a tiny chance of collision, but it's negligible
	for p1, h1 := range hashes {
		for p2, h2 := range hashes {
			if p1 != p2 && h1 == h2 {
				t.Errorf("Different passwords produced same hash: %s and %s", p1, p2)
			}
		}
	}
}

func TestPasswordConfig_DifferentPasswordsDifferentHashes(t *testing.T) {
	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	password1 := "password-one"
	password2 := "password-two"

	hash1, err := config.HashPassword(password1)
	if err != nil {
		t.Fatalf("HashPassword() failed for password1: %v", err)
	}

	hash2, err := config.HashPassword(password2)
	if err != nil {
		t.Fatalf("HashPassword() failed for password2: %v", err)
	}

	// Hashes should be different
	if hash1 == hash2 {
		t.Error("Different passwords should produce different hashes")
	}

	// Each password should only verify with its own hash
	if !config.VerifyPassword(password1, hash1) {
		t.Error("Password1 should verify with hash1")
	}

	if config.VerifyPassword(password1, hash2) {
		t.Error("Password1 should not verify with hash2")
	}

	if !config.VerifyPassword(password2, hash2) {
		t.Error("Password2 should verify with hash2")
	}

	if config.VerifyPassword(password2, hash1) {
		t.Error("Password2 should not verify with hash1")
	}
}

// Test integration scenarios
func TestPasswordConfig_EnvironmentVariableHandling(t *testing.T) {
	originalCost := os.Getenv("BCRYPT_COST")
	originalPepper := os.Getenv("PASSWORD_PEPPER")
	defer func() {
		os.Setenv("BCRYPT_COST", originalCost)
		os.Setenv("PASSWORD_PEPPER", originalPepper)
	}()

	// Test with missing environment variables
	os.Unsetenv("BCRYPT_COST")
	os.Unsetenv("PASSWORD_PEPPER")

	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("NewPasswordConfig() should work with missing env vars: %v", err)
	}

	if config.BcryptCost != 12 {
		t.Errorf("Expected default cost 12, got %d", config.BcryptCost)
	}

	if config.Pepper != "" {
		t.Errorf("Expected empty pepper, got %s", config.Pepper)
	}

	// Test with whitespace in environment variables
	// strconv.Atoi does not trim whitespace, so this should error
	os.Setenv("BCRYPT_COST", "  12  ")
	config2, err := NewPasswordConfig()
	if err == nil {
		t.Error("BCRYPT_COST with whitespace should error (strconv.Atoi doesn't trim)")
	}
	if config2 != nil {
		t.Error("NewPasswordConfig() should return nil when there's an error")
	}
}

func TestPasswordConfig_ConcurrentAccess(t *testing.T) {
	config, err := NewPasswordConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	password := "test-password"
	done := make(chan bool, 10)

	// Concurrently hash the same password
	for i := 0; i < 10; i++ {
		go func() {
			hash, err := config.HashPassword(password)
			if err != nil {
				t.Errorf("HashPassword() failed in goroutine: %v", err)
				done <- false
				return
			}

			if !config.VerifyPassword(password, hash) {
				t.Error("VerifyPassword() failed in goroutine")
				done <- false
				return
			}

			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		if !<-done {
			t.Fail()
		}
	}
}

func TestPasswordConfig_ConfigurationPersistence(t *testing.T) {
	originalCost := os.Getenv("BCRYPT_COST")
	originalPepper := os.Getenv("PASSWORD_PEPPER")
	defer func() {
		os.Setenv("BCRYPT_COST", originalCost)
		os.Setenv("PASSWORD_PEPPER", originalPepper)
	}()

	os.Setenv("BCRYPT_COST", "11")
	os.Setenv("PASSWORD_PEPPER", "test-pepper")

	// Create multiple configs
	config1, err1 := NewPasswordConfig()
	config2, err2 := NewPasswordConfig()
	config3, err3 := NewPasswordConfig()

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatalf("Failed to create configs: %v, %v, %v", err1, err2, err3)
	}

	// All configs should have same values
	if config1.BcryptCost != config2.BcryptCost || config2.BcryptCost != config3.BcryptCost {
		t.Error("Configs should have consistent BcryptCost")
	}

	if config1.Pepper != config2.Pepper || config2.Pepper != config3.Pepper {
		t.Error("Configs should have consistent Pepper")
	}

	if config1.BcryptCost != 11 {
		t.Errorf("Expected BcryptCost 11, got %d", config1.BcryptCost)
	}

	if config1.Pepper != "test-pepper" {
		t.Errorf("Expected Pepper 'test-pepper', got '%s'", config1.Pepper)
	}
}

// Performance benchmarks
func BenchmarkHashPassword_Cost10(b *testing.B) {
	originalCost := os.Getenv("BCRYPT_COST")
	defer os.Setenv("BCRYPT_COST", originalCost)

	os.Setenv("BCRYPT_COST", "10")
	config, _ := NewPasswordConfig()
	password := "benchmark-password-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = config.HashPassword(password)
	}
}

func BenchmarkHashPassword_Cost12(b *testing.B) {
	originalCost := os.Getenv("BCRYPT_COST")
	defer os.Setenv("BCRYPT_COST", originalCost)

	os.Setenv("BCRYPT_COST", "12")
	config, _ := NewPasswordConfig()
	password := "benchmark-password-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = config.HashPassword(password)
	}
}

func BenchmarkHashPassword_Cost14(b *testing.B) {
	originalCost := os.Getenv("BCRYPT_COST")
	defer os.Setenv("BCRYPT_COST", originalCost)

	os.Setenv("BCRYPT_COST", "14")
	config, _ := NewPasswordConfig()
	password := "benchmark-password-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = config.HashPassword(password)
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	config, _ := NewPasswordConfig()
	password := "benchmark-password-123"
	hash, _ := config.HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.VerifyPassword(password, hash)
	}
}

func BenchmarkVerifyPassword_WithPepper(b *testing.B) {
	originalPepper := os.Getenv("PASSWORD_PEPPER")
	defer os.Setenv("PASSWORD_PEPPER", originalPepper)

	os.Setenv("PASSWORD_PEPPER", "benchmark-pepper-123")
	config, _ := NewPasswordConfig()
	password := "benchmark-password-123"
	hash, _ := config.HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.VerifyPassword(password, hash)
	}
}
