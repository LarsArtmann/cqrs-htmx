//go:build integration

// MySQL integration tests using testcontainers.
//
// These tests spin up a real MySQL container and exercise the event store,
// session store, and read model paths against a live database. They are
// gated behind the "integration" build tag so they do NOT run in the normal
// test suite or CI matrix.
//
// The test-only deps (testcontainers, go-sql-driver/mysql) are NOT in go.mod
// because go mod tidy strips deps behind custom build tags. Add them manually
// before running:
//
//	cd usermgmt
//	GOEXPERIMENT=jsonv2 go get \
//	  github.com/testcontainers/testcontainers-go/modules/mysql \
//	  github.com/go-sql-driver/mysql
//	GOEXPERIMENT=jsonv2 go test -tags integration -run MySQL -v -timeout 300s
//	GOEXPERIMENT=jsonv2 go mod tidy   # clean up afterward
//
// Requires Docker.

package usermgmt

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	_ "github.com/go-sql-driver/mysql" // register MySQL driver for database/sql
)

// newMySQLDB starts a MySQL 8.4 container and returns a *sql.DB connected
// to it. The container is terminated and the DB is closed on test cleanup.
func newMySQLDB(t *testing.T) *sql.DB {
	t.Helper()

	ctx := context.Background()
	container, err := tcmysql.Run(ctx, "mysql:8.4",
		tcmysql.WithDatabase("testdb"),
		tcmysql.WithUsername("test"),
		tcmysql.WithPassword("test"),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	connStr, err := container.ConnectionString(ctx, "parseTime=true", "multiStatements=true")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Verify connectivity.
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping mysql: %v", err)
	}

	return db
}

// TestMySQLIntegration_EventStoreRoundTrip verifies the full event store
// lifecycle against a real MySQL instance: DDL migration, append, load,
// and load-from-version.
func TestMySQLIntegration_EventStoreRoundTrip(t *testing.T) {
	db := newMySQLDB(t)
	ctx := context.Background()

	store, err := NewSQLEventStore(ctx, db, dialectMySQL)
	if err != nil {
		t.Fatalf("NewSQLEventStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	streamID := id.NewStreamID()
	ref := id.StreamRef{ID: streamID, Type: aggregateTypeUser}

	// Save a UserRegistered event.
	evt, err := event.New(
		eventUserRegistered, streamID, aggregateTypeUser, 1,
		UserRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         "integration@test.com",
			Roles:         []Role{RoleUser},
		},
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load it back.
	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
	if loaded[0].Type() != eventUserRegistered {
		t.Errorf("event type = %q, want %q", loaded[0].Type(), eventUserRegistered)
	}
	if loaded[0].StreamID() != streamID {
		t.Errorf("stream ID mismatch")
	}
	if loaded[0].Version() != 1 {
		t.Errorf("version = %d, want 1", loaded[0].Version())
	}

	// Verify LoadFromVersion returns only events after the given version.
	loadedFromV1, err := store.LoadFromVersion(ctx, ref, 1)
	if err != nil {
		t.Fatalf("LoadFromVersion(1): %v", err)
	}
	if len(loadedFromV1) != 0 {
		t.Errorf("LoadFromVersion(1) should return 0 events, got %d", len(loadedFromV1))
	}
}

// TestMySQLIntegration_OptimisticConcurrency verifies that saving with a
// stale expected version fails with a conflict error.
func TestMySQLIntegration_OptimisticConcurrency(t *testing.T) {
	db := newMySQLDB(t)
	ctx := context.Background()

	store, err := NewSQLEventStore(ctx, db, dialectMySQL)
	if err != nil {
		t.Fatalf("NewSQLEventStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	streamID := id.NewStreamID()
	ref := id.StreamRef{ID: streamID, Type: aggregateTypeUser}

	evt, err := event.New(
		eventUserRegistered, streamID, aggregateTypeUser, 1,
		UserRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         "concurrency@test.com",
			Roles:         []Role{RoleUser},
		},
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	// First save with expected version 0 — should succeed.
	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	// Second save with the SAME expected version 0 — should fail (conflict).
	evt2, err := event.New(
		eventEmailChanged, streamID, aggregateTypeUser, 2,
		EmailChangedPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         "changed@test.com",
		},
	)
	if err != nil {
		t.Fatalf("create event2: %v", err)
	}

	err = store.Save(ctx, ref, []event.Event{evt2}, 0)
	if err == nil {
		t.Fatal("expected conflict error on stale version, got nil")
	}
}

// TestMySQLIntegration_SessionStore verifies the session store migration
// and CRUD operations against real MySQL.
func TestMySQLIntegration_SessionStore(t *testing.T) {
	db := newMySQLDB(t)
	ctx := context.Background()

	store, err := NewSQLSessionStore(ctx, db, dialectMySQL)
	if err != nil {
		t.Fatalf("NewSQLSessionStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	uid := GenerateUserID()
	sess, err := NewSession(uid, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	// Create.
	if err := store.Create(ctx, sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Find.
	loaded, err := store.Find(ctx, sess.Token)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if loaded.UserID != sess.UserID {
		t.Errorf("UserID = %s, want %s", loaded.UserID, sess.UserID)
	}

	// Delete.
	if err := store.Delete(ctx, sess.Token); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify deleted.
	_, err = store.Find(ctx, sess.Token)
	if err == nil {
		t.Fatal("expected error loading deleted session, got nil")
	}
}

// TestMySQLIntegration_MultipleStreams verifies that events from different
// streams are isolated correctly.
func TestMySQLIntegration_MultipleStreams(t *testing.T) {
	db := newMySQLDB(t)
	ctx := context.Background()

	store, err := NewSQLEventStore(ctx, db, dialectMySQL)
	if err != nil {
		t.Fatalf("NewSQLEventStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	id1 := id.NewStreamID()
	id2 := id.NewStreamID()
	ref1 := id.StreamRef{ID: id1, Type: aggregateTypeUser}
	ref2 := id.StreamRef{ID: id2, Type: aggregateTypeUser}

	evt1, _ := event.New(
		eventUserRegistered, id1, aggregateTypeUser, 1,
		UserRegisteredPayload{SchemaVersion: currentSchemaVersion, Email: "a@test.com", Roles: []Role{RoleUser}},
	)
	evt2, _ := event.New(
		eventUserRegistered, id2, aggregateTypeUser, 1,
		UserRegisteredPayload{SchemaVersion: currentSchemaVersion, Email: "b@test.com", Roles: []Role{RoleUser}},
	)

	if err := store.Save(ctx, ref1, []event.Event{evt1}, 0); err != nil {
		t.Fatalf("Save ref1: %v", err)
	}
	if err := store.Save(ctx, ref2, []event.Event{evt2}, 0); err != nil {
		t.Fatalf("Save ref2: %v", err)
	}

	// Each stream should have exactly 1 event.
	l1, err := store.Load(ctx, ref1)
	if err != nil {
		t.Fatalf("Load ref1: %v", err)
	}
	if len(l1) != 1 {
		t.Errorf("ref1: expected 1 event, got %d", len(l1))
	}

	l2, err := store.Load(ctx, ref2)
	if err != nil {
		t.Fatalf("Load ref2: %v", err)
	}
	if len(l2) != 1 {
		t.Errorf("ref2: expected 1 event, got %d", len(l2))
	}
}
