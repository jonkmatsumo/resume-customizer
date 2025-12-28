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

## 2. Architecture

At a high level, the system runs as a containerized Go application with PostgreSQL for artifact persistence:

```mermaid
flowchart LR
    subgraph Docker
        DB[(PostgreSQL)]
        APP[resume_agent]
    end
    
    LLM[Gemini API] <--> APP
    JOB[Job URL] --> APP
    EXP[Experience Bank] --> APP
    APP --> DB
    APP --> TEX[resume.tex]
```

The system uses hybrid ranking (deterministic heuristics + LLM semantic evaluation), validation loops that compile LaTeX and check the PDF, and persists every artifact to PostgreSQL for debugging. Under the hood, the pipeline orchestrates specialized agents that pass validated data between stages—green nodes below indicate LLM-powered steps:

```mermaid
flowchart TD
    %% ---------------- USER INPUTS ----------------
    JOB[Job Posting]
    EXP[Experience Bank]

    %% ---------------- RESEARCH SUBSYSTEM ----------------
    subgraph "Research"
        direction TB
        
        subgraph "Ingestion"
            JOB --> B[Ingest & Clean]
            B --> C[LLM: Extract Structure]
            
            C --> REQ[Requirements & Responsibilities]
            C --> EDU_REQ[LLM: Education Requirements]
            C --> T[Team/Cultural Notes]
            C --> S[Extracted Links]
        end
        
        subgraph "Company Research"
            S --> DI[LLM: Identify Company Domains]
            DI --> DF[LLM: Filter to Company Domains]
            
            DF --> SRCH{Search API?}
            SRCH -->|Yes| GS[Google Search]
            SRCH -->|No| PAT[Pattern URLs]
            GS --> FL[LLM: Filter & Prioritize]
            PAT --> FL
            
            FL --> FR[URL Frontier]
            FR --> FE[Fetch Pages]
            FE --> LIMIT{Page Limit?}
            LIMIT -->|No| EX[LLM: Extract Signals]
            EX --> FR
            LIMIT -->|Yes| AG[Aggregate Corpus]
            
            T -.->|Context| AG
            AG --> SV[LLM: Summarize Voice]
            SV --> CP[Company Profile]
        end
    end

    %% ---------------- GENERATION SUBSYSTEM ----------------
    subgraph "Resume Generation"
        direction TB
        
        subgraph "Planning"
            EXP --> RK[LLM: Rank Stories]
            REQ --> RK
            RK --> RS[Ranked Stories]
            
            EXP --> ES[LLM: Score Education]
            EDU_REQ --> ES
            ES --> SE[Selected Education]
            
            RS --> SP[Select Optimum Plan]
            SP --> PLAN[Resume Plan]
            PLAN --> MAT[Materialize Bullets]
        end
        
        subgraph "Drafting & Refining"
            MAT --> RW[LLM: Rewrite Bullets]
            CP -.->|Voice| RW
            REQ -.->|Keywords| RW
            
            RW --> TEX[Render LaTeX]
            SE --> TEX
            TEX --> PDF[Compile PDF]
            PDF --> VAL[Validate Constraints]
            
            VAL --> VIO{Violations?}
            VIO -->|No| FIN[✅ Final Resume]
            VIO -->|Yes| RL[LLM: Repair Plan]
            
            RL -->|Updates| MAT
        end
    end
    
    %% Styling
    classDef input fill:#6f8bb3,stroke:#333,stroke-width:2px,color:#000;
    classDef llm fill:#88b090,stroke:#333,stroke-width:2px,color:#000;
    classDef tool fill:#a0aab5,stroke:#333,stroke-width:1px,color:#000;
    classDef data fill:#bba6c7,stroke:#666,stroke-width:1px,stroke-dasharray: 5 5,color:#000;
    
    class C,DI,DF,FL,EX,SV,RW,RL,RK,ES,EDU_REQ llm;
    class B,FE,GS,PAT,AG,SP,MAT,TEX,PDF,VAL tool;
    class REQ,T,S,FR,CP,RS,PLAN,FIN,SE data;
    class JOB,EXP input;
```

---

## 4. Quick Start with Docker

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

## 5. Configuration

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

## 6. Database & Artifact Storage

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

## 7. Development

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
