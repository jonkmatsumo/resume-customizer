.PHONY: build test lint fmt clean build-clean docker-up docker-down docker-db test-jobs test-profiles test-companies test-experience test-artifacts test-research

# =============================================================================
# Local Development
# =============================================================================

# Build the binary locally
build:
	go build -o bin/resume_agent ./cmd/resume_agent

# Run all tests
test:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -race ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run unit tests only (no database required)
test-unit:
	go test -v -short ./...

# Run integration tests (requires TEST_DATABASE_URL)
test-integration:
	go test -v -tags=integration ./...

# Run companies-related tests
test-companies:
	go test -v ./internal/db/... -run 'Company|CrawledPage|Normalize|Hash|Extract'

# Run profile-related tests (Phase 2)
test-profiles:
	go test -v ./internal/db/... -run 'Profile|Signal|Source|Taboo'

# Run job posting-related tests (Phase 3)
test-jobs:
	go test -v ./internal/db/... -run 'Job|Posting|Requirement|Keyword'

# Run experience bank-related tests (Phase 4)
test-experience:
	go test -v ./internal/db/... -run 'Skill|Story|Bullet|Experience|Highlight'

# Run pipeline artifacts-related tests (Phase 5)
test-artifacts:
	go test -v ./internal/db/... -run 'RunRanked|RunResume|RunSelected|RunRewritten|RunViolation|Severity|Section'

# Run research-related tests (Phase 6)
test-research:
	go test -v ./internal/db/... -run 'Research|Frontier|BrandSignal|PageType'

# Linting
lint:
	@golangci-lint run

# Format code
fmt:
	@go fmt ./...
	@goimports -w .

# Install dependencies
deps:
	go mod tidy
	go mod download

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Clean Go cache and build
build-clean:
	go clean -cache
	$(MAKE) build

# CI checks
ci: fmt lint test build
	@echo "All CI checks passed!"

# =============================================================================
# Docker Commands
# =============================================================================

# Start database and app containers
docker-up:
	docker compose up -d

# Stop and remove containers (keeps data)
docker-down:
	docker compose down

# Stop and remove containers AND data
docker-reset:
	docker compose down -v
	docker compose up -d

# Rebuild app container
docker-build:
	docker compose build --no-cache app

# Open database shell
docker-db:
	docker compose exec db psql -U resume -d resume_customizer

# Show artifacts in database
docker-artifacts:
	docker compose exec db psql -U resume -d resume_customizer \
		-c "SELECT step, category FROM artifacts ORDER BY created_at;"

# Show pipeline runs
docker-runs:
	docker compose exec db psql -U resume -d resume_customizer \
		-c "SELECT id, company, role_title, status, created_at FROM pipeline_runs ORDER BY created_at DESC;"

