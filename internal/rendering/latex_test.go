// Package rendering provides functionality to render LaTeX resumes from templates.
package rendering

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTemplate_ValidTemplate(t *testing.T) {
	// Create a temporary template file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tex")
	templateContent := `\documentclass{article}
\begin{document}
Name: {{.Name}}
\end{document}`
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	tmpl, err := parseTemplate(templatePath)
	require.NoError(t, err)
	assert.NotNil(t, tmpl)
}

func TestParseTemplate_InvalidPath(t *testing.T) {
	_, err := parseTemplate("/nonexistent/template.tex")
	assert.Error(t, err)
	var templateErr *TemplateError
	assert.ErrorAs(t, err, &templateErr)
	assert.Contains(t, err.Error(), "template file not found")
}

func TestParseTemplate_InvalidTemplate(t *testing.T) {
	// Create a temporary file with invalid template syntax
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "invalid.tex")
	templateContent := `\documentclass{article}
\begin{document}
{{.InvalidSyntax{{}}
\end{document}`
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	_, err = parseTemplate(templatePath)
	assert.Error(t, err)
	var templateErr *TemplateError
	assert.ErrorAs(t, err, &templateErr)
}

func TestBuildTemplateData_ValidInput(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
		},
	}

	rewrittenBullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Built a system",
			},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				Company:   "Test Company",
				Role:      "Engineer",
				StartDate: "2020-01",
				EndDate:   "2023-01",
			},
		},
	}

	data, err := buildTemplateData(plan, rewrittenBullets, "John Doe", "john@example.com", "555-1234", experienceBank)
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, "John Doe", data.Name)
	assert.Equal(t, "john@example.com", data.Email)
	assert.Equal(t, "555-1234", data.Phone)
	require.Len(t, data.Companies, 1)
	assert.Equal(t, "Test Company", data.Companies[0].Company)
	require.Len(t, data.Companies[0].Roles, 1)
	assert.Equal(t, "Engineer", data.Companies[0].Roles[0].Role)
}

func TestGroupByCompanyAndRole_ValidInput(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001", "bullet_002"},
			},
			{
				StoryID:   "story_002",
				BulletIDs: []string{"bullet_003"},
			},
		},
	}

	rewrittenBullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "First bullet",
			},
			{
				OriginalBulletID: "bullet_002",
				FinalText:        "Second bullet",
			},
			{
				OriginalBulletID: "bullet_003",
				FinalText:        "Third bullet",
			},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:        "story_001",
				Company:   "Company A",
				Role:      "Engineer",
				StartDate: "2020-01",
				EndDate:   "2023-01",
			},
			{
				ID:        "story_002",
				Company:   "Company B",
				Role:      "Lead",
				StartDate: "2018-01",
				EndDate:   "2020-01",
			},
		},
	}

	companies, err := groupByCompanyAndRole(plan, rewrittenBullets, experienceBank)
	require.NoError(t, err)
	require.Len(t, companies, 2)

	// Check first company
	// Note: Company names are escaped for LaTeX, but "Company A" has no special chars so should match
	assert.Equal(t, "Company A", companies[0].Company)
	require.Len(t, companies[0].Roles, 1)
	// Role names are also escaped, but "Engineer" has no special chars
	assert.Equal(t, "Engineer", companies[0].Roles[0].Role)
	require.Len(t, companies[0].Roles[0].Bullets, 2, "should have 2 bullets")

	// Bullets now include LaTeX comments for mapping (format: % BULLET_START:id\ntext\n% BULLET_END:id)
	// Check that each bullet string contains the expected text
	bullets := companies[0].Roles[0].Bullets
	foundFirst := false
	foundSecond := false
	for _, bullet := range bullets {
		if strings.Contains(bullet, "First bullet") {
			foundFirst = true
			assert.Contains(t, bullet, "bullet_001", "should contain bullet ID")
			assert.Contains(t, bullet, "BULLET_START:bullet_001", "should contain start marker")
			assert.Contains(t, bullet, "BULLET_END:bullet_001", "should contain end marker")
		}
		if strings.Contains(bullet, "Second bullet") {
			foundSecond = true
			assert.Contains(t, bullet, "bullet_002", "should contain bullet ID")
			assert.Contains(t, bullet, "BULLET_START:bullet_002", "should contain start marker")
			assert.Contains(t, bullet, "BULLET_END:bullet_002", "should contain end marker")
		}
	}
	assert.True(t, foundFirst, "should find first bullet with text 'First bullet'")
	assert.True(t, foundSecond, "should find second bullet with text 'Second bullet'")

	// Check second company
	assert.Equal(t, "Company B", companies[1].Company)
	require.Len(t, companies[1].Roles, 1)
	assert.Equal(t, "Lead", companies[1].Roles[0].Role)
	assert.Len(t, companies[1].Roles[0].Bullets, 1)
	// Bullet now includes LaTeX comments, so check that the text is contained
	assert.Contains(t, companies[1].Roles[0].Bullets[0], "Third bullet")
	assert.Contains(t, companies[1].Roles[0].Bullets[0], "bullet_003")
}

