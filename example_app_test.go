package cqrshtmx_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// examplePingDecoder is shared by ExampleApp_Command_BeforeAfter and
// ExampleApp_AfterDispatch. It returns a HandlerOption that decodes a
// JSON body with an "email" field into a no-op command.
func examplePingDecoder() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ struct {
		Email string `json:"email"`
	},
	) (command.Command, error) {
		return nil, nil
	})
}

func ExampleNew() {
	disp := command.NewDispatcher()

	app, err := cqrshtmx.New(cqrshtmx.Config{
		Commands: disp,
	})
	if err != nil {
		panic(err)
	}

	_ = app
	fmt.Println("app created")
	// Output: app created
}

func ExampleApp_Command() {
	disp := command.NewDispatcher()
	app, _ := cqrshtmx.New(cqrshtmx.Config{Commands: disp})

	handler := app.Command(
		"CreateUser",
		cqrshtmx.DecodeJSON(func(_ struct {
			Email string `json:"email"`
		},
		) (command.Command, error) {
			return nil, nil
		}),
		cqrshtmx.Trigger("userCreated"),
		cqrshtmx.PushURL("/users"),
	)

	_ = handler
	fmt.Println("command handler created")
	// Output: command handler created
}

func ExampleApp_Query() {
	disp := query.NewDispatcher()
	app, _ := cqrshtmx.New(cqrshtmx.Config{Queries: disp})

	handler := app.Query(
		"GetUser",
		cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
			return nil, nil
		}),
		cqrshtmx.Render(func(_ http.ResponseWriter, _ *http.Request, _ any) error {
			return nil
		}),
	)

	_ = handler
	fmt.Println("query handler created")
	// Output: query handler created
}

func ExampleApp_HealthHandler() {
	disp := command.NewDispatcher()
	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: disp})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	app.HealthHandler().ServeHTTP(w, r)
	fmt.Println(w.Code)
	// Output: 200
}

func ExampleHTMXVersion() {
	fmt.Println(cqrshtmx.HTMXVersion())
	// Output: 2.0.10
}

func ExampleBroadcaster_BroadcastOnSuccessFunc() {
	b := cqrshtmx.NewBroadcaster()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// BroadcastOnSuccessFunc builds an AfterDispatchHook that emits a
	// dynamic SSE event derived from the request. This is the right choice
	// when the event payload depends on request data (URL, body, headers)
	// rather than a fixed template.
	hook := b.BroadcastOnSuccessFunc(itemUpdatedEventFunc)

	r := httptest.NewRequest(http.MethodPost, "/items/42", nil)
	hook(context.Background(), r, nil)

	evt := <-ch
	fmt.Println(evt.Event, evt.Data)
	// Output: itemUpdated <div>/items/42</div>
}

func ExampleApp_Query_typedRegister() {
	disp := query.NewDispatcher()
	app, _ := cqrshtmx.New(cqrshtmx.Config{Queries: disp})

	err := registerListUsersQuery(disp, []string{"alice", "bob"})
	if err != nil {
		panic(err)
	}

	users, err := query.DispatchTyped[[]string](
		context.Background(), disp, newTestListUsersQuery(),
	)
	if err != nil {
		panic(err)
	}

	_ = app
	fmt.Println(len(users))
	// Output: 2
}

func ExampleHTMXScriptTag() {
	fmt.Println(cqrshtmx.HTMXScriptTag("/static/htmx.js"))
	// Output: <script src="/static/htmx.js"></script>
}

