# Project: cqrs-htmx

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import this package into THEIR projects.
> There is no "main app."

A Go library that makes it very easy to use go-cqrs-lite with HTMX, templ, and Casbin authorization.

## Quick Reference

| Item        | Value                                                                                                 |
| ----------- | ----------------------------------------------------------------------------------------------------- |
| Language    | Go 1.26.3                                                                                             |
| Module      | github.com/larsartmann/cqrs-htmx/v3                                                                   |
| Test        | `nix run .#test` or `GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race`            |
| Build       | `nix run .#build` or `GONOSUMCHECK='github.com/larsartmann/*' go build ./...`                         |
| Lint        | `nix run .#lint` or `golangci-lint run`                                                               |
| Coverage    | `nix run .#coverage`                                                                                  |
| Fmt         | `nix fmt`                                                                                             |
| Flake       | `nix flake check` (formatting + devShells + apps)                                                     |
| Diagrams    | `nix run .#render-diagrams` (renders all `docs/**/*.d2` → SVG; dark canvas auto-detected → theme 200) |
| ErrorFamily | `branching-flow errorfamily .` (must report 0 — no stdlib error constructors)                         |
| DevShell    | `nix develop` (go, gopls, golangci-lint)                                                              |
| Coverage    | 95.4% root, 80.1% usermgmt (~747 usermgmt + ~135 root tests)                                          |

## Architecture

