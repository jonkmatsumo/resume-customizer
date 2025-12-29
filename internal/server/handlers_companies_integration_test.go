package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompaniesEndpoints_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	ctx := context.Background()

	// Create test company
	company, err := s.db.FindOrCreateCompany(ctx, "Integration Test Company")
	require.NoError(t, err)
	require.NotNil(t, company)

	// Cleanup
	defer func() {
		// Note: Companies are not deleted as they may be referenced by other data
		// In a real test, we'd use a test database that gets cleaned up
	}()

	// Test 1: Get company by ID
	t.Run("GetCompanyByID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/companies/"+company.ID.String(), nil)
		req.SetPathValue("id", company.ID.String())
		w := httptest.NewRecorder()

		s.handleGetCompany(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp db.Company
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, company.ID, resp.ID)
		assert.Equal(t, company.Name, resp.Name)
	})

	// Test 2: Get company by name
	t.Run("GetCompanyByName", func(t *testing.T) {
		// URL-encode the company name for the path
		encodedName := url.PathEscape(company.Name)
		req := httptest.NewRequest(http.MethodGet, "/companies/by-name/"+encodedName, nil)
		req.SetPathValue("name", company.Name)
		w := httptest.NewRecorder()

		s.handleGetCompanyByName(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp db.Company
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, company.ID, resp.ID)
	})

	// Test 3: List company domains
	t.Run("ListCompanyDomains", func(t *testing.T) {
		// Add a test domain
		testDomain := "integration-test.example.com"
		err := s.db.AddCompanyDomain(ctx, company.ID, testDomain, db.DomainTypePrimary)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/companies/"+company.ID.String()+"/domains", nil)
		req.SetPathValue("id", company.ID.String())
		w := httptest.NewRecorder()

		s.handleListCompanyDomains(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "domains")
		assert.Contains(t, resp, "count")
	})

	// Test 4: List companies (requires a profile)
	t.Run("ListCompanies", func(t *testing.T) {
		// Create a profile for the company so it appears in the list
		profileInput := &db.ProfileCreateInput{
			CompanyID:     company.ID,
			Tone:          "professional",
			DomainContext: "test",
		}
		_, err := s.db.CreateCompanyProfile(ctx, profileInput)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/companies?limit=10&offset=0", nil)
		w := httptest.NewRecorder()

		s.handleListCompanies(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "companies")
		assert.Contains(t, resp, "total")
		assert.Contains(t, resp, "limit")
		assert.Contains(t, resp, "offset")

		companies, ok := resp["companies"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(companies), 1)
	})

	// Test 5: Get company not found
	t.Run("GetCompanyNotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/companies/"+nonExistentID.String(), nil)
		req.SetPathValue("id", nonExistentID.String())
		w := httptest.NewRecorder()

		s.handleGetCompany(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "not found")
	})
}

func TestListCompanies_Pagination_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	ctx := context.Background()

	// Create multiple companies with profiles
	companyNames := []string{"Pagination Test 1", "Pagination Test 2", "Pagination Test 3"}

	for _, name := range companyNames {
		company, err := s.db.FindOrCreateCompany(ctx, name)
		require.NoError(t, err)
		require.NotNil(t, company)

		// Create profile so company appears in list
		profileInput := &db.ProfileCreateInput{
			CompanyID:     company.ID,
			Tone:          "professional",
			DomainContext: "test",
		}
		_, err = s.db.CreateCompanyProfile(ctx, profileInput)
		require.NoError(t, err)
	}

	// Test pagination
	t.Run("Pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/companies?limit=2&offset=0", nil)
		w := httptest.NewRecorder()

		s.handleListCompanies(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		limit, ok := resp["limit"].(float64)
		require.True(t, ok)
		assert.Equal(t, float64(2), limit)

		companies, ok := resp["companies"].([]any)
		require.True(t, ok)
		assert.LessOrEqual(t, len(companies), 2)
	})
}
