package selection

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
)

func TestSelectGreedy(t *testing.T) {
	stories := []*types.Story{
		{
			ID: "story1",
			Bullets: []types.Bullet{
				{ID: "b1", Text: "Python development", Skills: []string{"Python"}, LengthChars: 50}, // ~1 line
				{ID: "b2", Text: "Kubernetes orchestration", Skills: []string{"Kubernetes"}, LengthChars: 50},
			},
		},
		{
			ID: "story2",
			Bullets: []types.Bullet{
				{ID: "b3", Text: "Cloud infrastructure with AWS", Skills: []string{"AWS"}, LengthChars: 50},
			},
		},
	}

	skillTargets := &types.SkillTargets{
		Skills: []types.Skill{
			{Name: "Python", Weight: 10.0},
			{Name: "Kubernetes", Weight: 8.0},
			{Name: "AWS", Weight: 5.0},
		},
	}

	// Case 1: Enough space for all
	// Should select b1 (Python), b2 (K8s), b3 (AWS)
	selections, _, err := SelectGreedy(stories, skillTargets, 10)
	if err != nil {
		t.Fatalf("SelectGreedy failed: %v", err)
	}

	expectedIDs := map[string]bool{"b1": true, "b2": true, "b3": true}
	checkSelections(t, selections, expectedIDs)

	// Case 2: Limited space (only 1 line allowed)
	// Should select b1 only (highest weight skill: Python)
	selections, _, err = SelectGreedy(stories, skillTargets, 1) // assuming 50 chars = 1 line
	if err != nil {
		t.Fatalf("SelectGreedy failed: %v", err)
	}
	expectedIDs = map[string]bool{"b1": true}
	checkSelections(t, selections, expectedIDs)

	// Case 3: Limited space (2 lines)
	// Should select b1 (Python) and b2 (K8s) -> skipping AWS (lowest weight)
	selections, _, err = SelectGreedy(stories, skillTargets, 2)
	if err != nil {
		t.Fatalf("SelectGreedy failed: %v", err)
	}
	expectedIDs = map[string]bool{"b1": true, "b2": true}
	checkSelections(t, selections, expectedIDs)
}

func checkSelections(t *testing.T, selections []StorySelection, expectedIDs map[string]bool) {
	count := 0
	for _, sel := range selections {
		for _, bid := range sel.bulletIDs {
			if !expectedIDs[bid] {
				t.Errorf("Unexpectedly selected bullet %s", bid)
			}
			count++
		}
	}
	if count != len(expectedIDs) {
		t.Errorf("Expected %d selections, got %d", len(expectedIDs), count)
	}
}
