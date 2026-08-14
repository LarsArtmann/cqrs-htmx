# cqrs-htmx/auditlog

Optional bridge module wiring
[samber-do-auditlog](https://github.com/larsartmann/samber-do-auditlog) into a
cqrs-htmx application's samber/do v2 container in one call: the DI audit
plugin plus the live HTML viewer with SSE updates.

Consumers who do not import this module pay zero dependency cost.

## Usage

```go
import (
    cqrsauditlog "github.com/larsartmann/cqrs-htmx/auditlog/v4"
    auditlog "github.com/larsartmann/samber-do-auditlog"
    "github.com/larsartmann/samber-do-auditlog/live"
    "github.com/samber/do/v2"
)

setup, err := cqrsauditlog.WithAuditLog(
    auditlog.Config{MaxEvents: 10_000},
    live.Config{Prefix: "/auditlog"},
)
if err != nil {
    log.Fatal(err)
}

injector := do.NewWithOpts(setup.Opts)
defer func() { _ = injector.Shutdown() }()

mux.Handle("/auditlog/", setup.Viewer) // live dashboard + JSON/SSE API
```

Every do lifecycle event (service invoked, shutdown, ...) is recorded by the
plugin and streamed live to connected viewers. `setup.Plugin` exposes
reports and exports (JSON/CSV/Mermaid/D2/...).

## Composition

The plugin satisfies go-health's `HealthRecorder` interface implicitly, so it
composes with the `cqrs-htmx/health` module when you want DI audit and
projection health in one probe.
