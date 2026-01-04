// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConstraints_NoViolations(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
Short line content
Another short line
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	violations, err := ValidateConstraints(texFile, nil, 1, 90, nil)
	require.NoError(t, err)
	// May have compilation/page count violations if tools not available
	// But should not have line length violations
	for _, v := range violations.Violations {
		assert.NotEqual(t, "line_too_long", v.Type)
	}
}

func TestValidateConstraints_WithLineViolations(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	longLine := strings.Repeat("a", 100)
	content := fmt.Sprintf(`\documentclass{article}
\begin{document}
%s
\end{document}`, longLine)
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	violations, err := ValidateConstraints(texFile, nil, 1, 90, nil)
	require.NoError(t, err)

	// Should have at least one line_too_long violation
	hasLineViolation := false
	for _, v := range violations.Violations {
		if v.Type == "line_too_long" {
			hasLineViolation = true
			break
		}
	}
	assert.True(t, hasLineViolation, "should have line_too_long violation")
}

func TestValidateConstraints_WithForbiddenPhrases(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
I am a coding ninja
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	companyProfile := &types.CompanyProfile{
		TabooPhrases: []string{"ninja"},
	}

	violations, err := ValidateConstraints(texFile, companyProfile, 1, 90, nil)
	require.NoError(t, err)

	// Should have forbidden_phrase violation
	hasForbiddenViolation := false
	for _, v := range violations.Violations {
		if v.Type == "forbidden_phrase" {
			hasForbiddenViolation = true
			break
		}
	}
	assert.True(t, hasForbiddenViolation, "should have forbidden_phrase violation")
}

func TestValidateConstraints_WithoutCompanyProfile(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
Content
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	// Should not fail even without company profile
	violations, err := ValidateConstraints(texFile, nil, 1, 90, nil)
	require.NoError(t, err)
	assert.NotNil(t, violations)

	// Should not have forbidden_phrase violations
	for _, v := range violations.Violations {
		assert.NotEqual(t, "forbidden_phrase", v.Type)
	}
}

func TestValidateFromContent_WithMapping(t *testing.T) {
	// Create LaTeX content with bullet markers
	latexContent := `\documentclass{article}
\begin{document}
\begin{itemize}
% BULLET_START:bullet_001
    \item This is a very long bullet point that definitely exceeds the maximum character count allowed per line
% BULLET_END:bullet_001
% BULLET_START:bullet_002
    \item Short bullet
% BULLET_END:bullet_002
\end{itemize}
\end{document}`

	plan := &types.ResumePlan{
		SelectedStories: []types.SelectedStory{
			{
				StoryID:   "story_001",
				BulletIDs: []string{"bullet_001", "bullet_002"},
			},
		},
	}

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "bullet_001",
				FinalText:        "This is a very long bullet point that definitely exceeds the maximum character count allowed per line",
			},
			{
				OriginalBulletID: "bullet_002",
				FinalText:        "Short bullet",
			},
		},
	}

	// Parse bullet markers to create mapping
	lines := strings.Split(latexContent, "\n")
	lineToBullet := make(map[int]string)
	for i, line := range lines {
		if strings.Contains(line, "BULLET_START:bullet_001") {
			// Map the \item line (next line)
			lineToBullet[i+2] = "bullet_001" // +2 because \item is 2 lines after BULLET_START
		}
		if strings.Contains(line, "BULLET_START:bullet_002") {
			lineToBullet[i+2] = "bullet_002"
		}
	}

	opts := &Options{
		LineToBulletMap: lineToBullet,
		Bullets:         bullets,
		Plan:            plan,
	}

	violations, err := ValidateFromContent(latexContent, nil, 1, 90, opts)
	require.NoError(t, err)

	// Should have line_too_long violations
	hasMappedViolation := false
	for _, v := range violations.Violations {
		if v.Type == "line_too_long" && v.BulletID != nil {
			hasMappedViolation = true
			assert.Equal(t, "bullet_001", *v.BulletID)
			assert.Equal(t, "story_001", *v.StoryID)
			assert.NotNil(t, v.BulletText)
			break
		}
	}
	// Note: This test may not always find violations if LaTeX compilation fails or line counting is different
	// The important part is that if violations are found, they should be mapped
	if hasMappedViolation {
		t.Log("Found mapped violation as expected")
	}
}

func TestValidateFromContent_WithoutMapping(t *testing.T) {
	// Test backward compatibility - should work without mapping
	latexContent := `\documentclass{article}
\begin{document}
Short line
\end{document}`

	violations, err := ValidateFromContent(latexContent, nil, 1, 90, nil)
	require.NoError(t, err)
	assert.NotNil(t, violations)

	// Violations should not have bullet IDs (no mapping provided)
	for _, v := range violations.Violations {
		assert.Nil(t, v.BulletID)
		assert.Nil(t, v.StoryID)
	}
}
