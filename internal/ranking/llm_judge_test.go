package ranking

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jonathan/resume-customizer/internal/llm"
	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMClient implements llm.Client for testing
type MockLLMClient struct {
	GenerateContentFunc func(ctx context.Context, prompt string, tier llm.ModelTier) (string, error)
	GenerateJSONFunc    func(ctx context.Context, prompt string, tier llm.ModelTier) (string, error)
	GetModelFunc        func(tier llm.ModelTier) string
	CloseFunc           func() error
}

func (m *MockLLMClient) GenerateContent(ctx context.Context, prompt string, tier llm.ModelTier) (string, error) {
	if m.GenerateContentFunc != nil {
		return m.GenerateContentFunc(ctx, prompt, tier)
	}
	return "", nil
}

func (m *MockLLMClient) GenerateJSON(ctx context.Context, prompt string, tier llm.ModelTier) (string, error) {
	if m.GenerateJSONFunc != nil {
		return m.GenerateJSONFunc(ctx, prompt, tier)
	}
	return `{"relevance_score": 0.75, "reasoning": "Mock reasoning"}`, nil
}

func (m *MockLLMClient) GetModel(tier llm.ModelTier) string {
	if m.GetModelFunc != nil {
		return m.GetModelFunc(tier)
	}
	return "mock-model"
}

func (m *MockLLMClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func TestJudgeStoryRelevance_Success(t *testing.T) {
	mockClient := &MockLLMClient{
		GenerateJSONFunc: func(_ context.Context, _ string, _ llm.ModelTier) (string, error) {
			return `{"relevance_score": 0.85, "reasoning": "Strong alignment with Go and distributed systems requirements"}`, nil
		},
	}

	story := &types.Story{
		ID:      "story_001",
		Company: "TechCorp",
		Role:    "Software Engineer",
		Bullets: []types.Bullet{
			{Text: "Built Go microservices", Skills: []string{"Go", "Microservices"}},
		},
	}

	jobProfile := &types.JobProfile{
		Company:   "TargetCorp",
		RoleTitle: "Senior Go Developer",
		HardRequirements: []types.Requirement{
			{Skill: "Go", Evidence: "Required"},
		},
		Keywords: []string{"microservices", "distributed systems"},
	}

	ctx := context.Background()
	result, err := JudgeStoryRelevance(ctx, story, jobProfile, mockClient)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.InDelta(t, 0.85, result.Score, 0.001)
	assert.Contains(t, result.Reasoning, "alignment")
}

func TestJudgeStoryRelevance_LLMError(t *testing.T) {
	mockClient := &MockLLMClient{
		GenerateJSONFunc: func(_ context.Context, _ string, _ llm.ModelTier) (string, error) {
			return "", errors.New("API rate limit exceeded")
		},
	}

	story := &types.Story{
		ID:      "story_001",
		Bullets: []types.Bullet{{Text: "Test bullet"}},
	}

	jobProfile := &types.JobProfile{
		Company:   "TestCorp",
		RoleTitle: "Developer",
	}

	ctx := context.Background()
	result, err := JudgeStoryRelevance(ctx, story, jobProfile, mockClient)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "LLM generation failed")
}

func TestJudgeStoryRelevance_InvalidJSON(t *testing.T) {
	mockClient := &MockLLMClient{
		GenerateJSONFunc: func(_ context.Context, _ string, _ llm.ModelTier) (string, error) {
			return "not valid json", nil
		},
	}

	story := &types.Story{
		ID:      "story_001",
		Bullets: []types.Bullet{{Text: "Test bullet"}},
	}

	jobProfile := &types.JobProfile{
		Company: "TestCorp",
	}

	ctx := context.Background()
	result, err := JudgeStoryRelevance(ctx, story, jobProfile, mockClient)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse LLM response")
}

func TestJudgeStoryRelevance_ScoreClamping(t *testing.T) {
	tests := []struct {
		name          string
		llmScore      float64
		expectedScore float64
	}{
		{"Score above 1.0 is clamped", 1.5, 1.0},
		{"Score below 0.0 is clamped", -0.2, 0.0},
		{"Normal score unchanged", 0.75, 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLLMClient{
				GenerateJSONFunc: func(_ context.Context, _ string, _ llm.ModelTier) (string, error) {
					resp := llmJudgeResponse{
						RelevanceScore: tt.llmScore,
						Reasoning:      "Test reasoning",
					}
					jsonBytes, _ := json.Marshal(resp)
					return string(jsonBytes), nil
				},
			}

			story := &types.Story{
				ID:      "story_001",
				Bullets: []types.Bullet{{Text: "Test"}},
			}

			jobProfile := &types.JobProfile{Company: "TestCorp"}

			ctx := context.Background()
			result, err := JudgeStoryRelevance(ctx, story, jobProfile, mockClient)

			require.NoError(t, err)
			assert.InDelta(t, tt.expectedScore, result.Score, 0.001)
		})
	}
}

