package usermgmt

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	"github.com/larsartmann/go-idempotency"
)

// TestCommandMiddleware_IdempotencyShortCircuitsDuplicateCommandID proves the
// CommandMiddleware seam composes with upstream middleware.CommandIdempotency:
// dispatching the SAME command instance twice (same minted command ID — the
// offline-sync replay / client-retry shape) executes the handler exactly once;
// the replay short-circuits with idempotency.ErrDuplicate BEFORE the domain
// handler runs (not as the domain's own duplicate-email Conflict).
func TestCommandMiddleware_IdempotencyShortCircuitsDuplicateCommandID(t *testing.T) {
	t.Parallel()

	//nolint:staticcheck // SA1019: MemoryStore is deprecated for production use; tests are its sanctioned scope.
	store := idempotency.NewMemoryStore(time.Minute)

	svc := newTestServiceWithConfig(t, ServiceConfig{
		CommandMiddleware: []command.Middleware{
			middleware.CommandIdempotency(store, time.Minute, nil),
		},
	})
	defer svc.Close() //nolint:errcheck // test cleanup

	userID := GenerateUserID()
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}
	cmd := NewRegisterUserCmd(aggID, "idem@example.com", "", nil)

	if err := svc.dispatcher.Dispatch(t.Context(), cmd); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}

	err = svc.dispatcher.Dispatch(t.Context(), cmd)
	if err == nil {
		t.Fatal("replayed dispatch unexpectedly succeeded")
	}
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Errorf("replay error = %v, want idempotency.ErrDuplicate (short-circuit before the handler)", err)
	}

	// The registration happened exactly once: the user is queryable and the
	// duplicate never reached the domain (it would have surfaced as the
	// domain's own email/userID conflict, a different sentinel).
	if _, err := svc.GetUser(t.Context(), userID); err != nil {
		t.Errorf("GetUser after dedup: %v", err)
	}
}
