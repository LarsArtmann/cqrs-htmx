// Package main is a minimal onboarding example for cqrs-htmx.
//
// It demonstrates:
//   - Creating an App with command and query dispatchers
//   - HTMX handlers with JSON decoding via mapper functions
//   - Typed command/query handlers (no manual type assertions)
//   - SSE live updates via Broadcaster
//   - Embedded HTMX script serving
//
// Run: go run . and open http://localhost:8096
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// --- Domain types ---

type createItemRequest struct {
	Name string `json:"name"`
}

type item struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// createItemCmd is a custom command.Command carrying the decoded request data.
type createItemCmd struct {
	typ   command.Type
	aggID id.StreamID
	cmdID id.CommandID
	Name  string
}

func (c *createItemCmd) Type() command.Type    { return c.typ }
func (c *createItemCmd) StreamID() id.StreamID { return c.aggID }
func (c *createItemCmd) ID() id.CommandID      { return c.cmdID }

// listItemsQuery is a custom query.Query.
type listItemsQuery struct{}

func (q *listItemsQuery) Type() query.Type { return query.Type("ListItems") }

// listItemsPaginatedQuery carries pagination info.
type listItemsPaginatedQuery struct {
	pagination query.Pagination
}

func (q *listItemsPaginatedQuery) Type() query.Type { return query.Type("ListItemsPaginated") }

// --- Typed command/query types (no mapper needed) ---

// greetCmd is a typed command that implements command.Command directly.
// The JSON body shape matches the struct fields (minus the command methods).
type greetCmd struct {
	aggID id.StreamID
	cmdID id.CommandID
	Name  string `json:"name"`
}

func (c *greetCmd) Type() command.Type    { return "Greet" }
func (c *greetCmd) StreamID() id.StreamID { return c.aggID }
func (c *greetCmd) ID() id.CommandID      { return c.cmdID }

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
	broadcaster.Broadcast(cqrshtmx.SSEEvent{
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

func main() {
	// --- Command dispatcher ---
	cmdDisp := command.NewDispatcher()
	_ = cmdDisp.Register(command.Type("CreateItem"),
		func(_ context.Context, cmd command.Command) error {
			if c, ok := cmd.(*createItemCmd); ok {
				db.add(c.Name)
			}
			return nil
		})

	// Typed command handler: no type assertion needed — the dispatcher
	// calls the handler with the concrete *greetCmd directly.
	_ = command.RegisterTyped(cmdDisp, "Greet",
		func(_ context.Context, cmd *greetCmd) error {
			db.add("Hello, " + cmd.Name + "!")
			return nil
		})

	// --- Query dispatcher ---
	qryDisp := query.NewDispatcher()
	_ = qryDisp.Register(query.Type("ListItems"),
		func(_ context.Context, _ query.Query) (any, error) {
			return db.list(), nil
		})
	_ = qryDisp.Register(query.Type("ListItemsPaginated"),
		func(_ context.Context, q query.Query) (any, error) {
			if pq, ok := q.(*listItemsPaginatedQuery); ok {
				return db.listPaginated(pq.pagination), nil
			}
			return db.listPaginated(query.NewPagination(1, 10)), nil
		})

	// Typed query handler: returns a concrete result type (int).
	_ = query.RegisterTyped(qryDisp, "Sum",
		func(_ context.Context, q *sumQuery) (int, error) {
			return q.A + q.B, nil
		})

	// --- Build the App ---
	app := cqrshtmx.MustNew(cqrshtmx.Config{
		Commands: cmdDisp,
		Queries:  qryDisp,
	})

	// --- Wire routes ---
	mux := http.NewServeMux()
	mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())

	// POST /api/items — command handler
	mux.Handle("POST /api/items", app.Command(
		"CreateItem",
		cqrshtmx.DecodeJSON(func(req createItemRequest) (command.Command, error) {
			return &createItemCmd{
				typ:   command.Type("CreateItem"),
				aggID: id.NewStreamID(),
				cmdID: id.NewCommandID(),
				Name:  req.Name,
			}, nil
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

	// POST /api/greet — typed command handler (no mapper, no type assertion)
	mux.Handle("POST /api/greet", cqrshtmx.CommandTyped[*greetCmd](
		app, "Greet",
		cqrshtmx.DecodeJSONTyped[*greetCmd](),
		cqrshtmx.WithSuccessStatus(201),
	))

	// POST /api/sum — typed query handler (returns int, no mapper)
	mux.Handle("POST /api/sum", cqrshtmx.QueryTyped[*sumQuery, int](
		app, "Sum",
		cqrshtmx.DecodeJSONQueryTyped[*sumQuery](),
		cqrshtmx.RenderJSON[int](),
	))

	// GET /api/events — SSE live updates
	mux.HandleFunc("GET /api/events", func(w http.ResponseWriter, r *http.Request) {
		stream := cqrshtmx.NewSSEStream(w, r)
		ch := broadcaster.Subscribe()
		defer broadcaster.Unsubscribe(ch)
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

	addr := ":8096"
	fmt.Printf("cqrs-htmx Basic Example\nListening on http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func indexPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><script src="/htmx.js"></script></head>
<body>
  <h1>cqrs-htmx Basic Example</h1>

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
