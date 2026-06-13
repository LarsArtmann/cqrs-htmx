package cqrshtmx_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

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

func ExampleNewResponse() {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/items", nil)
	r.Header.Set("HX-Request", "true")

	cqrshtmx.NewResponse(w, r).
		Trigger("itemCreated").
		PushURL("/items/42").
		Retarget("#item-list").
		Reswap(cqrshtmx.SwapOuterHTML).
		NotifySuccess("Item created").
		Apply()

	fmt.Println(w.Header().Get("HX-Trigger"))
	fmt.Println(w.Header().Get("HX-Push-Url"))
	fmt.Println(w.Header().Get("HX-Retarget"))
	fmt.Println(w.Header().Get("HX-Reswap"))
	// Output: itemCreated,showMessage
	// /items/42
	// #item-list
	// outerHTML
}

func ExampleSwapStrategy() {
	strategies := []cqrshtmx.SwapStrategy{
		cqrshtmx.SwapInnerHTML,
		cqrshtmx.SwapOuterHTML,
		cqrshtmx.SwapBeforeBegin,
		cqrshtmx.SwapAfterEnd,
		cqrshtmx.SwapNone,
	}
	for _, s := range strategies {
		fmt.Println(string(s))
	}
	// Output: innerHTML
	// outerHTML
	// beforebegin
	// afterend
	// none
}

func ExampleHTMXMiddleware() {
	mux := http.NewServeMux()

	mux.Handle("/", cqrshtmx.HTMXMiddleware(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			if cqrshtmx.IsHTMXRequest(r) {
				fmt.Println("HTMX request")
			}
			if cqrshtmx.RenderPartial(r) {
				fmt.Println("render partial")
			}
		}),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Request", "true")
	mux.ServeHTTP(w, r)
	// Output: HTMX request
	// render partial
}

