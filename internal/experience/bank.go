// Package experience provides functionality to load and normalize experience bank files.
package experience

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathan/resume-customizer/internal/schemas"
	"github.com/jonathan/resume-customizer/internal/types"
)

// ExperienceBankSchemaPath is the path to the experience bank schema file (relative to repo root)
const ExperienceBankSchemaPath = "schemas/experience_bank.schema.json"

// LoadExperienceBank loads an experience bank from a JSON file, validating it against the schema
func LoadExperienceBank(path string) (*types.ExperienceBank, error) {
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &LoadError{
			Message: fmt.Sprintf("failed to read file %s", path),
			Cause:   err,
		}
	}

	// Validate against schema
	// Try schema path relative to repo root first, then relative to package directory
	schemaPath := ExperienceBankSchemaPath
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		// If not found, try relative to package directory (for tests)
		schemaPath = filepath.Join("..", "..", ExperienceBankSchemaPath)
	}
	if err := schemas.ValidateJSON(schemaPath, path); err != nil {
		return nil, &LoadError{
			Message: "schema validation failed",
			Cause:   err,
		}
	}

	// Unmarshal JSON
	var bank types.ExperienceBank
	if err := json.Unmarshal(content, &bank); err != nil {
		return nil, &LoadError{
			Message: "failed to unmarshal JSON",
			Cause:   err,
		}
	}

	return &bank, nil
}
