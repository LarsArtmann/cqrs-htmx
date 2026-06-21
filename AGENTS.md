# Project: cqrs-htmx

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import this package into THEIR projects.
> There is no "main app."

A Go library that makes it very easy to use go-cqrs-lite with HTMX, templ, and Casbin authorization.

## Quick Reference

| Item        | Value                                                                                      |
| ----------- | ------------------------------------------------------------------------------------------ |
| Language    | Go 1.26.3                                                                                  |
| Module      | github.com/larsartmann/cqrs-htmx                                                           |
| Test        | `nix run .#test` or `GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race` |
| Build       | `nix run .#build` or `GONOSUMCHECK='github.com/larsartmann/*' go build ./...`              |
| Lint        | `nix run .#lint` or `golangci-lint run`                                                    |
| Coverage    | `nix run .#coverage`                                                                       |
| Fmt         | `nix fmt`                                                                                  |
| Flake       | `nix flake check` (formatting + devShells + apps)                                          |
| Diagrams    | `nix run .#render-diagrams` (renders all `docs/**/*.d2` → SVG; dark canvas auto-detected → theme 200) |
| ErrorFamily | `branching-flow errorfamily .` (must report 0 — no stdlib error constructors)              |
| DevShell    | `nix develop` (go, gopls, golangci-lint)                                                   |
| Coverage    | 96.4% root, 88.7% usermgmt, 95.3% catalog (500+ tests)                                     |

## Architecture

