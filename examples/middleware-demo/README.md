# middleware-demo

Runnable proof that `go-cqrs-lite/middleware/v4` composes with cqrs-htmx's command dispatcher with **zero glue**.

## What it demonstrates

cqrs-htmx's `Config.Commands` accepts a `*command.Dispatcher` — the exact type go-cqrs-lite's middleware targets. Call `dispatcher.Use(...)` before building the `App`, and 27 middleware factories (recovery, retry, circuit breaker, logging, tracing, metrics, validation, idempotency) compose instantly.

This example wires 4 middleware factories:

1. `CommandRecovery()` — converts panics to errors (outermost)
2. `CommandRetry(DefaultRetryConfig())` — exponential backoff on `errorfamily.IsRetryable`
3. `CommandCircuitBreaker(DefaultCircuitBreakerConfig())` — failsafe-go breaker
4. `CommandLogging(logger)` — per-attempt log lines (innermost)

The command handler (`flakyService.ping`) fails transiently on the first 2 calls, then succeeds. The retry middleware makes both HTTP requests return **204**.

## How to run

```bash
cd examples/middleware-demo
go run .
```

Then POST to the server:

```bash
curl -X POST http://localhost:8098/ping -d '{"msg":"hello"}'
```

First request: retries twice (376ms), returns 204. Second request: succeeds immediately (0ms).

## Tests

```bash
go test -race -count=1 ./...
```

Three tests verify the retry→204 behavior programmatically using `httptest.NewServer`.

## See also

- [Leveraging go-cqrs-lite guide](../../docs/guides/leveraging-go-cqrs-lite.md) §1 — the full middleware recipe and ordering rules
