// Package rendering provides functionality to render LaTeX resumes from templates.
package rendering

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeLaTeX_EmptyString(t *testing.T) {
	result := EscapeLaTeX("")
	assert.Equal(t, "", result)
}

func TestEscapeLaTeX_NoSpecialCharacters(t *testing.T) {
	text := "This is normal text with no special characters"
	result := EscapeLaTeX(text)
	assert.Equal(t, text, result)
}

func TestEscapeLaTeX_Backslash(t *testing.T) {
	result := EscapeLaTeX("test\\backslash")
	assert.Equal(t, "test\\textbackslash{}backslash", result)
}

func TestEscapeLaTeX_CurlyBraces(t *testing.T) {
	result := EscapeLaTeX("text{with}braces")
	assert.Equal(t, "text\\{with\\}braces", result)
}

func TestEscapeLaTeX_DollarSign(t *testing.T) {
	result := EscapeLaTeX("cost $100")
	assert.Equal(t, "cost \\$100", result)
}

func TestEscapeLaTeX_Ampersand(t *testing.T) {
	result := EscapeLaTeX("A & B")
	assert.Equal(t, "A \\& B", result)
}

func TestEscapeLaTeX_Percent(t *testing.T) {
	result := EscapeLaTeX("100% complete")
	assert.Equal(t, "100\\% complete", result)
}

func TestEscapeLaTeX_Hash(t *testing.T) {
	result := EscapeLaTeX("issue #123")
	assert.Equal(t, "issue \\#123", result)
}

func TestEscapeLaTeX_Caret(t *testing.T) {
	result := EscapeLaTeX("x^2")
	assert.Equal(t, "x\\textasciicircum{}2", result)
}

func TestEscapeLaTeX_Underscore(t *testing.T) {
	result := EscapeLaTeX("variable_name")
	assert.Equal(t, "variable\\_name", result)
}

func TestEscapeLaTeX_Tilde(t *testing.T) {
	result := EscapeLaTeX("~approx")
	assert.Equal(t, "\\textasciitilde{}approx", result)
}

func TestEscapeLaTeX_MultipleSpecialCharacters(t *testing.T) {
	result := EscapeLaTeX("test${}~&%#^_\\")
	expected := "test\\$\\{\\}\\textasciitilde{}\\&\\%\\#\\textasciicircum{}\\_\\textbackslash{}"
	assert.Equal(t, expected, result)
}

func TestEscapeLaTeX_UnicodeCharacters(t *testing.T) {
	text := "résumé with unicode: α β γ"
	result := EscapeLaTeX(text)
	// Unicode should pass through unchanged
	assert.Equal(t, text, result)
}

func TestEscapeLaTeX_MixedContent(t *testing.T) {
	text := "Built system handling $1M+ requests/day with 99.9% uptime"
	result := EscapeLaTeX(text)
	assert.Contains(t, result, "\\$1M")
	assert.Contains(t, result, "99.9\\%")
	assert.Contains(t, result, "requests/day")
}
