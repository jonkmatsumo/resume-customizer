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

## 2. The Multi-Agent Workflow

The pipeline orchestrates specialized agents that pass validated data between stages. 

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
            
            C --> D[Requirements]
            C --> E[Responsibilities]
            C --> S[Extracted Links]
            C --> T[Team/Cultural Notes]
            C --> EDU_REQ[LLM: Education Requirements]
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
            
            T -.-|Context| AG
            AG --> SV[LLM: Summarize Voice]
            SV --> CP[Company Profile]
        end
    end

    %% ---------------- GENERATION SUBSYSTEM ----------------
    subgraph "Resume Generation"
        direction TB
        
        subgraph "Planning"
            EXP --> RK[LLM: Rank Stories]
            D --> RK
            E --> RK
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
            CP -.-|Voice| RW
            D -.-|Keywords| RW
            
            RW --> TEX[Render LaTeX]
            SE --> TEX
            TEX --> PDF[Compile PDF]
            PDF --> VAL[Validate Constraints]
            
            VAL --> VIO{Violations?}
            VIO -->|No| FIN[✅ Final Resume]
            VIO -->|Yes| RL[LLM: Repair Plan]
            
            RL -->|Updates| PLAN
            RL -->|Edits| RW
        end
    end
    
    %% Styling
    classDef input fill:#6f8bb3,stroke:#333,stroke-width:2px,color:#000;
    classDef llm fill:#88b090,stroke:#333,stroke-width:2px,color:#000;
    classDef tool fill:#a0aab5,stroke:#333,stroke-width:1px,color:#000;
    classDef data fill:#bba6c7,stroke:#666,stroke-width:1px,stroke-dasharray: 5 5,color:#000;
    
    class C,DI,DF,FL,EX,SV,RW,RL,RK,ES,EDU_REQ llm;
    class B,FE,GS,PAT,AG,SP,MAT,TEX,PDF,VAL tool;
    class D,E,T,S,FR,CP,RS,PLAN,FIN,SE data;
    class JOB,EXP input;
```

**Data Flow:**
- **JobProfile** combines: Requirements, Responsibilities, Level/Signals, Team Notes
- **Team Notes** are shared with Company Research for context enrichment
- **Company Profile** feeds into bullet rewriting for voice matching


---

## 3. Setup and Basic Usage

### Prerequisites
*   **Go 1.21+**
*   **Google Gemini API Key**: [Get it here](https://makersuite.google.com/app/apikey).
*   **LaTeX Distribution**: (e.g., TeX Live) for document compilation.
*   **pdfinfo/ghostscript**: For page-count validation.

### Setup
```bash
# 1. Install dependencies
go mod tidy

# 2. Build the tool
make build

# 3. Configure environment
cp .env.example .env
# Edit .env and add your GEMINI_API_KEY
```

### Full Pipeline Run
Use the `run` command to process all stages from ingestion to final PDF generation.

**Option 1: Using a config file (recommended)**
```bash
# Copy and customize the example config
cp config.example.json config.json

# Run with config file
./bin/resume_agent run --config config.json
```

**Option 2: Using command-line flags**
```bash
make resume-run ARGS="--job job.txt \
                     --experience history.json \
                     --out output/ \
                     --name \"Jane Smith\" \
                     --email \"jane@example.com\""
