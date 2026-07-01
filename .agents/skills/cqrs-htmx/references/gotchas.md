# Gotchas (complete consumer reference)

The most frequent and painful mistakes consumers make. Each has a concrete fix.

## 1. Module & versioning

### `/v3` suffix is mandatory

Go modules at v2+ require the major-version suffix in the import path. Forgetting it gives confusing "module not found" or silent wrong-version errors.

```go
// RIGHT
cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
"github.com/larsartmann/cqrs-htmx/adminui/v4"
"github.com/larsartmann/go-cqrs-lite/command/v3"

// WRONG (won't resolve, or resolves to an ancient v1)
"github.com/larsartmann/cqrs-htmx"
```

### go-cqrs-lite casing is lowercase

`github.com/larsartmann/go-cqrs-lite` — NOT `LarsArtmann/go-cqrs-lite`. The org casing differs from `cqrs-htmx`'s repo. Mix it up and `go mod` fails.

### `GOWORK=off` for submodules

`go.work` covers root + adminui + usermgmt + integration_test + examples. When building/testing a submodule in isolation, set `GOWORK=off` so it uses its own `go.mod`. The `flake.nix` per-module apps do this for you (`nix run .#test-usermgmt`, etc.).

### `go.work` doesn't affect `go get` / `go mod tidy`

If a consumer imports your library, they only see published tags. Make sure tags exist: `usermgmt/v3.x.y`, `adminui/v3.x.y` (not just `v3.x.y` at root).

## 2. Type system

### Two different `UserID` types

- Root `cqrshtmx.UserID` — ULID-backed (`go-cqrs-lite/id`). `String()` returns the ULID.
- usermgmt `usermgmt.UserID` — string-backed (`go-branded-id`). `String()` returns a **brand-prefixed** debug form.

They are **not assignable** to each other. Bridge with `.Get()` (raw value), never `.String()`:

```go
uid := userFromSession.ID                       // usermgmt.UserID
rootID, err := cqrshtmx.ParseUserID(uid.Get())  // convert → cqrshtmx.UserID
```

Wiring context without this conversion silently puts the wrong (brand-prefixed) string into root handlers.

### `usermgmt.ActorID` is a struct, not a brandid

`ActorID{kind, raw}` — kind-discriminated (`"user"`, `"bot"`, etc.). Bridge to root with `.PrefixedString()` → `cqrshtmx.NewActorID(...)`. `ParseActorID(s)` is the inverse. Using `.String()` here instead of `.PrefixedString()` is a known past bug.

### Root `ActorID` / `ImpersonatorID` are branded

`ImpersonatorID = ActorID` (type alias — an impersonator IS an actor). Construct with `cqrshtmx.NewActorID("user:01JX...")`, not `ActorID("...")` cast.

### Branded IDs don't cross-assign

`SSEEventID`, `CorrelationID`, `RequestID`, `TenantID`, `BotID` are all distinct branded types. Compile-time safety is the point — don't defeat it with `.String()` round-trips.

## 3. HandlerOption ordering

### Validator MUST come after the decoder

`ValidateCommand` / `ValidateQuery` read the decoded value off the request context. If you put them before `DecodeJSON`, validation silently no-ops (nothing decoded yet).

```go
// RIGHT: authorize → decode → validate → respond
cqrshtmx.Authorize("widgets", "create"),
cqrshtmx.DecodeJSON(mapRequest),
cqrshtmx.ValidateCommand(validateWidget),

// WRONG: validate runs before decode → does nothing
cqrshtmx.ValidateCommand(validateWidget),
cqrshtmx.DecodeJSON(mapRequest),
```

### Authorize first

Put `Authorize`/`RequireAuth` first in the option list so unauthorized requests fail fast before you read or decode the body.

### `App.Command("")` panics

Empty command/query type strings are rejected at registration. Always use real `command.Type("CreateItem")` / `query.Type("ListItems")` values.

## 4. Middleware ordering

### The cqrs-htmx trio: `CSRF → HTMX → enrichment`

