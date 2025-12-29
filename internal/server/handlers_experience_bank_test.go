package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleListStories_InvalidUserID tests list stories with invalid user ID
func TestHandleListStories_InvalidUserID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/users/not-a-uuid/experience-bank/stories", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleListStories(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid user ID")
}

// TestHandleGetStory_InvalidUserID tests get story with invalid user ID
func TestHandleGetStory_InvalidUserID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/users/not-a-uuid/experience-bank/stories/123", nil)
	req.SetPathValue("id", "not-a-uuid")
	req.SetPathValue("story_id", "123")
	w := httptest.NewRecorder()

	s.handleGetStory(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleGetStory_InvalidStoryID tests get story with invalid story ID
func TestHandleGetStory_InvalidStoryID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/users/123e4567-e89b-12d3-a456-426614174000/experience-bank/stories/not-a-uuid", nil)
	req.SetPathValue("id", "123e4567-e89b-12d3-a456-426614174000")
	req.SetPathValue("story_id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetStory(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid story ID")
}

// TestHandleGetStoryBullets_InvalidUserID tests get story bullets with invalid user ID
func TestHandleGetStoryBullets_InvalidUserID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/users/not-a-uuid/experience-bank/stories/123/bullets", nil)
	req.SetPathValue("id", "not-a-uuid")
	req.SetPathValue("story_id", "123")
	w := httptest.NewRecorder()

	s.handleGetStoryBullets(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleGetStoryBullets_InvalidStoryID tests get story bullets with invalid story ID
func TestHandleGetStoryBullets_InvalidStoryID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/users/123e4567-e89b-12d3-a456-426614174000/experience-bank/stories/not-a-uuid/bullets", nil)
	req.SetPathValue("id", "123e4567-e89b-12d3-a456-426614174000")
	req.SetPathValue("story_id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetStoryBullets(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleListSkills_InvalidUserID tests list skills with invalid user ID
func TestHandleListSkills_InvalidUserID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/users/not-a-uuid/experience-bank/skills", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleListSkills(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid user ID")
}

// TestHandleGetSkillBullets_InvalidUserID tests get skill bullets with invalid user ID
func TestHandleGetSkillBullets_InvalidUserID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/users/not-a-uuid/experience-bank/skills/123/bullets", nil)
	req.SetPathValue("id", "not-a-uuid")
	req.SetPathValue("skill_id", "123")
	w := httptest.NewRecorder()

	s.handleGetSkillBullets(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleGetSkillBullets_InvalidSkillID tests get skill bullets with invalid skill ID
func TestHandleGetSkillBullets_InvalidSkillID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/users/123e4567-e89b-12d3-a456-426614174000/experience-bank/skills/not-a-uuid/bullets", nil)
	req.SetPathValue("id", "123e4567-e89b-12d3-a456-426614174000")
	req.SetPathValue("skill_id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetSkillBullets(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid skill ID")
}