```
cqrs-htmx/
├── app.go            # App builder, Config, Command(), Query(), enrichUserID()
├── doc.go            # Package documentation
├── handler.go        # handleCommandDispatch(), handleQueryDispatch()
├── options_types.go  # handlerConfig struct, authMode enum (authNone/authRequired/authAuthorized)
├── options_decode.go # DecodeJSON/DecodeForm/DecodeQuery HandlerOption factories
├── options_render.go # Render/RenderTempl/RenderJSON/RenderPaginatedJSON HandlerOptions
├── options_htmx.go   # HTMX-specific HandlerOptions (Redirect/PushURL/Retarget)
├── options_json.go   # JSON response HandlerOption helpers
├── options_validate.go # ValidateCommand/ValidateQuery HandlerOptions
├── response.go       # HTMX response builder (fluent API) + notification methods
├── responsewriter.go # delegatingWriter — embeds http.ResponseWriter, delegates Flush/Hijack/Push/Unwrap
├── authz.go          # Enforcer interface, Authorize, Enforce, AuthorizeMiddleware
├── context.go        # UserID/CorrelationID/RequestID types, Parse*/MustParse*, context helpers
├── errors.go         # Error → HTTP status mapping, sentinels, LoginRedirect (go-error-family), SafeDetail, ProblemDetailsErrorHandler
├── errors_status.go  # HTTPStatusCarrier interface + WithHTTPStatus wrapper (ADR-0034)
├── htmx.go           # HTMXRequest struct, accessors, context storage, RenderPartial
├── htmx_embed.go     # Embedded HTMX v2.0.9 minified JS (go:embed), htmxVersion const
├── htmx_serve.go     # HTMXScriptHandler, HTMXScriptHandlerWith (custom JS), HTMXCDNScriptTag, HTMXScriptTag
├── notify.go         # Notification HandlerOptions + NotifyWithEvent builder
├── middleware.go      # HTTP middleware (HTMXMiddleware, ContextEnrichmentMiddleware, Chain)
├── csrf_config.go     # CSRFConfig (cookie/header/field names, TrustedProxies, SameSite, Secure)
├── csrf_context.go    # CSRF token context helpers (set/get token)
├── csrf_middleware.go # CSRFMiddleware (justinas/nosurf integration, origin checks)
├── csrf_handler.go   # CSRFProtect (per-handler CSRF HandlerOption)
├── csrf_helpers.go   # CSRFTokenHTMLMeta, CSRFTokenHXHeaders, CSRFTokenFormField
├── decoder.go        # Body reading, form/JSON decoding (go-playground/form/v4), MaxBodySize enforcement
├── httputil.go       # WriteJSON, ClientIP (delegates to larsartmann/httputil)
├── logging.go        # RequestLogging, RequestLoggingSlog, formatters
├── sse_event.go      # SSEEvent struct, WriteSSEEvent, SSEEventID branded type + ParseSSEEventID
├── sse_stream.go     # SSEStream (Send/SendHTML/Heartbeat/OnDisconnect/Close), NewSSEStream
├── sse_store.go      # SSEEventStore interface, ReplayEvents, LastEventIDFromRequest
├── event_store_sse.go # JournalSSEStore — PRODUCTION SSEEventStore backed by event.SeekableJournal
├── sse_broadcaster.go # SSE Broadcaster (embeds fanOut[SSEEvent]), BroadcastOnSuccess/OnError/Func hooks
├── ack.go            # CommandAck + BroadcastOnAck — ACK protocol for honest UI sync-state
├── idempotency.go   # Thin backward-compatible aliases over go-cqrs-lite/idempotency/v3 (Store, MemoryStore, ErrDuplicate)
├── structured_error.go # StructuredError (RFC 7807), NewStructuredError, NewStructuredErrorWithContext, JSON()
├── ws.go             # WebSocket protocol helpers: WSMessage, ParseWSMessage, ParseWSMessageInto[T], WSOOBHTML
├── ws_encoder.go     # WriteWSMessage, WriteWSMessageInto[T] — outbound WS message encoder
├── sse_broadcaster.go # SSE Broadcaster (embeds fanOut[SSEEvent]), BroadcastOnSuccess/OnError/Func hooks
├── ws_broadcaster.go # WSBroadcaster (embeds fanOut[string]), BroadcastOnSuccessWS/OnErrorWS/Func hooks
├── fanout.go         # fanOut[T] — generic transport-agnostic fan-out hub (shared by SSE + WS)
├── ws_dispatch.go    # DispatchWSCommand/DispatchWSQuery — WS→CQRS bridge, DecodeWSJSON[T]/DecodeWSJSONQuery[T]
├── ratelimit_config.go   # RateLimitConfig (Requests, Window, KeyExtractor, MaxKeys, TTL)
├── ratelimit_middleware.go # RateLimiterMiddleware, per-key token bucket, min-heap LRU eviction
├── security.go       # SecurityHeadersMiddleware, SecurityHeadersConfig, RecommendedCSP/HSTS
├── recovery.go       # RecoveryMiddleware (package-level), App.RecoverHandler() — panic recovery
├── server_timing.go  # ServerTiming collector + ServerTimingMiddleware/When — W3C Server-Timing API (opt-in, debug-gated)
├── usermgmt/         # User management submodule (EVENT-SOURCED CQRS, RBAC, sessions, password auth)
│   ├── go.mod        # Independent Go module
│   ├── id.go         # UserID (alias of id.UserID), ActorID struct, TenantID, BotID
│   ├── authz_types.go     # Authz wrapper around Casbin (RBAC with domains), AsEnforcer adapter
│   ├── authz_policies.go  # Apply, AddGroupPolicy, RemoveGroupPolicy, AddPolicy, RemovePolicy
│   ├── authz_roles.go     # RolesForUser, RolesForActor, ImplicitRolesForActor, role hierarchy seed (g2)
│   ├── es_constants.go    # Aggregate types (User, Membership, Tenant, Bot) + 21 event + 20 command constants + schema v2
│   ├── es_events.go       # 12 event payload structs (UserRegistered, CredentialAdded, EmailVerified, TOTPEnabled, ExternalAccountLinked, etc.)
│   ├── es_commands.go     # 11 command structs (RegisterUserCmd, AddCredentialCmd, VerifyEmailCmd, EnableTOTPCmd, LinkExternalAccountCmd, etc.)
│   ├── es_state.go        # UserState + foldUser() pure function (event → state)
│   ├── es_decide.go       # 7 pure decide functions (guards + event creation)
│   ├── es_membership_events.go    # 3 membership event payloads (MemberAdded, MemberRolesChanged, MemberRemoved)
│   ├── es_membership_commands.go  # 3 membership commands (AddMemberCmd, UpdateMemberRolesCmd, RemoveMemberCmd)
│   ├── es_membership_state.go     # MembershipState + foldMembership() pure function
│   ├── es_membership_decide.go    # MembershipDecider() + 3 decide functions
│   ├── es_membership_dispatch.go  # RegisterMembershipCommands — wires membership commands
│   ├── es_membership_readmodel.go # MembershipReadModel projection + FindByActor query
│   ├── es_tenant_events.go    # TenantCreated/Suspended/Reactivated/Deleted payloads
│   ├── es_tenant_commands.go  # CreateTenant/Suspend/Reactivate/DeleteTenant commands
│   ├── es_tenant_state.go     # TenantState + foldTenant
│   ├── es_tenant_decide.go    # TenantDecider + decide functions
│   ├── es_tenant_dispatch.go  # RegisterTenantCommands
│   ├── es_tenant_readmodel.go # TenantReadModel projection + FindByID/FindByName
│   ├── es_materialize_adapter.go # MaterializeProjection[V,K] — wraps stack.Materialize as event.Projection (ADR-0022)
│   ├── es_bot_events.go       # BotRegistered/Deleted payloads
│   ├── es_bot_commands.go     # RegisterBot/DeleteBot commands
│   ├── es_bot_state.go        # BotState + foldBot
│   ├── es_bot_decide.go       # BotDecider + decide functions
│   ├── es_bot_dispatch.go     # RegisterBotCommands
│   ├── es_bot_readmodel.go    # BotReadModel projection + FindByTokenHash
│   ├── es_migration.go        # MigrateRolesToMemberships — opt-in legacy role migration
│   ├── crypto.go              # GenerateToken, HashToken (HMAC-SHA256), VerifyToken
│   ├── api_token_middleware.go # NewAPITokenMiddleware, RequireBot
│   ├── service_tenant.go      # Service.CreateTenant/Suspend/Reactivate/Delete/GetTenant
│   ├── service_bot.go         # Service.RegisterBot/DeleteBot/GetBot/ResolveBotByToken
│   ├── service_impersonation.go # Service.BeginImpersonation/EndImpersonation with guards
│   ├── es_dispatch.go     # RegisterCommands — wires commands to decider.Repository
│   ├── es_setup.go        # EventSourcedConfig, DefaultEventSourcedSetup, UserDecider
│   ├── es_readmodel.go    # UserReadModel projection + email index
│   ├── es_casbin_projection.go  # CasbinProjection — derives policies from events
│   ├── es_projection_setup.go   # StartProjections — manual journal replay + bus.SubscribeAll
│   ├── sql_readmodel.go         # SQLUserReadModel (SQLite/Postgres) — UserView DTO + Handle/syncToSQL + FindByIDSQL/FindByEmailSQL/CountSQL
│   ├── sql_readmodel_extra.go   # SQLMembershipReadModel, SQLTenantReadModel, SQLBotReadModel — same pattern for 3 aggregates
│   ├── sqlite_setup.go          # NewSQLiteEventSourcedSetup — one-call SQLite stack preset (bundle + repos + SQL read models + projections)
│   ├── postgres_setup.go        # NewPostgresEventSourcedSetup — one-call Postgres stack preset (supports multi-DB split)
│   ├── service_core.go    # Service struct, ServiceConfig, NewService (event-sourced + WebAuthn wiring)
│   ├── service_register.go # RegisterRequest (email only), Service.Register
│   ├── service_login.go   # Service.Logout/Authenticate/Authorize (no password login)
│   ├── service_misc.go    # GetUser, UpdateRoles, ChangeEmail, ChangeDisplayName, DeleteUser, AddCredential, RemoveCredential
│   ├── id.go          # Branded types: UserID, TenantID, BotID + ActorID struct (kind-discriminated)
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
│   ├── es_upcaster.go    # UpcasterRegistry — schema version transforms (v1→v2 migration)
│   ├── credential_http.go # Credential management HTTP endpoints (list/delete)
│   ├── audit_log.go       # AuditLog — append-only audit event recorder
│   ├── eviction.go        # startPeriodicEviction — shared TTL sweep goroutine for ephemeral stores
│   ├── random.go          # randomBase64URLString — shared CSPRNG token generation (32 bytes)
├── adminui/             # Ready-made Admin Dashboard UI (7th Go module, templ + HTMX)
│   ├── go.mod          # Independent Go module — depends on root + usermgmt + a-h/templ
│   ├── config.go       # Config, Mode (SuperAdmin/TenantAdmin), defaults
│   ├── authz.go        # defaultAuthorizer, RequireAnyRole, RequireAuthenticated
│   ├── handler.go      # Handler, New(), Mount()/Handler(), routing, guard (auth+authz)
│   ├── render.go       # renderPage/renderPartial, triggerToast (HX-Trigger), redirect (HX-Redirect)
│   ├── assets.go       # go:embed admin-tw.css/admin.js/sync-worker.js; reuses root HTMXScriptHandler for htmx.js
│   ├── models.go       # view-model structs (navItem, pageData, per-section data)
│   ├── icons.go        # inline SVG icon set (stroke=currentColor)
│   ├── *.templ         # templ components (layout, dashboard, users, tenants, members, audit, components)
│   ├── *_templ.go      # GENERATED templ output — committed so consumers run no codegen
│   ├── handler_*.go    # per-section HTTP handlers (dashboard/users/tenants/members/audit)
│   └── assets/         # admin-tw.css (compiled Tailwind + sync-state CSS) + admin.js (toasts, mobile nav, SSE ACK, offline queue) + sync-worker.js (SharedWorker, ADR-0029)
├── integration_test/ # Cross-module integration tests (3rd Go module)
└── examples/
    ├── basic/         # Minimal cqrs-htmx consumer example (register/list items)
    ├── datastar-demo/ # Standalone go-cqrs-lite + datastar SSE example (4th Go module)
    ├── catalog-demo/  # Standalone catalog doc-server example (6th Go module, uses go-cqrs-lite/catalog)
    └── admin-demo/    # Runnable admin panel showcase (8th Go module) — go run ., open :8097
```

