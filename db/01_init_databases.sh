#!/bin/bash
set -e

# This script creates the 'resume' database if it doesn't exist
# The POSTGRES_DB environment variable creates resume_customizer by default
# We need to ensure 'resume' also exists for backward compatibility

# Check if resume database exists, and create it if it doesn't
# Connect to postgres database (which always exists) to create other databases
if ! psql -v ON_ERROR_STOP=0 --username "$POSTGRES_USER" --dbname "postgres" -tc "SELECT 1 FROM pg_database WHERE datname = 'resume'" | grep -q 1; then
    echo "Creating 'resume' database..."
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "postgres" -c "CREATE DATABASE resume"
    echo "Database 'resume' created successfully"
else
    echo "Database 'resume' already exists, skipping creation"
fi

