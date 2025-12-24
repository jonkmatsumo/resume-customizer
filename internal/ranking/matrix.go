package ranking

import (
	"strings"

	"github.com/jonathan/resume-customizer/internal/types"
)

// ScoreBulletAgainstSkill returns a score (0.0 to 1.0) representing how well
// a specific bullet point matches a specific target skill.
//
// Currently, this implementation checks for exact skill name matches within
// the bullet's pre-identified skill list. IF matched, it returns 1.0.
// Future improvements could include keyword density or vector similarity.
func ScoreBulletAgainstSkill(bullet *types.Bullet, skill types.Skill) float64 {
	// Normalize skill name for comparison
	skillName := strings.ToLower(strings.TrimSpace(skill.Name))
	if skillName == "" {
		return 0.0
	}

	// Check if the skill is explicitly tagged in the bullet
	for _, bulletSkill := range bullet.Skills {
		if strings.ToLower(strings.TrimSpace(bulletSkill)) == skillName {
			// Found a direct match.
			// We could use the skill's weight here, but the greedy selector uses
			// the weight to PRIORITIZE which skill to look for, so the score here
			// should probably reflect "Match Quality" or "Confidence".
			// For now, binary 1.0 is sufficient.
			return 1.0
		}
	}

	// Check if the skill name appears in the bullet text itself as a keyword fallback
	// This is useful if extraction missed it but it's present in the text.
	if strings.Contains(strings.ToLower(bullet.Text), skillName) {
		return 0.8 // Slightly lower confidence if just text match
	}

	return 0.0
}
