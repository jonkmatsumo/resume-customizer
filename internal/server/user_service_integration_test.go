package server

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB connects to the local DB for integration testing
func setupTestDB(t *testing.T) *db.DB {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Default to local docker connection
		dbURL = "postgres://resume:resume_dev@localhost:5432/resume_customizer?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping integration test: failed to connect to DB: %v", err)
	}
	return database
}

// setupPasswordConfig creates a password config for testing
func setupPasswordConfig(t *testing.T) *config.PasswordConfig {
	cfg, err := config.NewPasswordConfig()
	require.NoError(t, err)
	return cfg
}

func TestIntegration_UserService_Register(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()
	passwordConfig := setupPasswordConfig(t)
	service := NewUserService(database, passwordConfig)
	ctx := context.Background()

	// Test successful registration
	req := &types.CreateUserRequest{
		Name:     "Test User Register",
		Email:    "test-register-" + uuid.New().String() + "@example.com",
		Password: "password123",
		Phone:    "555-0100",
	}

	user, err := service.Register(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, user)
	defer database.DeleteUser(ctx, user.ID)

	assert.Equal(t, req.Name, user.Name)
	assert.Equal(t, req.Email, user.Email)
	assert.Equal(t, req.Phone, user.Phone)
	assert.True(t, user.PasswordSet)

	// Verify user exists in database
	dbUser, err := database.GetUser(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, dbUser)
	assert.NotEmpty(t, dbUser.PasswordHash)
	assert.NotEqual(t, req.Password, dbUser.PasswordHash) // Password should be hashed
	assert.True(t, dbUser.PasswordSet)

	// Verify password can be verified
	assert.True(t, passwordConfig.VerifyPassword(req.Password, dbUser.PasswordHash))

	// Test duplicate email registration fails
	duplicateReq := &types.CreateUserRequest{
		Name:     "Another User",
		Email:    req.Email, // Same email
		Password: "password456",
	}

	duplicateUser, err := service.Register(ctx, duplicateReq)
	assert.Nil(t, duplicateUser)
	require.Error(t, err)
	assert.IsType(t, &ErrEmailAlreadyExists{}, err)
}

func TestIntegration_UserService_Login(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()
	passwordConfig := setupPasswordConfig(t)
	service := NewUserService(database, passwordConfig)
	ctx := context.Background()

	// Register a user first
	registerReq := &types.CreateUserRequest{
		Name:     "Test User Login",
		Email:    "test-login-" + uuid.New().String() + "@example.com",
		Password: "password123",
		Phone:    "555-0100",
	}

	registeredUser, err := service.Register(ctx, registerReq)
	require.NoError(t, err)
	require.NotNil(t, registeredUser)
	defer database.DeleteUser(ctx, registeredUser.ID)

	// Test successful login
	loginReq := &types.LoginRequest{
		Email:    registerReq.Email,
		Password: registerReq.Password,
	}

	user, err := service.Login(ctx, loginReq)
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, registeredUser.ID, user.ID)
	assert.Equal(t, registeredUser.Email, user.Email)
	// Password hash should not be in response (types.User doesn't have that field)

	// Test login with wrong password
	wrongPasswordReq := &types.LoginRequest{
		Email:    registerReq.Email,
		Password: "wrongpassword",
	}

	wrongUser, err := service.Login(ctx, wrongPasswordReq)
	assert.Nil(t, wrongUser)
	require.Error(t, err)
	assert.IsType(t, &ErrInvalidCredentials{}, err)
	assert.Equal(t, "invalid email or password", err.Error())

	// Test login with non-existent email
	nonexistentReq := &types.LoginRequest{
		Email:    "nonexistent-" + uuid.New().String() + "@example.com",
		Password: "password123",
	}

	nonexistentUser, err := service.Login(ctx, nonexistentReq)
	assert.Nil(t, nonexistentUser)
	require.Error(t, err)
	assert.IsType(t, &ErrInvalidCredentials{}, err) // Generic error (security)
	assert.Equal(t, "invalid email or password", err.Error())
}

