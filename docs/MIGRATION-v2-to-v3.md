# Migrating from v2 to v3

This guide covers all breaking changes when upgrading from cqrs-htmx v2 to v3.

## Import Paths

All modules move to the `/v3` suffix:

```go
// v2
import "github.com/larsartmann/cqrs-htmx"
import "github.com/larsartmann/cqrs-htmx/usermgmt"

// v3
import "github.com/larsartmann/cqrs-htmx/v3"
import "github.com/larsartmann/cqrs-htmx/usermgmt/v3"
```

Update all import paths with a project-wide find-and-replace.

## Breaking Changes

### 1. UserID Is Now ULID-Backed (root module)

`UserID` changed from a string type alias to a branded ULID type:

```go
// v2
uid := cqrshtmx.UserID("123")
cqrshtmx.WithUserID(ctx, "123")

// v3
uid := cqrshtmx.NewUserID("123")       // accepts ULID or derives one
cqrshtmx.WithUserID(ctx, uid)
```

Use `ParseUserID(s)` for validated parsing, `NewUserID(s)` for tolerant construction.

### 2. CorrelationID and RequestID Are Branded Types

```go
// v2
ctx = cqrshtmx.WithCorrelationID(ctx, "abc-123")

// v3
cid, _ := cqrshtmx.ParseCorrelationID("01JX...")
ctx = cqrshtmx.WithCorrelationID(ctx, cid)
```

Non-ULID header values are silently dropped by the middleware.

### 3. Passwordless Authentication (usermgmt)

All password-related APIs are removed:

- `RegisterRequest.Password` — removed (registration is email-only)
- `Service.Authenticate(email, password)` — removed (use WebAuthn `BeginLogin`/`FinishLogin`)
- `Service.ChangePassword` — removed
- `PasswordHash` field on User — removed

Authentication is exclusively via WebAuthn (passkeys) or OAuth2/OIDC.

### 4. Event-Sourced User Aggregate (usermgmt)

The `UserStore` interface is removed. User state is reconstructed from events via `UserReadModel` projection:

```go
// v2: UserStore interface
store := usermgmt.NewInMemoryUserStore()
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{UserStore: store})

// v3: Event-sourced (automatic via NewService)
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
// UserReadModel is created internally; pass ReadModelDB for SQL persistence
```

### 5. Error Handling: go-error-family

`cockroachdb/errors` is replaced by `go-error-family` (re-exported via `event/v3`):

```go
// v2
import "github.com/cockroachdb/errors"
errors.New("something")           // banned in v3
errors.Wrap(err, "context")       // → event.Wrapf(err, family, code, msg)

// v3
import "github.com/larsartmann/go-cqrs-lite/event/v3"
event.NewRejection("code", "message")
event.WrapTransient(err, "code", "message")
```

Use `branching-flow errorfamily .` to verify zero stdlib error constructors.

### 6. CSRF: nosurf Replaces gorilla/csrf

```go
// v2
import "github.com/gorilla/csrf"
csrf.Protect(secret)(handler)

// v3
import cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})(handler)
```

No secret management needed. Per-handler CSRF via `CSRFProtect(cfg)`.

### 7. Removed API Constructors

```go
// v2 (removed)
q := query.MustNew[MyQuery]("type")
c := command.MustNew("type")
id.MustParse[MyID]("...")

// v3
q, err := query.New[MyQuery]("type")
c := command.New("type") // no error variant for commands
id, err := id.Parse[MyID]("...")
```

### 8. Authz Domain Params Typed (v3.x)

Authorization methods now take `TenantID` instead of raw `string`:

```go
// v2
roles, _ := authz.RolesForUser(uid, "tenant-123")

// v3
roles, _ := authz.RolesForUser(uid, usermgmt.NewTenantID("tenant-123"))
```

### 9. Bot OwnerID Typed (v3.x)

```go
// v2
req := usermgmt.RegisterBotRequest{OwnerID: "user-123"}

// v3
req := usermgmt.RegisterBotRequest{OwnerID: usermgmt.NewUserID("user-123")}
```

### 10. Ephemeral Store Interfaces (v3.x)

All ephemeral stores now have interfaces, enabling Redis/SQL alternatives:

```go
// Override the default in-memory stores for multi-instance:
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    SessionStore:           myRedisSessionStore,
    WebAuthnSessionStore:   myRedisWebAuthnStore,
    VerificationTokenStore: myRedisVerificationStore,
    PendingTOTPStore:       myRedisTOTPStore,
    Lockout:                myRedisLockout,
})
```

## Migration Checklist

- [ ] Update all import paths to `/v3`
- [ ] Replace password auth with WebAuthn or OAuth2
- [ ] Remove `UserStore` usage; rely on event-sourced `UserReadModel`
- [ ] Replace `cockroachdb/errors` with `go-error-family` via `event/v3`
- [ ] Replace gorilla/csrf with nosurf-based `CSRFMiddleware`
- [ ] Update UserID usage to use `NewUserID`/`ParseUserID`
- [ ] Update CorrelationID to branded type
- [ ] Wrap authz domain params with `NewTenantID(...)`
- [ ] Run `branching-flow errorfamily .` — must report 0
- [ ] Run tests with `-race` flag

## Need Help?

- See `AGENTS.md` for architecture details
- See `docs/adr/` for decision records
- Check `examples/` for working consumer code
