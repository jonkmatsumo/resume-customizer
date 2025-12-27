package observability

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestPrintJobProfile(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	profile := &types.JobProfile{
		Company:   "Acme Corp",
		RoleTitle: "Senior Engineer",
		HardRequirements: []types.Requirement{
			{Skill: "Go", Level: "expert"},
			{Skill: "Kubernetes"},
		},
		NiceToHaves: []types.Requirement{
			{Skill: "Rust"},
		},
	}

	p.PrintJobProfile(profile)
	output := buf.String()

	assert.Contains(t, output, "PARSED JOB PROFILE")
	assert.Contains(t, output, "Acme Corp")
	assert.Contains(t, output, "Senior Engineer")
	assert.Contains(t, output, "Go")
	assert.Contains(t, output, "expert")
	assert.Contains(t, output, "Rust")
}

func TestPrintJobProfile_Nil(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	p.PrintJobProfile(nil)

	assert.Empty(t, buf.String())
}

func TestPrintRankedStories(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	llmScore := 0.95
	stories := &types.RankedStories{
		Ranked: []types.RankedStory{
			{
				StoryID:        "story-1",
				RelevanceScore: 0.85,
				LLMScore:       &llmScore,
				MatchedSkills:  []string{"Go", "Kubernetes"},
			},
			{
				StoryID:        "story-2",
				RelevanceScore: 0.75,
				MatchedSkills:  []string{"Python"},
			},
		},
	}

	p.PrintRankedStories(stories)
	output := buf.String()

	assert.Contains(t, output, "TOP RANKED STORIES")
	assert.Contains(t, output, "story-1")
	assert.Contains(t, output, "0.85")
	assert.Contains(t, output, "LLM: 0.95")
	assert.Contains(t, output, "Go, Kubernetes")
}

func TestPrintSelectedBullets(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	bullets := &types.SelectedBullets{
		Bullets: []types.SelectedBullet{
			{
				ID:      "b1",
				StoryID: "story-1",
				Text:    "Implemented distributed cache",
				Skills:  []string{"Redis", "Go"},
			},
		},
	}

	p.PrintSelectedBullets(bullets)
	output := buf.String()

	assert.Contains(t, output, "SELECTED BULLETS")
	assert.Contains(t, output, "Implemented distributed cache")
	assert.Contains(t, output, "Redis, Go")
}

func TestPrintRewrittenBullets(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{
				OriginalBulletID: "b1",
				FinalText:        "Architected distributed cache serving 1M requests/day",
				StyleChecks: types.StyleChecks{
					StrongVerb:   true,
					Quantified:   true,
					NoTaboo:      true,
					TargetLength: true,
				},
			},
		},
	}

	p.PrintRewrittenBullets(bullets)
	output := buf.String()

	assert.Contains(t, output, "REWRITTEN BULLETS")
	assert.Contains(t, output, "Architected distributed cache")
	assert.Contains(t, output, "✓verb")
	assert.Contains(t, output, "✓metrics")
}

func TestPrintCompanyProfile(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	profile := &types.CompanyProfile{
		Company:    "TechCorp",
		Tone:       "professional yet approachable",
		StyleRules: []string{"Use active voice", "Be concise"},
		Values:     []string{"Innovation", "Collaboration"},
	}

	p.PrintCompanyProfile(profile)
	output := buf.String()

	assert.Contains(t, output, "COMPANY VOICE PROFILE")
	assert.Contains(t, output, "TechCorp")
	assert.Contains(t, output, "professional yet approachable")
	assert.Contains(t, output, "Use active voice")
	assert.Contains(t, output, "Innovation")
}

func TestPrintViolations_WithViolations(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	violations := &types.Violations{
		Violations: []types.Violation{
			{
				Type:    "length",
				Details: "Bullet exceeds maximum length",
			},
		},
	}

	p.PrintViolations(violations)
	output := buf.String()

	assert.Contains(t, output, "CONSTRAINT VIOLATIONS")
	assert.Contains(t, output, "length")
	assert.Contains(t, output, "Bullet exceeds maximum length")
}

func TestPrintViolations_NoViolations(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	violations := &types.Violations{
		Violations: []types.Violation{},
	}

	p.PrintViolations(violations)
	output := buf.String()

	assert.Contains(t, output, "NO VIOLATIONS FOUND")
}

func TestPrintBox_LongLines(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	// Test with a profile containing long text
	profile := &types.JobProfile{
		Company:   "A Very Long Company Name That Should Be Truncated To Fit",
		RoleTitle: "Senior Staff Principal Distinguished Engineer Level 99",
	}

	p.PrintJobProfile(profile)
	output := buf.String()

	// Should contain box characters
	assert.True(t, strings.Contains(output, "┌"))
	assert.True(t, strings.Contains(output, "└"))
	assert.True(t, strings.Contains(output, "│"))
}
