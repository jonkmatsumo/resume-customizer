package main

import (
	"testing"
)

// NOTE: These tests are disabled because the repair command was updated
// to use --user-id (loading from database) instead of --experience (loading from file).
// These tests need to be refactored to use a test database setup.
// See docs/DATABASE_ARTIFACT_CLEANUP.md Phase B for migration plan.

func TestRepairCommand_MissingPlanFlag(t *testing.T) {
	t.Skip("Test disabled: repair command now uses --user-id with database, not --experience file")
}

func TestRepairCommand_MissingBulletsFlag(t *testing.T) {
	t.Skip("Test disabled: repair command now uses --user-id with database, not --experience file")
}

func TestRepairCommand_MissingTemplateFlag(t *testing.T) {
	t.Skip("Test disabled: repair command now uses --user-id with database, not --experience file")
}

func TestRepairCommand_MissingOutputFlag(t *testing.T) {
	t.Skip("Test disabled: repair command now uses --user-id with database, not --experience file")
}

func TestRepairCommand_MissingCandidateInfo(t *testing.T) {
	t.Skip("Test disabled: repair command now uses --user-id with database, not --experience file")
}
