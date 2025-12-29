package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/pipeline/steps"
)

// RunCreateRequest represents the request to create a new pipeline run
type RunCreateRequest struct {
	UserID     string `json:"user_id"`     // REQUIRED
	JobURL     string `json:"job_url"`     // Required if job_text not provided
	JobText    string `json:"job_text"`    // Required if job_url not provided
	Template   string `json:"template"`    // optional
	MaxBullets int    `json:"max_bullets"` // optional
	MaxLines   int    `json:"max_lines"`   // optional
}

// RunCreateResponse represents the response for creating a run
type RunCreateResponse struct {
	RunID     string         `json:"run_id"`
	Status    string         `json:"status"`
	CreatedAt string         `json:"created_at"`
	Steps     RunStepsStatus `json:"steps"`
}

// RunStepsStatus represents the status of steps for a run
type RunStepsStatus struct {
	Completed []string `json:"completed"`
	Available []string `json:"available"`
	Blocked   []string `json:"blocked"`
}

// StepExecuteRequest represents the request to execute a step
type StepExecuteRequest struct {
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// StepExecuteResponse represents the response for executing a step
type StepExecuteResponse struct {
	Step        string              `json:"step"`
	Status      string              `json:"status"`
	RunID       string              `json:"run_id"`
	StartedAt   string              `json:"started_at,omitempty"`
	CompletedAt string              `json:"completed_at,omitempty"`
	DurationMs  *int                `json:"duration_ms,omitempty"`
	ArtifactID  *string             `json:"artifact_id,omitempty"`
	NextSteps   []string            `json:"next_steps,omitempty"`
	Checkpoint  *CheckpointResponse `json:"checkpoint,omitempty"`
}

// CheckpointResponse represents a checkpoint
type CheckpointResponse struct {
	Step        string                 `json:"step"`
	RunID       string                 `json:"run_id"`
	CompletedAt string                 `json:"completed_at"`
	Artifacts   map[string]interface{} `json:"artifacts"`
}

// StepStatusResponse represents the status of a single step
type StepStatusResponse struct {
	Step        string  `json:"step"`
	Status      string  `json:"status"`
	RunID       string  `json:"run_id"`
	StartedAt   *string `json:"started_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
	DurationMs  *int    `json:"duration_ms,omitempty"`
	ArtifactID  *string `json:"artifact_id,omitempty"`
	Error       *string `json:"error,omitempty"`
}

// RunStepsListResponse represents the list of all steps for a run
type RunStepsListResponse struct {
	RunID   string               `json:"run_id"`
	Status  string               `json:"status"`
	Steps   []StepStatusResponse `json:"steps"`
	Summary RunStepsSummary      `json:"summary"`
}

// RunStepsSummary represents a summary of step statuses
type RunStepsSummary struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Pending    int `json:"pending"`
	Failed     int `json:"failed"`
	Skipped    int `json:"skipped"`
}

// ResumeRequest represents the request to resume from a checkpoint
type ResumeRequest struct {
	AutoContinue bool `json:"auto_continue,omitempty"`
	MaxSteps     int  `json:"max_steps,omitempty"`
}

// ResumeResponse represents the response for resuming
type ResumeResponse struct {
	RunID              string   `json:"run_id"`
	ResumedFrom        string   `json:"resumed_from"`
	ExecutedSteps      []string `json:"executed_steps"`
	CurrentStatus      string   `json:"current_status"`
	NextAvailableSteps []string `json:"next_available_steps"`
}

// CheckpointGetResponse represents the response for getting a checkpoint
type CheckpointGetResponse struct {
	RunID              string                 `json:"run_id"`
	CheckpointStep     string                 `json:"checkpoint_step"`
	CheckpointAt       string                 `json:"checkpoint_at"`
	CompletedSteps     []string               `json:"completed_steps"`
	NextAvailableSteps []string               `json:"next_available_steps"`
	Artifacts          map[string]interface{} `json:"artifacts"`
}

// handleCreateRun creates a new pipeline run for step-by-step execution
func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var req RunCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate user_id is required
	if req.UserID == "" {
		s.jsonResponse(w, http.StatusBadRequest, map[string]string{
			"error":   "user_id is required",
			"details": "The user_id field is required and cannot be empty. Please provide a valid user UUID.",
		})
		return
	}

	// Validate user_id is a valid UUID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		s.jsonResponse(w, http.StatusBadRequest, map[string]string{
			"error":   "Invalid user_id format",
			"details": "The user_id must be a valid UUID format.",
		})
		return
	}

	// Validate user exists in database
	user, err := s.db.GetUser(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if user == nil {
		s.jsonResponse(w, http.StatusNotFound, map[string]string{
			"error":   "User not found",
			"details": "No user found with the provided user_id.",
		})
		return
	}

	// Validate job input
	if req.JobURL == "" && req.JobText == "" {
		s.errorResponse(w, http.StatusBadRequest, "Either job_url or job_text is required")
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

	// Create a pipeline run in the database
	// We'll create a minimal run record that will be populated as steps execute
	var companyName string
	if req.JobURL != "" {
		// Try to extract company name from URL or use a placeholder
		companyName = "Unknown"
	}

	runID, err := s.db.CreateRun(r.Context(), companyName, "", req.JobURL)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to create run: "+err.Error())
		return
	}

	// Update run with user_id
	_, err = s.db.Pool().Exec(r.Context(),
		"UPDATE pipeline_runs SET user_id = $1 WHERE id = $2",
		userID, runID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to update run: "+err.Error())
		return
	}

	// Get available steps (should be just ingest_job initially)
	available, err := steps.GetAvailableSteps(r.Context(), s.db, runID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to get available steps: "+err.Error())
		return
	}

	blocked, err := steps.GetBlockedSteps(r.Context(), s.db, runID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to get blocked steps: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, RunCreateResponse{
		RunID:     runID.String(),
		Status:    "created",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Steps: RunStepsStatus{
			Completed: []string{},
			Available: available,
			Blocked:   blocked,
		},
	})
}

// handleExecuteStep executes a specific pipeline step
func (s *Server) handleExecuteStep(w http.ResponseWriter, r *http.Request) {
	runIDStr := r.PathValue("run_id")
	stepName := r.PathValue("step_name")

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run_id format")
		return
	}

	// Verify run exists
	run, err := s.db.GetRun(r.Context(), runID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if run == nil {
		s.errorResponse(w, http.StatusNotFound, "Run not found")
		return
	}

	// Check if step is already completed or in progress
	existingStep, err := s.db.GetRunStep(r.Context(), runID, stepName)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if existingStep != nil {
		if existingStep.Status == db.StepStatusCompleted {
			s.errorResponse(w, http.StatusConflict, "Step already completed")
			return
		}
		if existingStep.Status == db.StepStatusInProgress {
			s.errorResponse(w, http.StatusConflict, "Step already in progress")
			return
		}
	}

	// Validate dependencies
	if err := steps.ValidateDependencies(r.Context(), s.db, runID, stepName); err != nil {
		var missingDeps []string
		if depErr, ok := err.(*steps.DependencyError); ok {
			missingDeps = depErr.MissingDependencies
		} else {
			missingDeps = []string{err.Error()}
		}

		available, _ := steps.GetAvailableSteps(r.Context(), s.db, runID)
		s.jsonResponse(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error": "Dependencies not met",
			"details": map[string]interface{}{
				"step":                 stepName,
				"missing_dependencies": missingDeps,
				"available_steps":      available,
			},
		})
		return
	}

	// Parse request body for parameters
	var stepReq StepExecuteRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&stepReq)
	}

	// Get step definition
	def, ok := steps.StepRegistry[stepName]
	if !ok {
		s.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("Unknown step: %s", stepName))
		return
	}

	// Create or update step record
	if existingStep == nil {
		stepInput := &db.RunStepInput{
			Step:       stepName,
			Category:   def.Category,
			Status:     db.StepStatusInProgress,
			Parameters: stepReq.Parameters,
		}
		_, err = s.db.CreateRunStep(r.Context(), runID, stepInput)
		if err != nil {
			s.errorResponse(w, http.StatusInternalServerError, "Failed to create step record: "+err.Error())
			return
		}
	} else {
		err = s.db.UpdateRunStepStatus(r.Context(), runID, stepName, db.StepStatusInProgress, nil, nil)
		if err != nil {
			s.errorResponse(w, http.StatusInternalServerError, "Failed to update step status: "+err.Error())
			return
		}
	}

	// TODO: Execute the actual step using step executors
	// For now, we'll mark it as completed immediately as a placeholder
	// This will be replaced with actual step execution logic
	startTime := time.Now()

	// Placeholder: In a real implementation, we would call:
	// executor := steps.GetExecutor(stepName)
	// result, err := executor.Execute(r.Context(), runID, stepReq.Parameters)

	// For now, simulate completion
	duration := int(time.Since(startTime).Milliseconds())
	err = s.db.UpdateRunStepStatus(r.Context(), runID, stepName, db.StepStatusCompleted, nil, nil)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to update step status: "+err.Error())
		return
	}

	// Get next available steps
	available, _ := steps.GetAvailableSteps(r.Context(), s.db, runID)

	// Create checkpoint
	checkpointInput := &db.RunCheckpointInput{
		Step:      stepName,
		Artifacts: make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
	}
	checkpoint, _ := s.db.CreateRunCheckpoint(r.Context(), runID, checkpointInput)

	var checkpointResp *CheckpointResponse
	if checkpoint != nil {
		checkpointResp = &CheckpointResponse{
			Step:        checkpoint.Step,
			RunID:       checkpoint.RunID.String(),
			CompletedAt: checkpoint.CompletedAt.Format(time.RFC3339),
			Artifacts:   checkpoint.Artifacts,
		}
	}

	completedAt := time.Now()
	s.jsonResponse(w, http.StatusOK, StepExecuteResponse{
		Step:        stepName,
		Status:      db.StepStatusCompleted,
		RunID:       runID.String(),
		StartedAt:   startTime.Format(time.RFC3339),
		CompletedAt: completedAt.Format(time.RFC3339),
		DurationMs:  &duration,
		NextSteps:   available,
		Checkpoint:  checkpointResp,
	})
}

// handleGetStepStatus returns the status of a specific step
func (s *Server) handleGetStepStatus(w http.ResponseWriter, r *http.Request) {
	runIDStr := r.PathValue("run_id")
	stepName := r.PathValue("step_name")

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run_id format")
		return
	}

	step, err := s.db.GetRunStep(r.Context(), runID, stepName)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if step == nil {
		s.errorResponse(w, http.StatusNotFound, "Step not found")
		return
	}

	var startedAt, completedAt *string
	if step.StartedAt != nil {
		s := step.StartedAt.Format(time.RFC3339)
		startedAt = &s
	}
	if step.CompletedAt != nil {
		c := step.CompletedAt.Format(time.RFC3339)
		completedAt = &c
	}

	var artifactID *string
	if step.ArtifactID != nil {
		a := step.ArtifactID.String()
		artifactID = &a
	}

	s.jsonResponse(w, http.StatusOK, StepStatusResponse{
		Step:        step.Step,
		Status:      step.Status,
		RunID:       step.RunID.String(),
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		DurationMs:  step.DurationMs,
		ArtifactID:  artifactID,
		Error:       step.ErrorMessage,
	})
}

// handleListRunSteps returns all steps for a run
func (s *Server) handleListRunSteps(w http.ResponseWriter, r *http.Request) {
	runIDStr := r.PathValue("run_id")

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run_id format")
		return
	}

	// Verify run exists
	run, err := s.db.GetRun(r.Context(), runID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if run == nil {
		s.errorResponse(w, http.StatusNotFound, "Run not found")
		return
	}

	// Get filter parameters
	var status, category *string
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status = &statusStr
	}
	if categoryStr := r.URL.Query().Get("category"); categoryStr != "" {
		category = &categoryStr
	}

	// Get all steps
	stepList, err := s.db.ListRunSteps(r.Context(), runID, status, category)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	// Build response with all step definitions
	allSteps := make(map[string]db.RunStep)
	for _, step := range stepList {
		allSteps[step.Step] = step
	}

	var stepsResp []StepStatusResponse
	summary := RunStepsSummary{}

	// Include all defined steps, even if not yet created
	for stepName := range steps.StepRegistry {
		var stepResp StepStatusResponse
		if existing, ok := allSteps[stepName]; ok {
			// Step exists in database
			var startedAt, completedAt *string
			if existing.StartedAt != nil {
				s := existing.StartedAt.Format(time.RFC3339)
				startedAt = &s
			}
			if existing.CompletedAt != nil {
				c := existing.CompletedAt.Format(time.RFC3339)
				completedAt = &c
			}
			var artifactID *string
			if existing.ArtifactID != nil {
				a := existing.ArtifactID.String()
				artifactID = &a
			}

			stepResp = StepStatusResponse{
				Step:        existing.Step,
				Status:      existing.Status,
				RunID:       existing.RunID.String(),
				StartedAt:   startedAt,
				CompletedAt: completedAt,
				DurationMs:  existing.DurationMs,
				ArtifactID:  artifactID,
				Error:       existing.ErrorMessage,
			}
		} else {
			// Step not yet created, check if dependencies are met
			stepResp = StepStatusResponse{
				Step:   stepName,
				Status: db.StepStatusPending,
				RunID:  runID.String(),
			}
			// Check if blocked
			if err := steps.ValidateDependencies(r.Context(), s.db, runID, stepName); err != nil {
				stepResp.Status = db.StepStatusBlocked
			}
		}

		stepsResp = append(stepsResp, stepResp)

		// Update summary
		summary.Total++
		switch stepResp.Status {
		case db.StepStatusCompleted:
			summary.Completed++
		case db.StepStatusInProgress:
			summary.InProgress++
		case db.StepStatusPending, db.StepStatusBlocked:
			summary.Pending++
		case db.StepStatusFailed:
			summary.Failed++
		case db.StepStatusSkipped:
			summary.Skipped++
		}
	}

	s.jsonResponse(w, http.StatusOK, RunStepsListResponse{
		RunID:   runID.String(),
		Status:  run.Status,
		Steps:   stepsResp,
		Summary: summary,
	})
}

// handleGetCheckpoint returns the current checkpoint for a run
func (s *Server) handleGetCheckpoint(w http.ResponseWriter, r *http.Request) {
	runIDStr := r.PathValue("run_id")

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run_id format")
		return
	}

	checkpoint, err := s.db.GetRunCheckpoint(r.Context(), runID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if checkpoint == nil {
		s.errorResponse(w, http.StatusNotFound, "No checkpoint found")
		return
	}

	// Get completed steps
	allSteps, err := s.db.ListRunSteps(r.Context(), runID, nil, nil)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	var completedSteps []string
	for _, step := range allSteps {
		if step.Status == db.StepStatusCompleted {
			completedSteps = append(completedSteps, step.Step)
		}
	}

	// Get next available steps
	available, _ := steps.GetAvailableSteps(r.Context(), s.db, runID)

	s.jsonResponse(w, http.StatusOK, CheckpointGetResponse{
		RunID:              runID.String(),
		CheckpointStep:     checkpoint.Step,
		CheckpointAt:       checkpoint.CompletedAt.Format(time.RFC3339),
		CompletedSteps:     completedSteps,
		NextAvailableSteps: available,
		Artifacts:          checkpoint.Artifacts,
	})
}

// handleResumeFromCheckpoint resumes execution from the last checkpoint
func (s *Server) handleResumeFromCheckpoint(w http.ResponseWriter, r *http.Request) {
	runIDStr := r.PathValue("run_id")

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run_id format")
		return
	}

	// Verify run exists
	run, err := s.db.GetRun(r.Context(), runID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if run == nil {
		s.errorResponse(w, http.StatusNotFound, "Run not found")
		return
	}

	// Get checkpoint
	checkpoint, err := s.db.GetRunCheckpoint(r.Context(), runID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if checkpoint == nil {
		s.errorResponse(w, http.StatusBadRequest, "No checkpoint available")
		return
	}

	// Parse request
	var resumeReq ResumeRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&resumeReq)
	}
	if resumeReq.MaxSteps == 0 {
		resumeReq.MaxSteps = 5 // default
	}

	// Get available steps
	available, err := steps.GetAvailableSteps(r.Context(), s.db, runID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to get available steps: "+err.Error())
		return
	}

	// For now, just return the available steps
	// TODO: Implement actual step execution in background
	executedSteps := []string{} // Placeholder

	s.jsonResponse(w, http.StatusOK, ResumeResponse{
		RunID:              runID.String(),
		ResumedFrom:        checkpoint.Step,
		ExecutedSteps:      executedSteps,
		CurrentStatus:      run.Status,
		NextAvailableSteps: available,
	})
}

// handleSkipStep marks a step as skipped
func (s *Server) handleSkipStep(w http.ResponseWriter, r *http.Request) {
	runIDStr := r.PathValue("run_id")
	stepName := r.PathValue("step_name")

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run_id format")
		return
	}

	// Verify run exists
	run, err := s.db.GetRun(r.Context(), runID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if run == nil {
		s.errorResponse(w, http.StatusNotFound, "Run not found")
		return
	}

	// Get or create step
	step, err := s.db.GetRunStep(r.Context(), runID, stepName)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	if step == nil {
		// Create step with skipped status
		def, ok := steps.StepRegistry[stepName]
		if !ok {
			s.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("Unknown step: %s", stepName))
			return
		}
		stepInput := &db.RunStepInput{
			Step:     stepName,
			Category: def.Category,
			Status:   db.StepStatusSkipped,
		}
		step, err = s.db.CreateRunStep(r.Context(), runID, stepInput)
		if err != nil {
			s.errorResponse(w, http.StatusInternalServerError, "Failed to create step: "+err.Error())
			return
		}
	} else {
		// Update to skipped
		err = s.db.UpdateRunStepStatus(r.Context(), runID, stepName, db.StepStatusSkipped, nil, nil)
		if err != nil {
			s.errorResponse(w, http.StatusInternalServerError, "Failed to update step: "+err.Error())
			return
		}
		step.Status = db.StepStatusSkipped
	}

	s.jsonResponse(w, http.StatusOK, StepStatusResponse{
		Step:   step.Step,
		Status: step.Status,
		RunID:  step.RunID.String(),
	})
}

// handleRetryStep retries a failed step
func (s *Server) handleRetryStep(w http.ResponseWriter, r *http.Request) {
	runIDStr := r.PathValue("run_id")
	stepName := r.PathValue("step_name")

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid run_id format")
		return
	}

	// Get step
	step, err := s.db.GetRunStep(r.Context(), runID, stepName)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if step == nil {
		s.errorResponse(w, http.StatusNotFound, "Step not found")
		return
	}

	if step.Status != db.StepStatusFailed {
		s.errorResponse(w, http.StatusBadRequest, "Step is not in failed state")
		return
	}

	// Reset step to pending and re-execute
	err = s.db.UpdateRunStepStatus(r.Context(), runID, stepName, db.StepStatusPending, nil, nil)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to reset step: "+err.Error())
		return
	}

	// Re-execute by calling handleExecuteStep logic
	// For now, just return success
	s.jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Step reset to pending, ready for retry",
		"step":    stepName,
		"run_id":  runID.String(),
	})
}
