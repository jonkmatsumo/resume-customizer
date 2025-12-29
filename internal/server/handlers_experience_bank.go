package server

import (
	"net/http"

	"github.com/google/uuid"
)

// handleListStories lists all stories for a user
func (s *Server) handleListStories(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	stories, err := s.db.ListStoriesByUser(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"stories": stories,
		"count":   len(stories),
	})
}

// handleGetStory retrieves a single story by its UUID, scoped to the user
func (s *Server) handleGetStory(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	storyIDStr := r.PathValue("story_id")
	storyID, err := uuid.Parse(storyIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid story ID")
		return
	}

	story, err := s.db.GetStoryByID(r.Context(), storyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if story == nil {
		s.errorResponse(w, http.StatusNotFound, "Story not found")
		return
	}

	// Verify story belongs to user (security check)
	if story.UserID != userID {
		s.errorResponse(w, http.StatusNotFound, "Story not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, story)
}

// handleGetStoryBullets retrieves all bullets for a specific story
func (s *Server) handleGetStoryBullets(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	storyIDStr := r.PathValue("story_id")
	storyID, err := uuid.Parse(storyIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid story ID")
		return
	}

	// First verify story exists and belongs to user
	story, err := s.db.GetStoryByID(r.Context(), storyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if story == nil {
		s.errorResponse(w, http.StatusNotFound, "Story not found")
		return
	}

	// Verify story belongs to user (security check)
	if story.UserID != userID {
		s.errorResponse(w, http.StatusNotFound, "Story not found")
		return
	}

	// Get bullets for the story
	bullets, err := s.db.GetBulletsByStoryID(r.Context(), storyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"bullets": bullets,
		"count":   len(bullets),
	})
}

// handleListSkills lists all unique skills used by the user across all their stories
func (s *Server) handleListSkills(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	skills, err := s.db.ListSkillsByUserID(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"skills": skills,
		"count":  len(skills),
	})
}

// handleGetSkillBullets retrieves all bullets that use a specific skill, scoped to the user
func (s *Server) handleGetSkillBullets(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	skillIDStr := r.PathValue("skill_id")
	skillID, err := uuid.Parse(skillIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid skill ID")
		return
	}

	bullets, err := s.db.GetBulletsBySkillIDAndUserID(r.Context(), skillID, userID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"bullets": bullets,
		"count":   len(bullets),
	})
}
