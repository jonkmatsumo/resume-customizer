package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/server/ratelimit"
	"github.com/jonathan/resume-customizer/internal/types"
)

// mockDB implements a minimal mock for testing
type mockDB struct {
	runs          map[uuid.UUID]*db.Run
	artifacts     map[uuid.UUID]*db.Artifact
	textArtifacts map[string]string // key: "runID:step", value: text content
}

func newMockDB() *mockDB {
	return &mockDB{
		runs:          make(map[uuid.UUID]*db.Run),
		artifacts:     make(map[uuid.UUID]*db.Artifact),
		textArtifacts: make(map[string]string),
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

func (m *mockDB) GetTextArtifact(_ context.Context, runID uuid.UUID, step string) (string, error) {
	key := runID.String() + ":" + step
	content, ok := m.textArtifacts[key]
	if !ok {
		return "", nil // Return empty string when not found (matches real behavior)
	}
	return content, nil
}

func (m *mockDB) SaveTextArtifact(_ context.Context, runID uuid.UUID, step, _ string, text string) error {
	key := runID.String() + ":" + step
	if m.textArtifacts == nil {
		m.textArtifacts = make(map[string]string)
	}
	m.textArtifacts[key] = text
	return nil
}

func (m *mockDB) Close() {}

// Stub implementations for all other DBClient interface methods
// These return zero values or errors as appropriate for unit tests

func (m *mockDB) CreateRun(_ context.Context, _, _, _ string) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockDB) ListRunsFiltered(_ context.Context, _ db.RunFilters) ([]db.Run, error) {
	return []db.Run{}, nil
}

func (m *mockDB) DeleteRun(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockDB) ListArtifacts(_ context.Context, _ db.ArtifactFilters) ([]db.ArtifactSummary, error) {
	return []db.ArtifactSummary{}, nil
}

func (m *mockDB) GetRunStep(_ context.Context, _ uuid.UUID, _ string) (*db.RunStep, error) {
	return nil, nil
}

func (m *mockDB) ListRunSteps(_ context.Context, _ uuid.UUID, _, _ *string) ([]db.RunStep, error) {
	return []db.RunStep{}, nil
}

func (m *mockDB) CreateRunStep(_ context.Context, _ uuid.UUID, _ *db.RunStepInput) (*db.RunStep, error) {
	return nil, nil
}

func (m *mockDB) UpdateRunStepStatus(_ context.Context, _ uuid.UUID, _ string, _ string, _ *string, _ *uuid.UUID) error {
	return nil
}

func (m *mockDB) GetRunCheckpoint(_ context.Context, _ uuid.UUID) (*db.RunCheckpoint, error) {
	return nil, nil
}

func (m *mockDB) CreateRunCheckpoint(_ context.Context, _ uuid.UUID, _ *db.RunCheckpointInput) (*db.RunCheckpoint, error) {
	return nil, nil
}

func (m *mockDB) GetUser(_ context.Context, _ uuid.UUID) (*db.User, error) {
	return nil, nil
}

func (m *mockDB) GetUserByEmail(_ context.Context, _ string) (*db.User, error) {
	return nil, nil
}

func (m *mockDB) CreateUser(_ context.Context, _, _, _ string) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockDB) UpdateUser(_ context.Context, _ *db.User) error {
	return nil
}

func (m *mockDB) DeleteUser(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockDB) UpdatePassword(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockDB) CheckEmailExists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (m *mockDB) CreateJob(_ context.Context, _ *db.Job) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockDB) ListJobs(_ context.Context, _ uuid.UUID) ([]db.Job, error) {
	return []db.Job{}, nil
}

func (m *mockDB) UpdateJob(_ context.Context, _ *db.Job) error {
	return nil
}

func (m *mockDB) DeleteJob(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockDB) CreateExperience(_ context.Context, _ *db.Experience) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockDB) ListExperiences(_ context.Context, _ uuid.UUID) ([]db.Experience, error) {
	return []db.Experience{}, nil
}

func (m *mockDB) UpdateExperience(_ context.Context, _ *db.Experience) error {
	return nil
}

