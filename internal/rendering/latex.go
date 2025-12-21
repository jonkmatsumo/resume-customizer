// Package rendering provides functionality to render LaTeX resumes from templates.
package rendering

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/jonathan/resume-customizer/internal/types"
)

// TemplateData represents the data structure passed to the LaTeX template
type TemplateData struct {
	Name      string
	Email     string
	Phone     string
	Companies []CompanySection
}

// CompanySection represents a company with one or more roles
type CompanySection struct {
	Company string
	Roles   []RoleSection
}

// RoleSection represents a role within a company with merged date ranges
type RoleSection struct {
	Role       string
	DateRanges string // e.g., "08/2020 - 10/2021, 07/2023 - 10/2023"
	Bullets    []string
}

// dateRange represents a single date range for sorting
type dateRange struct {
	StartDate string
	EndDate   string
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

	// Format experience section with grouping
	companies, err := groupByCompanyAndRole(plan, rewrittenBullets, experienceBank)
	if err != nil {
		return nil, fmt.Errorf("failed to format experience: %w", err)
	}

	return &TemplateData{
		Name:      escapedName,
		Email:     escapedEmail,
		Phone:     escapedPhone,
		Companies: companies,
	}, nil
}

// roleKey is used for grouping bullets by company and role
type roleKey struct {
	Company string
	Role    string
}

// bulletWithMeta holds bullet text along with its date range info
type bulletWithMeta struct {
	Text      string
	StartDate string
	EndDate   string
}

// groupByCompanyAndRole groups bullets by Company, then by Role, merging date ranges
func groupByCompanyAndRole(plan *types.ResumePlan, rewrittenBullets *types.RewrittenBullets, experienceBank *types.ExperienceBank) ([]CompanySection, error) {
	if plan == nil || len(plan.SelectedStories) == 0 {
		return []CompanySection{}, nil
	}

	// Build a map of rewritten bullets by original bullet ID
	bulletMap := make(map[string]*types.RewrittenBullet)
	for i := range rewrittenBullets.Bullets {
		bullet := &rewrittenBullets.Bullets[i]
		bulletMap[bullet.OriginalBulletID] = bullet
	}

	// Build a map of stories by story ID
	storyMap := make(map[string]*types.Story)
	if experienceBank != nil {
		for i := range experienceBank.Stories {
			story := &experienceBank.Stories[i]
			storyMap[story.ID] = story
		}
	}

	// Collect all bullets with their metadata, grouped by (Company, Role)
	roleData := make(map[roleKey][]bulletWithMeta)
	companyOrder := []string{}                    // Track order companies appear
	companyRoleOrder := make(map[string][]string) // Track order roles appear within each company
	seenCompanies := make(map[string]bool)
	seenRoles := make(map[roleKey]bool)

	for _, selectedStory := range plan.SelectedStories {
		story, found := storyMap[selectedStory.StoryID]
		if !found {
			// Fallback if story not in bank
			story = &types.Story{
				ID:        selectedStory.StoryID,
				Company:   selectedStory.StoryID,
				Role:      "Role",
				StartDate: "",
				EndDate:   "",
			}
		}

		key := roleKey{Company: story.Company, Role: story.Role}

		// Track company order
		if !seenCompanies[story.Company] {
			seenCompanies[story.Company] = true
			companyOrder = append(companyOrder, story.Company)
		}

		// Track role order within company
		if !seenRoles[key] {
			seenRoles[key] = true
			companyRoleOrder[story.Company] = append(companyRoleOrder[story.Company], story.Role)
		}

		// Add bullets for this story
		for _, bulletID := range selectedStory.BulletIDs {
			if bullet, ok := bulletMap[bulletID]; ok {
				roleData[key] = append(roleData[key], bulletWithMeta{
					Text:      EscapeLaTeX(bullet.FinalText),
					StartDate: story.StartDate,
					EndDate:   story.EndDate,
				})
			}
		}
	}

	// Build output structure
	companies := make([]CompanySection, 0, len(companyOrder))

	// Track latest end date for each company (for sorting)
	companyEndDates := make(map[string]string)

	for _, companyName := range companyOrder {
		roles := make([]RoleSection, 0)
		latestEndDate := ""

		for _, roleName := range companyRoleOrder[companyName] {
			key := roleKey{Company: companyName, Role: roleName}
			bullets := roleData[key]
			if len(bullets) == 0 {
				continue
			}

			// Collect and merge date ranges
			dateRanges := mergeDateRanges(bullets)

			// Track the latest end date for this company
			for _, b := range bullets {
				if b.EndDate > latestEndDate || b.EndDate == "present" {
					latestEndDate = b.EndDate
				}
			}

			// Extract bullet texts
			bulletTexts := make([]string, len(bullets))
			for i, b := range bullets {
				bulletTexts[i] = b.Text
			}

			roles = append(roles, RoleSection{
				Role:       EscapeLaTeX(roleName),
				DateRanges: dateRanges,
				Bullets:    bulletTexts,
			})
		}

		companyEndDates[companyName] = latestEndDate

		companies = append(companies, CompanySection{
			Company: EscapeLaTeX(companyName),
			Roles:   roles,
		})
	}

	// Sort companies by end date (most recent first)
	// "present" is treated as the latest possible date
	sort.Slice(companies, func(i, j int) bool {
		endI := companyEndDates[companies[i].Company]
		endJ := companyEndDates[companies[j].Company]

		// "present" or empty string treated as most recent
		if endI == "present" || endI == "" {
			return true
		}
		if endJ == "present" || endJ == "" {
			return false
		}

		// Compare dates (YYYY-MM format, lexicographic comparison works)
		return endI > endJ
	})

	return companies, nil
}

// mergeDateRanges collects unique date ranges from bullets, sorts them, and formats as comma-separated string
func mergeDateRanges(bullets []bulletWithMeta) string {
	// Collect unique date ranges
	seen := make(map[string]bool)
	ranges := []dateRange{}
	for _, b := range bullets {
		if b.StartDate == "" && b.EndDate == "" {
			continue
		}
		key := b.StartDate + "-" + b.EndDate
		if !seen[key] {
			seen[key] = true
			ranges = append(ranges, dateRange{StartDate: b.StartDate, EndDate: b.EndDate})
		}
	}

	if len(ranges) == 0 {
		return ""
	}

	// Sort by start date (chronologically)
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].StartDate < ranges[j].StartDate
	})

	// Format as comma-separated
	parts := make([]string, len(ranges))
	for i, r := range ranges {
		if r.EndDate == "present" {
			parts[i] = EscapeLaTeX(r.StartDate) + " -- Present"
		} else {
			parts[i] = EscapeLaTeX(r.StartDate) + " -- " + EscapeLaTeX(r.EndDate)
		}
	}

	return strings.Join(parts, ", ")
}
