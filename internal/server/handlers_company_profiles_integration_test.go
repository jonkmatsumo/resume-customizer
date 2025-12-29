package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompanyProfilesEndpoints_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	ctx := context.Background()

	// Create test company
	company, err := s.db.FindOrCreateCompany(ctx, "Profile Integration Test Company")
	require.NoError(t, err)
	require.NotNil(t, company)

	// Create profile with sub-resources
	profileInput := &db.ProfileCreateInput{
		CompanyID:     company.ID,
		Tone:          "professional and innovative",
		DomainContext: "FinTech, consumer finance",
		StyleRules: []string{
			"Use active voice",
			"Quantify achievements",
		},
		TabooPhrases: []db.TabooPhraseInput{
			{Phrase: "synergy", Reason: "overused corporate jargon"},
		},
		Values: []string{
			"Customer-first",
			"Transparency",
		},
		EvidenceURLs: []db.ProfileSourceInput{
			{URL: "https://example.com/values"},
			{URL: "https://example.com/culture"},
		},
	}

	profile, err := s.db.CreateCompanyProfile(ctx, profileInput)
	require.NoError(t, err)
	require.NotNil(t, profile)

	// Test 1: Get company profile
	t.Run("GetCompanyProfile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/companies/"+company.ID.String()+"/profile", nil)
		req.SetPathValue("company_id", company.ID.String())
		w := httptest.NewRecorder()

		s.handleGetCompanyProfile(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp db.CompanyProfile
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, profile.ID, resp.ID)
		assert.Equal(t, profile.Tone, resp.Tone)
		assert.NotEmpty(t, resp.StyleRules)
		assert.NotEmpty(t, resp.TabooPhrases)
		assert.NotEmpty(t, resp.Values)
		assert.NotEmpty(t, resp.EvidenceURLs)
	})

	// Test 2: Get style rules
	t.Run("GetStyleRules", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/companies/"+company.ID.String()+"/profile/style-rules", nil)
		req.SetPathValue("company_id", company.ID.String())
		w := httptest.NewRecorder()

		s.handleGetStyleRules(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "style_rules")
		assert.Contains(t, resp, "count")

		rules, ok := resp["style_rules"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(rules), 2)
	})

	// Test 3: Get taboo phrases
	t.Run("GetTabooPhrases", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/companies/"+company.ID.String()+"/profile/taboo-phrases", nil)
		req.SetPathValue("company_id", company.ID.String())
		w := httptest.NewRecorder()

		s.handleGetTabooPhrases(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "taboo_phrases")
		assert.Contains(t, resp, "count")

		phrases, ok := resp["taboo_phrases"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(phrases), 1)
	})

	// Test 4: Get values
	t.Run("GetValues", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/companies/"+company.ID.String()+"/profile/values", nil)
		req.SetPathValue("company_id", company.ID.String())
		w := httptest.NewRecorder()

		s.handleGetValues(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "values")
		assert.Contains(t, resp, "count")

		values, ok := resp["values"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(values), 2)
	})

	// Test 5: Get sources
	t.Run("GetSources", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/companies/"+company.ID.String()+"/profile/sources", nil)
		req.SetPathValue("company_id", company.ID.String())
		w := httptest.NewRecorder()

		s.handleGetSources(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "sources")
		assert.Contains(t, resp, "count")

		sources, ok := resp["sources"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(sources), 2)
	})

	// Test 6: Get profile not found
	t.Run("GetProfileNotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/companies/"+nonExistentID.String()+"/profile", nil)
		req.SetPathValue("company_id", nonExistentID.String())
		w := httptest.NewRecorder()

		s.handleGetCompanyProfile(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "not found")
	})

	// Test 7: Get sub-resource when profile doesn't exist
	t.Run("GetStyleRulesProfileNotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/companies/"+nonExistentID.String()+"/profile/style-rules", nil)
		req.SetPathValue("company_id", nonExistentID.String())
		w := httptest.NewRecorder()

		s.handleGetStyleRules(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "not found")
	})
}
