package setup_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestShellBundle_DefaultBusRecoversSubscriberPanic pins the default bus's
// safe-by-default posture: a panicking subscriber must not kill the process,
// and the bus must keep delivering afterwards. (B024 remediation, 2026-10-07 —
// consumers bringing their own EventBus keep full ownership and are untouched.)
func TestShellBundle_DefaultBusRecoversSubscriberPanic(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(newShellConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	bus := bundle.Stores.EventBus
	if bus == nil {
		t.Fatal("shell bundle must expose its default event bus")
	}

	aggID := id.NewStreamID()
	poison := newBusEvent(t, aggID, 1)
	follower := newBusEvent(t, aggID, 2)
	poisonID := poison.ID().String()

	var deliveries atomic.Int64
	afterPanic := make(chan event.Event, 1)
	if err := bus.SubscribeAll(func(ctx context.Context, evt event.Event) error {
		deliveries.Add(1)
		if evt.ID().String() == poisonID {
			panic("subscriber defect: poisoned delivery")
		}

		select {
		case afterPanic <- evt:
		default:
		}

		return nil
	}); err != nil {
		t.Fatalf("SubscribeAll: %v", err)
	}

	if err := bus.Publish(context.Background(), poison); err != nil {
		t.Logf("publish of the poisoned event returned via the recovery path: %v", err)
	}

	select {
	case got := <-afterPanic:
		t.Fatalf("poisoned event escaped recovery and delivered: %s", got.ID())
	default:
	}

	if err := bus.Publish(context.Background(), follower); err != nil {
		t.Fatalf("publish after a recovered subscriber panic: %v", err)
	}

	select {
	case got := <-afterPanic:
		if got.ID().String() != follower.ID().String() {
			t.Errorf("delivered event = %s, want the post-recovery event %s",
				got.ID(), follower.ID())
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("bus stopped delivering after a recovered panic (deliveries seen: %d)",
			deliveries.Load())
	}
}

func newBusEvent(t *testing.T, aggID id.StreamID, version int) event.Event {
	t.Helper()

	evt, err := event.New("user.created", aggID, "User", event.Version(version), struct{}{})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	return evt
}
