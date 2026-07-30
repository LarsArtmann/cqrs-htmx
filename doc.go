// Package cqrshtmx provides HTMX-aware CQRS handler integration with Casbin authorization.
//
// This library makes it easy to wire [go-cqrs-lite] command/query dispatchers into HTTP
// handlers with automatic HTMX response building, Casbin authorization, CSRF protection,
// rate limiting, SSE streaming, and error classification.
//
// Server-Timing instrumentation, CSRF core, and keyed rate limiting are re-exported
// from [github.com/larsartmann/httputil] via type/var aliases, so the consumer API is
// unchanged while the implementation lives in httputil.
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
//	    cqrshtmx.SecurityHeadersMiddleware,
//	    cqrshtmx.RecoveryMiddleware,
//	    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
//	    cqrshtmx.HTMXMiddleware,
//	    app.Middleware(),
//	)(mux)
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
// For unified RFC 7807 error responses across HTTP/SSE/WS, use:
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
//	broadcaster.Broadcast(cqrshtmx.SSEEvent{Event: "update", Data: html})
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
// StructuredError (RFC 7807) provides transport-agnostic error payloads for SSE and WS:
//
//	payload := cqrshtmx.NewStructuredError(err, r)
//	broadcaster.Broadcast(cqrshtmx.SSEEvent{Event: "error", Data: payload.JSON()})
//
// # SSE Reconnection with Durable Replay
//
// JournalSSEStore provides the production SSEEventStore implementation,
// backed by the go-cqrs-lite event journal. On SSE reconnection, missed
// events are replayed via cursor-based ReadFrom:
//
//	store := cqrshtmx.NewJournalSSEStore(eventJournal, func(evt event.Event) cqrshtmx.SSEEvent {
//	    return cqrshtmx.SSEEvent{
//	        Event: string(evt.Type()),
//	        Data:  renderHTML(evt),
//	        ID:    cqrshtmx.NewSSEEventID(evt.ID().String()),
//	    }
//	})
//
//	lastID := stream.LastEventID()
//	cqrshtmx.ReplayEvents(stream, store, lastID)
//
// # ACK Protocol (Honest UI)
//
// CommandAck provides structured command confirmation for honest UI sync-state
// transitions. Clients send X-Command-Id on requests; the server broadcasts
// an ACK back via SSE or WS when the command completes:
//
//	broadcaster := cqrshtmx.NewBroadcaster()
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    AfterDispatch: broadcaster.BroadcastOnAck(),
//	})
//	// Client receives: {"commandId":"abc","status":"confirmed"}
//	// or:            {"commandId":"abc","status":"rejected","error":"..."}
//
// WebSocket variant for real-time push:
//
//	wsBroadcaster := cqrshtmx.NewWSBroadcaster()
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    AfterDispatch: wsBroadcaster.BroadcastOnAckWS(),
//	})
//
// # WebSocket
//
// Bidirectional WS support with encoder, broadcaster, dispatch bridge, and hooks:
//
//	wsB := cqrshtmx.NewWSBroadcaster()
//	wsB.Broadcast("<div hx-swap-oob='true'>Updated</div>")
//	cqrshtmx.WriteWSMessage(w, cqrshtmx.WSMessage{Body: map[string]any{"status": "ok"}})
//
// Dispatch WS messages as CQRS commands/queries (the WS counterpart to App.Command/Query):
//
//	decoder := cqrshtmx.DecodeWSJSON(func(t CreateTaskInput) (command.Command, error) {
//	    return command.New("CreateTask", t)
//	})
//	err := app.DispatchWSCommand(r, "CreateTask", decoder, rawData)
//	result, err := app.DispatchWSQuery(r, "GetTasks", queryDecoder, rawData)
//
// Bridge WS broadcasts to the dispatch lifecycle via AfterDispatch hooks:
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    AfterDispatch: wsB.BroadcastOnSuccessWSFunc(func(r *http.Request) string {
//	        return renderTasksHTML()
//	    }),
//	})
//
// # Submodule: usermgmt
//
// The [github.com/larsartmann/cqrs-htmx/usermgmt] submodule provides passwordless
// user management with event-sourced CQRS, WebAuthn/Passkey authentication, RBAC,
// and session management. It is an independent Go module with its own go.mod.
//
// [go-cqrs-lite]: https://github.com/larsartmann/go-cqrs-lite
package cqrshtmx
