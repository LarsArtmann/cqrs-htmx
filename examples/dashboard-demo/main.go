// Command dashboard-demo is a runnable showcase of the CQRS/ES dashboard.
//
// It boots in-memory stores, seeds demo events (users, orders), commands,
// queries, and a snapshot, then mounts the dashboard at /dashboard.
// Open http://localhost:8098/dashboard/ to explore.
//
// This is a demo only — it uses in-memory storage and no authentication.
// Real applications back the dashboard with persistent stores and wrap it
// with auth middleware.
package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func main() {
	store := memorystorage.NewMemoryStore()
	cmdStore := memorystorage.NewMemoryCommandStore()
	queryStore := memorystorage.NewMemoryQueryStore()
	snapStore := memorystorage.NewMemorySnapshotStore()
	bus := eventtest.NewFakeBus()

	seedDemoData(store, cmdStore, queryStore, snapStore)

	reader := listing.NewInMemoryStreamReader(store)

	dash, err := dashboardui.New(dashboardui.Config{
		Title:          "CQRS Demo Dashboard",
		EventSource:    store,
		Journal:        store,
		StreamReader:   reader,
		CommandJournal: cmdStore,
		QueryJournal:   queryStore,
		SnapshotStore:  snapStore,
		EventBus:       bus,
		ReadOnly:       false,
		PageSize:       25,
	})
	if err != nil {
		log.Fatalf("dashboard: %v", err)
	}

	// Start a goroutine that publishes live events every 5 seconds so the
	// SSE feed in the dashboard shows real-time updates.
	go startLiveEvents(store, bus)

	mux := http.NewServeMux()
	dash.Mount(mux, "/dashboard/")

	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `<html><body style="font-family:sans-serif;padding:40px">
			<h1>CQRS Dashboard Demo</h1>
			<p><a href="/dashboard/">Open Dashboard</a></p>
			</body></html>`)
	})

	addr := ":8098"

	log.Printf("CQRS Dashboard Demo starting on http://localhost%s", addr)
	log.Printf("Dashboard at http://localhost%s/dashboard/", addr)
	log.Println("Press Ctrl+C to stop")

	server := &http.Server{
		Addr:              addr,
		Handler:           cqrshtmx.Chain(dash.Middleware())(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func seedDemoData(
	store *memorystorage.MemoryStore,
	cmdStore *memorystorage.MemoryCommandStore,
	queryStore *memorystorage.MemoryQueryStore,
	snapStore *memorystorage.MemorySnapshotStore,
) {
	ctx := context.Background()

	// --- Users ---
	for i, name := range []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Henry"} {
		aggID := id.NewStreamID()
		ref := id.NewStreamRef("User", aggID)

		payload, _ := json.Marshal(map[string]any{
			"name":  name,
			"email": fmt.Sprintf("%s@example.com", name),
		})

		//cqrs-lint:ignore(E006) demo data: no projection in this dashboard demo
		created, _ := event.New("user.created", aggID, "User", event.Version(1), jsontext.Value(payload)) //cqrs-lint:ignore(E004) demo data
		_ = store.Save(ctx, ref, []event.Event{created}, event.Version(0))

		//cqrs-lint:ignore(E006) demo data: no projection in this dashboard demo
		renamed, _ := event.New("user.renamed", aggID, "User", event.Version(2), map[string]any{"name": name + " Jr."}) //cqrs-lint:ignore(E004) demo data
		_ = store.Save(ctx, ref, []event.Event{renamed}, event.Version(1))

		// Record a command for this user
		cmdPayload, _ := json.Marshal(map[string]any{"userId": aggID.String(), "name": name})
		cmd, _ := command.NewPersistedCommand("create.user", ref, cmdPayload)
		_ = cmdStore.Save(ctx, ref, cmd)

		// Save a snapshot for the first user
		if i == 0 {
			snap := snapshot.Snapshot{
				StreamID:   aggID,
				StreamType: "User",
				Version:    event.Version(2),
				State:      []byte(`{"name":"Alice Jr.","email":"Alice@example.com"}`),
				CreatedAt:  time.Now(),
			}
			_ = snapStore.Save(ctx, snap)
		}
	}

	// --- Orders ---
	for i := 1; i <= 6; i++ {
		aggID := id.NewStreamID()
		ref := id.NewStreamRef("Order", aggID)

		//cqrs-lint:ignore(E006) demo data: no projection in this dashboard demo
		placed, _ := event.New("order.placed", aggID, "Order", event.Version(1), map[string]any{ //cqrs-lint:ignore(E004) demo data
			"customerId": fmt.Sprintf("cust-%d", i),
			"total":      float64(i * 2999),
			"items":      i,
		})
		_ = store.Save(ctx, ref, []event.Event{placed}, event.Version(0))

		//cqrs-lint:ignore(E006) demo data: no projection in this dashboard demo
		shipped, _ := event.New("order.shipped", aggID, "Order", event.Version(2), map[string]any{ //cqrs-lint:ignore(E004) demo data
			"trackingNumber": fmt.Sprintf("TRK%d", i*1000+i),
		})
		_ = store.Save(ctx, ref, []event.Event{shipped}, event.Version(1))
	}

	// --- Queries ---
	for _, qt := range []string{"get.user", "list.orders", "get.order", "search.users"} {
		payload, _ := json.Marshal(map[string]any{"query": qt})
		q, _ := query.NewPersistedQuery(query.Type(qt), payload)
		_ = queryStore.SaveQuery(ctx, q)
	}

	log.Println("Demo data seeded: 8 users, 6 orders, commands, queries, 1 snapshot")
}

// startLiveEvents periodically publishes new events to the event bus and store
// so the dashboard SSE feed and event browser show real-time activity.
func startLiveEvents(store *memorystorage.MemoryStore, bus *eventtest.FakeBus) {
	ctx := context.Background()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	eventTypes := []string{
		"user.login", "user.logout", "order.cancelled",
		"payment.received", "cart.updated", "notification.sent",
	}
	streamTypes := []id.StreamType{"User", "Order", "Payment", "Cart"}

	var counter int

	for range ticker.C {
		counter++
		aggID := id.NewStreamID()
		st := streamTypes[rand.IntN(len(streamTypes))]
		ref := id.NewStreamRef(st, aggID)
		et := eventTypes[rand.IntN(len(eventTypes))]

		payload, _ := json.Marshal(map[string]any{
			"source":  "live-demo",
			"counter": counter,
		})

		evt, _ := event.New(
			event.Type(et),
			aggID,
			st,
			event.Version(1),
			jsontext.Value(payload),
		)
		_ = store.Save(ctx, ref, []event.Event{evt}, event.Version(0)) //cqrs-lint:ignore(S003) demo with in-memory store: no signing needed
		_ = bus.Publish(ctx, evt)
	}
}