```

**Option 3: Mixed (config + overrides)**
```bash
# Config provides defaults, CLI flags override specific values
./bin/resume_agent run --config config.json --job-url https://... --verbose
```

### Config File Reference

Create a `config.json` file to store common settings:

```json
{
  "job_url": "https://job-boards.greenhouse.io/company/jobs/12345",
  "experience": "history.json",
  "output": "artifacts/",
  "name": "Jane Smith",
  "email": "jane@example.com",
  "max_bullets": 25,
  "max_lines": 35
}
```

| Field | Description |
|-------|-------------|
| `job` | Path to job posting text file (mutually exclusive with `job_url`) |
| `job_url` | URL to fetch job posting from (mutually exclusive with `job`) |
| `experience` | Path to experience bank JSON file |
| `output` | Output directory |
| `template` | Path to LaTeX template |
| `name`, `email`, `phone` | Candidate contact info |
| `max_bullets`, `max_lines` | Layout constraints |
| `api_key` | Gemini API key (or use `GEMINI_API_KEY` env var) |
| `verbose` | Enable debug logging |

---

## 4. Detailed Step-by-Step Breakdown

Each stage of the pipeline can be run individually for debugging or manual adjustments.

### Stage 1: Job Ingestion
Cleans raw text or HTML into a standard text format.
*   **Command**: `make resume-ingest-job ARGS="--text-file posting.txt --out artifacts/"`

### Stage 2: Job Parsing
Extracts structured requirements and skills.
*   **Command**: `make resume-parse-job ARGS="--in artifacts/job_posting.cleaned.txt --out artifacts/job_profile.json"`

### Stage 3: Experience Normalization
Ensures professional history is in the standard format used by the tool.
*   **Command**: `make resume-load-experience ARGS="--in history.json --out artifacts/history_normalized.json"`

### Stage 4: Experience Ranking
Scores professional stories against the parsed job requirements.
*   **Command**: `make resume-rank-stories ARGS="--job-profile artifacts/job_profile.json --experience artifacts/history_normalized.json --out artifacts/ranked_stories.json"`

### Stage 5: Layout Planning
Selects the optimal combination of bullets to fit a one-page layout.
*   **Command**: `make resume-plan ARGS="--ranked artifacts/ranked_stories.json --job-profile artifacts/job_profile.json --experience artifacts/history_normalized.json --max-bullets 8 --max-lines 45 --out artifacts/resume_plan.json"`

### Stage 6: Brand Voice Extraction
Researches the company tone and preferred communication style.
*   **Command**: `make resume-crawl-brand ARGS="--seed-url https://company.com --out artifacts/"`
*   **Next**: `make resume-summarize-voice ARGS="--in artifacts/company_corpus.txt --sources artifacts/sources.json --out artifacts/brand_profile.json"`

### Stage 7: Bullet Rewriting
Tailors bullet point text for keyword relevance and tone.
*   **Command**: `make resume-rewrite ARGS="--selected bullets.json --job-profile job.json --company-profile brand_profile.json --out tailored.json"`

### Stage 8: Rendering & Validation
Generates the LaTeX file and checks for formatting violations.
*   **Command**: `make resume-render-latex ARGS="--plan plan.json --bullets rewritten.json --experience history.json --out artifacts/resume.tex"`
*   **Validation**: `make resume-validate-latex ARGS="--in artifacts/resume.tex --out artifacts/violations.json"`

### Stage 9: Formatting Repair
Iteratively adjusts content to fix layout overflows or style violations.
*   **Command**: `make resume-repair ARGS="--plan plan.json --bullets rewritten.json --violations violations.json --job-profile job.json --out artifacts/"`

---

## 5. Technical Context and Constraints

### Architecture
Resume Customizer uses a **hybrid architecture** that combines deterministic algorithms with large language models (LLMs).
*   **Hybrid Ranking**: Story ranking uses both deterministic heuristics (skill overlap, keywords, recency) and LLM semantic evaluation, weighted 50/50 for optimal relevance scoring.
*   **LLM Integration**: LLMs are used for semantic tasks like summarization, relevance judging, and content tailoring.
*   **Validation Loops**: The system verifies its own output by compiling the LaTeX source and checking the resulting PDF against physical constraints.

### Key Constraints
*   **Page Budget**: The system enforces a strict one-page limit for the final document.
*   **Data Integrity**: Every intermediate file is validated against a JSON Schema to ensure data consistency between stages.
*   **Transparency**: Every step produces a readable JSON file, allowing users to inspect the system's reasoning at any point.

### Development

```bash
make test      # Run unit and integration tests
make lint      # Run static analysis
make ci        # Run all quality checks
```
