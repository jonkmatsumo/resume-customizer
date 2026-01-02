package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleCreateRun_MissingUserID(t *testing.T) {
	// Use integration test server setup instead of unit test setup
	// since we need a real database connection
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	req := RunCreateRequest{
		JobURL: "https://example.com/job",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/runs", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleCreateRun(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "user_id is required", resp["error"])
}

func TestHandleCreateRun_InvalidUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	req := RunCreateRequest{
		UserID: "not-a-uuid",
		JobURL: "https://example.com/job",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/runs", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleCreateRun(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Invalid user_id format", resp["error"])
}

func TestHandleCreateRun_MissingJobInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	ctx := httptest.NewRequest(http.MethodPost, "/", nil).Context()
	// Create user first with unique email
	uniqueEmail := "test-" + uuid.New().String() + "@example.com"
	userID, err := s.db.CreateUser(ctx, "Test User", uniqueEmail, "123")
	require.NoError(t, err)

	req := RunCreateRequest{
		UserID: userID.String(),
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/runs", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleCreateRun(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleGetStepStatus_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	runID := uuid.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/runs/"+runID.String()+"/steps/test_step", nil)
	httpReq.SetPathValue("run_id", runID.String())
	httpReq.SetPathValue("step_name", "test_step")
	w := httptest.NewRecorder()

	s.handleGetStepStatus(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleListRunSteps_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	runID := uuid.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/runs/"+runID.String()+"/steps", nil)
	httpReq.SetPathValue("run_id", runID.String())
	w := httptest.NewRecorder()

	s.handleListRunSteps(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleGetCheckpoint_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	runID := uuid.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/runs/"+runID.String()+"/checkpoint", nil)
	httpReq.SetPathValue("run_id", runID.String())
	w := httptest.NewRecorder()

	s.handleGetCheckpoint(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSkipStep_UnknownStep(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	ctx := httptest.NewRequest(http.MethodPost, "/", nil).Context()
	// Create run first
	runID, err := s.db.CreateRun(ctx, "Test", "", "")
	require.NoError(t, err)

	httpReq := httptest.NewRequest(http.MethodPost, "/runs/"+runID.String()+"/steps/unknown_step/skip", nil)
	httpReq.SetPathValue("run_id", runID.String())
	httpReq.SetPathValue("step_name", "unknown_step")
	w := httptest.NewRecorder()

	s.handleSkipStep(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleRetryStep_NotFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	ctx := httptest.NewRequest(http.MethodPost, "/", nil).Context()
	// Create run first
	runID, err := s.db.CreateRun(ctx, "Test", "", "")
	require.NoError(t, err)

	httpReq := httptest.NewRequest(http.MethodPost, "/runs/"+runID.String()+"/steps/test_step/retry", nil)
	httpReq.SetPathValue("run_id", runID.String())
	httpReq.SetPathValue("step_name", "test_step")
	w := httptest.NewRecorder()

	s.handleRetryStep(w, httpReq)

	// Should fail because step doesn't exist or isn't in failed state
	assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusBadRequest)
}

// TestRunStepsListResponse_JSON_Marshaling tests JSON marshaling of RunStepsListResponse
func TestRunStepsListResponse_JSON_Marshaling(t *testing.T) {
	t.Run("WithAllFields", func(t *testing.T) {
		company := "Test Corp"
		roleTitle := "Software Engineer"
		createdAt := "2024-01-01T10:00:00Z"

		resp := RunStepsListResponse{
			RunID:     "123e4567-e89b-12d3-a456-426614174000",
			Status:    "completed",
			Company:   &company,
			RoleTitle: &roleTitle,
			CreatedAt: createdAt,
			Steps:     []StepStatusResponse{},
			Summary: RunStepsSummary{
				Total:     1,
				Completed: 1,
			},
		}

		jsonBytes, err := json.Marshal(resp)
		require.NoError(t, err)

		var unmarshaled RunStepsListResponse
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, resp.RunID, unmarshaled.RunID)
		assert.Equal(t, resp.Status, unmarshaled.Status)
		assert.NotNil(t, unmarshaled.Company)
		assert.Equal(t, company, *unmarshaled.Company)
		assert.NotNil(t, unmarshaled.RoleTitle)
		assert.Equal(t, roleTitle, *unmarshaled.RoleTitle)
		assert.Equal(t, createdAt, unmarshaled.CreatedAt)
	})

	t.Run("WithNullFields", func(t *testing.T) {
		createdAt := "2024-01-01T10:00:00Z"

		resp := RunStepsListResponse{
			RunID:     "123e4567-e89b-12d3-a456-426614174000",
			Status:    "running",
			Company:   nil,
			RoleTitle: nil,
			CreatedAt: createdAt,
			Steps:     []StepStatusResponse{},
			Summary:   RunStepsSummary{Total: 0},
		}

		jsonBytes, err := json.Marshal(resp)
		require.NoError(t, err)

		// Verify omitempty behavior - null fields should not appear in JSON
		jsonStr := string(jsonBytes)
		assert.NotContains(t, jsonStr, "company")
		assert.NotContains(t, jsonStr, "role_title")

		var unmarshaled RunStepsListResponse
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		require.NoError(t, err)

		assert.Nil(t, unmarshaled.Company)
		assert.Nil(t, unmarshaled.RoleTitle)
		assert.Equal(t, createdAt, unmarshaled.CreatedAt)
	})

	t.Run("WithPartialFields", func(t *testing.T) {
		company := "Test Corp"
		createdAt := "2024-01-01T10:00:00Z"

		resp := RunStepsListResponse{
			RunID:     "123e4567-e89b-12d3-a456-426614174000",
			Status:    "queued",
			Company:   &company,
			RoleTitle: nil, // Empty role_title
			CreatedAt: createdAt,
			Steps:     []StepStatusResponse{},
			Summary:   RunStepsSummary{Total: 0},
		}

		jsonBytes, err := json.Marshal(resp)
		require.NoError(t, err)

		jsonStr := string(jsonBytes)
		assert.Contains(t, jsonStr, "company")
		assert.Contains(t, jsonStr, "Test Corp")
		assert.NotContains(t, jsonStr, "role_title") // Should be omitted

		var unmarshaled RunStepsListResponse
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		require.NoError(t, err)

		assert.NotNil(t, unmarshaled.Company)
		assert.Equal(t, company, *unmarshaled.Company)
		assert.Nil(t, unmarshaled.RoleTitle)
	})
}

// TestHandleListRunSteps_Integration_WithMetadata tests the handler with metadata fields
func TestHandleListRunSteps_Integration_WithMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	s := setupIntegrationTestServer(t)
	defer s.db.Close()
	ctx := context.Background()

	// Create a run with company and role_title
	runID, err := s.db.CreateRun(ctx, "Test Corp", "Software Engineer", "https://example.com/job")
	require.NoError(t, err)

	httpReq := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String()+"/steps", nil)
	httpReq.SetPathValue("run_id", runID.String())
	w := httptest.NewRecorder()

	s.handleListRunSteps(w, httpReq)

	require.Equal(t, http.StatusOK, w.Code)

	var resp RunStepsListResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, runID.String(), resp.RunID)
	assert.Equal(t, "running", resp.Status)
	assert.NotNil(t, resp.Company)
	assert.Equal(t, "Test Corp", *resp.Company)
	assert.NotNil(t, resp.RoleTitle)
	assert.Equal(t, "Software Engineer", *resp.RoleTitle)
	assert.NotEmpty(t, resp.CreatedAt)
	// Verify existing fields still work
	assert.NotNil(t, resp.Steps)
	assert.NotNil(t, resp.Summary)
}

