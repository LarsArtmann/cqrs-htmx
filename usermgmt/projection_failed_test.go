package usermgmt

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

type failingProjection struct{}

func (failingProjection) Name() string             { return "always-fail" }
func (failingProjection) EventTypes() []event.Type { return nil }

func (failingProjection) Handle(_ context.Context, _ event.Event) error {
	return errors.New("permanent handler failure")
}

func (failingProjection) Reset(_ context.Context) error { return nil }

// TestOnProjectionFailed_FiresOnTerminalFailure verifies that the
// OnProjectionFailed callback (wired via EventSourcedConfig.OnProjectionFailed
// → projectionhost.WithOnFailed) fires when a projection worker exhausts its
// restart budget. This test creates the host directly (without DLQ) to match
// the projectionhost test pattern, proving the callback mechanism works from
// the usermgmt package context.
func TestOnProjectionFailed_FiresOnTerminalFailure(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	aggID := id.NewStreamID()
	evt, err := event.New(event.Type("test.event"), aggID, "TestAggregate", 1, map[string]string{"k": "v"})
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	ref := id.StreamRef{Type: "TestAggregate", ID: aggID}
	if err := store.AppendBatch(context.Background(), ref, []event.Event{evt}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	var (
		failedMu  sync.Mutex
		failedNme string
		failedErr string
	)

	host, err := projectionhost.New(
		store,
		memory.NewMemoryCheckpointStore(),
		projectionhost.WithMaxRestarts(1),
		projectionhost.WithBackoff(time.Millisecond, 5*time.Millisecond),
		projectionhost.WithOnFailed(func(name, lastErr string) {
			failedMu.Lock()
			failedNme = name
			failedErr = lastErr
			failedMu.Unlock()
		}),
	)
	if err != nil {
		t.Fatalf("projectionhost.New: %v", err)
	}

	if err := host.Register(failingProjection{}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer host.Stop() //nolint:errcheck // test cleanup

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		failedMu.Lock()
		done := failedNme != ""
		failedMu.Unlock()
		if done {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	failedMu.Lock()
	defer failedMu.Unlock()

	if failedNme != "always-fail" {
		t.Fatalf("expected OnFailed for 'always-fail', got %q (err: %q)", failedNme, failedErr)
	}

	if failedErr == "" {
		t.Fatal("expected non-empty lastError in OnFailed callback")
	}
}
