// Package testutil holds shared test helpers for the planstar module.
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

	"github.com/go-sql-driver/mysql"
)

// MySQLEnv names the environment variable carrying the integration-test DSN.
// It is deliberately opt-in: an unset variable skips integration tests rather
// than silently pointing at a default (possibly production) server.
const MySQLEnv = "PLANSTAR_TEST_MYSQL_DSN"

// MySQL returns a live connection using the DSN's default database, for tests
// whose statements are fully qualified. It skips when MySQLEnv is unset or the
// server is unreachable, and closes the connection during test cleanup.
func MySQL(t *testing.T) *sql.DB {
	t.Helper()
	return open(t, "")
}

// MySQLDatabase returns a live connection whose default database is name. Repos
// that issue unqualified statements against a per-domain pool (community, shop,
// payment, order) need one connection per domain, exactly as production wires
// them, so cross-domain behaviour is reproduced faithfully.
func MySQLDatabase(t *testing.T, name string) *sql.DB {
	t.Helper()
	return open(t, name)
}

func open(t *testing.T, database string) *sql.DB {
	t.Helper()
	dsn := os.Getenv(MySQLEnv)
	if dsn == "" {
		t.Skipf("integration: set %s to run against a real MySQL", MySQLEnv)
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Skipf("integration: parse %s: %v", MySQLEnv, err)
	}
	if database != "" {
		cfg.DBName = database
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
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
