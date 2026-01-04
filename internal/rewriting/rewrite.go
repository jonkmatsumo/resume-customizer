// Package rewriting provides functionality to rewrite bullet points to match job requirements and company brand voice.
package rewriting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/prompts"
	"github.com/jonathan/resume-customizer/internal/types"
)

// RewriteBullets rewrites selected bullets to match job requirements and company voice
func RewriteBullets(ctx context.Context, selectedBullets *types.SelectedBullets, jobProfile *types.JobProfile, companyProfile *types.CompanyProfile, apiKey string) (*types.RewrittenBullets, error) {
	if apiKey == "" {
		return nil, &APICallError{Message: "API key is required"}
	}

	// Initialize LLM client with default config
	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, &APICallError{
			Message: "failed to create LLM client",
			Cause:   err,
		}
	}
	defer func() { _ = client.Close() }()

	// Track used verbs across the entire resume for diversity
	usedVerbs := []string{}

	// Rewrite each bullet
	rewrittenBullets := make([]types.RewrittenBullet, 0, len(selectedBullets.Bullets))

	for _, originalBullet := range selectedBullets.Bullets {
		// Build rewriting prompt with verbs to avoid
		prompt := buildRewritingPrompt(originalBullet, jobProfile, companyProfile, usedVerbs)

		// Use TierAdvanced for bullet rewriting (requires nuance and style matching)
		responseText, err := client.GenerateContent(ctx, prompt, llm.TierAdvanced)
		if err != nil {
			return nil, &APICallError{
				Message: fmt.Sprintf("failed to generate content for bullet %s", originalBullet.ID),
				Cause:   err,
			}
		}

		// Parse response (expects just the rewritten text)
		rewrittenText, err := parseBulletResponse(responseText)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response for bullet %s: %w", originalBullet.ID, err)
		}

		// Extract leading verb and add to used verbs list
		if verb := extractLeadingVerb(rewrittenText); verb != "" {
			usedVerbs = append(usedVerbs, verb)
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

// RewriteBulletsSelective rewrites only specified bullets, preserving others
func RewriteBulletsSelective(
	ctx context.Context,
	currentBullets *types.RewrittenBullets, // All current bullets
	bulletsToRewrite []string, // IDs of bullets to rewrite
	jobProfile *types.JobProfile,
	companyProfile *types.CompanyProfile,
	experienceBank *types.ExperienceBank,
	apiKey string,
) (*types.RewrittenBullets, error) {
	// If no bullets to rewrite, return preserved bullets immediately
	if len(bulletsToRewrite) == 0 {
		return &types.RewrittenBullets{
			Bullets: currentBullets.Bullets,
		}, nil
	}

	// Build lookup maps
	currentBulletMap := make(map[string]*types.RewrittenBullet)
	for i := range currentBullets.Bullets {
		bullet := &currentBullets.Bullets[i]
		currentBulletMap[bullet.OriginalBulletID] = bullet
	}

	bulletsToRewriteSet := make(map[string]bool)
	for _, bulletID := range bulletsToRewrite {
		bulletsToRewriteSet[bulletID] = true
	}

	// Split bullets into preserve and rewrite
	bulletsToPreserve := make([]types.RewrittenBullet, 0)

	// Collect preserved bullets (not in rewrite set)
	for _, bullet := range currentBullets.Bullets {
		if !bulletsToRewriteSet[bullet.OriginalBulletID] {
			bulletsToPreserve = append(bulletsToPreserve, bullet)
		}
	}

	// Extract verbs from preserved bullets for diversity
	usedVerbs := make([]string, 0)
	for _, bullet := range bulletsToPreserve {
		if verb := extractLeadingVerb(bullet.FinalText); verb != "" {
			usedVerbs = append(usedVerbs, verb)
		}
	}

	// Materialize bullets to rewrite from experienceBank
	// Build bullet ID -> (Bullet, Story) map for efficient lookup
	bulletToStoryMap := make(map[string]*types.Story) // bulletID -> Story
	bulletMap := make(map[string]*types.Bullet)       // bulletID -> Bullet
	for i := range experienceBank.Stories {
		story := &experienceBank.Stories[i]
		for j := range story.Bullets {
			bullet := &story.Bullets[j]
			bulletMap[bullet.ID] = bullet
			bulletToStoryMap[bullet.ID] = story
		}
	}

	// Build SelectedBullets for bullets to rewrite
	selectedBulletsList := make([]types.SelectedBullet, 0, len(bulletsToRewrite))
	for _, bulletID := range bulletsToRewrite {
		bullet, bulletExists := bulletMap[bulletID]
		story, storyExists := bulletToStoryMap[bulletID]

		if !bulletExists || !storyExists {
			// Bullet not found in experienceBank - skip with warning
			// This can happen if bullet was already dropped or not in experienceBank
			continue
		}

		// Copy skills slice to avoid sharing references
		skills := make([]string, len(bullet.Skills))
		copy(skills, bullet.Skills)

		selectedBullet := types.SelectedBullet{
			ID:          bullet.ID,
			StoryID:     story.ID,
			Text:        bullet.Text,
			Skills:      skills,
			Metrics:     bullet.Metrics,
			LengthChars: bullet.LengthChars,
		}

		selectedBulletsList = append(selectedBulletsList, selectedBullet)
	}

	// If no bullets found in experienceBank to rewrite, return preserved bullets
	if len(selectedBulletsList) == 0 {
		return &types.RewrittenBullets{
			Bullets: bulletsToPreserve,
		}, nil
	}

	// Check API key is required for rewriting
	if apiKey == "" {
		return nil, &APICallError{Message: "API key is required"}
	}

	// Rewrite selected bullets using existing function
	selectedBullets := &types.SelectedBullets{
		Bullets: selectedBulletsList,
	}

	// Create a modified version of RewriteBullets that accepts usedVerbs
	// For now, we'll call the existing function and then merge
	rewritten, err := rewriteBulletsWithVerbs(ctx, selectedBullets, jobProfile, companyProfile, usedVerbs, apiKey)
	if err != nil {
		return nil, err
	}

	// Merge preserved and rewritten bullets
	// Build map of rewritten bullets by OriginalBulletID
	rewrittenMap := make(map[string]*types.RewrittenBullet)
	for i := range rewritten.Bullets {
		bullet := &rewritten.Bullets[i]
		rewrittenMap[bullet.OriginalBulletID] = bullet
	}

	// Build final result maintaining order from currentBullets
	finalBullets := make([]types.RewrittenBullet, 0, len(currentBullets.Bullets)+len(rewritten.Bullets))
	for _, currentBullet := range currentBullets.Bullets {
		if bulletsToRewriteSet[currentBullet.OriginalBulletID] {
			// This bullet was rewritten - use rewritten version
			if rewrittenBullet, exists := rewrittenMap[currentBullet.OriginalBulletID]; exists {
				finalBullets = append(finalBullets, *rewrittenBullet)
			} else {
				// Rewritten bullet not found - this shouldn't happen, but preserve original as fallback
				finalBullets = append(finalBullets, currentBullet)
			}
		} else {
			// This bullet was preserved - use original
			finalBullets = append(finalBullets, currentBullet)
		}
	}

	// Add any new bullets that weren't in currentBullets (from swap_story)
	// These are bullets that were rewritten but didn't exist in currentBullets
	for _, rewrittenBullet := range rewritten.Bullets {
		if _, exists := currentBulletMap[rewrittenBullet.OriginalBulletID]; !exists {
			finalBullets = append(finalBullets, rewrittenBullet)
		}
	}

	return &types.RewrittenBullets{
		Bullets: finalBullets,
	}, nil
}

// rewriteBulletsWithVerbs is a helper that rewrites bullets with pre-populated used verbs
func rewriteBulletsWithVerbs(ctx context.Context, selectedBullets *types.SelectedBullets, jobProfile *types.JobProfile, companyProfile *types.CompanyProfile, initialUsedVerbs []string, apiKey string) (*types.RewrittenBullets, error) {
	// Initialize LLM client with default config
	config := llm.DefaultConfig()
	client, err := llm.NewClient(ctx, config, apiKey)
	if err != nil {
		return nil, &APICallError{
			Message: "failed to create LLM client",
			Cause:   err,
		}
	}
	defer func() { _ = client.Close() }()

	// Start with initial used verbs
	usedVerbs := make([]string, len(initialUsedVerbs))
	copy(usedVerbs, initialUsedVerbs)

	// Rewrite each bullet
	rewrittenBullets := make([]types.RewrittenBullet, 0, len(selectedBullets.Bullets))

	for _, originalBullet := range selectedBullets.Bullets {
		// Build rewriting prompt with verbs to avoid
		prompt := buildRewritingPrompt(originalBullet, jobProfile, companyProfile, usedVerbs)

		// Use TierAdvanced for bullet rewriting (requires nuance and style matching)
		responseText, err := client.GenerateContent(ctx, prompt, llm.TierAdvanced)
		if err != nil {
			return nil, &APICallError{
				Message: fmt.Sprintf("failed to generate content for bullet %s", originalBullet.ID),
				Cause:   err,
			}
		}

		// Parse response (expects just the rewritten text)
		rewrittenText, err := parseBulletResponse(responseText)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response for bullet %s: %w", originalBullet.ID, err)
		}

		// Extract leading verb and add to used verbs list
		if verb := extractLeadingVerb(rewrittenText); verb != "" {
			usedVerbs = append(usedVerbs, verb)
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

// extractLeadingVerb extracts the first word (assumed to be a verb) from a bullet point
func extractLeadingVerb(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	// Split on first space to get first word
	parts := strings.SplitN(text, " ", 2)
	if len(parts) > 0 {
		return strings.TrimSuffix(parts[0], ",") // Remove trailing comma if present
	}
	return ""
}

// buildRewritingPrompt constructs the prompt for bullet rewriting
func buildRewritingPrompt(bullet types.SelectedBullet, jobProfile *types.JobProfile, companyProfile *types.CompanyProfile, usedVerbs []string) string {
	var sb strings.Builder

	// Add intro from external prompt
	introTemplate := prompts.MustGet("rewriting.json", "rewrite-bullet-intro")
	sb.WriteString(prompts.Format(introTemplate, map[string]string{
		"BulletText": bullet.Text,
	}))

	// Add job requirements context (dynamic)
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

	// Add company voice context (dynamic)
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

	// Add preservation constraints to prevent hallucination
	preservationTemplate := prompts.MustGet("rewriting.json", "rewrite-bullet-preservation")
	sb.WriteString(preservationTemplate)

	// Add verbs to avoid for diversity
	usedVerbsStr := ""
	if len(usedVerbs) > 0 {
		usedVerbsStr = strings.Join(usedVerbs, ", ")
	}

	// Add requirements from external prompt
	reqsTemplate := prompts.MustGet("rewriting.json", "rewrite-bullet-requirements")
	sb.WriteString(prompts.Format(reqsTemplate, map[string]string{
		"TargetLength": fmt.Sprintf("%d", bullet.LengthChars),
		"UsedVerbs":    usedVerbsStr,
	}))

	return sb.String()
}

// parseBulletResponse parses the API response to extract rewritten text
// The API should return just the text, but we handle JSON wrapper if present
func parseBulletResponse(responseText string) (string, error) {
	text := strings.TrimSpace(responseText)

	// Remove markdown code blocks if present
	if strings.HasPrefix(text, "```") {
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
