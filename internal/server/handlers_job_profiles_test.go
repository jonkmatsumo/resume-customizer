package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleGetJobProfile_InvalidID tests get job profile with invalid UUID
func TestHandleGetJobProfile_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/job-profiles/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetJobProfile(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid job profile ID")
}

// TestHandleGetJobProfile_MissingID tests get job profile with missing ID
func TestHandleGetJobProfile_MissingID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/job-profiles/", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	s.handleGetJobProfile(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleGetJobProfileByPostingID_InvalidID tests get job profile by posting ID with invalid UUID
func TestHandleGetJobProfileByPostingID_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/job-postings/not-a-uuid/profile", nil)
	req.SetPathValue("posting_id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetJobProfileByPostingID(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid posting ID")
}

// TestHandleGetRequirements_InvalidID tests get requirements with invalid UUID
func TestHandleGetRequirements_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/job-profiles/not-a-uuid/requirements", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetRequirements(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid job profile ID")
}

// TestHandleGetResponsibilities_InvalidID tests get responsibilities with invalid UUID
func TestHandleGetResponsibilities_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/job-profiles/not-a-uuid/responsibilities", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetResponsibilities(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid job profile ID")
}

// TestHandleGetKeywords_InvalidID tests get keywords with invalid UUID
func TestHandleGetKeywords_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/job-profiles/not-a-uuid/keywords", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetKeywords(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid job profile ID")
}
