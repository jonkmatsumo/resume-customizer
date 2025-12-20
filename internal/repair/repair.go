// Package repair provides functionality to automatically fix violations in LaTeX resumes.
package repair

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
	"github.com/jonathan/resume-customizer/internal/types"
)

// ProposeRepairs uses LLM to analyze violations and propose structured repair actions
func ProposeRepairs(ctx context.Context, violations *types.Violations, plan *types.ResumePlan, rewrittenBullets *types.RewrittenBullets, rankedStories *types.RankedStories, jobProfile *types.JobProfile, companyProfile *types.CompanyProfile, apiKey string) (*types.RepairActions, error) {
	if apiKey == "" {
		return nil, &ProposeError{Message: "API key is required"}
	}

	// Initialize LLM client with default config
	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, &ProposeError{
			Message: "failed to create LLM client",
			Cause:   err,
		}
	}
	defer func() { _ = client.Close() }()

	// Build prompt
	prompt := buildRepairPrompt(violations, plan, rewrittenBullets, rankedStories, jobProfile, companyProfile)

	// Use TierAdvanced for repair proposal (requires complex reasoning)
	responseText, err := client.GenerateContent(ctx, prompt, llm.TierAdvanced)
	if err != nil {
		return nil, &ProposeError{
			Message: "failed to generate content",
			Cause:   err,
		}
	}

	// Parse JSON response
	repairActions, err := parseRepairResponse(responseText)
	if err != nil {
		return nil, &ProposeError{
			Message: "failed to parse repair actions from response",
			Cause:   err,
		}
	}

	// Validate proposed actions
	if err := validateProposedActions(repairActions, plan, rewrittenBullets, rankedStories); err != nil {
		return nil, &ProposeError{
			Message: "proposed actions failed validation",
			Cause:   err,
		}
	}

	return repairActions, nil
}

// buildRepairPrompt constructs the prompt for repair proposal
func buildRepairPrompt(violations *types.Violations, plan *types.ResumePlan, rewrittenBullets *types.RewrittenBullets, rankedStories *types.RankedStories, jobProfile *types.JobProfile, companyProfile *types.CompanyProfile) string {
	var sb strings.Builder

	// Add intro from external prompt
	sb.WriteString(prompts.MustGet("repair.json", "propose-repairs-intro"))

	// Add violations (dynamic)
	sb.WriteString("## Violations\n\n")
	for i, violation := range violations.Violations {
		sb.WriteString(fmt.Sprintf("%d. Type: %s, Severity: %s\n", i+1, violation.Type, violation.Severity))
		sb.WriteString(fmt.Sprintf("   Details: %s\n", violation.Details))
		if violation.LineNumber != nil {
			sb.WriteString(fmt.Sprintf("   Line: %d\n", *violation.LineNumber))
		}
		if violation.CharCount != nil {
			sb.WriteString(fmt.Sprintf("   Character count: %d\n", *violation.CharCount))
		}
		sb.WriteString("\n")
	}

	// Add resume plan (dynamic)
	sb.WriteString("## Current Resume Plan\n\n")
	sb.WriteString(fmt.Sprintf("Selected Stories: %d\n", len(plan.SelectedStories)))
	for _, story := range plan.SelectedStories {
		sb.WriteString(fmt.Sprintf("- Story ID: %s, Bullets: %v, Section: %s\n", story.StoryID, story.BulletIDs, story.Section))
	}
	sb.WriteString("\n")

	// Add rewritten bullets (dynamic)
	sb.WriteString("## Current Rewritten Bullets\n\n")
	for i, bullet := range rewrittenBullets.Bullets {
		sb.WriteString(fmt.Sprintf("%d. ID: %s, Length: %d chars, Lines: %d\n", i+1, bullet.OriginalBulletID, bullet.LengthChars, bullet.EstimatedLines))
		sb.WriteString(fmt.Sprintf("   Text: %s\n", bullet.FinalText))
		sb.WriteString("\n")
	}

	// Add alternative stories (dynamic)
	sb.WriteString("## Available Alternative Stories\n\n")
	for i, story := range rankedStories.Ranked {
		if i >= 10 { // Limit to top 10
			break
		}
		sb.WriteString(fmt.Sprintf("- Story ID: %s, Relevance: %.2f, Skills: %v\n", story.StoryID, story.RelevanceScore, story.MatchedSkills))
	}
	sb.WriteString("\n")

	// Add job requirements (dynamic)
	sb.WriteString("## Job Requirements\n\n")
	sb.WriteString(fmt.Sprintf("Role: %s\n", jobProfile.RoleTitle))
	sb.WriteString(fmt.Sprintf("Company: %s\n", jobProfile.Company))
	if len(jobProfile.HardRequirements) > 0 {
		sb.WriteString("Hard Requirements:\n")
		for _, req := range jobProfile.HardRequirements {
			sb.WriteString(fmt.Sprintf("- %s\n", req.Skill))
		}
	}
	sb.WriteString("\n")

	// Add company brand voice (dynamic)
	if companyProfile != nil {
		sb.WriteString("## Company Brand Voice\n\n")
		if len(companyProfile.TabooPhrases) > 0 {
			sb.WriteString("Taboo Phrases to Avoid:\n")
			for _, phrase := range companyProfile.TabooPhrases {
				sb.WriteString(fmt.Sprintf("- %s\n", phrase))
			}
		}
		sb.WriteString("\n")
	}

	// Add action types from external prompt
	sb.WriteString(prompts.MustGet("repair.json", "repair-action-types"))

	// Add instructions from external prompt
	sb.WriteString(prompts.MustGet("repair.json", "repair-instructions"))

	return sb.String()
}

