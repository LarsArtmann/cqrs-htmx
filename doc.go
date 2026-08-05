// Package cqrshtmx provides HTMX-aware CQRS handler integration with Casbin authorization.
//
// This library makes it easy to wire [go-cqrs-lite] command/query dispatchers into HTTP
// handlers with automatic HTMX response building, Casbin authorization, CSRF protection,
// rate limiting, SSE streaming, and error classification.
//
// Server-Timing instrumentation, CSRF core, and keyed rate limiting were
// originally implemented here but are now thin deprecated re-exports over
// [github.com/larsartmann/httputil]. Import httputil directly — these aliases
// (39 symbols across csrf_reexport.go, ratelimit_reexport.go,
// server_timing_reexport.go) will be removed in v5. See
// docs/guides/leveraging-httputil.md for the migration table.
//
// # HTTP Middleware (CORS, Compression, Body Limits, Production Server, …)
//
// For the everyday HTTP middleware a browser-facing CQRS app needs — CORS, response
// compression, body-size limits, client-IP extraction behind a proxy, a production
// server with timeouts, metrics, and dynamic ETagging — import httputil directly.
// cqrs-htmx deliberately does not re-export these (duck-typing: you import what you
// need). Every httputil middleware is a func(http.Handler) http.Handler and composes
// inside cqrshtmx.Chain. See docs/guides/leveraging-httputil.md for the full
// concern-to-middleware map and copy-paste recipes.
//
// # Quick Start
//
// Create an [App] with command/query dispatchers and a Casbin enforcer, then use
// [App.Command] and [App.Query] to build HTTP handlers that decode, authorize,
// dispatch, and respond automatically:
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    Commands:        cmdDisp,
//	    Queries:         qryDisp,
//	    Enforcer:        enforcer,
//	    UserIDExtractor: extractFunc,
//	})
//
//	mux := http.NewServeMux()
//	mux.Handle("POST /users", app.Command("CreateUser",
//	    cqrshtmx.Authorize("users", "create"),
//	    cqrshtmx.DecodeJSON(createUserMapper),
//	    cqrshtmx.NotifySuccess("User created"),
//	))
//
//	mux.Handle("GET /users", app.Query("ListUsers",
//	    cqrshtmx.Authorize("users", "read"),
//	    cqrshtmx.DecodeJSONQuery(listUsersMapper),
//	    cqrshtmx.RenderTemplResult(userListMapper),
//	))
//
// # Middleware Stack
//
// Apply middleware once to your router. The recommended order is security headers,
// panic recovery, CSRF protection, HTMX header parsing, and context enrichment:
//
//	handler := cqrshtmx.Chain(
//	    httputil.SecurityHeadersMiddleware,
//	    cqrshtmx.RecoveryMiddleware,
//	    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
//	    cqrshtmx.HTMXMiddleware,
//	    app.Middleware(),
//	)(mux)
//
// # Dispatch Middleware
//
// The [*command.Dispatcher] and [*query.Dispatcher] passed to [Config.Commands] and
// [Config.Queries] are go-cqrs-lite dispatchers. They implement Use(mw ...), so all 27
// middleware factories from [go-cqrs-lite/middleware/v4] compose with zero glue:
//
//	import "github.com/larsartmann/go-cqrs-lite/middleware/v4"
//
//	cmdDisp.Use(
//	    middleware.CommandRecovery(),
//	    middleware.CommandRetry(middleware.DefaultRetryConfig()),
//	)
//	qryDisp.Use(
//	    middleware.QueryRecovery(),
//	)
//
// Dispatch middleware runs at the CQRS layer (per-command/per-query), not the HTTP
// layer. This is strictly richer than [Config.BeforeDispatchHook] / [Config.AfterDispatchHook],
// which only see the HTTP request, not the command type. See
// docs/guides/leveraging-go-cqrs-lite.md and docs/guides/dispatch-middleware-ordering.md
// for the full middleware catalogue and ordering rules.
//
// A runnable example lives in examples/middleware-demo/.
//
// # HTMX Response Builder
//
// Build HTMX-aware responses with fluent chaining:
//
//	resp := cqrshtmx.NewResponse(w, r)
//	resp.Trigger("userCreated").
//	    PushURL("/users/1").
//	    Retarget("#user-list").
//	    NotifySuccess("User created").
//	    Apply()
//
// # Error Mapping
//
// CQRS error families automatically map to HTTP status codes. Auth errors use
// HX-Redirect for HTMX requests. 5xx detail is redacted by default:
//
//	Rejection      → 400 Bad Request
//	Conflict       → 409 Conflict
//	Corruption     → 500 Internal Server Error
//	Transient      → 503 Service Unavailable
//	Infrastructure → 503 Service Unavailable
//
// To override the family default, wrap an error with WithHTTPStatus:
//
//	err := cqrshtmx.WithHTTPStatus(sentinel, http.StatusNotFound)
//
// For unified RFC 7807 error responses across HTTP and SSE, use:
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    ErrorHandler: cqrshtmx.ProblemDetailsErrorHandler,
//	})
//
// # SSE Streaming
//
// First-class SSE support with fan-out, reconnection, heartbeat, error channel, and CQRS bridging:
//
//	broadcaster := cqrshtmx.NewBroadcaster()
//	ch := broadcaster.Subscribe()
//	broadcaster.Broadcast(sse.Event{Event: "update", Data: html})
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    AfterDispatch: broadcaster.BroadcastOnSuccess("itemUpdated", ""),
//	})
//
// BroadcastOnError closes the real-time error gap — SSE clients learn when commands fail:
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    AfterDispatch: broadcaster.BroadcastOnError("commandError"),
//	})
//
// Heartbeat prevents reverse proxies from killing idle SSE connections:
//
//	go stream.Heartbeat(stream.Context(), 15*time.Second)
//
// StructuredError (RFC 7807) provides transport-agnostic error payloads for SSE:
//
//	payload := cqrshtmx.NewStructuredError(err, r)
//	broadcaster.Broadcast(sse.Event{Event: "error", Data: payload.JSON()})
//
// # SSE Reconnection with Durable Replay
//
// JournalSSEStore provides the production sse.EventStore implementation,
// backed by the go-cqrs-lite event journal. On SSE reconnection, missed
// events are replayed via cursor-based ReadFrom:
//
//	store := cqrshtmx.NewJournalSSEStore(eventJournal, func(evt event.Event) sse.Event {
//	    return sse.Event{
//	        Event: string(evt.Type()),
//	        Data:  renderHTML(evt),
//	        ID:    sse.NewEventID(evt.ID().String()),
//	    }
//	})
//
//	lastID := stream.LastEventID()
//	sse.Replay(stream, store, lastID)
//
// # ACK Protocol (Honest UI)
//
// CommandAck provides structured command confirmation for honest UI sync-state
// transitions. Clients send X-Command-Id on requests; the server broadcasts
// an ACK back via SSE when the command completes:
//
//	broadcaster := cqrshtmx.NewBroadcaster()
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    AfterDispatch: broadcaster.BroadcastOnAck(),
//	})
//	// Client receives: {"commandId":"abc","status":"confirmed"}

// # Submodule: usermgmt
//
// The [github.com/larsartmann/cqrs-htmx/usermgmt] submodule provides passwordless
// user management with event-sourced CQRS, WebAuthn/Passkey authentication, RBAC,
// and session management. It is an independent Go module with its own go.mod.
//
// [go-cqrs-lite]: https://github.com/larsartmann/go-cqrs-lite
package cqrshtmx

// test pre-commit