```
cqrs-htmx/
├── app.go            # App builder, Config, Command(), Query(), enrichUserID()
├── handler.go        # handleCommandDispatch(), handleQueryDispatch()
├── options.go        # HandlerOption, decoders, Render/RenderTempl, validation, authMode enum
├── response.go       # HTMX response builder (fluent API) + notification methods
├── authz.go          # Enforcer interface, Authorize, Enforce, AuthorizeMiddleware
├── context.go        # UserID/CorrelationID/RequestID types, Parse*/MustParse*, context helpers
├── errors.go         # Error → HTTP status mapping, sentinels, LoginRedirect (go-error-family)
├── htmx.go           # HTMXRequest struct, accessors, context storage, RenderPartial
├── htmx_embed.go     # Embedded HTMX v2.0.9 minified JS (go:embed), htmxVersion const
├── htmx_serve.go     # HTMXScriptHandler, HTMXScriptHandlerWith (custom JS), HTMXCDNScriptTag, HTMXScriptTag
├── notify.go         # Notification HandlerOptions + NotifyWithEvent builder
├── middleware.go      # HTTP middleware (HTMXMiddleware, ContextEnrichmentMiddleware, Chain)
├── csrf.go           # CSRF middleware, CSRFConfig, context helpers (justinas/nosurf)
├── csrf_handler.go   # CSRFProtect (per-handler CSRF HandlerOption)
├── csrf_helpers.go   # CSRFTokenHTMLMeta, CSRFTokenHXHeaders, CSRFTokenFormField
├── decoder.go        # Body reading, form/JSON decoding, MaxBodySize enforcement
├── httputil.go       # WriteJSON, ClientIP (delegates to larsartmann/httputil)
├── logging.go        # RequestLogging, RequestLoggingSlog, formatters
├── sse.go            # SSE: SSEEvent, WriteSSEEvent, SSEStream, Broadcaster (O(1) unsubscribe), SSEEventStore, ReplayEvents, LastEventIDFromRequest, BroadcastOnSuccess/BroadcastOnSuccessFunc/BroadcastOnError/BroadcastOnErrorFunc, Heartbeat, OnDisconnect
├── structured_error.go # StructuredError (RFC 7807), NewStructuredError, NewStructuredErrorWithContext, JSON()
├── ws.go             # WebSocket protocol helpers: WSMessage, ParseWSMessage, ParseWSMessageInto[T], WSOOBHTML
├── ws_encoder.go     # WriteWSMessage, WriteWSMessageInto[T] — outbound WS message encoder
├── sse_broadcaster.go # SSE Broadcaster (embeds fanOut[SSEEvent]), BroadcastOnSuccess/OnError/Func hooks
├── ws_broadcaster.go # WSBroadcaster (embeds fanOut[string]), BroadcastOnSuccessWS/OnErrorWS/Func hooks
├── fanout.go         # fanOut[T] — generic transport-agnostic fan-out hub (shared by SSE + WS)
├── ws_dispatch.go    # DispatchWSCommand/DispatchWSQuery — WS→CQRS bridge, DecodeWSJSON[T]/DecodeWSJSONQuery[T]
├── ratelimit.go      # RateLimiterMiddleware, per-key token bucket, min-heap eviction
├── security.go       # SecurityHeadersMiddleware, SecurityHeadersConfig, RecommendedCSP/HSTS
├── recovery.go       # RecoveryMiddleware (package-level), App.RecoverHandler() — panic recovery
├── usermgmt/         # User management submodule (EVENT-SOURCED CQRS, RBAC, sessions, password auth)
│   ├── go.mod        # Independent Go module
│   ├── id.go         # Branded UserID type (go-branded-id), NewUserID constructor
│   ├── authz_types.go     # Authz wrapper around Casbin (RBAC with domains), AsEnforcer adapter
│   ├── authz_policies.go  # Apply, AddGroupPolicy, RemoveGroupPolicy, AddPolicy, RemovePolicy
│   ├── authz_roles.go     # RolesForUser, ImplicitRolesForUser, ImplicitPermissionsForUser
│   ├── es_constants.go    # Event-sourced aggregate type + 7 event + 7 command type constants
│   ├── es_events.go       # 7 event payload structs (UserRegistered, CredentialAdded, etc.)
│   ├── es_commands.go     # 7 command structs (RegisterUserCmd, AddCredentialCmd, etc.)
│   ├── es_state.go        # UserState + foldUser() pure function (event → state)
│   ├── es_decide.go       # 7 pure decide functions (guards + event creation)
│   ├── es_dispatch.go     # RegisterCommands — wires commands to decider.Repository
│   ├── es_setup.go        # EventSourcedConfig, DefaultEventSourcedSetup, UserDecider
│   ├── es_readmodel.go    # UserReadModel projection + email index
│   ├── es_casbin_projection.go  # CasbinProjection — derives policies from events
│   ├── es_projection_setup.go   # StartProjections — projection.Runner orchestration
│   ├── service_core.go    # Service struct, ServiceConfig, NewService (event-sourced + WebAuthn wiring)
│   ├── service_register.go # RegisterRequest (email only), Service.Register
│   ├── service_login.go   # Service.Logout/Authenticate/Authorize (no password login)
│   ├── service_misc.go    # GetUser, UpdateRoles, ChangeEmail, ChangeDisplayName, DeleteUser, AddCredential, RemoveCredential
│   ├── credential.go      # WebAuthnCredential type (passkey credential stored as event)
│   ├── email.go          # Email value type with ParseEmail/MustParseEmail
│   ├── email_verification.go # Verification token store, SendVerificationEmail, VerifyEmail
│   ├── external_account.go # ExternalAccount value type (OAuth2 provider account linked to User)
│   ├── oauth2.go          # OAuth2/OIDC: provider config, PKCE, state store, token exchange, OIDC discovery, OAuth2StateStore interface
│   ├── service_oauth2.go  # Service.BeginOAuthLogin, FinishOAuthLogin, UnlinkExternalAccount, matchOrCreateUser (subject-first matching)
│   ├── oauth2_http.go     # HTTP handlers for OAuth2 begin/callback/unlink endpoints
│   ├── import_export.go  # ImportUsersFromJSON/CSV, ExportUsersToJSON/CSV, ImportUser.Validate
│   ├── totp.go           # TOTP MFA via pquerna/otp/totp (EnableTOTP, VerifyTOTP, DisableTOTP)
│   ├── verification_totp_http.go # HTTP handlers for verification, TOTP, import/export endpoints
│   ├── webauthn_adapter.go # Adapts domain User → webauthn.User interface
│   ├── webauthn_session.go # WebAuthnConfig + in-memory challenge store
│   ├── webauthn_service.go # BeginRegistration/FinishRegistration/BeginLogin/FinishLogin
│   ├── webauthn_http.go   # HTTP handlers for WebAuthn ceremony endpoints
│   ├── user.go       # User/Session types (immutable read model — no mutation methods)
│   ├── store.go      # SessionStore interface + InMemorySessionStore only
│   ├── events.go     # EventHandler callback + notification event structs (backward compat)
│   ├── http.go       # AuthHandler (register, logout, me, webauthn endpoints)
│   ├── middleware.go  # User context helpers, UserIDFromRequest bridge
│   ├── lockout.go    # AccountLockout (configurable max attempts + duration)
│   └── errors.go     # Sentinel errors
├── catalog/             # API documentation generation (5th Go module, opt-in)
│   ├── go.mod           # Independent Go module — depends only on go-cqrs-lite/catalog/v2
│   ├── builder.go       # Builder, New(), Command[T]/Query[T]/Event[T], Build()/BuildValid()
│   └── serve.go         # OpenAPIHandler, AsyncAPIHandler, D2Handler, GenerateEventCatalog, HealthCheckHandler
├── integration_test/ # Cross-module integration tests (3rd Go module)
└── examples/
    ├── datastar-demo/ # Standalone go-cqrs-lite + datastar SSE example (4th Go module)
    └── catalog-demo/  # Standalone catalog doc-server example (6th Go module)
```

### Module Layout

| Module           | go.mod                                              | Tests | Notes                                             |
| ---------------- | --------------------------------------------------- | ----- | ------------------------------------------------- |
| Root             | `github.com/larsartmann/cqrs-htmx`                  | Yes   | Core library                                      |
| usermgmt         | `github.com/larsartmann/cqrs-htmx/usermgmt`         | Yes   | Independent submodule                             |
| integration_test | `github.com/larsartmann/cqrs-htmx/integration_test` | Yes   | Tests cross-module bridges                        |
| datastar-demo    | `examples/datastar-demo/`                           | No    | Standalone example (main package)                 |
| catalog-demo     | `examples/catalog-demo/`                            | No    | Catalog doc-server example (main package)         |
| catalog          | `github.com/larsartmann/cqrs-htmx/catalog`          | Yes   | API doc generation (opt-in, no root/usermgmt dep) |

## Dependencies

