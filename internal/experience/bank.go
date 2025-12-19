// Package experience provides functionality to load and normalize experience bank files.
package experience

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jonathan/resume-customizer/internal/types"
)

// LoadExperienceBank loads an experience bank from a JSON file
func LoadExperienceBank(path string) (*types.ExperienceBank, error) {
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &LoadError{
			Message: fmt.Sprintf("failed to read file %s", path),
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
