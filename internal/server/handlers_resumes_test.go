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

	s.db.(*mockDB).runs[runID] = &db.Run{
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

	s.db.(*mockDB).runs[runID] = &db.Run{
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

	s.db.(*mockDB).runs[runID] = &db.Run{
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
	s := newTestServer()
	runID := uuid.New()

	// Create a mock DB that returns an error
	errorDB := &errorMockDB{}
	s.db = errorDB

	req := httptest.NewRequest(http.MethodGet, "/v1/status/"+runID.String(), nil)
	req.SetPathValue("id", runID.String())
	w := httptest.NewRecorder()

	s.handleV1Status(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
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