| Dependency                  | Purpose                                                                 | Used in          |
| --------------------------- | ----------------------------------------------------------------------- | ---------------- |
| go-cqrs-lite v2.6.0         | CQRS dispatch, pagination, event sourcing (decider, memory, projection) | All modules      |
| casbin/casbin/v3            | Authorization                                                           | Root, usermgmt   |
| justinas/nosurf v1.2.0      | CSRF protection                                                         | Root             |
| go-error-family v0.4.0      | Error classification                                                    | All modules      |
| larsartmann/httputil        | ClientIP extraction                                                     | Root             |
| go-branded-id v0.3.1        | Branded types                                                           | usermgmt         |
| go-webauthn v0.17.4         | WebAuthn/Passkey passwordless authentication                            | usermgmt         |
| pquerna/otp v1.5.0          | TOTP (RFC 6238) multi-factor authentication                             | usermgmt         |
| golang.org/x/oauth2 v0.36.0 | OAuth2 authorization code flow with PKCE                                | usermgmt         |
| coreos/go-oidc/v3 v3.19.0   | OIDC provider discovery + ID token verification                         | usermgmt         |
| go-jose/go-jose/v4 v4.1.4   | JWT/JWS signing (transitive from go-oidc, used in tests)                | usermgmt (tests) |
| golang.org/x/time           | Rate limiting                                                           | Root             |
| onsi/ginkgo/v2 + gomega     | BDD test framework                                                      | All test modules |

## Key Decisions

### Architecture

- **Framework-agnostic**: Works with `net/http`, Gin, Chi, etc. — no router dependency
- **Enforcer interface**: `authz.go` defines `Enforcer{ Enforce(...any) (bool, error) }` — `*casbin.Enforcer` satisfies it automatically; enables mock/fake enforcers
- **templ duck-typing**: `TemplComponent` interface matches `templ.Component` without importing templ
- **authMode enum**: `authNone`/`authRequired`/`authAuthorized` — impossible states unrepresentable
- **Library principle**: Never enforce defaults that consumers might disagree with (no mandatory CSP/HSTS, no mandatory CSRF)
- **Root module is intentionally a single flat package**: 19 files form a cohesive "HTMX-aware CQRS HTTP integration" library. The errors↔response↔csrf cycle prevents further splitting. Sub-package extraction would harm consumer UX. See `docs/modularization/PROPOSAL.md` for full analysis
- **Root ↔ usermgmt: zero mutual imports**: Clean module boundary. Cross-module bridging happens in `integration_test/` only

### Error Handling

- **go-error-family v0.4.0**: Replaced `cockroachdb/errors` for error classification. Re-exported via `go-cqrs-lite/event/v2` (`event.NewRejection`, `event.WrapTransient`, `event.Classify`, etc.). Root + usermgmt import it transitively via `event/v2` (indirect); `catalog` imports `go-error-family` directly (no event/v2 dep). `ErrDispatchFailed` is now natively classified (`event.NewTransient`) — the old `sync.Once` + `RegisterClassification` machinery was removed
- **NO stdlib error constructors**: `errors.New`, `fmt.Errorf` (as error), and `errors.Join` are banned in non-test code. Enforced by `branching-flow errorfamily .` (must report 0). Use `event.New*/Wrap*/Wrapf/Newf` instead. Exception: `fmt.Sprintf` is fine when building a _message string_ (not an error object), e.g. `http.go`/`verification_totp_http.go` format a 400 response body
- **Family assignment rules** (maps to HTTP status via `MapError`):
  - **Rejection (400)** — caller/user input invalid: parse failures, validation (`ParseEmail`, `ImportUser.Validate`), bad config (`OAuth2ProviderConfig.Validate`, unsupported SQL dialect), missing/invalid IDs
  - **Conflict (409)** — state conflict: duplicate user/email/credential/external-account
  - **Transient (503)** — retryable system/external: DB I/O (`SQLSessionStore`/`SQLEventStore` ops), OAuth2 provider calls (discovery/exchange/userinfo), SSE/WS stream writes
  - **Corruption (422)** — stored data damage: projection payload unmarshal (`unmarshalPayload` → `event.WrapCorruption`), upcaster failures
  - **Infrastructure (500)** — non-retryable system/bug: marshal failures (`marshalPayload`), event construction, nil dependencies, command registration, `aggIDFromUser`, crypto/rand
- **Dispatch wrapping preserves family**: never force a family on a dispatch error (it may carry a domain Rejection/Conflict). Use `event.Wrapf(err, event.Classify(err), code, msg)` — wraps with the inner error's own family (root `handler.go`/`ws_dispatch.go`, usermgmt service/totp/webauthn/email_verification dispatch sites)
- **Preserve sentinel identity where tested**: `errors.Is(err, ErrValidation)` is relied upon (`service_register_test.go`, `http.go:349`) — wrap `ErrValidation` as the cause (`event.WrapRejection(ErrValidation, ...)`) in `ParseEmail`/`ImportUser.Validate`
- **Error → HTTP mapping**: `MapError` classifies errors into families (Rejection, NotFound, Conflict, etc.) → HTTP status
- **HTMX-aware errors**: All error handlers check for HTMX requests; auth errors use HX-Redirect
- **Request ID in errors**: `Config.IncludeRequestIDInErrors` auto-selects request-ID-aware error handlers
- **text/plain default**: `DefaultErrorHandlerWithRedirect` uses text/plain (no HTML escaping needed)

### CSRF