// parseRepairResponse parses JSON response into RepairActions
func parseRepairResponse(responseText string) (*types.RepairActions, error) {
	// Try to extract JSON from response (might have markdown code blocks)
	jsonText := extractJSONFromText(responseText)

	var repairActions types.RepairActions
	if err := json.Unmarshal([]byte(jsonText), &repairActions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Limit number of actions (safety check)
	maxActions := 5
	if len(repairActions.Actions) > maxActions {
		repairActions.Actions = repairActions.Actions[:maxActions]
	}

	return &repairActions, nil
}

// extractJSONFromText extracts JSON from text that might contain markdown code blocks
func extractJSONFromText(text string) string {
	text = strings.TrimSpace(text)

	// Try to find JSON in markdown code block
	if idx := strings.Index(text, "```json"); idx >= 0 {
		start := idx + 7 // Length of "```json"
		if endIdx := strings.Index(text[start:], "```"); endIdx >= 0 {
			return strings.TrimSpace(text[start : start+endIdx])
		}
	}

	// Try to find JSON in generic code block
	if idx := strings.Index(text, "```"); idx >= 0 {
		start := idx + 3 // Length of "```"
		if endIdx := strings.Index(text[start:], "```"); endIdx >= 0 {
			return strings.TrimSpace(text[start : start+endIdx])
		}
	}

	// Try to find JSON object directly
	if strings.HasPrefix(text, "{") {
		// Find matching closing brace
		braceCount := 0
		for i := 0; i < len(text); i++ {
			switch text[i] {
			case '{':
				braceCount++
			case '}':
				braceCount--
				if braceCount == 0 {
					return text[:i+1]
				}
			}
		}
	}

	// Fallback: return text as-is (might already be JSON)
	return text
}

// validateProposedActions validates that proposed actions are valid
func validateProposedActions(actions *types.RepairActions, plan *types.ResumePlan, rewrittenBullets *types.RewrittenBullets, rankedStories *types.RankedStories) error {
	// Build sets for validation
	planStoryIDs := make(map[string]bool)
	planBulletIDs := make(map[string]bool)
	for _, story := range plan.SelectedStories {
		planStoryIDs[story.StoryID] = true
		for _, bulletID := range story.BulletIDs {
			planBulletIDs[bulletID] = true
		}
	}

	rewrittenBulletIDs := make(map[string]bool)
	for _, bullet := range rewrittenBullets.Bullets {
		rewrittenBulletIDs[bullet.OriginalBulletID] = true
	}

	rankedStoryIDs := make(map[string]bool)
	for _, story := range rankedStories.Ranked {
		rankedStoryIDs[story.StoryID] = true
	}

	for i, action := range actions.Actions {
		switch action.Type {
		case "shorten_bullet":
			if action.BulletID == "" {
				return fmt.Errorf("action %d: bullet_id is required for shorten_bullet", i)
			}
			if !rewrittenBulletIDs[action.BulletID] {
				return fmt.Errorf("action %d: bullet_id %s not found in rewritten bullets", i, action.BulletID)
			}
			if action.TargetChars == nil {
				return fmt.Errorf("action %d: target_chars is required for shorten_bullet", i)
			}
			if *action.TargetChars <= 0 {
				return fmt.Errorf("action %d: target_chars must be positive", i)
			}

		case "drop_bullet":
			if action.BulletID == "" {
				return fmt.Errorf("action %d: bullet_id is required for drop_bullet", i)
			}
			// Bullet should be in plan, but might not be in rewritten bullets if already processed
			if !planBulletIDs[action.BulletID] {
				return fmt.Errorf("action %d: bullet_id %s not found in plan", i, action.BulletID)
			}

		case "swap_story":
			if action.StoryID == "" {
				return fmt.Errorf("action %d: story_id is required for swap_story", i)
			}
			if !planStoryIDs[action.StoryID] {
				return fmt.Errorf("action %d: story_id %s not found in plan", i, action.StoryID)
			}
			// Check that there are alternative stories available
			foundAlternative := false
			for _, rankedStory := range rankedStories.Ranked {
				if rankedStory.StoryID != action.StoryID && rankedStory.StoryID != "" {
					foundAlternative = true
					break
				}
			}
			if !foundAlternative {
				return fmt.Errorf("action %d: no alternative stories available for swap", i)
			}

		case "tighten_section", "adjust_template_params":
			// Not implemented - validation would pass but action won't be applied
			// Could return error here if we want strict validation

		default:
			return fmt.Errorf("action %d: unknown action type: %s", i, action.Type)
		}

		if action.Reason == "" {
			return fmt.Errorf("action %d: reason is required", i)
		}
	}

	return nil
}
