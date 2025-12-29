package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleGetCrawledPage_InvalidID tests get crawled page with invalid UUID
func TestHandleGetCrawledPage_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/crawled-pages/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetCrawledPage(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid crawled page ID")
}

// TestHandleGetCrawledPage_MissingID tests get crawled page with missing ID
func TestHandleGetCrawledPage_MissingID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/crawled-pages/", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	s.handleGetCrawledPage(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleGetCrawledPageByURL_MissingURL tests get crawled page by URL with missing url parameter
func TestHandleGetCrawledPageByURL_MissingURL(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/crawled-pages/by-url", nil)
	w := httptest.NewRecorder()

	s.handleGetCrawledPageByURL(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "url query parameter is required")
}

// TestHandleListCrawledPagesByCompany_InvalidID tests list crawled pages by company with invalid UUID
func TestHandleListCrawledPagesByCompany_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/companies/not-a-uuid/crawled-pages", nil)
	req.SetPathValue("company_id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleListCrawledPagesByCompany(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid company ID")
}
