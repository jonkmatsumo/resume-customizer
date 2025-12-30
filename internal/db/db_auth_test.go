package db

// Note: Unit tests for authentication repository methods are not included here
// because these methods require database access. All testing is done via
// integration tests in db_auth_integration_test.go which test:
// - GetUserByEmail: success, not found, error cases
// - UpdatePassword: success, user not found, timestamp updates
// - CheckEmailExists: existing email, non-existent email, empty email
//
// These methods follow the same patterns as other DB methods and are
// comprehensively tested in the integration test file.
