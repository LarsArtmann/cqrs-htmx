// Package main is a runnable demo of the cqrs-htmx frontend sync protocol
// (ADR-0056): an offline-tolerant command queue, batched typed push,
// cursor-based event pull with a permission filter, per-command conflict
// outcomes, and a served OpenAPI contract for both sync endpoints.
//
// It makes the snippets in docs/guides/frontend-sync.md executable:
//
//   - POST /api/notes is the offline-queueable endpoint (data-sync-command-type)
//   - POST /sync/push replays queued commands in batches with classified outcomes
//   - GET /sync/pull serves the permission-filtered event feed (cursor + ETag)
//   - confidential notes ride the "SecretNote" stream type and are invisible
//     to pulls unless the demo_role cookie says "admin" (the filter seam)
//   - duplicate note names are rejected (event.Rejection) — watch the form's
//     rejected state and retry button instead of a silent loss
//
// Run: go run . and open http://localhost:8098 — then try the offline
// walkthrough in README.md.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"sync"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/openapi"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// noteForm is the HTML form body of the offline-queueable endpoint.
type noteForm struct {
	Name         string `json:"name"`
	Confidential bool   `json:"confidential"`
}

// createNoteCmd is a typed command (accessor methods, not field access).
type createNoteCmd struct {
	command.BasicCommand
	name         string
	confidential bool
}

func (c *createNoteCmd) NoteName() string     { return c.name }
func (c *createNoteCmd) Secret() bool         { return c.confidential }

func (c *createNoteCmd) StreamTypeName() string {
	if c.confidential {
		return "SecretNote"
	}

	return "Note"
}

// note is the demo's server-side read model (what a real projection would
// own; kept inline because this demo has exactly one query).
type note struct {
	Name         string `json:"name"`
	Confidential bool   `json:"confidential"`
}

// noteBoard is the write-side state: the journal IS the source of truth; the
// name set exists only to demonstrate conflict rejection (duplicate names).
type noteBoard struct {
	mu    sync.Mutex
	names map[string]bool
	notes []note
}

func (b *noteBoard) create(name string, confidential bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if name == "" {
		return errorfamily.NewRejection("note.name_required", "note name is required")
	}

	if b.names[name] {
		return errorfamily.NewRejection("note.duplicate_name", "a note with this name already exists")
	}

	b.names[name] = true
	b.notes = append(b.notes, note{Name: name, Confidential: confidential})

	return nil
}

func (b *noteBoard) list() []note {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]note, len(b.notes))
	copy(out, b.notes)

	return out
}

// newHandler builds the fully-wired HTTP handler. main serves it; the smoke
// test exercises it in-process.
func newHandler() http.Handler {
	store := memorystorage.NewMemoryStore()
	board := &noteBoard{names: map[string]bool{}}
	broadcaster := cqrshtmx.NewBroadcaster()
	ackHook := broadcaster.BroadcastOnAck()

	cmdDisp := command.NewDispatcher()
	_ = command.RegisterTyped(cmdDisp, "CreateNote", func(ctx context.Context, c *createNoteCmd) error {
		if err := board.create(c.NoteName(), c.Secret()); err != nil {
			return err
		}

		streamType := id.StreamType(c.StreamTypeName())
		aggID := id.NewStreamID()
		evt, err := event.New(event.Type("Note.created"), aggID, streamType, event.Version(1),
			map[string]string{"name": c.NoteName()})
		if err != nil {
			return err
		}

		return store.AppendBatch(ctx, id.StreamRef{Type: streamType, ID: aggID}, []event.Event{evt})
	})

	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp, AfterDispatch: ackHook})

	spec := cqrshtmx.AddSyncOpenAPI(openapi.New("sync-demo", "1.0.0"), "/sync/pull", "/sync/push")
	openAPIHandler, err := cqrshtmx.OpenAPISpecHandler(spec)
	if err != nil {
		log.Fatalf("openapi: %v", err)
	}

	mux := http.NewServeMux()

	// Embedded JS assets: HTMX core + the sync worker/client pair.
	mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())
	mux.Handle("GET /sync-worker.js", cqrshtmx.SyncWorkerHandler())
	mux.Handle("GET /sync-client.js", cqrshtmx.SyncClientHandler())

	mux.HandleFunc("GET /{$}", indexHandler)

	// The offline-queueable command endpoint: the sync client queues this
	// form when offline and replays it as a typed batch envelope later.
	mux.Handle("POST /api/notes", app.Command("CreateNote",
		cqrshtmx.DecodeForm(func(f noteForm) (command.Command, error) {
			core, err := command.New("CreateNote", id.NewStreamID())
			if err != nil {
				return nil, err
			}

			return &createNoteCmd{BasicCommand: *core, name: f.Name, confidential: f.Confidential}, nil
		}),
		cqrshtmx.WithSuccessStatus(http.StatusCreated),
	))

	// The sync halves (ADR-0056): batched command push up, filtered event
	// pull down. The filter hides confidential notes unless the demo_role
	// cookie says admin — the permission seam, fail-closed.
	mux.Handle("POST /sync/push", app.SyncPushHandler())
	mux.Handle("GET /sync/pull", cqrshtmx.SyncPullHandler(store,
		cqrshtmx.WithSyncPullBackendID("sync-demo-v1"),
		cqrshtmx.WithSyncPullFilter(func(r *http.Request, evt event.Event) bool {
			if evt.StreamType() != "SecretNote" {
				return true
			}

			role, err := r.Cookie("demo_role")

			return err == nil && role.Value == "admin"
		}),
	))

	// ACK delivery for the sync indicator (SSE, same shape as any cqrs-htmx feed).
	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		broadcaster.ServeSSE(w, r)
	})

	// Server truth: what a real projection would serve (compare against the
	// offline cache panel — this one is empty while you are offline).
	mux.HandleFunc("GET /api/notes", notesHandler(board))
	mux.Handle("GET /openapi.json", openAPIHandler)

	return mux
}

