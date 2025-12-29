package server

import (
	"bytes"
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
