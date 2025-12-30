// Package middleware provides HTTP middleware for authentication and authorization.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// ContextKey is a typed key for context values to avoid collisions.
type ContextKey string

// userIDKey is the context key for storing the authenticated user ID.
const userIDKey ContextKey = "userID"

// TokenValidator is an interface for validating JWT tokens.
// This allows the middleware to work with any JWT service implementation.
type TokenValidator interface {
	ValidateToken(tokenString string) (UserIDGetter, error)
}

// UserIDGetter is an interface for extracting user ID from token claims.
type UserIDGetter interface {
	GetUserID() uuid.UUID
}

// AuthMiddleware creates middleware that validates JWT tokens and adds user ID to request context.
func AuthMiddleware(jwtService TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Parse Bearer token
			// Handle case-insensitive "Bearer" prefix
			parts := strings.Fields(authHeader)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimSpace(parts[1])
			if tokenString == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Validate token
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Extract user ID from claims
			userID := claims.GetUserID()

			// Add user ID to request context
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the authenticated user ID from the request context.
func GetUserID(r *http.Request) (uuid.UUID, error) {
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("user ID not found in request context")
	}
	return userID, nil
}

// UserIDKey returns the context key for user ID (for testing purposes).
func UserIDKey() ContextKey {
	return userIDKey
}
