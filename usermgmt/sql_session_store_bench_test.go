package usermgmt

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// newBenchSQLiteSessionStore builds an isolated in-memory SQLite session store for benchmarks.
// Uses the shared in-memory cache (single connection) so the schema persists across statements.
func newBenchSQLiteSessionStore(b *testing.B) *SQLSessionStore {
	b.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		b.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := OptimizeSQLiteDB(context.Background(), db); err != nil {
		b.Fatalf("OptimizeSQLiteDB: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })

	store, err := NewSQLSessionStore(context.Background(), db, "sqlite")
	if err != nil {
		b.Fatalf("NewSQLSessionStore: %v", err)
	}
	b.Cleanup(func() { _ = store.Close() })
	return store
}

// benchCreateSession creates and stores a session for benchmark setup.
func benchCreateSession(b *testing.B, store SessionStore, ctx context.Context, uid UserID, ttl time.Duration) *Session {
	b.Helper()
	s, err := NewSession(uid, ttl)
	if err != nil {
		b.Fatalf("NewSession: %v", err)
	}
	if err := store.Create(ctx, s); err != nil {
		b.Fatalf("store.Create: %v", err)
	}
	return s
}

// BenchmarkSQLSessionStore_Create measures the throughput of inserting a new
// session row: token generation (crypto-random), INSERT, return Session struct.
func BenchmarkSQLSessionStore_Create(b *testing.B) {
	store := newBenchSQLiteSessionStore(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := range b.N {
		benchCreateSession(b, store, ctx, NewUserID(fmt.Sprintf("bench-user-%d", i)), time.Hour)
	}
}

// BenchmarkSQLSessionStore_Find measures the throughput of looking up a session
// by its token. Sessions are pre-populated to avoid measuring Create.
func BenchmarkSQLSessionStore_Find(b *testing.B) {
	store := newBenchSQLiteSessionStore(b)
	ctx := context.Background()

	const prePopulated = 100
	tokens := make([]string, prePopulated)
	for i := range prePopulated {
		s := benchCreateSession(b, store, ctx, NewUserID(fmt.Sprintf("find-user-%d", i)), time.Hour)
		tokens[i] = s.Token
	}

	b.ResetTimer()
	for i := range b.N {
		_, err := store.Find(ctx, tokens[i%prePopulated])
		if err != nil {
			b.Fatalf("Find: %v", err)
		}
	}
}

// BenchmarkSQLSessionStore_FindMiss measures the throughput of looking up a
// token that does not exist (returns ErrSessionNotFound).
func BenchmarkSQLSessionStore_FindMiss(b *testing.B) {
	store := newBenchSQLiteSessionStore(b)
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_, err := store.Find(ctx, "nonexistent-token-for-benchmark")
		if err == nil {
			b.Fatal("expected ErrSessionNotFound, got nil")
		}
	}
}

// BenchmarkSQLSessionStore_Delete measures the throughput of deleting a session
// by token. Each iteration deletes a fresh row to avoid no-op fast paths.
func BenchmarkSQLSessionStore_Delete(b *testing.B) {
	store := newBenchSQLiteSessionStore(b)
	ctx := context.Background()

	tokens := make([]string, b.N)
	for i := range b.N {
		s := benchCreateSession(b, store, ctx, NewUserID("pre-create"), time.Hour)
		tokens[i] = s.Token
	}

	b.ResetTimer()
	for i := range b.N {
		if err := store.Delete(ctx, tokens[i]); err != nil {
			b.Fatalf("Delete: %v", err)
		}
	}
}

// BenchmarkSQLSessionStore_DeleteByUserID measures the throughput of deleting
// all sessions for a user. Each iteration deletes one user's sessions.
func BenchmarkSQLSessionStore_DeleteByUserID(b *testing.B) {
	store := newBenchSQLiteSessionStore(b)
	ctx := context.Background()

	userIDs := make([]UserID, b.N)
	for i := range b.N {
		uid := NewUserID(fmt.Sprintf("del-user-%d", i))
		userIDs[i] = uid
		benchCreateSession(b, store, ctx, uid, time.Hour)
	}

	b.ResetTimer()
	for i := range b.N {
		if err := store.DeleteByUserID(ctx, userIDs[i]); err != nil {
			b.Fatalf("DeleteByUserID: %v", err)
		}
	}
}

// BenchmarkSQLSessionStore_EvictExpired measures the throughput of expired-session
// eviction. Inserts N expired rows and measures how fast EvictExpired removes them.
func BenchmarkSQLSessionStore_EvictExpired(b *testing.B) {
	ctx := context.Background()

	// Each iteration needs fresh expired rows. Pre-create b.N worth, then
	// measure eviction in batches. We measure the cost of the DELETE statement
	// (which scales with the number of expired rows).
	for range b.N {
		// Use a fresh store per iteration to keep the workload constant.
		b.StopTimer()
		store := newBenchSQLiteSessionStore(b)
		const expiredCount = 50
		for range expiredCount {
			benchCreateSession(b, store, ctx, NewUserID("expired"), -1*time.Hour)
		}
		b.StartTimer()

		n, err := store.EvictExpired(ctx)
		if err != nil {
			b.Fatalf("EvictExpired: %v", err)
		}
		if n != expiredCount {
			b.Fatalf("evicted %d, want %d", n, expiredCount)
		}
	}
}
