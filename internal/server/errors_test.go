package server

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestErrEmailAlreadyExists(t *testing.T) {
	err := &ErrEmailAlreadyExists{Email: "test@example.com"}
	assert.Equal(t, "email already registered: test@example.com", err.Error())
	assert.Equal(t, http.StatusConflict, HTTPStatus(err))
}

func TestErrInvalidCredentials(t *testing.T) {
	err := &ErrInvalidCredentials{}
	assert.Equal(t, "invalid email or password", err.Error())
	assert.Equal(t, http.StatusUnauthorized, HTTPStatus(err))
}

func TestErrUserNotFound(t *testing.T) {
	userID := uuid.New()
	err := &ErrUserNotFound{UserID: userID}
	assert.Equal(t, "user not found: "+userID.String(), err.Error())
	assert.Equal(t, http.StatusNotFound, HTTPStatus(err))
}

func TestErrPasswordMismatch(t *testing.T) {
	err := &ErrPasswordMismatch{}
	assert.Equal(t, "current password is incorrect", err.Error())
	assert.Equal(t, http.StatusUnauthorized, HTTPStatus(err))
}

func TestErrValidation(t *testing.T) {
	err := &ErrValidation{Field: "email", Message: "invalid format"}
	assert.Equal(t, "validation error: email - invalid format", err.Error())
	assert.Equal(t, http.StatusBadRequest, HTTPStatus(err))
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "ErrEmailAlreadyExists",
			err:      &ErrEmailAlreadyExists{Email: "test@example.com"},
			expected: http.StatusConflict,
		},
		{
			name:     "ErrInvalidCredentials",
			err:      &ErrInvalidCredentials{},
			expected: http.StatusUnauthorized,
		},
		{
			name:     "ErrPasswordMismatch",
			err:      &ErrPasswordMismatch{},
			expected: http.StatusUnauthorized,
		},
		{
			name:     "ErrUserNotFound",
			err:      &ErrUserNotFound{UserID: uuid.New()},
			expected: http.StatusNotFound,
		},
		{
			name:     "ErrValidation",
			err:      &ErrValidation{Field: "password", Message: "too short"},
			expected: http.StatusBadRequest,
		},
		{
			name:     "Unknown error",
			err:      assert.AnError,
			expected: http.StatusInternalServerError,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, HTTPStatus(tt.err))
		})
	}
}
