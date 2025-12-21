package ingestion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Metadata contains metadata about an ingested job posting
type Metadata struct {
	URL            string            `json:"url,omitempty"`
	Timestamp      string            `json:"timestamp"`                 // RFC3339 format
	Hash           string            `json:"hash"`                      // SHA256 hex digest
	Platform       string            `json:"platform,omitempty"`        // Detected job board platform
	Company        string            `json:"company,omitempty"`         // Detected company name
	AboutCompany   string            `json:"about_company,omitempty"`   // Verbatim "About Us" text
	AdminInfo      map[string]string `json:"admin_info,omitempty"`      // Salary, Clearance, Citizenship, etc.
	ExtractedLinks []string          `json:"extracted_links,omitempty"` // Links found in the job posting
}

// NewMetadata creates a new Metadata instance with current timestamp
func NewMetadata(content string, url string) *Metadata {
	return &Metadata{
		URL:       url,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Hash:      computeHash(content),
	}
}

// computeHash computes SHA256 hash of content and returns hex string
func computeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// ToJSON marshals Metadata to pretty-printed JSON
func (m *Metadata) ToJSON() ([]byte, error) {
	// Use standard encoding/json but format nicely
	jsonBytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata to JSON: %w", err)
	}
	return jsonBytes, nil
}
