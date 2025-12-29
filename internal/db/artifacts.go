package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jonathan/resume-customizer/internal/types"
)

// GetJobProfileByRunID loads job profile from database for a run
func (db *DB) GetJobProfileByRunID(ctx context.Context, runID uuid.UUID) (*types.JobProfile, error) {
	content, err := db.GetArtifact(ctx, runID, StepJobProfile)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}

	var profile types.JobProfile
	if err := json.Unmarshal(content, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job profile: %w", err)
	}
	return &profile, nil
}

// GetRankedStoriesByRunID loads ranked stories from database for a run
func (db *DB) GetRankedStoriesByRunID(ctx context.Context, runID uuid.UUID) (*types.RankedStories, error) {
	content, err := db.GetArtifact(ctx, runID, StepRankedStories)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}

	var stories types.RankedStories
	if err := json.Unmarshal(content, &stories); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ranked stories: %w", err)
	}
	return &stories, nil
}

// GetResumePlanByRunID loads resume plan from database for a run
func (db *DB) GetResumePlanByRunID(ctx context.Context, runID uuid.UUID) (*types.ResumePlan, error) {
	content, err := db.GetArtifact(ctx, runID, StepResumePlan)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}

	var plan types.ResumePlan
	if err := json.Unmarshal(content, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resume plan: %w", err)
	}
	return &plan, nil
}

// GetSelectedBulletsByRunID loads selected bullets from database for a run
func (db *DB) GetSelectedBulletsByRunID(ctx context.Context, runID uuid.UUID) (*types.SelectedBullets, error) {
	content, err := db.GetArtifact(ctx, runID, StepSelectedBullets)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}

	var bullets types.SelectedBullets
	if err := json.Unmarshal(content, &bullets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal selected bullets: %w", err)
	}
	return &bullets, nil
}

// GetRewrittenBulletsByRunID loads rewritten bullets from database for a run
func (db *DB) GetRewrittenBulletsByRunID(ctx context.Context, runID uuid.UUID) (*types.RewrittenBullets, error) {
	content, err := db.GetArtifact(ctx, runID, StepRewrittenBullets)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}

	var bullets types.RewrittenBullets
	if err := json.Unmarshal(content, &bullets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rewritten bullets: %w", err)
	}
	return &bullets, nil
}

// GetCompanyProfileByRunID loads company profile from database for a run
func (db *DB) GetCompanyProfileByRunID(ctx context.Context, runID uuid.UUID) (*types.CompanyProfile, error) {
	content, err := db.GetArtifact(ctx, runID, StepCompanyProfile)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}

	var profile types.CompanyProfile
	if err := json.Unmarshal(content, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal company profile: %w", err)
	}
	return &profile, nil
}

// GetJobMetadataByRunID loads job metadata from database for a run
// Returns raw JSON bytes to avoid import cycle with ingestion package
func (db *DB) GetJobMetadataByRunID(ctx context.Context, runID uuid.UUID) ([]byte, error) {
	content, err := db.GetArtifact(ctx, runID, StepJobMetadata)
	if err != nil {
		return nil, err
	}
	return content, nil
}

// GetViolationsByRunID loads violations from database for a run
func (db *DB) GetViolationsByRunID(ctx context.Context, runID uuid.UUID) (*types.Violations, error) {
	content, err := db.GetArtifact(ctx, runID, StepViolations)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}

	var violations types.Violations
	if err := json.Unmarshal(content, &violations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal violations: %w", err)
	}
	return &violations, nil
}
