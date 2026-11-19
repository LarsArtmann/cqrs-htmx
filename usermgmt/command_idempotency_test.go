package usermgmt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	errorfamily "github.com/larsartmann/go-error-family"
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

// attachIdempotencyKey applies the replay key the default guard reads — the
// same metadata key the B2 decode hook populates from the HTTP header.
func attachIdempotencyKey(cmd command.Command, key string) {
	if applier, ok := cmd.(commandOptionApplier); ok {
		applier.ApplyOptions(command.WithCustomMetadata(string(IdempotencyMetadataKey), key))
	}
}

// newKeyedRegisterUser builds one registration command with a replay key
// (an empty key leaves the command key-less — the guard's pass-through form)
// and returns it with the user ID its aggregate maps to.
func newKeyedRegisterUser(t *testing.T, email, key string) (command.Command, UserID) {
	t.Helper()
	userID := GenerateUserID()
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}
	cmd := NewRegisterUserCmd(aggID, email, "", nil)
	attachIdempotencyKey(cmd, key)
	return cmd, userID
}

// TestCommandIdempotency_DefaultMountDedupesKeyedRetry proves the K2 default
// mount: two DIFFERENT registration commands (distinct aggregates, distinct
// emails) carrying the same actor-scoped Idempotency-Key dedupe — the second
// dispatch answers the re-familied replay Conflict without executing, and the
// second user is never created. No config needed: the guard mounts by
// default.
func TestCommandIdempotency_DefaultMountDedupesKeyedRetry(t *testing.T) {
	t.Parallel()

	svc := newTestServiceWithConfig(t, ServiceConfig{})
	defer svc.Close() //nolint:errcheck // test cleanup

	first, _ := newKeyedRegisterUser(t, "k2-first@example.com", "retry-42")
	if err := svc.dispatcher.Dispatch(t.Context(), first); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}

	second, secondID := newKeyedRegisterUser(t, "k2-second@example.com", "retry-42")
	err := svc.dispatcher.Dispatch(t.Context(), second)
	if err == nil {
		t.Fatal("keyed replay unexpectedly executed")
	}
	if got := errorfamily.Code(err); got != "usermgmt.command.replayed" {
		t.Errorf("replay error code = %q, want the default guard's replay Conflict", got)
	}

	// The deduped command never reached the domain: its user does not exist.
	if _, err := svc.GetUser(t.Context(), secondID); err == nil {
		t.Error("replayed command's user exists — the deduped dispatch executed")
	}
}

// TestCommandIdempotency_NoKeyPassesThroughUnDeduped proves the pass-through
// branch: the same command instance dispatched twice WITHOUT an
// Idempotency-Key is never deduped by the default mount — the second dispatch
// reaches the domain and fails with the DOMAIN's own conflict (a different
// code from the guard's replay marker), exactly as if the guard were absent.
func TestCommandIdempotency_NoKeyPassesThroughUnDeduped(t *testing.T) {
	t.Parallel()

	svc := newTestServiceWithConfig(t, ServiceConfig{})
	defer svc.Close() //nolint:errcheck // test cleanup

	cmd, _ := newKeyedRegisterUser(t, "no-key@example.com", "")
	if err := svc.dispatcher.Dispatch(t.Context(), cmd); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}

	err := svc.dispatcher.Dispatch(t.Context(), cmd)
	if err == nil {
		t.Fatal("second dispatch unexpectedly succeeded (expected the domain's own conflict)")
	}
	if errors.Is(err, idempotency.ErrDuplicate) {
		t.Error("second dispatch failed with ErrDuplicate — the guard deduped a key-less command")
	}
	if got := errorfamily.Code(err); got == "usermgmt.command.replayed" {
		t.Error("second dispatch failed with the guard's replay code — the guard engaged without a key")
	}
}

// TestCommandIdempotency_OptOutDisablesGuard proves ServiceConfig.DisableCommandIdempotency
// removes the guard: two distinct commands carrying the same key BOTH execute
// (the replay semantics return to recipe-only posture).
func TestCommandIdempotency_OptOutDisablesGuard(t *testing.T) {
	t.Parallel()

	svc := newTestServiceWithConfig(t, ServiceConfig{DisableCommandIdempotency: true})
	defer svc.Close() //nolint:errcheck // test cleanup

	first, _ := newKeyedRegisterUser(t, "optout-first@example.com", "retry-7")
	if err := svc.dispatcher.Dispatch(t.Context(), first); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}

	second, secondID := newKeyedRegisterUser(t, "optout-second@example.com", "retry-7")
	if err := svc.dispatcher.Dispatch(t.Context(), second); err != nil {
		t.Fatalf("opted-out keyed replay executed: %v", err)
	}
	if _, err := svc.GetUser(t.Context(), secondID); err != nil {
		t.Errorf("second user missing — the opted-out replay did not execute: %v", err)
	}
}

// expiringStore simulates an idempotency entry whose TTL expired between
// calls: every claim succeeds, nothing is ever "seen". Against it the guard
// must let the second same-key command EXECUTE AGAIN — the documented TTL
// honesty (expiry is not a dedup guarantee).
type expiringStore struct{}

func (expiringStore) Seen(context.Context, string) (bool, error) { return false, nil }

func (expiringStore) Record(context.Context, string, time.Duration) error { return nil }

func (expiringStore) CheckAndRecord(context.Context, string, time.Duration) error {
	return nil
}

// TestCommandIdempotency_TTLExpiryExecutesAgain proves the documented
// honesty: a key whose store entry has expired re-executes. With a store
// that forgets immediately, two same-key commands both register.
func TestCommandIdempotency_TTLExpiryExecutesAgain(t *testing.T) {
	t.Parallel()

	svc := newTestServiceWithConfig(t, ServiceConfig{IdempotencyStore: expiringStore{}})
	defer svc.Close() //nolint:errcheck // test cleanup

	first, _ := newKeyedRegisterUser(t, "ttl-first@example.com", "expired-key")
	if err := svc.dispatcher.Dispatch(t.Context(), first); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}

	second, secondID := newKeyedRegisterUser(t, "ttl-second@example.com", "expired-key")
	if err := svc.dispatcher.Dispatch(t.Context(), second); err != nil {
		t.Fatalf("post-expiry dispatch should execute again: %v", err)
	}
	if _, err := svc.GetUser(t.Context(), secondID); err != nil {
		t.Errorf("post-expiry user missing — the second command did not execute: %v", err)
	}
}
