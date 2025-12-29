package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExperienceBankEndpoints_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	ctx := context.Background()

	// Create test user
	userID, err := s.db.CreateUser(ctx, "Test User", "test-experience@example.com", "")
	require.NoError(t, err)

	// Create test job
	job := &db.Job{
		UserID:    userID,
		Company:   "Test Company",
		RoleTitle: "Software Engineer",
	}
	jobID, err := s.db.CreateJob(ctx, job)
	require.NoError(t, err)
	job.ID = jobID

	// Create test story
	story, err := s.db.CreateStory(ctx, &db.StoryCreateInput{
		StoryID: "test-story-1",
		UserID:  userID,
		JobID:   job.ID,
		Title:   "Test Story",
		Bullets: []db.BulletCreateInput{
			{
				BulletID:         "bullet-1",
				Text:             "Implemented feature X using Go",
				EvidenceStrength: db.EvidenceStrengthHigh,
				Skills:           []string{"Go", "Backend"},
				Ordinal:          1,
			},
			{
				BulletID:         "bullet-2",
				Text:             "Optimized database queries",
				EvidenceStrength: db.EvidenceStrengthMedium,
				Skills:           []string{"SQL", "Database"},
				Ordinal:          2,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, story)

	// Get the skill ID for testing
	skill, err := s.db.GetSkillByName(ctx, "Go")
	require.NoError(t, err)
	require.NotNil(t, skill)

	// Create another user for security testing
	otherUserID, err := s.db.CreateUser(ctx, "Other User", "other-user@example.com", "")
	require.NoError(t, err)

	// Test 1: List stories for user
	t.Run("ListStories_Integration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/experience-bank/stories", nil)
		req.SetPathValue("id", userID.String())
		w := httptest.NewRecorder()

		s.handleListStories(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "stories")
		assert.Contains(t, resp, "count")

		stories, ok := resp["stories"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(stories), 1)
	})

	// Test 2: Get story by ID
	t.Run("GetStory_Integration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/experience-bank/stories/"+story.ID.String()+"/bullets", nil)
		req.SetPathValue("id", userID.String())
		req.SetPathValue("story_id", story.ID.String())
		w := httptest.NewRecorder()

		s.handleGetStory(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var storyResp db.Story
		err := json.Unmarshal(w.Body.Bytes(), &storyResp)
		require.NoError(t, err)
		assert.Equal(t, story.ID, storyResp.ID)
		assert.Equal(t, story.StoryID, storyResp.StoryID)
		assert.Equal(t, userID, storyResp.UserID)
		assert.NotNil(t, storyResp.Bullets)
		assert.GreaterOrEqual(t, len(storyResp.Bullets), 2)
	})

	// Test 3: Get story not belonging to user (security check)
	t.Run("GetStory_NotBelongingToUser", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/"+otherUserID.String()+"/experience-bank/stories/"+story.ID.String(), nil)
		req.SetPathValue("id", otherUserID.String())
		req.SetPathValue("story_id", story.ID.String())
		w := httptest.NewRecorder()

		s.handleGetStory(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "not found")
	})

	// Test 4: Get story bullets
	t.Run("GetStoryBullets_Integration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/experience-bank/stories/"+story.ID.String()+"/bullets", nil)
		req.SetPathValue("id", userID.String())
		req.SetPathValue("story_id", story.ID.String())
		w := httptest.NewRecorder()

		s.handleGetStoryBullets(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "bullets")
		assert.Contains(t, resp, "count")

		bullets, ok := resp["bullets"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(bullets), 2)

		// Verify bullets are ordered by ordinal
		if len(bullets) >= 2 {
			firstBullet, ok := bullets[0].(map[string]any)
			require.True(t, ok)
			secondBullet, ok := bullets[1].(map[string]any)
			require.True(t, ok)
			firstOrdinal, _ := firstBullet["ordinal"].(float64)
			secondOrdinal, _ := secondBullet["ordinal"].(float64)
			assert.LessOrEqual(t, firstOrdinal, secondOrdinal)
		}
	})

	// Test 5: List skills for user
	t.Run("ListSkills_Integration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/experience-bank/skills", nil)
		req.SetPathValue("id", userID.String())
		w := httptest.NewRecorder()

		s.handleListSkills(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "skills")
		assert.Contains(t, resp, "count")

		skills, ok := resp["skills"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(skills), 1)

		// Verify skills are unique and ordered
		skillNames := make(map[string]bool)
		for _, skillAny := range skills {
			skill, ok := skillAny.(map[string]any)
			require.True(t, ok)
			name, ok := skill["name"].(string)
			require.True(t, ok)
			assert.False(t, skillNames[name], "Skill %s should be unique", name)
			skillNames[name] = true
		}
	})

	// Test 6: Get bullets with a specific skill
	t.Run("GetSkillBullets_Integration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/experience-bank/skills/"+skill.ID.String()+"/bullets", nil)
		req.SetPathValue("id", userID.String())
		req.SetPathValue("skill_id", skill.ID.String())
		w := httptest.NewRecorder()

		s.handleGetSkillBullets(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "bullets")
		assert.Contains(t, resp, "count")

		bullets, ok := resp["bullets"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(bullets), 1)

		// Verify bullets contain the skill
		if len(bullets) > 0 {
			firstBullet, ok := bullets[0].(map[string]any)
			require.True(t, ok)
			skills, ok := firstBullet["skills"].([]any)
			require.True(t, ok)
			hasGoSkill := false
			for _, skillAny := range skills {
				if skillStr, ok := skillAny.(string); ok && skillStr == "Go" {
					hasGoSkill = true
					break
				}
			}
			assert.True(t, hasGoSkill, "Bullet should contain Go skill")
		}
	})

	// Test 7: Get story not found
	t.Run("GetStory_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/experience-bank/stories/"+nonExistentID.String(), nil)
		req.SetPathValue("id", userID.String())
		req.SetPathValue("story_id", nonExistentID.String())
		w := httptest.NewRecorder()

		s.handleGetStory(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "not found")
	})
}
