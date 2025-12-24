// Package skills provides functionality to build weighted skill targets from job profiles.
package skills

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/parsing"
	"github.com/jonathan/resume-customizer/internal/types"
)

const (
	// Weight constants for skill sources (requirement level)
	weightHardRequirement = 1.0
	weightNiceToHave      = 0.5
	weightKeyword         = 0.3

	// Source constants
	sourceHardRequirement = "hard_requirement"
	sourceNiceToHave      = "nice_to_have"
	sourceKeyword         = "keyword"
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

// BuildSkillTargetsWithSpecificity builds skill targets and applies LLM-judged specificity.
// specificityWeight controls the blend: FinalWeight = ReqWeight * (1-ratio) + Specificity * ratio.
func BuildSkillTargetsWithSpecificity(
	ctx context.Context,
	jobProfile *types.JobProfile,
	client llm.Client,
	specificityWeight float64,
) (*types.SkillTargets, error) {
	// Build base targets first
	targets, err := BuildSkillTargets(jobProfile)
	if err != nil {
		return nil, err
	}

	if client == nil || specificityWeight <= 0 {
		// No LLM or specificity disabled, return base targets
		return targets, nil
	}

	// Collect skill names for LLM evaluation
	skillNames := make([]string, len(targets.Skills))
	for i, skill := range targets.Skills {
		skillNames[i] = skill.Name
	}

	// Get specificity scores from LLM
	specificityScores, err := JudgeSkillSpecificity(ctx, skillNames, client)
	if err != nil {
		// Log warning but continue with base weights (graceful degradation)
		// In production, you might want to log this error
		return targets, nil
	}

	// Apply blended weights
	for i := range targets.Skills {
		skill := &targets.Skills[i]
		normalized := strings.ToLower(strings.TrimSpace(skill.Name))
		specificity := specificityScores[normalized]

		// Store raw specificity
		skill.Specificity = specificity

		// Blend: FinalWeight = ReqWeight * (1 - ratio) + Specificity * ratio
		// Note: We normalize both to 0-1 range first
		reqWeight := skill.Weight // Already 0-1 range (1.0, 0.5, 0.3)
		blendedWeight := reqWeight*(1-specificityWeight) + specificity*specificityWeight

		skill.Weight = blendedWeight
	}

	// Re-sort by new blended weight
	sort.Slice(targets.Skills, func(i, j int) bool {
		return targets.Skills[i].Weight > targets.Skills[j].Weight
	})

	return targets, nil
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
