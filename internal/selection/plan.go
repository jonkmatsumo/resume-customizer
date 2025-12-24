// Package selection provides functionality to select optimal stories and bullets for a resume plan.
package selection

import (
	"fmt"

	"github.com/jonathan/resume-customizer/internal/skills"
	"github.com/jonathan/resume-customizer/internal/types"
)

// SelectPlan selects the optimal stories and bullets for a resume plan using dynamic programming
func SelectPlan(
	rankedStories *types.RankedStories,
	jobProfile *types.JobProfile,
	experienceBank *types.ExperienceBank,
	spaceBudget *types.SpaceBudget,
) (*types.ResumePlan, error) {
	if rankedStories == nil || len(rankedStories.Ranked) == 0 {
		return &types.ResumePlan{
			SelectedStories: []types.SelectedStory{},
			SpaceBudget:     *spaceBudget,
			Coverage: types.Coverage{
				TopSkillsCovered: []string{},
				CoverageScore:    0.0,
			},
		}, nil
	}

	// Build skill targets from job profile
	skillTargets, err := skills.BuildSkillTargets(jobProfile)
	if err != nil {
		return nil, &Error{
			Message: "failed to build skill targets",
			Cause:   err,
		}
	}

	// Create story lookup map: storyID -> Story
	storyMap := make(map[string]*types.Story)
	for i := range experienceBank.Stories {
		storyMap[experienceBank.Stories[i].ID] = &experienceBank.Stories[i]
	}

	// Create ranked story lookup map: storyID -> RankedStory
	rankedStoryMap := make(map[string]*types.RankedStory)
	for i := range rankedStories.Ranked {
		rankedStoryMap[rankedStories.Ranked[i].StoryID] = &rankedStories.Ranked[i]
	}

	// Build arrays of stories in ranked order with their ranked info
	stories := make([]*types.Story, 0, len(rankedStories.Ranked))
	rankedList := make([]*types.RankedStory, 0, len(rankedStories.Ranked))
	for _, rankedStory := range rankedStories.Ranked {
		if story, exists := storyMap[rankedStory.StoryID]; exists {
			stories = append(stories, story)
			rankedList = append(rankedList, rankedStoryMap[rankedStory.StoryID])
		}
	}

	if len(stories) == 0 {
		return &types.ResumePlan{
			SelectedStories: []types.SelectedStory{},
			SpaceBudget:     *spaceBudget,
			Coverage: types.Coverage{
				TopSkillsCovered: []string{},
				CoverageScore:    0.0,
			},
		}, nil
	}

	// Pre-compute values for all story/bullet combinations
	storyValues := make(map[int][]storyValue)
	for i, story := range stories {
		rankedStory := rankedList[i]
		combinations := generateBulletCombinations(story.Bullets)
		values := make([]storyValue, 0, len(combinations))
		for _, combo := range combinations {
			value := computeStoryValue(rankedStory, combo, skillTargets)
			values = append(values, value)
		}
		storyValues[i] = values
	}

	// Use Hybrid Selection Strategy (Greedy + Knapsack)
	ratio := spaceBudget.SkillMatchRatio
	if ratio == 0 {
		ratio = 0.8 // Safety default
	}
	selections, _, err := SelectHybrid(stories, rankedStories, skillTargets, spaceBudget.MaxLines, ratio)
	if err != nil {
		return nil, fmt.Errorf("failed to select content: %w", err)
	}

	// Materialize the plan from selections
	selectedStories := make([]types.SelectedStory, 0, len(selections))
	allSelectedBullets := make([]types.Bullet, 0)

	for _, selection := range selections {
		story, exists := storyMap[selection.storyID]
		if !exists {
			continue
		}

		// Find selected bullets
		bulletIDSet := make(map[string]bool)
		for _, bulletID := range selection.bulletIDs {
			bulletIDSet[bulletID] = true
		}

		totalLines := 0
		for _, bullet := range story.Bullets {
			if bulletIDSet[bullet.ID] {
				allSelectedBullets = append(allSelectedBullets, bullet)
				totalLines += estimateLines(bullet.LengthChars)
			}
		}

		selectedStory := types.SelectedStory{
			StoryID:        selection.storyID,
			BulletIDs:      selection.bulletIDs,
			Section:        "experience", // Default to experience section
			EstimatedLines: totalLines,
		}

		selectedStories = append(selectedStories, selectedStory)
	}

	// Compute coverage metrics
	coverage := computeCoverage(allSelectedBullets, skillTargets)

	return &types.ResumePlan{
		SelectedStories: selectedStories,
		SpaceBudget:     *spaceBudget,
		Coverage:        coverage,
	}, nil
}

// computeCoverage computes skill coverage metrics for the selected bullets
func computeCoverage(selectedBullets []types.Bullet, skillTargets *types.SkillTargets) types.Coverage {
	if skillTargets == nil || len(skillTargets.Skills) == 0 {
		return types.Coverage{
			TopSkillsCovered: []string{},
			CoverageScore:    0.0,
		}
	}

	// Build skill weight map
	skillWeightMap := make(map[string]float64)
	for _, skill := range skillTargets.Skills {
		skillWeightMap[skill.Name] = skill.Weight
	}

	// Collect covered skills with their weights
	coveredSkillsMap := make(map[string]float64)
	for _, bullet := range selectedBullets {
		for _, skill := range bullet.Skills {
			if weight, found := skillWeightMap[skill]; found {
				// Use max weight if skill appears multiple times
				if existingWeight, exists := coveredSkillsMap[skill]; !exists || weight > existingWeight {
					coveredSkillsMap[skill] = weight
				}
			}
		}
	}

	// Get top skills (sorted by weight, take top 10)
	type skillWeight struct {
		name   string
		weight float64
	}
	topSkillsList := make([]skillWeight, 0, len(coveredSkillsMap))
	for name, weight := range coveredSkillsMap {
		topSkillsList = append(topSkillsList, skillWeight{name: name, weight: weight})
	}

	// Sort by weight descending
	for i := 0; i < len(topSkillsList); i++ {
		for j := i + 1; j < len(topSkillsList); j++ {
			if topSkillsList[i].weight < topSkillsList[j].weight {
				topSkillsList[i], topSkillsList[j] = topSkillsList[j], topSkillsList[i]
			}
		}
	}

	// Take top 10
	maxTopSkills := 10
	if len(topSkillsList) > maxTopSkills {
		topSkillsList = topSkillsList[:maxTopSkills]
	}

	topSkillsCovered := make([]string, 0, len(topSkillsList))
	for _, sw := range topSkillsList {
		topSkillsCovered = append(topSkillsCovered, sw.name)
	}

	// Compute coverage score (same as skill coverage score)
	coverageScore := computeSkillCoverageScore(selectedBullets, skillTargets)

	return types.Coverage{
		TopSkillsCovered: topSkillsCovered,
		CoverageScore:    coverageScore,
	}
}
