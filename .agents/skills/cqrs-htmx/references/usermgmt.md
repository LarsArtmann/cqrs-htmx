# usermgmt reference

Import: `"github.com/larsartmann/cqrs-htmx/usermgmt/v4"`. This submodule provides event-sourced user management: register/login/logout, WebAuthn passkeys, OAuth2/OIDC, TOTP, roles/tenants/bots, and Casbin authorization.

**Authentication is passwordless.** There is no password column. Login is via WebAuthn (passkeys) or OAuth2/OIDC. TOTP is an optional second factor.

## Setup matrix — pick one

| Persistence                           | How                                                              | What you get                                                                                                                   |
| ------------------------------------- | ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| **In-memory** (dev/prototyping)       | `NewService(ServiceConfig{})`                                    | `*Service` with in-memory event store, bus, read models. Lost on restart.                                                      |
| **SQL read models, in-memory events** | `NewService(ServiceConfig{ReadModelDB: db})`                     | Persistent read models, ephemoral event log.                                                                                   |
| **SQLite full stack** (one call)      | `NewSQLiteEventSourcedSetup(SQLiteSetupConfig{DSN: "users.db"})` | `*SQLiteEventSourcedSetup` (repos, bundle, `*sql.DB`, read models, projections). Wire into `NewService` or use repos directly. |
| **Postgres full stack** (one call)    | `NewPostgresEventSourcedSetup(PostgresSetupConfig{DSN: ...})`    | Same, Postgres. Supports `EventDSN`/`QueryDSN` for I/O isolation.                                                              |
| **Event signing/encryption** (opt-in) | `ServiceConfig{SecurityHooks: SecurityHooks{...}}`               | Wrap store + bus for at-rest encryption / signing. See "Security hooks" below.                                                 |

For HTTP endpoints you always want a `*Service` (the one-call setups return infrastructure, not HTTP). The simplest end-to-end path is `NewService` → `NewAuthHandler` → `RegisterRoutes`.

## `NewService` and `ServiceConfig`

```go
svc, err := usermgmt.NewService(cfg)
```

Every field is optional (zero value = default):

```go
type ServiceConfig struct {
	EventStore  event.Store         // nil → in-memory
	EventBus    event.Bus           // nil → watermill GoChannel (read-your-writes consistency)
	ReadModelDB *sql.DB             // nil → in-memory read models; set for SQL-backed reads
	Authz       *Authz              // nil → new Authz with default RBAC policies
	SessionStore SessionStore       // nil → InMemorySessionStore
	SessionTTL   time.Duration      // 0 → 24h
	Logger      *slog.Logger        // nil → slog.Default()
	Lockout     LockoutStore        // nil → no lockout

	WebAuthnConfig         *WebAuthnConfig         // REQUIRED for passkey login
	WebAuthnSessionStore   WebAuthnSessionStore    // nil → in-memory (use Redis for multi-instance)
	EmailVerification      *EmailVerificationConfig
	VerificationTokenStore VerificationTokenStore
	TOTPConfig             *TOTPConfig
	PendingTOTPStore       PendingTOTPStore
	OAuth2Config           *OAuth2Config
	OAuth2StateStore       OAuth2StateStore        // nil → in-memory (use Redis for multi-instance)

	AuditLog        *AuditLog
	CheckpointStore event.CheckpointStore          // nil → full journal replay on startup
	TokenPepper     TokenPepper                    // REQUIRED for bot registration / API token auth
	SecurityHooks                                  // embedded; optional event signing/encryption
}
```

### Lifecycle

```go
svc.Stop()               // stop background eviction goroutines only (idempotent)
svc.Close()              // Stop() + close bus/store if they are io.Closer (idempotent)
svc.GracefulClose(ctx)   // Close() + return ctx.Err() if context cancelled
defer svc.Close()        // typical usage
```

## HTTP: `AuthHandler` and routes