// TestHandleListRunSteps_Integration_EmptyFields tests the handler with empty company/role_title
func TestHandleListRunSteps_Integration_EmptyFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	s := setupIntegrationTestServer(t)
	defer s.db.Close()
	ctx := context.Background()

	// Create a run with empty company and role_title
	runID, err := s.db.CreateRun(ctx, "", "", "https://example.com/job")
	require.NoError(t, err)

	httpReq := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String()+"/steps", nil)
	httpReq.SetPathValue("run_id", runID.String())
	w := httptest.NewRecorder()

	s.handleListRunSteps(w, httpReq)

	require.Equal(t, http.StatusOK, w.Code)

	var resp RunStepsListResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, runID.String(), resp.RunID)
	assert.Equal(t, "running", resp.Status)
	assert.Nil(t, resp.Company)   // Should be null due to omitempty
	assert.Nil(t, resp.RoleTitle) // Should be null due to omitempty
	assert.NotEmpty(t, resp.CreatedAt)
	// Verify existing fields still work (backward compatibility)
	assert.NotNil(t, resp.Steps)
	assert.NotNil(t, resp.Summary)
}

// TestHandleListRunSteps_BackwardCompatibility tests that existing fields still work
func TestHandleListRunSteps_BackwardCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	s := setupIntegrationTestServer(t)
	defer s.db.Close()
	ctx := context.Background()

	runID, err := s.db.CreateRun(ctx, "Test Corp", "Engineer", "https://example.com/job")
	require.NoError(t, err)

	httpReq := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String()+"/steps", nil)
	httpReq.SetPathValue("run_id", runID.String())
	w := httptest.NewRecorder()

	s.handleListRunSteps(w, httpReq)

	require.Equal(t, http.StatusOK, w.Code)

	var resp RunStepsListResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Verify all existing required fields are present
	assert.Equal(t, runID.String(), resp.RunID)
	assert.Equal(t, "running", resp.Status)
	assert.NotNil(t, resp.Steps)
	assert.NotNil(t, resp.Summary)
	assert.GreaterOrEqual(t, resp.Summary.Total, 0) // At least Total should be present
}
