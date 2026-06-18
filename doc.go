// Package cqrshtmx provides HTMX-aware CQRS handler integration with Casbin authorization.
//
// This library makes it easy to wire [go-cqrs-lite] command/query dispatchers into HTTP
// handlers with automatic HTMX response building, Casbin authorization, CSRF protection,
// rate limiting, SSE streaming, and error classification.
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
// HX-Redirect for HTMX requests:
//
//	Rejection      → 400 Bad Request
//	Conflict       → 409 Conflict
//	Corruption     → 422 Unprocessable Entity
//	Transient      → 503 Service Unavailable
//	Infrastructure → 500 Internal Server Error
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
// # WebSocket
//
// Bidirectional WS support with encoder, broadcaster, and AfterDispatch hooks:
//
//	wsB := cqrshtmx.NewWSBroadcaster()
//	wsB.Broadcast("<div hx-swap-oob='true'>Updated</div>")
//	cqrshtmx.WriteWSMessage(w, cqrshtmx.WSMessage{Body: map[string]any{"status": "ok"}})
//
// # Submodule: usermgmt
//
// The [github.com/larsartmann/cqrs-htmx/usermgmt] submodule provides passwordless
// user management with event-sourced CQRS, WebAuthn/Passkey authentication, RBAC,
// and session management. It is an independent Go module with its own go.mod.
//
// [go-cqrs-lite]: https://github.com/larsartmann/go-cqrs-lite
package cqrshtmx
