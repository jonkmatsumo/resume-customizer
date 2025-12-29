package server

import (
	"net/http"

	"github.com/google/uuid"
)

// handleGetJobProfile retrieves a job profile by its ID
func (s *Server) handleGetJobProfile(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	profileID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid job profile ID")
		return
	}

	profile, err := s.db.GetJobProfileByID(r.Context(), profileID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if profile == nil {
		s.errorResponse(w, http.StatusNotFound, "Job profile not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, profile)
}

// handleGetJobProfileByPostingID retrieves a job profile for a posting
func (s *Server) handleGetJobProfileByPostingID(w http.ResponseWriter, r *http.Request) {
	postingIDStr := r.PathValue("posting_id")
	postingID, err := uuid.Parse(postingIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid posting ID")
		return
	}

	profile, err := s.db.GetJobProfileByPostingID(r.Context(), postingID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if profile == nil {
		s.errorResponse(w, http.StatusNotFound, "Job profile not found for this posting")
		return
	}

	s.jsonResponse(w, http.StatusOK, profile)
}

// handleGetRequirements retrieves requirements for a job profile
func (s *Server) handleGetRequirements(w http.ResponseWriter, r *http.Request) {
	profileIDStr := r.PathValue("id")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid job profile ID")
		return
	}

	// Verify profile exists
	profile, err := s.db.GetJobProfileByID(r.Context(), profileID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if profile == nil {
		s.errorResponse(w, http.StatusNotFound, "Job profile not found")
		return
	}

	requirements, err := s.db.GetRequirementsByProfileID(r.Context(), profileID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"requirements": requirements,
		"count":        len(requirements),
	})
}

// handleGetResponsibilities retrieves responsibilities for a job profile
func (s *Server) handleGetResponsibilities(w http.ResponseWriter, r *http.Request) {
	profileIDStr := r.PathValue("id")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid job profile ID")
		return
	}

	// Verify profile exists
	profile, err := s.db.GetJobProfileByID(r.Context(), profileID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if profile == nil {
		s.errorResponse(w, http.StatusNotFound, "Job profile not found")
		return
	}

	responsibilities, err := s.db.GetResponsibilitiesByProfileID(r.Context(), profileID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"responsibilities": responsibilities,
		"count":            len(responsibilities),
	})
}

// handleGetKeywords retrieves keywords for a job profile
func (s *Server) handleGetKeywords(w http.ResponseWriter, r *http.Request) {
	profileIDStr := r.PathValue("id")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid job profile ID")
		return
	}

	// Verify profile exists
	profile, err := s.db.GetJobProfileByID(r.Context(), profileID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if profile == nil {
		s.errorResponse(w, http.StatusNotFound, "Job profile not found")
		return
	}

	keywords, err := s.db.GetKeywordsByProfileID(r.Context(), profileID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"keywords": keywords,
		"count":    len(keywords),
	})
}
