# Resume Customizer

A **schema-first, CLI-driven, multi-step agent** that generates a **strictly formatted, one-page LaTeX resume** tailored to a specific job posting and company brand voice.

This system is designed for **incremental development, determinism, and debuggability**—not free-form resume writing.

## Quick Start

### Prerequisites

- Go 1.21 or later
- Make (optional, for convenience commands)

### Build

```bash
# Install dependencies
go mod tidy

# Build binary
make build
# or
go build -o bin/resume_agent ./cmd/resume_agent
```

### Validate JSON Against Schema

```bash
./bin/resume_agent validate \
  --schema schemas/job_profile.schema.json \
  --json testdata/valid/job_profile.json
```

## CLI Usage

### Validate Command

Validate a JSON file against a JSON Schema:

```bash
resume_agent validate --schema <schema_path> --json <json_path>
```

**Flags:**
- `--schema`, `-s`: Path to JSON Schema file (required)
- `--json`, `-j`: Path to JSON file to validate (required)

**Exit Codes:**
- `0`: Validation passed
- `1`: Validation failed (returns `ValidationError`)
- `2`: Usage error or schema loading error (missing flags, file not found, invalid schema, etc.)

**Examples:**

```bash
# Validate a valid job profile
resume_agent validate \
  --schema schemas/job_profile.schema.json \
  --json testdata/valid/job_profile.json

# Validate invalid JSON (will show validation errors)
resume_agent validate \
  --schema schemas/job_profile.schema.json \
  --json testdata/invalid/missing_field.json
```

## Schema Files

All JSON schemas are located in the `schemas/` directory:

- `common.schema.json`: Shared definitions (Skill, SkillTargets, Requirement, Evidence, EvalSignals)
- `job_profile.schema.json`: Job posting structure with inline definitions
- `company_profile.schema.json`: Company brand voice and style rules
- `experience_bank.schema.json`: Canonical store of reusable experience stories
- `ranked_stories.schema.json`: Deterministically scored candidate stories
- `resume_plan.schema.json`: Selection contract defining which stories and bullets to use
- `bullets.schema.json`: Selected and rewritten bullet points
- `violations.schema.json`: Structured validation failures
- `repair_actions.schema.json`: Structured fixes for violations
- `state.schema.json`: Full pipeline state (optional, for DAG orchestration)

**Note:** Schemas use internal `$defs` references (e.g., `#/$defs/Requirement`) rather than external file references to avoid HTTP resolution issues.

## Project Structure

```
resume-customizer/
├── cmd/resume_agent/          # CLI entrypoint
│   ├── main.go                # Root command and validate command
│   └── validate_test.go       # CLI tests
├── internal/
│   └── schemas/               # Schema validation logic
│       ├── validate.go        # Validation functions and error types
│       ├── validate_test.go   # Validation unit tests
│       └── testdata/          # Test fixtures
├── pkg/types/                 # Go type definitions (for future use)
├── schemas/                   # JSON Schema files
│   └── schemas_test.go        # Schema file validation tests
├── testdata/                  # Test fixtures
│   ├── valid/                 # Valid JSON examples
│   └── invalid/               # Invalid JSON examples
├── artifacts/                 # Output directory (gitignored)
├── docs/                      # Design documents (gitignored)
├── go.mod                     # Go module definition
├── Makefile                   # Build and test commands
└── README.md                  # This file
```

## Development

### Prerequisites

Install development tools:

```bash
# Install goimports (for formatting)
go install golang.org/x/tools/cmd/goimports@latest

# Install golangci-lint (for linting)
# macOS
brew install golangci-lint
# or via script
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2

# Install pre-commit (optional, for git hooks)
brew install pre-commit
# or
pip install pre-commit
```

### Pre-commit Hooks (Optional)

Set up git hooks to run linting and formatting before commits:

```bash
# Install pre-commit hooks
pre-commit install

# Run on all files (optional)
pre-commit run --all-files
```

Hooks will automatically run on `git commit`. You can skip them with `git commit --no-verify` if needed.

### Local Development Workflow

```bash
# Format code
make fmt

# Check formatting (CI use)
make fmt-check

# Run linter
make lint

# Run all tests
make test

# Run tests with race detector
make test-race

# Run tests with coverage
make test-coverage

# Run all CI checks locally
make ci

# Build binary
make build

# Validate schema files
make validate-schemas
```

### CI/CD

The project uses GitHub Actions for continuous integration. The CI pipeline runs on every push and pull request:

1. **Lint**: Runs `golangci-lint` to check code quality
2. **Format Check**: Verifies code is properly formatted
3. **Test**: Runs all tests with race detector and coverage
4. **Build**: Verifies code compiles successfully
5. **Schema Validation**: Validates all JSON schema files

All checks must pass for PRs to be mergeable. See `.github/workflows/ci.yml` for details.

## Error Types

The validation system uses structured error types:

- **`ValidationError`**: Returned when JSON fails schema validation. Contains field-level error details.
- **`SchemaLoadError`**: Returned when the schema itself cannot be loaded or parsed (invalid schema syntax, missing referenced definitions, etc.).

All errors implement the standard Go `error` interface and can be unwrapped for inspection.

## Core Principles

1. **Schema-First**: Every artifact must conform to a JSON Schema in `/schemas`. If it doesn't validate from the CLI, it's incorrect.

2. **CLI-First**: Every feature runs as a standalone CLI command with explicit inputs/outputs before any orchestration.

3. **Determinism**: Prefer deterministic logic, explicit heuristics, and explainable scoring over LLM guessing.

4. **LLM Boundaries**: LLMs only handle text interpretation, classification, and rewriting under constraints. They never browse the web, invent facts, or bypass validation.

5. **No Silent Fixes**: All repairs must be explicit and auditable via structured `RepairActions`.

## Architecture

See `docs/DESIGN.md` for detailed architecture documentation and `docs/IMPLEMENTATION_PLAN.md` for the implementation roadmap.
