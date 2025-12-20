// Package rendering provides functionality to render LaTeX resumes from templates.
package rendering

import (
	"os"
	"path/filepath"
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
	assert.Len(t, data.Experience, 1)
	assert.Equal(t, "Test Company", data.Experience[0].Company)
	assert.Equal(t, "Engineer", data.Experience[0].Role)
}

func TestFormatExperience_ValidInput(t *testing.T) {
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

	experience, err := formatExperience(plan, rewrittenBullets, experienceBank)
	require.NoError(t, err)
	require.Len(t, experience, 2)

	// Check first story
	assert.Equal(t, "Company A", experience[0].Company)
	assert.Equal(t, "Engineer", experience[0].Role)
	assert.Len(t, experience[0].Bullets, 2)
	assert.Contains(t, experience[0].Bullets, "First bullet")
	assert.Contains(t, experience[0].Bullets, "Second bullet")

	// Check second story
	assert.Equal(t, "Company B", experience[1].Company)
	assert.Equal(t, "Lead", experience[1].Role)
	assert.Len(t, experience[1].Bullets, 1)
	assert.Contains(t, experience[1].Bullets, "Third bullet")
}

func TestFormatExperience_NoExperienceBank(t *testing.T) {
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
	experience, err := formatExperience(plan, rewrittenBullets, nil)
	require.NoError(t, err)
	require.Len(t, experience, 1)
	// Should use story ID as fallback (escaped for LaTeX)
	assert.Equal(t, "story\\_001", experience[0].Company)
}

func TestFormatExperience_EmptyPlan(t *testing.T) {
	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{},
	}

	rewrittenBullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{},
	}

	experience, err := formatExperience(plan, rewrittenBullets, nil)
	require.NoError(t, err)
	assert.Empty(t, experience)
}

func TestRenderLaTeX_Success(t *testing.T) {
	// Create a minimal test template
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tex")
	templateContent := `\documentclass{article}
\begin{document}
Name: {{.Name}}
Email: {{.Email}}
{{range .Experience}}
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

	latex, err := RenderLaTeX(plan, rewrittenBullets, templatePath, "John Doe", "john@example.com", "555-1234", experienceBank)
	require.NoError(t, err)
	assert.NotEmpty(t, latex)
	assert.Contains(t, latex, "John Doe")
	assert.Contains(t, latex, "Test Company")
}

func TestRenderLaTeX_MissingTemplate(t *testing.T) {
	plan := &types.ResumePlan{SelectedStories: []types.SelectedStory{}}
	rewrittenBullets := &types.RewrittenBullets{Bullets: []types.RewrittenBullet{}}

	_, err := RenderLaTeX(plan, rewrittenBullets, "/nonexistent/template.tex", "John", "john@example.com", "", nil)
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
	latex, err := RenderLaTeX(plan, rewrittenBullets, templatePath, "John & Jane", "test@example.com", "", nil)
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
{{range .Experience}}
{{range .Bullets}}
Bullet: {{.}}
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

	latex, err := RenderLaTeX(plan, rewrittenBullets, templatePath, "Name", "email@example.com", "", experienceBank)
	require.NoError(t, err)
	// Should escape the dollar sign - verify escaped version exists
	assert.Contains(t, latex, `\$1M`, "should contain escaped dollar sign before '1M'")
	// Check that we don't have unescaped pattern "with $1M" (dollar sign not preceded by backslash)
	unescapedPattern := "with $1M"
	assert.NotContains(t, latex, unescapedPattern, "should not contain unescaped dollar sign")
}
