// Package observability provides formatted output utilities for verbose CLI mode.
package observability

import (
	"fmt"
	"io"
	"strings"

	"github.com/jonathan/resume-customizer/internal/types"
)

const (
	// boxWidth is the default width for formatted output boxes
	boxWidth = 60
	// maxItemsToShow is the default number of items to display in lists
	maxItemsToShow = 5
)

// Printer handles formatted output for verbose mode
type Printer struct {
	out io.Writer
}

// NewPrinter creates a new Printer that writes to the given writer
func NewPrinter(out io.Writer) *Printer {
	return &Printer{out: out}
}

// printBox prints a formatted box with a title and content
//
//nolint:errcheck // writing to stdout; errors are not recoverable
func (p *Printer) printBox(title string, content string) {
	border := strings.Repeat("─", boxWidth-2)
	fmt.Fprintf(p.out, "┌%s┐\n", border)
	fmt.Fprintf(p.out, "│ %-*s │\n", boxWidth-4, title)
	fmt.Fprintf(p.out, "├%s┤\n", border)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Truncate long lines
		if len(line) > boxWidth-4 {
			line = line[:boxWidth-7] + "..."
		}
		fmt.Fprintf(p.out, "│ %-*s │\n", boxWidth-4, line)
	}

	fmt.Fprintf(p.out, "└%s┘\n", border)
}

// PrintJobProfile outputs a human-readable summary of the parsed job profile.
func (p *Printer) PrintJobProfile(profile *types.JobProfile) {
	if profile == nil {
		return
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Company:  %s\n", profile.Company))
	sb.WriteString(fmt.Sprintf("Role:     %s\n", profile.RoleTitle))
	sb.WriteString("\n")

	// Hard requirements
	if len(profile.HardRequirements) > 0 {
		sb.WriteString("Hard Requirements:\n")
		count := min(len(profile.HardRequirements), maxItemsToShow)
		for i := 0; i < count; i++ {
			req := profile.HardRequirements[i]
			sb.WriteString(fmt.Sprintf("  • %s", req.Skill))
			if req.Level != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", req.Level))
			}
			sb.WriteString("\n")
		}
		if len(profile.HardRequirements) > maxItemsToShow {
			sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(profile.HardRequirements)-maxItemsToShow))
		}
		sb.WriteString("\n")
	}

	// Nice to haves
	if len(profile.NiceToHaves) > 0 {
		sb.WriteString("Nice-to-haves:\n")
		count := min(len(profile.NiceToHaves), 3)
		for i := 0; i < count; i++ {
			sb.WriteString(fmt.Sprintf("  • %s\n", profile.NiceToHaves[i].Skill))
		}
		if len(profile.NiceToHaves) > 3 {
			sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(profile.NiceToHaves)-3))
		}
	}

	p.printBox("PARSED JOB PROFILE", strings.TrimSuffix(sb.String(), "\n"))
}