func (m *mockDB) DeleteExperience(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockDB) CreateEducation(_ context.Context, _ *db.Education) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockDB) ListEducation(_ context.Context, _ uuid.UUID) ([]db.Education, error) {
	return []db.Education{}, nil
}

func (m *mockDB) UpdateEducation(_ context.Context, _ *db.Education) error {
	return nil
}

func (m *mockDB) DeleteEducation(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockDB) ListCompaniesWithProfiles(_ context.Context, _, _ int) ([]db.Company, int, error) {
	return []db.Company{}, 0, nil
}

func (m *mockDB) GetCompanyByID(_ context.Context, _ uuid.UUID) (*db.Company, error) {
	return nil, nil
}

func (m *mockDB) GetCompanyByNormalizedName(_ context.Context, _ string) (*db.Company, error) {
	return nil, nil
}

func (m *mockDB) ListCompanyDomains(_ context.Context, _ uuid.UUID) ([]db.CompanyDomain, error) {
	return []db.CompanyDomain{}, nil
}

func (m *mockDB) FindOrCreateCompany(_ context.Context, _ string) (*db.Company, error) {
	return nil, nil
}

func (m *mockDB) AddCompanyDomain(_ context.Context, _ uuid.UUID, _ string, _ string) error {
	return nil
}

func (m *mockDB) GetCompanyProfileByCompanyID(_ context.Context, _ uuid.UUID) (*db.CompanyProfile, error) {
	return nil, nil
}

func (m *mockDB) CreateCompanyProfile(_ context.Context, _ *db.ProfileCreateInput) (*db.CompanyProfile, error) {
	return nil, nil
}

func (m *mockDB) GetStyleRulesByProfileID(_ context.Context, _ uuid.UUID) ([]db.CompanyStyleRule, error) {
	return []db.CompanyStyleRule{}, nil
}

func (m *mockDB) GetTabooPhrasesByProfileID(_ context.Context, _ uuid.UUID) ([]db.CompanyTabooPhrase, error) {
	return []db.CompanyTabooPhrase{}, nil
}

func (m *mockDB) GetValuesByProfileID(_ context.Context, _ uuid.UUID) ([]db.CompanyValue, error) {
	return []db.CompanyValue{}, nil
}

func (m *mockDB) GetSourcesByProfileID(_ context.Context, _ uuid.UUID) ([]db.CompanyProfileSource, error) {
	return []db.CompanyProfileSource{}, nil
}

func (m *mockDB) ListJobPostings(_ context.Context, _ db.ListJobPostingsOptions) ([]db.JobPosting, int, error) {
	return []db.JobPosting{}, 0, nil
}

func (m *mockDB) GetJobPostingByID(_ context.Context, _ uuid.UUID) (*db.JobPosting, error) {
	return nil, nil
}

func (m *mockDB) GetJobPostingByURL(_ context.Context, _ string) (*db.JobPosting, error) {
	return nil, nil
}

func (m *mockDB) ListJobPostingsByCompany(_ context.Context, _ uuid.UUID) ([]db.JobPosting, error) {
	return []db.JobPosting{}, nil
}

func (m *mockDB) UpsertJobPosting(_ context.Context, _ *db.JobPostingCreateInput) (*db.JobPosting, error) {
	return nil, nil
}

func (m *mockDB) GetJobProfileByID(_ context.Context, _ uuid.UUID) (*db.JobProfile, error) {
	return nil, nil
}

func (m *mockDB) GetJobProfileByPostingID(_ context.Context, _ uuid.UUID) (*db.JobProfile, error) {
	return nil, nil
}

func (m *mockDB) GetRequirementsByProfileID(_ context.Context, _ uuid.UUID) ([]db.JobRequirement, error) {
	return []db.JobRequirement{}, nil
}

func (m *mockDB) GetResponsibilitiesByProfileID(_ context.Context, _ uuid.UUID) ([]db.JobResponsibility, error) {
	return []db.JobResponsibility{}, nil
}

func (m *mockDB) GetKeywordsByProfileID(_ context.Context, _ uuid.UUID) ([]db.JobKeyword, error) {
	return []db.JobKeyword{}, nil
}

