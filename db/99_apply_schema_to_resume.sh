#!/bin/bash
set -e

# This script applies the same schema to the 'resume' database
# It runs after all SQL files have been executed against resume_customizer
# This ensures both databases have the same schema

# List of SQL files to apply (same ones that were applied to resume_customizer)
SQL_FILES=(
    "users.sql"
    "companies.sql"
    "company_profiles.sql"
    "job_postings.sql"
    "experience_bank.sql"
    "pipeline_artifacts.sql"
    "research.sql"
    "resumes.sql"
    "run_steps.sql"
)

# Apply each SQL file to the resume database
for sql_file in "${SQL_FILES[@]}"; do
    if [ -f "/docker-entrypoint-initdb.d/$sql_file" ]; then
        echo "Applying $sql_file to 'resume' database..."
        psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "resume" -f "/docker-entrypoint-initdb.d/$sql_file"
    fi
done

echo "Schema successfully applied to 'resume' database"