// PrintRankedStories outputs the top N ranked stories with scores and matched skills.
func (p *Printer) PrintRankedStories(stories *types.RankedStories) {
	if stories == nil || len(stories.Ranked) == 0 {
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Total stories ranked: %d\n\n", len(stories.Ranked)))

	count := min(len(stories.Ranked), maxItemsToShow)
	for i := 0; i < count; i++ {
		story := stories.Ranked[i]
		sb.WriteString(fmt.Sprintf("#%d  %s\n", i+1, story.StoryID))
		sb.WriteString(fmt.Sprintf("    Score: %.2f", story.RelevanceScore))
		if story.LLMScore != nil {
			sb.WriteString(fmt.Sprintf(" (LLM: %.2f)", *story.LLMScore))
		}
		sb.WriteString("\n")
		if len(story.MatchedSkills) > 0 {
			skills := strings.Join(story.MatchedSkills, ", ")
			if len(skills) > 40 {
				skills = skills[:37] + "..."
			}
			sb.WriteString(fmt.Sprintf("    Skills: %s\n", skills))
		}
		if i < count-1 {
			sb.WriteString("\n")
		}
	}

	if len(stories.Ranked) > maxItemsToShow {
		sb.WriteString(fmt.Sprintf("\n... and %d more stories", len(stories.Ranked)-maxItemsToShow))
	}

	p.printBox("TOP RANKED STORIES", sb.String())
}

// PrintSelectedBullets outputs the bullets selected before rewriting.
func (p *Printer) PrintSelectedBullets(bullets *types.SelectedBullets) {
	if bullets == nil || len(bullets.Bullets) == 0 {
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Selected %d bullets:\n\n", len(bullets.Bullets)))

	count := min(len(bullets.Bullets), maxItemsToShow)
	for i := 0; i < count; i++ {
		bullet := bullets.Bullets[i]
		text := bullet.Text
		if len(text) > 50 {
			text = text[:47] + "..."
		}
		sb.WriteString(fmt.Sprintf("• %s\n", text))
		if len(bullet.Skills) > 0 {
			skills := strings.Join(bullet.Skills, ", ")
			if len(skills) > 40 {
				skills = skills[:37] + "..."
			}
			sb.WriteString(fmt.Sprintf("  [%s]\n", skills))
		}
		if i < count-1 {
			sb.WriteString("\n")
		}
	}

	if len(bullets.Bullets) > maxItemsToShow {
		sb.WriteString(fmt.Sprintf("\n... and %d more bullets", len(bullets.Bullets)-maxItemsToShow))
	}

	p.printBox("SELECTED BULLETS (before rewrite)", sb.String())
}

// PrintRewrittenBullets outputs the rewritten bullets with style check indicators.
func (p *Printer) PrintRewrittenBullets(bullets *types.RewrittenBullets) {
	if bullets == nil || len(bullets.Bullets) == 0 {
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Rewritten %d bullets:\n\n", len(bullets.Bullets)))

	count := min(len(bullets.Bullets), maxItemsToShow)
	for i := 0; i < count; i++ {
		bullet := bullets.Bullets[i]
		text := bullet.FinalText
		if len(text) > 50 {
			text = text[:47] + "..."
		}
		sb.WriteString(fmt.Sprintf("• %s\n", text))

		// Style checks indicators
		checks := []string{}
		if bullet.StyleChecks.StrongVerb {
			checks = append(checks, "✓verb")
		}
		if bullet.StyleChecks.Quantified {
			checks = append(checks, "✓metrics")
		}
		if bullet.StyleChecks.NoTaboo {
			checks = append(checks, "✓style")
		}
		if bullet.StyleChecks.TargetLength {
			checks = append(checks, "✓length")
		}
		if len(checks) > 0 {
			sb.WriteString(fmt.Sprintf("  [%s]\n", strings.Join(checks, " ")))
		}
		if i < count-1 {
			sb.WriteString("\n")
		}
	}

	if len(bullets.Bullets) > maxItemsToShow {
		sb.WriteString(fmt.Sprintf("\n... and %d more bullets", len(bullets.Bullets)-maxItemsToShow))
	}

	p.printBox("REWRITTEN BULLETS", sb.String())
}

// PrintCompanyProfile outputs the extracted company voice profile.
func (p *Printer) PrintCompanyProfile(profile *types.CompanyProfile) {
	if profile == nil {
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Company:  %s\n", profile.Company))
	sb.WriteString(fmt.Sprintf("Tone:     %s\n", profile.Tone))
	sb.WriteString("\n")

	if len(profile.StyleRules) > 0 {
		sb.WriteString("Style Rules:\n")
		count := min(len(profile.StyleRules), 3)
		for i := 0; i < count; i++ {
			rule := profile.StyleRules[i]
			if len(rule) > 50 {
				rule = rule[:47] + "..."
			}
			sb.WriteString(fmt.Sprintf("  • %s\n", rule))
		}
		if len(profile.StyleRules) > 3 {
			sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(profile.StyleRules)-3))
		}
		sb.WriteString("\n")
	}

	if len(profile.Values) > 0 {
		sb.WriteString("Values:\n")
		count := min(len(profile.Values), 3)
		for i := 0; i < count; i++ {
			sb.WriteString(fmt.Sprintf("  • %s\n", profile.Values[i]))
		}
		if len(profile.Values) > 3 {
			sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(profile.Values)-3))
		}
	}

	p.printBox("COMPANY VOICE PROFILE", strings.TrimSuffix(sb.String(), "\n"))
}

// PrintViolations outputs any constraint violations found.
//
//nolint:errcheck // writing to stdout; errors are not recoverable
func (p *Printer) PrintViolations(violations *types.Violations) {
	if violations == nil || len(violations.Violations) == 0 {
		fmt.Fprintf(p.out, "┌%s┐\n", strings.Repeat("─", boxWidth-2))
		fmt.Fprintf(p.out, "│ %-*s │\n", boxWidth-4, "✅ NO VIOLATIONS FOUND")
		fmt.Fprintf(p.out, "└%s┘\n", strings.Repeat("─", boxWidth-2))
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d violations:\n\n", len(violations.Violations)))

	for i, v := range violations.Violations {
		details := v.Details
		if len(details) > 45 {
			details = details[:42] + "..."
		}
		sb.WriteString(fmt.Sprintf("⚠ %s\n", v.Type))
		sb.WriteString(fmt.Sprintf("  %s\n", details))
		if i < len(violations.Violations)-1 {
			sb.WriteString("\n")
		}
	}

	p.printBox("CONSTRAINT VIOLATIONS", sb.String())
}
