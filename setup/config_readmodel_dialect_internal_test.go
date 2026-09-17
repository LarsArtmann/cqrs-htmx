package setup

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// Tests in this file pin the ReadModelDB dialect fail-fast (M1 of the
// 2026-09-17 gap-bundle plan): the flattened config path always builds
// SQLite-dialect read models, so a handle that does not speak SQLite must be
// rejected at New with a pointer to ServiceConfig.ReadModelDialect — not
// silently accepted and discovered via corrupted tables later.
//
// database/sql does not expose the registered driver name, so detection is a
// dialect probe ("SELECT sqlite_version()"). The fakes below simulate the two
// probe-failure classes the validator must tell apart:
//
//   - reachable, wrong dialect (queries answer, sqlite_version does not exist)
//   - unreachable (no connection at all — must PASS validation and surface
//     later as usermgmt's own infrastructure error)

// errFakeOpenNeverUsed satisfies driver.Driver.Open; sql.OpenDB bypasses it.
var errFakeOpenNeverUsed = errors.New("fake driver: Open is never used (sql.OpenDB bypasses it)")

// fakeDBConnector builds *sql.DB handles whose connections behave as
// configured. connectErr != nil simulates an unreachable database; queryErr
// != nil simulates a reachable database that rejects the probe query.
type fakeDBConnector struct {
	connectErr error
	queryErr   error
}

func (c *fakeDBConnector) Connect(context.Context) (driver.Conn, error) {
	if c.connectErr != nil {
		return nil, c.connectErr
	}

	return &fakeConn{queryErr: c.queryErr}, nil
}

func (c *fakeDBConnector) Driver() driver.Driver { return fakeDBDriver{} }

type fakeDBDriver struct{}

func (fakeDBDriver) Open(string) (driver.Conn, error) { return nil, errFakeOpenNeverUsed }

// fakeConn answers queries and pings; Prepare/Begin stay on the embedded nil
// interface (they are never reached by the probe path).
type fakeConn struct {
	driver.Conn
	queryErr error
}

func (c *fakeConn) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	return nil, c.queryErr
}

func (c *fakeConn) PingContext(context.Context) error { return nil }

func (c *fakeConn) Close() error { return nil }

// nonSQLiteDB returns a handle that behaves like a reachable non-SQLite
// engine: every query fails the way "SELECT sqlite_version()" fails on
// Postgres, while Ping succeeds.
func nonSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()

	db := sql.OpenDB(&fakeDBConnector{
		queryErr: errors.New(`ERROR: function sqlite_version() does not exist (SQLSTATE 42883)`),
	})
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// unreachableDB returns a handle whose connections can never be established.
func unreachableDB(t *testing.T) *sql.DB {
	t.Helper()

	db := sql.OpenDB(&fakeDBConnector{
		connectErr: errors.New("dial tcp 127.0.0.1:5432: connect: connection refused"),
	})
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func openSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "dialect-probe.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// --- validateReadModelDialect classification ---

func TestValidateReadModelDialect_NilDBPasses(t *testing.T) {
	t.Parallel()

	if err := (Config{}).validateReadModelDialect(); err != nil {
		t.Fatalf("nil ReadModelDB must pass: %v", err)
	}
}

func TestValidateReadModelDialect_SQLitePasses(t *testing.T) {
	t.Parallel()

	if err := (Config{ReadModelDB: openSQLiteDB(t)}).validateReadModelDialect(); err != nil {
		t.Fatalf("SQLite ReadModelDB must pass: %v", err)
	}
}

func TestValidateReadModelDialect_UnreachableDBPasses(t *testing.T) {
	t.Parallel()

	// An unreachable database cannot be disproven to be SQLite; rejecting it
	// here would misattribute a connectivity failure as a dialect rejection.
	// It must fail later, inside usermgmt, as an infrastructure error.
	if err := (Config{ReadModelDB: unreachableDB(t)}).validateReadModelDialect(); err != nil {
		t.Fatalf("unreachable ReadModelDB must pass validation: %v", err)
	}
}

func TestValidateReadModelDialect_ReachableNonSQLiteRejected(t *testing.T) {
	t.Parallel()

	err := (Config{ReadModelDB: nonSQLiteDB(t)}).validateReadModelDialect()
	if err == nil {
		t.Fatal("reachable non-SQLite ReadModelDB must be rejected")
	}

	if !strings.Contains(err.Error(), "ReadModelDialect") {
		t.Errorf("rejection must point at ServiceConfig.ReadModelDialect, got: %v", err)
	}
}

// --- New-level contract ---

func TestNew_NonSQLiteReadModelDBRejectedAtNew(t *testing.T) {
	t.Parallel()

	_, err := New(Config{Title: "Dialect Footgun", ReadModelDB: nonSQLiteDB(t)})
	if err == nil {
		t.Fatal("New must reject a flattened ReadModelDB that does not speak SQLite")
	}

	for _, want := range []string{"does not speak SQLite", "ReadModelDialect"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("rejection must mention %q, got: %v", want, err)
		}
	}
}

func TestNew_UnreachableReadModelDBFailsAsInfrastructure(t *testing.T) {
	t.Parallel()

	_, err := New(Config{Title: "Down DB", ReadModelDB: unreachableDB(t)})
	if err == nil {
		t.Fatal("New must fail: the flattened path migrates read-model tables at construction")
	}

	if strings.Contains(err.Error(), "does not speak SQLite") {
		t.Errorf("connectivity failure must not be reported as a dialect rejection: %v", err)
	}
}
