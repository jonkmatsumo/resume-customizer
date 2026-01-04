// Package repair provides functionality to automatically fix violations in LaTeX resumes.
package repair

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestExtractBulletIDsFromActions(t *testing.T) {
	tests := []struct {
		name     string
		actions  *types.RepairActions
		expected []string
	}{
		{
			name: "Single shorten_bullet action",
			actions: &types.RepairActions{
				Actions: []types.RepairAction{
					{
						Type:     "shorten_bullet",
						BulletID: "bullet_001",
					},
				},
			},
			expected: []string{"bullet_001"},
		},
		{
			name: "Multiple shorten_bullet actions",
			actions: &types.RepairActions{
				Actions: []types.RepairAction{
					{
						Type:     "shorten_bullet",
						BulletID: "bullet_001",
					},
					{
						Type:     "shorten_bullet",
						BulletID: "bullet_002",
					},
				},
			},
			expected: []string{"bullet_001", "bullet_002"},
		},
		{
			name: "Mixed actions - only shorten_bullet extracted",
			actions: &types.RepairActions{
				Actions: []types.RepairAction{
					{
						Type:     "drop_bullet",
						BulletID: "bullet_001",
					},
					{
						Type:     "shorten_bullet",
						BulletID: "bullet_002",
					},
				},
			},
			expected: []string{"bullet_002"},
		},
		{
			name: "No shorten_bullet actions",
			actions: &types.RepairActions{
				Actions: []types.RepairAction{
					{
						Type:     "drop_bullet",
						BulletID: "bullet_001",
					},
				},
			},
			expected: []string{},
		},
		{
			name:     "Empty actions",
			actions:  &types.RepairActions{Actions: []types.RepairAction{}},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBulletIDsFromActions(tt.actions)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindNewBulletIDs(t *testing.T) {
	tests := []struct {
		name           string
		plan           *types.ResumePlan
		currentBullets *types.RewrittenBullets
		expected       []string
	}{
		{
			name: "All bullets are new",
			plan: &types.ResumePlan{
				SelectedStories: []types.SelectedStory{
					{
						StoryID:   "story_001",
						BulletIDs: []string{"bullet_001", "bullet_002"},
					},
				},
			},
			currentBullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{}, // Empty
			},
			expected: []string{"bullet_001", "bullet_002"},
		},
		{
			name: "Some bullets are new",
			plan: &types.ResumePlan{
				SelectedStories: []types.SelectedStory{
					{
						StoryID:   "story_001",
						BulletIDs: []string{"bullet_001", "bullet_002", "bullet_003"},
					},
				},
			},
			currentBullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{
					{
						OriginalBulletID: "bullet_001",
						FinalText:        "Existing",
					},
				},
			},
			expected: []string{"bullet_002", "bullet_003"},
		},
		{
			name: "No new bullets",
			plan: &types.ResumePlan{
				SelectedStories: []types.SelectedStory{
					{
						StoryID:   "story_001",
						BulletIDs: []string{"bullet_001"},
					},
				},
			},
			currentBullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{
					{
						OriginalBulletID: "bullet_001",
						FinalText:        "Existing",
					},
				},
			},
			expected: []string{},
		},
		{
			name: "Multiple stories with new bullets",
			plan: &types.ResumePlan{
				SelectedStories: []types.SelectedStory{
					{
						StoryID:   "story_001",
						BulletIDs: []string{"bullet_001", "bullet_002"},
					},
					{
						StoryID:   "story_002",
						BulletIDs: []string{"bullet_003"},
					},
				},
			},
			currentBullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{
					{
						OriginalBulletID: "bullet_001",
						FinalText:        "Existing",
					},
				},
			},
			expected: []string{"bullet_002", "bullet_003"},
		},
		{
			name: "Empty plan",
			plan: &types.ResumePlan{
				SelectedStories: []types.SelectedStory{},
			},
			currentBullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{
					{
						OriginalBulletID: "bullet_001",
						FinalText:        "Existing",
					},
				},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findNewBulletIDs(tt.plan, tt.currentBullets)
			assert.Equal(t, tt.expected, result)
		})
	}
}
