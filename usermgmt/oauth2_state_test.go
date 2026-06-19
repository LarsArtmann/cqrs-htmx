package usermgmt

import (
	"sync"
	"sync/atomic"
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

// TestOAuth2StateStore_Consume_ConcurrentOneTimeUse verifies that when multiple
// goroutines race to Consume the same state token, exactly one succeeds and all
// others get ErrOAuthInvalidState. This is the CSRF replay-prevention guarantee
// under concurrent load — a real attacker could fire multiple callback requests
// simultaneously with a stolen state token.
func TestOAuth2StateStore_Consume_ConcurrentOneTimeUse(t *testing.T) {
	store := newOAuth2StateStore()
	state, err := store.Save("google", "pkce-race", 5*time.Minute)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	const numGoroutines = 50
	var successCount atomic.Int64
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			_, _, consumeErr := store.Consume(state)
			if consumeErr == nil {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := successCount.Load(); got != 1 {
		t.Errorf("expected exactly 1 successful Consume under concurrent access, got %d", got)
	}
}
