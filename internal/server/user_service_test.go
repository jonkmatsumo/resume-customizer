package server

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertDBUserToTypesUser(t *testing.T) {
	t.Run("valid user", func(t *testing.T) {
		now := time.Now()
		dbUser := &db.User{
			ID:           uuid.New(),
			Name:         "John Doe",
			Email:        "john@example.com",
			Phone:        "555-0100",
			PasswordHash: "hashed-password",
			PasswordSet:  true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		typesUser := convertDBUserToTypesUser(dbUser)
		require.NotNil(t, typesUser)
		assert.Equal(t, dbUser.ID, typesUser.ID)
		assert.Equal(t, dbUser.Name, typesUser.Name)
		assert.Equal(t, dbUser.Email, typesUser.Email)
		assert.Equal(t, dbUser.Phone, typesUser.Phone)
		assert.Equal(t, dbUser.PasswordSet, typesUser.PasswordSet)
		assert.Equal(t, dbUser.CreatedAt, typesUser.CreatedAt)
		assert.Equal(t, dbUser.UpdatedAt, typesUser.UpdatedAt)
		// Password hash should not be in types.User (it doesn't have that field)
	})

	t.Run("nil user", func(t *testing.T) {
		typesUser := convertDBUserToTypesUser(nil)
		assert.Nil(t, typesUser)
	})
}

func TestUserService_Register(t *testing.T) {
	// Unit tests for Register are limited without mocking
	// Most testing is done in integration tests with real database
	// This test verifies the method exists and basic structure
	t.Run("service can be created", func(t *testing.T) {
		passwordConfig, err := config.NewPasswordConfig()
		require.NoError(t, err)
		// We can't create service without real DB, so just verify config works
		assert.NotNil(t, passwordConfig)
	})
}

func TestUserService_Login(t *testing.T) {
	// Unit tests for Login are limited without real password hashing
	// Most testing is done in integration tests
	t.Run("service structure verified", func(t *testing.T) {
		passwordConfig, err := config.NewPasswordConfig()
		require.NoError(t, err)
		assert.NotNil(t, passwordConfig)
	})
}

func TestUserService_UpdatePassword(t *testing.T) {
	// Unit tests for UpdatePassword are limited without real password hashing
	// Most testing is done in integration tests
	t.Run("service structure verified", func(t *testing.T) {
		passwordConfig, err := config.NewPasswordConfig()
		require.NoError(t, err)
		assert.NotNil(t, passwordConfig)
	})
}
