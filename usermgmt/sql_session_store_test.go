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

	_, err := store.Create(ctx, uid, -1*time.Second)
	if err != nil {
		t.Fatalf("Create expired session: %v", err)
	}
	_, err = store.Create(ctx, uid, 1*time.Hour)
	if err != nil {
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

func TestSQLSessionStore_StartCleanupSweeper(t *testing.T) {
	store := newTestSQLiteSessionStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	uid := NewUserID("sweeper-test")
	_, err := store.Create(ctx, uid, -1*time.Second)
	if err != nil {
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
