package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/pipeline"
	"github.com/jonathan/resume-customizer/internal/server/middleware"
)

// RunRequest represents the request body for /run
type RunRequest struct {
	JobURL     string `json:"job_url,omitempty"`
	JobPath    string `json:"job,omitempty"`
	UserID     string `json:"user_id"` // UUID of user in DB (required)
	Name       string `json:"name,omitempty"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Template   string `json:"template,omitempty"`
	MaxBullets int    `json:"max_bullets,omitempty"`
	MaxLines   int    `json:"max_lines,omitempty"`
}

// RunResponse represents the response for /run
type RunResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// StatusResponse represents the response for /status
type StatusResponse struct {
	RunID     string `json:"run_id"`
	Company   string `json:"company"`
	RoleTitle string `json:"role_title"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// ArtifactResponse represents the response for /artifact
type ArtifactResponse struct {
	RunID       string `json:"run_id"`
	Step        string `json:"step"`
	Category    string `json:"category"`
	Content     any    `json:"content,omitempty"`
	TextContent string `json:"text_content,omitempty"`
}

// handleRun starts a new pipeline run
func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.JobURL == "" && req.JobPath == "" {
		s.errorResponse(w, http.StatusBadRequest, "Either job_url or job is required")
		return
	}
	if req.UserID == "" {
		s.errorResponse(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Set defaults
	if req.Template == "" {
		req.Template = "templates/one_page_resume.tex"
	}
	if req.MaxBullets == 0 {
		req.MaxBullets = 25
	}
	if req.MaxLines == 0 {
		req.MaxLines = 35
	}

	// Build pipeline options
	opts := pipeline.RunOptions{
		JobURL:         req.JobURL,
		JobPath:        req.JobPath,
		TemplatePath:   req.Template,
		CandidateName:  req.Name,
		CandidateEmail: req.Email,
		CandidatePhone: req.Phone,
		MaxBullets:     req.MaxBullets,
		MaxLines:       req.MaxLines,
		APIKey:         s.apiKey,
		DatabaseURL:    s.databaseURL,
		Verbose:        true,
	}

	// Fetch experience data from DB using UserID
	uid, err := uuid.Parse(req.UserID)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user_id")
		return
	}

	// Fetch user profile if name/email not provided in request
	if req.Name == "" || req.Email == "" {
		u, err := s.db.GetUser(r.Context(), uid)
		if err != nil {
			s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch user profile: "+err.Error())
			return
		}
		if u == nil {
			s.errorResponse(w, http.StatusBadRequest, "User not found")
			return
		}
		if req.Name == "" {
			opts.CandidateName = u.Name
		}
		if req.Email == "" {
			opts.CandidateEmail = u.Email
		}
		if req.Phone == "" {
			opts.CandidatePhone = u.Phone
		}
	}

	expData, err := s.fetchExperienceBankFromDB(r.Context(), uid)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch experience data: "+err.Error())
		return
	}
	opts.ExperienceData = expData

	// Generate a preliminary run ID for the response
	// The actual run will be created in the pipeline
	preliminaryID := uuid.New().String()

	log.Printf("Starting pipeline run (preliminary ID: %s)", preliminaryID)

	// Run pipeline in background
	go func() {
		ctx := context.Background()
		if err := pipeline.RunPipeline(ctx, opts); err != nil {
			log.Printf("Pipeline run failed: %v", err)
		}
	}()

	s.jsonResponse(w, http.StatusAccepted, RunResponse{
		RunID:  preliminaryID,
		Status: "started",
	})
}

