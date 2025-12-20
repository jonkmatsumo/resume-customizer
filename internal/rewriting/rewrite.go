// Package rewriting provides functionality to rewrite bullet points to match job requirements and company brand voice.
package rewriting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/jonathan/resume-customizer/internal/types"
)

const (
	// DefaultModel is the Gemini model to use for bullet rewriting
	DefaultModel = "gemini-1.5-flash"
	// DefaultTemperature is the temperature setting for structured output
	DefaultTemperature = 0.1
)

// RewriteBullets rewrites selected bullets to match job requirements and company voice
func RewriteBullets(ctx context.Context, selectedBullets *types.SelectedBullets, jobProfile *types.JobProfile, companyProfile *types.CompanyProfile, apiKey string) (*types.RewrittenBullets, error) {
	if apiKey == "" {
		return nil, &APICallError{Message: "API key is required"}
	}

	// Initialize Gemini client
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, &APICallError{
			Message: "failed to create Gemini client",
			Cause:   err,
		}
	}
	defer func() { _ = client.Close() }()

	model := client.GenerativeModel(DefaultModel)
	model.SetTemperature(DefaultTemperature)

	// Rewrite each bullet
	rewrittenBullets := make([]types.RewrittenBullet, 0, len(selectedBullets.Bullets))

	for _, originalBullet := range selectedBullets.Bullets {
		// Build rewriting prompt
		prompt := buildRewritingPrompt(originalBullet, jobProfile, companyProfile)

		// Call Gemini API
		resp, err := model.GenerateContent(ctx, genai.Text(prompt))
		if err != nil {
			return nil, &APICallError{
				Message: fmt.Sprintf("failed to generate content for bullet %s", originalBullet.ID),
				Cause:   err,
			}
		}

		// Extract text from response
		responseText, err := extractTextFromResponse(resp)
		if err != nil {
			return nil, &APICallError{
				Message: fmt.Sprintf("failed to extract text for bullet %s", originalBullet.ID),
				Cause:   err,
			}
		}

		// Parse response (expects just the rewritten text)
		rewrittenText, err := parseBulletResponse(responseText)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response for bullet %s: %w", originalBullet.ID, err)
		}

		// Post-process bullet
		rewrittenBullet, err := postProcessBullet(rewrittenText, originalBullet, companyProfile)
		if err != nil {
			return nil, fmt.Errorf("failed to post-process bullet %s: %w", originalBullet.ID, err)
		}

		rewrittenBullets = append(rewrittenBullets, *rewrittenBullet)
	}

	return &types.RewrittenBullets{
		Bullets: rewrittenBullets,
	}, nil
}

