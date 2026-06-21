package usermgmt

import (
	"context"
	"testing"
	"time"
)

func TestInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()

	session := createTestSession(t, store, context.Background(), NewUserID("user-1"))

	found, err := store.Find(context.Background(), session.Token)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if found.UserID != NewUserID("user-1") {
		t.Errorf("expected UserID user-1, got %s", found.UserID)
	}

	_, err = store.Find(context.Background(), "nonexistent")
	assertErrorIs(t, err, ErrSessionNotFound, "ErrSessionNotFound")

	if err := store.Delete(context.Background(), session.Token); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.Find(context.Background(), session.Token)
	assertErrorIs(t, err, ErrSessionNotFound, "ErrSessionNotFound")
}

func assertSessionNotFound(t *testing.T, store *InMemorySessionStore, token, msg string) {
	t.Helper()
	_, err := store.Find(context.Background(), token)
	assertErrorIs(t, err, ErrSessionNotFound, msg)
}

func TestInMemorySessionStore_DeleteByUserID(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()
	s1 := createTestSession(t, store, ctx, NewUserID("user-1"))
	createTestSession(t, store, ctx, NewUserID("user-1"))
	createTestSession(t, store, ctx, NewUserID("user-2"))

	if err := store.DeleteByUserID(ctx, NewUserID("user-1")); err != nil {
		t.Fatalf("DeleteByUserID: %v", err)
	}

	assertSessionNotFound(t, store, s1.Token, "expected s1 deleted")
}

func TestInMemorySessionStore_EvictExpired(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	s1, err := NewSession(NewUserID("u1"), -time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, s1); err != nil {
		t.Fatal(err)
	}
	s2 := createTestSession(t, store, ctx, NewUserID("u2"))

	evicted := store.EvictExpired()
	if evicted != 1 {
		t.Errorf("expected 1 eviction, got %d", evicted)
	}
	assertSessionNotFound(t, store, s1.Token, "expected expired session to be evicted")
	if _, err := store.Find(ctx, s2.Token); err != nil {
		t.Errorf("expected valid session to remain: %v", err)
	}

	if store.EvictExpired() != 0 {
		t.Error("expected 0 evictions on clean store")
	}
}