```go
auth := usermgmt.NewAuthHandler(svc /*, optional HandlerConfig */)  // *AuthHandler
mux := http.NewServeMux()
auth.RegisterRoutes(mux)   // mutates YOUR mux — you own routing
```

Routes registered (Go 1.22+ method patterns):

| Method    | Path                                                                 | Notes                                                 |
| --------- | -------------------------------------------------------------------- | ----------------------------------------------------- |
| POST      | `/auth/register`                                                     | email-only registration; returns user + session token |
| POST      | `/auth/webauthn/register/begin` \ `/finish`                          | passkey enrollment ceremony                           |
| POST      | `/auth/webauthn/login/begin` \ `/finish`                             | passkey login ceremony                                |
| POST      | `/auth/logout`                                                       | invalidate session                                    |
| GET       | `/auth/me`                                                           | current user JSON                                     |
| GET       | `/auth/credentials` \ DELETE `/auth/credentials/{id}`                | manage passkeys                                       |
| (various) | `/auth/verify/*`, `/auth/totp/*`, `/auth/import/*`, `/auth/export/*` | from `RegisterVerificationTOTPRoutes`                 |
| GET       | `/auth/oauth/{provider}/begin` \ `/callback`, POST `/unlink`         | OAuth2 (when configured)                              |

`HandlerConfig` knobs (all optional): per-endpoint `RateLimitConfig`s, `CookieName`, `Secure *bool` (nil→true; set with a `*bool`), `SessionMaxAge`, `Timeout`, `OAuth2SuccessURL`/`OAuth2ErrorURL`, `ImportExportAuthorizer` (default `RequireAdminRole`).

## Session middleware (auth for your own routes)

```go
sessionMW := usermgmt.NewSessionMiddleware(svc, "session")  // (*Service, cookieName)
```

Reads the cookie, validates the token, and injects the authenticated `*usermgmt.User` into the request context. Put it **outside** (before) CSRF in your chain. To read the user in a handler: `usermgmt.UserFromContext(r.Context())`.

## Auth methods

### WebAuthn (passkeys) — primary login

```go
usermgmt.WebAuthnConfig{
	RPID:          "example.com",        // domain (no scheme/port)
	RPDisplayName: "My App",
	RPOrigins:     []string{"https://example.com"},
}
```

Pass to `ServiceConfig.WebAuthnConfig`. The HTTP ceremony endpoints are mounted by `RegisterRoutes`. Registration and login are two-step (begin → finish) JSON POST flows. On localhost use `RPID: "localhost"`, `RPOrigins: []string{"http://localhost:8080"}`.

### OAuth2 / OIDC (Google, GitHub, etc.)

```go
usermgmt.OAuth2Config{
	Providers: map[string]usermgmt.OAuth2ProviderConfig{
		"google": {ClientID, ClientSecret, RedirectURL,
			IssuerURL: "https://accounts.google.com"},   // OIDC discovery
		"github": {ClientID, ClientSecret, RedirectURL,
			AuthURL: "...", TokenURL: "...", UserInfoURL: "..."},  // pure OAuth2
	},
	StateTTL: 10 * time.Minute,
}
```

- `IssuerURL` set → OIDC discovery + ID-token verification.
- `IssuerURL` empty → explicit endpoints + UserInfo fetch.
- PKCE (S256) is always on; CSRF state tokens are one-time-use.
- **Subject-first matching**: returning users are matched by `(provider, subject)` first, then email. Multiple providers can link to one user.
- Auto-trusts provider emails when `email_verified: true`.
- `UnlinkExternalAccount` is rejected if it would leave the user with 0 passkeys AND ≤1 external accounts (lockout prevention).

### TOTP (optional second factor)

```go
usermgmt.TOTPConfig{Issuer: "My App", Window: 1}  // Window=1 → ±30s drift
```

EnableTOTP returns a secret + QR URL; VerifyTOTP checks a code; DisableTOTP requires a valid code (prevents MFA stripping). Routes mounted via `RegisterRoutes`.

### Email verification

