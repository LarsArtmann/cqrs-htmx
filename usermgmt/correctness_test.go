package usermgmt

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestStartPeriodicEviction_RunsAndStops verifies that the periodic eviction
// goroutine calls the evict function at each interval and that the stop
// function terminates it cleanly.
func TestStartPeriodicEviction_RunsAndStops(t *testing.T) {
	t.Parallel()

	var calls int32

	stop := startPeriodicEviction(func() int {
		atomic.AddInt32(&calls, 1)
		return 0
	}, 20*time.Millisecond)

	time.Sleep(100 * time.Millisecond)
	stop()

	time.Sleep(30 * time.Millisecond) // let in-flight calls drain
	before := atomic.LoadInt32(&calls)
	time.Sleep(80 * time.Millisecond)
	after := atomic.LoadInt32(&calls)

	if before < 2 {
		t.Fatalf("expected at least 2 eviction calls in 100ms, got %d", before)
	}

	if after > before {
		t.Fatalf("eviction continued after stop: before=%d after=%d", before, after)
	}
}

// TestStateCache_ServesUpdatedStateAfterWrite verifies that the state cache
// is correctly invalidated/updated after a command write. If the cache served
// stale state, the second ChangeEmail would fail (version mismatch) or
// produce incorrect state.
func TestStateCache_ServesUpdatedStateAfterWrite(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	defer svc.Close() //nolint:errcheck // test cleanup

	ctx := context.Background()
	resp, err := svc.Register(ctx, RegisterRequest{
		ID:    GenerateUserID(),
		Email: "initial@example.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	userID := resp.User.ID

	if err := svc.ChangeEmail(ctx, userID, "first@example.com"); err != nil {
		t.Fatalf("first ChangeEmail: %v", err)
	}

	if err := svc.ChangeEmail(ctx, userID, "second@example.com"); err != nil {
		t.Fatalf("second ChangeEmail: %v", err)
	}

	user, err := svc.GetUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}

	if user.Email != "second@example.com" {
		t.Errorf("expected email 'second@example.com', got %q — cache served stale state", user.Email)
	}
}
