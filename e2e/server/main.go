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
	"flag"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	dashboardui "github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// Server timeout posture for the e2e server. ReadHeader/Read/Idle timeouts
// mirror setup.Bundle HTTP server defaults; there is deliberately NO
// WriteTimeout — a write timeout kills every SSE stream mid-flight (setup Run
// paths disable it for exactly this reason).
const (
	e2eReadHeaderTimeout = 5 * time.Second
	e2eReadTimeout       = 10 * time.Second
	e2eIdleTimeout       = 60 * time.Second
)

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

	//cqrs-lint:ignore(C017) ephemeral Playwright test server — in-memory DLQ is intentional, the process is disposable per test run
	deadStore := projectionhost.NewMemoryDeadLetterStore()

	var host *projectionhost.Host

	if seekable, ok := any(dstore).(event.SeekableJournal); ok {
		h, hErr := projectionhost.New(seekable, memorystorage.NewMemoryCheckpointStore())
		if hErr != nil {
			log.Fatalf("projectionhost.New: %v", hErr)
		}

		host = h
	}

	dash, err := dashboardui.New(dashboardui.Config{
		EventSource:     dstore,
		Journal:         dstore,
		CommandJournal:  emptyCommandJournal{},
		QueryJournal:    emptyQueryJournal{},
		SnapshotStore:   emptySnapshotStore{},
		DeadLetterStore: deadStore,
		ProjectionHost:  host,
	})
	if err != nil {
		log.Fatalf("dashboardui.New: %v", err)
	}

	if err := mountLoginPage(mux); err != nil {
		log.Fatalf("mountLoginPage: %v", err)
	}

	seedDashboard(dstore, host, deadStore)
	dash.Mount(mux, "/dashboard/")

	// --- HTML page --- ({$}: exact root only, else it conflicts with the
	// /dashboard/ subtree's method-agnostic pattern)
	mux.HandleFunc("GET /{$}", indexHandler)

	// --- SSE endpoint (sync:ack delivery) ---
	mux.HandleFunc("GET /events", sseHandler(broadcaster))

	// --- Command endpoint: POST form data -> store -> broadcast ACK ---
	mux.HandleFunc("POST /api/items", itemsPostHandler(store, ackHook))

	// --- ADR-0056 sync halves: typed commands batch-push through
	// POST /sync/push; events journal back via GET /sync/pull. The pull
	// backend identity is rotatable (debug endpoint) so specs can pin the
	// sync:reset-on-backendId-change contract. ---
	syncStore := memorystorage.NewMemoryStore()
	seedSyncEvents(syncStore)

	cmdDisp := command.NewDispatcher()
	//cqrs-lint:ignore(C028) e2e server: event append failures fail the test command outright
	_ = command.RegisterTyped(cmdDisp, "SyncAddItem", func(ctx context.Context, c *syncAddItemCmd) error {
		aggID := id.NewStreamID()
		ref := id.StreamRef{Type: itemStreamType, ID: aggID}

		evt, err := event.New(event.Type("Item.synced"), aggID, itemStreamType, event.Version(1),
			map[string]string{"name": c.ItemName()}, event.WithSchemaVersion(1))
		if err != nil {
			return err //nolint:wrapcheck // e2e: raw construction error is the intended command failure
		}

		if err := syncStore.AppendBatch(
			ctx,
			ref,
			[]event.Event{evt},
		); err != nil { //cqrs-lint:ignore(C033) e2e: the raw store error is the intended command failure
			return err //nolint:wrapcheck // e2e: the raw store error is the intended command failure
		}

		store.add(c.ItemName())

		return nil
	})

	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp, AfterDispatch: ackHook})

	mux.Handle("POST /api/sync-items", app.Command("SyncAddItem",
		cqrshtmx.DecodeForm(func(form syncItemForm) (command.Command, error) {
			core, err := command.New("SyncAddItem", id.NewStreamID())
			if err != nil {
				return nil, err //nolint:wrapcheck // e2e: raw construction error is the intended decode failure
			}

			return &syncAddItemCmd{BasicCommand: *core, name: form.Name}, nil
		}),
		cqrshtmx.WithSuccessStatus(http.StatusCreated),
	))

	mux.Handle("POST /sync/push", app.SyncPushHandler())

	pullBackendA := cqrshtmx.SyncPullHandler(syncStore, cqrshtmx.WithSyncPullBackendID("e2e-backend-A"))
	pullBackendB := cqrshtmx.SyncPullHandler(syncStore, cqrshtmx.WithSyncPullBackendID("e2e-backend-B"))

	var rotated atomic.Bool

	mux.Handle("GET /sync/pull", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rotated.Load() {
			pullBackendB.ServeHTTP(w, r)

			return
		}

		pullBackendA.ServeHTTP(w, r)
	}))
	// TEST-ONLY debug endpoint: rotating the pull backend's identity is the
	// only way to e2e-pin the client's sync:reset behavior (cache wipe +
	// queue retention on backend switch). Never ship this in a consumer app.
	mux.HandleFunc("POST /api/debug/rotate-backend", func(w http.ResponseWriter, _ *http.Request) {
		rotated.Store(true)
		w.WriteHeader(http.StatusNoContent)
	})

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
		IdleTimeout:       e2eIdleTimeout,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