func TestIntegration_UserService_UpdatePassword(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()
	passwordConfig := setupPasswordConfig(t)
	service := NewUserService(database, passwordConfig)
	ctx := context.Background()

	// Register a user first
	registerReq := &types.CreateUserRequest{
		Name:     "Test User Update",
		Email:    "test-update-" + uuid.New().String() + "@example.com",
		Password: "oldpassword123",
		Phone:    "555-0100",
	}

	registeredUser, err := service.Register(ctx, registerReq)
	require.NoError(t, err)
	require.NotNil(t, registeredUser)
	defer database.DeleteUser(ctx, registeredUser.ID)

	// Get initial password hash
	dbUserBefore, err := database.GetUser(ctx, registeredUser.ID)
	require.NoError(t, err)
	require.NotNil(t, dbUserBefore)
	oldPasswordHash := dbUserBefore.PasswordHash

	// Update password
	err = service.UpdatePassword(ctx, registeredUser.ID, "oldpassword123", "newpassword456")
	require.NoError(t, err)

	// Verify old password no longer works
	oldLoginReq := &types.LoginRequest{
		Email:    registerReq.Email,
		Password: "oldpassword123",
	}
	oldUser, err := service.Login(ctx, oldLoginReq)
	assert.Nil(t, oldUser)
	require.Error(t, err)
	assert.IsType(t, &ErrInvalidCredentials{}, err)

	// Verify new password works
	newLoginReq := &types.LoginRequest{
		Email:    registerReq.Email,
		Password: "newpassword456",
	}
	newUser, err := service.Login(ctx, newLoginReq)
	require.NoError(t, err)
	require.NotNil(t, newUser)
	assert.Equal(t, registeredUser.ID, newUser.ID)

	// Verify password hash is updated in database
	dbUserAfter, err := database.GetUser(ctx, registeredUser.ID)
	require.NoError(t, err)
	require.NotNil(t, dbUserAfter)
	assert.NotEqual(t, oldPasswordHash, dbUserAfter.PasswordHash)
	assert.True(t, passwordConfig.VerifyPassword("newpassword456", dbUserAfter.PasswordHash))

	// Test wrong current password
	err = service.UpdatePassword(ctx, registeredUser.ID, "wrongcurrent", "newpassword789")
	require.Error(t, err)
	assert.IsType(t, &ErrPasswordMismatch{}, err)

	// Test user not found
	nonExistentID := uuid.New()
	err = service.UpdatePassword(ctx, nonExistentID, "current", "new")
	require.Error(t, err)
	assert.IsType(t, &ErrUserNotFound{}, err)
}

func TestIntegration_UserService_PasswordPepper(t *testing.T) {
	// Skip if pepper not configured
	pepper := os.Getenv("PASSWORD_PEPPER")
	if pepper == "" {
		t.Skip("Skipping pepper test: PASSWORD_PEPPER not set")
	}

	database := setupTestDB(t)
	defer database.Close()
	passwordConfig := setupPasswordConfig(t)
	service := NewUserService(database, passwordConfig)
	ctx := context.Background()

	// Register user with pepper
	registerReq := &types.CreateUserRequest{
		Name:     "Test User Pepper",
		Email:    "test-pepper-" + uuid.New().String() + "@example.com",
		Password: "password123",
	}

	registeredUser, err := service.Register(ctx, registerReq)
	require.NoError(t, err)
	require.NotNil(t, registeredUser)
	defer database.DeleteUser(ctx, registeredUser.ID)

	// Verify login works with pepper
	loginReq := &types.LoginRequest{
		Email:    registerReq.Email,
		Password: registerReq.Password,
	}

	user, err := service.Login(ctx, loginReq)
	require.NoError(t, err)
	assert.NotNil(t, user)

	// Update password with pepper
	err = service.UpdatePassword(ctx, registeredUser.ID, "password123", "newpassword456")
	require.NoError(t, err)

	// Verify new password works
	newLoginReq := &types.LoginRequest{
		Email:    registerReq.Email,
		Password: "newpassword456",
	}
	newUser, err := service.Login(ctx, newLoginReq)
	require.NoError(t, err)
	assert.NotNil(t, newUser)
}
