// Package server provides the HTTP REST API for the resume customizer.
package server

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/types"
)

// UserService provides business logic for user authentication operations
type UserService struct {
	db             DBClient
	passwordConfig *config.PasswordConfig
}

// NewUserService creates a new UserService with the given dependencies
func NewUserService(db DBClient, passwordConfig *config.PasswordConfig) *UserService {
	return &UserService{
		db:             db,
		passwordConfig: passwordConfig,
	}
}

// convertDBUserToTypesUser converts db.User to types.User, excluding password hash
func convertDBUserToTypesUser(dbUser *db.User) *types.User {
	if dbUser == nil {
		return nil
	}
	return &types.User{
		ID:          dbUser.ID,
		Name:        dbUser.Name,
		Email:       dbUser.Email,
		Phone:       dbUser.Phone,
		PasswordSet: dbUser.PasswordSet,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
	}
}

// Register creates a new user with password authentication
func (s *UserService) Register(ctx context.Context, req *types.CreateUserRequest) (*types.User, error) {
	// Check if email already exists
	exists, err := s.db.CheckEmailExists(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, &ErrEmailAlreadyExists{Email: req.Email}
	}

	// Hash password
	passwordHash, err := s.passwordConfig.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user (two-step: create user, then set password)
	userID, err := s.db.CreateUser(ctx, req.Name, req.Email, req.Phone)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Set password
	err = s.db.UpdatePassword(ctx, userID, passwordHash)
	if err != nil {
		// If password update fails, we should clean up the user
		// For now, just return error (in production, consider transaction or cleanup)
		return nil, fmt.Errorf("failed to set password: %w", err)
	}

	// Retrieve created user
	dbUser, err := s.db.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created user: %w", err)
	}
	if dbUser == nil {
		return nil, fmt.Errorf("created user not found: %s", userID)
	}

	// Convert and return (password hash excluded)
	return convertDBUserToTypesUser(dbUser), nil
}

// Login authenticates a user and returns user data
func (s *UserService) Login(ctx context.Context, req *types.LoginRequest) (*types.User, error) {
	// Get user by email
	dbUser, err := s.db.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	// Security: Always return generic error if user not found or password wrong
	if dbUser == nil {
		return nil, &ErrInvalidCredentials{}
	}

	// Verify password
	if !s.passwordConfig.VerifyPassword(req.Password, dbUser.PasswordHash) {
		return nil, &ErrInvalidCredentials{}
	}

	// Check if password is set (for migration scenarios)
	if !dbUser.PasswordSet {
		return nil, &ErrInvalidCredentials{}
	}

	// Convert and return (password hash excluded)
	return convertDBUserToTypesUser(dbUser), nil
}

// UpdatePassword updates a user's password
func (s *UserService) UpdatePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	// Get user by ID
	dbUser, err := s.db.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if dbUser == nil {
		return &ErrUserNotFound{UserID: userID}
	}

	// Verify current password
	if !s.passwordConfig.VerifyPassword(currentPassword, dbUser.PasswordHash) {
		return &ErrPasswordMismatch{}
	}

	// Hash new password
	newPasswordHash, err := s.passwordConfig.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password in database
	err = s.db.UpdatePassword(ctx, userID, newPasswordHash)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
