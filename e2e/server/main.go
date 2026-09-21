// Package main is a minimal HTTP server for Playwright E2E testing of the
// cqrs-htmx offline sync stack (sync-worker.js + sync-client.js).
//
// It serves an HTML page with HTMX + the sync client, mounts the sync-worker.js
// and sync-client.js handlers, provides a POST endpoint that stores items and
// broadcasts sync:ack events via SSE, and exposes debug endpoints for test
// assertions.
//
// Usage:
//
//	GOEXPERIMENT=jsonv2 go run . [-addr :18923]
package main

import (
	"context"
	"encoding/json/v2"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	dashboardui "github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-sse"
)

// Server timeout posture for the e2e server (mirrors setup.Bundle's HTTP
// server defaults; SSE endpoints need generous write windows).
const (
	e2eReadHeaderTimeout = 5 * time.Second
	e2eReadTimeout       = 10 * time.Second
	e2eWriteTimeout      = 30 * time.Second
	e2eIdleTimeout       = 60 * time.Second
)

type emptyCommandJournal struct{}

func (emptyCommandJournal) ReadAll(context.Context) ([]*command.PersistedCommand, error) {
	return nil, nil
}

type emptyQueryJournal struct{}

func (emptyQueryJournal) ReadAllQueries(context.Context) ([]*query.PersistedQuery, error) {
	return nil, nil
}

type emptySnapshotStore struct{}

func (emptySnapshotStore) Save(context.Context, snapshot.Snapshot) error { return nil }
func (emptySnapshotStore) Delete(context.Context, id.StreamRef) error    { return nil }
func (emptySnapshotStore) Load(context.Context, id.StreamRef) (*snapshot.Snapshot, error) {
	return nil, nil
}

func (emptySnapshotStore) LoadAtVersion(context.Context, id.StreamRef, event.Version) (*snapshot.Snapshot, error) {
	return nil, nil
}

// demoProjection is a minimal projection.Projection: it processes every
// event without error so the worker stays healthy for badge assertions.
type demoProjection struct{}

func (demoProjection) Name() string                                  { return "demo-projection" }
func (demoProjection) Handle(_ context.Context, _ event.Event) error { return nil }
func (demoProjection) EventTypes() []event.Type                      { return nil }

// seedDashboard appends a few generic events across two streams and starts
// the projection host, so the browser-truth specs assert real tables, badges,
// and worker states instead of empty states.
func seedDashboard(store *memorystorage.MemoryStore, host *projectionhost.Host) {
	ctx := context.Background()

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
			)
			if eErr != nil {
				log.Fatalf("event.New: %v", eErr)
			}

			if aErr := store.AppendBatch(ctx, ref, []event.Event{evt}); aErr != nil {
				log.Fatalf("AppendBatch: %v", aErr)
			}
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
}

