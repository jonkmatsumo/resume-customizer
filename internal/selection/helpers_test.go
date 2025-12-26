package selection

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestEstimateLines(t *testing.T) {
	tests := []struct {
		name        string
		lengthChars int
		expected    int
	}{
		{
			name:        "exact fit",
			lengthChars: 100,
			expected:    1,
		},
		{
			name:        "just over one line",
			lengthChars: 101,
			expected:    2,
		},
		{
			name:        "multiple lines",
			lengthChars: 250,
			expected:    3, // ceil(250/100) = 3
		},
		{
			name:        "zero chars",
			lengthChars: 0,
			expected:    1, // minimum 1 line
		},
		{
			name:        "small text",
			lengthChars: 45,
			expected:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateLines(tt.lengthChars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComputeSkillCoverageScore(t *testing.T) {
	skillTargets := &types.SkillTargets{
		Skills: []types.Skill{
			{Name: "Go", Weight: 1.0},
			{Name: "Kubernetes", Weight: 0.7},
			{Name: "Python", Weight: 0.5},
		},
	}

	tests := []struct {
		name     string
		bullets  []types.Bullet
		expected float64
	}{
		{
			name: "all skills covered",
			bullets: []types.Bullet{
				{Skills: []string{"Go", "Kubernetes", "Python"}},
			},
			expected: 1.0, // (1.0 + 0.7 + 0.5) / (1.0 + 0.7 + 0.5) = 1.0
		},
		{
			name: "partial coverage",
			bullets: []types.Bullet{
				{Skills: []string{"Go", "Kubernetes"}},
			},
			expected: (1.0 + 0.7) / (1.0 + 0.7 + 0.5), // 0.7727...
		},
		{
			name: "single skill",
			bullets: []types.Bullet{
				{Skills: []string{"Go"}},
			},
			expected: 1.0 / (1.0 + 0.7 + 0.5), // 0.4545...
		},
		{
			name:     "no skills",
			bullets:  []types.Bullet{},
			expected: 0.0,
		},
		{
			name: "unmatched skills",
			bullets: []types.Bullet{
				{Skills: []string{"Java", "C++"}},
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeSkillCoverageScore(tt.bullets, skillTargets)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestComputeSkillCoverageScore_EmptySkillTargets(t *testing.T) {
	bullets := []types.Bullet{
		{Skills: []string{"Go"}},
	}

	result := computeSkillCoverageScore(bullets, nil)
	assert.Equal(t, 0.0, result)

	result = computeSkillCoverageScore(bullets, &types.SkillTargets{Skills: []types.Skill{}})
	assert.Equal(t, 0.0, result)
}
