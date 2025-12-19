package parsing

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeSkillName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Golang to Go", "Golang", "Go"},
		{"golang to Go", "golang", "Go"},
		{"GOLANG to Go", "GOLANG", "Go"},
		{"go lang to Go", "go lang", "Go"},
		{"JavaScript normalization", "javascript", "JavaScript"},
		{"JS to JavaScript", "js", "JavaScript"},
		{"JS to JavaScript uppercase", "JS", "JavaScript"},
		{"TypeScript normalization", "typescript", "TypeScript"},
		{"TS to TypeScript", "ts", "TypeScript"},
		{"K8s to Kubernetes", "k8s", "Kubernetes"},
		{"Kubernetes stays Kubernetes", "Kubernetes", "Kubernetes"},
		{"react.js to React", "react.js", "React"},
		{"reactjs to React", "reactjs", "React"},
		{"vue.js to Vue", "vue.js", "Vue"},
		{"node.js stays node.js", "node.js", "Node.js"},
		{"nodejs to Node.js", "nodejs", "Node.js"},
		{"Python stays Python", "Python", "Python"},
		{"python to Python", "python", "Python"},
		{"PYTHON to Python", "PYTHON", "Python"},
		{"Empty string", "", ""},
		{"Whitespace only", "   ", ""},
		{"Multi-word stays as-is", "Distributed Systems", "Distributed Systems"},
		{"Already normalized", "Go", "Go"},
		{"Mixed case single word", "JavaScript", "JavaScript"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeSkillName(tt.input)
			assert.Equal(t, tt.expected, result, "should normalize skill name correctly")
		})
	}
}

func TestNormalizeRequirements(t *testing.T) {
	tests := []struct {
		name     string
		input    []types.Requirement
		expected []types.Requirement
	}{
		{
			name: "Normalize skill names",
			input: []types.Requirement{
				{Skill: "Golang", Level: "3+ years", Evidence: "test"},
				{Skill: "javascript", Level: "5+ years", Evidence: "test"},
			},
			expected: []types.Requirement{
				{Skill: "Go", Level: "3+ years", Evidence: "test"},
				{Skill: "JavaScript", Level: "5+ years", Evidence: "test"},
			},
		},
		{
			name: "Deduplicate with normalization",
			input: []types.Requirement{
				{Skill: "Go", Level: "3+ years", Evidence: "first"},
				{Skill: "Golang", Level: "5+ years", Evidence: "second"},
			},
			expected: []types.Requirement{
				{Skill: "Go", Level: "3+ years", Evidence: "first"},
			},
		},
		{
			name: "Merge levels when deduplicating",
			input: []types.Requirement{
				{Skill: "Go", Level: "", Evidence: "first"},
				{Skill: "Golang", Level: "5+ years", Evidence: "second"},
			},
			expected: []types.Requirement{
				{Skill: "Go", Level: "5+ years", Evidence: "first"},
			},
		},
		{
			name:     "Empty requirements",
			input:    []types.Requirement{},
			expected: []types.Requirement{},
		},
		{
			name: "Filter empty skill names",
			input: []types.Requirement{
				{Skill: "", Level: "1+ years", Evidence: "test"},
				{Skill: "Go", Level: "3+ years", Evidence: "test"},
			},
			expected: []types.Requirement{
				{Skill: "Go", Level: "3+ years", Evidence: "test"},
			},
		},
		{
			name: "Multiple duplicates",
			input: []types.Requirement{
				{Skill: "Go", Level: "3+ years", Evidence: "first"},
				{Skill: "JavaScript", Level: "2+ years", Evidence: "second"},
				{Skill: "Golang", Level: "5+ years", Evidence: "third"},
				{Skill: "js", Level: "1+ years", Evidence: "fourth"},
			},
			expected: []types.Requirement{
				{Skill: "Go", Level: "3+ years", Evidence: "first"},
				{Skill: "JavaScript", Level: "2+ years", Evidence: "second"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeRequirements(tt.input)
			assert.Equal(t, tt.expected, result, "should normalize and deduplicate requirements")
		})
	}
}
