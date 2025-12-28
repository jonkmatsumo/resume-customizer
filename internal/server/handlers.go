package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/pipeline"
)

// RunRequest represents the request body for /run
type RunRequest struct {
	JobURL     string `json:"job_url,omitempty"`
	JobPath    string `json:"job,omitempty"`
	Experience string `json:"experience"`
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
	if req.Experience == "" {
		s.errorResponse(w, http.StatusBadRequest, "experience is required")
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
		ExperiencePath: req.Experience,
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
	if req.Experience == "" {
		s.errorResponse(w, http.StatusBadRequest, "experience is required")
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

	// Setup SSE writer
	sse, err := NewSSEWriter(w)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("Starting streaming pipeline run...")

	// Build pipeline options with progress callback
	opts := pipeline.RunOptions{
		JobURL:         req.JobURL,
		JobPath:        req.JobPath,
		ExperiencePath: req.Experience,
		TemplatePath:   req.Template,
		CandidateName:  req.Name,
		CandidateEmail: req.Email,
		CandidatePhone: req.Phone,
		MaxBullets:     req.MaxBullets,
		MaxLines:       req.MaxLines,
		APIKey:         s.apiKey,
		DatabaseURL:    s.databaseURL,
		Verbose:        true,
		OnProgress: func(event pipeline.ProgressEvent) {
			if err := sse.WriteEvent("step", event); err != nil {
				log.Printf("Error writing SSE event: %v", err)
			}
		},
	}

	// Run pipeline synchronously (blocking until complete)
	ctx := r.Context()
	if err := pipeline.RunPipeline(ctx, opts); err != nil {
		log.Printf("Pipeline run failed: %v", err)
		sse.WriteError(err.Error())
		return
	}

	sse.WriteComplete("", "completed")
	log.Printf("Streaming pipeline run completed")
}
