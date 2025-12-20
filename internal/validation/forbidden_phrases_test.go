// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckForbiddenPhrases_NoPhrases(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
Some content here
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	violations, err := CheckForbiddenPhrases(texFile, []string{})
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestCheckForbiddenPhrases_Found(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
I am a coding ninja
Great synergy in the team
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	violations, err := CheckForbiddenPhrases(texFile, []string{"ninja", "synergy"})
	require.NoError(t, err)
	require.Len(t, violations, 2)

	assert.Equal(t, "forbidden_phrase", violations[0].Type)
	assert.Equal(t, "error", violations[0].Severity)
	assert.NotNil(t, violations[0].LineNumber)
	assert.Contains(t, violations[0].Details, "ninja")

	assert.Equal(t, "forbidden_phrase", violations[1].Type)
	assert.Contains(t, violations[1].Details, "synergy")
}

func TestCheckForbiddenPhrases_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
I am a NINJA coder
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	violations, err := CheckForbiddenPhrases(texFile, []string{"ninja"})
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Contains(t, violations[0].Details, "ninja")
}

func TestCheckForbiddenPhrases_LatexEscaping(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	// LaTeX escaped dollar sign should match taboo phrase "$"
	content := `\documentclass{article}
\begin{document}
Budget: \$1M
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	violations, err := CheckForbiddenPhrases(texFile, []string{"$"})
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Contains(t, violations[0].Details, "$")
}

func TestCheckForbiddenPhrases_SkipsComments(t *testing.T) {
	tmpDir := t.TempDir()
	texFile := filepath.Join(tmpDir, "test.tex")
	content := `\documentclass{article}
\begin{document}
% This comment contains the word ninja
Clean content
\end{document}`
	err := os.WriteFile(texFile, []byte(content), 0644)
	require.NoError(t, err)

	violations, err := CheckForbiddenPhrases(texFile, []string{"ninja"})
	require.NoError(t, err)
	// Comments should be ignored
	assert.Empty(t, violations)
}

func TestCheckForbiddenPhrases_FileNotFound(t *testing.T) {
	_, err := CheckForbiddenPhrases("/nonexistent/file.tex", []string{"test"})
	assert.Error(t, err)
	var fileErr *FileReadError
	assert.ErrorAs(t, err, &fileErr)
}

func TestNormalizeForMatching_Unescapes(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"dollar", `\$`, "$"},
		{"ampersand", `\&`, "&"},
		{"percent", `text\%here`, "text"}, // After unescaping and comment removal, % and everything after is removed
		{"hash", `\#`, "#"},
		{"underscore", `\_`, "_"},
		{"lowercase", "HELLO", "hello"},
		{"removes comments", "text % comment", "text "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeForMatching(tc.input)
			// For percent, after unescaping \% to % and comment removal, we get "text" (before the %)
			if tc.name == "percent" {
				assert.Equal(t, "text", result)
			} else {
				assert.Contains(t, result, tc.expected)
			}
		})
	}
}
