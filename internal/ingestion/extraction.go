package ingestion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jonathan/resume-customizer/internal/llm"
)

// ExtractedContent represents the structured output from the ingestion LLM
type ExtractedContent struct {
	Requirements     []string          `json:"requirements"`
	Responsibilities []string          `json:"responsibilities"`
	AdminInfo        map[string]string `json:"admin_info"` // Salary, Clearance, Citizenship, Location, etc.
}

// ExtractWithLLM uses LLM to separate core content from administrative metadata
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

	prompt := fmt.Sprintf(`
You are an expert resume parsing assistant. Your task is to extract information from a raw job posting text.
Do NOT rewrite or summarize the text. Extract it verbatim where possible.

Goal: Separate the core job description (Requirements, Responsibilities) from administrative metadata (Salary, Clearance, Citizenship, EEO statements, Benefits, Location).

Input Text:
"""
%s
"""

Instructions:
1. Extract "requirements" as a list of strings. These are hard or soft skills, qualifications, years of experience, etc.
2. Extract "responsibilities" as a list of strings. These are what the person will do day-to-day.
3. Extract "admin_info" as a key-value map. Capture ANY information about:
    - Salary / Compensation
    - Security Clearance
    - Citizenship / Visa status
    - Location / Remote status
    - Start Date
    - Benefits summary (if short)
    - Company generic descriptions (EEO, "About Us" boilerplate)

Output JSON Schema:
{
  "requirements": ["string"],
  "responsibilities": ["string"],
  "admin_info": {"key": "value"}
}
`, text)

	// Use TierLite for simple extraction task
	jsonResp, err := client.GenerateJSON(ctx, prompt, llm.TierLite)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	var extracted ExtractedContent
	if err := json.Unmarshal([]byte(jsonResp), &extracted); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w (content: %s)", err, jsonResp)
	}

	return &extracted, nil
}