`EmailVerificationConfig` + a `SendVerificationEmailFunc`. `VerifyEmail` marks the email verified; `RegisterRoutes` mounts `/auth/verify/*`.

## Roles, authorization, and Casbin

Built-in roles (`usermgmt/authz_types.go`):

- `RoleSuperAdmin` — unrestricted global access across all tenants.
- `RoleAdmin` — unrestricted access (default policy: wildcard allow).
- `RoleUser` — standard role assigned on registration.
- `RoleOwner`, `RoleViewer` — additional roles.

New users are registered with `[RoleViewer, RoleUser]`. Super-admin grants: `svc.Authz().AddGroupPolicy(usermgmt.GroupPolicy{Subject, Role: RoleSuperAdmin, Domain})`.

`Authz` wraps Casbin (RBAC with domains). `*casbin.Enforcer` satisfies the root `cqrshtmx.Enforcer` interface — pass `svc.Authz().AsEnforcer()` to `cqrshtmx.Config.Enforcer` to use the same policy store for route-level authz. Policies are **derived from events** by `CasbinProjection` (a read model of the event stream), so they're always consistent with aggregate state.

Useful authz methods: `RolesForUser`, `RolesForActor`, `ImplicitRolesForActor`, `AddGroupPolicy`, `RemoveGroupPolicy`, `AddPolicy`, `RemovePolicy`, `UsersForRole`.

## Tenants, members, bots, impersonation (service-level APIs)

These are intentionally **service-level, not HTTP routes** — the routing scheme and auth middleware are consumer-specific decisions. Wire your own endpoints.

| Concern               | Service methods                                                                                                                                                                                                |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Tenants**           | `CreateTenant`, `SuspendTenant`, `ReactivateTenant`, `DeleteTenant`, `GetTenant`, `TenantMembers(ctx, tenantID)`                                                                                               |
| **Members**           | `AddMember(ctx, actor, tenantID, roles)`, `UpdateMemberRoles`, `RemoveMember`                                                                                                                                  |
| **Bots** (API tokens) | `RegisterBot`, `DeleteBot`, `GetBot`, `ResolveBotByToken`. Requires `ServiceConfig.TokenPepper`. Use `NewAPITokenMiddleware(svc)` + `RequireBot` for bot auth. Tokens are HMAC-hashed, never stored plaintext. |
| **Impersonation**     | `BeginImpersonation(ctx, adminID, targetID, reason)`, `EndImpersonation`. Guards: super-admin only, reason required, self-impersonation blocked.                                                               |

Identity types: `usermgmt.ActorID` is a kind-discriminated struct (`{kind, raw}`) — bridging to root uses `.PrefixedString()` → `cqrshtmx.NewActorID(...)`. `ParseActorID(s)` is the inverse. `TenantID`/`BotID` are branded string types.

## SQL persistence details

- **`OptimizeSQLiteDB(ctx, db)`** — call BEFORE creating stores. Enables WAL + `synchronous=NORMAL` + busy_timeout + 64MB cache + mmap. No-op on Postgres/MySQL. Opt-in.
- Construction order: `OptimizeSQLiteDB` → `NewSQLSessionStore` → `NewSQLEventStore` (or the one-call setup).
- **`SQLEventStore.Close()` does not close the `*sql.DB`** (upstream borrows the handle) — close the DB yourself.
- **MySQL is not supported for the event store** (upstream has no MySQL dialect). `SQLSessionStore` still supports MySQL.
- Migration script for Postgres upgrades from <v2.5.0: `usermgmt/migrations/0001_user_events_to_events.sql`.

## Security hooks (event signing / encryption)

Opt-in seams — the consumer imports `go-cqrs-lite/signing/v2` or `/encryption/v2`; usermgmt never imports them.

```go
type SecurityHooks struct {
	StoreWrapper      func(event.Store) (event.Store, error)  // wrap store before journal detection
	PublishMiddleware []event.PublishMiddleware               // applied via bus.UsePublish (sign/encrypt)
	HandlerMiddleware []event.Middleware                       // applied via bus.Use (verify/decrypt)
}
```

