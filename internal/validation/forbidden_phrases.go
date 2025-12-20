// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jonathan/resume-customizer/internal/types"
)

// CheckForbiddenPhrases checks if the LaTeX file contains any forbidden phrases
func CheckForbiddenPhrases(texPath string, tabooPhrases []string) ([]types.Violation, error) {
	if len(tabooPhrases) == 0 {
		return []types.Violation{}, nil
	}

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

		// Normalize line for matching (unescape LaTeX, convert to lowercase)
		normalizedLine := normalizeForMatching(line)

		// Check each taboo phrase
		for _, phrase := range tabooPhrases {
			normalizedPhrase := strings.ToLower(strings.TrimSpace(phrase))
			if normalizedPhrase == "" {
				continue
			}

			// Case-insensitive search
			if strings.Contains(normalizedLine, normalizedPhrase) {
				violations = append(violations, types.Violation{
					Type:       "forbidden_phrase",
					Severity:   "error",
					Details:    fmt.Sprintf("Line %d contains forbidden phrase: %s", lineNum, phrase),
					LineNumber: intPtr(lineNum),
				})
				break // Only report one violation per line (first match)
			}
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

// normalizeForMatching normalizes LaTeX content for phrase matching
// It unescapes LaTeX special characters and converts to lowercase
func normalizeForMatching(text string) string {
	// First unescape common LaTeX escapes (must do before comment removal)
	text = strings.ReplaceAll(text, `\$`, "$")
	text = strings.ReplaceAll(text, `\&`, "&")
	text = strings.ReplaceAll(text, `\%`, "%")
	text = strings.ReplaceAll(text, `\#`, "#")
	text = strings.ReplaceAll(text, `\_`, "_")
	text = strings.ReplaceAll(text, `\{`, "{")
	text = strings.ReplaceAll(text, `\}`, "}")
	text = strings.ReplaceAll(text, `\textbackslash{}`, "\\")

	// Remove LaTeX comments (after unescaping, so \% doesn't trigger comment removal)
	if idx := strings.Index(text, "%"); idx >= 0 {
		text = text[:idx]
	}

	// Convert to lowercase for case-insensitive matching
	return strings.ToLower(text)
}
