.PHONY: build test validate-schemas clean

# Build the binary
build:
	go build -o bin/resume_agent ./cmd/resume_agent

# Run all tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Validate all schema files are valid JSON
validate-schemas:
	@echo "Validating schema files..."
	@for schema in schemas/*.json; do \
		echo "Checking $$schema..."; \
		python3 -m json.tool "$$schema" > /dev/null || (echo "$$schema is invalid JSON" && exit 1); \
	done
	@echo "All schema files are valid JSON"

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install dependencies
deps:
	go mod tidy
	go mod download