func TestGroupByCompanyAndRole_NoExperienceBank(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
		},
	}

	rewrittenBullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Some bullet",
			},
		},
	}

	// No experienceBank provided
	companies, err := groupByCompanyAndRole(plan, rewrittenBullets, nil)
	require.NoError(t, err)
	require.Len(t, companies, 1)
	// Should use story ID as fallback (escaped for LaTeX)
	assert.Equal(t, "story\\_001", companies[0].Company)
}

func TestGroupByCompanyAndRole_EmptyPlan(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{},
	}

	rewrittenBullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{},
	}

	companies, err := groupByCompanyAndRole(plan, rewrittenBullets, nil)
	require.NoError(t, err)
	assert.Empty(t, companies)
}

func TestRenderLaTeX_Success(t *testing.T) {
	// Create a minimal test template
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tex")
	templateContent := `\documentclass{article}
\begin{document}
Name: {{.Name}}
Email: {{.Email}}
{{range .Companies}}
Company: {{.Company}}
{{end}}
\end{document}`
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
		},
	}

	rewrittenBullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Built a system",
			},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test Company",
				Role:    "Engineer",
			},
		},
	}

	latex, _, err := RenderLaTeX(plan, rewrittenBullets, templatePath, "John Doe", "john@example.com", "555-1234", experienceBank, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, latex)
	assert.Contains(t, latex, "John Doe")
	assert.Contains(t, latex, "Test Company")
}

func TestRenderLaTeX_MissingTemplate(t *testing.T) {
	plan := &types.ResumePlan{SelectedStories: []types.SelectedStory{}}
	rewrittenBullets := &types.RewrittenBullets{Bullets: []types.RewrittenBullet{}}

	_, _, err := RenderLaTeX(plan, rewrittenBullets, "/nonexistent/template.tex", "John", "john@example.com", "", nil, nil)
	assert.Error(t, err)
	var templateErr *TemplateError
	assert.ErrorAs(t, err, &templateErr)
}

func TestRenderLaTeX_EscapesSpecialCharacters(t *testing.T) {
	// Create a minimal test template
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tex")
	templateContent := `\documentclass{article}
\begin{document}
Name: {{.Name}}
\end{document}`
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	plan := &types.ResumePlan{SelectedStories: []types.SelectedStory{}}
	rewrittenBullets := &types.RewrittenBullets{Bullets: []types.RewrittenBullet{}}

	// Name with special LaTeX characters
	latex, _, err := RenderLaTeX(plan, rewrittenBullets, templatePath, "John & Jane", "test@example.com", "", nil, nil)
	require.NoError(t, err)
	// Should escape the ampersand
	assert.Contains(t, latex, `\&`)
	assert.NotContains(t, latex, "John & Jane")
}

func TestRenderLaTeX_EscapesBulletText(t *testing.T) {
	// Create a minimal test template
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tex")
	templateContent := `\documentclass{article}
\begin{document}
{{range .Companies}}
{{range .Roles}}
{{range .Bullets}}
Bullet: {{.}}
{{end}}
{{end}}
{{end}}
\end{document}`
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
		},
	}

	rewrittenBullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Built system with $1M budget",
			},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Company",
				Role:    "Role",
			},
		},
	}

	latex, _, err := RenderLaTeX(plan, rewrittenBullets, templatePath, "Name", "email@example.com", "", experienceBank, nil)
	require.NoError(t, err)
	// Should escape the dollar sign - verify escaped version exists
	assert.Contains(t, latex, `\$1M`, "should contain escaped dollar sign before '1M'")
	// Check that we don't have unescaped pattern "with $1M" (dollar sign not preceded by backslash)
	unescapedPattern := "with $1M"
	assert.NotContains(t, latex, unescapedPattern, "should not contain unescaped dollar sign")
}

func TestRenderLaTeX_WithEducation(t *testing.T) {
	// Create a minimal test template
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tex")
	templateContent := `\documentclass{article}
\begin{document}
{{range .Education}}
School: {{.School}}
Degree: {{.Degree}}
{{end}}
\end{document}`
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	plan := &types.ResumePlan{SelectedStories: []types.SelectedStory{}}
	bullets := &types.RewrittenBullets{Bullets: []types.RewrittenBullet{}}
	education := []types.Education{
		{
			ID:     "edu_001",
			School: "MIT",
			Degree: "master",
			Field:  "CS",
		},
	}

	latex, _, err := RenderLaTeX(plan, bullets, templatePath, "Name", "email@example.com", "", nil, education)
	require.NoError(t, err)
	assert.Contains(t, latex, "School: MIT")
	assert.Contains(t, latex, "Degree: Master") // Should be normalized/capitalized if your code does that
}

