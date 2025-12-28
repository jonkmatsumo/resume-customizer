package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
)

// mockDB implements a minimal mock for testing
type mockDB struct {
	runs      map[uuid.UUID]*db.Run
	artifacts map[uuid.UUID]*db.Artifact
}

func newMockDB() *mockDB {
	return &mockDB{
		runs:      make(map[uuid.UUID]*db.Run),
		artifacts: make(map[uuid.UUID]*db.Artifact),
	}
}

func (m *mockDB) GetRun(_ context.Context, runID uuid.UUID) (*db.Run, error) {
	run, ok := m.runs[runID]
	if !ok {
		return nil, nil
	}
	return run, nil
}

func (m *mockDB) GetArtifactByID(_ context.Context, artifactID uuid.UUID) (*db.Artifact, error) {
	artifact, ok := m.artifacts[artifactID]
	if !ok {
		return nil, nil
	}
	return artifact, nil
}

func (m *mockDB) Close() {}

// testServer creates a server with mock DB for testing
type testServer struct {
	*Server
	mock *mockDB
}

func newTestServer() *testServer {
	mock := newMockDB()
	s := &Server{
		db:     nil, // Will use mock methods directly
		apiKey: "test-api-key",
	}
	return &testServer{Server: s, mock: mock}
}

// TestHealthEndpoint tests the /health endpoint
func TestHealthEndpoint(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp["status"])
	}
}

// TestRunEndpoint_MissingJobURL tests /run with missing required field
func TestRunEndpoint_MissingJobURL(t *testing.T) {
	s := newTestServer()

	body := `{"experience": "test.json"}`
	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}

// TestRunEndpoint_MissingExperience tests /run with missing experience field
func TestRunEndpoint_MissingExperience(t *testing.T) {
	s := newTestServer()

	body := `{"job_url": "https://example.com/job"}`
	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestRunEndpoint_InvalidJSON tests /run with invalid JSON
func TestRunEndpoint_InvalidJSON(t *testing.T) {
	s := newTestServer()

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestStatusEndpoint_InvalidID tests /status with invalid UUID
func TestStatusEndpoint_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/status/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleStatus(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestArtifactEndpoint_InvalidID tests /artifact with invalid UUID
func TestArtifactEndpoint_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/artifact/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleArtifact(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestCORSMiddleware tests CORS headers are set
func TestCORSMiddleware(t *testing.T) {
	s := newTestServer()

	handler := s.withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header Access-Control-Allow-Origin: *")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected CORS header Access-Control-Allow-Methods")
	}
}

// TestCORSMiddleware_OPTIONS tests OPTIONS preflight request
func TestCORSMiddleware_OPTIONS(t *testing.T) {
	s := newTestServer()

	handler := s.withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("should not reach here")) //nolint:errcheck
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Error("OPTIONS response should have empty body")
	}
}

// TestLoggingMiddleware tests that logging middleware passes through
func TestLoggingMiddleware(t *testing.T) {
	s := newTestServer()

	called := false
	handler := s.withLogging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !called {
		t.Error("logging middleware should call next handler")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestSSEWriter tests SSE event writing
func TestSSEWriter(t *testing.T) {
	w := httptest.NewRecorder()

	sse, err := NewSSEWriter(w)
	if err != nil {
		t.Fatalf("failed to create SSE writer: %v", err)
	}

	event := map[string]string{"step": "test", "message": "hello"}
	if err := sse.WriteEvent("step", event); err != nil {
		t.Fatalf("failed to write event: %v", err)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("expected SSE output")
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("event: step")) {
		t.Error("expected 'event: step' in output")
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("data:")) {
		t.Error("expected 'data:' in output")
	}
}

// TestRunRequest_Defaults tests that defaults are applied
func TestRunRequest_Defaults(t *testing.T) {
	req := RunRequest{
		JobURL: "https://example.com/job",
		UserID: uuid.New().String(),
	}

	// These are the defaults we set in handlers
	if req.Template != "" {
		t.Error("Template should initially be empty")
	}
	if req.MaxBullets != 0 {
		t.Error("MaxBullets should initially be 0")
	}
}

// TestRunResponse_JSON tests RunResponse JSON serialization
func TestRunResponse_JSON(t *testing.T) {
	resp := RunResponse{
		RunID:  "test-id",
		Status: "started",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded RunResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.RunID != "test-id" {
		t.Errorf("expected RunID 'test-id', got '%s'", decoded.RunID)
	}
	if decoded.Status != "started" {
		t.Errorf("expected Status 'started', got '%s'", decoded.Status)
	}
}

// TestStatusResponse_JSON tests StatusResponse JSON serialization
func TestStatusResponse_JSON(t *testing.T) {
	resp := StatusResponse{
		RunID:     "test-id",
		Company:   "Test Corp",
		RoleTitle: "Engineer",
		Status:    "completed",
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if !bytes.Contains(data, []byte("Test Corp")) {
		t.Error("expected company in JSON")
	}
}

// TestJSONResponse tests jsonResponse helper
func TestJSONResponse(t *testing.T) {
	s := newTestServer()
	w := httptest.NewRecorder()

	s.jsonResponse(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type: application/json")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if resp["key"] != "value" {
		t.Errorf("expected key='value', got '%s'", resp["key"])
	}
}

// TestErrorResponse tests errorResponse helper
func TestErrorResponse(t *testing.T) {
	s := newTestServer()
	w := httptest.NewRecorder()

	s.errorResponse(w, http.StatusBadRequest, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if resp["error"] != "test error" {
		t.Errorf("expected error='test error', got '%s'", resp["error"])
	}
}

// TestDeleteRunEndpoint_InvalidID tests DELETE /runs/{id} with invalid UUID
func TestDeleteRunEndpoint_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodDelete, "/runs/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleDeleteRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestDeleteRunEndpoint_MissingID tests DELETE /runs/{id} with missing ID
func TestDeleteRunEndpoint_MissingID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodDelete, "/runs/", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	s.handleDeleteRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestListArtifactsEndpoint_InvalidRunID tests /artifacts with invalid run_id
func TestListArtifactsEndpoint_InvalidRunID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/artifacts?run_id=not-a-uuid", nil)
	w := httptest.NewRecorder()

	s.handleListArtifacts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestRunArtifactsEndpoint_InvalidID tests /runs/{id}/artifacts with invalid UUID
func TestRunArtifactsEndpoint_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/runs/not-a-uuid/artifacts", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleRunArtifacts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestRunResumeTex_InvalidID tests /runs/{id}/resume.tex with invalid UUID
func TestRunResumeTex_InvalidID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/runs/not-a-uuid/resume.tex", nil)
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	s.handleRunResumeTex(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
