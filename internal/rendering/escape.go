// Package rendering provides functionality to render LaTeX resumes from templates.
package rendering

import "strings"

// EscapeLaTeX escapes special LaTeX characters in text
// Special characters: \ { } $ & % # ^ _ ~
func EscapeLaTeX(text string) string {
	if text == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(text) * 2) // Pre-allocate space for potential escaping

	for _, r := range text {
		switch r {
		case '\\':
			result.WriteString(`\textbackslash{}`)
		case '{':
			result.WriteString(`\{`)
		case '}':
			result.WriteString(`\}`)
		case '$':
			result.WriteString(`\$`)
		case '&':
			result.WriteString(`\&`)
		case '%':
			result.WriteString(`\%`)
		case '#':
			result.WriteString(`\#`)
		case '^':
			result.WriteString(`\textasciicircum{}`)
		case '_':
			result.WriteString(`\_`)
		case '~':
			result.WriteString(`\textasciitilde{}`)
		default:
			result.WriteRune(r)
		}
	}

	return result.String()
}