Recommended: store-level encryption (`NewEncryptedStore`) + bus-level signing (`SignMiddleware`). For sign+encrypt, sign plaintext _before_ encrypting; for decrypt+verify, decrypt _before_ verifying. See `docs/adr/0011-event-signing-encryption.md`.

## Read-your-writes consistency

The default `watermill.EventBus` (GoChannel, `BlockPublishUntilSubscriberAck: true`) blocks publishers until projection handlers complete, so read models update _before_ `Execute()` returns. `StartProjections` replays all historical events synchronously first, then subscribes to live events with replay→live dedup. No `time.Sleep`-based catch-up. This means a `Register` followed immediately by a `FindByID` returns the new user.

## UserID bridge (root ↔ usermgmt)

- `usermgmt.UserID` = `brandid.ID[userBrand, string]` (string-backed).
- `cqrshtmx.UserID` = ULID-backed (`go-cqrs-lite/id`).
- They are **not interconvertible by assignment**. Bridge with `.Get()` (raw value), never `.String()` (which is brand-prefixed and meant for debug).
- When wiring usermgmt session context into root `App` handlers, convert explicitly via `cqrshtmx.ParseUserID(...)` on the raw value.

## Import / export users

Bulk-load or dump users via the `Service`. Routes are mounted by `RegisterRoutes` (`/auth/import/*`, `/auth/export/*`); authorizer defaults to `RequireAdminRole` (override via `HandlerConfig.ImportExportAuthorizer`).

```go
// Import (JSON or CSV) — returns counts of created/skipped/failed:
res, err := svc.ImportUsersFromJSON(ctx, r.Body)
res, err := svc.ImportUsersFromCSV(ctx, r.Body)

// Export:
err := svc.ExportUsersToJSON(ctx, w)
err := svc.ExportUsersToCSV(ctx, w)
```

`ImportUser` is the per-row input struct (with `.Validate()`); `UserDataFormat` constants are `UserDataFormatJSON` / `UserDataFormatCSV`. Per-endpoint rate limiting via `HandlerConfig.ImportRateLimit`.

## Account lockout (brute-force protection)

Opt-in. Defaults: 5 attempts, 15-minute lockout. State is **in-memory only** (lost on restart) — use a distributed store in production.

```go
lockout := usermgmt.NewAccountLockout()                              // defaults
lockout := usermgmt.NewAccountLockout(usermgmt.LockoutConfig{
    MaxAttempts: 10, Duration: 30 * time.Minute,
})
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{Lockout: lockout})
```

Call `lockout.EvictStale()` periodically (or rely on the service's background eviction) to prune expired entries.

## Server-Timing header (debug profiling)

Opt-in W3C `Server-Timing` response header for latency debugging. Three entry points (see AGENTS.md §"Server Timing API" for internals):

```go
// 1. Always-on middleware:
http.ListenAndServe(addr, cqrshtmx.ServerTimingMiddleware()(mux))

// 2. Debug-gated (recommended) — only emit when ?debug=1 or admin:
http.ListenAndServe(addr, cqrshtmx.ServerTimingMiddlewareWhen(func(r *http.Request) bool {
    return r.URL.Query().Get("debug") == "1"
})(mux))

// 3. One-line on App-managed routes (no separate middleware):
app := cqrshtmx.MustNew(cqrshtmx.Config{
    Commands:     cmdDisp,
    ServerTiming: func(r *http.Request) bool { return r.URL.Query().Get("debug") == "1" },
})

// Record a span inside a handler:
stop := cqrshtmx.MeasureServerTiming(r.Context(), "db")
// ... db query ...
stop()  // end BEFORE WriteHeader/Write or it misses the header
```

**TTFB gotcha:** a metric's region must end before the response is committed — `defer stop()` records at function return (after the write), missing the header. Zero overhead when off (nil-receiver pattern: 4.3 ns/op, 0 allocs).
