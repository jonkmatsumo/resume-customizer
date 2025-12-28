# Resume Customizer

## 1. Introduction

Resume Customizer is a multi-agentic system that automates the tailoring of resumes to specific job postings. It processes a job description, researches the target company, and selects relevant experiences from a user's professional history to generate a focused, one-page LaTeX resume.

The system utilizes specialized AI agents to handle different stages of the process:
*   **Requirement Extraction**: Identifies key skills and qualifications from job descriptions.
*   **Company Research**: Analyzes company websites to understand their tone and values.
*   **Experience Selection**: Matches professional stories to job requirements based on relevance.
*   **Content Tailoring**: Rewrites bullet points to align with job keywords and company style.
*   **Layout Validation**: Checks that the document fits on one page and adheres to formatting rules.

---

## 2. Quick Start with Docker

### Prerequisites
*   **Docker Desktop** (includes Docker Compose)
*   **Google Gemini API Key**: [Get it here](https://makersuite.google.com/app/apikey)
*   *(Optional)* **Google Search API** for company research: [Custom Search JSON API](https://developers.google.com/custom-search/v1/overview)

### Setup

```bash
# 1. Clone and configure environment
cp .env.example .env
# Edit .env and add your GEMINI_API_KEY

# 2. Start the database and build the app
docker compose up -d
```

### Run the Pipeline

```bash
# Create a config file
cp config.example.json config.json
# Edit config.json with your job URL, experience file, etc.

# Run the pipeline
docker compose run --rm app run --config /app/config.json --verbose
```

### Verify Artifacts in Database

```bash
# List all artifacts from the run
docker compose exec db psql -U resume -d resume_customizer \
  -c "SELECT step, category FROM artifacts ORDER BY created_at;"

# View pipeline run status
docker compose exec db psql -U resume -d resume_customizer \
  -c "SELECT company, role_title, status FROM pipeline_runs;"

# Query artifacts by category
docker compose exec db psql -U resume -d resume_customizer \
  -c "SELECT step FROM artifacts WHERE category = 'research';"
```

---

## 3. Configuration

### Config File Reference

Create a `config.json` file for your settings:

```json
{
  "job_url": "https://job-boards.greenhouse.io/company/jobs/12345",
  "experience": "history.json",
  "out": "artifacts/",
  "name": "Jane Smith",
  "email": "jane@example.com",
  "max_bullets": 25,
  "max_lines": 35
}
```

| Field | Description |
|-------|-------------|
| `job` | Path to job posting text file (mutually exclusive with `job_url`) |
| `job_url` | URL to fetch job posting from |
| `experience` | Path to experience bank JSON file |
| `out` | Output directory |
| `template` | Path to LaTeX template |
| `name`, `email`, `phone` | Candidate contact info |
| `max_bullets`, `max_lines` | Layout constraints |
| `verbose` | Enable debug logging |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GEMINI_API_KEY` | Required. Google Gemini API key |
| `GOOGLE_SEARCH_API_KEY` | Optional. Enables company website discovery |
| `GOOGLE_SEARCH_CX` | Optional. Custom Search Engine ID |
| `DATABASE_URL` | Auto-set in Docker. For local runs: `postgres://resume:resume_dev@localhost:5432/resume_customizer?sslmode=disable` |

---

## 4. Database & Artifact Storage

All pipeline artifacts are persisted to PostgreSQL for history and debugging.

### Artifact Categories

| Category | Artifacts |
|----------|-----------|
| ingestion | job_posting, job_metadata, job_profile, education_requirements |
| experience | experience_bank, ranked_stories, education_scores, resume_plan, selected_bullets |
| research | sources, company_corpus, research_session, company_profile |
| rewriting | rewritten_bullets |
| validation | resume_tex, violations |

### Useful Queries

```sql
-- Get all artifacts for a specific run
SELECT step, category, created_at FROM artifacts 
WHERE run_id = 'your-uuid' ORDER BY created_at;

-- Find runs for a company
SELECT * FROM pipeline_runs WHERE company ILIKE '%google%';

-- Compare job profiles across runs
SELECT pr.company, a.content->>'role_title'
FROM artifacts a 
JOIN pipeline_runs pr ON a.run_id = pr.id 
WHERE a.step = 'job_profile';
```

---

## 5. Development

### Local Development (without Docker)

```bash
# Install Go dependencies
go mod tidy

# Build locally
make build

# Run with local binary + Docker database
./bin/resume_agent run --config config.json \
  --db-url "postgres://resume:resume_dev@localhost:5432/resume_customizer?sslmode=disable"
```

### Testing & Linting

```bash
make test      # Run unit tests
make lint      # Run static analysis
make fmt       # Format code
make ci        # Run all quality checks
```

### Rebuild Docker Image

```bash
docker compose build --no-cache app
```

### Reset Database

```bash
docker compose down -v  # Removes volume
docker compose up -d    # Fresh schema
```

---

## 6. Architecture

```mermaid
flowchart LR
    subgraph Docker
        DB[(PostgreSQL)]
        APP[resume_agent]
    end
    
    JOB[Job URL] --> APP
    EXP[Experience Bank] --> APP
    APP --> DB
    APP --> TEX[resume.tex]
```

The pipeline orchestrates specialized agents that pass validated data between stages:
*   **Hybrid Ranking**: Story ranking uses both deterministic heuristics and LLM semantic evaluation.
*   **Validation Loops**: The system verifies its own output by compiling LaTeX and checking the PDF.
*   **Artifact Persistence**: Every step saves to PostgreSQL for debugging and history.
