// Package server provides the HTTP REST API for the resume customizer.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jonathan/resume-customizer/internal/config"
	"github.com/jonathan/resume-customizer/internal/db"
	"github.com/jonathan/resume-customizer/internal/server/ratelimit"
)

// Server represents the HTTP server
type Server struct {
	httpServer  *http.Server
	db          *db.DB
	apiKey      string
	databaseURL string
	rateLimiter *ratelimit.Limiter
	jwtService  *JWTService //nolint:unused // Reserved for Phase 8 (routes with authentication)
	userService *UserService
	authHandler *AuthHandler
}

// Config holds server configuration
type Config struct {
	Port        int
	DatabaseURL string
	APIKey      string
}

// New creates a new server instance
func New(cfg Config) (*Server, error) {
	// Connect to database
	database, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &Server{
		db:          database,
		apiKey:      cfg.APIKey,
		databaseURL: cfg.DatabaseURL,
	}

	// Initialize rate limiter
	s.rateLimiter = ratelimit.NewLimiter(ratelimit.LoadConfig())

	// Initialize authentication services
	passwordConfig, err := config.NewPasswordConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create password config: %w", err)
	}
	s.userService = NewUserService(database, passwordConfig)

	jwtConfig, err := config.NewJWTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT config: %w", err)
	}
	jwtService := NewJWTService(jwtConfig)
	s.jwtService = jwtService // Store for future use in Phase 8 (routes)

	s.authHandler = NewAuthHandler(s.userService, jwtService)

	// Setup router
	mux := http.NewServeMux()
	mux.HandleFunc("POST /run", s.handleRun)
	mux.HandleFunc("POST /run/stream", s.handleRunStream)
	mux.HandleFunc("GET /status/{id}", s.handleStatus)
	mux.HandleFunc("GET /artifact/{id}", s.handleArtifact)
	mux.HandleFunc("GET /health", s.handleHealth)

	// Step-by-step pipeline API endpoints
	mux.HandleFunc("POST /runs", s.handleCreateRun)
	mux.HandleFunc("POST /runs/{run_id}/steps/{step_name}", s.handleExecuteStep)
	mux.HandleFunc("GET /runs/{run_id}/steps", s.handleListRunSteps)
	mux.HandleFunc("GET /runs/{run_id}/steps/{step_name}", s.handleGetStepStatus)
	mux.HandleFunc("GET /runs/{run_id}/checkpoint", s.handleGetCheckpoint)
	mux.HandleFunc("POST /runs/{run_id}/resume", s.handleResumeFromCheckpoint)
	mux.HandleFunc("POST /runs/{run_id}/steps/{step_name}/skip", s.handleSkipStep)
	mux.HandleFunc("POST /runs/{run_id}/steps/{step_name}/retry", s.handleRetryStep)

	// CRUD endpoints for runs
	mux.HandleFunc("GET /runs", s.handleListRuns)
	mux.HandleFunc("DELETE /runs/{id}", s.handleDeleteRun)
	mux.HandleFunc("GET /runs/{id}/artifacts", s.handleRunArtifacts)
	mux.HandleFunc("GET /runs/{id}/resume.tex", s.handleRunResumeTex)

	// CRUD endpoints for artifacts
	mux.HandleFunc("GET /artifacts", s.handleListArtifacts)

	// User Profile endpoints
	mux.HandleFunc("POST /users", s.handleCreateUser)
	mux.HandleFunc("GET /users/{id}", s.handleGetUser)
	mux.HandleFunc("PUT /users/{id}", s.handleUpdateUser)
	mux.HandleFunc("DELETE /users/{id}", s.handleDeleteUser)

	// Job endpoints
	mux.HandleFunc("GET /users/{id}/jobs", s.handleListJobs)
	mux.HandleFunc("POST /users/{id}/jobs", s.handleCreateJob)
	mux.HandleFunc("PUT /jobs/{id}", s.handleUpdateJob)
	mux.HandleFunc("DELETE /jobs/{id}", s.handleDeleteJob)

	// Experience endpoints
	mux.HandleFunc("GET /jobs/{id}/experiences", s.handleListExperiences)
	mux.HandleFunc("POST /jobs/{id}/experiences", s.handleCreateExperience)
	mux.HandleFunc("PUT /experiences/{id}", s.handleUpdateExperience)
	mux.HandleFunc("DELETE /experiences/{id}", s.handleDeleteExperience)

	// Education endpoints
	mux.HandleFunc("GET /users/{id}/education", s.handleListEducation)
	mux.HandleFunc("POST /users/{id}/education", s.handleCreateEducation)
	mux.HandleFunc("PUT /education/{id}", s.handleUpdateEducation)
	mux.HandleFunc("DELETE /education/{id}", s.handleDeleteEducation)

	// Export endpoint
	mux.HandleFunc("GET /users/{id}/experience-bank", s.handleGetExperienceBank)
	mux.HandleFunc("GET /users/{id}/experience-bank/stories", s.handleListStories)
	mux.HandleFunc("GET /users/{id}/experience-bank/stories/{story_id}", s.handleGetStory)
	mux.HandleFunc("GET /users/{id}/experience-bank/stories/{story_id}/bullets", s.handleGetStoryBullets)
	mux.HandleFunc("GET /users/{id}/experience-bank/skills", s.handleListSkills)
	mux.HandleFunc("GET /users/{id}/experience-bank/skills/{skill_id}/bullets", s.handleGetSkillBullets)

	// Companies endpoints
	// Note: In Go 1.22+ ServeMux, the route /companies/by-name/{name} conflicts
	// with /companies/{id}/domains because both could match /companies/by-name/domains.
	// Solution: Change /companies/by-name/{name} to use query parameter /companies/by-name?name={name}
	// This avoids the route conflict while maintaining functionality.
	mux.HandleFunc("GET /companies", s.handleListCompanies)
	mux.HandleFunc("GET /companies/by-name", s.handleGetCompanyByName) // Changed to use query parameter
	mux.HandleFunc("GET /companies/{id}", s.handleGetCompany)
	mux.HandleFunc("GET /companies/{id}/domains", s.handleListCompanyDomains)

	// Company profiles endpoints
	mux.HandleFunc("GET /companies/{company_id}/profile", s.handleGetCompanyProfile)
	mux.HandleFunc("GET /companies/{company_id}/profile/style-rules", s.handleGetStyleRules)
	mux.HandleFunc("GET /companies/{company_id}/profile/taboo-phrases", s.handleGetTabooPhrases)
	mux.HandleFunc("GET /companies/{company_id}/profile/values", s.handleGetValues)
	mux.HandleFunc("GET /companies/{company_id}/profile/sources", s.handleGetSources)

	// Job Postings endpoints
	mux.HandleFunc("GET /job-postings", s.handleListJobPostings)
	mux.HandleFunc("GET /job-postings/{id}", s.handleGetJobPosting)
	mux.HandleFunc("GET /job-postings/by-url", s.handleGetJobPostingByURL)
	mux.HandleFunc("GET /companies/{company_id}/job-postings", s.handleListJobPostingsByCompany)

	// Job Profiles endpoints
	mux.HandleFunc("GET /job-profiles/{id}", s.handleGetJobProfile)
	mux.HandleFunc("GET /job-postings/{posting_id}/profile", s.handleGetJobProfileByPostingID)
	mux.HandleFunc("GET /job-profiles/{id}/requirements", s.handleGetRequirements)
	mux.HandleFunc("GET /job-profiles/{id}/responsibilities", s.handleGetResponsibilities)
	mux.HandleFunc("GET /job-profiles/{id}/keywords", s.handleGetKeywords)

	// Crawled Pages endpoints
	mux.HandleFunc("GET /crawled-pages/{id}", s.handleGetCrawledPage)
	mux.HandleFunc("GET /crawled-pages/by-url", s.handleGetCrawledPageByURL)
	mux.HandleFunc("GET /companies/{company_id}/crawled-pages", s.handleListCrawledPagesByCompany)

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.withRateLimit(s.withLogging(s.withCORS(mux))),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second, // Long timeout for pipeline runs
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// Start begins listening for requests
func (s *Server) Start() error {
	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	// Stop rate limiter cleanup goroutine
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}

	s.db.Close()
	log.Println("Server stopped")
	return nil
}

