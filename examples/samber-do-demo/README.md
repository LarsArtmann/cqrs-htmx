# samber/do v2 Integration Demo

This example demonstrates best-practice wiring of [cqrs-htmx](https://github.com/larsartmann/cqrs-htmx) with the [samber/do v2](https://github.com/samber/do) dependency-injection container.

## Why This Example Exists

cqrs-htmx is a **library**, not an application — it deliberately avoids imposing a DI container. Consumers choose their own composition strategy. This example shows how to use samber/do v2 as that composition layer.

## Patterns Demonstrated

| Pattern | File | What it shows |
|---------|------|---------------|
| Composition root with cleanup | `container.go` `NewContainer` | `do.New()` wrapped with returned cleanup function |
| Eager foundation values | `container.go` `ProvideValue` | `AppConfig` registered eagerly |
| Lazy singletons | `container.go` `Provide` | `*usermgmt.Service`, `*cqrshtmx.App` |
| Named services | `container.go` `ProvideNamed` | TOTP provider registered as `"auth.totp"` |
| Lifecycle adapter | `container.go` `serviceLifecycle` | Third-party `Close()` adapted to `ShutdownerWithContextAndError` |
| Typed accessors | `container.go` `Container.Service()` | Centralized resolution instead of raw `do.Invoke` |
| Test container with overrides | `container_test.go` | `do.OverrideNamed` to swap TOTP for a stub |
| Singleton verification | `container_test.go` | Asserts same instance on repeated invocations |
| Shutdown verification | `container_test.go` | Cleanup function calls `Close()` via lifecycle adapter |

## Running

```bash
cd examples/samber-do-demo
GOEXPERIMENT=jsonv2 go run .
# Open http://localhost:8098/
```

## Running Tests

```bash
GOEXPERIMENT=jsonv2 go test ./... -count=1 -v
```

## Key Design Decisions

1. **Container is the only type holding `do.Injector`** — no service stores the injector (DO-8 rule).
2. **Lifecycle adapter wraps `*usermgmt.Service`** — because the library's `Close() error` doesn't match samber/do's `Shutdown()` interface. The adapter bridges this.
3. **Eagerly invoked lifecycle** — `NewContainer` invokes `*serviceLifecycle` to ensure `injector.Shutdown()` always calls `Service.Close()`, even if nobody resolved the Service.
4. **Named services for auth providers** — `"auth.totp"` demonstrates the pattern for when multiple implementations of the same interface coexist (WebAuthn, TOTP, OAuth2).
5. **`OverrideNamed` (not `OverrideNamedValue`) in tests** — preserves the interface type for correct resolution.
