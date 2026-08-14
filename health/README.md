# cqrs-htmx/health

Optional bridge module wiring cqrs-htmx projection health into the
[go-health](https://github.com/larsartmann/go-health) ecosystem: a
`gohealth.Probe` with one named check per projection, plus an optional
[go-health-dashboard](https://github.com/larsartmann/go-health-dashboard) UI.

Consumers who do not import this module pay zero dependency cost.

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
    gohealth.WithCriticalServices("user-read-model", "casbin-projection"),
)
if err != nil {
    log.Fatal(err)
}
if err := probe.Start(ctx); err != nil {
    log.Fatal(err)
}
defer probe.Shutdown()

probe.RegisterRoutes(mux, gohealth.DefaultRoutes()) // /healthz /readyz /startedz
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
