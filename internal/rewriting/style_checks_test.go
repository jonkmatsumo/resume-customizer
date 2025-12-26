// Package rewriting provides functionality to rewrite bullet points to match job requirements and company brand voice.
package rewriting

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestCheckStrongVerb(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"Strong verb - built", "built a system", true},
		{"Strong verb - achieved", "achieved 50% improvement", true},
		{"Strong verb - designed", "designed architecture", true},
		{"Strong verb - past tense ed", "optimized performance", true},
		{"Weak start - I", "I worked on", false},
		{"Weak start - The", "The system was", false},
		{"Empty text", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkStrongVerb(tt.text)
			assert.Equal(t, tt.expected, result, "checkStrongVerb(%q) = %v, want %v", tt.text, result, tt.expected)
		})
	}
}

func TestCheckQuantifiedImpact(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"Has digits", "Increased revenue by 50%", true},
		{"Has percentage", "Improved efficiency by 30%", true},
		{"Has number", "Handled 1M requests", true},
		{"No numbers", "Built a great system", false},
		{"Empty text", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkQuantifiedImpact(tt.text)
			assert.Equal(t, tt.expected, result, "checkQuantifiedImpact(%q) = %v, want %v", tt.text, result, tt.expected)
		})
	}
}

func TestCheckNoTaboo(t *testing.T) {
	companyProfile := &types.CompanyProfile{
		TabooPhrases: []string{"synergy", "ninja", "rockstar"},
	}

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"No taboo", "built a system", true},
		{"Contains taboo - synergy", "created synergy between teams", false},
		{"Contains taboo - ninja", "python ninja", false},
		{"Contains taboo - rockstar", "rockstar developer", false},
		{"Taboo at word boundary", "synergy-focused approach", false},
		{"No company profile", "synergy", true}, // No profile means no taboo check
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := companyProfile
			if tt.name == "No company profile" {
				profile = nil
			}
			result := checkNoTaboo(tt.text, profile)
			assert.Equal(t, tt.expected, result, "checkNoTaboo(%q) = %v, want %v", tt.text, result, tt.expected)
		})
	}
}

func TestCheckTargetLength(t *testing.T) {
	tests := []struct {
		name            string
		rewrittenLength int
		originalLength  int
		expected        bool
	}{
		{"Exact match", 50, 50, true},
		{"Within tolerance - slightly longer", 55, 50, true},  // 10% longer
		{"Within tolerance - slightly shorter", 45, 50, true}, // 10% shorter
		{"Too long", 100, 50, false},
		{"Too short", 10, 50, false},
		{"Original zero - any length OK", 50, 0, true},
		{"Original zero - zero length", 0, 0, false}, // Must have some length
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkTargetLength(tt.rewrittenLength, tt.originalLength)
			assert.Equal(t, tt.expected, result, "checkTargetLength(%d, %d) = %v, want %v",
				tt.rewrittenLength, tt.originalLength, result, tt.expected)
		})
	}
}

func TestValidateStyle(t *testing.T) {
	companyProfile := &types.CompanyProfile{
		TabooPhrases: []string{"synergy"},
	}

	tests := []struct {
		name            string
		rewrittenText   string
		originalLength  int
		expectedStrong  bool
		expectedQuant   bool
		expectedNoTaboo bool
		expectedLength  bool
	}{
		{
			name:            "All checks pass",
			rewrittenText:   "Built system handling 1M requests/day",
			originalLength:  35,
			expectedStrong:  true,
			expectedQuant:   true,
			expectedNoTaboo: true,
			expectedLength:  true,
		},
		{
			name:            "Missing quantified impact",
			rewrittenText:   "Built a great system",
			originalLength:  20,
			expectedStrong:  true,
			expectedQuant:   false,
			expectedNoTaboo: true,
			expectedLength:  true,
		},
		{
			name:            "Contains taboo",
			rewrittenText:   "Created synergy between teams",
			originalLength:  30,
			expectedStrong:  true,
			expectedQuant:   false,
			expectedNoTaboo: false,
			expectedLength:  true,
		},
		{
			name:            "No strong verb",
			rewrittenText:   "The system was improved",
			originalLength:  25,
			expectedStrong:  false,
			expectedQuant:   false,
			expectedNoTaboo: true,
			expectedLength:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateStyle(tt.rewrittenText, companyProfile, tt.originalLength)

			assert.Equal(t, tt.expectedStrong, result.StrongVerb, "StrongVerb")
			assert.Equal(t, tt.expectedQuant, result.Quantified, "Quantified")
			assert.Equal(t, tt.expectedNoTaboo, result.NoTaboo, "NoTaboo")
			assert.Equal(t, tt.expectedLength, result.TargetLength, "TargetLength")
		})
	}
}

func TestEstimateLines(t *testing.T) {
	tests := []struct {
		name          string
		lengthChars   int
		expectedLines int
	}{
		{"Short text", 45, 1},
		{"One line", 100, 1},
		{"Two lines", 101, 2},
		{"Two lines exact", 200, 2},
		{"Three lines", 201, 3},
		{"Zero length", 0, 1}, // Minimum 1 line
		{"Very long", 500, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateLines(tt.lengthChars)
			assert.Equal(t, tt.expectedLines, result, "EstimateLines(%d) = %d, want %d",
				tt.lengthChars, result, tt.expectedLines)
		})
	}
}

func TestComputeLengthChars(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"Normal text", "Built a system", 14},
		{"Empty string", "", 0},
		{"With spaces", "Built a system with metrics", 27},
		{"With newlines", "Built\na\nsystem", 14}, // Newlines count as characters
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeLengthChars(tt.text)
			assert.Equal(t, tt.expected, result, "ComputeLengthChars(%q) = %d, want %d",
				tt.text, result, tt.expected)
		})
	}
}