func ExampleChain() {
	// cqrs-htmx is OpenTelemetry-agnostic. The BeforeDispatch / AfterDispatch
	// hooks are the right integration point for any tracing system —
	// OpenTelemetry, Zipkin, Datadog, etc. This example shows the
	// pattern using only the standard library: a "span" represented
	// by a struct, started before dispatch and ended after.
	disp := command.NewDispatcher()
	_ = command.RegisterTyped(disp, "Ping", func(_ context.Context, _ *testCreateUserCmd) error {
		return nil
	})

	type spanKey struct{}
	type span struct {
		name      string
		startTime time.Time
	}

	var observed *span

	app, _ := cqrshtmx.New(cqrshtmx.Config{
		Commands: disp,
		BeforeDispatch: func(_ context.Context, r *http.Request) context.Context {
			s := &span{name: r.URL.Path, startTime: time.Now()}
			return context.WithValue(context.Background(), spanKey{}, s)
		},
		AfterDispatch: func(ctx context.Context, _ *http.Request, _ error) {
			if s, ok := ctx.Value(spanKey{}).(*span); ok {
				observed = s
			}
		},
	})

	handler := app.Command("Ping", examplePingDecoder())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/ping", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w, r)

	fmt.Println(observed.name)
	fmt.Println(time.Since(observed.startTime) >= 0)
	// Output: /api/ping
	// true
}

func ExampleApp_Query_typedDispatch() {
	disp := query.NewDispatcher()
	app, _ := cqrshtmx.New(cqrshtmx.Config{Queries: disp})

	err := query.RegisterTyped(
		disp, "GetUserName",
		func(_ context.Context, _ *testGetUserNameQuery) (string, error) {
			return "alice", nil
		},
	)
	if err != nil {
		panic(err)
	}

	name, err := query.DispatchTyped[string](
		context.Background(), disp, newTestGetUserNameQuery(),
	)
	if err != nil {
		panic(err)
	}

	_ = app
	fmt.Println(name)
	// Output: alice
}

func ExampleRegisterTyped() {
	disp := command.NewDispatcher()

	err := command.RegisterTyped(
		disp, "CreateUser",
		func(_ context.Context, cmd *testCreateUserCmd) error {
			fmt.Printf("creating user: %s\n", cmd.email)
			return nil
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("typed handler registered")
	// Output: typed handler registered
}

func ExampleConfig_BeforeDispatch() {
	// Hooks let you inject cross-cutting behavior around command/query
	// dispatch. Common uses: timing, metrics, tracing spans, audit logging.
	//
	// Here we measure dispatch duration with BeforeDispatch + AfterDispatch.
	disp := command.NewDispatcher()

	var total time.Duration
	var calls int

	app, _ := cqrshtmx.New(cqrshtmx.Config{
		Commands: disp,
		BeforeDispatch: func(_ context.Context, _ *http.Request) context.Context {
			// Inject a start-time marker into the request context.
			return context.WithValue(context.Background(), startKey{}, time.Now())
		},
		AfterDispatch: func(ctx context.Context, _ *http.Request, err error) {
			if v, ok := ctx.Value(startKey{}).(time.Time); ok {
				total += time.Since(v)
				calls++
			}
			_ = err
		},
	})

	_ = command.RegisterTyped(disp, "Ping", func(_ context.Context, _ *testCreateUserCmd) error {
		return nil
	})

	handler := app.Command("Ping", examplePingDecoder())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/ping", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w, r)

	fmt.Println(calls >= 1)
	fmt.Println(total >= 0)
	// Output: true
	// true
}

type startKey struct{}

// ExampleServerTimingMiddleware demonstrates the W3C Server-Timing API: gate
// timing behind a debug query param, record sub-metrics from handlers, and
// let the middleware auto-inject the total metric at response commit time.
//
// The emitted header looks like:
//
//	Server-Timing: total;desc="Total request";dur=1, db;dur=0
func ExampleServerTimingMiddleware() {
	// Gate Server-Timing behind ?debug=1 — disabled requests pay zero overhead.
	handler := cqrshtmx.ServerTimingMiddlewareWhen(func(r *http.Request) bool {
		return r.URL.Query().Has("debug")
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Time a region that completes BEFORE the response is written.
		stop := cqrshtmx.MeasureServerTiming(r.Context(), "db")
		time.Sleep(time.Millisecond)
		stop()
		w.WriteHeader(http.StatusOK)
	}))

	// Request WITH debug → header present.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?debug=1", nil))
	hasHeader := rec.Header().Get("Server-Timing") != ""

	// Request WITHOUT debug → no header, zero overhead.
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/", nil))
	noHeader := rec2.Header().Get("Server-Timing") == ""

	fmt.Println(hasHeader, noHeader)
	// Output: true true
}
