// Package main is a minimal onboarding example for cqrs-htmx.
//
// It demonstrates:
//   - Creating an App with command and query dispatchers
//   - HTMX handlers with decode via mapper functions — DecodeForm for the
//     page's forms (htmx posts urlencoded) and DecodeJSON for API-shaped
//     endpoints (/api/audit, /api/items/search)
//   - Typed command/query handlers (no manual type assertions)
//   - Partial-render helpers: RenderPartialOrFull, OOBHTML, RenderIf (/demo/items)
//   - SSE live updates via Broadcaster
//   - Embedded HTMX script serving
//
// Run: go run . and open http://localhost:8096
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-sse"
	"github.com/larsartmann/httputil"
)

// listAuditQuery reads back the audit trail.
type listAuditQuery struct{}

func (q *listAuditQuery) Type() query.Type { return query.Type("ListAudit") }

// --- Actor metadata demo types ---

type auditRequest struct {
	Action string `json:"action"`
}

// auditCmd embeds *command.BasicCommand so the pipeline can attach the
// actor/user/correlation metadata copied from the request context.
type auditCmd struct {
	*command.BasicCommand
	Action string
}

type auditEntry struct {
	Action string `json:"action"`
	Actor  string `json:"actor"`
	User   string `json:"user"`
}

// --- Actor metadata demo ---

// auditStore records who did what, read back from command metadata.
type auditStore struct {
	mu      sync.RWMutex
	entries []auditEntry
}

var auditLog = &auditStore{}

func (s *auditStore) record(e auditEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
}

func (s *auditStore) list() []auditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]auditEntry, len(s.entries))
	copy(cp, s.entries)
	return cp
}

// demoActorMiddleware simulates what usermgmt's session middleware does in
// production: put the acting user (and its actor form) into the request
// context so decoders can propagate them into command/query metadata.
func demoActorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := cqrshtmx.GenerateUserID()
		ctx := cqrshtmx.WithUserID(r.Context(), uid)
		ctx = cqrshtmx.WithActorID(ctx, cqrshtmx.ActorIDFromUser(uid))
		ctx = cqrshtmx.WithCorrelationID(ctx, cqrshtmx.NewCorrelationID())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// --- Domain types ---

type createItemRequest struct {
	Name string `json:"name"`
}

type item struct {
	//cqrs-lint:ignore(A032) example DTO: simple string ID for demo
	ID   string `json:"id"`
	Name string `json:"name"`
}

// createItemCmd embeds *command.BasicCommand for Type()/StreamID()/ID().
type createItemCmd struct {
	*command.BasicCommand
	Name string
}

// listItemsQuery is a custom query.Query.
type listItemsQuery struct{}

func (q *listItemsQuery) Type() query.Type { return query.Type("ListItems") }

// listItemsPaginatedQuery carries pagination info.
type listItemsPaginatedQuery struct {
	pagination query.Pagination
}

func (q *listItemsPaginatedQuery) Type() query.Type { return query.Type("ListItemsPaginated") }

// searchItemsRequest is the request body for the QUERY-method showcase.
type searchItemsRequest struct {
	Filter string `json:"filter"`
}

// searchItemsQuery is the HTTP QUERY (RFC 10008) showcase: a read whose
// criteria travel in the request body instead of the URL — safe and
// idempotent like GET, body-carrying like POST.
type searchItemsQuery struct {
	Filter string `json:"filter"`
}

func (q *searchItemsQuery) Type() query.Type { return query.Type("SearchItems") }

// --- Typed command/query types (no mapper needed) ---

// greetCmd implements command.Command directly (no embedded BasicCommand)
// because DecodeJSONTyped creates a zero-value struct — an embedded pointer
// would be nil. This is the documented pattern for typed commands.
type greetCmd struct {
	aggID id.StreamID
	cmdID id.CommandID
	Name  string `json:"name"`
}

func (c *greetCmd) Type() command.Type    { return "Greet" }
func (c *greetCmd) StreamID() id.StreamID { return c.aggID }

// cqrs-lint:ignore(A001) typed command: DecodeJSONTyped requires manual methods (embedded *BasicCommand would be nil)
func (c *greetCmd) ID() id.CommandID { return c.cmdID }

// sumQuery is a typed query that implements query.Query directly.
type sumQuery struct {
	A int `json:"a"`
	B int `json:"b"`
}

func (q *sumQuery) Type() query.Type { return "Sum" }

// --- In-memory store + SSE broadcaster ---

var (
	broadcaster = cqrshtmx.NewBroadcaster()
	db          = &itemStore{}
)

