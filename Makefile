.PHONY: build test validate-schemas clean lint fmt fmt-check test-race test-coverage ci

# Build the binary
build:
	go build -o bin/resume_agent ./cmd/resume_agent

# Run all tests
test:
	go test -v ./...

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

