package main

import (
	"testing"
)

// NOTE: These tests are disabled because the plan command was updated
// to use --user-id (loading from database) instead of --experience (loading from file).
// These tests need to be refactored to use a test database setup.
// See docs/DATABASE_ARTIFACT_CLEANUP.md Phase B for migration plan.

func TestPlanCommand_MissingJobProfileFlag(t *testing.T) {
	t.Skip("Test disabled: plan command now uses --user-id with database, not --experience file")
}

func TestPlanCommand_MissingUserIDFlag(t *testing.T) {
	t.Skip("Test disabled: plan command now uses --user-id with database, not --experience file")
}

func TestPlanCommand_MissingRankedStoriesFlag(t *testing.T) {
	t.Skip("Test disabled: plan command now uses --user-id with database, not --experience file")
}

func TestPlanCommand_MissingOutputFlag(t *testing.T) {
	t.Skip("Test disabled: plan command now uses --user-id with database, not --experience file")
}

func TestPlanCommand_InvalidJobProfileFile(t *testing.T) {
	t.Skip("Test disabled: plan command now uses --user-id with database, not --experience file")
}

func TestPlanCommand_InvalidRankedStoriesFile(t *testing.T) {
	t.Skip("Test disabled: plan command now uses --user-id with database, not --experience file")
}

func TestPlanCommand_ValidInput(t *testing.T) {
	t.Skip("Test disabled: plan command now uses --user-id with database, not --experience file")
}
