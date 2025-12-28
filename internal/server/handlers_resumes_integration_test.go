package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/stretchr/testify/require"
)

func TestResumeRun_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()
	ctx := context.Background()

	// 1. Setup Data: User, Job, Experience
	uid, _ := s.db.CreateUser(ctx, "Resume User", "resume-"+uuid.New().String()+"@test.com", "123")
	defer s.db.DeleteUser(ctx, uid)

	jid, _ := s.db.CreateJob(ctx, &db.Job{UserID: uid, Company: "Resume Corp", RoleTitle: "Engineer"})
	_, _ = s.db.CreateExperience(ctx, &db.Experience{JobID: jid, BulletText: "Did stuff", Skills: []string{"Go"}})

	// 2. Test Run Request with UserID
	runBody := map[string]string{
		"user_id": uid.String(),
		"job_url": "https://example.com",
	}
	bodyBytes, _ := json.Marshal(runBody)
	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleRun(w, req)

	// Since pipeline runs in background, we expect 202 Accepted
	require.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.NotEmpty(t, resp["run_id"])
	require.Equal(t, "started", resp["status"])
}

func TestResumeRun_MissingUserID(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	// Test Run Request WITHOUT UserID (should fail after our changes)
	runBody := map[string]string{
		"job_url": "https://example.com",
		// "experience": "path/to/file.json" -- simulated legacy request
	}
	bodyBytes, _ := json.Marshal(runBody)
	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleRun(w, req)

	// Expect 400 Bad Request
	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.Contains(t, resp["error"], "user_id is required")
}
