package usermgmt

import (
	"testing"
	"time"
)

func TestOAuth2StateStore_SaveAndConsume(t *testing.T) {
	store := newOAuth2StateStore()
	state, err := store.Save("google", "pkce-abc", 5*time.Minute)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if state == "" {
		t.Fatal("expected non-empty state")
	}

	provider, pkce, err := store.Consume(state)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if provider != "google" || pkce != "pkce-abc" {
		t.Fatalf("wrong data: provider=%q pkce=%q", provider, pkce)
	}
}

func TestOAuth2StateStore_Consume_NotFound(t *testing.T) {
	store := newOAuth2StateStore()
	_, _, err := store.Consume("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown state")
	}
}

func TestOAuth2StateStore_Consume_Expired(t *testing.T) {
	store := newOAuth2StateStore()
	state, _ := store.Save("google", "pkce-123", 1*time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	_, _, err := store.Consume(state)
	if err == nil {
		t.Fatal("expected error for expired state")
	}
}

func TestOAuth2StateStore_Consume_OneTimeUse(t *testing.T) {
	store := newOAuth2StateStore()
	state, _ := store.Save("google", "pkce-123", 5*time.Minute)
	_, _, _ = store.Consume(state)
	// Second consume should fail
	_, _, err := store.Consume(state)
	if err == nil {
		t.Fatal("expected error for double consume")
	}
}

func TestOAuth2StateStore_EvictExpired(t *testing.T) {
	store := newOAuth2StateStore()
	store.Save("google", "a", 1*time.Millisecond) //nolint:errcheck
	store.Save("github", "b", 5*time.Minute)      //nolint:errcheck
	time.Sleep(2 * time.Millisecond)
	evicted := store.EvictExpired()
	if evicted != 1 {
		t.Fatalf("expected 1 evicted, got %d", evicted)
	}
	// Remaining state should still be valid
	_, _, err := store.Consume(consumeAllRemaining(store))
	if err != nil {
		t.Fatalf("unexpected error after eviction: %v", err)
	}
}

// consumeAllRemaining returns the first remaining state key for testing.
func consumeAllRemaining(s *oauth2StateStore) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for state := range s.states {
		return state
	}
	return ""
}
