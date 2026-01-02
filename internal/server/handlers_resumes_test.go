package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleV1Status_Success tests successful response
func TestHandleV1Status_Success(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()
	now := time.Now()
	completedAt := now.Add(5 * time.Minute)

	s.mock.runs[runID] = &db.Run{
		ID:          runID,
		Company:     "Test Corp",
		RoleTitle:   "Software Engineer",
		Status:      "completed",
		CreatedAt:   now,
		CompletedAt: &completedAt,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/status/"+runID.String(), nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleV1Status(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp RunStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, runID.String(), resp.ID)
	assert.NotNil(t, resp.Company)
	assert.Equal(t, "Test Corp", *resp.Company)
	assert.NotNil(t, resp.Role)
	assert.Equal(t, "Software Engineer", *resp.Role)
	assert.Equal(t, "completed", resp.Status)
	assert.Equal(t, now.Format(time.RFC3339), resp.CreatedAt)
	assert.Equal(t, completedAt.Format(time.RFC3339), resp.UpdatedAt)
	assert.Nil(t, resp.Message)
}

// TestHandleV1Status_NoCompletedAt tests response when completed_at is null
func TestHandleV1Status_NoCompletedAt(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()
	now := time.Now()

	s.mock.runs[runID] = &db.Run{
		ID:          runID,
		Company:     "Test Corp",
		RoleTitle:   "Engineer",
		Status:      "running",
		CreatedAt:   now,
		CompletedAt: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/status/"+runID.String(), nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleV1Status(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp RunStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, now.Format(time.RFC3339), resp.UpdatedAt) // Should use created_at
}

// TestHandleV1Status_NullableFields tests null company and role handling
func TestHandleV1Status_NullableFields(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()

	s.mock.runs[runID] = &db.Run{
		ID:          runID,
		Company:     "", // Empty string
		RoleTitle:   "", // Empty string
		Status:      "queued",
		CreatedAt:   time.Now(),
		CompletedAt: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/status/"+runID.String(), nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleV1Status(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp RunStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Nil(t, resp.Company) // Should be null in JSON
	assert.Nil(t, resp.Role)    // Should be null in JSON
}

// TestHandleV1Status_MissingID tests missing ID parameter
func TestHandleV1Status_MissingID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/status/", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	s.handleV1Status(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleV1Status_InvalidUUID tests invalid UUID format
func TestHandleV1Status_InvalidUUID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/status/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleV1Status(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"], "Invalid run ID format")
}

// TestHandleV1Status_NotFound tests run not found
func TestHandleV1Status_NotFound(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/v1/status/"+runID.String(), nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleV1Status(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errorResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"], "Run not found")
}

// TestHandleV1Status_DatabaseError tests database error handling
func TestHandleV1Status_DatabaseError(t *testing.T) {
	// TODO: Database error testing requires Server.db to be an interface
	// For now, skip this test as the test infrastructure doesn't support mocking
	// when Server.db is a concrete *db.DB type
	t.Skip("Database error testing requires interface-based Server.db")
}

// TestRunStatusResponse_JSON tests JSON serialization
func TestRunStatusResponse_JSON(t *testing.T) {
	company := "Test Corp"
	role := "Engineer"
	now := time.Now().Format(time.RFC3339)

	resp := RunStatusResponse{
		ID:        uuid.New().String(),
		Company:   &company,
		Role:      &role,
		Status:    "completed",
		CreatedAt: now,
		UpdatedAt: now,
		Message:   nil,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Test Corp")
	assert.Contains(t, string(data), "Engineer")
	assert.Contains(t, string(data), "completed")
}

// TestRunStatusResponse_JSON_NullFields tests JSON with null fields
func TestRunStatusResponse_JSON_NullFields(t *testing.T) {
	now := time.Now().Format(time.RFC3339)

	resp := RunStatusResponse{
		ID:        uuid.New().String(),
		Company:   nil,
		Role:      nil,
		Status:    "queued",
		CreatedAt: now,
		UpdatedAt: now,
		Message:   nil,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	// Verify null fields are omitted or null
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Company and role should be omitted (not present in JSON)
	_, hasCompany := result["company"]
	_, hasRole := result["role"]
	// With omitempty, they should be omitted when nil
	assert.False(t, hasCompany)
	assert.False(t, hasRole)
}

// TestHandleGetRun_Success tests successful response
func TestHandleGetRun_Success(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	completedAt := now.Add(5 * time.Minute)

	s.mock.runs[runID] = &db.Run{
		ID:          runID,
		UserID:      &userID,
		Company:     "Test Corp",
		RoleTitle:   "Software Engineer",
		JobURL:      "https://example.com/job",
		Status:      "completed",
		CreatedAt:   now,
		CompletedAt: &completedAt,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String(), nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleGetRun(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp RunGetResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, runID.String(), resp.ID)
	assert.NotNil(t, resp.UserID)
	assert.Equal(t, userID.String(), *resp.UserID)
	assert.Equal(t, "Test Corp", resp.Company)
	assert.Equal(t, "Software Engineer", resp.RoleTitle)
	assert.Equal(t, "https://example.com/job", resp.JobURL)
	assert.Equal(t, "completed", resp.Status)
	assert.Equal(t, now.Format(time.RFC3339), resp.CreatedAt)
	assert.NotNil(t, resp.CompletedAt)
	assert.Equal(t, completedAt.Format(time.RFC3339), *resp.CompletedAt)
}

// TestHandleGetRun_NoCompletedAt tests response when completed_at is null
func TestHandleGetRun_NoCompletedAt(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()
	now := time.Now()

	s.mock.runs[runID] = &db.Run{
		ID:          runID,
		UserID:      nil,
		Company:     "Test Corp",
		RoleTitle:   "Engineer",
		JobURL:      "https://example.com/job",
		Status:      "running",
		CreatedAt:   now,
		CompletedAt: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String(), nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleGetRun(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp RunGetResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Nil(t, resp.UserID)
	assert.Nil(t, resp.CompletedAt)
}

// TestHandleGetRun_NullableFields tests null user_id and completed_at handling
func TestHandleGetRun_NullableFields(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()

	s.mock.runs[runID] = &db.Run{
		ID:          runID,
		UserID:      nil,
		Company:     "Test Corp",
		RoleTitle:   "Engineer",
		JobURL:      "https://example.com/job",
		Status:      "queued",
		CreatedAt:   time.Now(),
		CompletedAt: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String(), nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleGetRun(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp RunGetResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Nil(t, resp.UserID)
	assert.Nil(t, resp.CompletedAt)
}

// TestHandleGetRun_MissingID tests missing ID parameter
func TestHandleGetRun_MissingID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	s.handleGetRun(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleGetRun_InvalidUUID tests invalid UUID format
func TestHandleGetRun_InvalidUUID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleGetRun(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"], "Invalid run ID format")
}

// TestHandleGetRun_NotFound tests run not found
func TestHandleGetRun_NotFound(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String(), nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleGetRun(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errorResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"], "Run not found")
}

// TestHandleGetRun_DatabaseError tests database error handling
func TestHandleGetRun_DatabaseError(t *testing.T) {
	// TODO: Database error testing requires Server.db to be an interface
	// For now, skip this test as the test infrastructure doesn't support mocking
	// when Server.db is a concrete *db.DB type
	t.Skip("Database error testing requires interface-based Server.db")
}

// TestRunGetResponse_JSON tests JSON serialization
func TestRunGetResponse_JSON(t *testing.T) {
	userID := uuid.New().String()
	completedAt := time.Now().Format(time.RFC3339)

	resp := RunGetResponse{
		ID:          uuid.New().String(),
		UserID:      &userID,
		Company:     "Test Corp",
		RoleTitle:   "Engineer",
		JobURL:      "https://example.com/job",
		Status:      "completed",
		CreatedAt:   time.Now().Format(time.RFC3339),
		CompletedAt: &completedAt,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Test Corp")
	assert.Contains(t, string(data), "Engineer")
	assert.Contains(t, string(data), "completed")
	assert.Contains(t, string(data), userID)
}

// TestRunGetResponse_JSON_NullFields tests JSON with null fields
func TestRunGetResponse_JSON_NullFields(t *testing.T) {
	resp := RunGetResponse{
		ID:          uuid.New().String(),
		UserID:      nil,
		Company:     "Test Corp",
		RoleTitle:   "Engineer",
		JobURL:      "https://example.com/job",
		Status:      "queued",
		CreatedAt:   time.Now().Format(time.RFC3339),
		CompletedAt: nil,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	// Verify null fields are omitted
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// user_id and completed_at should be omitted (not present in JSON)
	_, hasUserID := result["user_id"]
	_, hasCompletedAt := result["completed_at"]
	// With omitempty, they should be omitted when nil
	assert.False(t, hasUserID)
	assert.False(t, hasCompletedAt)
}

// TestHandleRunResumeTex_ViewMode_True tests view=true query parameter (no attachment header)
func TestHandleRunResumeTex_ViewMode_True(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()
	texContent := "\\documentclass{article}\n\\begin{document}\nHello World\n\\end{document}"

	// Setup mock text artifact
	key := runID.String() + ":resume_tex"
	s.mock.textArtifacts[key] = texContent

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String()+"/resume.tex?view=true", nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleRunResumeTex(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Empty(t, w.Header().Get("Content-Disposition"), "Should not have Content-Disposition header when view=true")
	assert.Equal(t, texContent, w.Body.String())
}

// TestHandleRunResumeTex_ViewMode_Default tests default behavior (with attachment header)
func TestHandleRunResumeTex_ViewMode_Default(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()
	texContent := "\\documentclass{article}\n\\begin{document}\nHello World\n\\end{document}"

	key := runID.String() + ":resume_tex"
	s.mock.textArtifacts[key] = texContent

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String()+"/resume.tex", nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleRunResumeTex(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "attachment; filename=resume.tex", w.Header().Get("Content-Disposition"))
	assert.Equal(t, texContent, w.Body.String())
}

// TestHandleRunResumeTex_ViewMode_False tests view=false query parameter (should behave like default)
func TestHandleRunResumeTex_ViewMode_False(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()
	texContent := "\\documentclass{article}\n\\begin{document}\nHello World\n\\end{document}"

	key := runID.String() + ":resume_tex"
	s.mock.textArtifacts[key] = texContent

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String()+"/resume.tex?view=false", nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleRunResumeTex(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "attachment; filename=resume.tex", w.Header().Get("Content-Disposition"), "Should have attachment header when view=false")
	assert.Equal(t, texContent, w.Body.String())
}

// TestHandleRunResumeTex_ViewMode_OtherValues tests other view parameter values (should behave like default)
func TestHandleRunResumeTex_ViewMode_OtherValues(t *testing.T) {
	testCases := []string{"1", "yes", "YES", "True", "on", ""}

	for _, viewValue := range testCases {
		t.Run("view="+viewValue, func(t *testing.T) {
			s := newTestServer()
			runID := uuid.New()
			texContent := "\\documentclass{article}\n\\begin{document}\nHello World\n\\end{document}"

			key := runID.String() + ":resume_tex"
			s.mock.textArtifacts[key] = texContent

			req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String()+"/resume.tex?view="+viewValue, nil)
			req.SetPathValue("id", runID.String())
			w := httptest.NewRecorder()

			s.handleRunResumeTex(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "attachment; filename=resume.tex", w.Header().Get("Content-Disposition"),
				"Should have attachment header when view=%s", viewValue)
		})
	}
}

// TestHandleRunResumeTex_ViewMode_CaseSensitive tests that view parameter is case-sensitive
func TestHandleRunResumeTex_ViewMode_CaseSensitive(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()
	texContent := "\\documentclass{article}\n\\begin{document}\nHello World\n\\end{document}"

	key := runID.String() + ":resume_tex"
	s.mock.textArtifacts[key] = texContent

	// Test with "True" (capital T) - should NOT be treated as view mode
	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String()+"/resume.tex?view=True", nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleRunResumeTex(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "attachment; filename=resume.tex", w.Header().Get("Content-Disposition"),
		"Should have attachment header when view=True (case-sensitive)")
}

// TestHandleRunResumeTex_ViewMode_MultipleQueryParams tests behavior with multiple query params
func TestHandleRunResumeTex_ViewMode_MultipleQueryParams(t *testing.T) {
	s := newTestServer()
	runID := uuid.New()
	texContent := "\\documentclass{article}\n\\begin{document}\nHello World\n\\end{document}"

	key := runID.String() + ":resume_tex"
	s.mock.textArtifacts[key] = texContent

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String()+"/resume.tex?view=true&other=param", nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleRunResumeTex(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Content-Disposition"), "Should not have Content-Disposition header when view=true")
	assert.Equal(t, texContent, w.Body.String())
}
