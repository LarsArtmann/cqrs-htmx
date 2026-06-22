// Package main is a minimal onboarding example for cqrs-htmx.
//
// It demonstrates:
//   - Creating an App with command and query dispatchers
//   - HTMX handlers with JSON decoding via mapper functions
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

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
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
	aggID id.AggregateID
	Name  string
}

func (c *createItemCmd) Type() command.Type          { return c.typ }
func (c *createItemCmd) AggregateID() id.AggregateID { return c.aggID }

// listItemsQuery is a custom query.Query.
type listItemsQuery struct{}

func (q *listItemsQuery) Type() query.Type { return query.Type("ListItems") }

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
	it := item{ID: id.NewAggregateID().String(), Name: name}
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

	// --- Query dispatcher ---
	qryDisp := query.NewDispatcher()
	_ = qryDisp.Register(query.Type("ListItems"),
		func(_ context.Context, _ query.Query) (any, error) {
			return db.list(), nil
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
				aggID: id.NewAggregateID(),
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
  <form hx-post="/api/items" hx-swap="none">
    <input name="name" placeholder="Item name" required>
    <button>Add</button>
  </form>
  <div id="items" hx-get="/api/items" hx-trigger="load, itemCreated from:body">Loading...</div>
  <script>
    var es = new EventSource('/api/events');
    es.addEventListener('itemCreated', function() {
      htmx.ajax('GET', '/api/items', {target: '#items'});
    });
  </script>
</body>
</html>`)
}