- **justinas/nosurf**: Replaced gorilla/csrf. Simpler API, no secret management needed
- **Custom header/field translation**: `translateCSRFHeaders` maps consumer-defined names to nosurf defaults
- **Per-handler CSRF**: `CSRFProtect(cfg)` caches nosurf instance per handler config
- **CSRFConfig.Validate()**: Called automatically by `CSRFMiddleware` — logs errors for SameSite=None+Secure=false and unsafe TrustedOrigins

### Type Safety

- **Strongly-typed IDs**: `UserID`, `CorrelationID`, `RequestID` are all ULID-backed branded types via `go-cqrs-lite/pkg/id`
- **usermgmt UserID**: Separate branded type via `go-branded-id` (string-backed, not ULID). Bridge: `.Get()` for cross-module conversion
- **Context key sentinels**: Empty-struct types (`userIDKey{}`, `correlationIDKey{}`, `htmxKey{}`) — collision-free

### Dispatch

- **Timeout wraps dispatch only**: `Config.Timeout` applies to `Dispatch`, not decode/auth — intentional separation
- **Lifecycle hooks**: `BeforeDispatchHook`/`AfterDispatchHook` for tracing, logging, metrics
- **Validation order**: `ValidateCommand`/`ValidateQuery` must come AFTER decoder in HandlerOption list
- **Dispatch error logging**: `handleErr` logs method, path, and error at warn level before calling error handler
- **Response.JSON error handling**: Returns HTTP 500 on json.Marshal failure instead of empty body
- **usermgmt writeJSON**: Buffers before WriteHeader so encode failures don't commit success status
- **Empty type validation (v2.3.0)**: `App.Command("")` and `App.Query("")` panic — uses `command.Type.IsZero()`/`query.Type.IsZero()` from go-cqrs-lite v2.3.0
- **Deadline propagation (v2.3.0)**: `EventOptionsFromContext` now propagates context deadline via `event.FromContext` — downstream events inherit request timeouts
- **TypedHandler (v2.3.0)**: `command.RegisterTyped[T]` eliminates manual type assertions in handlers — demonstrated in `examples/datastar-demo/`

### Performance Optimizations (2026-06-14)

- **CSRF WARN logging**: Moved from per-request to construction-time via `sync.Once`. The `isTrustedProxy()` function no longer logs — `warnEmptyTrustedProxies()` fires once at `CSRFMiddleware` creation
- **HealthHandler pre-allocation**: Response bodies are package-level `[]byte` vars (`healthyBody`, `unhealthyBody`) — zero allocations per health check
- **RequestLoggingSlog inlined**: Context ID extraction is inlined directly into the attrs slice — no intermediate `map[string]string` allocation
- **JSONLogFormatter pooled buffer**: Uses `sync.Pool` for `bytes.Buffer` + `json.Encoder` — buffer reused across requests
- **Broadcaster fan-out holds RLock**: `Broadcast()` iterates `f.subscribers` while holding the RLock, doing a non-blocking `select` send per channel. This is both allocation-free (no snapshot slice) and — critically — race-safe: a concurrent `Unsubscribe()` (which needs the write lock to `close()` a channel) cannot run during iteration, so sends can never hit a closed channel. The earlier "snapshot under RLock, release, then iterate" optimization was reverted because it allowed `Unsubscribe` to close a channel between snapshot-release and send, panicking the process (`send on closed channel`). Regression test: `sse_broadcaster_test.go` "never panics when unsubscribe races with broadcast".
- **WriteSSEEvent single-write**: Builds entire SSE frame in one `[]byte` via `append`, one `w.Write` call — replaces 3-5 `fmt.Fprintf` calls
- **splitSSELines fast-path**: Single-line data (common case) returns without allocating backing array
- **setTriggerWithDetail common path**: Uses `strings.Builder` instead of `map[string]any + json.Marshal` when no existing trigger
- **ParseWSMessageInto direct decode**: Decodes body fields directly into struct T via `json.Unmarshal` (ignoring unknown HEADERS key), then does a second lightweight decode for HEADERS only — eliminates marshal→unmarshal round-trip
- **Error responses via io.WriteString**: `DefaultErrorHandlerWithRedirect` uses `io.WriteString(w, err.Error())` instead of `[]byte(err.Error())` — avoids string→[]byte copy
- **Auth endpoints single decode**: `handleAuthEndpoint` callbacks decode body directly via `json.NewDecoder(LimitReader(...))` — eliminates double JSON decode (RawMessage → typed struct)

### Recovery

- **Package-level RecoveryMiddleware**: Uses `DefaultErrorHandler` for panics
- **App.RecoverHandler()**: Uses the App's configured error handler (renamed from RecoveryMiddleware to avoid naming collision)

### Embedded HTMX JS

- **HTMX v2.0.9 minified**: Embedded via `//go:embed htmx.min.js` in `htmx_embed.go` (~49KB). No CDN dependency — consumers serve it from their Go binary
- **HTMXScriptHandler()**: Returns `http.Handler` with correct Content-Type, ETag, Cache-Control (1 year, immutable). Supports GET/HEAD, returns 405 for others. 304 Not Modified via If-None-Match
- **HTMXScriptHandlerWith(js, version)**: Serves custom JS (e.g., htmx 4.0 beta, custom build). ETag derived from version string. `HTMXScriptHandler()` delegates to this
- **HTMXCDNScriptTag(version)**: Returns `<script src="unpkg.com/htmx.org@VERSION">`. Empty version uses embedded version. For consumers who prefer CDN over self-hosting
- **HTMXVersion()**: Returns `"2.0.9"` — useful for cache-busting query params
- **HTMXScriptTag(path)**: Returns `<script src="path"></script>` — convenience for templ/templates
- **Pluggable**: Consumers can self-host embedded version, self-host custom version, or use CDN. Not wired into any middleware or handler by default

