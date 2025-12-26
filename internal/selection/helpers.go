// Package selection provides functionality to select optimal stories and bullets for a resume plan.
package selection

import (
	"math"

	"github.com/jonathan/resume-customizer/internal/types"
)

const (
	// charsPerLine is the estimated number of characters per line in the resume
	charsPerLine = 100
	// defaultRelevanceWeight is the weight for relevance score in value computation
	defaultRelevanceWeight = 0.6
	// defaultSkillWeight is the weight for skill coverage in value computation
	defaultSkillWeight = 0.4
)

// estimateLines calculates the estimated number of lines for a bullet point
func estimateLines(lengthChars int) int {
	if lengthChars <= 0 {
		return 1 // Minimum 1 line
	}
	return int(math.Ceil(float64(lengthChars) / charsPerLine))
}

// computeSkillCoverageScore calculates the skill coverage score for a set of bullets
// by summing the weights of all skills covered by those bullets
func computeSkillCoverageScore(bullets []types.Bullet, skillTargets *types.SkillTargets) float64 {
	if skillTargets == nil || len(skillTargets.Skills) == 0 {
		return 0.0
	}

	// Build skill weight map for efficient lookup
	skillWeightMap := make(map[string]float64)
	for _, skill := range skillTargets.Skills {
		skillWeightMap[skill.Name] = skill.Weight
	}

	// Collect unique skills from bullets
	coveredSkills := make(map[string]bool)
	for _, bullet := range bullets {
		for _, skill := range bullet.Skills {
			coveredSkills[skill] = true
		}
	}

	// Sum weights of covered skills
	totalWeight := 0.0
	for skill := range coveredSkills {
		if weight, found := skillWeightMap[skill]; found {
			totalWeight += weight
		}
	}

	// Normalize by total possible weight
	totalPossibleWeight := 0.0
	for _, skill := range skillTargets.Skills {
		totalPossibleWeight += skill.Weight
	}

	if totalPossibleWeight == 0 {
		return 0.0
	}

	return totalWeight / totalPossibleWeight
}
