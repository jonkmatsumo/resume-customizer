package parsing

import (
	"strings"

	"github.com/jonathan/resume-customizer/internal/types"
)

// skillNormalizations maps common skill name variants to canonical names
var skillNormalizations = map[string]string{
	"golang":     "Go",
	"golanglang": "Go",
	"go lang":    "Go",
	"javascript": "JavaScript",
	"js":         "JavaScript",
	"typescript": "TypeScript",
	"ts":         "TypeScript",
	"k8s":        "Kubernetes",
	"kubernetes": "Kubernetes",
	"react.js":   "React",
	"reactjs":    "React",
	"vue.js":     "Vue",
	"vuejs":      "Vue",
	"node.js":    "Node.js",
	"nodejs":     "Node.js",
}

// NormalizeSkillName normalizes a skill name to its canonical form
func NormalizeSkillName(skillName string) string {
	if skillName == "" {
		return ""
	}

	// Trim whitespace
	normalized := strings.TrimSpace(skillName)

	// Check for exact match in normalization map (case-insensitive)
	lower := strings.ToLower(normalized)
	if canonical, ok := skillNormalizations[lower]; ok {
		return canonical
	}

	// Handle case normalization for common patterns
	// If it's all uppercase, try to find a canonical form
	if normalized == strings.ToUpper(normalized) && len(normalized) > 1 {
		lowerCanonical, ok := skillNormalizations[lower]
		if ok {
			return lowerCanonical
		}
		// For all-caps single words that aren't acronyms, capitalize first letter only
		if !strings.Contains(lower, " ") {
			return strings.ToUpper(normalized[:1]) + strings.ToLower(normalized[1:])
		}
	}

	// For skills starting with lowercase, capitalize first letter if it's a single word
	if normalized != strings.ToUpper(normalized) && normalized != strings.ToLower(normalized) {
		// Already has mixed case, return as-is
		return normalized
	}

	// If all lowercase and single word, capitalize first letter
	if normalized == strings.ToLower(normalized) && !strings.Contains(normalized, " ") && len(normalized) > 0 {
		return strings.ToUpper(normalized[:1]) + normalized[1:]
	}

	return normalized
}

// NormalizeRequirements normalizes skill names and deduplicates requirements
func NormalizeRequirements(reqs []types.Requirement) []types.Requirement {
	if len(reqs) == 0 {
		return reqs
	}

	normalized := make([]types.Requirement, 0, len(reqs))
	seen := make(map[string]int) // normalized skill name -> index in normalized slice

	for _, req := range reqs {
		normalizedSkill := NormalizeSkillName(req.Skill)
		if normalizedSkill == "" {
			continue // Skip empty skill names
		}

		// Check if we've seen this normalized skill before
		if idx, exists := seen[normalizedSkill]; exists {
			// Merge levels if the existing requirement doesn't have one
			if normalized[idx].Level == "" && req.Level != "" {
				normalized[idx].Level = req.Level
			}
			// Keep the first evidence (or merge if needed)
			// For now, we keep the first occurrence
			continue
		}

		// Add new requirement with normalized skill name
		newReq := types.Requirement{
			Skill:    normalizedSkill,
			Level:    req.Level,
			Evidence: req.Evidence,
		}
		normalized = append(normalized, newReq)
		seen[normalizedSkill] = len(normalized) - 1
	}

	return normalized
}