// handleStatus returns the status of a pipeline run
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		s.errorResponse(w, http.StatusBadRequest, "Run ID is required")
		return
	}

	runID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run ID format")
		return
	}

	run, err := s.db.GetRun(r.Context(), runID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if run == nil {
		s.errorResponse(w, http.StatusNotFound, "Run not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, StatusResponse{
		RunID:     run.ID.String(),
		Company:   run.Company,
		RoleTitle: run.RoleTitle,
		Status:    run.Status,
		CreatedAt: run.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// handleArtifact returns an artifact by ID
func (s *Server) handleArtifact(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		s.errorResponse(w, http.StatusBadRequest, "Artifact ID is required")
		return
	}

	// Parse as artifact UUID
	artifactID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid artifact ID format")
		return
	}

	// Get artifact by ID
	artifact, err := s.db.GetArtifactByID(r.Context(), artifactID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if artifact == nil {
		s.errorResponse(w, http.StatusNotFound, "Artifact not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, artifact)
}

// handleRunStream starts a pipeline and streams progress via SSE
func (s *Server) handleRunStream(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.JobURL == "" && req.JobPath == "" {
		s.errorResponse(w, http.StatusBadRequest, "Either job_url or job is required")
		return
	}
	if req.UserID == "" {
		s.errorResponse(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Set defaults
	if req.Template == "" {
		req.Template = "templates/one_page_resume.tex"
	}
	if req.MaxBullets == 0 {
		req.MaxBullets = 25
	}
	if req.MaxLines == 0 {
		req.MaxLines = 35
	}

	// Fetch experience data from DB using UserID
	uid, err := uuid.Parse(req.UserID)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user_id")
		return
	}

	// Fetch user profile if name/email not provided in request
	if req.Name == "" || req.Email == "" {
		u, err := s.db.GetUser(r.Context(), uid)
		if err != nil {
			s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch user profile: "+err.Error())
			return
		}
		if u == nil {
			s.errorResponse(w, http.StatusBadRequest, "User not found")
			return
		}
		if req.Name == "" {
			req.Name = u.Name
		}
		if req.Email == "" {
			req.Email = u.Email
		}
		if req.Phone == "" {
			req.Phone = u.Phone
		}
	}

	expData, err := s.fetchExperienceBankFromDB(r.Context(), uid)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch experience data: "+err.Error())
		return
	}

	// Setup SSE writer
	sse, err := NewSSEWriter(w)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create run early (before ingestion) so we can send run_id as first event
	ctx := r.Context()
	var runID *uuid.UUID
	if s.databaseURL != "" {
		// Create run with placeholder company/role (will be updated after parsing)
		jobURL := req.JobURL
		if jobURL == "" {
			jobURL = req.JobPath // Use path if URL not provided
		}
		createdRunID, err := s.db.CreateRun(ctx, "", "", jobURL)
		if err != nil {
			log.Printf("Warning: Failed to create database run: %v", err)
		} else {
			runID = &createdRunID
			// Send run_id as the FIRST SSE event before any ingestion
			// Use the same format as the pipeline's emitRunStarted for consistency
			// This MUST be sent and flushed before pipeline starts
			runStartedEvent := pipeline.ProgressEvent{
				Step:     db.StepRunStarted,
				Category: db.CategoryLifecycle,
				Message:  "Pipeline run started",
				RunID:    createdRunID.String(),
			}
			if err := sse.WriteEvent("step", runStartedEvent); err != nil {
				log.Printf("Error writing run_started SSE event: %v", err)
			} else {
				log.Printf("Created run %s, sent run_id as first SSE event (before pipeline start)", createdRunID)
				// WriteEvent already flushes, but we ensure it's sent before pipeline starts
			}
		}
	}

	// Ensure we have a run ID before starting pipeline
	if runID == nil && s.databaseURL != "" {
		log.Printf("Warning: Failed to create run before pipeline start, pipeline will create one later")
	}

	log.Printf("Starting streaming pipeline run...")

	// Build pipeline options with progress callback
	opts := pipeline.RunOptions{
		JobURL:         req.JobURL,
		JobPath:        req.JobPath,
		ExperienceData: expData,
		TemplatePath:   req.Template,
		CandidateName:  req.Name,
		CandidateEmail: req.Email,
		CandidatePhone: req.Phone,
		MaxBullets:     req.MaxBullets,
		MaxLines:       req.MaxLines,
		APIKey:         s.apiKey,
		DatabaseURL:    s.databaseURL,
		Verbose:        true,
		ExistingRunID:  runID,        // Pass existing run ID to pipeline
		RunStartedSent: runID != nil, // Mark that we already sent run_started
		OnProgress: func(event pipeline.ProgressEvent) {
			if err := sse.WriteEvent("step", event); err != nil {
				log.Printf("Error writing SSE event: %v", err)
			}
		},
	}

	// Run pipeline synchronously (blocking until complete)
	if err := pipeline.RunPipeline(ctx, opts); err != nil {
		log.Printf("Pipeline run failed: %v", err)
		sse.WriteError(err.Error())
		return
	}

	sse.WriteComplete("", "completed")
	log.Printf("Streaming pipeline run completed")
}

// handleListRuns returns a list of pipeline runs with optional filters
func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	filters := db.RunFilters{
		Company: r.URL.Query().Get("company"),
		Status:  r.URL.Query().Get("status"),
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filters.Limit = limit
		}
	}

	runs, err := s.db.ListRunsFiltered(r.Context(), filters)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	// Convert to response format
	type RunItem struct {
		ID        string `json:"id"`
		Company   string `json:"company"`
		RoleTitle string `json:"role_title"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	}
	response := make([]RunItem, 0, len(runs))
	for _, run := range runs {
		response = append(response, RunItem{
			ID:        run.ID.String(),
			Company:   run.Company,
			RoleTitle: run.RoleTitle,
			Status:    run.Status,
			CreatedAt: run.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"runs":  response,
		"count": len(response),
	})
}

// handleListUserRuns returns a list of pipeline runs for a specific user
func (s *Server) handleListUserRuns(w http.ResponseWriter, r *http.Request) {
	// Get user ID from path parameter
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Verify the authenticated user matches the user ID in the path
	authenticatedUserID, err := middleware.GetUserID(r)
	if err != nil {
		s.errorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if authenticatedUserID != userID {
		s.errorResponse(w, http.StatusForbidden, "You can only view your own runs")
		return
	}

	// Parse query parameters for filtering
	filters := db.RunFilters{
		Company: r.URL.Query().Get("company"),
		Status:  r.URL.Query().Get("status"),
		UserID:  &userID, // Filter by user ID
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filters.Limit = limit
		}
	}

	runs, err := s.db.ListRunsFiltered(r.Context(), filters)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	// Convert to response format (same as handleListRuns)
	type RunItem struct {
		ID        string `json:"id"`
		Company   string `json:"company"`
		RoleTitle string `json:"role_title"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	}
	response := make([]RunItem, 0, len(runs))
	for _, run := range runs {
		response = append(response, RunItem{
			ID:        run.ID.String(),
			Company:   run.Company,
			RoleTitle: run.RoleTitle,
			Status:    run.Status,
			CreatedAt: run.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"runs":  response,
		"count": len(response),
	})
}

// handleDeleteRun deletes a pipeline run and its artifacts
func (s *Server) handleDeleteRun(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		s.errorResponse(w, http.StatusBadRequest, "Run ID is required")
		return
	}

	runID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run ID format")
		return
	}

	if err := s.db.DeleteRun(r.Context(), runID); err != nil {
		if err.Error() == "run not found: "+runID.String() {
			s.errorResponse(w, http.StatusNotFound, "Run not found")
			return
		}
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleListArtifacts returns a list of artifacts with optional filters
func (s *Server) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	filters := db.ArtifactFilters{
		Step:     r.URL.Query().Get("step"),
		Category: r.URL.Query().Get("category"),
	}

	if runIDStr := r.URL.Query().Get("run_id"); runIDStr != "" {
		runID, err := uuid.Parse(runIDStr)
		if err != nil {
			s.errorResponse(w, http.StatusBadRequest, "Invalid run_id format")
			return
		}
		filters.RunID = runID
	}

	artifacts, err := s.db.ListArtifacts(r.Context(), filters)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"artifacts": artifacts,
		"count":     len(artifacts),
	})
}

// handleRunArtifacts returns artifacts for a specific run
func (s *Server) handleRunArtifacts(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		s.errorResponse(w, http.StatusBadRequest, "Run ID is required")
		return
	}

	runID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run ID format")
		return
	}

	artifacts, err := s.db.ListArtifacts(r.Context(), db.ArtifactFilters{RunID: runID})
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"run_id":    runID.String(),
		"artifacts": artifacts,
		"count":     len(artifacts),
	})
}

// handleRunResumeTex returns the resume.tex for a specific run as plain text
func (s *Server) handleRunResumeTex(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		s.errorResponse(w, http.StatusBadRequest, "Run ID is required")
		return
	}

	runID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run ID format")
		return
	}

	tex, err := s.db.GetTextArtifact(r.Context(), runID, "resume_tex")
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if tex == "" {
		s.errorResponse(w, http.StatusNotFound, "resume.tex not found for this run")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=resume.tex")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(tex))
}
