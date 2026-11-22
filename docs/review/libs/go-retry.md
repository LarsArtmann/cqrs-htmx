# go-retry Adoption Report (NOT adopted — correctly)

> Audit date: 2026-11-22 (prior: 2026-11-14) · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-retry` v0.9.0 · Checkout: `/home/lars/projects/go-retry`

## Verdict

| Direct usage       | Transitive                                                                                                 | Latest                       | Verdict                            |
| ------------------ | ---------------------------------------------------------------------------------------------------------- | ---------------------------- | ---------------------------------- |
| **0 import sites** | yes — via `go-cqrs-lite/middleware` (`CommandRetry`/`DefaultRetryConfig` recipes; direct v0.9.0 pin there, middleware/v4/go.mod:18) | v0.9.0 (verified 2026-11-22) | **Correctly not adopted directly** |

## What it is

A dependency-light retry loop with exponential backoff + additive jitter over go-error-family, explicitly positioned for CLIs/batch jobs/simple services. Its own README routes CQRS contexts to [`go-cqrs-lite/middleware/v4`](https://github.com/LarsArtmann/go-cqrs-lite) — the wrapped variant with message adapters, OTel spans, and dead-letter entries.

## Why cqrs-htmx should NOT import it directly

1. **The library's own guidance says so** — in a CQRS context the retry belongs at dispatch middleware, which cqrs-htmx already documents as the canonical seam (`ServiceConfig.CommandMiddleware`, [docs/guides/leveraging-go-cqrs-lite.md](../../guides/leveraging-go-cqrs-lite.md) §1).
2. **No hand-rolled retry exists to replace** — verified 2026-11-22: the only backoff constants in prod are `projectionhost.WithBackoff` parameters (`systemadapter/projections.go:20-21,155`, `systemadapter/domain_config.go:91`), i.e. go-cqrs-lite's own projection restart policy; the other `backoff` hits are `ProjectionStatusBackoff` status strings and comments (projection_readiness.go:20-21,37), not retry logic.
3. **Library principle** — cqrs-htmx never imposes retry policy; consumers compose it via middleware factories.

Upstream check (2026-11-22): go-retry's `[Unreleased]` is a CI-lint-stack bump only (golangci action v2.15 + the errorfamily wrapcheck exemption) — no API or behavioral change that alters the verdict.

## Flip conditions

A non-CQRS retry need inside the library itself (e.g. an HTTP client helper with retries) would flip this to "adopt" — none exists today.

## How to verify

```bash
rg -n 'larsartmann/go-retry' --type go -g '!scripts/**' . | wc -l          # 0
rg -n 'backoff|Backoff' -g '*.go' -g '!*_test.go' -g '!scripts/**' . | rg -v 'WithBackoff|ProjectionStatusBackoff|// '   # 4 lines, no retry logic: backoffMin/Max consts feeding WithBackoff (projections.go:20-21) + projection worker-status literals (usermgmt/es_projection_setup.go:264, dashboardui/core/overview.go:92)
rg -n 'CommandRetry' docs/guides/leveraging-go-cqrs-lite.md | head -3      # :15,:40,:60 — the canonical dispatch-middleware recipe
rg -n 'go-retry' ~/projects/go-cqrs-lite/middleware/v4/go.mod             # :18 — v0.9.0, the transitive pin
git -C ~/projects/go-retry ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1   # v0.9.0
```

## Prior reports

- None prior; 2026-11-05 report (superseded by this refresh) — same verdict. Transitive context in the [go-cqrs-lite report](go-cqrs-lite.md).
- 2026-11-14 (prior state of this file): not-adopted verdict, no score (not-adopted convention) · delta: verdict unchanged; receipts re-derived (guide hits :15/:40/:60, cqrs-lite pin middleware/v4/go.mod:18, WithBackoff trio confirmed incl. domain_config.go:91); upstream `[Unreleased]` CI-only.
