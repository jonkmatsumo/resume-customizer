package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleListJobPostings_InvalidCompanyID tests list job postings with invalid company_id
func TestHandleListJobPostings_InvalidCompanyID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/job-postings?company_id=not-a-uuid", nil)
	w := httptest.NewRecorder()

	s.handleListJobPostings(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid company_id")
}

// TestHandleGetJobPosting_InvalidID tests get job posting with invalid UUID
func TestHandleGetJobPosting_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/job-postings/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetJobPosting(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid job posting ID")
}

// TestHandleGetJobPosting_MissingID tests get job posting with missing ID
func TestHandleGetJobPosting_MissingID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/job-postings/", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	s.handleGetJobPosting(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleGetJobPostingByURL_MissingURL tests get job posting by URL with missing url parameter
func TestHandleGetJobPostingByURL_MissingURL(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/job-postings/by-url", nil)
	w := httptest.NewRecorder()

	s.handleGetJobPostingByURL(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "url query parameter is required")
}

// TestHandleListJobPostingsByCompany_InvalidID tests list job postings by company with invalid UUID
func TestHandleListJobPostingsByCompany_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/companies/not-a-uuid/job-postings", nil)
	req.SetPathValue("company_id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleListJobPostingsByCompany(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid company ID")
}