// buildRewritingPrompt constructs the prompt for bullet rewriting
func buildRewritingPrompt(bullet types.SelectedBullet, jobProfile *types.JobProfile, companyProfile *types.CompanyProfile) string {
	var sb strings.Builder

	sb.WriteString("Rewrite the following resume bullet point to match the job requirements and company brand voice.\n\n")
	sb.WriteString("Original bullet:\n")
	sb.WriteString(bullet.Text)
	sb.WriteString("\n\n")

	// Add job requirements context
	if jobProfile != nil {
		sb.WriteString("Job requirements:\n")
		if len(jobProfile.HardRequirements) > 0 {
			sb.WriteString("- Hard requirements: ")
			reqs := make([]string, len(jobProfile.HardRequirements))
			for i, req := range jobProfile.HardRequirements {
				reqs[i] = req.Skill
			}
			sb.WriteString(strings.Join(reqs, ", "))
			sb.WriteString("\n")
		}
		if len(jobProfile.NiceToHaves) > 0 {
			sb.WriteString("- Preferred skills: ")
			reqs := make([]string, len(jobProfile.NiceToHaves))
			for i, req := range jobProfile.NiceToHaves {
				reqs[i] = req.Skill
			}
			sb.WriteString(strings.Join(reqs, ", "))
			sb.WriteString("\n")
		}
		if len(jobProfile.Keywords) > 0 {
			sb.WriteString("- Keywords: ")
			sb.WriteString(strings.Join(jobProfile.Keywords, ", "))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Add company voice context
	if companyProfile != nil {
		sb.WriteString("Company brand voice:\n")
		sb.WriteString(fmt.Sprintf("- Tone: %s\n", companyProfile.Tone))
		if len(companyProfile.StyleRules) > 0 {
			sb.WriteString("- Style rules:\n")
			for _, rule := range companyProfile.StyleRules {
				sb.WriteString(fmt.Sprintf("  * %s\n", rule))
			}
		}
		if len(companyProfile.TabooPhrases) > 0 {
			sb.WriteString("- Avoid these phrases: ")
			sb.WriteString(strings.Join(companyProfile.TabooPhrases, ", "))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Requirements:\n")
	sb.WriteString("- Start with a strong action verb\n")
	sb.WriteString("- Include quantified impact/metrics where possible\n")
	sb.WriteString("- Match the company's tone and style rules\n")
	sb.WriteString("- Do NOT use any taboo phrases\n")
	sb.WriteString("- Keep similar length to original (approximately ")
	sb.WriteString(fmt.Sprintf("%d characters)\n", bullet.LengthChars))
	sb.WriteString("- Align with job requirements and keywords\n")
	sb.WriteString("- Return ONLY the rewritten bullet text, no markdown, no explanation, no code blocks\n")

	return sb.String()
}

// extractTextFromResponse extracts text content from Gemini API response
func extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if len(resp.Candidates) == 0 {
		return "", &ParseError{Message: "no candidates in API response"}
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", &ParseError{Message: "no content in API response"}
	}

	var parts []string
	for _, part := range candidate.Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			parts = append(parts, string(textPart))
		}
	}

	if len(parts) == 0 {
		return "", &ParseError{Message: "no text content in response"}
	}

	// Join all text parts
	text := strings.Join(parts, "")

	// Remove markdown code blocks if present
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		// Remove code block markers
		lines := strings.Split(text, "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], "```") {
			lines = lines[1:]
		}
		if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
			lines = lines[:len(lines)-1]
		}
		text = strings.Join(lines, "\n")
		text = strings.TrimSpace(text)
	}

	return text, nil
}

// parseBulletResponse parses the API response to extract rewritten text
// The API should return just the text, but we handle JSON wrapper if present
func parseBulletResponse(responseText string) (string, error) {
	text := strings.TrimSpace(responseText)

	// Try to parse as JSON first (in case LLM returns wrapped JSON)
	var jsonResp struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(text), &jsonResp); err == nil && jsonResp.Text != "" {
		return strings.TrimSpace(jsonResp.Text), nil
	}

	// Otherwise, treat as plain text
	return text, nil
}

// postProcessBullet computes metadata and validates style for a rewritten bullet
func postProcessBullet(rewrittenText string, originalBullet types.SelectedBullet, companyProfile *types.CompanyProfile) (*types.RewrittenBullet, error) {
	// Compute length
	lengthChars := ComputeLengthChars(rewrittenText)

	// Estimate lines
	estimatedLines := EstimateLines(lengthChars)
	if estimatedLines < 1 {
		estimatedLines = 1
	}

	// Validate style
	styleChecks := ValidateStyle(rewrittenText, companyProfile, originalBullet.LengthChars)

	return &types.RewrittenBullet{
		OriginalBulletID: originalBullet.ID,
		FinalText:        rewrittenText,
		LengthChars:      lengthChars,
		EstimatedLines:   estimatedLines,
		StyleChecks: types.StyleChecks{
			StrongVerb:   styleChecks.StrongVerb,
			Quantified:   styleChecks.Quantified,
			NoTaboo:      styleChecks.NoTaboo,
			TargetLength: styleChecks.TargetLength,
		},
	}, nil
}
