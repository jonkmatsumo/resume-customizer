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

	violations, err := ValidateConstraints(texFile, nil, 1, 90)
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

	violations, err := ValidateConstraints(texFile, nil, 1, 90)
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

	violations, err := ValidateConstraints(texFile, companyProfile, 1, 90)
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
	violations, err := ValidateConstraints(texFile, nil, 1, 90)
	require.NoError(t, err)
	assert.NotNil(t, violations)
	
	// Should not have forbidden_phrase violations
	for _, v := range violations.Violations {
		assert.NotEqual(t, "forbidden_phrase", v.Type)
	}
}

