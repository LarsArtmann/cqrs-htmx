package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

type emptyCommandJournal struct{}

func (emptyCommandJournal) ReadAll(context.Context) ([]*command.PersistedCommand, error) {
	return nil, nil
}

type emptyQueryJournal struct{}

func (emptyQueryJournal) ReadAllQueries(context.Context) ([]*query.PersistedQuery, error) {
	return nil, nil
}

// cqrs-lint:ignore-start(A023,F001) no-op store by design: dashboard empty-state snapshot panel for e2e specs
type emptySnapshotStore struct{}

func (emptySnapshotStore) Save(context.Context, snapshot.Snapshot) error { return nil }
func (emptySnapshotStore) Delete(context.Context, id.StreamRef) error    { return nil }
func (emptySnapshotStore) Load(context.Context, id.StreamRef) (*snapshot.Snapshot, error) {
	return nil, snapshot.ErrSnapshotNotFound
}

func (emptySnapshotStore) LoadAtVersion(context.Context, id.StreamRef, event.Version) (*snapshot.Snapshot, error) {
	return nil, snapshot.ErrSnapshotNotFound
}

//cqrs-lint:ignore-end

// demoProjection is a minimal projection.Projection: it processes every
// event without error so the worker stays healthy for badge assertions.
type demoProjection struct{}

func (demoProjection) Name() string                                  { return "demo-projection" }
func (demoProjection) Handle(_ context.Context, _ event.Event) error { return nil }
func (demoProjection) EventTypes() []event.Type                      { return nil }

// seedDashboard appends a few generic events across two streams, starts
// the projection host, and seeds two dead-letter entries so the browser-truth
// specs assert real tables, badges, worker states, and the DLQ count notice
// instead of empty states.
func seedDashboard(
	store *memorystorage.MemoryStore,
	host *projectionhost.Host,
	deadStore projectionhost.DeadLetterStore,
) {
	ctx := context.Background()

	var lastEvent event.Event

	for _, streamType := range []string{"User", "Tenant"} {
		aggID := id.NewStreamID()
		ref := id.StreamRef{Type: id.StreamType(streamType), ID: aggID}

		for v := uint64(1); v <= 2; v++ {
			evt, eErr := event.New(
				event.Type(streamType+".registered"),
				aggID,
				id.StreamType(streamType),
				event.Version(v),
				map[string]string{"seq": strconv.FormatUint(v, 10)},
				event.WithSchemaVersion(1),
			)
			if eErr != nil {
				log.Fatalf("event.New: %v", eErr)
			}

			if aErr := store.AppendBatch(ctx, ref, []event.Event{evt}); aErr != nil {
				log.Fatalf("AppendBatch: %v", aErr)
			}

			lastEvent = evt
		}
	}

	if host == nil {
		return
	}

	if rErr := host.Register(demoProjection{}); rErr != nil {
		log.Fatalf("host.Register: %v", rErr)
	}

	if sErr := host.Start(ctx); sErr != nil {
		log.Fatalf("host.Start: %v", sErr)
	}

	// Deterministic DLQ seed: two poison entries for the demo projection (a
	// failing handler + retry exhaustion would be timing-dependent; direct
	// Store calls make the count notice assertable: "Showing 2 items.").
	for i := 1; i <= 2; i++ {
		entry := projectionhost.DeadLetterEntry{
			ProjectionName: demoProjection{}.Name(),
			EventID:        fmt.Sprintf("e2e-dlq-%d", i),
			EventType:      lastEvent.Type().String(),
			StreamID:       lastEvent.StreamID().String(),
			Event:          lastEvent,
			Error:          "synthetic e2e poison event",
			ErrorFamily:    "corruption",
			FailedAt:       time.Now().Add(-time.Duration(i) * time.Minute),
		}

		if dErr := deadStore.Store(ctx, entry); dErr != nil {
			log.Fatalf("deadStore.Store: %v", dErr)
		}
	}
}

// emitDebugHandler backs POST /api/debug/emit: one distinctive event lands in
// the dashboard journal (fresh clients see it after reload — replay) and on
// the bus (connected dashboards append the row live). The name rides the
// payload so specs can assert the exact row.
func emitDebugHandler(store *memorystorage.MemoryStore, bus *eventtest.FakeBus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.FormValue("name")
		if name == "" {
			name = "unnamed-probe"
		}

		aggID := id.NewStreamID()
		ref := id.StreamRef{Type: e2eDebugStreamType, ID: aggID}

		evt, err := event.New(
			e2eDebugEventType,
			aggID,
			e2eDebugStreamType,
			event.Version(1),
			map[string]string{"name": name},
			event.WithSchemaVersion(1),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		ctx := r.Context()
		if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		if err := bus.Publish(ctx, evt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprintf(w, `{"id":%q,"name":%q}`, evt.ID().String(), name)
	}
}

// e2eDebugStreamType/EventType are deliberately outside the seeded User/
// Tenant vocabulary: specs must be able to assert exactly the emitted row,
// never a seed row.
const (
	e2eDebugStreamType = id.StreamType("E2e")
	e2eDebugEventType  = event.Type("E2e.debugEmitted")
)
