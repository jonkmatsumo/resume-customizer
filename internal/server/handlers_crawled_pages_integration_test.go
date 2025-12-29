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

func TestCrawledPagesEndpoints_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	ctx := context.Background()

	// Create test company
	company, err := s.db.FindOrCreateCompany(ctx, "Crawled Pages Test Company")
	require.NoError(t, err)
	require.NotNil(t, company)

	// Create test crawled page
	htmlContent := "<html><body>Test content</body></html>"
	pageType := db.PageTypeValues
	parsedText := "Test content"
	page := &db.CrawledPage{
		CompanyID:   &company.ID,
		URL:         "https://example.com/test-page",
		PageType:    &pageType,
		RawHTML:     &htmlContent,
		ParsedText:  &parsedText,
		FetchStatus: db.FetchStatusSuccess,
	}
	err = s.db.UpsertCrawledPage(ctx, page)
	require.NoError(t, err)
	require.NotNil(t, page.ID)

	// Cleanup
	defer func() {
		// Note: In a real test, we'd use a test database that gets cleaned up
	}()

	// Test 1: Get crawled page by ID (without HTML)
	t.Run("GetCrawledPageByID_WithoutHTML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/crawled-pages/"+page.ID.String(), nil)
		req.SetPathValue("id", page.ID.String())
		w := httptest.NewRecorder()

		s.handleGetCrawledPage(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp CrawledPageResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, page.ID, resp.ID)
		assert.Equal(t, page.URL, resp.URL)
		assert.Nil(t, resp.RawHTML, "raw_html should be nil by default")
	})

	// Test 2: Get crawled page by ID (with HTML)
	t.Run("GetCrawledPageByID_WithHTML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/crawled-pages/"+page.ID.String()+"?include_html=true", nil)
		req.SetPathValue("id", page.ID.String())
		w := httptest.NewRecorder()

		s.handleGetCrawledPage(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp CrawledPageResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, page.ID, resp.ID)
		assert.NotNil(t, resp.RawHTML, "raw_html should be included when include_html=true")
		assert.Equal(t, htmlContent, *resp.RawHTML)
	})

	// Test 3: Get crawled page by URL
	t.Run("GetCrawledPageByURL", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/crawled-pages/by-url?url="+page.URL, nil)
		w := httptest.NewRecorder()

		s.handleGetCrawledPageByURL(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp CrawledPageResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, page.ID, resp.ID)
		assert.Equal(t, page.URL, resp.URL)
	})

	// Test 4: List crawled pages by company
	t.Run("ListCrawledPagesByCompany", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/companies/"+company.ID.String()+"/crawled-pages", nil)
		req.SetPathValue("company_id", company.ID.String())
		w := httptest.NewRecorder()

		s.handleListCrawledPagesByCompany(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "pages")
		assert.Contains(t, resp, "count")

		pages, ok := resp["pages"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(pages), 1)

		// Verify HTML is not included in list responses
		if len(pages) > 0 {
			firstPage, ok := pages[0].(map[string]any)
			require.True(t, ok)
			_, hasHTML := firstPage["raw_html"]
			assert.False(t, hasHTML, "raw_html should not be in list responses")
		}
	})

	// Test 5: Get crawled page not found
	t.Run("GetCrawledPageNotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/crawled-pages/"+nonExistentID.String(), nil)
		req.SetPathValue("id", nonExistentID.String())
		w := httptest.NewRecorder()

		s.handleGetCrawledPage(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "not found")
	})
}
