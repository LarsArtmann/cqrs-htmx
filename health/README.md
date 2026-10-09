# cqrs-htmx/health

Optional bridge module wiring cqrs-htmx projection health into the
[go-health](https://github.com/larsartmann/go-health) ecosystem: a
`gohealth.Probe` with one named check per projection, plus an optional
[go-health-dashboard](https://github.com/LarsArtmann/go-health-dashboard) UI.

Consumers who do not import this module pay zero dependency cost.

## Which surface for which consumer

| You want | Surface |
| --- | --- |
| Kubernetes liveness/readiness/startup probes for a `*usermgmt.Service`, no DI container | `health.NewProbe(svc, ...)` → `Start` → `RegisterRoutes` |
| One probe checking DI services AND projections | `gohealth.New(injector, gohealth.WithHealthRecorder(health.Recorder(svc)))` |
| One probe also observing DI lifecycle (samber-do-auditlog) next to projections | `gohealth.New(injector, gohealth.WithHealthRecorder(health.RecorderChain(health.Recorder(svc), auditPlugin)))` |
| A human dashboard over any of the above | `health.NewDashboard(probe)` |
| Only readiness on `/health` (no go-health at all) | root's `cqrshtmx.ProjectionReadinessCheck` + `ReadinessHandler`, or the `setup` module's gated `/health` |

Every projection check carries the shared `ProjectionStatuses()` read time as
`duration_ns`, on every path (the recorder implements go-health's optional
`DetailedHealthRecorder`).

Note: `gohealth.WithHealthRecorder` passed to `NewProbe` has no effect — the
provider owns the batch. Compose extra recorders with `RecorderChain` and
build the probe yourself, as in the table above.

## Semantics

A projection is healthy when its worker is `live` or `stopped` (identical to
`cqrshtmx.ProjectionReadinessCheck`). Drain states (`idle`, `running`,
`backoff`, `draining`) report a transient error ("catching up"); `failed`
reports an infrastructure error carrying the projection's last error.

## Usage

```go
import (
    gohealth "github.com/larsartmann/go-health"
    "github.com/larsartmann/cqrs-htmx/health/v4"
)

probe, err := health.NewProbe(svc,
    gohealth.WithVersion("1.2.3"),
    gohealth.WithInstanceID("pod-7"), // distinguishes replicas on shared dashboards
    gohealth.WithCriticalServices("user-read-model", "casbin-projection"),
)
if err != nil {
    log.Fatal(err)
}
if err := probe.Start(ctx); err != nil {
    log.Fatal(err)
}
defer probe.Shutdown()

probe.RegisterRoutes(mux, gohealth.DefaultRoutes()) // /healthz /readyz /startupz
```

Optional dashboard (HTML + SSE + JSON by Accept header):

```go
import healthdashboard "github.com/larsartmann/go-health-dashboard"

dash := health.NewDashboard(probe, healthdashboard.WithTitle("My App"))
mux.Handle("GET /health/ui", dash.Handler())
```

## samber/do v2 users

`Recorder` merges projection checks with your injector's own service checks:

```go
probe := gohealth.New(injector, gohealth.WithHealthRecorder(health.Recorder(svc)))
```

`*usermgmt.Service` and `*usermgmt.EventSourcedSetup` satisfy
`health.ProjectionStatusProvider` directly.

Bounding each injector service's check: samber/do runs its own per-service
health checks with a timeout you set on the injector, not on the probe:

```go
injector := do.NewWithOpts(do.InjectorOpts{HealthCheckTimeout: 2 * time.Second})
```

Extra checks beyond projections (disk, memory, HTTP, database) come from
go-health's own batteries and compose as standalone named checks:

```go
import gohealthchecks "github.com/larsartmann/go-health/checks"

probe := gohealth.NewChecks(map[string]gohealth.CheckFunc{
    "disk": gohealthchecks.Disk("/var/lib/app", 1<<30),
}, gohealth.WithCriticalServices("disk"))
```

Fleet pattern: `gohealth.WithEvaluationHook` fires on every evaluated
response — wire it to [go-flightrecorder](https://github.com/LarsArtmann/go-flightrecorder)
to snapshot system metrics the moment health degrades, before anyone thinks
to look.
