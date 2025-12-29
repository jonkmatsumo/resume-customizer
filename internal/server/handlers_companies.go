package server

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
)

// parseQueryInt parses an integer query parameter with default and max values
func parseQueryInt(r *http.Request, key string, defaultValue, maxValue int) int {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil || val < 0 {
		return defaultValue
	}
	if maxValue > 0 && val > maxValue {
		return maxValue
	}
	return val
}

// handleListCompanies lists all companies with research profiles
func (s *Server) handleListCompanies(w http.ResponseWriter, r *http.Request) {
	limit := parseQueryInt(r, "limit", 50, 100)
	offset := parseQueryInt(r, "offset", 0, 0)

	companies, total, err := s.db.ListCompaniesWithProfiles(r.Context(), limit, offset)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"companies": companies,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// handleGetCompany retrieves a company by ID
func (s *Server) handleGetCompany(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	companyID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	company, err := s.db.GetCompanyByID(r.Context(), companyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if company == nil {
		s.errorResponse(w, http.StatusNotFound, "Company not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, company)
}

// handleGetCompanyByName retrieves a company by normalized name
func (s *Server) handleGetCompanyByName(w http.ResponseWriter, r *http.Request) {
	// Changed from path parameter to query parameter to avoid route conflict
	name := r.URL.Query().Get("name")
	if name == "" {
		s.errorResponse(w, http.StatusBadRequest, "Company name is required")
		return
	}

	// Normalize the name
	normalized := db.NormalizeName(name)
	if normalized == "" {
		s.errorResponse(w, http.StatusBadRequest, "Invalid company name")
		return
	}

	company, err := s.db.GetCompanyByNormalizedName(r.Context(), normalized)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if company == nil {
		s.errorResponse(w, http.StatusNotFound, "Company not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, company)
}

// handleListCompanyDomains lists all domains for a company
func (s *Server) handleListCompanyDomains(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	companyID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	domains, err := s.db.ListCompanyDomains(r.Context(), companyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"domains": domains,
		"count":   len(domains),
	})
}