func ExampleRequestLogging() {
	mux := http.NewServeMux()

	// Log every request to stdout with the default plain-text formatter.
	logged := cqrshtmx.RequestLogging(nil, func(_ string) {
		fmt.Println("logged")
	})

	mux.Handle("/", logged(
		okHandler(),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users", nil)
	mux.ServeHTTP(w, r)
	// Output: logged
}

func ExampleJSONLogFormatter() {
	mux := http.NewServeMux()

	logged := cqrshtmx.RequestLogging(cqrshtmx.JSONLogFormatter, func(line string) {
		fmt.Println(line)
	})

	mux.Handle("/", logged(
		createdHandler(),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/items", nil)
	mux.ServeHTTP(w, r)
}

func ExampleRateLimiterMiddleware() {
	mux := http.NewServeMux()

	// Allow 10 requests per minute per IP address.
	limited := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
		Limit:        10,
		Window:       time.Minute,
		KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
	})

	mux.Handle("/", limited(
		okHandler(),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	mux.ServeHTTP(w, r)
	fmt.Println(w.Code)
	// Output: 200
}

func ExampleRequireMethod() {
	disp := command.NewDispatcher()
	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: disp})

	handler := app.Command("CreateUser", cqrshtmx.RequireMethod(http.MethodPost))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users", nil)
	handler.ServeHTTP(w, r)
	fmt.Println(w.Code)
	// Output: 405
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

func ExampleResponse_JSON() {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	cqrshtmx.NewResponse(w, r).JSON(map[string]string{"s": "ok"})
	fmt.Println(w.Header().Get("Content-Type"))
	// Output: application/json; charset=utf-8
}

func ExampleRecommendedHSTS() {
	handler := cqrshtmx.SecurityHeadersMiddlewareWithConfig(cqrshtmx.SecurityHeadersConfig{
		StrictTransportSecurity: cqrshtmx.RecommendedHSTS,
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)
	fmt.Println(w.Header().Get("Strict-Transport-Security") != "")
	// Output: true
}

func ExampleWriteSSEEvent() {
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "text/event-stream")

	err := cqrshtmx.WriteSSEEvent(w, cqrshtmx.SSEEvent{
		Event: "todoCreated",
		Data:  "<li>Buy milk</li>",
		ID:    "evt-1",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(w.Body.String())
	// Output: event: todoCreated
	// data: <li>Buy milk</li>
	// id: evt-1
	//
}

func ExampleSSEStream() {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := cqrshtmx.NewSSEStream(w, r)
	defer stream.Close()

	_ = stream.Send(cqrshtmx.SSEEvent{Event: "update", Data: "<div>new</div>"})
	_ = stream.SendHTML("update", "<div>newer</div>")

	fmt.Println(w.Header().Get("Content-Type"))
	// Output: text/event-stream
}

func ExampleBroadcaster() {
	b := cqrshtmx.NewBroadcaster()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Broadcast(cqrshtmx.SSEEvent{Event: "itemCreated", Data: "<li>item</li>"})

	evt := <-ch
	fmt.Println(evt.Event, evt.Data)
	// Output: itemCreated <li>item</li>
}

func ExampleParseWSMessage() {
	data := []byte(`{"message":"hello","room":"general","HEADERS":{"HX-Request":"true"}}`)

	msg, err := cqrshtmx.ParseWSMessage(data)
	if err != nil {
		panic(err)
	}

	fmt.Println(msg.StringBody("message"), msg.StringBody("room"), msg.Headers["HX-Request"])
	// Output: hello general true
}

func ExampleWSOOBHTML() {
	html := cqrshtmx.WSOOBHTML("todos", "<ul><li>Buy milk</li></ul>")
	fmt.Println(html)
	// Output: <div id="todos" hx-swap-oob="true"><ul><li>Buy milk</li></ul></div>
}

func ExampleWSOOBHTML_swapStrategy() {
	html := cqrshtmx.WSOOBHTML("notifications", "New message", cqrshtmx.SwapBeforeEnd)
	fmt.Println(html)
	// Output: <div id="notifications" hx-swap-oob="beforeend">New message</div>
}

func ExampleHTMXScriptHandler() {
	mux := http.NewServeMux()
	mux.Handle("/static/htmx.js", cqrshtmx.HTMXScriptHandler())
	fmt.Println("mounted")
	// Output: mounted
}

func ExampleHTMXScriptTag() {
	fmt.Println(cqrshtmx.HTMXScriptTag("/static/htmx.js"))
	// Output: <script src="/static/htmx.js"></script>
}

func ExampleHTMXVersion() {
	fmt.Println(cqrshtmx.HTMXVersion())
	// Output: 2.0.9
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

func ExampleApp_Query_typedRegister() {
	disp := query.NewDispatcher()
	app, _ := cqrshtmx.New(cqrshtmx.Config{Queries: disp})

	err := query.RegisterTyped(
		disp, "ListUsers",
		func(_ context.Context, _ *testListUsersQuery) ([]string, error) {
			return []string{"alice", "bob"}, nil
		},
	)
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

func ExampleBroadcaster_BroadcastOnSuccessFunc() {
	b := cqrshtmx.NewBroadcaster()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// BroadcastOnSuccessFunc builds an AfterDispatchHook that emits a
	// dynamic SSE event derived from the request. This is the right choice
	// when the event payload depends on request data (URL, body, headers)
	// rather than a fixed template.
	hook := b.BroadcastOnSuccessFunc(func(r *http.Request) cqrshtmx.SSEEvent {
		return cqrshtmx.SSEEvent{
			Event: "itemUpdated",
			Data:  "<div>" + r.URL.Path + "</div>",
		}
	})

	r := httptest.NewRequest(http.MethodPost, "/items/42", nil)
	hook(context.Background(), r, nil)

	evt := <-ch
	fmt.Println(evt.Event, evt.Data)
	// Output: itemUpdated <div>/items/42</div>
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

	handler := app.Command("Ping", cqrshtmx.DecodeJSON(func(_ struct {
		Email string `json:"email"`
	},
	) (command.Command, error) {
		return nil, nil
	}))

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

	handler := app.Command(
		"Ping",
		cqrshtmx.DecodeJSON(func(_ struct {
			Email string `json:"email"`
		},
		) (command.Command, error) {
			return nil, nil
		}),
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/ping", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w, r)

	fmt.Println(observed.name)
	fmt.Println(time.Since(observed.startTime) >= 0)
	// Output: /api/ping
	// true
}