- `app.Middleware()` (context enrichment) reads HTMX context set by `HTMXMiddleware`. So HTMX must run **before** enrichment.
- CSRF must wrap mutations, so it runs **outside** HTMX.
- Getting this order wrong causes silent CSRF gaps or missing `HTMXRequest` in context.

### Session/auth goes OUTSIDE CSRF

```
Recovery → SecurityHeaders → [session/auth] → CSRF → HTMX → app.Middleware() → handler
```

If CSRF is outside session, you may leak CSRF tokens to unauthenticated clients or skip auth checks.

### `SecurityHeadersMiddleware` and `CSRFMiddleware` are never auto-applied

Library principle: never enforce defaults consumers might disagree with. You must add them to your chain. For production, use `SecurityHeadersMiddlewareWithConfig` with `RecommendedCSP`/`RecommendedHSTS`.

## 5. Errors (the `errorfamily` rule)

### Banned: `errors.New` / `fmt.Errorf` (as error) / `errors.Join`

Enforced by `branching-flow errorfamily .` (must report 0). Inside this codebase, use `event.New*`/`event.Wrap*`/`event.Wrapf`/`event.Newf` from `go-cqrs-lite/event/v3`:

```go
return event.NewRejection("app.bad_input", "email is required")     // → HTTP 400
return event.NewConflict("app.duplicate", "email exists")           // → HTTP 409
return event.Wrapf(err, event.Classify(err), code, msg)            // preserve inner family
```

`fmt.Sprintf` is fine **when building a message string** (not an error object), e.g. formatting a 400 response body.

### Never force a family on dispatch errors

A dispatch error may carry a domain Rejection/Conflict. Forcing it to Transient would change the HTTP status and break clients. Use `event.Wrapf(err, event.Classify(err), code, msg)` to preserve the inner family.

### Preserve sentinel identity where `errors.Is` is relied upon

`errors.Is(err, ErrValidation)` is used in places (`ParseEmail`, `ImportUser.Validate`). Wrap `ErrValidation` as the cause with `event.WrapRejection(ErrValidation, ...)` — don't replace it.

## 6. CSRF specifics

### `CSRFConfig.Secure` and `HandlerConfig.Secure` use `*bool`

In usermgmt, `Secure` is `*bool` — nil defaults to true. To set false explicitly you must use a pointer:

```go
Secure: new(bool),              // false
// or
sec := true
Secure: &sec,
```

The zero-value `HandlerConfig{}` is safe (defaults to secure).

### `CSRFConfig.TrustedProxies` for plain HTTP origins

`setPlaintextHTTPOrigin` auto-sets `Sec-Fetch-Site: same-origin` for plain HTTP only when the remote is loopback OR in `TrustedProxies`. Empty config logs a warning but allows it (back-compat). **Set `TrustedProxies` in production** to prevent CSRF bypass via origin-header stripping.

### nosurf custom header/field names

If you use custom `HeaderName`/`FieldName` in `CSRFConfig`, the library translates them to nosurf's defaults internally. Don't try to register them with nosurf directly.

### MaxBodySize defaults

When both `Config.MaxBodySize` and per-handler `WithMaxBodySize` are zero, `DefaultMaxBodySize` (10 MB) applies. Set one of them to enforce a different limit.

## 7. usermgmt specifics

### `SQLEventStore.Close()` does NOT close the `*sql.DB`

The upstream store borrows the handle — you must close the DB yourself. Same for `SQLSessionStore`.

### MySQL not supported for the event store

`go-cqrs-lite/storage/v3` has no MySQL dialect. `SQLSessionStore` still supports MySQL (it manages its own schema). Use Postgres or SQLite for events.

### `OptimizeSQLiteDB` before creating stores

Call `usermgmt.OptimizeSQLiteDB(ctx, db)` **before** `NewSQLSessionStore` / `NewSQLEventStore`. It enables WAL + `synchronous=NORMAL` + busy_timeout. No-op on Postgres/MySQL. Opt-in.

### WebAuthnConfig is REQUIRED for passkey login

