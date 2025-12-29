package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleListCompanies tests the list companies endpoint
func TestHandleListCompanies(t *testing.T) {
	// This test requires a real database connection
	// For unit tests, we'll test error handling with invalid inputs
	t.Skip("Requires database connection - covered in integration tests")
}

// TestHandleGetCompany_InvalidID tests get company with invalid UUID
func TestHandleGetCompany_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/companies/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetCompany(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid company ID")
}

// TestHandleGetCompany_MissingID tests get company with missing ID
func TestHandleGetCompany_MissingID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/companies/", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	s.handleGetCompany(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleGetCompanyByName_EmptyName tests get company by name with empty name
func TestHandleGetCompanyByName_EmptyName(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/companies/by-name/", nil)
	req.SetPathValue("name", "")
	w := httptest.NewRecorder()

	s.handleGetCompanyByName(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Company name is required")
}

// TestHandleListCompanyDomains_InvalidID tests list domains with invalid UUID
func TestHandleListCompanyDomains_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/companies/not-a-uuid/domains", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleListCompanyDomains(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid company ID")
}

// TestParseQueryInt tests the parseQueryInt helper function
func TestParseQueryInt(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		key          string
		defaultValue int
		maxValue     int
		want         int
	}{
		{
			name:         "valid value",
			query:        "?limit=25",
			key:          "limit",
			defaultValue: 50,
			maxValue:     100,
			want:         25,
		},
		{
			name:         "missing value uses default",
			query:        "?offset=10",
			key:          "limit",
			defaultValue: 50,
			maxValue:     100,
			want:         50,
		},
		{
			name:         "value exceeds max",
			query:        "?limit=200",
			key:          "limit",
			defaultValue: 50,
			maxValue:     100,
			want:         100,
		},
		{
			name:         "invalid value uses default",
			query:        "?limit=abc",
			key:          "limit",
			defaultValue: 50,
			maxValue:     100,
			want:         50,
		},
		{
			name:         "negative value uses default",
			query:        "?limit=-10",
			key:          "limit",
			defaultValue: 50,
			maxValue:     100,
			want:         50,
		},
		{
			name:         "zero value",
			query:        "?offset=0",
			key:          "offset",
			defaultValue: 0,
			maxValue:     0,
			want:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/companies"+tt.query, nil)
			got := parseQueryInt(req, tt.key, tt.defaultValue, tt.maxValue)
			assert.Equal(t, tt.want, got)
		})
	}
}
