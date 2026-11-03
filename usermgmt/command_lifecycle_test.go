package usermgmt

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// lifecycleEventTypes filters a store's whole journal down to the
// command-lifecycle event types, in append order.
func lifecycleEventTypes(t *testing.T, store *memory.MemoryStore) []string {
	t.Helper()

	all, err := store.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	var types []string
	for _, evt := range all {
		if strings.HasPrefix(string(evt.Type()), "command.") {
			types = append(types, string(evt.Type()))
		}
	}
	return types
}

// TestCommandLifecycle_FailureRecordsReceivedFailedDeadLettered pins the
// opt-in ServiceConfig.CommandLifecycle seam: a non-rejection failure through
// the real dispatcher lands received -> failed -> dead-lettered on the
// command-lifecycle stream (adjacent outer+attempt pair per upstream docs).
func TestCommandLifecycle_FailureRecordsReceivedFailedDeadLettered(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	infrastructureFailure := errors.New("synthetic infrastructure failure")
	svc := newTestServiceWithConfig(t, ServiceConfig{
		EventStore:       store,
		CommandLifecycle: commandlifecycle.NewRecorder(store),
		CommandMiddleware: []command.Middleware{
			func(next command.Handler) command.Handler {
				return func(_ context.Context, _ command.Command) error {
					_ = next // every dispatched command fails before the handler
					return infrastructureFailure
				}
			},
		},
	})
	defer svc.Close() //nolint:errcheck // test cleanup

	userID := GenerateUserID()
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}

	dispatchErr := svc.dispatcher.Dispatch(t.Context(), NewRegisterUserCmd(aggID, "doomed@example.com", "Doomed", nil))
	if !errors.Is(dispatchErr, infrastructureFailure) {
		t.Fatalf("dispatch error = %v, want the synthetic failure", dispatchErr)
	}

	got := lifecycleEventTypes(t, store)
	want := []string{
		"command.received",
		"command.failed",
		"command.dead-lettered",
	}
	if len(got) != len(want) {
		t.Fatalf("lifecycle events = %v, want exactly %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("lifecycle events = %v, want %v (order matters)", got, want)
		}
	}

	if _, ok := svc.readModel.FindByEmail("doomed@example.com"); ok {
		t.Error("failed command reached the domain (read model mutated)")
	}
}

// TestCommandLifecycle_RejectionNeverDeadLetters pins the classification
// contract: a Rejection-family error lands received -> rejected, never
// failed/dead-lettered — rejections never retry and never DLQ.
func TestCommandLifecycle_RejectionNeverDeadLetters(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	svc := newTestServiceWithConfig(t, ServiceConfig{
		EventStore:       store,
		CommandLifecycle: commandlifecycle.NewRecorder(store),
	})
	defer svc.Close() //nolint:errcheck // test cleanup

	userID := GenerateUserID()
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}

	// Empty email: the decider itself rejects (Rejection family) — the
	// opt-in validation middleware is deliberately absent here so the
	// classification under test is the lifecycle pair's, not validation's.
	if err := svc.dispatcher.Dispatch(t.Context(), NewRegisterUserCmd(aggID, "", "X", nil)); err == nil {
		t.Fatal("empty email dispatch unexpectedly succeeded")
	}

	got := lifecycleEventTypes(t, store)
	want := []string{"command.received", "command.rejected"}
	if len(got) != len(want) {
		t.Fatalf("lifecycle events = %v, want exactly %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("lifecycle events = %v, want %v", got, want)
		}
	}
}

// TestCommandLifecycle_SuccessRecordsReceivedCompleted: the happy path lands
// received -> completed and the domain event still flows.
func TestCommandLifecycle_SuccessRecordsReceivedCompleted(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	svc := newTestServiceWithConfig(t, ServiceConfig{
		EventStore:       store,
		CommandLifecycle: commandlifecycle.NewRecorder(store),
	})
	defer svc.Close() //nolint:errcheck // test cleanup

	userID := GenerateUserID()
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}

	if err := svc.dispatcher.Dispatch(
		t.Context(),
		NewRegisterUserCmd(aggID, "lifecycle@example.com", "Lifecycle", nil),
	); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	got := lifecycleEventTypes(t, store)
	want := []string{"command.received", "command.completed"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("lifecycle events = %v, want %v", got, want)
	}

	if _, ok := svc.readModel.FindByEmail("lifecycle@example.com"); !ok {
		t.Error("domain event did not flow to the read model under lifecycle middleware")
	}
}
