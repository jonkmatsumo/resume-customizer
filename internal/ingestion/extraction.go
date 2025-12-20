package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// ExtractedContent represents the structured output from the ingestion LLM
type ExtractedContent struct {
	Requirements     []string          `json:"requirements"`
	Responsibilities []string          `json:"responsibilities"`
	AdminInfo        map[string]string `json:"admin_info"` // Salary, Clearance, Citizenship, Location, etc.
}

// ExtractWithLLM uses Gemini to separate core content from administrative metadata
func ExtractWithLLM(ctx context.Context, text string, apiKey string) (*ExtractedContent, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key required for LLM extraction")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.1) // Low temperature for factual extraction
	model.ResponseMIMEType = "application/json"

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

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	part, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response type")
	}

	var extracted ExtractedContent
	// Clean markdown block if present
	cleanJSON := cleanJSONBlock(string(part))

	if err := json.Unmarshal([]byte(cleanJSON), &extracted); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w (content: %s)", err, cleanJSON)
	}

	return &extracted, nil
}

func cleanJSONBlock(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}
