package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GetUserByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create test user with known email
	name := "Test User Email"
	email := "test-email-" + uuid.New().String() + "@example.com"
	phone := "555-0100"
	userID, err := db.CreateUser(ctx, name, email, phone)
	require.NoError(t, err)
	defer db.DeleteUser(ctx, userID)

	// Test successful retrieval
	user, err := db.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, name, user.Name)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, phone, user.Phone)
	// Password fields should be populated (even if empty)
	assert.NotNil(t, user.PasswordHash)
	assert.False(t, user.PasswordSet) // New users have password_set = FALSE by default

	// Test with non-existent email
	nonExistentEmail := "nonexistent-" + uuid.New().String() + "@example.com"
	user2, err := db.GetUserByEmail(ctx, nonExistentEmail)
	require.NoError(t, err)
	assert.Nil(t, user2) // Should return nil, nil (matching GetUser pattern)

	// Test with empty email
	user3, err := db.GetUserByEmail(ctx, "")
	require.NoError(t, err)
	assert.Nil(t, user3)
}

func TestIntegration_UpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create test user
	name := "Test User Password"
	email := "test-password-" + uuid.New().String() + "@example.com"
	phone := "555-0100"
	userID, err := db.CreateUser(ctx, name, email, phone)
	require.NoError(t, err)
	defer db.DeleteUser(ctx, userID)

	// Get initial user to check timestamps
	userBefore, err := db.GetUser(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, userBefore)
	initialUpdatedAt := userBefore.UpdatedAt

	// Wait a moment to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Update password hash
	newPasswordHash := "$2a$12$testhashedpassword12345678901234567890123456789012345678901234567890"
	err = db.UpdatePassword(ctx, userID, newPasswordHash)
	require.NoError(t, err)

	// Verify password was updated
	userAfter, err := db.GetUser(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, userAfter)
	assert.Equal(t, newPasswordHash, userAfter.PasswordHash)
	assert.True(t, userAfter.PasswordSet) // Should be set to TRUE
	assert.True(t, userAfter.UpdatedAt.After(initialUpdatedAt), "updated_at should be updated")

	// Test with non-existent user ID
	nonExistentID := uuid.New()
	err = db.UpdatePassword(ctx, nonExistentID, newPasswordHash)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")

	// Test with empty password hash (edge case - should still update)
	err = db.UpdatePassword(ctx, userID, "")
	require.NoError(t, err)
	userAfterEmpty, err := db.GetUser(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "", userAfterEmpty.PasswordHash)
	assert.True(t, userAfterEmpty.PasswordSet) // Still TRUE
}

func TestIntegration_CheckEmailExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create test user with known email
	name := "Test User Exists"
	email := "test-exists-" + uuid.New().String() + "@example.com"
	phone := "555-0100"
	userID, err := db.CreateUser(ctx, name, email, phone)
	require.NoError(t, err)
	defer db.DeleteUser(ctx, userID)

	// Test with existing email
	exists, err := db.CheckEmailExists(ctx, email)
	require.NoError(t, err)
	assert.True(t, exists)

	// Test with non-existent email
	nonExistentEmail := "nonexistent-" + uuid.New().String() + "@example.com"
	exists, err = db.CheckEmailExists(ctx, nonExistentEmail)
	require.NoError(t, err)
	assert.False(t, exists)

	// Test with empty email
	exists, err = db.CheckEmailExists(ctx, "")
	require.NoError(t, err)
	assert.False(t, exists)

	// Test case sensitivity (emails should be case-sensitive in database)
	upperEmail := "TEST-EXISTS-" + uuid.New().String() + "@EXAMPLE.COM"
	exists, err = db.CheckEmailExists(ctx, upperEmail)
	require.NoError(t, err)
	assert.False(t, exists) // Different case = different email
}
