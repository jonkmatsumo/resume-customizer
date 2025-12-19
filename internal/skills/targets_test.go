package skills

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSkillTargets_OnlyHardRequirements(t *testing.T) {
	profile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
			{Skill: "Python", Evidence: "Required"},
		},
	}

	targets, err := BuildSkillTargets(profile)
	require.NoError(t, err)
	require.Len(t, targets.Skills, 2)

	// Should be sorted by weight (descending), so order may vary, but both should have weight 1.0
	for _, skill := range targets.Skills {
		assert.Equal(t, 1.0, skill.Weight)
		assert.Equal(t, "hard_requirement", skill.Source)
		assert.Contains(t, []string{"Go", "Python"}, skill.Name)
	}
}

func TestBuildSkillTargets_OnlyNiceToHaves(t *testing.T) {
	profile := &types.JobProfile{
		NiceToHaves: []types.Requirement{
			{Skill: "Kubernetes", Evidence: "Preferred"},
			{Skill: "Docker", Evidence: "Preferred"},
		},
	}

	targets, err := BuildSkillTargets(profile)
	require.NoError(t, err)
	require.Len(t, targets.Skills, 2)

	for _, skill := range targets.Skills {
		assert.Equal(t, 0.5, skill.Weight)
		assert.Equal(t, "nice_to_have", skill.Source)
		assert.Contains(t, []string{"Kubernetes", "Docker"}, skill.Name)
	}
}

func TestBuildSkillTargets_OnlyKeywords(t *testing.T) {
	profile := &types.JobProfile{
		Keywords: []string{"microservices", "distributed systems"},
	}

	targets, err := BuildSkillTargets(profile)
	require.NoError(t, err)
	require.Len(t, targets.Skills, 2)

	for _, skill := range targets.Skills {
		assert.Equal(t, 0.3, skill.Weight)
		assert.Equal(t, "keyword", skill.Source)
		// NormalizeSkillName capitalizes first letter of single words ("microservices" -> "Microservices")
		// but multi-word lowercase stays lowercase ("distributed systems" -> "distributed systems")
		assert.Contains(t, []string{"Microservices", "distributed systems"}, skill.Name)
	}
}

func TestBuildSkillTargets_MixedSources(t *testing.T) {
	profile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
		NiceToHaves: []types.Requirement{
			{Skill: "Kubernetes", Evidence: "Preferred"},
		},
		Keywords: []string{"microservices"},
	}

	targets, err := BuildSkillTargets(profile)
	require.NoError(t, err)
	require.Len(t, targets.Skills, 3)

	// Should be sorted by weight descending: Go (1.0), Kubernetes (0.5), Microservices (0.3)
	assert.Equal(t, "Go", targets.Skills[0].Name)
	assert.Equal(t, 1.0, targets.Skills[0].Weight)
	assert.Equal(t, "hard_requirement", targets.Skills[0].Source)

	assert.Equal(t, "Kubernetes", targets.Skills[1].Name)
	assert.Equal(t, 0.5, targets.Skills[1].Weight)
	assert.Equal(t, "nice_to_have", targets.Skills[1].Source)

	assert.Equal(t, "Microservices", targets.Skills[2].Name)
	assert.Equal(t, 0.3, targets.Skills[2].Weight)
	assert.Equal(t, "keyword", targets.Skills[2].Source)
}

func TestBuildSkillTargets_EmptyProfile(t *testing.T) {
	profile := &types.JobProfile{}

	_, err := BuildSkillTargets(profile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no skills found")
}

func TestBuildSkillTargets_SkillNormalization(t *testing.T) {
	profile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "golang", Evidence: "Required"},
			{Skill: "JS", Evidence: "Required"},
			{Skill: "k8s", Evidence: "Required"},
		},
	}

	targets, err := BuildSkillTargets(profile)
	require.NoError(t, err)
	require.Len(t, targets.Skills, 3)

	skillNames := make(map[string]bool)
	for _, skill := range targets.Skills {
		skillNames[skill.Name] = true
		assert.Equal(t, 1.0, skill.Weight)
		assert.Equal(t, "hard_requirement", skill.Source)
	}

	// Should be normalized
	assert.True(t, skillNames["Go"], "golang should normalize to Go")
	assert.True(t, skillNames["JavaScript"], "JS should normalize to JavaScript")
	assert.True(t, skillNames["Kubernetes"], "k8s should normalize to Kubernetes")
}

