.PHONY: build test test-failures validate-schemas clean lint fmt fmt-check test-race test-coverage ci \
	resume-validate resume-ingest-job resume-parse-job resume-load-experience \
	resume-build-skill-targets resume-rank-stories resume-plan resume-materialize \
	resume-crawl-brand resume-summarize-voice resume-rewrite resume-render-latex \
	resume-validate-latex resume-repair

# Build the binary
build:
	go build -o bin/resume_agent ./cmd/resume_agent

# Run all tests
test:
	go test -v ./...

# Show only failing tests
test-failures:
	@make test 2>&1 | grep -A 5 "FAIL" || true

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detector
test-race:
	go test -race ./...

# Linting
lint:
	@golangci-lint run

# Formatting
fmt:
	@go fmt ./...
	@goimports -w .

fmt-check:
	@if [ $$(gofmt -s -l . | wc -l) -gt 0 ]; then \
		echo "Code is not formatted. Run 'make fmt' to fix."; \
		gofmt -s -d .; \
		exit 1; \
	fi
	@if [ $$(goimports -l . | wc -l) -gt 0 ]; then \
		echo "Imports are not formatted. Run 'make fmt' to fix."; \
		goimports -d .; \
		exit 1; \
	fi

# Validate all schema files are valid JSON
validate-schemas:
	@echo "Validating schema files..."
	@for schema in schemas/*.json; do \
		echo "Checking $$schema..."; \
		python3 -m json.tool "$$schema" > /dev/null || (echo "$$schema is invalid JSON" && exit 1); \
	done
	@echo "All schema files are valid JSON"

# CI checks (run locally)
ci: fmt-check lint test validate-schemas build
	@echo "All CI checks passed!"

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install dependencies
deps:
	go mod tidy
	go mod download

# Resume Agent CLI command aliases
# Usage: make resume-<command> ARGS="--flag value ..."
# Example: make resume-plan ARGS="--ranked ranked.json --experience exp.json --out plan.json"

BINARY := ./bin/resume_agent

resume-validate:
	@$(BINARY) validate $(ARGS)

resume-ingest-job:
	@$(BINARY) ingest-job $(ARGS)

resume-parse-job:
	@$(BINARY) parse-job $(ARGS)

resume-load-experience:
	@$(BINARY) load-experience $(ARGS)

resume-build-skill-targets:
	@$(BINARY) build-skill-targets $(ARGS)

resume-rank-stories:
	@$(BINARY) rank-stories $(ARGS)

resume-plan:
	@$(BINARY) plan $(ARGS)

resume-materialize:
	@$(BINARY) materialize $(ARGS)

resume-crawl-brand:
	@$(BINARY) crawl-brand $(ARGS)

resume-summarize-voice:
	@$(BINARY) summarize-voice $(ARGS)

resume-rewrite:
	@$(BINARY) rewrite $(ARGS)

resume-render-latex:
	@$(BINARY) render-latex $(ARGS)

resume-validate-latex:
	@$(BINARY) validate-latex $(ARGS)

resume-repair:
	@$(BINARY) repair $(ARGS)