### Module Layout

| Module           | go.mod                                              | Tests | Notes                                                     |
| ---------------- | --------------------------------------------------- | ----- | --------------------------------------------------------- |
| Root             | `github.com/larsartmann/cqrs-htmx/v3`               | Yes   | Core library                                              |
| usermgmt         | `github.com/larsartmann/cqrs-htmx/usermgmt/v3`      | Yes   | Independent submodule; imports root for RateLimiter       |
| adminui          | `github.com/larsartmann/cqrs-htmx/adminui/v3`       | Yes   | Admin Dashboard UI (templ+HTMX), depends on root+usermgmt |
| integration_test | `github.com/larsartmann/cqrs-htmx/integration_test` | Yes   | Tests cross-module bridges                                |
| datastar-demo    | `examples/datastar-demo/`                           | No    | Standalone example (main package)                         |
| catalog-demo     | `examples/catalog-demo/`                            | No    | Catalog doc-server example (main package)                 |
| admin-demo       | `examples/admin-demo/`                              | No    | Runnable admin panel showcase (main package)              |
| basic            | `examples/basic/`                                   | No    | Minimal cqrs-htmx consumer example                        |

## Dependencies

| Dependency                   | Purpose                                                                                                             | Used in          |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------- | ---------------- |
| go-cqrs-lite v3.4.0          | CQRS dispatch, pagination, event sourcing (decider, storage/memory, watermill bus, SQL view stores, typed metadata) | All modules      |
| casbin/casbin/v3             | Authorization                                                                                                       | Root, usermgmt   |
| justinas/nosurf v1.2.0       | CSRF protection                                                                                                     | Root             |
| go-error-family v0.5.1       | Error classification                                                                                                | All modules      |
| larsartmann/httputil         | ClientIP extraction                                                                                                 | Root             |
| go-branded-id v0.3.1         | Branded types (ActorID, UserID, TenantID, BotID, SSEEventID)                                                        | Root, usermgmt   |
| go-webauthn v0.17.4          | WebAuthn/Passkey passwordless authentication                                                                        | usermgmt         |
| pquerna/otp v1.5.0           | TOTP (RFC 6238) multi-factor authentication                                                                         | usermgmt         |
| golang.org/x/oauth2 v0.36.0  | OAuth2 authorization code flow with PKCE                                                                            | usermgmt         |
| coreos/go-oidc/v3 v3.19.0    | OIDC provider discovery + ID token verification                                                                     | usermgmt         |
| go-jose/go-jose/v4 v4.1.4    | JWT/JWS signing (transitive from go-oidc, used in tests)                                                            | usermgmt (tests) |
| golang.org/x/time            | Rate limiting                                                                                                       | Root             |
| go-playground/form/v4 v4.3.0 | Form decoding (url.Values → struct, zero transitive deps, json tag mode for backward compat)                        | Root             |
| a-h/templ v0.3.1020          | Type-safe HTML templating (admin UI components)                                                                     | adminui          |
| onsi/ginkgo/v2 + gomega      | BDD test framework                                                                                                  | All test modules |

## Key Decisions

### Architecture

- **Framework-agnostic**: Works with `net/http`, Gin, Chi, etc. — no router dependency
- **Enforcer interface**: `authz.go` defines `Enforcer{ Enforce(...any) (bool, error) }` — `*casbin.Enforcer` satisfies it automatically; enables mock/fake enforcers
- **templ duck-typing**: `TemplComponent` interface matches `templ.Component` without importing templ
- **authMode enum**: `authNone`/`authRequired`/`authAuthorized` — impossible states unrepresentable
- **Library principle**: Never enforce defaults that consumers might disagree with (no mandatory CSP/HSTS, no mandatory CSRF)
- **Root module is intentionally a single flat package**: 40 files form a cohesive "HTMX-aware CQRS HTTP integration" library. The errors↔response↔csrf cycle prevents further splitting. Sub-package extraction would harm consumer UX. See `docs/modularization/PROPOSAL.md` for full analysis
- **Root → usermgmt: zero imports** (clean boundary). **usermgmt → root: YES** (RateLimiter). usermgmt imports `cqrs-htmx/v3` for `RateLimiter` (rate limiting unification). This is a one-way dependency — root never imports usermgmt. Cross-module bridging tests happen in `integration_test/`

### Error Handling

