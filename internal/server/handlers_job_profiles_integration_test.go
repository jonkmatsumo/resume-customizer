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

func TestJobProfilesEndpoints_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	ctx := context.Background()

	// Create test company
	company, err := s.db.FindOrCreateCompany(ctx, "Job Profiles Test Company")
	require.NoError(t, err)
	require.NotNil(t, company)

	// Create test job posting
	postingInput := &db.JobPostingCreateInput{
		URL:         "https://example.com/job-profiles-test",
		RoleTitle:   "Test Engineer",
		Platform:    db.PlatformGreenhouse,
		CleanedText: "Test job description",
		CompanyID:   &company.ID,
	}
	posting, err := s.db.UpsertJobPosting(ctx, postingInput)
	require.NoError(t, err)
	require.NotNil(t, posting)

	// Create test job profile
	profileInput := &db.JobProfileCreateInput{
		PostingID:        posting.ID,
		CompanyName:      company.Name,
		RoleTitle:        "Test Engineer",
		Responsibilities: []string{"Write tests", "Review code"},
		HardRequirements: []db.RequirementInput{
			{Skill: "Go", Evidence: "Required in job description"},
		},
		NiceToHaves: []db.RequirementInput{
			{Skill: "Python", Evidence: "Nice to have"},
		},
		Keywords: []string{"testing", "go", "backend"},
	}
	profile, err := s.db.CreateJobProfile(ctx, profileInput)
	require.NoError(t, err)
	require.NotNil(t, profile)

	// Cleanup
	defer func() {
		// Note: In a real test, we'd use a test database that gets cleaned up
	}()

	// Test 1: Get job profile by ID
	t.Run("GetJobProfileByID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job-profiles/"+profile.ID.String(), nil)
		req.SetPathValue("id", profile.ID.String())
		w := httptest.NewRecorder()

		s.handleGetJobProfile(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp db.JobProfile
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, profile.ID, resp.ID)
		assert.Equal(t, profile.RoleTitle, resp.RoleTitle)
		assert.Len(t, resp.Responsibilities, 2)
	})

	// Test 2: Get job profile by posting ID
	t.Run("GetJobProfileByPostingID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job-postings/"+posting.ID.String()+"/profile", nil)
		req.SetPathValue("posting_id", posting.ID.String())
		w := httptest.NewRecorder()

		s.handleGetJobProfileByPostingID(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp db.JobProfile
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, profile.ID, resp.ID)
		assert.Equal(t, profile.RoleTitle, resp.RoleTitle)
	})

	// Test 3: Get requirements
	t.Run("GetRequirements", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job-profiles/"+profile.ID.String()+"/requirements", nil)
		req.SetPathValue("id", profile.ID.String())
		w := httptest.NewRecorder()

		s.handleGetRequirements(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "requirements")
		assert.Contains(t, resp, "count")

		requirements, ok := resp["requirements"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(requirements), 2) // At least hard + nice-to-have
	})

	// Test 4: Get responsibilities
	t.Run("GetResponsibilities", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job-profiles/"+profile.ID.String()+"/responsibilities", nil)
		req.SetPathValue("id", profile.ID.String())
		w := httptest.NewRecorder()

		s.handleGetResponsibilities(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "responsibilities")
		assert.Contains(t, resp, "count")

		responsibilities, ok := resp["responsibilities"].([]any)
		require.True(t, ok)
		assert.Equal(t, 2, len(responsibilities))
	})

	// Test 5: Get keywords
	t.Run("GetKeywords", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/job-profiles/"+profile.ID.String()+"/keywords", nil)
		req.SetPathValue("id", profile.ID.String())
		w := httptest.NewRecorder()

		s.handleGetKeywords(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "keywords")
		assert.Contains(t, resp, "count")

		keywords, ok := resp["keywords"].([]any)
		require.True(t, ok)
		assert.Equal(t, 3, len(keywords))
	})

	// Test 6: Get job profile not found
	t.Run("GetJobProfileNotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/job-profiles/"+nonExistentID.String(), nil)
		req.SetPathValue("id", nonExistentID.String())
		w := httptest.NewRecorder()

		s.handleGetJobProfile(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "not found")
	})
}
