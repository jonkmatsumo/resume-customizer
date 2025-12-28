// Package config provides configuration loading and validation for the CLI.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the CLI configuration that can be loaded from a JSON file.
// All fields are optional; missing values use defaults or must be provided via CLI flags.
type Config struct {
	// Paths
	Job         string `json:"job,omitempty"`          // Path to job posting text file
	JobURL      string `json:"job_url,omitempty"`      // URL to fetch job posting from
	CompanySeed string `json:"company_seed,omitempty"` // Company seed URL for brand research
	Template    string `json:"template,omitempty"`     // Path to LaTeX template

	// Candidate Info
	UserID string `json:"user_id,omitempty"` // User UUID (required for DB-based runs)
	Name   string `json:"name,omitempty"`    // Candidate name
	Email  string `json:"email,omitempty"`   // Candidate email
	Phone  string `json:"phone,omitempty"`   // Candidate phone

	// Limits
	MaxBullets int `json:"max_bullets,omitempty"` // Maximum bullets allowed
	MaxLines   int `json:"max_lines,omitempty"`   // Maximum lines allowed

	// Behavior
	APIKey            string  `json:"api_key,omitempty"`            // Gemini API key
	UseBrowser        bool    `json:"use_browser,omitempty"`        // Use headless browser for SPA sites
	Verbose           bool    `json:"verbose,omitempty"`            // Print detailed debug information
	SkillMatchRatio   float64 `json:"skill_match_ratio,omitempty"`  // Ratio of space reserved for skill matching (0.0-1.0)
	SpecificityWeight float64 `json:"specificity_weight,omitempty"` // Weight for specificity vs requirement level (0.0-1.0)
	DatabaseURL       string  `json:"database_url,omitempty"`       // PostgreSQL connection URL
}

// LoadConfig loads configuration from a JSON file.
// Returns an error if the file cannot be read or parsed.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is empty")
	}

	// Resolve path relative to current directory if not absolute
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		path = filepath.Join(cwd, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return &cfg, nil
}

// Validate checks that the configuration has valid values.
// Note: This doesn't check for required fields since those are handled
// by CLI flag validation after merging.
func (c *Config) Validate() error {
	// Validate mutually exclusive fields
	if c.Job != "" && c.JobURL != "" {
		return fmt.Errorf("config error: 'job' and 'job_url' are mutually exclusive")
	}

	// Validate numeric ranges
	if c.MaxBullets < 0 {
		return fmt.Errorf("config error: 'max_bullets' must be non-negative")
	}
	if c.MaxLines < 0 {
		return fmt.Errorf("config error: 'max_lines' must be non-negative")
	}

	// Validate file paths exist (if specified)
	if c.Template != "" {
		if _, err := os.Stat(c.Template); os.IsNotExist(err) {
			return fmt.Errorf("config error: template file not found: %s", c.Template)
		}
	}

	if c.Job != "" {
		if _, err := os.Stat(c.Job); os.IsNotExist(err) {
			return fmt.Errorf("config error: job file not found: %s", c.Job)
		}
	}

	return nil
}

// MergeWithDefaults returns a new Config with empty string fields filled from defaults.
// This is used to apply config file values as defaults for CLI flags.
func (c *Config) MergeWithDefaults(defaults Config) Config {
	result := *c

	// String fields: use default if empty
	if result.Job == "" {
		result.Job = defaults.Job
	}
	if result.JobURL == "" {
		result.JobURL = defaults.JobURL
	}
	if result.CompanySeed == "" {
		result.CompanySeed = defaults.CompanySeed
	}
	if result.Template == "" {
		result.Template = defaults.Template
	}
	if result.Name == "" {
		result.Name = defaults.Name
	}
	if result.Email == "" {
		result.Email = defaults.Email
	}
	if result.Phone == "" {
		result.Phone = defaults.Phone
	}
	if result.APIKey == "" {
		result.APIKey = defaults.APIKey
	}
	if result.DatabaseURL == "" {
		result.DatabaseURL = defaults.DatabaseURL
	}

	// Int fields: use default if zero
	if result.MaxBullets == 0 {
		result.MaxBullets = defaults.MaxBullets
	}
	if result.MaxLines == 0 {
		result.MaxLines = defaults.MaxLines
	}

	// Float fields
	if result.SkillMatchRatio == 0 {
		if defaults.SkillMatchRatio > 0 {
			result.SkillMatchRatio = defaults.SkillMatchRatio
		} else {
			result.SkillMatchRatio = 0.8 // Default to 80% skill match
		}
	}

	// Bool fields: cannot distinguish unset from false, so we don't merge
	// (CLI flags should always win for bools)

	return result
}
