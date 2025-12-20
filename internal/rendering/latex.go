// Package rendering provides functionality to render LaTeX resumes from templates.
package rendering

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/jonathan/resume-customizer/internal/types"
)

// TemplateData represents the data structure passed to the LaTeX template
type TemplateData struct {
	Name       string
	Email      string
	Phone      string
	Experience []ExperienceEntry
}

// ExperienceEntry represents a single experience entry in the resume
type ExperienceEntry struct {
	Company   string
	Role      string
	StartDate string
	EndDate   string
	Bullets   []string
}

// RenderLaTeX renders a LaTeX resume from a template using ResumePlan and RewrittenBullets
// Note: This function requires access to ExperienceBank to get story details (company, role, dates).
// The caller is responsible for providing this information or loading the ExperienceBank.
// For now, we'll work with what's available in ResumePlan and RewrittenBullets.
func RenderLaTeX(plan *types.ResumePlan, rewrittenBullets *types.RewrittenBullets, templatePath string, name, email, phone string, experienceBank *types.ExperienceBank) (string, error) {
	// Read and parse template
	tmpl, err := parseTemplate(templatePath)
	if err != nil {
		return "", err
	}

	// Build template data
	data, err := buildTemplateData(plan, rewrittenBullets, name, email, phone, experienceBank)
	if err != nil {
		return "", &RenderError{
			Message: "failed to build template data",
			Cause:   err,
		}
	}

	// Execute template
	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", &TemplateError{
			Message: "failed to execute template",
			Cause:   err,
		}
	}

	return result.String(), nil
}

// parseTemplate reads and parses a LaTeX template file
func parseTemplate(templatePath string) (*template.Template, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &TemplateError{
				Message: fmt.Sprintf("template file not found: %s", templatePath),
				Cause:   err,
			}
		}
		return nil, &TemplateError{
			Message: fmt.Sprintf("failed to read template file: %s", templatePath),
			Cause:   err,
		}
	}

	// Parse template with custom functions for LaTeX escaping
	tmpl, err := template.New("resume").Funcs(template.FuncMap{
		"escape": EscapeLaTeX,
	}).Parse(string(content))
	if err != nil {
		return nil, &TemplateError{
			Message: "failed to parse template",
			Cause:   err,
		}
	}

	return tmpl, nil
}

// buildTemplateData constructs the template data structure from inputs
func buildTemplateData(plan *types.ResumePlan, rewrittenBullets *types.RewrittenBullets, name, email, phone string, experienceBank *types.ExperienceBank) (*TemplateData, error) {
	// Escape contact information
	escapedName := EscapeLaTeX(name)
	escapedEmail := EscapeLaTeX(email)
	escapedPhone := EscapeLaTeX(phone)

	// Format experience section
	experience, err := formatExperience(plan, rewrittenBullets, experienceBank)
	if err != nil {
		return nil, fmt.Errorf("failed to format experience: %w", err)
	}

	return &TemplateData{
		Name:       escapedName,
		Email:      escapedEmail,
		Phone:      escapedPhone,
		Experience: experience,
	}, nil
}

// formatExperience formats the experience section from ResumePlan and RewrittenBullets
func formatExperience(plan *types.ResumePlan, rewrittenBullets *types.RewrittenBullets, experienceBank *types.ExperienceBank) ([]ExperienceEntry, error) {
	if plan == nil || len(plan.SelectedStories) == 0 {
		return []ExperienceEntry{}, nil
	}

	// Build a map of rewritten bullets by original bullet ID for quick lookup
	bulletMap := make(map[string]*types.RewrittenBullet)
	for i := range rewrittenBullets.Bullets {
		bullet := &rewrittenBullets.Bullets[i]
		bulletMap[bullet.OriginalBulletID] = bullet
	}

	// Build a map of stories by story ID for quick lookup (if experienceBank provided)
	storyMap := make(map[string]*types.Story)
	if experienceBank != nil {
		for i := range experienceBank.Stories {
			story := &experienceBank.Stories[i]
			storyMap[story.ID] = story
		}
	}

	// Format each selected story
	experience := make([]ExperienceEntry, 0, len(plan.SelectedStories))
	for _, selectedStory := range plan.SelectedStories {
		// Get story details from experienceBank if available
		var company, role, startDate, endDate string
		if story, found := storyMap[selectedStory.StoryID]; found {
			company = EscapeLaTeX(story.Company)
			role = EscapeLaTeX(story.Role)
			startDate = EscapeLaTeX(story.StartDate)
			endDate = EscapeLaTeX(story.EndDate)
		} else {
			// Fallback: use story ID if experienceBank not provided
			// This is a limitation - ideally we'd always have experienceBank
			company = EscapeLaTeX(selectedStory.StoryID)
			role = EscapeLaTeX("Role") // Placeholder, escaped
			startDate = ""
			endDate = ""
		}

		// Collect bullets for this story
		bullets := make([]string, 0)
		for _, bulletID := range selectedStory.BulletIDs {
			if bullet, found := bulletMap[bulletID]; found {
				// Bullet text is already escaped when building TemplateData
				escapedText := EscapeLaTeX(bullet.FinalText)
				bullets = append(bullets, escapedText)
			}
		}

		experience = append(experience, ExperienceEntry{
			Company:   company,
			Role:      role,
			StartDate: startDate,
			EndDate:   endDate,
			Bullets:   bullets,
		})
	}

	return experience, nil
}
