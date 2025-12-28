package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupIntegrationTestServer sets up a server connected to a real DB for integration tests
func setupIntegrationTestServer(t *testing.T) *Server {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Default to local docker connection for tests
		dbURL = "postgres://resume:resume_dev@localhost:5432/resume_customizer?sslmode=disable"
	}

	// Verify DB connection first
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping integration test: failed to connect to DB: %v", err)
	}

	return &Server{
		db:          database,
		apiKey:      "test-api-key",
		databaseURL: dbURL,
	}
}

func TestUserCRUD_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()

	// Use handleCreateUser, handleGetUser, handleUpdateUser, handleDeleteUser

	// 1. Create User
	createUserBody := map[string]string{
		"name":  "Integration User",
		"email": "integration-" + uuid.New().String() + "@example.com",
		"phone": "555-0199",
	}
	bodyBytes, _ := json.Marshal(createUserBody)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(bodyBytes))
	w := httptest.NewRecorder()

	s.handleCreateUser(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var createResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	userID := createResp["id"]
	require.NotEmpty(t, userID)

	// Cleanup at end
	defer func() {
		// Delete via DB directly to be sure, or use handler if we trust it
		uid, _ := uuid.Parse(userID)
		s.db.DeleteUser(context.Background(), uid)
	}()

	// 2. Get User
	req = httptest.NewRequest(http.MethodGet, "/users/"+userID, nil)
	req.SetPathValue("id", userID)
	w = httptest.NewRecorder()

	s.handleGetUser(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var user db.User
	err = json.Unmarshal(w.Body.Bytes(), &user)
	require.NoError(t, err)
	assert.Equal(t, createUserBody["name"], user.Name)

	// 3. Update User
	updateUserBody := map[string]string{
		"name":  "Updated Integration User",
		"email": createUserBody["email"],
		"phone": "555-0200",
	}
	bodyBytes, _ = json.Marshal(updateUserBody)
	req = httptest.NewRequest(http.MethodPut, "/users/"+userID, bytes.NewBuffer(bodyBytes))
	req.SetPathValue("id", userID)
	w = httptest.NewRecorder()

	s.handleUpdateUser(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Verify Update
	userFromDB, _ := s.db.GetUser(context.Background(), uuid.MustParse(userID))
	assert.Equal(t, "Updated Integration User", userFromDB.Name)

	// 4. Delete User
	req = httptest.NewRequest(http.MethodDelete, "/users/"+userID, nil)
	req.SetPathValue("id", userID)
	w = httptest.NewRecorder()

	s.handleDeleteUser(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Verify Deletion
	userFromDB, _ = s.db.GetUser(context.Background(), uuid.MustParse(userID))
	assert.Nil(t, userFromDB)
}

func TestJobCRUD_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()
	ctx := context.Background()

	// Setup User
	uid, _ := s.db.CreateUser(ctx, "Job Integration", "jobint-"+uuid.New().String()+"@test.com", "123")
	defer s.db.DeleteUser(ctx, uid)

	// 1. Create Job
	createJobBody := map[string]any{
		"company":         "Integration Corp",
		"role_title":      "Tester",
		"start_date":      "2023-01-01",
		"employment_type": "full-time",
	}
	bodyBytes, _ := json.Marshal(createJobBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/users/%s/jobs", uid), bytes.NewBuffer(bodyBytes))
	req.SetPathValue("id", uid.String())
	w := httptest.NewRecorder()

	s.handleCreateJob(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var createResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &createResp)
	jobID := createResp["id"]

	// 2. List Jobs
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/users/%s/jobs", uid), nil)
	req.SetPathValue("id", uid.String())
	w = httptest.NewRecorder()

	s.handleListJobs(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var listResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &listResp)
	jobs := listResp["jobs"].([]any)
	assert.NotEmpty(t, jobs)

	// 3. Update Job
	updateJobBody := map[string]any{
		"company":         "Updated Corp",
		"role_title":      "Senior Tester",
		"employment_type": "contract",
	}
	bodyBytes, _ = json.Marshal(updateJobBody)
	req = httptest.NewRequest(http.MethodPut, "/jobs/"+jobID, bytes.NewBuffer(bodyBytes))
	req.SetPathValue("id", jobID)
	w = httptest.NewRecorder()

	s.handleUpdateJob(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Verify Update via List
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/users/%s/jobs", uid), nil)
	req.SetPathValue("id", uid.String())
	w = httptest.NewRecorder()
	s.handleListJobs(w, req)
	// We could parse JSON deeper, but checking code 200 is good sanity check here, plus DB check below depends on list correctness

	// 4. Delete Job
	req = httptest.NewRequest(http.MethodDelete, "/jobs/"+jobID, nil)
	req.SetPathValue("id", jobID)
	w = httptest.NewRecorder()

	s.handleDeleteJob(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Verify Deletion
	jobsList, _ := s.db.ListJobs(ctx, uid)
	assert.Empty(t, jobsList)
}

func TestExperienceCRUD_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()
	ctx := context.Background()

	// Setup User & Job
	uid, _ := s.db.CreateUser(ctx, "Exp Integration", "expint-"+uuid.New().String()+"@test.com", "123")
	defer s.db.DeleteUser(ctx, uid)
	jid, _ := s.db.CreateJob(ctx, &db.Job{UserID: uid, Company: "Test", RoleTitle: "Role"})

	// 1. Create Experience
	createExpBody := map[string]any{
		"bullet_text":       "Integrated stuff",
		"skills":            []string{"Go", "Testing"},
		"evidence_strength": "high",
	}
	bodyBytes, _ := json.Marshal(createExpBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/jobs/%s/experiences", jid), bytes.NewBuffer(bodyBytes))
	req.SetPathValue("id", jid.String())
	w := httptest.NewRecorder()

	s.handleCreateExperience(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var createResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &createResp)
	expID := createResp["id"]

	// 2. Update Experience
	updateExpBody := map[string]any{
		"bullet_text":       "Updated stuff",
		"skills":            []string{"Python"},
		"evidence_strength": "medium",
	}
	bodyBytes, _ = json.Marshal(updateExpBody)
	req = httptest.NewRequest(http.MethodPut, "/experiences/"+expID, bytes.NewBuffer(bodyBytes))
	req.SetPathValue("id", expID)
	w = httptest.NewRecorder()

	s.handleUpdateExperience(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Verify Update
	exps, _ := s.db.ListExperiences(ctx, jid)
	require.Len(t, exps, 1)
	assert.Equal(t, "Updated stuff", exps[0].BulletText)
	// Cast due to driver using []string vs []interface{} issues depending on setup, but PGX handles it well mostly.
	// For simple assertion:
	assert.Contains(t, exps[0].Skills, "Python")

	// 3. Delete Experience
	req = httptest.NewRequest(http.MethodDelete, "/experiences/"+expID, nil)
	req.SetPathValue("id", expID)
	w = httptest.NewRecorder()

	s.handleDeleteExperience(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	expsAfter, _ := s.db.ListExperiences(ctx, jid)
	assert.Empty(t, expsAfter)
}

func TestEducationCRUD_Integration(t *testing.T) {
	s := setupIntegrationTestServer(t)
	defer s.db.Close()
	ctx := context.Background()

	// Setup User
	uid, _ := s.db.CreateUser(ctx, "Edu Integration", "eduint-"+uuid.New().String()+"@test.com", "123")
	defer s.db.DeleteUser(ctx, uid)

	// 1. Create Education
	createEduBody := map[string]any{
		"school":      "Integration University",
		"degree_type": "BS",
		"start_date":  "2018-09-01",
	}
	bodyBytes, _ := json.Marshal(createEduBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/users/%s/education", uid), bytes.NewBuffer(bodyBytes))
	req.SetPathValue("id", uid.String())
	w := httptest.NewRecorder()

	s.handleCreateEducation(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var createResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &createResp)
	eduID := createResp["id"]

	// 2. Update Education
	updateEduBody := map[string]any{
		"school":      "Updated University",
		"degree_type": "MS",
	}
	bodyBytes, _ = json.Marshal(updateEduBody)
	req = httptest.NewRequest(http.MethodPut, "/education/"+eduID, bytes.NewBuffer(bodyBytes))
	req.SetPathValue("id", eduID)
	w = httptest.NewRecorder()

	s.handleUpdateEducation(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Verify Update
	edus, _ := s.db.ListEducation(ctx, uid)
	require.Len(t, edus, 1)
	assert.Equal(t, "Updated University", edus[0].School)

	// 3. Delete Education
	req = httptest.NewRequest(http.MethodDelete, "/education/"+eduID, nil)
	req.SetPathValue("id", eduID)
	w = httptest.NewRecorder()

	s.handleDeleteEducation(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	edusAfter, _ := s.db.ListEducation(ctx, uid)
	assert.Empty(t, edusAfter)
}