func TestJudgeStoriesRelevance_PartialFailure(t *testing.T) {
	callCount := 0
	mockClient := &MockLLMClient{
		GenerateJSONFunc: func(_ context.Context, _ string, _ llm.ModelTier) (string, error) {
			callCount++
			if callCount == 2 {
				// Second story fails
				return "", errors.New("temporary API error")
			}
			return `{"relevance_score": 0.8, "reasoning": "Good match"}`, nil
		},
	}

	stories := []types.Story{
		{ID: "story_001", Bullets: []types.Bullet{{Text: "First"}}},
		{ID: "story_002", Bullets: []types.Bullet{{Text: "Second"}}}, // This one fails
		{ID: "story_003", Bullets: []types.Bullet{{Text: "Third"}}},
	}

	jobProfile := &types.JobProfile{Company: "TestCorp"}

	ctx := context.Background()
	results := JudgeStoriesRelevance(ctx, stories, jobProfile, mockClient)

	// Should have results for all stories
	assert.Len(t, results, 3)

	// Story 1 and 3 should have valid results
	assert.NotNil(t, results["story_001"])
	assert.InDelta(t, 0.8, results["story_001"].Score, 0.001)

	// Story 2 should be nil (failed)
	assert.Nil(t, results["story_002"])

	// Story 3 should have valid result
	assert.NotNil(t, results["story_003"])
}

func TestJudgeStoriesRelevance_ContextCancellation(t *testing.T) {
	mockClient := &MockLLMClient{
		GenerateJSONFunc: func(_ context.Context, _ string, _ llm.ModelTier) (string, error) {
			return `{"relevance_score": 0.8, "reasoning": "Match"}`, nil
		},
	}

	stories := []types.Story{
		{ID: "story_001", Bullets: []types.Bullet{{Text: "First"}}},
		{ID: "story_002", Bullets: []types.Bullet{{Text: "Second"}}},
	}

	jobProfile := &types.JobProfile{Company: "TestCorp"}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	results := JudgeStoriesRelevance(ctx, stories, jobProfile, mockClient)

	// Should return early with no results
	assert.Empty(t, results)
}

func TestBuildJudgePrompt_Content(t *testing.T) {
	story := &types.Story{
		ID:      "story_001",
		Company: "TechCorp",
		Role:    "Staff Engineer",
		Bullets: []types.Bullet{
			{Text: "Built scalable Go services", Skills: []string{"Go", "Microservices"}},
			{Text: "Led team of 5 engineers", Skills: []string{"Leadership"}},
		},
	}

	jobProfile := &types.JobProfile{
		Company:   "TargetCompany",
		RoleTitle: "Senior Software Engineer",
		HardRequirements: []types.Requirement{
			{Skill: "Go"},
			{Skill: "Kubernetes"},
		},
		NiceToHaves: []types.Requirement{
			{Skill: "Leadership"},
		},
		Keywords: []string{"microservices", "cloud", "distributed"},
	}

	prompt := buildJudgePrompt(story, jobProfile)

	// Check all important content is in the prompt
	assert.Contains(t, prompt, "TargetCompany")
	assert.Contains(t, prompt, "Senior Software Engineer")
	assert.Contains(t, prompt, "Go")
	assert.Contains(t, prompt, "Kubernetes")
	assert.Contains(t, prompt, "Leadership (nice to have)")
	assert.Contains(t, prompt, "microservices")
	assert.Contains(t, prompt, "TechCorp")
	assert.Contains(t, prompt, "Staff Engineer")
	assert.Contains(t, prompt, "Built scalable Go services")
	assert.Contains(t, prompt, "Led team of 5 engineers")
}

func TestBuildJudgePrompt_EmptyFields(t *testing.T) {
	story := &types.Story{
		ID:      "story_001",
		Company: "",
		Role:    "",
		Bullets: []types.Bullet{},
	}

	jobProfile := &types.JobProfile{
		Company:          "",
		RoleTitle:        "",
		HardRequirements: nil,
		NiceToHaves:      nil,
		Keywords:         nil,
	}

	prompt := buildJudgePrompt(story, jobProfile)

	// Should handle empty fields gracefully with "Not specified" placeholders
	assert.Contains(t, prompt, "Not specified")
}
