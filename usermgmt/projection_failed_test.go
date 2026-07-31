package usermgmt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

type failingProjection struct{}

func (failingProjection) Name() string { return "always-fail" }

func (failingProjection) Handle(_ context.Context, _ event.Event) error {
	return errors.New("intentional projection failure")
}

func (failingProjection) EventTypes() []event.Type { return nil }

func (failingProjection) Reset(_ context.Context) error { return nil }

func TestOnProjectionFailed_FiresOnTerminalFailure(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	aggID := id.NewStreamID()
	evt, err := event.New(event.Type("test.event"), aggID, "TestAggregate", 1, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	ref := id.StreamRef{Type: "TestAggregate", ID: aggID}
	if err := store.AppendBatch(context.Background(), ref, []event.Event{evt}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	failed := make(chan struct {
		name string
		err  string
	}, 1)

	onFailed := func(projectionName, lastError string) {
		failed <- struct {
			name string
			err  string
		}{projectionName, lastError}
	}

	host, err := startProjectionHost(
		context.Background(),
		store,
		watermill.NewEventBus(),
		memory.NewMemoryCheckpointStore(),
		[]projection.Projection{failingProjection{}},
		projectionhost.WithOnFailed(onFailed),
	)
	if err != nil {
		t.Fatalf("startProjectionHost: %v", err)
	}

	defer host.Stop() //nolint:errcheck // test cleanup

	select {
	case f := <-failed:
		if f.name != "always-fail" {
			t.Errorf("expected projection name 'always-fail', got %q", f.name)
		}

		if f.err == "" {
			t.Error("expected non-empty lastError in OnFailed callback")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("OnProjectionFailed callback did not fire within 15s")
	}
}
