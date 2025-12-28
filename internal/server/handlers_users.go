package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/types"
)

// ---------------------------------------------------------------------
// User Handlers
// ---------------------------------------------------------------------

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.Email == "" {
		s.errorResponse(w, http.StatusBadRequest, "Name and Email are required")
		return
	}

	id, err := s.db.CreateUser(r.Context(), req.Name, req.Email, req.Phone)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := s.db.GetUser(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if user == nil {
		s.errorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, user)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req db.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.ID = userID

	if err := s.db.UpdateUser(r.Context(), &req); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := s.db.DeleteUser(r.Context(), userID); err != nil {
		if err.Error() == "user not found: "+userID.String() {
			s.errorResponse(w, http.StatusNotFound, "User not found")
			return
		}
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---------------------------------------------------------------------
// Job Handlers
// ---------------------------------------------------------------------

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	jobs, err := s.db.ListJobs(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req db.Job
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.UserID = userID

	id, err := s.db.CreateJob(r.Context(), &req)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (s *Server) handleUpdateJob(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	var req db.Job
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.ID = jobID

	if err := s.db.UpdateJob(r.Context(), &req); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	if err := s.db.DeleteJob(r.Context(), jobID); err != nil {
		if err.Error() == "job not found: "+jobID.String() {
			s.errorResponse(w, http.StatusNotFound, "Job not found")
			return
		}
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---------------------------------------------------------------------
// Experience Handlers
// ---------------------------------------------------------------------

func (s *Server) handleListExperiences(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	experiences, err := s.db.ListExperiences(r.Context(), jobID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"experiences": experiences,
		"count":       len(experiences),
	})
}

func (s *Server) handleCreateExperience(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	var req db.Experience
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.JobID = jobID

	id, err := s.db.CreateExperience(r.Context(), &req)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (s *Server) handleUpdateExperience(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	expID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid experience ID")
		return
	}

	var req db.Experience
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.ID = expID

	if err := s.db.UpdateExperience(r.Context(), &req); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteExperience(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	expID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid experience ID")
		return
	}

	if err := s.db.DeleteExperience(r.Context(), expID); err != nil {
		if err.Error() == "experience not found: "+expID.String() {
			s.errorResponse(w, http.StatusNotFound, "Experience not found")
			return
		}
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---------------------------------------------------------------------
// Education Handlers
// ---------------------------------------------------------------------

func (s *Server) handleListEducation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	education, err := s.db.ListEducation(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"education": education,
		"count":     len(education),
	})
}

func (s *Server) handleCreateEducation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req db.Education
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.UserID = userID

	id, err := s.db.CreateEducation(r.Context(), &req)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (s *Server) handleUpdateEducation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	eduID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid education ID")
		return
	}

	var req db.Education
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.ID = eduID

	if err := s.db.UpdateEducation(r.Context(), &req); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteEducation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	eduID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid education ID")
		return
	}

	if err := s.db.DeleteEducation(r.Context(), eduID); err != nil {
		if err.Error() == "education not found: "+eduID.String() {
			s.errorResponse(w, http.StatusNotFound, "Education not found")
			return
		}
		s.errorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---------------------------------------------------------------------
// Experience Bank Export
// ---------------------------------------------------------------------

func (s *Server) handleGetExperienceBank(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	expBank, err := s.fetchExperienceBankFromDB(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch experience bank: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, expBank)
}

// fetchExperienceBankFromDB fetches user data and converts it to ExperienceBank structure
func (s *Server) fetchExperienceBankFromDB(ctx context.Context, userID uuid.UUID) (*types.ExperienceBank, error) {
	// 1. Get Jobs
	jobs, err := s.db.ListJobs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching jobs: %w", err)
	}

	// 2. Get Education
	education, err := s.db.ListEducation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching education: %w", err)
	}

	// 3. Construct Stories from Jobs + Experiences
	stories := make([]types.Story, 0, len(jobs))
	for _, job := range jobs {
		exps, err := s.db.ListExperiences(ctx, job.ID)
		if err != nil {
			return nil, fmt.Errorf("fetching experiences for job %s: %w", job.ID, err)
		}

		bullets := make([]types.Bullet, 0, len(exps))
		for _, e := range exps {
			bullets = append(bullets, types.Bullet{
				ID:               e.ID.String(),
				Text:             e.BulletText,
				Skills:           e.Skills,
				LengthChars:      len(e.BulletText),
				EvidenceStrength: e.EvidenceStrength,
				RiskFlags:        e.RiskFlags,
			})
		}

		sDate := ""
		if job.StartDate != nil {
			sDate = job.StartDate.Format("2006-01")
		}
		eDate := ""
		if job.EndDate != nil {
			eDate = job.EndDate.Format("2006-01")
		}

		stories = append(stories, types.Story{
			ID:        job.ID.String(),
			Company:   job.Company,
			Role:      job.RoleTitle,
			StartDate: sDate,
			EndDate:   eDate,
			Bullets:   bullets,
		})
	}

	// 4. Transform Education
	eduItems := make([]types.Education, 0, len(education))
	for _, e := range education {
		sDate := ""
		if e.StartDate != nil {
			sDate = e.StartDate.Format("2006-01")
		}
		eDate := ""
		if e.EndDate != nil {
			eDate = e.EndDate.Format("2006-01")
		}
		eduItems = append(eduItems, types.Education{
			ID:        e.ID.String(),
			School:    e.School,
			Degree:    e.DegreeType,
			Field:     e.Field,
			StartDate: sDate,
			EndDate:   eDate,
			GPA:       e.GPA,
		})
	}

	return &types.ExperienceBank{
		Stories:   stories,
		Education: eduItems,
	}, nil
}
