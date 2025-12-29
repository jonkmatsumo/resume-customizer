package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleGetCompanyProfile_InvalidID tests get profile with invalid UUID
func TestHandleGetCompanyProfile_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/companies/not-a-uuid/profile", nil)
	req.SetPathValue("company_id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetCompanyProfile(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid company ID")
}

// TestHandleGetStyleRules_InvalidID tests get style rules with invalid UUID
func TestHandleGetStyleRules_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/companies/not-a-uuid/profile/style-rules", nil)
	req.SetPathValue("company_id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetStyleRules(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid company ID")
}

// TestHandleGetTabooPhrases_InvalidID tests get taboo phrases with invalid UUID
func TestHandleGetTabooPhrases_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/companies/not-a-uuid/profile/taboo-phrases", nil)
	req.SetPathValue("company_id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetTabooPhrases(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid company ID")
}

// TestHandleGetValues_InvalidID tests get values with invalid UUID
func TestHandleGetValues_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/companies/not-a-uuid/profile/values", nil)
	req.SetPathValue("company_id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetValues(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid company ID")
}

// TestHandleGetSources_InvalidID tests get sources with invalid UUID
func TestHandleGetSources_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/companies/not-a-uuid/profile/sources", nil)
	req.SetPathValue("company_id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetSources(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid company ID")
}
