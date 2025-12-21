package ranking

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestComputeEducationRuleScore_NoRequirements(t *testing.T) {
	edu := types.Education{
		ID:     "edu-1",
		School: "MIT",
		Degree: "bachelor",
		Field:  "Computer Science",
	}

	score := computeEducationRuleScore(edu, nil)
	assert.Equal(t, 1.0, score, "No requirements should return full score")
}

func TestComputeEducationRuleScore_DegreeMatch(t *testing.T) {
	edu := types.Education{
		ID:     "edu-1",
		School: "MIT",
		Degree: "master",
		Field:  "Computer Science",
	}

	req := &types.EducationRequirements{
		MinDegree: "bachelor",
	}

	score := computeEducationRuleScore(edu, req)
	assert.Equal(t, 1.0, score, "Master exceeds bachelor requirement")
}

func TestComputeEducationRuleScore_DegreeBelowRequirement(t *testing.T) {
	edu := types.Education{
		ID:     "edu-1",
		School: "MIT",
		Degree: "bachelor",
		Field:  "Computer Science",
	}

	req := &types.EducationRequirements{
		MinDegree: "phd",
	}

	score := computeEducationRuleScore(edu, req)
	assert.Less(t, score, 0.5, "Bachelor below PhD requirement should score low")
}

func TestComputeEducationRuleScore_FieldMatch(t *testing.T) {
	edu := types.Education{
		ID:     "edu-1",
		School: "MIT",
		Degree: "bachelor",
		Field:  "Computer Science",
	}

	req := &types.EducationRequirements{
		PreferredFields: []string{"Computer Science", "Data Science"},
	}

	score := computeEducationRuleScore(edu, req)
	assert.Equal(t, 1.0, score, "Exact field match should return full score")
}

func TestComputeEducationRuleScore_RelatedField(t *testing.T) {
	edu := types.Education{
		ID:     "edu-1",
		School: "MIT",
		Degree: "bachelor",
		Field:  "Software Engineering",
	}

	req := &types.EducationRequirements{
		PreferredFields: []string{"Computer Science"},
	}

	score := computeEducationRuleScore(edu, req)
	assert.GreaterOrEqual(t, score, 0.6, "Related field should score moderately")
}

func TestComputeFieldMatchScore_ExactMatch(t *testing.T) {
	score := computeFieldMatchScore("Computer Science", []string{"Computer Science"})
	assert.Equal(t, 1.0, score)
}

func TestComputeFieldMatchScore_PartialMatch(t *testing.T) {
	score := computeFieldMatchScore("Computer Science and Engineering", []string{"Computer Science"})
	assert.Equal(t, 1.0, score, "Substring match should return full score")
}

func TestComputeFieldMatchScore_NoMatch(t *testing.T) {
	score := computeFieldMatchScore("Biology", []string{"Computer Science", "Data Science"})
	assert.Equal(t, 0.2, score, "Unrelated field should return low score")
}