### SSE (Server-Sent Events)

- **No external dependencies**: SSE is pure HTTP (text/event-stream), no new imports required (uses `reflect` for channel identity in Broadcaster)
- **SSEEvent**: Protocol-level type with Event, Data, ID, Retry fields. Multi-line data auto-split per SSE spec. CRLF normalized to LF
- **SSEStream**: Sets correct headers (Content-Type, Cache-Control, Connection). Flushes after each Send. Context-aware (cancelled when client disconnects). `LastEventID()` for reconnection
- **Broadcaster**: Thread-safe fan-out using buffered channels (64 capacity). O(1) Unsubscribe via `uintptr` channel identity. Non-blocking broadcast — slow consumers have events dropped. Subscribe/Unsubscribe with channel close
- **Reconnection**: `LastEventIDFromRequest()` parses `Last-Event-ID` header (returns branded `SSEEventID`). `SSEEventStore` interface for event replay. `ReplayEvents()` sends missed events to stream. `SSEEventID` branded type (`ParseSSEEventID`/`MustParseSSEEventID`/`NewSSEEventID`) prevents cross-assignment with other string IDs; rejects newlines that would corrupt the SSE wire format.
- **CQRS bridge**: `BroadcastOnSuccess(eventName, data)` / `BroadcastOnSuccessFunc(fn)` — broadcast on success. `BroadcastOnError(eventName)` / `BroadcastOnErrorFunc(fn)` — broadcast StructuredError on failure. Full dispatch feedback for SSE clients.
- **Heartbeat**: `SSEStream.Heartbeat(ctx, interval)` sends SSE comment-frame pings (": keepalive\n\n"). Prevents proxy/LB idle disconnects (Nginx 30s, Cloudflare 100s).
- **OnDisconnect**: `SSEStream.OnDisconnect(fn)` registers cleanup callbacks fired on Close().
- **StructuredError**: RFC 7807-shaped transport-agnostic error payload. `NewStructuredError(err, r)` maps via MapError + extracts request ID. Used by BroadcastOnError, BroadcastOnErrorWS, and WS dispatch bridge.
- **Not tied to App**: SSE building blocks work independently. Consumers wire Broadcaster → SSEStream in their own HTTP handlers

### WebSocket

- **No WebSocket library dependency**: Protocol helpers only (message format, OOB HTML). Consumers choose their own WebSocket library
- **WSMessage/ParseWSMessage**: Parses incoming HTMX WebSocket JSON — extracts HEADERS separately from body fields. `StringBody` helper for typed access
- **ParseWSMessageInto[T]**: Generic typed parser — deserializes body fields into a typed struct T while separating HEADERS. Compile-time safe alternative to `ParseWSMessage`
- **WSOOBHTML**: Wraps HTML with hx-swap-oob attributes for HTMX OOB swap. Uses existing `SwapStrategy` type. Passthrough when HTML already contains hx-swap-oob
- **WS Encoder**: `WriteWSMessage(w, msg)` / `WriteWSMessageInto[T](w, body, headers)` — outbound counterpart to the parsers. Round-trips perfectly through ParseWSMessage.
- **WSBroadcaster**: Thread-safe fan-out for WebSocket messages. Mirrors SSE Broadcaster API — Subscribe/Unsubscribe/Broadcast/SubscriberCount. O(1) unsubscribe via channel pointer identity.
- **WS hooks**: `BroadcastOnSuccessWS(msg)` / `BroadcastOnErrorWS()` — AfterDispatchHook factories for WS, mirroring SSE hooks.
- **WS Dispatch**: `DispatchWSCommand(r, type, decoder, data)` / `DispatchWSQuery(r, type, decoder, data)` — decode raw WS message bytes → dispatch via App. Runs before/after hooks. No auth/CSRF/response-writing (WS is authenticated at upgrade time). `DecodeWSJSON[T]` / `DecodeWSJSONQuery[T]` decoder factories. Pass the upgrade `*http.Request` for context extraction.
- **Decision documented in**: `docs/adr/0004-sse-websocket-support.md` and `docs/adr/0010-transport-parity.md`

### Pagination (go-cqrs-lite v2.3.0)

- **DecodePagination(r)**: Extracts `page`/`page_size` from query params, delegates to `query.NewPagination` for defaults and validation
- **RenderPaginatedJSON[T]()**: HandlerOption that renders `query.PaginatedResult[T]` as JSON with 200 OK. Type-safe via generic parameter
- **go-cqrs-lite PaginatedResult[T]**: `Data`, `TotalCount`, `Page`, `PageSize`, `TotalPages`, `HasNext()`, `HasPrev()`

### Domain Model (usermgmt) — Passwordless Event-Sourced CQRS (2026-06-16)

