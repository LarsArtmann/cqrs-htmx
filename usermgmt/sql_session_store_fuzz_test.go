package usermgmt

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// FuzzSQLSessionStore_CreateFindRoundTrip fuzzes the SQLSessionStore round-trip
// with arbitrary userID strings. Verifies that whatever we Create can be found
// back by its token with identical fields (token, userID, timestamps within tolerance).
//
// Seed corpus covers:
//   - empty / short / long userIDs
//   - unicode (ààà, russian, emoji)
//   - SQL injection attempts (`); DROP TABLE--; --)
//   - whitespace-only / null bytes / quotes
func FuzzSQLSessionStore_CreateFindRoundTrip(f *testing.F) {
	seeds := []string{
		"user-1",
		"",
		strings.Repeat("a", 1024),
		"ààà-üser",
		"пользователь",
		"🎉",
		"'); DROP TABLE user_sessions;--",
		"' OR '1'='1",
		"\x00\x00\x00",
		"   ",
		"\"quoted\"",
		`"\`,
	}
	for _, s := range seeds {
		f.Add(s, int64(time.Hour))
		f.Add(s, int64(-time.Second))
		f.Add(s, int64(0))
	}

	f.Fuzz(func(t *testing.T, userIDRaw string, ttlNS int64) {
		ttl := time.Duration(ttlNS)
		if ttl < -24*time.Hour || ttl > 24*365*time.Hour {
			t.Skip("ttl out of realistic range")
		}

		store := newTestSQLiteSessionStore(t)
		ctx := context.Background()
		userID := NewUserID(userIDRaw)

		created, err := store.Create(ctx, userID, ttl)
		if err != nil {
			t.Fatalf("Create(userID=%q, ttl=%v): %v", userIDRaw, ttl, err)
		}

		found := findSessionOrFatal(t, ctx, store, created.Token)
		verifySessionRoundTrip(t, userID, created, found)

		deleteAndVerifyIdempotent(t, ctx, store, created.Token)
		verifySessionGone(t, ctx, store, created.Token)
	})
}

// findSessionOrFatal finds a session by token, failing the test on error.
func findSessionOrFatal(t *testing.T, ctx context.Context, store *SQLSessionStore, token string) *Session {
	t.Helper()
	s, err := store.Find(ctx, token)
	if err != nil {
		t.Fatalf("Find(token=%q): %v", token, err)
	}
	return s
}

// verifySessionRoundTrip asserts the found session matches the created session field-by-field.
func verifySessionRoundTrip(t *testing.T, userID UserID, created, found *Session) {
	t.Helper()
	if found.Token != created.Token {
		t.Errorf("token mismatch: created=%q found=%q", created.Token, found.Token)
	}
	if found.UserID.Get() != userID.Get() {
		t.Errorf("userID mismatch: in=%q out=%q", userID.Get(), found.UserID.Get())
	}
	if !found.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt mismatch: created=%v found=%v", created.CreatedAt, found.CreatedAt)
	}
	if !found.ExpiresAt.Equal(created.ExpiresAt) {
		t.Errorf("ExpiresAt mismatch: created=%v found=%v", created.ExpiresAt, found.ExpiresAt)
	}
}

// deleteAndVerifyIdempotent deletes the token twice — first call removes, second is a no-op.
func deleteAndVerifyIdempotent(t *testing.T, ctx context.Context, store *SQLSessionStore, token string) {
	t.Helper()
	if err := store.Delete(ctx, token); err != nil {
		t.Fatalf("Delete(token=%q) first call: %v", token, err)
	}
	if err := store.Delete(ctx, token); err != nil {
		t.Fatalf("Delete(token=%q) second call (idempotent): %v", token, err)
	}
}

// verifySessionGone asserts that Find returns ErrSessionNotFound after deletion.
func verifySessionGone(t *testing.T, ctx context.Context, store *SQLSessionStore, token string) {
	t.Helper()
	_, err := store.Find(ctx, token)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Find after Delete: expected ErrSessionNotFound, got %v", err)
	}
}

// FuzzSQLSessionStore_DeleteByUserID fuzzes DeleteByUserID with arbitrary
// userID strings. Verifies that sessions created under a userID are removed
// when DeleteByUserID is called, regardless of the userID's content.
func FuzzSQLSessionStore_DeleteByUserID(f *testing.F) {
	seeds := []string{
		"user-a",
		"",
		"unicode-Üser-1",
		"'); DROP TABLE--;",
		"\x00\x01\x02",
		"   ",
		strings.Repeat("x", 256),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, userIDRaw string) {
		store := newTestSQLiteSessionStore(t)
		ctx := context.Background()
		userID := NewUserID(userIDRaw)

		s1, err := store.Create(ctx, userID, time.Hour)
		if err != nil {
			t.Fatalf("Create s1: %v", err)
		}
		s2, err := store.Create(ctx, userID, time.Hour)
		if err != nil {
			t.Fatalf("Create s2: %v", err)
		}

		if err := store.DeleteByUserID(ctx, userID); err != nil {
			t.Fatalf("DeleteByUserID(%q): %v", userIDRaw, err)
		}

		if _, err := store.Find(ctx, s1.Token); !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("s1 still present after DeleteByUserID: err=%v", err)
		}
		if _, err := store.Find(ctx, s2.Token); !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("s2 still present after DeleteByUserID: err=%v", err)
		}

		if err := store.DeleteByUserID(ctx, userID); err != nil {
			t.Errorf("DeleteByUserID idempotent second call: %v", err)
		}
	})
}
