// Package skills provides functionality to build weighted skill targets from job profiles.
package skills

import (
	"fmt"
	"sort"

	"github.com/jonathan/resume-customizer/internal/parsing"
	"github.com/jonathan/resume-customizer/internal/types"
)

const (
	// Weight constants for skill sources
	weightHardRequirement = 1.0
	weightNiceToHave     = 0.5
	weightKeyword        = 0.3

	// Source constants
	sourceHardRequirement = "hard_requirement"
	sourceNiceToHave     = "nice_to_have"
	sourceKeyword        = "keyword"
)

// BuildSkillTargets builds a weighted list of target skills from a JobProfile.
// Skills are normalized, deduplicated (taking max weight when duplicates exist),
// and sorted by weight (descending).
func BuildSkillTargets(jobProfile *types.JobProfile) (*types.SkillTargets, error) {
	// Map: normalized skill name -> skill info (weight, source)
	skillMap := make(map[string]*skillInfo)

	// Process hard requirements
	for _, req := range jobProfile.HardRequirements {
		normalizedSkill := parsing.NormalizeSkillName(req.Skill)
		if normalizedSkill == "" {
			continue
		}
		addOrUpdateSkill(skillMap, normalizedSkill, weightHardRequirement, sourceHardRequirement)
	}

	// Process nice-to-haves
	for _, req := range jobProfile.NiceToHaves {
		normalizedSkill := parsing.NormalizeSkillName(req.Skill)
		if normalizedSkill == "" {
			continue
		}
		addOrUpdateSkill(skillMap, normalizedSkill, weightNiceToHave, sourceNiceToHave)
	}

	// Process keywords
	for _, keyword := range jobProfile.Keywords {
		normalizedSkill := parsing.NormalizeSkillName(keyword)
		if normalizedSkill == "" {
			continue
		}
		addOrUpdateSkill(skillMap, normalizedSkill, weightKeyword, sourceKeyword)
	}

	// Convert map to slice
	skills := make([]types.Skill, 0, len(skillMap))
	for name, info := range skillMap {
		skills = append(skills, types.Skill{
			Name:   name,
			Weight: info.weight,
			Source: info.source,
		})
	}

	if len(skills) == 0 {
		return nil, fmt.Errorf("no skills found in job profile")
	}

	// Sort by weight (descending)
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Weight > skills[j].Weight
	})

	return &types.SkillTargets{Skills: skills}, nil
}

// skillInfo holds temporary information about a skill during building
type skillInfo struct {
	weight float64
	source string
}

// addOrUpdateSkill adds a skill to the map or updates it if it exists,
// taking the maximum weight when duplicates are found.
func addOrUpdateSkill(skillMap map[string]*skillInfo, skillName string, weight float64, source string) {
	if existing, exists := skillMap[skillName]; exists {
		// Take maximum weight
		if weight > existing.weight {
			existing.weight = weight
			existing.source = source
		}
		// If weights are equal, prioritize source by: hard_requirement > nice_to_have > keyword
		if weight == existing.weight && getSourcePriority(source) > getSourcePriority(existing.source) {
			existing.source = source
		}
	} else {
		skillMap[skillName] = &skillInfo{
			weight: weight,
			source: source,
		}
	}
}

// getSourcePriority returns a numeric priority for source types.
// Higher numbers indicate higher priority.
func getSourcePriority(source string) int {
	switch source {
	case sourceHardRequirement:
		return 3
	case sourceNiceToHave:
		return 2
	case sourceKeyword:
		return 1
	default:
		return 0
	}
}

