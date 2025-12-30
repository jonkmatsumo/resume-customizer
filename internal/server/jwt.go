// Package server provides the HTTP REST API for the resume customizer.
package server

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/jonathan/resume-customizer/internal/server/middleware"
)

// Claims represents JWT claims with user ID.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// GetUserID returns the user ID from the claims.
// This implements the middleware.UserIDGetter interface.
func (c *Claims) GetUserID() uuid.UUID {
	return c.UserID
}

// AsTokenValidator returns a TokenValidator adapter for this JWTService.
// This allows the JWTService to be used with middleware without creating import cycles.
func (s *JWTService) AsTokenValidator() middleware.TokenValidator {
	return &jwtServiceValidator{service: s}
}

// jwtServiceValidator adapts JWTService to middleware.TokenValidator interface.
type jwtServiceValidator struct {
	service *JWTService
}

func (v *jwtServiceValidator) ValidateToken(tokenString string) (middleware.UserIDGetter, error) {
	claims, err := v.service.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// JWTService provides JWT token generation and validation functionality.
type JWTService struct {
	config *config.JWTConfig
}

// NewJWTService creates a new JWT service with the given configuration.
func NewJWTService(cfg *config.JWTConfig) *JWTService {
	return &JWTService{
		config: cfg,
	}
}

// GenerateToken generates a JWT token for the given user ID.
func (s *JWTService) GenerateToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(s.config.ExpirationHours) * time.Hour)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims.
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token string is empty")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.Secret), nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return nil, fmt.Errorf("invalid token signature: %w", err)
		}
		if err == jwt.ErrTokenExpired {
			return nil, fmt.Errorf("token expired: %w", err)
		}
		if err == jwt.ErrTokenMalformed {
			return nil, fmt.Errorf("malformed token: %w", err)
		}
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	return claims, nil
}
