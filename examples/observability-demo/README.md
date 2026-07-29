# observability-demo

Runnable proof of OTel tracing + Prometheus metrics wiring using go-cqrs-lite's `otel` and `prometheus` modules with cqrs-htmx.

## What it demonstrates

1. **OTel tracing** via `cqrsotel.Setup()` with a stdout exporter — pretty-printed spans to console
2. **Prometheus metrics** via `cqrsprom.Setup()` — `/metrics` endpoint in Prometheus format
3. **Dispatch middleware composition**: recovery, retry, tracing (`CommandTracing`), metrics (`CommandTypedMetrics`), logging — all wired onto the same `*command.Dispatcher` that cqrs-htmx uses

The key insight: `cqrsotel.NewMeter()` resolves from the global meter provider, so you must override it with the Prometheus provider after setup:

```go
otel.SetMeterProvider(promProvider.AsMeterProvider())
```

## How to run

```bash
cd examples/observability-demo
go run .
```

Then:

```bash
# Dispatch a command (traces emitted to stdout, metrics recorded)
curl -X POST http://localhost:8099/ping -d '{"msg":"hello"}'

# View Prometheus metrics
curl http://localhost:8099/metrics
```

## Tests

```bash
go test -race -count=1 ./...
```

Two tests verify: POST /ping returns 204, and /metrics returns Prometheus format with `cqrs_operation` metrics.

## See also

- [Leveraging go-cqrs-lite guide](../../docs/guides/leveraging-go-cqrs-lite.md) §2 — OTel & Prometheus section
