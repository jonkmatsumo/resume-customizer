// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateLineLengths_NoViolations(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
Short line
Another short line
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	violations, err := ValidateLineLengths(texFile, 90)
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestValidateLineLengths_WithViolations(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	// Create a line that exceeds 90 characters
	longLine := strings.Repeat("a", 100)
	content := fmt.Sprintf(`\documentclass{article}
\begin{document}
%s
Short line
\end{document}`, longLine)
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	violations, err := ValidateLineLengths(texFile, 90)
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, "line_too_long", violations[0].Type)
	assert.Equal(t, "warning", violations[0].Severity)
	assert.NotNil(t, violations[0].LineNumber)
	assert.Equal(t, 3, *violations[0].LineNumber)
	assert.NotNil(t, violations[0].CharCount)
	assert.Greater(t, *violations[0].CharCount, 90)
}

func TestValidateLineLengths_SkipsComments(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	longComment := strings.Repeat("a", 100)
	content := fmt.Sprintf(`\documentclass{article}
%% %s
\begin{document}
Short line
\end{document}`, longComment)
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	violations, err := ValidateLineLengths(texFile, 90)
	require.NoError(t, err)
	// Comment lines should be skipped
	assert.Empty(t, violations)
}

func TestValidateLineLengths_FileNotFound(t *testing.T) {
	_, err := ValidateLineLengths("/nonexistent/file.tex", 90)
	assert.Error(t, err)
	var fileErr *FileReadError
	assert.ErrorAs(t, err, &fileErr)
}

func TestCountContentChars_SimpleText(t *testing.T) {
	result := countContentChars("Simple text line")
	assert.Greater(t, result, 0)
	assert.LessOrEqual(t, result, 20)
}

func TestCountContentChars_WithLaTeXCommands(t *testing.T) {
	// \textbf{text} should count "text", not the command
	result := countContentChars(`\textbf{Important text}`)
	// Should count approximately the content length
	assert.Greater(t, result, 0)
	assert.Less(t, result, 30) // Less than full line length
}

func TestCountContentChars_WithComments(t *testing.T) {
	line := "Content here % this is a comment"
	result := countContentChars(line)
	// Should not count the comment part significantly
	assert.Greater(t, result, 0)
}
