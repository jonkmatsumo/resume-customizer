//nolint:revive // types is a standard Go package name pattern
package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUserRequest_Validation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name    string
		request CreateUserRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: CreateUserRequest{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "password123",
				Phone:    "555-0100",
			},
			wantErr: false,
		},
		{
			name: "valid request without phone",
			request: CreateUserRequest{
				Name:     "Jane Doe",
				Email:    "jane@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			request: CreateUserRequest{
				Email:    "john@example.com",
				Password: "password123",
			},
			wantErr: true,
			errMsg:  "required",
		},
		{
			name: "empty name",
			request: CreateUserRequest{
				Name:     "",
				Email:    "john@example.com",
				Password: "password123",
			},
			wantErr: true,
			errMsg:  "required", // Empty string fails required, not min
		},
		{
			name: "missing email",
			request: CreateUserRequest{
				Name:     "John Doe",
				Password: "password123",
			},
			wantErr: true,
			errMsg:  "required",
		},
		{
			name: "invalid email format",
			request: CreateUserRequest{
				Name:     "John Doe",
				Email:    "not-an-email",
				Password: "password123",
			},
			wantErr: true,
			errMsg:  "email",
		},
		{
			name: "missing password",
			request: CreateUserRequest{
				Name:  "John Doe",
				Email: "john@example.com",
			},
			wantErr: true,
			errMsg:  "required",
		},
		{
			name: "password too short",
			request: CreateUserRequest{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "short",
			},
			wantErr: true,
			errMsg:  "min",
		},
		{
			name: "password exactly 8 characters",
			request: CreateUserRequest{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "12345678",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.request)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoginRequest_Validation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name    string
		request LoginRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: LoginRequest{
				Email:    "john@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "missing email",
			request: LoginRequest{
				Password: "password123",
			},
			wantErr: true,
			errMsg:  "required",
		},
		{
			name: "invalid email format",
			request: LoginRequest{
				Email:    "not-an-email",
				Password: "password123",
			},
			wantErr: true,
			errMsg:  "email",
		},
		{
			name: "missing password",
			request: LoginRequest{
				Email: "john@example.com",
			},
			wantErr: true,
			errMsg:  "required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.request)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdatePasswordRequest_Validation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name    string
		request UpdatePasswordRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: UpdatePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "newpassword456",
			},
			wantErr: false,
		},
		{
			name: "missing current password",
			request: UpdatePasswordRequest{
				NewPassword: "newpassword456",
			},
			wantErr: true,
			errMsg:  "required",
		},
		{
			name: "missing new password",
			request: UpdatePasswordRequest{
				CurrentPassword: "oldpassword123",
			},
			wantErr: true,
			errMsg:  "required",
		},
		{
			name: "new password too short",
			request: UpdatePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "short",
			},
			wantErr: true,
			errMsg:  "min",
		},
		{
			name: "new password exactly 8 characters",
			request: UpdatePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "12345678",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.request)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoginResponse_Serialization(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	user := &User{
		ID:          userID,
		Name:        "John Doe",
		Email:       "john@example.com",
		Phone:       "555-0100",
		PasswordSet: true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	token := "test-jwt-token-12345"

	response := LoginResponse{
		User:  user,
		Token: token,
	}

	// Test JSON marshaling
	jsonBytes, err := json.Marshal(response)
	require.NoError(t, err)
	require.NotEmpty(t, jsonBytes)

	// Verify JSON contains expected fields
	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "user")
	assert.Contains(t, jsonStr, "token")
	assert.Contains(t, jsonStr, userID.String())
	assert.Contains(t, jsonStr, "John Doe")
	assert.Contains(t, jsonStr, token)

	// Verify password_hash is not in JSON (should be excluded from User type)
	assert.NotContains(t, jsonStr, "password_hash")

	// Test JSON unmarshaling
	var unmarshaled LoginResponse
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, token, unmarshaled.Token)
	assert.NotNil(t, unmarshaled.User)
	assert.Equal(t, userID, unmarshaled.User.ID)
	assert.Equal(t, "John Doe", unmarshaled.User.Name)
	assert.Equal(t, "john@example.com", unmarshaled.User.Email)
}

func TestCreateUserRequest_ValidateMethod(t *testing.T) {
	req := CreateUserRequest{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
	}
	err := req.Validate()
	require.NoError(t, err)

	req.Email = "invalid-email"
	err = req.Validate()
	require.Error(t, err)
}

func TestLoginRequest_ValidateMethod(t *testing.T) {
	req := LoginRequest{
		Email:    "john@example.com",
		Password: "password123",
	}
	err := req.Validate()
	require.NoError(t, err)

	req.Email = "invalid-email"
	err = req.Validate()
	require.Error(t, err)
}

func TestUpdatePasswordRequest_ValidateMethod(t *testing.T) {
	req := UpdatePasswordRequest{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword456",
	}
	err := req.Validate()
	require.NoError(t, err)

	req.NewPassword = "short"
	err = req.Validate()
	require.Error(t, err)
}
