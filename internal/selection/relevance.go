// Package selection provides functionality to select and score resume content.
package selection

import (
	"sort"

	"github.com/jonathan/resume-customizer/internal/types"
)

const (
	// Scoring weights for bullet relevance
	weightStoryRelevance   = 0.40
	weightSkillCoverage    = 0.30
	weightLengthEfficiency = 0.20
	weightStyleQuality     = 0.10

	// Constants for normalization
	maxSkillsNorm  = 5.0 // Normalize skill count to max of 5
	maxLengthChars = 200 // Target max length for efficiency calculation
)

// ScoredBullet represents a bullet with its relevance score
type ScoredBullet struct {
	BulletID       string
	StoryID        string
	RelevanceScore float64
	Components     ScoreComponents
}

// ScoreComponents holds the individual scoring factors
type ScoreComponents struct {
	StoryRelevance   float64
	SkillCoverage    float64
	LengthEfficiency float64
	StyleQuality     float64
}

// ScoreBulletRelevance calculates the relevance score for a single bullet.
// Higher scores indicate more relevant bullets that should be kept.
// Lower scores indicate less relevant bullets that can be dropped.
func ScoreBulletRelevance(
	bullet types.RewrittenBullet,
	storyID string,
	_ *types.ResumePlan, // Reserved for future use (e.g., position-based scoring)
	_ *types.JobProfile, // Reserved for future use (e.g., skill matching)
	rankedStories *types.RankedStories,
	experienceBank *types.ExperienceBank,
) float64 {
	components := calculateScoreComponents(bullet, storyID, rankedStories, experienceBank)
	return calculateFinalScore(components)
}

// ScoreAllBullets scores all bullets and returns them sorted by relevance (lowest first).
// This allows easy identification of least relevant bullets for dropping.
func ScoreAllBullets(
	bullets *types.RewrittenBullets,
	plan *types.ResumePlan,
	_ *types.JobProfile, // Reserved for future use
	rankedStories *types.RankedStories,
	experienceBank *types.ExperienceBank,
) []ScoredBullet {
	if bullets == nil || len(bullets.Bullets) == 0 {
		return []ScoredBullet{}
	}

	// Build bullet ID to story ID mapping from plan
	bulletToStory := buildBulletToStoryMap(plan)

	// Score all bullets
	scoredBullets := make([]ScoredBullet, 0, len(bullets.Bullets))
	for _, bullet := range bullets.Bullets {
		storyID := bulletToStory[bullet.OriginalBulletID]
		components := calculateScoreComponents(bullet, storyID, rankedStories, experienceBank)
		score := calculateFinalScore(components)

		scoredBullets = append(scoredBullets, ScoredBullet{
			BulletID:       bullet.OriginalBulletID,
			StoryID:        storyID,
			RelevanceScore: score,
			Components:     components,
		})
	}

	// Sort by relevance score (lowest first - these are candidates for dropping)
	sort.Slice(scoredBullets, func(i, j int) bool {
		return scoredBullets[i].RelevanceScore < scoredBullets[j].RelevanceScore
	})

	return scoredBullets
}

// calculateScoreComponents computes the individual scoring factors for a bullet
func calculateScoreComponents(
	bullet types.RewrittenBullet,
	storyID string,
	rankedStories *types.RankedStories,
	experienceBank *types.ExperienceBank,
) ScoreComponents {
	return ScoreComponents{
		StoryRelevance:   calculateStoryRelevance(storyID, rankedStories),
		SkillCoverage:    calculateSkillCoverage(bullet.OriginalBulletID, experienceBank),
		LengthEfficiency: calculateLengthEfficiency(bullet.LengthChars),
		StyleQuality:     calculateStyleQuality(bullet.StyleChecks),
	}
}

// calculateFinalScore combines the score components using weighted average
func calculateFinalScore(components ScoreComponents) float64 {
	return components.StoryRelevance*weightStoryRelevance +
		components.SkillCoverage*weightSkillCoverage +
		components.LengthEfficiency*weightLengthEfficiency +
		components.StyleQuality*weightStyleQuality
}

// calculateStoryRelevance looks up the story's relevance score from ranked stories
func calculateStoryRelevance(storyID string, rankedStories *types.RankedStories) float64 {
	if rankedStories == nil || storyID == "" {
		return 0.5 // Default mid-level relevance if unknown
	}

	for _, story := range rankedStories.Ranked {
		if story.StoryID == storyID {
			return story.RelevanceScore
		}
	}

	return 0.5 // Default if story not found in ranked list
}

// calculateSkillCoverage counts unique skills covered by the bullet
func calculateSkillCoverage(bulletID string, experienceBank *types.ExperienceBank) float64 {
	if experienceBank == nil || bulletID == "" {
		return 0.5 // Default mid-level coverage if unknown
	}

	// Find bullet in experience bank
	for _, story := range experienceBank.Stories {
		for _, bullet := range story.Bullets {
			if bullet.ID == bulletID {
				// Normalize skill count (more skills = higher score, capped at maxSkillsNorm)
				skillCount := float64(len(bullet.Skills))
				if skillCount > maxSkillsNorm {
					skillCount = maxSkillsNorm
				}
				return skillCount / maxSkillsNorm
			}
		}
	}

	return 0.5 // Default if bullet not found
}

// calculateLengthEfficiency scores based on character length
// Shorter bullets are more efficient (better use of space)
func calculateLengthEfficiency(lengthChars int) float64 {
	if lengthChars <= 0 {
		return 0.5
	}

	// Efficiency decreases as length increases
	// A bullet at maxLengthChars (200) gets 0.5 efficiency
	// Shorter bullets get higher efficiency (up to 1.0)
	// Longer bullets get lower efficiency (down to 0.0)
	efficiency := 1.0 - (float64(lengthChars) / float64(maxLengthChars*2))
	if efficiency < 0 {
		efficiency = 0
	}
	if efficiency > 1 {
		efficiency = 1
	}

	return efficiency
}

// calculateStyleQuality scores based on style check results
func calculateStyleQuality(styleChecks types.StyleChecks) float64 {
	score := 0.0
	total := 4.0

	if styleChecks.StrongVerb {
		score += 1.0
	}
	if styleChecks.Quantified {
		score += 1.0
	}
	if styleChecks.NoTaboo {
		score += 1.0
	}
	if styleChecks.TargetLength {
		score += 1.0
	}

	return score / total
}

// buildBulletToStoryMap creates a mapping from bullet ID to story ID
func buildBulletToStoryMap(plan *types.ResumePlan) map[string]string {
	bulletToStory := make(map[string]string)
	if plan == nil {
		return bulletToStory
	}

	for _, story := range plan.SelectedStories {
		for _, bulletID := range story.BulletIDs {
			bulletToStory[bulletID] = story.StoryID
		}
	}

	return bulletToStory
}