func (m *mockDB) CreateJobProfile(_ context.Context, _ *db.JobProfileCreateInput) (*db.JobProfile, error) {
	return nil, nil
}

func (m *mockDB) ListStoriesByUser(_ context.Context, _ uuid.UUID) ([]db.Story, error) {
	return []db.Story{}, nil
}

func (m *mockDB) GetStoryByID(_ context.Context, _ uuid.UUID) (*db.Story, error) {
	return nil, nil
}

func (m *mockDB) CreateStory(_ context.Context, _ *db.StoryCreateInput) (*db.Story, error) {
	return nil, nil
}

func (m *mockDB) GetBulletsByStoryID(_ context.Context, _ uuid.UUID) ([]db.Bullet, error) {
	return []db.Bullet{}, nil
}

func (m *mockDB) ListSkillsByUserID(_ context.Context, _ uuid.UUID) ([]db.Skill, error) {
	return []db.Skill{}, nil
}

func (m *mockDB) GetSkillByName(_ context.Context, _ string) (*db.Skill, error) {
	return nil, nil
}

func (m *mockDB) GetBulletsBySkillIDAndUserID(_ context.Context, _, _ uuid.UUID) ([]db.Bullet, error) {
	return []db.Bullet{}, nil
}

func (m *mockDB) GetCrawledPageByID(_ context.Context, _ uuid.UUID) (*db.CrawledPage, error) {
	return nil, nil
}

func (m *mockDB) GetCrawledPageByURL(_ context.Context, _ string) (*db.CrawledPage, error) {
	return nil, nil
}

func (m *mockDB) ListCrawledPagesByCompany(_ context.Context, _ uuid.UUID) ([]db.CrawledPage, error) {
	return []db.CrawledPage{}, nil
}

func (m *mockDB) UpsertCrawledPage(_ context.Context, _ *db.CrawledPage) error {
	return nil
}

func (m *mockDB) GetExperienceBank(_ context.Context, _ uuid.UUID) (*types.ExperienceBank, error) {
	return nil, nil
}

func (m *mockDB) Pool() *pgxpool.Pool {
	return nil // Unit tests don't use Pool()
}

// errorMockDB returns errors for testing error paths
// TODO: Use this in error path tests when needed
//nolint:unused // Reserved for future error path testing
type errorMockDB struct {
	mockDB
}

//nolint:unused // Reserved for future error path testing
func (m *errorMockDB) GetRun(_ context.Context, _ uuid.UUID) (*db.Run, error) {
	return nil, fmt.Errorf("database connection failed")
}

// testServer creates a server with mock DB for testing
type testServer struct {
	*Server
	mock *mockDB
}

func newTestServer() *testServer {
	mock := newMockDB()
	// Create rate limiter with test config (disabled by default for most tests)
	rateLimitConfig := &ratelimit.Config{
		Enabled:       false,
		DefaultLimit:  1000,
		DefaultWindow: time.Minute,
	}
	s := &Server{
		db:          mock,
		apiKey:      "test-api-key",
		rateLimiter: ratelimit.NewLimiter(rateLimitConfig),
	}
	return &testServer{Server: s, mock: mock}
}

func newTestServerWithRateLimit(enabled bool, limit int, window time.Duration) *testServer {
	mock := newMockDB()
	rateLimitConfig := &ratelimit.Config{
		Enabled:       enabled,
		DefaultLimit:  limit,
		DefaultWindow: window,
		EndpointConfigs: []ratelimit.EndpointConfig{
			{Path: "/run", Method: "POST", Limit: 5, Window: time.Hour, Burst: 5},
			{Path: "/health", Method: "GET", Limit: 0, Window: 0}, // Unlimited
		},
	}
	s := &Server{
		db:          mock,
		apiKey:      "test-api-key",
		rateLimiter: ratelimit.NewLimiter(rateLimitConfig),
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

// TestRateLimitMiddleware_Headers tests that rate limit headers are set
func TestRateLimitMiddleware_Headers(t *testing.T) {
	s := newTestServerWithRateLimit(true, 10, time.Minute)

	handler := s.withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("expected X-RateLimit-Limit header")
	}
	if w.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header")
	}
	if w.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset header")
	}
}

