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

func TestJobPostingsEndpoints_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	ctx := context.Background()

	// Create test company
	company, err := s.db.FindOrCreateCompany(ctx, "Job Postings Test Company")
	require.NoError(t, err)
	require.NotNil(t, company)

	// Create test job posting
	postingInput := &db.JobPostingCreateInput{
		URL:         "https://example.com/job-postings-test",
		RoleTitle:   "Test Engineer",
		Platform:    db.PlatformGreenhouse,
		CleanedText: "Test job description",
		CompanyID:   &company.ID,
	}
	posting, err := s.db.UpsertJobPosting(ctx, postingInput)
	require.NoError(t, err)
	require.NotNil(t, posting)

	// Cleanup
	defer func() {
		// Note: In a real test, we'd use a test database that gets cleaned up
	}()

	// Test 1: Get job posting by ID
	t.Run("GetJobPostingByID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job-postings/"+posting.ID.String(), nil)
		req.SetPathValue("id", posting.ID.String())
		w := httptest.NewRecorder()

		s.handleGetJobPosting(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp db.JobPosting
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, posting.ID, resp.ID)
		assert.Equal(t, posting.URL, resp.URL)
	})

	// Test 2: Get job posting by URL
	t.Run("GetJobPostingByURL", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job-postings/by-url?url="+posting.URL, nil)
		w := httptest.NewRecorder()

		s.handleGetJobPostingByURL(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp db.JobPosting
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, posting.ID, resp.ID)
		assert.Equal(t, posting.URL, resp.URL)
	})

	// Test 3: List job postings by company
	t.Run("ListJobPostingsByCompany", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/companies/"+company.ID.String()+"/job-postings", nil)
		req.SetPathValue("company_id", company.ID.String())
		w := httptest.NewRecorder()

		s.handleListJobPostingsByCompany(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "postings")
		assert.Contains(t, resp, "count")

		postings, ok := resp["postings"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(postings), 1)
	})

	// Test 4: List job postings with filters
	t.Run("ListJobPostings", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job-postings?limit=10&offset=0&platform=greenhouse", nil)
		w := httptest.NewRecorder()

		s.handleListJobPostings(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp ListJobPostingsResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		// Check that posting is in the list
		found := false
		for _, p := range resp.Postings {
			if p.ID == posting.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected posting to be in the list")
		assert.GreaterOrEqual(t, resp.Count, 1)
	})

	// Test 5: Get job posting not found
	t.Run("GetJobPostingNotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/job-postings/"+nonExistentID.String(), nil)
		req.SetPathValue("id", nonExistentID.String())
		w := httptest.NewRecorder()

		s.handleGetJobPosting(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "not found")
	})
}

func TestListJobPostings_Pagination_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	ctx := context.Background()

	// Create test company
	company, err := s.db.FindOrCreateCompany(ctx, "Pagination Test Company")
	require.NoError(t, err)
	require.NotNil(t, company)

	// Create multiple job postings
	for i := 0; i < 5; i++ {
		postingInput := &db.JobPostingCreateInput{
			URL:         "https://example.com/job-pagination-" + string(rune('0'+i)),
			RoleTitle:   "Test Engineer " + string(rune('0'+i)),
			Platform:    db.PlatformGreenhouse,
			CleanedText: "Test job description",
			CompanyID:   &company.ID,
		}
		_, err := s.db.UpsertJobPosting(ctx, postingInput)
		require.NoError(t, err)
	}

	// Test pagination
	t.Run("Pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job-postings?limit=2&offset=0", nil)
		w := httptest.NewRecorder()

		s.handleListJobPostings(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp ListJobPostingsResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(resp.Postings), 2)
		assert.Equal(t, 2, resp.Limit)
		assert.Equal(t, 0, resp.Offset)
	})
}