type itemStore struct {
	mu    sync.RWMutex
	items []item
}

func (s *itemStore) add(name string) item {
	s.mu.Lock()
	defer s.mu.Unlock()
	it := item{ID: id.NewStreamID().String(), Name: name}
	s.items = append(s.items, it)
	broadcaster.Broadcast(sse.Event{
		Event: "itemCreated",
		Data:  fmt.Sprintf(`{"id":"%s","name":"%s"}`, it.ID, it.Name),
	})
	return it
}

func (s *itemStore) list() []item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]item, len(s.items))
	copy(cp, s.items)
	return cp
}

func (s *itemStore) listPaginated(p query.Pagination) query.PaginatedResult[item] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := uint(len(s.items))
	start := min((p.Page-1)*p.PageSize, total)
	end := min(start+p.PageSize, total)
	return query.NewPaginatedResult(s.items[start:end], total, p)
}

// newHandler builds the fully-wired HTTP handler. main serves it; the smoke
// test exercises it in-process.
func newHandler() http.Handler {
	// --- Command dispatcher ---
	cmdDisp := command.NewDispatcher()
	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = command.RegisterTyped(cmdDisp, "CreateItem",
		func(_ context.Context, c *createItemCmd) error {
			db.add(c.Name)
			return nil
		})

	// Typed command handler: no type assertion needed — the dispatcher
	// calls the handler with the concrete *greetCmd directly.
	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = command.RegisterTyped(cmdDisp, "Greet",
		func(_ context.Context, cmd *greetCmd) error {
			db.add("Hello, " + cmd.Name + "!")
			return nil
		})

	// The audit handler reads WHO issued the command straight from the
	// metadata the endpoint's decoder attached (actor propagation proof).
	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = command.RegisterTyped(cmdDisp, "AuditAction",
		func(_ context.Context, c *auditCmd) error {
			md := c.Metadata()
			auditLog.record(auditEntry{
				Action: c.Action,
				Actor:  md.ActorID.PrefixedString(),
				User:   md.UserID.Get().String(),
			})
			return nil
		})

	// --- Query dispatcher ---
	qryDisp := query.NewDispatcher()
	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = query.RegisterTyped(qryDisp, "ListItems",
		func(_ context.Context, _ *listItemsQuery) ([]item, error) {
			return db.list(), nil
		})
	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = query.RegisterTyped(qryDisp, "ListItemsPaginated",
		func(_ context.Context, q *listItemsPaginatedQuery) (query.PaginatedResult[item], error) {
			return db.listPaginated(q.pagination), nil
		})

	// Typed query handler: returns a concrete result type (int).
	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = query.RegisterTyped(qryDisp, "Sum",
		func(_ context.Context, q *sumQuery) (int, error) {
			return q.A + q.B, nil
		})

	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = query.RegisterTyped(qryDisp, "ListAudit",
		func(_ context.Context, _ *listAuditQuery) ([]auditEntry, error) {
			return auditLog.list(), nil
		})

	// QUERY-method showcase handler: case-insensitive name filter.
	//cqrs-lint:ignore(C028) example: error handling omitted for brevity
	_ = query.RegisterTyped(qryDisp, "SearchItems",
		func(_ context.Context, q *searchItemsQuery) ([]item, error) {
			matched := make([]item, 0)
			for _, it := range db.list() {
				if q.Filter == "" || strings.Contains(strings.ToLower(it.Name), strings.ToLower(q.Filter)) {
					matched = append(matched, it)
				}
			}
			return matched, nil
		})

	// --- Build the App ---
	app := cqrshtmx.MustNew(cqrshtmx.Config{
		Commands: cmdDisp,
		Queries:  qryDisp,
	})

	// --- Wire routes ---
	mux := http.NewServeMux()
	mux.Handle("GET /htmx.js", htmx.ScriptHandler())

	// POST /api/items — command handler. The index page's form posts
	// urlencoded (native htmx), so this uses the form decoder; the same
	// mapper shape as the JSON variant.
	mux.Handle("POST /api/items", app.Command(
		"CreateItem",
		cqrshtmx.DecodeForm(func(req createItemRequest) (command.Command, error) {
			core, err := command.New("CreateItem", id.NewStreamID())
			if err != nil {
				return nil, err
			}
			return &createItemCmd{BasicCommand: core, Name: req.Name}, nil
		}),
		cqrshtmx.WithSuccessStatus(201),
	))

	// GET /api/items — query handler
	mux.Handle("GET /api/items", app.Query(
		"ListItems",
		cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
			return &listItemsQuery{}, nil
		}),
		cqrshtmx.RenderJSON[[]item](),
	))

	// GET /api/items/paginated — paginated query handler
	mux.Handle("GET /api/items/paginated", app.Query(
		"ListItemsPaginated",
		cqrshtmx.DecodeFormQueryWithRequest(func(r *http.Request, _ struct{}) (query.Query, error) {
			return &listItemsPaginatedQuery{pagination: cqrshtmx.DecodePagination(r)}, nil
		}),
		cqrshtmx.RenderPaginatedJSON[item](),
	))

	// QUERY /api/items/search — HTTP QUERY method (RFC 10008): a safe,
	// idempotent read whose criteria travel as a JSON body, so complex
	// filters never get squeezed into URL params. App.Query is
	// method-agnostic; the mux pattern carries the method.
	mux.Handle("QUERY /api/items/search", app.Query(
		"SearchItems",
		cqrshtmx.DecodeJSONQuery(func(req searchItemsRequest) (query.Query, error) {
			return &searchItemsQuery{Filter: req.Filter}, nil
		}),
		cqrshtmx.RenderJSON[[]item](),
	))

	// POST /api/items/search — the fallback twin: the identical handler for
	// clients and proxies that cannot send QUERY (htmx 2.x, older WAFs).
	// Same JSON body, ordinary POST. With MethodOverrideMiddleware a
	// POST + "X-HTTP-Method-Override: QUERY" header reaches the QUERY route
	// above and this registration becomes unnecessary — see
	// docs/guides/http-query-method.md (the fallback ladder).
	mux.Handle("POST /api/items/search", app.Query(
		"SearchItems",
		cqrshtmx.DecodeJSONQuery(func(req searchItemsRequest) (query.Query, error) {
			return &searchItemsQuery{Filter: req.Filter}, nil
		}),
		cqrshtmx.RenderJSON[[]item](),
	))

	// POST /api/greet — typed command handler (no mapper, no type assertion).
	// Form-decoded: the page form is its only caller.
	mux.Handle("POST /api/greet", cqrshtmx.CommandTyped[*greetCmd](
		app, "Greet",
		cqrshtmx.DecodeFormTyped[*greetCmd](),
		cqrshtmx.WithSuccessStatus(201),
	))

	// POST /api/sum — typed query handler (returns int, no mapper).
	// Form-decoded: the page form posts a=3&b=4.
	mux.Handle("POST /api/sum", cqrshtmx.QueryTyped[*sumQuery, int](
		app, "Sum",
		cqrshtmx.DecodeFormQueryTyped[*sumQuery](),
		cqrshtmx.RenderJSON[int](),
	))

	// POST /api/audit — actor-metadata demo: the dispatch path copies the
	// acting user/actor/correlation ID out of the request context onto the
	// command automatically (any command exposing ApplyOptions qualifies,
	// so since root v4.11.0 no manual option application is needed), and
	// the handler reads it back from the command metadata.
	mux.Handle("POST /api/audit", demoActorMiddleware(app.Command(
		"AuditAction",
		cqrshtmx.DecodeJSONWithRequest(func(r *http.Request, req auditRequest) (command.Command, error) {
			core, err := command.New("AuditAction", id.NewStreamID())
			if err != nil {
				return nil, err
			}
			return &auditCmd{BasicCommand: core, Action: req.Action}, nil
		}),
		cqrshtmx.WithSuccessStatus(202),
	)))

	// GET /api/audit — the recorded audit trail (actor + action pairs).
	mux.Handle("GET /api/audit", app.Query(
		"ListAudit",
		cqrshtmx.DecodeJSONQueryTyped[*listAuditQuery](),
		cqrshtmx.RenderJSON[[]auditEntry](),
	))

	// GET /demo/items — partial-render showcase. RenderPartialOrFull picks the
	// shape: a direct visit gets the full templ page, an HTMX request gets just
	// the list fragment plus the OOBHTML block that refreshes #item-count.
	mux.Handle("GET /demo/items", app.Query(
		"ListItems",
		cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
			return &listItemsQuery{}, nil
		}),
		cqrshtmx.RenderPartialOrFull(
			func(items []item) cqrshtmx.TemplComponent { return itemsPartial(items) },
			func(items []item) cqrshtmx.TemplComponent { return demoPage(items) },
		),
	))

	// GET /demo/items/count — RenderIf demo: the custom predicate inspects
	// HX-Target, so an HTMX request aimed at #item-count gets the bare count
	// while every other request gets the full showcase page.
	mux.Handle("GET /demo/items/count", app.Query(
		"ListItems",
		cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
			return &listItemsQuery{}, nil
		}),
		cqrshtmx.RenderIf(
			func(r *http.Request) bool { return htmx.Target(r) == "#item-count" },
			renderItemCount,
			renderDemoPage,
		),
	))

	// GET /api/events — SSE live updates
	mux.HandleFunc("GET /api/events", func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		ch := broadcaster.Subscribe()
		defer broadcaster.Unsubscribe(ch)
		go stream.Heartbeat(r.Context(), 15*time.Second)
		for {
			select {
			case <-stream.Context().Done():
				return
			case evt := <-ch:
				if err := stream.Send(evt); err != nil {
					return
				}
			}
		}
	})

	mux.HandleFunc("GET /", indexPage)

	// Standard health endpoints: /health, /health/live, /health/ready.
	// Registered manually (not via httputil.RegisterHealth) so the readiness
	// probe checks that both dispatchers are wired — RegisterHealth would
	// claim /health/ready with the default probe and a duplicate registration
	// panics the mux.
	mux.HandleFunc("GET /health", httputil.HealthHandler())
	mux.HandleFunc("GET /health/live", httputil.LiveHandler())
	mux.HandleFunc("GET /health/ready", httputil.ReadyHandlerWithProbe(func() bool {
		return app.HasCommands() && app.HasQueries()
	}))

	// Compression wraps the mux for smaller JSON/HTML payloads (5-10x for text).
	return httputil.Compression(httputil.DefaultCompressionConfig())(mux)
}