- **PASSWORDLESS**: ALL password code removed. No bcrypt, no PasswordHash, no ChangePassword. Authentication is exclusively via WebAuthn (Passkeys/FIDO2) using go-webauthn v0.17.4.
- **FULLY EVENT-SOURCED**: User aggregate uses go-cqrs-lite Decider pattern. All state changes are events persisted to an event store. `UserStore` interface REMOVED — replaced by `UserReadModel` projection.
- **7 events**: `UserRegistered`, `RolesUpdated`, `EmailChanged`, `DisplayNameChanged`, `UserDeleted` (tombstone), `CredentialAdded`, `CredentialRemoved`
- **7 commands**: `RegisterUser`, `UpdateRoles`, `ChangeEmail`, `ChangeDisplayName`, `DeleteUser`, `AddCredential`, `RemoveCredential`
- **Pure domain layer**: `foldUser()` reconstructs state from events. `decide*()` functions validate guards and emit events. No I/O in domain code.
- **Write path**: Service → CommandDispatcher → DeciderRepository.Execute (load→fold→decide→save→publish)
- **Read path**: Service queries `UserReadModel` (projection from events) — `FindByID`, `FindByEmail`
- **Casbin as projection**: `CasbinProjection` subscribes to events and derives all policies via public Authz methods only.
- **WebAuthn ceremonies**: BeginRegistration/FinishRegistration/BeginLogin/FinishLogin via go-webauthn. Challenge sessions stored in-memory (webauthnSessionStore).
- **Registration = email only**: RegisterRequest has ID + Email + DisplayName. No password field.
- **Read-your-writes consistency**: `MemoryBus` blocks publishers until handlers complete, so projections update before `Execute()` returns. Projection startup is deterministic (go-cqrs-lite v2.6.0): `StartProjections` calls `Runner.RunReplay` synchronously (catches up history), then `Runner.RunLive` in the background, and blocks on a `subscribeSignal` until the live bus subscription is registered — no `time.Sleep`-based catch-up. See `es_projection_setup.go`.
- **Sessions NOT event-sourced**: `SessionStore` interface unchanged. Ephemeral auth artifacts.
- **SQL event store delegates to upstream (RESOLVED 2026-06-19)**: `usermgmt.SQLEventStore` is now a type alias over `storage.SQLEventStore` from `go-cqrs-lite/storage/v2` (v2.6.0). The hand-rolled 413-LOC store was replaced by a 78-LOC facade. The upstream store provides a richer schema (`schema_version`, `payload_encoding`, `created_at`), OpenTelemetry tracing, `event.SeekableJournal`/`BackwardsSource`, and `event.WrapInfrastructure` error wrapping. `NewSQLEventStore` maps dialect strings → `sqlpkg.Dialect`, creates the upstream store, and applies the event schema DDL. **Breaking changes**: (1) `Close()` no longer closes the `*sql.DB` — upstream uses a borrowed handle; callers must close the DB separately. (2) `Load()` on empty aggregate returns `event.ErrAggregateNotFound` (decider's `Repository` handles this by returning Initial state). (3) MySQL is no longer supported for the event store (upstream has no MySQL dialect). `SQLSessionStore` retains MySQL support — it manages its own schema. **Migration script**: `usermgmt/migrations/0001_user_events_to_events.sql` for Postgres deployments upgrading from `< v2.5.0`. **Fuzz tests**: `FuzzSQLSessionStore_CreateFindRoundTrip` and `FuzzSQLSessionStore_DeleteByUserID` cover arbitrary userID strings (SQL injection, unicode, null bytes). **Benchmarks**: `BenchmarkSQLSessionStore_{Create,Find,FindMiss,Delete,DeleteByUserID,EvictExpired}`.
- **UserID bridge**: `usermgmt.UserID` ↔ `id.AggregateID` via `id.ParseAggregateID(userID.Get())`. Conversion at Service boundary.
- **Email uniqueness**: Pre-checked in Service.Register via read model before dispatching command.
- **DeleteUser revokes sessions**: Sessions deleted on user deletion for security.
- **Old EventHandler backward compat**: If `EventHandler` configured in `ServiceConfig`, Service bridges bus events to old callback.
- **See**: `docs/adr/0006-event-sourced-user-aggregate.md`
- **TOTP via pquerna/otp**: Replaced hand-rolled RFC 6238 with `github.com/pquerna/otp/totp` v1.5.0. `DisableTOTP` requires a valid code to prevent MFA stripping.
- **Import/export authorization**: `ImportExportAuthorizer` (type `AuthorizerFunc`) defaults to `RequireAdminRole`. Configurable via `HandlerConfig`.
- **Per-endpoint rate limiting**: `HandlerConfig.RegistrationRateLimit`, `ImportRateLimit`, `TOTPRateLimit`, `VerificationRateLimit`, `WebAuthnRateLimit`. All use the shared `RateLimitConfig` type. All disabled by default.
- **Email value type**: `type Email string` with `ParseEmail`/`MustParseEmail`. Used in `ExportUser`. Internal types stay `string` for event backward compat.
- **UserDataFormat** (renamed from ExportFormat): Used for both import and export. Constants: `UserDataFormatJSON`, `UserDataFormatCSV`.
- **See**: `docs/adr/0006-event-sourced-user-aggregate.md`

### Event Signing & Encryption (Opt-in) — 2026-06-18

- **Dependency-free seams**: `ServiceConfig` gains three opt-in fields that wire go-cqrs-lite's `signing/v2` and `encryption/v2` into usermgmt's owned infrastructure — **without** importing those modules (consumer imports them, true opt-in).
- **`StoreWrapper func(event.Store) (event.Store, error)`**: Wraps the event store _before_ journal detection + repository creation. Use `encryption.NewEncryptedStore` for transparent encryption-at-rest. Wrapper must implement `event.Journal` when inner does (NewEncryptedStore does).
- **`PublishMiddleware []event.PublishMiddleware`**: Applied via `bus.UsePublish` _before_ projections subscribe. Use `signing.SignMiddleware` / `encryption.EncryptMiddleware`. Outermost-first (registration order).
- **`HandlerMiddleware []event.Middleware`**: Applied via `bus.Use` _before_ projections subscribe. Use `signing.VerifyMiddleware` / `encryption.DecryptMiddleware`. Outermost-first.
- **Recommended pattern**: Store-level encryption (`NewEncryptedStore`) + bus-level signing (`SignMiddleware`). Store-level encryption keeps in-process projections seeing plaintext while persisted events are ciphertext.
- **Middleware ordering**: For sign+encrypt, sign plaintext _before_ encrypting. For decrypt+verify, decrypt _before_ verifying.
- **Why seams not hard deps**: Consistent with existing duck-typing (`Enforcer` ← casbin, `TemplComponent` ← templ). Consumers who don't need crypto pull zero new deps.
- **Root module**: No integration — root `App` doesn't own a bus/store. Consumers wire signing/encryption directly via go-cqrs-lite.
- **See**: `docs/adr/0011-event-signing-encryption.md`

### OAuth2/OIDC Integration — 2026-06-18

- **Dual-mode providers**: `OAuth2ProviderConfig.IssuerURL` set → OIDC discovery + ID token verification. Empty → explicit OAuth2 endpoints + UserInfo fetch. Covers Google (OIDC) and GitHub (pure OAuth2).
- **PKCE (S256) on every flow**: `oauth2.GenerateVerifier()` + `oauth2.S256ChallengeOption()` + `oauth2.VerifierOption()`. Prevents authorization code interception.
- **CSRF state tokens**: 32-byte random base64url, 10-minute TTL, one-time use. `OAuth2StateStore` interface (mirrors `SessionStore`) enables Redis adapter for multi-instance.
- **Subject-first matching**: `matchOrCreateUser` checks `FindByExternalAccount(provider, subject)` FIRST (stable ID), then email. Recognizes returning users even when their provider email changed.
- **Global provider+subject uniqueness**: `UserReadModel.externalAccounts` index enforces a provider+subject pair links to only one user. `ErrExternalAccountAlreadyLinked` (HTTP 409) on cross-user duplicate.
- **Email-based linking**: New provider with matching email links to existing user. Multiple providers (Google+GitHub) can link to same user.
- **Auto-trust provider emails**: When `email_verified: true`, user's email marked verified. Known security tradeoff documented in ADR 0014.
- **Last-auth-method guard**: `UnlinkExternalAccount` rejected if user has 0 WebAuthn credentials AND <=1 external accounts. Prevents account lockout.
- **2 events**: `ExternalAccountLinked` (Provider, Subject, Email, DisplayName), `ExternalAccountUnlinked` (Provider, Subject)
- **2 commands**: `LinkExternalAccount`, `UnlinkExternalAccount`
- **HTTP endpoints**: `GET /auth/oauth/{provider}/begin`, `GET /auth/oauth/{provider}/callback`, `POST /auth/oauth/{provider}/unlink`. Rate-limited via `HandlerConfig.OAuthRateLimit`.
- **Libraries**: `golang.org/x/oauth2` (Go team), `coreos/go-oidc/v3` (Red Hat), `go-jose/go-jose/v4` (JWT signing in tests)
- **See**: `docs/adr/0014-oauth2-oidc-integration.md`
- **Split brain fixed**: Both `NewService(ServiceConfig)` and `NewEventSourcedSetup(EventSourcedConfig)` support the same security hooks. `DefaultEventSourcedSetup()` delegates to `NewEventSourcedSetup(EventSourcedConfig{})`.

## Key Gotchas

### Module & Build

1. **GOWORK=off required**: `go.work` covers root + usermgmt + integration_test. `GOWORK=off` needed for CI/commands using per-module go.mod
2. **Module path casing**: go-cqrs-lite uses lowercase `github.com/larsartmann/go-cqrs-lite` (not `LarsArtmann`)
3. **go-cqrs-lite v2.6.0**: Per-module tags (`command/v2.6.0`, `event/v2.6.0`, etc.) published. All `go.mod` files declare `v2.6.0`. No replace directives needed
4. **Removed APIs in v2.3.0+**: `query.MustNew`, `command.MustNew`, `id.MustParse[T]` removed — use `query.New()`, `command.New()`, `id.Parse[T]()` with error check instead. Our `MustParseUserID`/`MustParseCorrelationID`/`MustParseRequestID` are local wrappers around `Parse`
5. **golangci-lint v2 format**: `.golangci.yml` uses `version: "2"`. Exclusions under `linters.exclusions.rules`, NOT `issues.exclude-rules`
6. **LSP vs CLI discrepancy**: LSP may show stale warnings after golangci.yml changes; CLI (`golangci-lint run`) is authoritative. Both report 0 issues as of go-error-family v0.4.0 adoption
7. **flake.nix uses flake-parts + treefmt**: Nix formatting via `nix fmt` (treefmt with nixfmt + gofmt). No package builds in nix due to private Go deps — use `nix run .#build`/`nix run .#test` apps instead

### Type System

6. **UserID breaking change**: `WithUserID`/`UserIDFromContext` use `id.UserID` (ULID-backed). `UserIDExtractor` still returns `string`. Use `ParseUserID`/`MustParseUserID`
7. **CorrelationID breaking change**: Branded `id.CorrelationID`. Non-ULID header values silently dropped by middleware
8. **usermgmt UserID is different type**: `usermgmt.UserID = brandid.ID[userBrand, string]` — NOT the same as root `id.UserID`. Cross-module bridge uses `.Get()` (not `.String()` which includes brand prefix)
9. **GroupPolicy.User/Domain remain string**: Casbin boundary types. Intentional.

### HTTP & Security

10. **CSRF middleware ordering**: `Chain(CSRFMiddleware, HTMXMiddleware, app.Middleware())` — CSRF first, then HTMX, then enrichment
11. **nosurf custom headers**: Must use `translateCSRFHeaders` to map consumer header/field names to nosurf defaults
12. **HX-Redirect sanitization**: `Response.Redirect()` sanitizes URLs for both HTMX and non-HTMX requests
13. **HandlerConfig.Secure uses \*bool**: `Secure` is `*bool` in usermgmt — nil defaults to true. Use `new(bool)` (false) or a local variable (true) to explicitly set. The zero-value `HandlerConfig{}` is safe
14. **Max password length 128**: Enforced in `RegisterRequest.Validate()` and `Service.ChangePassword()`. Prevents bcrypt CPU abuse
15. **CSRF TrustedProxies**: `setPlaintextHTTPOrigin` only auto-sets `Sec-Fetch-Site: same-origin` for plain HTTP requests when the remote is loopback OR in `CSRFConfig.TrustedProxies` (single IP or CIDR). Empty config logs a warning but allows it (back-compat). Set `TrustedProxies` in production to prevent CSRF bypass via origin-header stripping.
16. **Rate limiter RLock scope**: `perKeyLimiter.limiter()` must read `entry.lastUsed` **while holding RLock** — the field is written under the write lock during refresh. Reading it after `RUnlock` causes a data race under concurrent load (fixed 2026-06-14; was ~20% race-detector failure rate).

### Error Handling

15. **No stdlib error constructors**: `errors.New`/`fmt.Errorf`/`errors.Join` are banned in non-test code — `branching-flow errorfamily .` enforces 0 violations. Use `event.New*/Wrap*/Wrapf/Newf` (re-exported via `event/v2`). For dispatch errors that must preserve the inner domain family, use `event.Wrapf(err, event.Classify(err), code, msg)` — never force a family. `fmt.Sprintf` is allowed only for message strings, not error objects
16. **Middleware logs invalid IDs**: Correlation ID parse failures logged at debug level. Request ID parse failures silently generate a new ID
17. **DefaultMaxBodySize**: 10 MB when both `Config.MaxBodySize` and per-handler `WithMaxBodySize` are zero

### Testing

18. **Flaky test anti-pattern**: Never use `time.After` + `select` for timeouts — use `<-ctx.Done()` blocking
19. **Test type consolidation**: All BDD types use `bdd` prefix (e.g., `bddCreateUserCmd`)
20. **Benchmark/example lint exclusions**: `.golangci.yml` relaxes `intrange`, `noctx`, `nilnil` for benchmark/example test files only
21. **usermgmt golines**: `.golangci.yml` uses golines formatter (100-char default). Long signatures must be split

## Test Commands

### Via Nix (preferred)

```bash
nix run .#test       # All tests across all modules (with -race)
nix run .#build      # Build all modules
nix run .#lint       # Lint root + usermgmt
nix run .#coverage   # Coverage report for root + usermgmt
nix fmt              # Format all Go + Nix files
nix flake check      # Verify formatting, devShells, and apps
nix develop          # Enter dev shell (go, gopls, golangci-lint)
```

### Manual (GOWORK=off for individual modules)

Root module runs in workspace mode (uses go.work). Submodules need `GOWORK=off` so each uses its own go.mod.

The `flake.nix` provides per-module nix apps that set `GOWORK=off` automatically:

- `nix run .#test-root` — run root tests in isolation
- `nix run .#test-usermgmt` — run usermgmt tests in isolation
- `nix run .#test-integration` — run integration_test in isolation
- `nix run .#build-datastar-demo` — build the example

The multi-module `nix run .#test` and `nix run .#build` apps run all four modules in sequence.

```bash
# All tests (root)
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1

# With verbose output
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -v

# Race detector
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Coverage
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -coverprofile=coverage.out
```

### usermgmt submodule

```bash
# All tests
cd usermgmt && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Build
cd usermgmt && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go build ./...
```

### integration_test module

```bash
# All tests
cd integration_test && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race
```

### datastar-demo example

```bash
# Build only (no tests — it's a main package)
cd examples/datastar-demo && GOWORK=off go build ./...
```
