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

### Running Tests

```bash
# Run all tests
make test
# or
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/schemas/...
```

### Building

```bash
# Build binary
make build

# Clean build artifacts
make clean
```

### Schema Validation

```bash
# Validate all schema files are valid JSON
make validate-schemas
```

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
