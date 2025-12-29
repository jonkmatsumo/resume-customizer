package server

import (
	"net/http"

	"github.com/google/uuid"
)

// handleGetCompanyProfile retrieves the profile for a company
func (s *Server) handleGetCompanyProfile(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("company_id")
	companyID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	profile, err := s.db.GetCompanyProfileByCompanyID(r.Context(), companyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if profile == nil {
		s.errorResponse(w, http.StatusNotFound, "Company profile not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, profile)
}

// handleGetStyleRules retrieves style rules for a company profile
func (s *Server) handleGetStyleRules(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("company_id")
	companyID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	// Get profile first to get profile ID
	profile, err := s.db.GetCompanyProfileByCompanyID(r.Context(), companyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if profile == nil {
		s.errorResponse(w, http.StatusNotFound, "Company profile not found")
		return
	}

	rules, err := s.db.GetStyleRulesByProfileID(r.Context(), profile.ID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"style_rules": rules,
		"count":       len(rules),
	})
}

// handleGetTabooPhrases retrieves taboo phrases for a company profile
func (s *Server) handleGetTabooPhrases(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("company_id")
	companyID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	profile, err := s.db.GetCompanyProfileByCompanyID(r.Context(), companyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if profile == nil {
		s.errorResponse(w, http.StatusNotFound, "Company profile not found")
		return
	}

	phrases, err := s.db.GetTabooPhrasesByProfileID(r.Context(), profile.ID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"taboo_phrases": phrases,
		"count":         len(phrases),
	})
}

// handleGetValues retrieves company values for a profile
func (s *Server) handleGetValues(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("company_id")
	companyID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	profile, err := s.db.GetCompanyProfileByCompanyID(r.Context(), companyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if profile == nil {
		s.errorResponse(w, http.StatusNotFound, "Company profile not found")
		return
	}

	values, err := s.db.GetValuesByProfileID(r.Context(), profile.ID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"values": values,
		"count":  len(values),
	})
}

// handleGetSources retrieves evidence URLs for a company profile
func (s *Server) handleGetSources(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("company_id")
	companyID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	profile, err := s.db.GetCompanyProfileByCompanyID(r.Context(), companyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if profile == nil {
		s.errorResponse(w, http.StatusNotFound, "Company profile not found")
		return
	}

	sources, err := s.db.GetSourcesByProfileID(r.Context(), profile.ID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"sources": sources,
		"count":   len(sources),
	})
}
