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
	"encoding/json/v2"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"
	"sync"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func main() {
	addr := flag.String("addr", ":18923", "listen address")

	flag.Parse()

	store := &itemStore{}
	broadcaster := cqrshtmx.NewBroadcaster()
	ackHook := broadcaster.BroadcastOnAck()

	mux := http.NewServeMux()

	// --- Static JS assets (embedded via go:embed in cqrs-htmx) ---
	mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())
	mux.Handle("GET /sync-worker.js", cqrshtmx.SyncWorkerHandler())
	mux.Handle("GET /sync-client.js", cqrshtmx.SyncClientHandler())

	// --- HTML page ---
	mux.HandleFunc("GET /", indexHandler)

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

	if err := http.ListenAndServe(*addr, mux); err != nil {
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

func sseHandler(bc *cqrshtmx.Broadcaster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stream := cqrshtmx.NewSSEStream(w, r)
		defer func() { _ = stream.Close() }()

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		ch := bc.Subscribe()
		defer bc.Unsubscribe(ch)

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