Passkeys are the default auth method. If you don't set `WebAuthnConfig`, login endpoints can't function. For localhost: `RPID: "localhost"`, `RPOrigins: []string{"http://localhost:8080"}`.

### `TokenPepper` is REQUIRED for bots

`Service.RegisterBot` / `ResolveBotByToken` need a `TokenPepper` set in `ServiceConfig`. Without it, bot registration panics or returns an error.

### Session stores, OAuth2 states, WebAuthn sessions are NOT event-sourced

These are ephemeral auth artifacts. Defaults are in-memory — for multi-instance deployments, implement `SessionStore` / `OAuth2StateStore` / `WebAuthnSessionStore` against Redis. The `Service.Stop()`/`Close()` methods manage their eviction goroutines.

### `Service.Close()` may not close the bus/store

In-memory `memory.NewMemoryStore()` and `watermill.NewEventBus()` may or may not implement `io.Closer`. `Close()` handles both via type assertion. If you provided your own store/bus, close them yourself if needed.

### `DeleteUser` revokes sessions

For security, deleting a user deletes their sessions. Don't expect a deleted user's session to remain valid.

### Email-based OAuth2 linking can surprise

A new OAuth2 provider with a matching email **links to the existing user** (not a new account). Multiple providers (Google + GitHub) can link to one user. `UnlinkExternalAccount` is rejected if it would leave 0 passkeys AND ≤1 external accounts (lockout prevention).

## 8. Realtime

### `Broadcaster.Unsubscribe` closes the channel

Never send on a channel after Unsubscribe. The read-loop pattern (`defer broadcaster.Unsubscribe(ch)`) is safe because the read loop exits when the channel closes.

### Broadcasts are non-blocking

Slow consumers silently drop events. If you need guaranteed delivery, pair with idempotency + ACK so the client retries on reconnect.

### ACK is opt-in (`X-Command-Id` header)

Without the header, no ACK is broadcast and idempotency is a no-op. The frontend must generate and attach the ID for every mutation. See `adminui/assets/admin.js` for the reference client implementation.

### SSE replay needs monotonic event IDs

For `JournalSSEStore` replay to work, your events need ULID IDs the journal can order. The default event store provides these.

### Server-Timing header must be set before the body

Like all HTTP headers, `Server-Timing` is committed at first `WriteHeader`/`Write`. `MeasureServerTiming(ctx, "db")` with `defer` records at function return — which is AFTER the write for non-streaming handlers, so it misses the header. For a metric you want in the header, end its region before the write.

## 9. Testing & build tooling (contributor concerns)

The testing conventions (`bdd` type prefix, no `time.After`/`select`), the nix-based build commands (`nix run .#test`/`.#lint`/`.#coverage`), the `errorfamily` and coverage CI gates, and the LSP-vs-CLI discrepancy are all documented in the project's **AGENTS.md** under "Test Commands" and "Key Gotchas." They are contributor concerns, not consumer concerns — consult AGENTS.md directly when working inside this repo.

## 10. adminui

### The panel reads `*usermgmt.User` from context

Your session middleware must put it there. `usermgmt.NewSessionMiddleware` already does. If you roll your own session middleware, use `usermgmt.WithUser(ctx, u)`.

### `Mount` requires a trailing slash

`panel.Mount(mux, "/admin/")` — the standard mux requires the trailing slash for prefix matching. Use `"/"` to host at the site root.

### `ModeTenantAdmin` requires `TenantID`

`Config.TenantID` must be set when `Mode == ModeTenantAdmin`. Super-admin mode ignores it. Tenant-admin mode does NOT register `/users` or `/tenants` routes — they 404 by design (scoped panel).

### templ files are pre-compiled

`*_templ.go` is committed. Consumers import the generated files directly — they never run `templ generate`. Only contributors editing `.templ` run `templ generate` (CLI v0.3.x) in `adminui/`.

### The panel serves htmx.js internally

On Path C the panel registers its own htmx.js route (`adminui/assets.go`). **Do NOT** also register `cqrshtmx.HTMXScriptHandler()` yourself — that creates a duplicate route. This is the Path A/B exception: only self-host htmx.js when you're NOT using adminui.
