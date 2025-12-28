package main

import (
	"testing"
)

// NOTE: These tests are disabled because the materialize command was updated
// to use --user-id (loading from database) instead of --experience (loading from file).
// These tests need to be refactored to use a test database setup.
// See docs/DATABASE_ARTIFACT_CLEANUP.md Phase B for migration plan.

func TestMaterializeCommand_MissingPlanFlag(t *testing.T) {
	t.Skip("Test disabled: materialize command now uses --user-id with database, not --experience file")
}

func TestMaterializeCommand_MissingUserIDFlag(t *testing.T) {
	t.Skip("Test disabled: materialize command now uses --user-id with database, not --experience file")
}

func TestMaterializeCommand_MissingOutputFlag(t *testing.T) {
	t.Skip("Test disabled: materialize command now uses --user-id with database, not --experience file")
}

func TestMaterializeCommand_InvalidPlanFile(t *testing.T) {
	t.Skip("Test disabled: materialize command now uses --user-id with database, not --experience file")
}

func TestMaterializeCommand_InvalidPlanJSON(t *testing.T) {
	t.Skip("Test disabled: materialize command now uses --user-id with database, not --experience file")
}

func TestMaterializeCommand_ValidInput(t *testing.T) {
	t.Skip("Test disabled: materialize command now uses --user-id with database, not --experience file")
}

func TestMaterializeCommand_StoryNotFound(t *testing.T) {
	t.Skip("Test disabled: materialize command now uses --user-id with database, not --experience file")
}

func TestMaterializeCommand_BulletNotFound(t *testing.T) {
	t.Skip("Test disabled: materialize command now uses --user-id with database, not --experience file")
}
