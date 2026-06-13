package usermgmt

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStore_Save_EmailTakenByOtherUser(t *testing.T) {
	store := NewInMemoryUserStore()
	u1 := NewUser(NewUserID("u1"), "a@b.com", "One")
	_ = store.Create(context.Background(), u1)

	u2 := NewUser(NewUserID("u2"), "c@d.com", "Two")
	_ = store.Create(context.Background(), u2)

	conflicting := NewUser(NewUserID("u2"), "a@b.com", "Two")
	if err := store.Save(context.Background(), conflicting); !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestEvictExpired_NoExpired(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()
	_, _ = store.Create(ctx, NewUserID("u1"), time.Hour)

	evicted := store.EvictExpired()
	if evicted != 0 {
		t.Errorf("expected 0 evictions, got %d", evicted)
	}
}

func TestEvictExpired_WithExpired(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()
	s, _ := store.Create(ctx, NewUserID("u1"), time.Millisecond)

	s.ExpiresAt = time.Now().Add(-time.Hour)
	store.mu.Lock()
	store.sessions[s.Token] = s
	store.mu.Unlock()

	time.Sleep(2 * time.Millisecond)
	evicted := store.EvictExpired()
	if evicted != 1 {
		t.Errorf("expected 1 eviction, got %d", evicted)
	}
}

func TestEvictStale_NoStale(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 5, Duration: time.Hour})
	l.RecordFailure("active@test.com")

	evicted := l.EvictStale()
	if evicted != 0 {
		t.Errorf("expected 0 evictions, got %d", evicted)
	}
}

func TestEvictStale_WithStale(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 1, Duration: 50 * time.Millisecond})
	l.RecordFailure("stale@test.com")

	time.Sleep(100 * time.Millisecond)
	evicted := l.EvictStale()
	if evicted != 1 {
		t.Errorf("expected 1 eviction, got %d", evicted)
	}
}
