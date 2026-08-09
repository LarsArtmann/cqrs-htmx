# Leveraging samber/do v2 — Dependency Injection for cqrs-htmx Consumers

> How to wire `cqrs-htmx` with [`github.com/samber/do/v2`](https://github.com/samber/do) as your application's dependency-injection container. Includes the composition-root pattern, lifecycle adapters for `usermgmt.Service.Close()`, named auth providers, test containers, and an anti-pattern checklist.

## Why this guide exists

`cqrs-htmx` is a **library/SDK**, not an application. It deliberately avoids imposing a DI container — the library principle is "never enforce defaults consumers might disagree with." Every example in `examples/` uses plain Go construction: create components, pass them to a `Config{}` struct, call `New()`.

This is correct for a library. But **applications** that consume cqrs-htmx often need a DI container for their own composition root — especially when the app wires many services (usermgmt, auth strategies, observability, database pools) with complex dependency graphs.

`samber/do` v2 is the dominant DI container in the LarsArtmann ecosystem. This guide shows how to use it well with cqrs-htmx, avoiding the six anti-patterns (DO-1 through DO-6) and leveraging samber/do's lifecycle hooks for automatic shutdown.

**Runnable reference:** `examples/samber-do-demo/`

## When to use a DI container with cqrs-htmx

| Situation                                          | Use samber/do? | Why                                                |
| -------------------------------------------------- | -------------- | -------------------------------------------------- |
| Simple app, <5 services                            | No             | Plain construction is clearer                      |
| Multiple auth providers (TOTP + WebAuthn + OAuth2) | Yes            | Named services prevent wire-up duplication         |
| Test containers with dependency overrides          | Yes            | `do.Override*` is cleaner than manual swap         |
| Long-running service with health checks            | Yes            | `do.Healthchecker*` interface for readiness probes |
| Library code (inside cqrs-htmx itself)             | **Never**      | Libraries must not impose a container              |

## cqrs-htmx types → samber/do patterns

| cqrs-htmx type                   | Lifetime       | samber/do function                      | Rationale                                                 |
| -------------------------------- | -------------- | --------------------------------------- | --------------------------------------------------------- |
| `AppConfig` (your config struct) | Eager          | `do.ProvideValue`                       | Must exist before any lazy provider resolves              |
| `*slog.Logger`                   | Eager or lazy  | `do.ProvideValue` or `do.Provide`       | Eager if you want it available immediately                |
| `*usermgmt.Service`              | Lazy singleton | `do.Provide`                            | Heavy to construct (event store, projections, goroutines) |
| `*cqrshtmx.App`                  | Lazy singleton | `do.Provide`                            | Depends on dispatchers                                    |
| `*cqrshtmx.Broadcaster`          | Lazy singleton | `do.Provide`                            | Lightweight, but only needed if SSE is used               |
| `usermgmt.TOTPProvider`          | Named lazy     | `do.ProvideNamed("auth.totp", ...)`     | Multiple auth strategies coexist                          |
| `usermgmt.WebAuthnProvider`      | Named lazy     | `do.ProvideNamed("auth.webauthn", ...)` | Same reason                                               |
| `usermgmt.OAuth2Provider`        | Named lazy     | `do.ProvideNamed("auth.oauth2", ...)`   | Same reason                                               |
| `*command.Dispatcher`            | Lazy singleton | `do.Provide`                            | Shared across all command handlers                        |
| `*query.Dispatcher`              | Lazy singleton | `do.Provide`                            | Shared across all query handlers                          |
| `*sql.DB` (read model)           | Eager          | `do.ProvideValue`                       | Connection pool must be alive even if no query ran yet    |

## Recipes

### 1. Composition root with cleanup function

The container is created once, and the cleanup function is deferred. This is the only type allowed to hold `do.Injector`.

```go
type Container struct {
    injector do.Injector
}

func NewContainer(cfg AppConfig) (*Container, func()) {
    injector := do.New()
    registerProviders(injector, cfg)
    return &Container{injector: injector}, func() {
        report := injector.Shutdown()
        // log report if needed
    }
}
```

```go
// main.go
container, cleanup := NewContainer(cfg)
defer cleanup()
```

### 2. Eager foundation values

Config, logger, and DB pools must exist before any lazy provider resolves them.

```go
do.ProvideValue(injector, cfg)
do.ProvideValue(injector, dbPool)      // *sql.DB
do.ProvideValue(injector, slog.Default())
```

### 3. Lazy singleton: usermgmt.Service

The Service is heavy (starts projections, eviction goroutines). Register it lazy so it's only built when first needed.

```go
do.Provide(injector, func(i do.Injector) (*usermgmt.Service, error) {
    totpProvider, err := do.InvokeNamed[usermgmt.TOTPProvider](i, "auth.totp")
    if err != nil {
        return nil, err
    }
    return usermgmt.NewService(usermgmt.ServiceConfig{
        AuditLog: usermgmt.NewAuditLog(),
        TOTP:     totpProvider,
    })
})
```

### 4. Named services: multiple auth providers

When multiple implementations of the same interface exist (WebAuthn, TOTP, OAuth2), register them by name.

```go
do.ProvideNamed(injector, "auth.totp", func(i do.Injector) (usermgmt.TOTPProvider, error) {
    cfg, _ := do.Invoke[AppConfig](i)
    return totp.New(totp.Config{Issuer: cfg.TOTPIssuer}), nil
})

do.ProvideNamed(injector, "auth.webauthn", func(i do.Injector) (usermgmt.WebAuthnProvider, error) {
    cfg, _ := do.Invoke[AppConfig](i)
    return webauthn.New(webauthn.Config{RPID: cfg.WebAuthnRPID}), nil
})
```

Resolve via typed accessor:

```go
func (c *Container) TOTPProvider() (usermgmt.TOTPProvider, error) {
    return do.InvokeNamed[usermgmt.TOTPProvider](c.injector, "auth.totp")
}
```

### 5. Lifecycle adapter: bridging Close() to Shutdown()

`*usermgmt.Service` has `Close() error`, not `Shutdown()`. samber/do only calls `Shutdown()` on types implementing its lifecycle interfaces. The adapter bridges this:

```go
type serviceLifecycle struct {
    svc *usermgmt.Service
}

var _ do.ShutdownerWithContextAndError = (*serviceLifecycle)(nil)

func (l *serviceLifecycle) Shutdown(_ context.Context) error {
    return l.svc.Close()
}
```

Register the lifecycle as a lazy provider that depends on the Service:

```go
do.Provide(injector, func(i do.Injector) (*serviceLifecycle, error) {
    svc, err := do.Invoke[*usermgmt.Service](i)
    if err != nil {
        return nil, err
    }
    return &serviceLifecycle{svc: svc}, nil
})
```

**Critical:** eagerly invoke the lifecycle in `NewContainer` so `injector.Shutdown()` always calls `Service.Close()`, even if nobody resolved the Service:

```go
func NewContainer(cfg AppConfig) (*Container, func()) {
    injector := do.New()
    registerProviders(injector, cfg)
    // Force-create the lifecycle wrapper so it's tracked for shutdown
    if _, err := do.Invoke[*serviceLifecycle](injector); err != nil {
        slog.Error("failed to initialize service lifecycle", "error", err)
    }
    return &Container{injector: injector}, func() { injector.Shutdown() }
}
```

### 6. do.Package for modular registration

Group related providers into a package for reuse:

```go
var AuthProviders = do.Package(
    do.LazyNamed("auth.totp", func(i do.Injector) (usermgmt.TOTPProvider, error) {
        // ...
    }),
    do.LazyNamed("auth.webauthn", func(i do.Injector) (usermgmt.WebAuthnProvider, error) {
        // ...
    }),
    do.LazyNamed("auth.oauth2", func(i do.Injector) (usermgmt.OAuth2Provider, error) {
        // ...
    }),
)

injector := do.New(AuthProviders)
```

### 7. Test container with overrides

`do.Override*` is safe in `_test.go` files (DO-3 rule). Create a test container that swaps production dependencies:

```go
func newTestContainer(t *testing.T) (*Container, func()) {
    t.Helper()
    container, cleanup := NewContainer(AppConfig{})

    // Override the TOTP provider with a stub
    do.OverrideNamed(container.injector, "auth.totp", func(_ do.Injector) (usermgmt.TOTPProvider, error) {
        return stubTOTP{}, nil
    })

    return container, cleanup
}
```

> **Gotcha:** use `do.OverrideNamed` (not `do.OverrideNamedValue`) when overriding a named service registered with an interface type. `OverrideNamedValue` uses the concrete type, which creates a mismatch.

### 8. Health checks via do.Healthchecker

Implement `do.HealthcheckerWithContext` on a wrapper to enable readiness probes:

```go
type serviceHealthcheck struct {
    svc *usermgmt.Service
}

var _ do.HealthcheckerWithContext = (*serviceHealthcheck)(nil)

func (h *serviceHealthcheck) HealthCheck(ctx context.Context) error {
    // Check projection health, DB connectivity, etc.
    return nil
}
```

samber/do's `injector.HealthCheck()` runs all registered healthchecks.

### 9. Scopes for multi-tenant isolation

```go
func TenantScope(root do.Injector, tenantID string) do.Injector {
    return root.Scope("tenant-" + tenantID)
}
```

Child scopes can resolve parent services; parent scopes cannot see child services.

## Anti-pattern checklist (cqrs-htmx specific)

| Rule | What to avoid                                                       | Fix                                                        |
| ---- | ------------------------------------------------------------------- | ---------------------------------------------------------- |
| DO-1 | `do.MustInvoke[*usermgmt.Service]` inside an HTTP handler           | Inject the Service into the handler struct at startup      |
| DO-2 | `do.New()` without `defer cleanup()`                                | Always return and defer the cleanup function               |
| DO-3 | `do.Override*` in production code (non-test)                        | Use aliasing or `ProvideNamed` instead                     |
| DO-4 | Package-level `var injector = do.New()`                             | Pass the container from `main()`                           |
| DO-5 | `do.Invoke` inside a request loop                                   | Resolve once before the loop                               |
| DO-6 | `serviceLifecycle.Shutdown()` calls `do.Invoke` for another service | Shutdown must be self-contained — close only `svc.Close()` |
| DO-7 | `do.Invoke` inside business-logic methods                           | Only accessors and constructors may invoke                 |
| DO-8 | Service struct stores `do.Injector` as a field                      | Only `Container` may hold the injector                     |

## Complete reference

See `examples/samber-do-demo/` for a runnable, tested example demonstrating all patterns above.
