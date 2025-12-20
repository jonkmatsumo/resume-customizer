// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jonathan/resume-customizer/internal/types"
)

var (
	// LaTeX command pattern matches commands like \textbf{content} or \begin{environment}
	latexCommandPattern = regexp.MustCompile(`\\([a-zA-Z]+|.)\{[^}]*\}`)
	// Comment pattern matches LaTeX comments (% ...)
	commentPattern = regexp.MustCompile(`%.*$`)
)

// ValidateLineLengths checks if any lines in the LaTeX file exceed the maximum character count
func ValidateLineLengths(texPath string, maxChars int) ([]types.Violation, error) {
	file, err := os.Open(texPath)
	if err != nil {
		return nil, &FileReadError{
			Message: fmt.Sprintf("failed to open LaTeX file: %s", texPath),
			Cause:   err,
		}
	}
	defer func() { _ = file.Close() }()

	var violations []types.Violation
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip comment-only lines
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "%") {
			continue
		}

		// Remove comments from the line
		lineWithoutComments := commentPattern.ReplaceAllString(line, "")

		// Count characters in content (approximate by removing LaTeX commands)
		// For simplicity, we'll count the entire line length, but this could be refined
		// to parse LaTeX commands and count only content
		contentLength := countContentChars(lineWithoutComments)

		if contentLength > maxChars {
			violations = append(violations, types.Violation{
				Type:       "line_too_long",
				Severity:   "warning",
				Details:    fmt.Sprintf("Line %d has %d characters, maximum is %d", lineNum, contentLength, maxChars),
				LineNumber: intPtr(lineNum),
				CharCount:  intPtr(contentLength),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, &FileReadError{
			Message: "failed to read LaTeX file",
			Cause:   err,
		}
	}

	return violations, nil
}

// countContentChars approximates the character count of actual content in a LaTeX line
// This is a simplified implementation that removes LaTeX commands and counts remaining text
func countContentChars(line string) int {
	// Remove LaTeX commands like \textbf{text} - we want to count "text", not the command
	// For simplicity, remove all LaTeX commands and count the remaining content
	processed := latexCommandPattern.ReplaceAllStringFunc(line, func(match string) string {
		// Extract content from commands like \command{content}
		if strings.HasPrefix(match, "\\") && strings.Contains(match, "{") {
			start := strings.Index(match, "{")
			end := strings.LastIndex(match, "}")
			if start >= 0 && end > start {
				return match[start+1 : end]
			}
		}
		return ""
	})

	// Trim whitespace and count remaining characters
	trimmed := strings.TrimSpace(processed)
	return len([]rune(trimmed))
}

// intPtr returns a pointer to an integer
func intPtr(i int) *int {
	return &i
}
