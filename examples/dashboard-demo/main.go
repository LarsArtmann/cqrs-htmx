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
	"net/http"
	"time"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
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
		ReadOnly:       false,
		PageSize:       25,
	})
	if err != nil {
		log.Fatalf("dashboard: %v", err)
	}

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
	for i, name := range []string{"Alice", "Bob", "Charlie", "Diana"} {
		aggID := id.NewStreamID()
		ref := id.NewStreamRef("User", aggID)

		payload, _ := json.Marshal(map[string]any{
			"name":  name,
			"email": fmt.Sprintf("%s@example.com", name),
		})

		created, _ := event.New("user.created", aggID, "User", event.Version(1), jsontext.Value(payload))
		_ = store.Save(ctx, ref, []event.Event{created}, event.Version(0))

		renamed, _ := event.New("user.renamed", aggID, "User", event.Version(2), map[string]any{"name": name + " Jr."})
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
	for i := 1; i <= 3; i++ {
		aggID := id.NewStreamID()
		ref := id.NewStreamRef("Order", aggID)

		placed, _ := event.New("order.placed", aggID, "Order", event.Version(1), map[string]any{
			"customerId": fmt.Sprintf("cust-%d", i),
			"total":      float64(i * 2999),
			"items":      i,
		})
		_ = store.Save(ctx, ref, []event.Event{placed}, event.Version(0))

		shipped, _ := event.New("order.shipped", aggID, "Order", event.Version(2), map[string]any{
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

	log.Println("Demo data seeded: 4 users, 3 orders, commands, queries, 1 snapshot")
}