func TestBuildSkillTargets_Deduplication_MaxWeight(t *testing.T) {
	profile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
		Keywords: []string{"Go"},
	}

	targets, err := BuildSkillTargets(profile)
	require.NoError(t, err)
	require.Len(t, targets.Skills, 1, "Go should appear only once after deduplication")

	skill := targets.Skills[0]
	assert.Equal(t, "Go", skill.Name)
	assert.Equal(t, 1.0, skill.Weight, "Should take max weight (1.0 from hard_requirement)")
	assert.Equal(t, "hard_requirement", skill.Source, "Should reflect highest priority source")
}

func TestBuildSkillTargets_Deduplication_NormalizedNames(t *testing.T) {
	profile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "golang", Evidence: "Required"},
		},
		Keywords: []string{"Go"},
	}

	targets, err := BuildSkillTargets(profile)
	require.NoError(t, err)
	require.Len(t, targets.Skills, 1, "golang and Go should normalize to same skill")

	skill := targets.Skills[0]
	assert.Equal(t, "Go", skill.Name)
	assert.Equal(t, 1.0, skill.Weight, "Should take max weight")
	assert.Equal(t, "hard_requirement", skill.Source)
}

func TestBuildSkillTargets_SourcePriority(t *testing.T) {
	profile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
		NiceToHaves: []types.Requirement{
			{Skill: "Go", Evidence: "Preferred"},
		},
		Keywords: []string{"Go"},
	}

	targets, err := BuildSkillTargets(profile)
	require.NoError(t, err)
	require.Len(t, targets.Skills, 1)

	skill := targets.Skills[0]
	assert.Equal(t, "Go", skill.Name)
	assert.Equal(t, 1.0, skill.Weight)
	assert.Equal(t, "hard_requirement", skill.Source, "Should reflect highest priority source")
}

func TestBuildSkillTargets_SortingByWeight(t *testing.T) {
	profile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "HardSkill", Evidence: "Required"},
		},
		NiceToHaves: []types.Requirement{
			{Skill: "NiceSkill1", Evidence: "Preferred"},
			{Skill: "NiceSkill2", Evidence: "Preferred"},
		},
		Keywords: []string{"Keyword1", "Keyword2", "Keyword3"},
	}

	targets, err := BuildSkillTargets(profile)
	require.NoError(t, err)
	require.Len(t, targets.Skills, 6)

	// Verify descending order by weight
	prevWeight := 1.1 // Start higher than any possible weight
	for _, skill := range targets.Skills {
		assert.LessOrEqual(t, skill.Weight, prevWeight, "Skills should be sorted by weight descending")
		prevWeight = skill.Weight
	}

	// Verify first skill has highest weight
	assert.Equal(t, 1.0, targets.Skills[0].Weight)
	assert.Equal(t, "HardSkill", targets.Skills[0].Name)
}

func TestBuildSkillTargets_WeightsMatchRules(t *testing.T) {
	profile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "Hard", Evidence: "Required"},
		},
		NiceToHaves: []types.Requirement{
			{Skill: "Nice", Evidence: "Preferred"},
		},
		Keywords: []string{"Keyword"},
	}

	targets, err := BuildSkillTargets(profile)
	require.NoError(t, err)
	require.Len(t, targets.Skills, 3)

	weights := make(map[string]float64)
	for _, skill := range targets.Skills {
		weights[skill.Name] = skill.Weight
	}

	assert.Equal(t, 1.0, weights["Hard"], "Hard requirements should have weight 1.0")
	assert.Equal(t, 0.5, weights["Nice"], "Nice-to-haves should have weight 0.5")
	assert.Equal(t, 0.3, weights["Keyword"], "Keywords should have weight 0.3")
}

func TestBuildSkillTargets_EmptySkillNamesSkipped(t *testing.T) {
	profile := &types.JobProfile{
		HardRequirements: []types.Requirement{
			{Skill: "", Evidence: "Empty"},
			{Skill: "   ", Evidence: "Whitespace"},
			{Skill: "Valid", Evidence: "Valid"},
		},
	}

	targets, err := BuildSkillTargets(profile)
	require.NoError(t, err)
	require.Len(t, targets.Skills, 1, "Empty and whitespace-only skill names should be skipped")

	assert.Equal(t, "Valid", targets.Skills[0].Name)
}