// TestRateLimitMiddleware_429Response tests 429 response when limit exceeded
func TestRateLimitMiddleware_429Response(t *testing.T) {
	s := newTestServerWithRateLimit(true, 2, time.Minute)

	handler := s.withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	// Make requests up to limit
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 for request %d, got %d", i+1, w.Code)
		}
	}

	// 3rd request should be rate limited
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["error"] != "rate_limit_exceeded" {
		t.Errorf("expected error 'rate_limit_exceeded', got '%v'", resp["error"])
	}

	if w.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header in 429 response")
	}
}

// TestRateLimitMiddleware_EndpointSpecific tests different limits for different endpoints
func TestRateLimitMiddleware_EndpointSpecific(t *testing.T) {
	s := newTestServerWithRateLimit(true, 1000, time.Minute)

	handler := s.withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/run", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	// Make 5 requests (endpoint limit)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 for request %d, got %d", i+1, w.Code)
		}
		limit, _ := strconv.Atoi(w.Header().Get("X-RateLimit-Limit"))
		if limit != 5 {
			t.Errorf("expected limit 5 for /run endpoint, got %d", limit)
		}
	}

	// 6th request should be rate limited
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}
}

// TestRateLimitMiddleware_HealthCheckExempt tests that health check is unlimited
func TestRateLimitMiddleware_HealthCheckExempt(t *testing.T) {
	s := newTestServerWithRateLimit(true, 1, time.Minute)

	handler := s.withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	// Make many requests - all should succeed
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 for health check request %d, got %d", i+1, w.Code)
		}
		// Health check should not have rate limit headers (unlimited)
		if w.Header().Get("X-RateLimit-Limit") != "" {
			t.Error("health check should not have rate limit headers")
		}
	}
}

// TestRateLimitMiddleware_Whitelist tests whitelist functionality
func TestRateLimitMiddleware_Whitelist(t *testing.T) {
	mock := newMockDB()
	config := &ratelimit.Config{
		Enabled:       true,
		DefaultLimit:  1,
		DefaultWindow: time.Minute,
		Whitelist:     map[string]bool{"127.0.0.1": true},
	}
	s := &Server{
		db:          mock,
		apiKey:      "test-api-key",
		rateLimiter: ratelimit.NewLimiter(config),
	}

	handler := s.withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	// Make many requests - all should succeed (whitelisted)
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 for whitelisted request %d, got %d", i+1, w.Code)
		}
	}
}

// TestRateLimitMiddleware_Blacklist tests blacklist functionality
func TestRateLimitMiddleware_Blacklist(t *testing.T) {
	mock := newMockDB()
	config := &ratelimit.Config{
		Enabled:       true,
		DefaultLimit:  1000,
		DefaultWindow: time.Minute,
		Blacklist:     map[string]bool{"192.168.1.1": true},
	}
	s := &Server{
		db:          mock,
		apiKey:      "test-api-key",
		rateLimiter: ratelimit.NewLimiter(config),
	}

	handler := s.withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Request should be denied (blacklisted)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429 for blacklisted IP, got %d", w.Code)
	}
}

// TestRateLimitMiddleware_Disabled tests that rate limiting can be disabled
func TestRateLimitMiddleware_Disabled(t *testing.T) {
	s := newTestServerWithRateLimit(false, 1, time.Minute)

	handler := s.withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	// Make many requests - all should succeed (rate limiting disabled)
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 when rate limiting disabled, got %d", w.Code)
		}
	}
}

// TestExtractClientID tests client ID extraction
func TestExtractClientID(t *testing.T) {
	s := newTestServer()

	tests := []struct {
		remoteAddr string
		expected   string
	}{
		{"127.0.0.1:12345", "127.0.0.1"},
		{"192.168.1.1:8080", "192.168.1.1"},
		{"[::1]:12345", "::1"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = tt.remoteAddr
		got := s.extractClientID(req)
		if got != tt.expected {
			t.Errorf("extractClientID(%q) = %q, want %q", tt.remoteAddr, got, tt.expected)
		}
	}
}
