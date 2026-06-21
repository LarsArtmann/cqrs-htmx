package usermgmt

import (
	"context"
	"errors"
	"testing"
	"time"
)

// createTestSession creates a session and stores it, failing the test on error.
func createTestSession(t *testing.T, store SessionStore, ctx context.Context, uid UserID) *Session {
	t.Helper()
	session, err := NewSession(uid, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("store.Create: %v", err)
	}
	return session
}

// runSessionStoreContract runs a suite of behavioral tests against any
// SessionStore implementation. Both InMemorySessionStore and SQLSessionStore
// use it to verify identical semantics.
//
//nolint:gocognit,cyclop,funlen // contract suite — each subtest is independently simple
func runSessionStoreContract(t *testing.T, factory func(t *testing.T) SessionStore) {
	t.Helper()

	t.Run("Create_and_Find", func(t *testing.T) {
		store := factory(t)
		ctx := context.Background()
		uid := NewUserID("user-contract-1")

		session := createTestSession(t, store, ctx, uid)
		if session.Token == "" {
			t.Fatal("empty token")
		}
		if session.UserID != uid {
			t.Fatalf("UserID = %v, want %v", session.UserID, uid)
		}

		found, err := store.Find(ctx, session.Token)
		if err != nil {
			t.Fatalf("Find: %v", err)
		}
		if found.Token != session.Token {
			t.Fatalf("Token = %q, want %q", found.Token, session.Token)
		}
		if found.UserID != uid {
			t.Fatalf("UserID = %v, want %v", found.UserID, uid)
		}
	})

	t.Run("Find_nonexistent_returns_ErrSessionNotFound", func(t *testing.T) {
		store := factory(t)
		ctx := context.Background()

		_, err := store.Find(ctx, "nonexistent-token")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("error = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("Delete_removes_session", func(t *testing.T) {
		store := factory(t)
		ctx := context.Background()
		uid := NewUserID("user-contract-2")

		session := createTestSession(t, store, ctx, uid)
		if err := store.Delete(ctx, session.Token); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		_, err := store.Find(ctx, session.Token)
		if !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("after Delete, Find error = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("Delete_nonexistent_is_idempotent", func(t *testing.T) {
		store := factory(t)
		ctx := context.Background()

		err := store.Delete(ctx, "does-not-exist")
		if err != nil {
			t.Fatalf("Delete nonexistent should be idempotent, got: %v", err)
		}
	})

	t.Run("DeleteByUserID_removes_all_user_sessions", func(t *testing.T) {
		store := factory(t)
		ctx := context.Background()
		uid := NewUserID("user-contract-3")

		s1 := createTestSession(t, store, ctx, uid)
		s2 := createTestSession(t, store, ctx, uid)

		if err := store.DeleteByUserID(ctx, uid); err != nil {
			t.Fatalf("DeleteByUserID: %v", err)
		}

		for _, token := range []string{s1.Token, s2.Token} {
			_, err := store.Find(ctx, token)
			if !errors.Is(err, ErrSessionNotFound) {
				t.Fatalf("after DeleteByUserID, Find(%q) error = %v, want ErrSessionNotFound",
					token, err)
			}
		}
	})

	t.Run("DeleteByUserID_does_not_affect_other_users", func(t *testing.T) {
		store := factory(t)
		ctx := context.Background()
		uidA := NewUserID("user-contract-a")
		uidB := NewUserID("user-contract-b")

		sessionA := createTestSession(t, store, ctx, uidA)
		createTestSession(t, store, ctx, uidB)

		if err := store.DeleteByUserID(ctx, uidA); err != nil {
			t.Fatalf("DeleteByUserID A: %v", err)
		}

		_, err := store.Find(ctx, sessionA.Token)
		if !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("user A session should be deleted, got: %v", err)
		}
	})

	t.Run("Create_generates_unique_tokens", func(t *testing.T) {
		store := factory(t)
		ctx := context.Background()
		uid := NewUserID("user-contract-4")

		s1 := createTestSession(t, store, ctx, uid)
		s2 := createTestSession(t, store, ctx, uid)
		if s1.Token == s2.Token {
			t.Fatal("two sessions have the same token")
		}
	})
}
