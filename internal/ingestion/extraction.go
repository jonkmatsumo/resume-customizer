package ingestion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jonathan/resume-customizer/internal/llm"
)

// ExtractedContent represents the structured output from the ingestion LLM
type ExtractedContent struct {
	// Company metadata
	Company       string `json:"company,omitempty"`
	CompanyDomain string `json:"company_domain,omitempty"` // e.g., "doordash.com"

	// Position metadata
	Title        string   `json:"title,omitempty"`
	Level        string   `json:"level,omitempty"`         // junior, mid, senior, staff, principal, lead
	LevelSignals []string `json:"level_signals,omitempty"` // Phrases indicating level

	// Team context
	TeamContext string `json:"team_context,omitempty"`

	// Core content
	AboutCompany     string            `json:"about_company,omitempty"`
	Requirements     []string          `json:"requirements"`
	Responsibilities []string          `json:"responsibilities"`
	NiceToHave       []string          `json:"nice_to_have,omitempty"`
	AdminInfo        map[string]string `json:"admin_info,omitempty"`
}

// ExtractWithLLM uses LLM to separate core content from administrative metadata.
// It uses the generic JobRequirementsSchema for consistent extraction.
func ExtractWithLLM(ctx context.Context, text string, apiKey string) (*ExtractedContent, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key required for LLM extraction")
	}

	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Use the generic JobRequirementsSchema for prompt construction
	schema := llm.JobRequirementsSchema()
	prompt := llm.BuildExtractionPrompt(schema, text)

	// Use TierLite for simple extraction task
	jsonResp, err := client.GenerateJSON(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Clean any markdown wrappers
	jsonResp = llm.CleanJSONBlock(jsonResp)

	var extracted ExtractedContent
	if err := json.Unmarshal([]byte(jsonResp), &extracted); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w (content: %s)", err, jsonResp)
	}

	return &extracted, nil
}