- **go-error-family v0.5.1**: Replaced `cockroachdb/errors` for error classification. Re-exported via `go-cqrs-lite/event/v3` (`event.NewRejection`, `event.WrapTransient`, `event.Classify`, etc.). Root + usermgmt import it transitively via `event/v3` (indirect). `ErrDispatchFailed` is now natively classified (`event.NewTransient`) — the old `sync.Once` + `RegisterClassification` machinery was removed
- **NO stdlib error constructors**: `errors.New`, `fmt.Errorf` (as error), and `errors.Join` are banned in non-test code. Enforced by `branching-flow errorfamily .` (must report 0). Use `event.New*/Wrap*/Wrapf/Newf` instead. Exception: `fmt.Sprintf` is fine when building a _message string_ (not an error object), e.g. `http.go`/`verification_totp_http.go` format a 400 response body
- **Family assignment rules** (maps to HTTP status via `MapError`):
  - **Rejection (400)** — caller/user input invalid: parse failures, validation (`ParseEmail`, `ImportUser.Validate`), bad config (`OAuth2ProviderConfig.Validate`, unsupported SQL dialect), missing/invalid IDs
  - **Conflict (409)** — state conflict: duplicate user/email/credential/external-account
  - **Transient (503)** — retryable system/external: DB I/O (`SQLSessionStore`/`SQLEventStore` ops), OAuth2 provider calls (discovery/exchange/userinfo), SSE/WS stream writes
  - **Corruption (500)** — stored data damage: projection payload unmarshal (`unmarshalPayload` → `event.WrapCorruption`), upcaster failures
  - **Infrastructure (503)** — non-retryable system/bug: marshal failures (`marshalPayload`), event construction, nil dependencies, command registration, `aggIDFromUser`, crypto/rand
- **Dispatch wrapping preserves family**: never force a family on a dispatch error (it may carry a domain Rejection/Conflict). Use `event.Wrapf(err, event.Classify(err), code, msg)` — wraps with the inner error's own family (root `handler.go`/`ws_dispatch.go`, usermgmt service/totp/webauthn/email_verification dispatch sites)
- **Preserve sentinel identity where tested**: `errors.Is(err, ErrValidation)` is relied upon (`service_register_test.go`, `http.go:349`) — wrap `ErrValidation` as the cause (`event.WrapRejection(ErrValidation, ...)`) in `ParseEmail`/`ImportUser.Validate`
- **Error → HTTP mapping**: `MapError` resolves status in 3 layers: (1) `HTTPStatusCarrier` interface (via `WithHTTPStatus(err, status)`) — highest authority, (2) explicit sentinel overrides (auth/HTTP-semantic/panic), (3) `Family.HTTPStatus()` from go-error-family. See ADR-0034
- **HTTPStatusCarrier**: Rejection-family errors can pin a non-default status (401/403/404/429) via `cqrshtmx.WithHTTPStatus(event.NewRejection(...), http.StatusNotFound)`. The wrapper preserves the cause's family + sentinel identity (errors.Is traverses). usermgmt sentinels use this pattern; `errorStatus()` is a one-liner delegating to `MapError`
- **5xx detail redaction**: `SafeDetail(err, status, includeInternal)` replaces 5xx error text with the family's public-safe default message. 4xx detail (raw error) is preserved. `Config.IncludeInternalDetails` opts back in for dev. `StructuredError.Detail` is also redacted for SSE/WS
- **StructuredError metadata**: Exposes `Message`, `Why`, `Fix` (RFC 7807 extensions from `Family.DefaultMessage/Why/Fix`). Same JSON shape across HTTP/SSE/WS
- **ProblemDetailsErrorHandler**: Emits `StructuredError` as `application/problem+json` — unified RFC 7807 shape across all transports. Opt in via `Config.ErrorHandler`
- **Exported auth codes**: `CodeUnauthorized`/`CodeForbidden` — compile-time-safe shared between root and usermgmt
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

### Server Timing API (Server-Timing header) — 2026-06-29

