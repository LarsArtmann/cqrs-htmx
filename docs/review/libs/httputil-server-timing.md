# httputil/server_timing Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/httputil/server_timing` (submodule of [httputil](httputil.md)) · Checkout: `/home/lars/projects/httputil`

## Verdict at a glance

| Pin | Latest (verified 2026-11-05) | Prod import sites | Adoption | Score |
| --- | --- | --- | --- | --- |
| v1.0.3 (root direct; admin-demo direct) | v1.0.3 (= latest) | 2 files (re-export + example) | **Used — deprecated re-export on the v5 glide path** | **n/a — wind-down, correctly managed** |

## What the library is

The Server-Timing middleware suite (W3C Server Timing API: middleware + `MeasureServerTiming` spans emitting `Server-Timing` response headers), extracted from httputil into its own module so the root stays lighter.

## How cqrs-htmx uses it today

- Root `server_timing_reexport.go` re-exports the symbols as deprecated aliases (`cqrshtmx.ServerTimingMiddleware`/`MeasureServerTiming`-class) — v5 removes them; internal modules use **zero** deprecated aliases (verified).
- `examples/admin-demo` consumes the upstream module directly — the canonical direct-usage demo.

## Capability inventory

| # | Capability | Status | Evidence / reason |
| --- | --- | --- | --- |
| 1 | Server-Timing middleware + measurement spans | 🟢 Fully leveraged (as re-export) | alias delegates 1:1 upstream; example demonstrates direct use |
| 2 | Direct upstream adoption inside library modules | ⚪ N/A (by design) | library code doesn't emit Server-Timing by default (never enforce headers consumers might not want); consumers opt in via httputil |

## Score derivation

Not scored: this is a managed deprecation, not an adoption question. The glide path (re-export → v5 removal, example demonstrates upstream) is the correct end state and is already mechanized.

## Version currency

v1.0.3 pinned = v1.0.3 latest (verified `git ls-remote --tags origin`, 2026-11-05).

## Should cqrs-htmx use it (more)?

No — hold the course: keep the example direct-importing, delete the aliases with v5.

## How to verify

```bash
cat server_timing_reexport.go
rg -n 'httputil/server_timing' --type go -g '!scripts/**' . | grep -v _test
git -C ~/projects/httputil ls-remote --tags origin | grep 'server_timing' | sort -V | tail -2
```

## Prior reports

- Parent context: [httputil report](httputil.md).