// renderItemCount is the RenderIf "match" branch: the bare count text.
func renderItemCount(w http.ResponseWriter, _ *http.Request, result any) error {
	items, err := asItems(result)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, itemCountLabel(items))
	return nil
}

// renderDemoPage is the RenderIf "no match" branch: the full showcase page.
func renderDemoPage(w http.ResponseWriter, r *http.Request, result any) error {
	items, err := asItems(result)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return demoPage(items).Render(r.Context(), w)
}

func asItems(result any) ([]item, error) {
	items, ok := result.([]item)
	if !ok {
		return nil, errorfamily.NewRejection("unexpected_result_type",
			fmt.Sprintf("unexpected result type %T", result))
	}
	return items, nil
}

func main() {
	handler := newHandler()

	// 8096 avoids clashing with unrelated services on well-known dev ports;
	// the e2e battery overrides it (BASIC_DEMO_ADDR=:18932) the same way
	// admin-demo and datastar-demo do, so the shipped example itself stays
	// the artifact under test.
	addr := envOr("BASIC_DEMO_ADDR", ":8096")
	// Serve with real timeouts via httputil.NewServer (never bare http.ListenAndServe,
	// which sets no Read/Write/Idle timeouts). See docs/guides/leveraging-httputil.md.
	cfg := httputil.DefaultServerConfig()
	cfg.Addr = addr
	srv, err := httputil.NewServer(cfg, handler)
	if err != nil {
		log.Fatalf("invalid server config: %v", err)
	}
	fmt.Printf("cqrs-htmx Basic Example\nListening on http://localhost%s\n", addr)
	if err := <-srv.Start(); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func indexPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><script src="/htmx.js"></script></head>
<body>
  <h1>cqrs-htmx Basic Example</h1>

  <p><a href="/demo/items">Partial-render showcase (RenderPartialOrFull + OOBHTML + RenderIf)</a></p>

  <h2>Untyped (mapper-based)</h2>
  <form hx-post="/api/items" hx-swap="none">
    <input name="name" placeholder="Item name" required>
    <button>Add</button>
  </form>
  <div id="items" hx-get="/api/items" hx-trigger="load, itemCreated from:body">Loading...</div>

  <h2>Typed command: Greet</h2>
  <form hx-post="/api/greet" hx-swap="none">
    <input name="name" placeholder="Your name" required>
    <button>Greet</button>
  </form>

  <h2>Typed query: Sum</h2>
  <form hx-post="/api/sum" hx-target="#sum-result">
    <input name="a" type="number" value="3" required>
    <input name="b" type="number" value="4" required>
    <button>Calculate</button>
  </form>
  <pre id="sum-result"></pre>

  <script>
    var es = new EventSource('/api/events');
    es.addEventListener('itemCreated', function() {
      htmx.ajax('GET', '/api/items', {target: '#items'});
    });
  </script>
</body>
</html>`)
}
