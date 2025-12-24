package ranking

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
)

func TestScoreBulletAgainstSkill(t *testing.T) {
	tests := []struct {
		name     string
		bullet   types.Bullet
		skill    types.Skill
		expected float64
	}{
		{
			name: "Direct Match in Skills List",
			bullet: types.Bullet{
				Text:   "Used Go to build servers",
				Skills: []string{"Go", "Docker"},
			},
			skill: types.Skill{
				Name: "Go",
			},
			expected: 1.0,
		},
		{
			name: "Case Insensitive Match",
			bullet: types.Bullet{
				Text:   "Used Go to build servers",
				Skills: []string{"go", "docker"},
			},
			skill: types.Skill{
				Name: "Go",
			},
			expected: 1.0,
		},
		{
			name: "Text Match Fallback",
			bullet: types.Bullet{
				Text:   "Orchestrated with Kubernetes clusters",
				Skills: []string{}, // Extraction missed it
			},
			skill: types.Skill{
				Name: "Kubernetes",
			},
			expected: 0.8,
		},
		{
			name: "No Match",
			bullet: types.Bullet{
				Text:   "Writing docs",
				Skills: []string{"Markdown"},
			},
			skill: types.Skill{
				Name: "Java",
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ScoreBulletAgainstSkill(&tt.bullet, tt.skill)
			if score != tt.expected {
				t.Errorf("expected score %v, got %v", tt.expected, score)
			}
		})
	}
}
