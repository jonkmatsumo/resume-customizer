package server

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/config"
)

func BenchmarkPasswordHashing_Cost10(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	passwordConfig := &config.PasswordConfig{
		BcryptCost: 10,
		Pepper:     "",
	}

	password := "testpassword123"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := passwordConfig.HashPassword(password)
		if err != nil {
			b.Fatalf("Failed to hash password: %v", err)
		}
	}
}

func BenchmarkPasswordHashing_Cost12(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	passwordConfig := &config.PasswordConfig{
		BcryptCost: 12,
		Pepper:     "",
	}

	password := "testpassword123"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := passwordConfig.HashPassword(password)
		if err != nil {
			b.Fatalf("Failed to hash password: %v", err)
		}
	}
}

func BenchmarkPasswordHashing_WithPepper(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	passwordConfig := &config.PasswordConfig{
		BcryptCost: 10,
		Pepper:     "test-pepper-32-bytes-long-enough",
	}

	password := "testpassword123"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := passwordConfig.HashPassword(password)
		if err != nil {
			b.Fatalf("Failed to hash password: %v", err)
		}
	}
}

func BenchmarkPasswordVerification(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	passwordConfig := &config.PasswordConfig{
		BcryptCost: 10,
		Pepper:     "",
	}

	password := "testpassword123"
	hash, err := passwordConfig.HashPassword(password)
	if err != nil {
		b.Fatalf("Failed to hash password: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = passwordConfig.VerifyPassword(password, hash)
	}
}

func BenchmarkTokenGeneration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	jwtConfig := &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing-minimum-32-bytes",
		ExpirationHours: 24,
	}

	jwtSvc := NewJWTService(jwtConfig)
	userID := uuid.New()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := jwtSvc.GenerateToken(userID)
		if err != nil {
			b.Fatalf("Failed to generate token: %v", err)
		}
	}
}

func BenchmarkTokenValidation(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	jwtConfig := &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing-minimum-32-bytes",
		ExpirationHours: 24,
	}

	jwtSvc := NewJWTService(jwtConfig)
	userID := uuid.New()

	token, err := jwtSvc.GenerateToken(userID)
	if err != nil {
		b.Fatalf("Failed to generate token: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := jwtSvc.ValidateToken(token)
		if err != nil {
			b.Fatalf("Failed to validate token: %v", err)
		}
	}
}

func BenchmarkConcurrentPasswordHashing(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	passwordConfig := &config.PasswordConfig{
		BcryptCost: 10,
		Pepper:     "",
	}

	password := "testpassword123"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := passwordConfig.HashPassword(password)
			if err != nil {
				b.Fatalf("Failed to hash password: %v", err)
			}
		}
	})
}

func BenchmarkConcurrentTokenGeneration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	jwtConfig := &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing-minimum-32-bytes",
		ExpirationHours: 24,
	}

	jwtSvc := NewJWTService(jwtConfig)
	userID := uuid.New()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := jwtSvc.GenerateToken(userID)
			if err != nil {
				b.Fatalf("Failed to generate token: %v", err)
			}
		}
	})
}

func BenchmarkConcurrentTokenValidation(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	jwtConfig := &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing-minimum-32-bytes",
		ExpirationHours: 24,
	}

	jwtSvc := NewJWTService(jwtConfig)
	userID := uuid.New()

	token, err := jwtSvc.GenerateToken(userID)
	if err != nil {
		b.Fatalf("Failed to generate token: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := jwtSvc.ValidateToken(token)
			if err != nil {
				b.Fatalf("Failed to validate token: %v", err)
			}
		}
	})
}
