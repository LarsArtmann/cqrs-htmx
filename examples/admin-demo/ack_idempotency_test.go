package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// TestAckMiddleware_IdempotencyKeyHeaderDedupsBotReplays pins the bot-class
// replay contract: a client without JS sends Idempotency-Key, the middleware
// dedups on it exactly like the browser's X-Command-Id path — first request
// passes, the verbatim replay 409s, a DIFFERENT key is a distinct command.
func TestAckMiddleware_IdempotencyKeyHeaderDedupsBotReplays(t *testing.T) {
	t.Parallel()

	//nolint:staticcheck // SA1019: MemoryStore is deprecated for production use; tests are its sanctioned scope.
	store := idempotency.NewMemoryStore(10 * time.Minute)
	mw := ackMiddleware(nil, store) // broadcaster nil: this test never reaches the ACK hook

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := mw(next)

	post := func(key string) int {
		req := httptest.NewRequest(http.MethodPost, "/admin/users", nil)
		req.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		return rec.Code
	}

	if got := post("bot-key-1"); got != http.StatusOK {
		t.Fatalf("first POST with Idempotency-Key: got %d, want 200", got)
	}
	if got := post("bot-key-1"); got != http.StatusConflict {
		t.Fatalf("replayed POST with same key: got %d, want 409", got)
	}
	if got := post("bot-key-2"); got != http.StatusOK {
		t.Fatalf("POST with a different key: got %d, want 200 (distinct command)", got)
	}
}

// TestAckMiddleware_NoHeaderPassesThrough pins the pass-through: a mutation
// with neither X-Command-Id nor Idempotency-Key is never deduped (the
// middleware is opt-in per request).
func TestAckMiddleware_NoHeaderPassesThrough(t *testing.T) {
	t.Parallel()

	//nolint:staticcheck // SA1019: MemoryStore is deprecated for production use; tests are its sanctioned scope.
	store := idempotency.NewMemoryStore(10 * time.Minute)
	mw := ackMiddleware(nil, store)

	calls := 0
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))

	for i := range 3 {
		req := httptest.NewRequest(http.MethodPost, "/admin/users", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST %d without headers: got %d, want 200", i+1, rec.Code)
		}
	}
	if calls != 3 {
		t.Fatalf("handler invoked %d times, want 3 (no dedup without a client key)", calls)
	}
}
