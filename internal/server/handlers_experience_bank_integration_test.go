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

	// Create test user with unique email to avoid conflicts
	uniqueEmail := "test-experience-" + uuid.New().String() + "@example.com"
	userID, err := s.db.CreateUser(ctx, "Test User", uniqueEmail, "")
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

	// Create test story with unique story_id to avoid conflicts from previous test runs
	uniqueStoryID := "test-story-" + uuid.New().String()
	story, err := s.db.CreateStory(ctx, &db.StoryCreateInput{
		StoryID: uniqueStoryID,
		UserID:  userID,
		JobID:   job.ID,
		Title:   "Test Story",
		Bullets: []db.BulletCreateInput{
			{
				BulletID:         "bullet-1-" + uuid.New().String(),
				Text:             "Implemented feature X using Go",
				EvidenceStrength: db.EvidenceStrengthHigh,
				Skills:           []string{"Go", "Backend"},
				Ordinal:          1,
			},
			{
				BulletID:         "bullet-2-" + uuid.New().String(),
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

	// Create another user for security testing with unique email
	otherUniqueEmail := "other-user-" + uuid.New().String() + "@example.com"
	otherUserID, err := s.db.CreateUser(ctx, "Other User", otherUniqueEmail, "")
	require.NoError(t, err)

	// Test 1: List stories for user
	t.Run("ListStories_Integration", func(t *testing.T) {
		// Verify story was created
		verifyStory, verifyErr := s.db.GetStoryByID(ctx, story.ID)
		require.NoError(t, verifyErr)
		require.NotNil(t, verifyStory, "Story should exist in database")
		require.Equal(t, userID, verifyStory.UserID, "Story should belong to the test user")

		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/experience-bank/stories", nil)
		req.SetPathValue("id", userID.String())
		w := httptest.NewRecorder()

		s.handleListStories(w, req)

		require.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err, "Failed to unmarshal response: %s", w.Body.String())
		t.Logf("Response: %+v", resp)
		require.Contains(t, resp, "stories", "Response keys: %v", resp)
		require.Contains(t, resp, "count")

		// JSON unmarshaling converts []Story to []interface{} where each element is map[string]interface{}
		storiesAny := resp["stories"]
		// Handle case where stories might be nil (empty array in JSON)
		if storiesAny == nil {
			t.Logf("stories is nil, checking if it's an empty array in JSON")
			// Try to check if the response has stories as an empty array
			bodyStr := w.Body.String()
			if bodyStr != "" {
				t.Logf("Full response body: %s", bodyStr)
			}
			// If stories is nil, it means the array was empty, which is valid
			// But we expect at least one story from the test setup
			storiesSlice := []interface{}{}
			assert.GreaterOrEqual(t, len(storiesSlice), 1, "Expected at least one story, but got empty array")
		} else {
			storiesSlice, ok := storiesAny.([]interface{})
			require.True(t, ok, "stories should be an array, got: %T, value: %v", storiesAny, storiesAny)
			assert.GreaterOrEqual(t, len(storiesSlice), 1)
		}

		count, ok := resp["count"].(float64)
		require.True(t, ok, "count should be a number")
		assert.GreaterOrEqual(t, int(count), 1)
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

		bulletsAny, ok := resp["bullets"]
		require.True(t, ok, "bullets key should exist")
		bulletsSlice, ok := bulletsAny.([]interface{})
		require.True(t, ok, "bullets should be an array, got: %T", bulletsAny)
		assert.GreaterOrEqual(t, len(bulletsSlice), 2)

		count, ok := resp["count"].(float64)
		require.True(t, ok, "count should be a number")
		assert.GreaterOrEqual(t, int(count), 2)

		// Verify bullets are ordered by ordinal
		if len(bulletsSlice) >= 2 {
			firstBullet, ok := bulletsSlice[0].(map[string]any)
			require.True(t, ok)
			secondBullet, ok := bulletsSlice[1].(map[string]any)
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

		skillsAny, ok := resp["skills"]
		require.True(t, ok, "skills key should exist")
		skillsSlice, ok := skillsAny.([]interface{})
		require.True(t, ok, "skills should be an array, got: %T", skillsAny)
		assert.GreaterOrEqual(t, len(skillsSlice), 1)

		count, ok := resp["count"].(float64)
		require.True(t, ok, "count should be a number")
		assert.GreaterOrEqual(t, int(count), 1)

		// Verify skills are unique and ordered
		skillNames := make(map[string]bool)
		for _, skillAny := range skillsSlice {
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

		bulletsAny, ok := resp["bullets"]
		require.True(t, ok, "bullets key should exist")
		bulletsSlice, ok := bulletsAny.([]interface{})
		require.True(t, ok, "bullets should be an array, got: %T", bulletsAny)
		assert.GreaterOrEqual(t, len(bulletsSlice), 1)

		count, ok := resp["count"].(float64)
		require.True(t, ok, "count should be a number")
		assert.GreaterOrEqual(t, int(count), 1)

		// Verify bullets contain the skill
		if len(bulletsSlice) > 0 {
			firstBullet, ok := bulletsSlice[0].(map[string]any)
			require.True(t, ok, "bullet should be a map")
			skillsAny, ok := firstBullet["skills"]
			require.True(t, ok, "skills key should exist in bullet")
			skillsSlice, ok := skillsAny.([]interface{})
			require.True(t, ok, "skills should be an array, got: %T", skillsAny)
			hasGoSkill := false
			for _, skillAny := range skillsSlice {
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