- **W3C Server-Timing** (`server_timing.go`): opt-in middleware that emits the `Server-Timing` response header (e.g. `total;desc="Total request";dur=12, db;dur=8`). Spec: https://w3c.github.io/server-timing/
- **The header-before-body constraint**: like all HTTP headers, `Server-Timing` must be set before `WriteHeader`/`Write`. The library's `AfterDispatchHook` cannot set headers (no `ResponseWriter`, response already committed). So `ServerTimingMiddleware` wraps the `ResponseWriter` (`serverTimingWriter`) and injects the header at the **first** `WriteHeader`/`Write` — the same wrapping pattern `StatusRecorder` uses (`logging.go:129`)
- **`ServerTiming` collector**: thread-safe, stored in context via `WithServerTiming`/`ServerTimingFromContext` (mirrors `WithRequestID`). Uses **nil-receiver pattern**: disabled = nil in context = every method is a natural no-op. No `enabled` field, no branching. Helpers `RecordServerTiming(ctx,...)` and `MeasureServerTiming(ctx, name)` delegate directly to the (possibly nil) collector
- **Recording**: `defer st.Measure("db")()` OR explicit `stop := MeasureServerTiming(ctx,"db"); ...; stop()`. **TTFB gotcha**: a metric's region must END before the response is committed (Write/WriteHeader) to appear in the header — `defer Measure()()` records at function return, which is AFTER the write for non-streaming handlers, so it misses the header. The `total` metric is captured at flush time (TTFB) by design
- **Three entry points**: (1) `ServerTimingMiddleware()` always-on; (2) `ServerTimingMiddlewareWhen(pred)` — the "debug mode" gate (e.g. `?debug=1`, admin role, localhost); (3) **`Config.ServerTiming` predicate** — 1-line integration into `App.Command()`/`App.Query()` handlers, no separate middleware needed for App-managed routes
- **Zero-overhead when off** (verified by benchmark): disabled = 4.3 ns/op, 0 allocs (just a `context.Value` lookup + nil check). When `pred` returns false (or `Config.ServerTiming` is nil), no `ResponseWriter` wrapping and no collector allocation — the nil-receiver methods are no-ops. Enabled Measure = 141 ns/op, 1 alloc (closure); HeaderValue (5 metrics) = 311 ns/op, 7 allocs
- **Interface preservation is critical**: the wrapper delegates `Flusher` (SSE does `w.(flusher)` at `sse_stream.go:63`), `Hijacker` (WebSocket upgrades), and `Pusher` (HTTP/2). Also implements `Unwrap()` for `http.ResponseController`. Losing any of these silently breaks SSE/WS
- **Spec compliance**: metric names are RFC 7230 tokens (invalid chars → `_`, empty dropped); descriptions are RFC 7230 quoted-strings (escape `"` and `\`, so commas/semicolons are safe); `dur` rendered in ms with shortest round-trip float (sub-ms preserved); zero `dur` omits the param
- **OPT-IN, never auto-applied** (library principle)

### Lifecycle & Shutdown (usermgmt)

- **`Service.Stop()`**: Stops all background eviction goroutines (WebAuthn sessions, verification tokens, TOTP, OAuth2 state). Does NOT close bus/store. Idempotent.
- **`Service.Close()`**: Full shutdown — calls `Stop()` then closes the event bus and event store (if they implement `io.Closer`). Idempotent. Returns first error encountered.
- **`Service.GracefulClose(ctx)`**: Same as `Close()` but also returns `ctx.Err()` if the context is cancelled. Use in servers with graceful shutdown signals.
- **`EventSourcedSetup.Close()`**: Closes the setup's bus and store (if they implement `io.Closer`). Idempotent.
- **`closeBus()` helper** (`es_setup.go`): Internal — used at error-return sites during setup to clean up partially-wired infrastructure.
- **In-memory defaults are NOT io.Closer**: `memory.NewMemoryStore()` and `watermill.NewEventBus()` may or may not implement `io.Closer` — `Close()` handles both cases via type assertion.

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
- **Production SSEEventStore**: `JournalSSEStore` (`event_store_sse.go`) wraps `event.Journal`/`event.SeekableJournal` for durable replay. Uses `ReadFrom(afterEventID, limit)` for efficient cursor-based position replay. Falls back to `ReadAll` + in-memory filter when `SeekableJournal` is unavailable. Consumer provides `EventToSSEMapper` to render event payloads. `WithMaxReplay(n)` limits initial sync (default: 1000). See `docs/adr/0023-command-sync.md`.
- **ACK Protocol**: `CommandAck` (`ack.go`) carries `{commandId, status, error}` JSON over SSE for honest UI sync-state transitions. `BroadcastOnAck()` hook factory on `Broadcaster` broadcasts when the request carries `X-Command-Id` header (opt-in). `BroadcastOnAckFunc(fn)` for custom payloads. See `docs/adr/0024-honest-ui.md`.
- **Idempotency**: `idempotency.go` is now thin backward-compatible aliases over `go-cqrs-lite/idempotency/v3` (v3.4.0 tagged). Re-exports `IdempotencyStore` (= `idempotency.Store`), `MemoryIdempotencyStore` (= `idempotency.MemoryStore`), `ErrDuplicateCommand` (= `idempotency.ErrDuplicate`), and `NewMemoryIdempotencyStore` wrapper. `CheckAndRecord(ctx, key, ttl)` atomicity (single lock in-memory, future `SET NX` for Redis, `INSERT ON CONFLICT` for SQL). `ErrDuplicateCommand` maps to HTTP 409 Conflict. NOT auto-wired — consumers opt in via `BeforeDispatchHook` or middleware. See `docs/adr/0026-command-idempotency-store.md`.
- **decide() stays on the server** (ADR-0027): The library provides the queue/sync/ACK protocol; the client never runs domain validation. Queue-Only is not just the MVP — it's the only architecturally correct choice for a library that doesn't own the consumer's domain logic. WASM/TS pre-validation is a consumer concern.
- **Offline command queue — Phase 2a** (ADR-0029): `adminui/assets/sync-worker.js` is a SharedWorker (~80 lines vanilla JS) that queues command IDs when the network is down. On reconnect, it tells each originating tab to retry via `htmx.trigger()`. The worker is a **coordinator, not a proxy** — it does NOT send HTTP requests (tabs do, via HTMX), does NOT own the SSE connection (tabs keep per-tab EventSource), and does NOT persist to disk (in-memory only). Offline detection is reactive: `htmx:sendError` (network failure) enqueues; `htmx:responseError` (HTTP error) rejects. Shipped via `go:embed` + served at `GET /-/sync-worker.js`. IndexedDB is banned (ADR-0029); OPFS deferred to Phase 2b. Served automatically when `Config.SSEURL` is set.
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
- **12 events**: `UserRegistered`, `RolesUpdated` (legacy), `EmailChanged`, `DisplayNameChanged`, `UserDeleted` (tombstone), `CredentialAdded`, `CredentialRemoved`, `EmailVerified`, `TOTPEnabled`, `TOTPDisabled`, `ExternalAccountLinked`, `ExternalAccountUnlinked`
- **11 commands**: `RegisterUser`, `ChangeEmail`, `ChangeDisplayName`, `DeleteUser`, `AddCredential`, `RemoveCredential`, `VerifyEmail`, `EnableTOTP`, `DisableTOTP`, `LinkExternalAccount`, `UnlinkExternalAccount`
- **Pure domain layer**: `foldUser()` reconstructs state from events. `decide*()` functions validate guards and emit events. No I/O in domain code.
- **Write path**: Service → CommandDispatcher → DeciderRepository.Execute (load→fold→decide→save→publish)
- **Read path**: Service queries `UserReadModel` (projection from events) — `FindByID`, `FindByEmail`
- **Casbin as projection**: `CasbinProjection` subscribes to events and derives all policies via public Authz methods only.
- **WebAuthn ceremonies**: BeginRegistration/FinishRegistration/BeginLogin/FinishLogin via go-webauthn. Challenge sessions stored in-memory (webauthnSessionStore).
- **Registration = email only**: RegisterRequest has ID + Email + DisplayName. No password field.
- **Read-your-writes consistency**: `watermill.EventBus` (GoChannel backend, `BlockPublishUntilSubscriberAck: true`) blocks publishers until handlers complete, so projections update before `Execute()` returns. `StartProjections` replays historical events from the journal synchronously, then subscribes to live events via `bus.SubscribeAll` with replay→live dedup. No `time.Sleep`-based catch-up. See `es_projection_setup.go`.
- **Checkpoint-based replay (v3.3.0)**: When `EventSourcedConfig.CheckpointStore` is set AND the journal implements `event.SeekableJournal`, `StartProjections` resumes from the last checkpoint via `ReadFrom(afterEventID, 0)` instead of `ReadAll()`. This avoids re-processing the full journal on every restart. Checkpoint saved after each replayed event. When `CheckpointStore` is nil, full journal replay is used (backward compatible). See ADR-0031.
- **BasicCommand embedding (v3.3.0)**: All 20 usermgmt commands embed `*command.BasicCommand`, which promotes `Type()`, `AggregateID()`, `ID()` methods automatically. This structurally eliminates the zero-cmdID bug class (7 constructors previously forgot to mint command IDs, silently breaking idempotency dedup and Watermill message UUIDs). The `mustCommand` helper panics on construction failure (empty type or zero aggID — both programming bugs). See ADR-0032.
- **Sessions NOT event-sourced**: `SessionStore` interface unchanged. Ephemeral auth artifacts.
- **SQLite optimization (v3.1.0)**: `usermgmt.OptimizeSQLiteDB(ctx, db)` wraps `storage.SQLiteEnableWAL` + `storage.SQLiteApplyOptimizations` — WAL mode, synchronous=NORMAL (3-10x write throughput, safe with WAL), busy_timeout=5000ms, 64 MB cache, temp_store=MEMORY, 256 MB mmap. Call BEFORE creating stores: `OptimizeSQLiteDB` → `NewSQLSessionStore` → `NewSQLEventStore`. For Postgres/MySQL it's a no-op. Opt-in — never auto-applied (library principle).
- **SQL event store delegates to upstream (RESOLVED 2026-06-19)**: `usermgmt.SQLEventStore` is now a type alias over `storage.SQLEventStore` from `go-cqrs-lite/storage/v3` (v3.1.0). The hand-rolled 413-LOC store was replaced by a 78-LOC facade. The upstream store provides a richer schema (`schema_version`, `payload_encoding`, `created_at`), OpenTelemetry tracing, `event.SeekableJournal`/`BackwardsSource`, and `event.WrapInfrastructure` error wrapping. `NewSQLEventStore` maps dialect strings → `sqlpkg.Dialect`, creates the upstream store, and applies the event schema DDL. **Breaking changes**: (1) `Close()` no longer closes the `*sql.DB` — upstream uses a borrowed handle; callers must close the DB separately. (2) `Load()` on empty aggregate returns `event.ErrAggregateNotFound` (decider's `Repository` handles this by returning Initial state). (3) MySQL is no longer supported for the event store (upstream has no MySQL dialect). `SQLSessionStore` retains MySQL support — it manages its own schema. **Migration script**: `usermgmt/migrations/0001_user_events_to_events.sql` for Postgres deployments upgrading from `< v2.5.0`. **Fuzz tests**: `FuzzSQLSessionStore_CreateFindRoundTrip` and `FuzzSQLSessionStore_DeleteByUserID` cover arbitrary userID strings (SQL injection, unicode, null bytes). **Benchmarks**: `BenchmarkSQLSessionStore_{Create,Find,FindMiss,Delete,DeleteByUserID,EvictExpired}`.
- **UserID bridge**: `usermgmt.UserID` ↔ `id.AggregateID` via `id.ParseAggregateID(userID.Get())`. Conversion at Service boundary.
- **Email uniqueness**: Pre-checked in Service.Register via read model before dispatching command.
- **DeleteUser revokes sessions**: Sessions deleted on user deletion for security.
- **Old EventHandler backward compat**: If `EventHandler` configured in `ServiceConfig`, Service bridges bus events to old callback.
- **See**: `docs/adr/0006-event-sourced-user-aggregate.md`
- **TOTP via pquerna/otp**: Replaced hand-rolled RFC 6238 with `github.com/pquerna/otp/totp` v1.5.0. `DisableTOTP` requires a valid code to prevent MFA stripping.
- **Import/export authorization**: `ImportExportAuthorizer` (type `AuthorizerFunc`) defaults to `RequireAdminRole`. Configurable via `HandlerConfig`.
- **Per-endpoint rate limiting**: `HandlerConfig.RegistrationRateLimit`, `ImportRateLimit`, `TOTPRateLimit`, `VerificationRateLimit`, `WebAuthnRateLimit`. All use the shared `RateLimitConfig` type (which creates root `cqrshtmx.RateLimiter` instances — token-bucket, proxy-aware KeyExtractor, TTL eviction). All disabled by default.
- **Email value type**: `type Email string` with `ParseEmail`/`MustParseEmail`. Used in `ExportUser`. Internal types stay `string` for event backward compat.
- **UserDataFormat** (renamed from ExportFormat): Used for both import and export. Constants: `UserDataFormatJSON`, `UserDataFormatCSV`.
- **Bot & Impersonation are service-level APIs** (not ghost systems): `Service.RegisterBot/DeleteBot/GetBot/ResolveBotByToken`, `NewAPITokenMiddleware`, and `Service.BeginImpersonation/EndImpersonation` are intentionally service-level — consumers wire their own HTTP routes. These have full security guards (super_admin checks, reason-required, self-impersonation prevention, HMAC token hashing). No HTTP routes exist because the routing scheme, path prefixes, and auth middleware are consumer-specific decisions.
- **Pagination unified**: Both root (`DecodePagination`) and usermgmt (`credential_http.go`) delegate to `query.NewPagination` from go-cqrs-lite. Same defaults (page=1, pageSize=20, max=100). Requesting a page beyond the last page returns an empty page (standard REST, no silent clamping).

### Rate Limiter Unification — 2026-06-28

- **usermgmt → root dependency**: usermgmt now imports `cqrs-htmx/v3` for `RateLimiter`. This is a one-way dependency — root never imports usermgmt. The old claim of "zero mutual imports" is obsolete.
- **One algorithm**: usermgmt's `perIPRateLimiter` (fixed-window, hardcoded RemoteAddr, unbounded memory leak) is DELETED. All 6 per-endpoint limiters now create root `RateLimiter` instances (token-bucket, pluggable `KeyExtractor`, TTL eviction, `MaxKeys` cap).
- **`RateLimiter.Check(r)`**: New non-middleware entry point on root `RateLimiter` — allows per-handler rate checks without wrapping as middleware.
- **`newLimiterFromConfig`**: Creates `cqrshtmx.NewRateLimiter` with `KeyExtractorFromRemoteAddr()` by default. Consumers can upgrade to `KeyExtractorFromClientIP()` for proxy-aware rate limiting.
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

### Admin UI module (adminui) — 2026-06-27

- **PURPOSE**: A ready-made, good-looking Admin Dashboard for usermgmt-backed apps. One-call mount behind session middleware. See `adminui/README.md`.
- **Leaf integration module**: `adminui` depends on root `cqrs-htmx/v3` (reuses `HTMXScriptHandler` for embedded htmx.js) + `usermgmt/v3` + `a-h/templ`. Nothing depends on it. `examples/admin-demo/` is the runnable showcase.
- **templ adopted here (first module to use it)**: Markup authored in `.templ`, compiled to committed `_templ.go`. Consumers import the generated files directly — they never run `templ generate`. The templ runtime is a normal transitive dep. `flake.nix` treefmt already had `templ.enable`. After editing `.templ`, contributors run `templ generate` (CLI v0.3.x) in `adminui/`, then commit both `.templ` and `_templ.go`. Module pins `a-h/templ v0.3.1020` (latest published); CLI may be newer.
- **Two scopes**: `ModeSuperAdmin` (global: dashboard/users/tenants/audit) and `ModeTenantAdmin` (scoped to `Config.TenantID`: dashboard/members/audit). Route table is built per-mode — tenant-admin mode does NOT register `/users` or `/tenants`, so they 404 (dashboard anchored with `GET /{$}` so it doesn't catch-all).
- **Auth-agnostic**: Panel reads `*usermgmt.User` from context (consumer's `NewSessionMiddleware`). No user → 401; authorizer fail → 403. Default authorizer is role-based (overridable via `Config.Authorizer`); helpers `RequireAnyRole(service, domain, roles...)` and `RequireAuthenticated()`.
- **No build step for consumers**: Modern CSS design system (`assets/admin.css`) + tiny vanilla JS (`assets/admin.js`) embedded via `go:embed`. Light/dark via `prefers-color-scheme`; accent color injected inline from `Config.AccentColor`. No Tailwind, no JS framework.
- **HTMX patterns**: Live user search = `GET /users` returns full page, but `HX-Request` returns just the table fragment (`renderPartial`); `hx-select`/`hx-target` swap it; `hx-push-url` syncs URL. Destructive actions use `hx-confirm` + `data-confirm` (JS double-confirm) → POST → `HX-Redirect` (via `redirect()`). Toasts via `HX-Trigger: {"adminui:toast": {...}}` consumed by `admin.js`.
- **usermgmt additions**: `Service.TenantMembers(ctx, tenantID) []*Membership` (read), plus `Service.AddMember/UpdateMemberRoles/RemoveMember` (write) and `ParseActorID(s) ActorID` (inverse of `ActorID.PrefixedString`, for round-tripping actor identity through member URLs). The panel manages members end-to-end: add by email + role, remove per row — both super-admin (on `/tenants/{id}`) and tenant-admin (on `/members`).
- **errorfamily**: adminui follows the no-stdlib-error-constructors rule (`errConfig`/`errForbidden` use `event.NewRejection`). HTTP handlers mostly emit message strings (not error objects), so the rule is naturally satisfied.
- **Tests**: stdlib `testing` + `httptest` (not ginkgo) — pragmatic for a rendering/HTTP-glue module. `seed_render_test.go` is the end-to-end smoke test (seeds users+tenants, asserts they render, exercises search + tenant suspend via HX-Redirect).

## Key Gotchas

### Module & Build

1. **GOWORK=off required**: `go.work` covers root + adminui + usermgmt + integration_test + examples. `GOWORK=off` needed for CI/commands using per-module go.mod
2. **Module path casing**: go-cqrs-lite uses lowercase `github.com/larsartmann/go-cqrs-lite` (not `LarsArtmann`)
3. **go-cqrs-lite v3.4.0**: Per-module tags (`command/v3.4.0`, `event/v3.4.0`, etc.) published. Modules declare the latest tag per module (most at v3.4.0; decider/id/otel/watermill/codec/dispatcher/catalog/schema at v3.3.0/v3.3.1; transport/http at v3.3.1). No replace directives needed. v3.4.0 adds managed projection host (`projectionhost`), durable scheduling, scenario-testing DSL, and `go mod tidy` sweep. v3.3.0 added `command.Command.ID()` (embed `command.BasicCommand`), `event.Projection` moved to `projection/`, SQL dead-letter store. v3.1.0 added SQL-backed view stores, typed metadata, multi-DB split presets
4. **Removed APIs in v2.3.0+**: `query.MustNew`, `command.MustNew`, `id.MustParse[T]` removed — use `query.New()`, `command.New()`, `id.Parse[T]()` with error check instead. Our `MustParseUserID`/`MustParseCorrelationID`/`MustParseRequestID` are local wrappers around `Parse`
5. **golangci-lint v2 format**: `.golangci.yml` uses `version: "2"`. Exclusions under `linters.exclusions.rules`, NOT `issues.exclude-rules`
6. **LSP vs CLI discrepancy**: LSP may show stale warnings after golangci.yml changes; CLI (`golangci-lint run`) is authoritative. Both report 0 issues as of go-error-family v0.5.1 adoption
7. **flake.nix uses flake-parts + treefmt**: Nix formatting via `nix fmt` (treefmt with nixfmt + gofmt). No package builds in nix due to private Go deps — use `nix run .#build`/`nix run .#test` apps instead

### Type System

6. **UserID breaking change**: `WithUserID`/`UserIDFromContext` use `id.UserID` (ULID-backed). `UserIDExtractor` still returns `string`. Use `ParseUserID`/`MustParseUserID`
7. **CorrelationID breaking change**: Branded `id.CorrelationID`. Non-ULID header values silently dropped by middleware
8. **usermgmt UserID is different type**: `usermgmt.UserID = brandid.ID[userBrand, string]` — NOT the same as root `id.UserID`. Cross-module bridge uses `.Get()` (not `.String()` which includes brand prefix)
9. **ActorID/ImpersonatorID/SSEEventID branded**: Root's `ActorID`, `ImpersonatorID`, and `SSEEventID` are now `brandid.ID[brand, string]` — phantom-typed with `.Get()`, `.IsZero()`, `.Equal()`. `ImpersonatorID = ActorID` (type alias — an impersonator IS an actor). Use `NewActorID("user:01JX...")` constructor, not `ActorID("...")` cast. `.String()` returns brand-prefixed form for debug; use `.Get()` for raw value. See ADR-0028
10. **usermgmt ActorID is a struct**: `usermgmt.ActorID` is a kind-discriminated struct (`{kind, raw}`), NOT a brandid type. Bridging to root uses `.PrefixedString()` → `cqrshtmx.NewActorID()`. The middleware bridging bug (`.String()` instead of `.PrefixedString()`) is fixed
11. **GroupPolicy.User/Domain remain string**: Casbin boundary types. Intentional.

### HTTP & Security

10. **CSRF middleware ordering**: `Chain(CSRFMiddleware, HTMXMiddleware, app.Middleware())` — CSRF first, then HTMX, then enrichment
11. **nosurf custom headers**: Must use `translateCSRFHeaders` to map consumer header/field names to nosurf defaults
12. **HX-Redirect sanitization**: `Response.Redirect()` sanitizes URLs for both HTMX and non-HTMX requests
13. **HandlerConfig.Secure uses \*bool**: `Secure` is `*bool` in usermgmt — nil defaults to true. Use `new(bool)` (false) or a local variable (true) to explicitly set. The zero-value `HandlerConfig{}` is safe
14. **Max password length 128**: Enforced in `RegisterRequest.Validate()` and `Service.ChangePassword()`. Prevents bcrypt CPU abuse
15. **CSRF TrustedProxies**: `setPlaintextHTTPOrigin` only auto-sets `Sec-Fetch-Site: same-origin` for plain HTTP requests when the remote is loopback OR in `CSRFConfig.TrustedProxies` (single IP or CIDR). Empty config logs a warning but allows it (back-compat). Set `TrustedProxies` in production to prevent CSRF bypass via origin-header stripping.
16. **Rate limiter RLock scope**: `perKeyLimiter.limiter()` must read `entry.lastUsed` **while holding RLock** — the field is written under the write lock during refresh. Reading it after `RUnlock` causes a data race under concurrent load (fixed 2026-06-14; was ~20% race-detector failure rate).

### Error Handling

15. **No stdlib error constructors**: `errors.New`/`fmt.Errorf`/`errors.Join` are banned in non-test code — `branching-flow errorfamily .` enforces 0 violations. Use `event.New*/Wrap*/Wrapf/Newf` (re-exported via `event/v3`). For dispatch errors that must preserve the inner domain family, use `event.Wrapf(err, event.Classify(err), code, msg)` — never force a family. `fmt.Sprintf` is allowed only for message strings, not error objects
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

### adminui module

```bash
# All tests (renders the full panel via httptest + seeded data)
cd adminui && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Regenerate templ after editing *.templ (CLI: templ v0.3.x). Commit *_templ.go too.
cd adminui && templ generate
```

### datastar-demo example

```bash
# Build only (no tests — it's a main package)
cd examples/datastar-demo && GOWORK=off go build ./...
```

### admin-demo example

```bash
# Build & run the admin panel showcase, then open http://localhost:8097/
cd examples/admin-demo && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go run .
```

### SQL-Backed Read Models (v3.1.0)

- **`SQLUserReadModel`**: Persistent read model wrapping `storage.SQLViewStore[UserView, UserID]`. Data survives restart. Wraps in-memory `UserReadModel` for read-your-writes consistency.
- **`SQLMembershipReadModel`**, **`SQLTenantReadModel`**, **`SQLBotReadModel`**: Same pattern for all 4 aggregates.
- **`UserView`**: DTO with scalar SQL columns (email, display_name, etc.) + JSON blob column (`Data`) for full User reconstruction. Time fields stored as RFC3339 strings (SQLite driver limitation).
- **Construction**: `NewSQLiteUserReadModel(db)` or `NewSQLUserReadModel(db)` (Postgres). Auto-migrates the view table.
- **Query methods**: `FindByIDSQL`, `FindByEmailSQL`, `CountSQL`, `FindByActorSQL`, `FindByNameSQL`, `FindByTokenHashSQL` — all use `kv.ViewQuery` DSL for server-side filtering.
- **ServiceConfig hook**: `ReadModelDB *sql.DB` — when set, `NewService` creates SQL-backed read models automatically.
- **EventSourcedConfig hook**: Same `ReadModelDB` field.
- **`AutoMapper[V]`**: Generates `ViewMapper` from `view:"col"` struct tags. No manual column mapping needed.

### Stack Preset Adoption (v3.1.0)

- **`NewSQLiteEventSourcedSetup(cfg)`**: One-call setup using `stack/sqlite.New(dsn)`. Creates event store + bus + 4 SQL read models + projections + graceful shutdown. Recommended for production single-process deployments.
- **`NewPostgresEventSourcedSetup(cfg)`**: Same for PostgreSQL. Supports multi-DB split (`EventDSN`/`QueryDSN` for I/O isolation).
- **SecurityHooks preserved**: Stack presets don't expose injection points directly, but `stack.Repository(bundle, decider)` accesses the internal store. The wrapping happens via `wrapEventStore` before repository creation.
- **`stack.Materialize[V,K]`**: Evaluated in depth (ADR-0022). A **generic `MaterializeProjection[V,K]` adapter** (`es_materialize_adapter.go`) wraps any `stack.Materialize` as `event.Projection` using `watermill.EventToMessage` → `Materialize.HandlerFunc` round-trip — zero dispatch replication. Any read model can adopt Materialize with a single `NewMaterializeProjection(mat, name, eventTypes)` call. **Per-read-model fit**: Tenant/Bot = good fit (few events, tombstone-marked). Membership = moderate (no tombstone marking yet). UserReadModel = rejected (composite `externalAccounts` index doesn't map to `ViewQuery`; 12-event switch moves but doesn't shrink). CasbinProjection = never (side-effecting, no value state). Key correction: the "12-event switch doesn't fit" rationale was partially wrong — `OnUpdate` receives ALL events and you branch internally; the real value is the `kv.ViewStore` backend (`ViewQuerier`/`ViewCounter`/`ViewResetter`/`TombstoneQuerier`).
- **`CatchUpSubscriber`**: Not yet adopted — `StartProjections` still uses manual journal replay + `bus.SubscribeAll`. Migration is a future task.
- **`stack/v3` + `stack/sqlite/v3` + `stack/postgres/v3`**: New dependencies in usermgmt. Added `pgx/v5` transitively.

### CI Gates

- **`nix run .#errorfamily`**: Runs `branching-flow errorfamily` on root + usermgmt. Verifies zero stdlib error constructors.
- **`nix run .#coverage-gate`**: Tests all modules and fails if coverage drops below thresholds (root 90%, usermgmt 75%).