func main() {
	addr := flag.String("addr", ":8098", "listen address")
	flag.Parse()

	// ReadHeaderTimeout keeps security scanners happy; no WriteTimeout on
	// purpose — SSE needs an unbounded write window.
	server := &http.Server{
		Addr:              *addr,
		Handler:           newHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("sync-demo listening on %s (open http://localhost%s)", *addr, *addr)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

func notesHandler(board *noteBoard) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_ = cqrshtmx.WriteJSON(w, http.StatusOK, board.list())
	}
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>sync-demo — offline notes</title>
  <script src="/htmx.js"></script>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 44rem; margin: 2rem auto; padding: 0 1rem; }
    #sync-indicator { position: fixed; top: 1rem; right: 1rem; padding: .25rem .75rem; border-radius: 999px;
      background: #dcfce7; font-size: .875rem; }
    #sync-indicator[data-sync-state="offline"] { background: #fef9c3; }
    #sync-indicator[data-sync-state="rejected"] { background: #fee2e2; }
    form { display: flex; gap: .5rem; margin: 1rem 0; }
    input[type=text] { flex: 1; padding: .375rem .5rem; }
    .panel { border: 1px solid #e5e7eb; border-radius: .5rem; padding: 1rem; margin: 1rem 0; }
    .panel h2 { margin-top: 0; font-size: 1rem; }
    ul { padding-left: 1.25rem; }
    .confidential { color: #92400e; }
    label.role { display: inline-flex; gap: .35rem; align-items: center; font-size: .875rem; }
  </style>
</head>
<body data-sync-pull-url="/sync/pull" data-sync-push-url="/sync/push">
  <div id="sync-indicator" data-sync-status="idle" data-sync-state="offline">checking…</div>
  <h1>offline notes</h1>
  <p>A runnable version of the <a href="https://github.com/LarsArtmann/cqrs-htmx">cqrs-htmx</a>
     frontend sync protocol: queue commands offline, batch-push them on reconnect,
     read the cached event feed while offline. The contract is served at
     <a href="/openapi.json">/openapi.json</a>.</p>

  <form data-sync-command-type="CreateNote" hx-post="/api/notes">
    <input type="text" name="name" placeholder="note name (must be unique)" required autocomplete="off">
    <label><input type="checkbox" name="confidential" value="true"> confidential</label>
    <button type="submit">Add (queued when offline)</button>
  </form>
  <label class="role"><input type="checkbox" id="admin-role"> admin (pull confidential notes)</label>

  <div class="panel">
    <h2>cached event feed — offline reads via <code>window.cqrsSync.getEvents()</code></h2>
    <ul id="cached-feed"><li>…waiting for the first pull</li></ul>
  </div>

  <div class="panel">
    <h2>server truth — what a projection would serve (empty while you are offline)</h2>
    <div hx-get="/api/notes" hx-trigger="load, every 3s"><em>loading…</em></div>
  </div>

  <p style="font-size:.875rem;color:#6b7280">Try it: DevTools → Network → Offline, add two notes
     (watch the queue), back online (batch flush), duplicate a name (rejected + retry), tick
     admin and reload (confidential notes appear in the feed).</p>

  <script src="/sync-client.js"></script>
  <script>
    function noteName(evt) {
      if (evt.payloadEncoding === "json" && evt.payload && typeof evt.payload.name === "string") {
        return evt.payload.name;
      }
      if (evt.payloadB64) {
        try { return JSON.parse(atob(evt.payloadB64)).name; } catch { /* opaque bytes: show the id */ }
      }
      return evt.eventId;
    }

    async function renderCachedFeed() {
      const events = await window.cqrsSync.getEvents();
      const feed = document.getElementById("cached-feed");
      feed.replaceChildren(...events
        .filter((e) => e.type === "Note.created")
        .slice(-20).reverse()
        .map((e) => {
          const li = document.createElement("li");
          li.textContent = noteName(e) + (e.streamType === "SecretNote" ? " · confidential" : "");
          if (e.streamType === "SecretNote") li.className = "confidential";
          return li;
        }));
    }

    document.addEventListener("cqrshtmx:sync-events", renderCachedFeed);

    document.getElementById("admin-role").addEventListener("change", function () {
      document.cookie = "demo_role=" + (this.checked ? "admin" : "user") + ";path=/;samesite=strict";
      location.reload();
    });
  </script>
</body>
</html>`
