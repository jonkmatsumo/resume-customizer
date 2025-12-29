package main

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseJobCommand_FlagsValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantError   bool
		errorString string
	}{
		{
			name:        "Missing --run-id flag",
			args:        []string{"parse-job", "--db-url", "postgres://test"},
			wantError:   true,
			errorString: "required",
		},
		{
			name:        "Missing --db-url flag",
			args:        []string{"parse-job", "--run-id", "00000000-0000-0000-0000-000000000000"},
			wantError:   true,
			errorString: "required",
		},
	}

	binaryPath := getBinaryPath(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.args...)
			output, err := cmd.CombinedOutput()

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, string(output), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseJobCommand_MissingAPIKey(t *testing.T) {
	// Skip - requires database setup with test fixtures
	// TODO: Add comprehensive database integration tests
	t.Skip("Skipping - requires database setup. TODO: Add database integration tests")
}
