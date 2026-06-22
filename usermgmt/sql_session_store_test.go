package usermgmt

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newTestSQLiteSessionStore(t *testing.T) *SQLSessionStore {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1) // SQLite :memory: is per-connection — single conn for shared state
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewSQLSessionStore(context.Background(), db, "sqlite")
	if err != nil {
		t.Fatalf("NewSQLSessionStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestSQLSessionStore_Contract(t *testing.T) {
	runSessionStoreContract(t, func(t *testing.T) SessionStore {
		t.Helper()
		return newTestSQLiteSessionStore(t)
	})
}

func TestInMemorySessionStore_Contract(t *testing.T) {
	runSessionStoreContract(t, func(_ *testing.T) SessionStore {
		return NewInMemorySessionStore()
	})
}

func TestSQLSessionStore_EvictExpired(t *testing.T) {
	store := newTestSQLiteSessionStore(t)
	ctx := context.Background()
	uid := NewUserID("expire-test")

	s1, err := NewSession(uid, -1*time.Second)
	if err != nil {
		t.Fatalf("NewSession expired: %v", err)
	}
	if err := store.Create(ctx, s1); err != nil {
		t.Fatalf("Create expired session: %v", err)
	}
	s2, err := NewSession(uid, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewSession live: %v", err)
	}
	if err := store.Create(ctx, s2); err != nil {
		t.Fatalf("Create live session: %v", err)
	}

	n, err := store.EvictExpired(ctx)
	if err != nil {
		t.Fatalf("EvictExpired: %v", err)
	}
	if n != 1 {
		t.Fatalf("EvictExpired removed %d, want 1", n)
	}
}

func TestSQLSessionStore_UnsupportedDialect(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, err = NewSQLSessionStore(context.Background(), db, "oracle")
	if err == nil {
		t.Fatal("expected error for unsupported dialect")
	}
}

func TestSQLSessionStore_MigratesLegacySchema(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1) // SQLite :memory: is per-connection — single conn for shared state
	t.Cleanup(func() { _ = db.Close() })

	// Legacy schema predating the origin_type/origin_data columns.
	const legacyDDL = `
	CREATE TABLE user_sessions (
		token      TEXT PRIMARY KEY,
		user_id    TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		expires_at DATETIME NOT NULL
	);`
	if _, err := db.Exec(legacyDDL); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}

	// Legacy row written by old code (no origin columns).
	now := time.Now().UTC()
	if _, err := db.Exec(
		`INSERT INTO user_sessions (token, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		"legacy-token", "legacy-user", now, now.Add(time.Hour),
	); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	// Constructing the store must add the origin columns to the legacy table.
	store, err := NewSQLSessionStore(context.Background(), db, "sqlite")
	if err != nil {
		t.Fatalf("NewSQLSessionStore over legacy schema: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// Legacy row reads back as DirectLogin (NULL origin -> default).
	found, err := store.Find(context.Background(), "legacy-token")
	if err != nil {
		t.Fatalf("Find legacy row: %v", err)
	}
	if _, ok := found.Origin.(DirectLogin); !ok {
		t.Fatalf("legacy row origin = %T, want DirectLogin", found.Origin)
	}
	if found.UserID.Get().String() == "" {
		t.Fatal("legacy row UserID empty")
	}

	// A post-migration impersonation session must round-trip its origin.
	target := ActorIDFromUser(NewUserID("mig-target"))
	admin := ActorIDFromUser(NewUserID("mig-admin"))
	imp, err := NewImpersonationSession(target, admin, "migration check", time.Hour)
	if err != nil {
		t.Fatalf("NewImpersonationSession: %v", err)
	}
	if err := store.Create(context.Background(), imp); err != nil {
		t.Fatalf("Create post-migration: %v", err)
	}
	foundImp, err := store.Find(context.Background(), imp.Token)
	if err != nil {
		t.Fatalf("Find post-migration: %v", err)
	}
	if _, ok := foundImp.Origin.(Impersonation); !ok {
		t.Fatalf("post-migration origin = %T, want Impersonation", foundImp.Origin)
	}
}

func TestSQLSessionStore_StartCleanupSweeper(t *testing.T) {
	store := newTestSQLiteSessionStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	uid := NewUserID("sweeper-test")
	expired, err := NewSession(uid, -1*time.Second)
	if err != nil {
		t.Fatalf("NewSession expired: %v", err)
	}
	if err := store.Create(ctx, expired); err != nil {
		t.Fatalf("Create expired session: %v", err)
	}

	stop := store.StartCleanupSweeper(ctx, 10*time.Millisecond)
	defer stop()

	waitCtx, waitCancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer waitCancel()
	<-waitCtx.Done()

	n, err := store.EvictExpired(ctx)
	if err != nil {
		t.Fatalf("EvictExpired after sweeper: %v", err)
	}
	if n != 0 {
		t.Fatalf("sweeper should have evicted expired sessions, found %d remaining", n)
	}
}
