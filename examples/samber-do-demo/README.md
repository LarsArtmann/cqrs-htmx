# samber/do v2 Integration Demo

This example demonstrates best-practice wiring of [cqrs-htmx](https://github.com/larsartmann/cqrs-htmx) with the [samber/do v2](https://github.com/samber/do) dependency-injection container.

## Why This Example Exists

cqrs-htmx is a **library**, not an application — it deliberately avoids imposing a DI container. Consumers choose their own composition strategy. This example shows how to use samber/do v2 as that composition layer.

## Patterns Demonstrated

| Pattern                        | File                                 | What it shows                                                       |
| ------------------------------ | ------------------------------------ | ------------------------------------------------------------------- |
| Composition root with cleanup  | `container.go` `NewContainer`        | `do.NewWithOpts` wrapped with returned cleanup function            |
| Eager foundation values        | `container.go` `ProvideValue`        | `AppConfig` registered eagerly                                      |
| Lazy singletons                | `container.go` `Provide`             | `*usermgmt.Service`, `*cqrshtmx.App`                               |
| Interface consumption          | `container.go` `do.As` + `InvokeAs`  | Concrete `*totp.Provider` consumed as `identitymodel.TOTPProvider`  |
| Lifecycle adapter              | `container.go` `serviceLifecycle`    | Third-party `Close()` adapted to `ShutdownerWithContextAndError`    |
| Native lifecycle services      | `container.go` probe + dashboard     | `*gohealth.Probe`/`*Dashboard` implement `do.Shutdowner` themselves |
| Eager start of the health surface | `container.go` `probe.Start`/`dash.Start` | Started at boot so the dashboard shows real state (never an empty-green page) |
| Typed accessors                | `container.go` `Container.Service()` | Centralized resolution instead of raw `do.Invoke`                   |
| Test container with overrides  | `container_test.go`                  | `do.OverrideNamed` on the CONCRETE service (deterministic)          |
| Singleton verification         | `container_test.go`                  | Asserts same instance on repeated invocations                       |
| Shutdown verification          | `container_test.go`                  | Cleanup function calls `Close()` via lifecycle adapter              |
| Health-truth regression tests  | `container_test.go`                  | Probe cache non-empty; `/healthz`//`readyz`//`startupz` serve real verdicts; dashboard renders the projection checks |

## Running

```bash
cd examples/samber-do-demo
GOEXPERIMENT=jsonv2 go run .
# Open http://localhost:8098/ (or: PORT=9090 go run .)
```

## Endpoints

| Path         | What it is                                                                       |
| ------------ | -------------------------------------------------------------------------------- |
| `/`          | Index page                                                                       |
| `/healthz`   | Liveness (Kubernetes) — dependency-blind, always 200 while the process serves    |
| `/readyz`    | Readiness — 503 while projections drain or a CRITICAL projection fails; warn (200) for non-critical failures |
| `/startupz`  | Startup latch — 503 until every critical check passed once, then 200 forever     |
| `/health-ui` | Live health dashboard (projection checks, SSE updates)                           |
| `/audit/`    | Live samber/do audit viewer (registrations, invocations, shutdowns, health checks) |
| `/command/hello` | Demo command dispatch through the DI-managed dispatcher                      |

Critical projections (readiness-gating): `user-read-model`, `casbin-projection` —
every other worker reports `warn` on failure without leaving the load-balancer
rotation.

## Running Tests

```bash
GOEXPERIMENT=jsonv2 go test ./... -count=1 -v
```

## Key Design Decisions

1. **Container is the only type holding `do.Injector`** — no service stores the injector (DO-8 rule).
2. **Lifecycle adapter wraps `*usermgmt.Service`** — because the library's `Close() error` doesn't match samber/do's `Shutdown()` interface. The adapter bridges this.
3. **Eagerly invoked AND started health surface** — `NewContainer` invokes the lifecycle wrappers and calls `probe.Start` + `dashboard.Start`, propagating every error. A lazily-provided health dashboard reports green precisely when it knows the least (samber-linter HW-4), and an unstarted probe is worse: `CachedResponse()` answers a zero-value pass with zero checks forever.
4. **Boot errors are fatal** — eager `do.Invoke`/`Start` failures return from `NewContainer` instead of logging-and-continuing; an app with broken lifecycle wiring must not serve.
5. **Interface consumption via `do.As` + `InvokeAs`** — the concrete `*totp.Provider` is registered once and consumed as `identitymodel.TOTPProvider`; `do.ProvideNamed` is the pattern for when multiple implementations must coexist.
6. **Test overrides target the CONCRETE service** — overriding an already-invoked `do.As` ALIAS name is nondeterministic in samber/do v2.1.0 (18/20 runs returned the original service in a minimal repro); overriding a plain lazy service is deterministic. Override the concrete type and assert through the alias.
7. **Mount conflicts are real** — go-health registers method-less `/healthz`-style patterns and the audit viewer serves its own prefix: the consuming mux needs a method-less `/` root (a `GET /` root conflicts with method-less specific patterns under Go 1.22+ ServeMux rules) and must NOT `StripPrefix` the viewer.
