package usermgmt

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// BenchmarkProjectionReplay measures the time to drain a journal of N events
// through projectionhost into all usermgmt read models. This is the projection
// startup cost on restart — the path that checkpoint-based replay optimizes.
//
// Sizes: 100 (small), 1K (medium), 10K (large) events.
// Each iteration creates a fresh memory store, appends N UserRegistered events,
// then calls StartProjections (which drains synchronously before returning).
func BenchmarkProjectionReplay(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("events=%d", size), func(b *testing.B) {
			events := makeRegistrationEvents(size)

			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				b.StopTimer()

				store := memory.NewMemoryStore()
				ctx := context.Background()
				for _, evt := range events {
					ref := id.AggregateRef{ID: evt.AggregateID(), Type: "User"}
					if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
						b.Fatalf("AppendBatch: %v", err)
					}
				}

				bus := watermill.NewEventBus()

				readModel := NewUserReadModel()
				authz, err := NewAuthz()
				if err != nil {
					b.Fatalf("NewAuthz: %v", err)
				}
				casbinProj, err := NewCasbinProjection(authz)
				if err != nil {
					b.Fatalf("NewCasbinProjection: %v", err)
				}

				b.StartTimer()

				host, err := StartProjections(
					store, bus, nil,
					readModel, nil, nil, nil, casbinProj, nil,
				)
				if err != nil {
					b.Fatalf("StartProjections: %v", err)
				}

				b.StopTimer()
				_ = host.Stop()
				closeBus(bus)
			}
		})
	}
}

// makeRegistrationEvents creates n synthetic UserRegistered events with unique
// aggregate IDs. Uses event.NewEvent to build realistic events that the
// UserReadModel projection can actually handle.
func makeRegistrationEvents(n int) []event.Event {
	events := make([]event.Event, n)
	for i := range n {
		evt, _ := event.NewEvent(
			"UserRegistered",
			id.NewStreamID(),
			"User",
			1,
			[]byte(fmt.Sprintf(`{"email":"user%d@example.com"}`, i)),
		)
		events[i] = evt
	}
	return events
}