func main() {
	addr := flag.String("addr", ":18923", "listen address")

	flag.Parse()

	store := &itemStore{mu: sync.Mutex{}, items: nil}
	broadcaster := cqrshtmx.NewBroadcaster()
	ackHook := broadcaster.BroadcastOnAck()

	mux := http.NewServeMux()

	// --- Static JS assets (embedded via go:embed in cqrs-htmx) ---
	mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())
	mux.Handle("GET /sync-worker.js", cqrshtmx.SyncWorkerHandler())
	mux.Handle("GET /sync-client.js", cqrshtmx.SyncClientHandler())

	// --- Dashboard (screenshots + browser-truth e2e; empty journal renders
	// every page's empty state, which is itself an adopted surface). Empty
	// capability stubs unlock the command/query/DLQ/projection/snapshot
	// panels so all nine pages render without a full event-sourced stack. ---
	dstore := memorystorage.NewMemoryStore()

	var host *projectionhost.Host

	if seekable, ok := any(dstore).(event.SeekableJournal); ok {
		h, hErr := projectionhost.New(seekable, memorystorage.NewMemoryCheckpointStore())
		if hErr != nil {
			log.Fatalf("projectionhost.New: %v", hErr)
		}

		host = h
	}

	dash, err := dashboardui.New(dashboardui.Config{
		EventSource:    dstore,
		Journal:        dstore,
		CommandJournal: emptyCommandJournal{},
		QueryJournal:   emptyQueryJournal{},
		SnapshotStore:  emptySnapshotStore{},
		//cqrs-lint:ignore(C017) ephemeral Playwright test server — in-memory DLQ is intentional, the process is disposable per test run
		DeadLetterStore: projectionhost.NewMemoryDeadLetterStore(),
		ProjectionHost:  host,
	})
	if err != nil {
		log.Fatalf("dashboardui.New: %v", err)
	}

	seedDashboard(dstore, host)
	dash.Mount(mux, "/dashboard/")

	// --- HTML page --- ({$}: exact root only, else it conflicts with the
	// /dashboard/ subtree's method-agnostic pattern)
	mux.HandleFunc("GET /{$}", indexHandler)

	// --- SSE endpoint (sync:ack delivery) ---
	mux.HandleFunc("GET /events", sseHandler(broadcaster))

	// --- Command endpoint: POST form data -> store -> broadcast ACK ---
	mux.HandleFunc("POST /api/items", itemsPostHandler(store, ackHook))

	// --- Query endpoint: list stored items (for test assertions) ---
	mux.HandleFunc("GET /api/debug/items", itemsDebugHandler(store))

	// --- Health check (used by Playwright webServer readiness probe) ---
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("sync-e2e server listening on %s", *addr)

	// Explicit timeouts (G114): the e2e server is local-only, but the race
	// detector and security scanners flag a bare ListenAndServe, and timeouts
	// keep a hung Playwright run from pinning connections forever. Values
	// mirror setup.Bundle's server posture (ReadHeaderTimeout 5s, idle 60s).
	server := &http.Server{ //nolint:exhaustruct,exhaustruct_v5 // optional server knobs intentionally default
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: e2eReadHeaderTimeout,
		ReadTimeout:       e2eReadTimeout,
		WriteTimeout:      e2eWriteTimeout,
		IdleTimeout:       e2eIdleTimeout,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// --- HTML page ---

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Sync E2E Test</title>
  <script src="/htmx.js"></script>
</head>
<body data-sse-url="/events">
  <div id="sync-indicator" data-sync-status="idle">Synced</div>
  <main>
    <h1>Items</h1>
    <div data-sync-target>
      <form id="add-form" hx-post="/api/items" hx-target="#item-list" hx-swap="beforeend">
        <input type="text" name="name" placeholder="Item name" required autocomplete="off">
        <button type="submit">Add</button>
      </form>
      <ul id="item-list"></ul>
    </div>
  </main>
  <script src="/sync-client.js"></script>
</body>
</html>`

// --- SSE handler ---

func sseHandler(broadcaster *cqrshtmx.Broadcaster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		ch := broadcaster.Subscribe()
		defer broadcaster.Unsubscribe(ch)

		for {
			select {
			case <-stream.Context().Done():
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}

				if err := stream.Send(evt); err != nil {
					return
				}
			}
		}
	}
}

// --- Items command handler ---

func itemsPostHandler(store *itemStore, ackHook cqrshtmx.AfterDispatchHook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)

			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			http.Error(w, "name required", http.StatusBadRequest)

			return
		}

		store.add(name)

		cmdID := cqrshtmx.CommandIDFromRequest(r)
		if cmdID != "" {
			ackHook(r.Context(), r, nil)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<li data-sync-state="confirmed">%s</li>`, html.EscapeString(name))
	}
}

// --- Debug endpoint (test assertions) ---

func itemsDebugHandler(store *itemStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items := store.list()

		w.Header().Set("Content-Type", "application/json")
		_ = json.MarshalWrite(w, items)
	}
}

// --- In-memory item store ---

type itemStore struct {
	mu    sync.Mutex
	items []string
}

func (s *itemStore) add(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = append(s.items, name)
}

func (s *itemStore) list() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]string, len(s.items))
	copy(out, s.items)

	return out
}