// withCORS adds CORS headers
func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// withRateLimit adds rate limiting middleware
func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client identifier (IP address)
		clientID := s.extractClientID(r)

		// Check rate limit
		allowed, info := s.rateLimiter.Allow(clientID, r.URL.Path, r.Method)

		if !allowed {
			// Set rate limit headers
			s.setRateLimitHeaders(w, info)
			// Return 429 Too Many Requests
			s.rateLimitResponse(w, info)
			return
		}

		// Set rate limit headers for successful requests
		s.setRateLimitHeaders(w, info)
		next.ServeHTTP(w, r)
	})
}

// withLogging adds request logging
func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[%s] %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s completed in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// jsonResponse writes a JSON response
func (s *Server) jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// errorResponse writes an error JSON response
func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}

// handleRegister handles user registration requests.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	s.authHandler.Register(w, r)
}

// handleLogin handles user login requests.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	s.authHandler.Login(w, r)
}

// handleUpdatePassword handles password update requests.
func (s *Server) handleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	s.authHandler.UpdatePassword(w, r)
}

// extractClientID extracts the client identifier from the request.
// For MVP, this uses the IP address from RemoteAddr.
// In the future, this could use X-Forwarded-For header (only from trusted proxies).
func (s *Server) extractClientID(r *http.Request) string {
	// Get IP from RemoteAddr (format: "IP:port")
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If parsing fails, use the whole RemoteAddr
		return r.RemoteAddr
	}
	return ip
}

// setRateLimitHeaders sets standard rate limit headers on the response.
func (s *Server) setRateLimitHeaders(w http.ResponseWriter, info ratelimit.Info) {
	if info.Limit > 0 {
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.ResetTime.Unix()))
	}
}

// rateLimitResponse writes a 429 Too Many Requests response with rate limit information.
func (s *Server) rateLimitResponse(w http.ResponseWriter, info ratelimit.Info) {
	response := map[string]interface{}{
		"error":     "rate_limit_exceeded",
		"message":   "Rate limit exceeded. Please try again later.",
		"limit":     info.Limit,
		"remaining": info.Remaining,
		"reset_at":  info.ResetTime.Format(time.RFC3339),
	}

	if info.RetryAfter > 0 {
		response["retry_after"] = int(info.RetryAfter.Seconds())
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(info.RetryAfter.Seconds())))
	}

	// Log rate limit hit
	log.Printf("[rate-limit] Rate limit exceeded: Limit=%d Remaining=%d Reset=%s",
		info.Limit, info.Remaining, info.ResetTime.Format(time.RFC3339))

	s.jsonResponse(w, http.StatusTooManyRequests, response)
}
