package server

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
)

// CrawledPageResponse represents a crawled page response (without raw_html by default)
type CrawledPageResponse struct {
	ID                 uuid.UUID  `json:"id"`
	CompanyID          *uuid.UUID `json:"company_id,omitempty"`
	URL                string     `json:"url"`
	PageType           *string    `json:"page_type,omitempty"`
	ParsedText         *string    `json:"parsed_text,omitempty"`
	ContentHash        *string    `json:"content_hash,omitempty"`
	HTTPStatus         *int       `json:"http_status,omitempty"`
	FetchStatus        string     `json:"fetch_status"`
	ErrorMessage       *string    `json:"error_message,omitempty"`
	IsPermanentFailure bool       `json:"is_permanent_failure"`
	RetryCount         int        `json:"retry_count"`
	RetryAfter         *string    `json:"retry_after,omitempty"` // ISO 8601 string
	FetchedAt          string     `json:"fetched_at"`            // ISO 8601 string
	ExpiresAt          *string    `json:"expires_at,omitempty"`  // ISO 8601 string
	LastAccessedAt     string     `json:"last_accessed_at"`      // ISO 8601 string
	CreatedAt          string     `json:"created_at"`            // ISO 8601 string
	UpdatedAt          string     `json:"updated_at"`            // ISO 8601 string
	RawHTML            *string    `json:"raw_html,omitempty"`    // Only included if include_html=true
}

// handleGetCrawledPage retrieves a crawled page by its ID
func (s *Server) handleGetCrawledPage(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	pageID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid crawled page ID")
		return
	}

	// Check if HTML should be included
	includeHTML := r.URL.Query().Get("include_html") == "true"

	page, err := s.db.GetCrawledPageByID(r.Context(), pageID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if page == nil {
		s.errorResponse(w, http.StatusNotFound, "Crawled page not found")
		return
	}

	// Convert to response model
	response := convertCrawledPageToResponse(page, includeHTML)
	s.jsonResponse(w, http.StatusOK, response)
}

// handleGetCrawledPageByURL retrieves a crawled page by its URL
func (s *Server) handleGetCrawledPageByURL(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		s.errorResponse(w, http.StatusBadRequest, "url query parameter is required")
		return
	}

	// Check if HTML should be included
	includeHTML := r.URL.Query().Get("include_html") == "true"

	page, err := s.db.GetCrawledPageByURL(r.Context(), url)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if page == nil {
		s.errorResponse(w, http.StatusNotFound, "Crawled page not found")
		return
	}

	// Convert to response model
	response := convertCrawledPageToResponse(page, includeHTML)
	s.jsonResponse(w, http.StatusOK, response)
}

// handleListCrawledPagesByCompany lists all crawled pages for a company
func (s *Server) handleListCrawledPagesByCompany(w http.ResponseWriter, r *http.Request) {
	companyIDStr := r.PathValue("company_id")
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	pages, err := s.db.ListCrawledPagesByCompany(r.Context(), companyID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	// Convert to response models (never include HTML in list responses)
	responses := make([]CrawledPageResponse, len(pages))
	for i, page := range pages {
		responses[i] = convertCrawledPageToResponse(&page, false)
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"pages": responses,
		"count": len(responses),
	})
}

// convertCrawledPageToResponse converts a db.CrawledPage to CrawledPageResponse
func convertCrawledPageToResponse(page *db.CrawledPage, includeHTML bool) CrawledPageResponse {
	response := CrawledPageResponse{
		ID:                 page.ID,
		CompanyID:          page.CompanyID,
		URL:                page.URL,
		PageType:           page.PageType,
		ParsedText:         page.ParsedText,
		ContentHash:        page.ContentHash,
		HTTPStatus:         page.HTTPStatus,
		FetchStatus:        page.FetchStatus,
		ErrorMessage:       page.ErrorMessage,
		IsPermanentFailure: page.IsPermanentFailure,
		RetryCount:         page.RetryCount,
		FetchedAt:          page.FetchedAt.Format(time.RFC3339),
		LastAccessedAt:     page.LastAccessedAt.Format(time.RFC3339),
		CreatedAt:          page.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          page.UpdatedAt.Format(time.RFC3339),
	}

	if page.ExpiresAt != nil {
		expiresAt := page.ExpiresAt.Format(time.RFC3339)
		response.ExpiresAt = &expiresAt
	}

	if page.RetryAfter != nil {
		retryAfter := page.RetryAfter.Format(time.RFC3339)
		response.RetryAfter = &retryAfter
	}

	// Only include raw_html if explicitly requested
	if includeHTML {
		response.RawHTML = page.RawHTML
	}

	return response
}
