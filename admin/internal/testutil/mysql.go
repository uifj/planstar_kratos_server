// Package testutil holds shared test helpers for the admin module.
//
// Unit tests stay colocated with the code they cover (package foo / foo_test).
// Integration tests are colocated too but guarded by the "integration" build
// tag and obtain a real database through MySQL, which skips when no DSN is
// configured so that plain `go test ./...` never requires infrastructure.
package testutil

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLEnv names the environment variable carrying the integration-test DSN.
// It is deliberately opt-in: an unset variable skips integration tests rather
// than silently pointing at a default (possibly production) server.
const MySQLEnv = "ADMIN_TEST_MYSQL_DSN"

// MySQL returns a live connection for integration tests. It skips the test when
// MySQLEnv is unset or the server cannot be reached, and closes the connection
// during test cleanup. Queries should use fully-qualified database names.
func MySQL(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv(MySQLEnv)
	if dsn == "" {
		t.Skipf("integration: set %s to run against a real MySQL", MySQLEnv)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("integration: open MySQL: %v", err)
	}
	db.SetMaxOpenConns(4)
	db.SetConnMaxLifetime(time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("integration: MySQL unreachable: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// Exec runs a statement and fails the test on error.
func Exec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// Cleanup registers a statement (typically a DELETE of seeded rows) to run after
// the test, keeping integration runs reversible.
func Cleanup(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	t.Cleanup(func() { _, _ = db.Exec(query, args...) })
}
