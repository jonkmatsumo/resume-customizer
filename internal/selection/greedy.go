package selection

import (
	"sort"

	"github.com/jonathan/resume-customizer/internal/ranking"
	"github.com/jonathan/resume-customizer/internal/types"
)

// SelectGreedy selects the best bullets to cover the target skills using a greedy approach.
// It prioritizes skills with higher weights and selects the best available bullet for each skill.
//
// key constraints:
// - maxLines: total estimated lines of selected bullets must not exceed this.
// - Each bullet can be selected only once (global uniqueness).
// - We try to cover as many unique, high-weight skills as possible.
func SelectGreedy(
	stories []*types.Story,
	skillTargets *types.SkillTargets,
	maxLines int,
) ([]StorySelection, float64, error) {

	// 1. Flatten all bullets from all stories into a candidate list
	// Map bullet ID to its story ID and the bullet object for easy lookup
	type candidateBullet struct {
		Bullet  *types.Bullet
		StoryID string
	}
	candidates := make([]candidateBullet, 0)
	for _, story := range stories {
		for i := range story.Bullets {
			// Take address of bullet in the slice
			candidates = append(candidates, candidateBullet{
				Bullet:  &story.Bullets[i],
				StoryID: story.ID,
			})
		}
	}

	// 2. Sort target skills by weight (highest first)
	// We want to "spend" our limited line budget on the most important skills first.
	sortedSkills := make([]types.Skill, len(skillTargets.Skills))
	copy(sortedSkills, skillTargets.Skills)
	sort.Slice(sortedSkills, func(i, j int) bool {
		return sortedSkills[i].Weight > sortedSkills[j].Weight
	})

	// 3. Greedy Selection Loop
	selectedBulletIDs := make(map[string]bool)
	totalLinesUsed := 0
	totalScore := 0.0

	// Track which story has which selected bullets
	selectionMap := make(map[string][]string) // storyID -> []bulletID

	// Iterate through skills in descending order of importance
	for _, skill := range sortedSkills {
		// Stop if we are full
		if totalLinesUsed >= maxLines {
			break
		}

		// Check if this skill is already "covered" by a previously selected bullet?
		// The user request says: "Then for each skill we try to select the top bullet point that has not been selected already."
		// It doesn't explicitly say "skip if covered". It says "select the top bullet point".
		// IF a previously selected bullet ALREADY covers this skill well, we might want to skip to save space?
		// However, the prompt says "one Kubernetes experience is a strong match... but might not score highly overall".
		// "This will also ensure we optimize for all of the points".
		// Interpretation: We want at least one dedicated bullet for this skill, if possible.
		// Let's check if we already have a STRONG match for this skill in our selection.
		alreadyCovered := false
		for bID := range selectedBulletIDs {
			// Find the candidate from ID (inefficient, but N is small)
			var existing candidateBullet
			for _, c := range candidates {
				if c.Bullet.ID == bID {
					existing = c
					break
				}
			}
			if ranking.ScoreBulletAgainstSkill(existing.Bullet, skill) >= 1.0 { // threshold for "Good Match"
				alreadyCovered = true
				break
			}
		}

		if alreadyCovered {
			continue // Skill already well-represented
		}

		// Find the best available (unselected) bullet for this skill
		var bestCandidate *candidateBullet
		bestScore := 0.0

		for i := range candidates {
			cand := &candidates[i]
			if selectedBulletIDs[cand.Bullet.ID] {
				continue // Already used
			}

			score := ranking.ScoreBulletAgainstSkill(cand.Bullet, skill)
			if score > bestScore {
				bestScore = score
				bestCandidate = cand
			}
		}

		// If we found a match worth taking (score > 0)
		if bestCandidate != nil && bestScore > 0 {
			// Check if we have space
			lines := estimateLines(bestCandidate.Bullet.LengthChars)
			if totalLinesUsed+lines <= maxLines {
				// Select it
				selectedBulletIDs[bestCandidate.Bullet.ID] = true
				selectionMap[bestCandidate.StoryID] = append(selectionMap[bestCandidate.StoryID], bestCandidate.Bullet.ID)
				totalLinesUsed += lines
				// Score: The "Value" of the selection is subjective here.
				// We can sum the weights of covered skills.
				totalScore += skill.Weight * bestScore
			}
		}
	}

	// 4. Convert selectionMap to []StorySelection
	selections := make([]StorySelection, 0, len(selectionMap))
	for sID, bIDs := range selectionMap {
		selections = append(selections, StorySelection{
			storyID:   sID,
			bulletIDs: bIDs,
		})
	}

	return selections, totalScore, nil
}
