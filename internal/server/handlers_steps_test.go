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
	s := setupTestServer(t)
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
	s := setupTestServer(t)
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
	s := setupTestServer(t)
	defer s.db.Close()

	userID := uuid.New()
	// Create user first
	_, err := s.db.CreateUser(httptest.NewRequest(http.MethodPost, "/", nil).Context(), "Test User", "test@example.com", "123")
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
	s := setupTestServer(t)
	defer s.db.Close()

	runID := uuid.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/runs/"+runID.String()+"/steps/test_step", nil)
	w := httptest.NewRecorder()

	s.handleGetStepStatus(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleListRunSteps_NotFound(t *testing.T) {
	s := setupTestServer(t)
	defer s.db.Close()

	runID := uuid.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/runs/"+runID.String()+"/steps", nil)
	w := httptest.NewRecorder()

	s.handleListRunSteps(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleGetCheckpoint_NotFound(t *testing.T) {
	s := setupTestServer(t)
	defer s.db.Close()

	runID := uuid.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/runs/"+runID.String()+"/checkpoint", nil)
	w := httptest.NewRecorder()

	s.handleGetCheckpoint(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSkipStep_UnknownStep(t *testing.T) {
	s := setupTestServer(t)
	defer s.db.Close()

	runID := uuid.New()
	// Create run first
	_, err := s.db.CreateRun(httptest.NewRequest(http.MethodPost, "/", nil).Context(), "Test", "", "")
	require.NoError(t, err)

	httpReq := httptest.NewRequest(http.MethodPost, "/runs/"+runID.String()+"/steps/unknown_step/skip", nil)
	w := httptest.NewRecorder()

	s.handleSkipStep(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleRetryStep_NotFailed(t *testing.T) {
	s := setupTestServer(t)
	defer s.db.Close()

	runID := uuid.New()
	// Create run first
	_, err := s.db.CreateRun(httptest.NewRequest(http.MethodPost, "/", nil).Context(), "Test", "", "")
	require.NoError(t, err)

	httpReq := httptest.NewRequest(http.MethodPost, "/runs/"+runID.String()+"/steps/test_step/retry", nil)
	w := httptest.NewRecorder()

	s.handleRetryStep(w, httpReq)

	// Should fail because step doesn't exist or isn't in failed state
	assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusBadRequest)
}

// setupTestServer creates a test server with a test database
func setupTestServer(t *testing.T) *Server {
	// Use a test database URL - in real tests this would be a test database
	// For now, we'll skip tests that require database
	if testing.Short() {
		t.Skip("Skipping test that requires database")
	}

	// This is a placeholder - real integration tests would set up a test database
	return nil
}