func TestParseBulletMarkers_SingleBullet(t *testing.T) {
	latex := `\documentclass{article}
\begin{document}
\begin{itemize}
% BULLET_START:bullet_001
    \item Test bullet text
% BULLET_END:bullet_001
\end{itemize}
\end{document}`

	mapping := parseBulletMarkers(latex)
	require.NotNil(t, mapping)

	// Check line-to-bullet mapping
	// The \item line should map to bullet_001
	// Line numbers: 1=documentclass, 2=begin, 3=itemize, 4=BULLET_START, 5=\item, 6=BULLET_END
	assert.Equal(t, "bullet_001", mapping.LineToBullet[5]) // \item line

	// Check bullet-to-line mapping
	lines := mapping.BulletToLine["bullet_001"]
	assert.Contains(t, lines, 5) // \item line should be included
}

func TestParseBulletMarkers_MultipleBullets(t *testing.T) {
	latex := `\documentclass{article}
\begin{document}
\begin{itemize}
% BULLET_START:bullet_001
    \item First bullet
% BULLET_END:bullet_001
% BULLET_START:bullet_002
    \item Second bullet
% BULLET_END:bullet_002
\end{itemize}
\end{document}`

	mapping := parseBulletMarkers(latex)
	require.NotNil(t, mapping)

	// Both bullets should be mapped
	assert.Equal(t, "bullet_001", mapping.LineToBullet[5]) // First \item
	assert.Equal(t, "bullet_002", mapping.LineToBullet[8]) // Second \item

	// Check bullet-to-line mappings
	assert.Contains(t, mapping.BulletToLine["bullet_001"], 5)
	assert.Contains(t, mapping.BulletToLine["bullet_002"], 8)
}

func TestParseBulletMarkers_MultipleStories(t *testing.T) {
	latex := `\documentclass{article}
\begin{document}
\begin{itemize}
% BULLET_START:bullet_001
    \item Story 1 bullet 1
% BULLET_END:bullet_001
% BULLET_START:bullet_002
    \item Story 1 bullet 2
% BULLET_END:bullet_002
% BULLET_START:bullet_003
    \item Story 2 bullet 1
% BULLET_END:bullet_003
\end{itemize}
\end{document}`

	mapping := parseBulletMarkers(latex)
	require.NotNil(t, mapping)

	// All three bullets should be mapped
	assert.Equal(t, "bullet_001", mapping.LineToBullet[5])
	assert.Equal(t, "bullet_002", mapping.LineToBullet[8])
	assert.Equal(t, "bullet_003", mapping.LineToBullet[11])

	// Check all bullets are in the mapping
	assert.Equal(t, 3, len(mapping.BulletToLine))
}

func TestParseBulletMarkers_NoMarkers(t *testing.T) {
	latex := `\documentclass{article}
\begin{document}
\begin{itemize}
    \item Regular bullet without markers
\end{itemize}
\end{document}`

	mapping := parseBulletMarkers(latex)
	require.NotNil(t, mapping)

	// Should have empty mappings
	assert.Equal(t, 0, len(mapping.LineToBullet))
	assert.Equal(t, 0, len(mapping.BulletToLine))
}

func TestRenderLaTeX_GeneratesBulletMarkers(t *testing.T) {
	// Create a minimal test template
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tex")
	templateContent := `\documentclass{article}
\begin{document}
{{range .Companies}}
{{range .Roles}}
\begin{itemize}
{{range .Bullets}}
    {{.}}
{{end}}
\end{itemize}
{{end}}
{{end}}
\end{document}`
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001"},
			},
		},
	}

	rewrittenBullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "Test bullet text",
			},
		},
	}

	experienceBank := &types.ExperienceBank{
		Stories: []types.Story{
			{
				ID:      "story_001",
				Company: "Test Company",
				Role:    "Test Role",
			},
		},
	}

	latex, mapping, err := RenderLaTeX(plan, rewrittenBullets, templatePath, "Name", "email@example.com", "", experienceBank, nil)
	require.NoError(t, err)

	// Check that markers are in the LaTeX
	assert.Contains(t, latex, "% BULLET_START:bullet_001")
	assert.Contains(t, latex, "% BULLET_END:bullet_001")

	// Check that mapping was generated
	require.NotNil(t, mapping)
	assert.Greater(t, len(mapping.LineToBullet), 0, "should have line-to-bullet mappings")
	assert.Greater(t, len(mapping.BulletToLine), 0, "should have bullet-to-line mappings")
}
