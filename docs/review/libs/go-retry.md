# go-retry Adoption Report (NOT adopted — correctly)

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-retry` v0.9.0 · Checkout: `/home/lars/projects/go-retry`

## Verdict

| Direct usage       | Transitive                                                                        | Latest                       | Verdict                            |
| ------------------ | --------------------------------------------------------------------------------- | ---------------------------- | ---------------------------------- |
| **0 import sites** | yes — via `go-cqrs-lite/middleware` (`CommandRetry`/`DefaultRetryConfig` recipes) | v0.9.0 (verified 2026-11-05) | **Correctly not adopted directly** |

## What it is

A dependency-light retry loop with exponential backoff + additive jitter over go-error-family, explicitly positioned for CLIs/batch jobs/simple services. Its own README routes CQRS contexts to [`go-cqrs-lite/middleware/v4`](https://github.com/LarsArtmann/go-cqrs-lite) — the wrapped variant with message adapters, OTel spans, and dead-letter entries.

## Why cqrs-htmx should NOT import it directly

1. **The library's own guidance says so** — in a CQRS context the retry belongs at dispatch middleware, which cqrs-htmx already documents as the canonical seam (`ServiceConfig.CommandMiddleware`, [docs/guides/leveraging-go-cqrs-lite.md](../../guides/leveraging-go-cqrs-lite.md) §1).
2. **No hand-rolled retry exists to replace** — verified: the only backoff constants in prod are `projectionhost.WithBackoff` parameters (`systemadapter/projections.go:20-21,155`), i.e. go-cqrs-lite's own projection restart policy; `time.Sleep` appears only in a drain poll loop, not a retry.
3. **Library principle** — cqrs-htmx never imposes retry policy; consumers compose it via middleware factories.

## Flip conditions

A non-CQRS retry need inside the library itself (e.g. an HTTP client helper with retries) would flip this to "adopt" — none exists today.

## How to verify

```bash
rg -n 'larsartmann/go-retry' --type go -g '!scripts/**' . | wc -l          # 0
rg -n 'backoff|Backoff' -g '*.go' -g '!*_test.go' -g '!scripts/**' .       # only projectionhost.WithBackoff params
rg -n 'CommandRetry' docs/guides/leveraging-go-cqrs-lite.md | head -3
git -C ~/projects/go-retry ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- None; transitive context in the [go-cqrs-lite report](go-cqrs-lite.md).
