package server

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
)

// ListJobPostingsResponse represents the response for listing job postings
type ListJobPostingsResponse struct {
	Postings []db.JobPosting `json:"postings"`
	Count    int             `json:"count"`
	Limit    int             `json:"limit"`
	Offset   int             `json:"offset"`
}

// handleListJobPostings lists job postings with optional filters and pagination
func (s *Server) handleListJobPostings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := parseQueryInt(r, "limit", 50, 100)
	offset := parseQueryInt(r, "offset", 0, 0)

	opts := db.ListJobPostingsOptions{
		Limit:  limit,
		Offset: offset,
	}

	// Parse optional filters
	if platform := r.URL.Query().Get("platform"); platform != "" {
		opts.Platform = &platform
	}

	if companyIDStr := r.URL.Query().Get("company_id"); companyIDStr != "" {
		companyID, err := uuid.Parse(companyIDStr)
		if err != nil {
			s.errorResponse(w, http.StatusBadRequest, "Invalid company_id")
			return
		}
		opts.CompanyID = &companyID
	}

	postings, total, err := s.db.ListJobPostings(ctx, opts)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, ListJobPostingsResponse{
		Postings: postings,
		Count:    total,
		Limit:    limit,
		Offset:   offset,
	})
}

// handleGetJobPosting retrieves a job posting by its ID
func (s *Server) handleGetJobPosting(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	postingID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid job posting ID")
		return
	}

	posting, err := s.db.GetJobPostingByID(r.Context(), postingID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if posting == nil {
		s.errorResponse(w, http.StatusNotFound, "Job posting not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, posting)
}

// handleGetJobPostingByURL retrieves a job posting by its URL
func (s *Server) handleGetJobPostingByURL(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		s.errorResponse(w, http.StatusBadRequest, "url query parameter is required")
		return
	}

	posting, err := s.db.GetJobPostingByURL(r.Context(), url)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if posting == nil {
		s.errorResponse(w, http.StatusNotFound, "Job posting not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, posting)
}

// handleListJobPostingsByCompany lists all job postings for a company
func (s *Server) handleListJobPostingsByCompany(w http.ResponseWriter, r *http.Request) {
	companyIDStr := r.PathValue("company_id")
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	postings, err := s.db.ListJobPostingsByCompany(r.Context(), companyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"postings": postings,
		"count":    len(postings),
	})
}
