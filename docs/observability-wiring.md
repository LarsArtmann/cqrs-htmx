# Observability: OpenTelemetry & Prometheus Wiring

cqrs-htmx is intentionally **dependency-free** for observability — it never imports `go.opentelemetry.io/otel` or `prometheus/client_golang`. Instead, it provides **lifecycle hooks** (`BeforeDispatchHook` / `AfterDispatchHook`) and **middleware** that let consumers bolt on any observability stack.

## How the Hooks Work

```go
// app.go — the two hook types
type BeforeDispatchHook func(ctx context.Context, r *http.Request) context.Context
type AfterDispatchHook  func(ctx context.Context, r *http.Request, err error)
```

Hooks run **inside** the dispatch path, so they see the actual command/query type and dispatch result. HTTP middleware outside the dispatch can only see the method/path — it cannot distinguish a `CreateUser` command from an `UpdateUser` command.

Wire them via `Config`:

```go
app, _ := cqrshtmx.New(cqrshtmx.Config{
    Commands:       disp,
    Queries:        queryDisp,
    BeforeDispatch: myTracingHook,
    AfterDispatch:  myRecordingHook,
})
```

---

## OpenTelemetry (Tracing)

### Dependencies (consumer-side)

```bash
go get go.opentelemetry.io/otel \
       go.opentelemetry.io/otel/trace \
       go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
```

### Wiring

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

func OtelBeforeDispatch() cqrshtmx.BeforeDispatchHook {
    tr := otel.Tracer("cqrs-htmx")
    return func(ctx context.Context, _ *http.Request) context.Context {
        ctx, _ = tr.Start(ctx, "cqrs.dispatch")
        return ctx
    }
}

func OtelAfterDispatch() cqrshtmx.AfterDispatchHook {
    return func(ctx context.Context, _ *http.Request, err error) {
        span := trace.SpanFromContext(ctx)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        }
        span.End()
    }
}
```

### Usage

```go
app, _ := cqrshtmx.New(cqrshtmx.Config{
    Commands:       disp,
    BeforeDispatch: OtelBeforeDispatch(),
    AfterDispatch:  OtelAfterDispatch(),
})
```

Every command/query dispatch now produces a span. Downstream handlers and repositories can attach attributes via `trace.SpanFromContext(ctx)`.

> **Reference:** `example_otel_test.go` contains a self-contained example using stub types.

---

## Prometheus (Metrics)

### Dependencies (consumer-side)

```bash
go get github.com/prometheus/client_golang
```

### Wiring

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var dispatchCounter = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "cqrs_dispatch_total",
        Help: "Total number of CQRS dispatches",
    },
    []string{"type", "result"}, // type = command/query, result = success/error
)

var dispatchDuration = promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "cqrs_dispatch_duration_seconds",
        Help:    "Time spent dispatching",
        Buckets: prometheus.DefBuckets,
    },
    []string{"type"},
)

func PromBeforeDispatch() cqrshtmx.BeforeDispatchHook {
    return func(ctx context.Context, _ *http.Request) context.Context {
        return context.WithValue(ctx, dispatchStartKey{}, time.Now())
    }
}

func PromAfterDispatch() cqrshtmx.AfterDispatchHook {
    return func(ctx context.Context, _ *http.Request, err error) {
        start, _ := ctx.Value(dispatchStartKey{}).(time.Time)
        result := "success"
        if err != nil {
            result = "error"
        }
        dispatchCounter.WithLabelValues("command", result).Inc()
        dispatchDuration.WithLabelValues("command").Observe(time.Since(start).Seconds())
    }
}

type dispatchStartKey struct{}
```

### Expose `/metrics`

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

mux.Handle("/metrics", promhttp.Handler())
```

---

## Server-Timing (Built-in, No Dependency)

cqrs-htmx includes a W3C Server-Timing middleware for lightweight, browser-visible timing — no OTel or Prometheus needed. See ADR-0033.

```go
// Always-on
app, _ := cqrshtmx.New(cqrshtmx.Config{
    ServerTiming: func(r *http.Request) bool { return true },
})

// Debug-gated (recommended)
app, _ := cqrshtmx.New(cqrshtmx.Config{
    ServerTiming: func(r *http.Request) bool {
        return r.URL.Query().Get("debug") == "1"
    },
})
```

Inside handlers, record timing:

```go
defer cqrshtmx.MeasureServerTiming(ctx, "db")()

stop := cqrshtmx.MeasureServerTiming(ctx, "render")
// ... render work ...
stop()
```

The `Server-Timing` header appears in browser DevTools automatically.

---

## go-cqrs-lite Upstream Modules

For deeper integration, go-cqrs-lite provides dedicated observability modules that consumers can wire directly into their dispatch pipelines:

| Module                       | Purpose                                                     |
| ---------------------------- | ----------------------------------------------------------- |
| `go-cqrs-lite/otel/v3`       | OTel spans for event store, bus, and repository operations  |
| `go-cqrs-lite/middleware/v3` | Dispatch middleware (command/query) for tracing and metrics |
| `go-cqrs-lite/prometheus/v3` | Prometheus collectors for event store and bus metrics       |

These operate at the CQRS layer (below cqrs-htmx's HTTP hooks). Combine both layers for full-stack observability: cqrs-htmx hooks for HTTP→dispatch tracing, go-cqrs-lite modules for event store/bus internals.
