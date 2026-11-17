# httputil/server_timing Utilization Report

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/httputil/server_timing` (submodule of [httputil](httputil.md)) · Checkout: `/home/lars/projects/httputil`

## Verdict at a glance

| Pin                                     | Latest (verified 2026-11-14) | Prod import sites                    | Adoption                                                                                  | Score                                  |
| --------------------------------------- | ---------------------------- | ------------------------------------ | ----------------------------------------------------------------------------------------- | -------------------------------------- |
| v1.0.3 (root direct; admin-demo direct) | v1.0.3 (= latest)            | 3 files (re-export, app.go, example) | **Used — deprecated re-export on the v5 glide path; root App consumes upstream directly** | **n/a — wind-down, correctly managed** |

## What the library is

The Server-Timing middleware suite (W3C Server Timing API: middleware + `MeasureServerTiming` spans emitting `Server-Timing` response headers, plus `WrapServerTiming`/`ServerTimingMiddlewareWhen`), extracted from httputil into its own module so the root stays lighter.

## How cqrs-htmx uses it today

- **Root App wiring** (`app.go:17` direct import): `Config.ServerTiming` predicate (`app.go:93`; nil = disabled, zero overhead) wraps every `App.Command()`/`App.Query()` handler via `applyServerTiming` → `servertiming.WrapServerTiming` (`app.go:370-380`, wired at `:254,:280,:308,:335`). The root module imports upstream directly and uses **zero** of its own deprecated aliases (verified; the only mention is a doc comment at `app.go:84`). Present since 2026-07-30 (`eefb0b8a`) — the prior report's "2 files" undercounted; corrected to 3.
- Root `server_timing_reexport.go` re-exports 8 symbols as deprecated aliases (`ServerTiming` type, 6 vars, 1 const) — v5 removes them (wind-down class 1b, ADR-0062 `docs/adr/0062-direct-imports-only.md:80`; `doc.go:12` corroborates the v5 deletion).
- `examples/admin-demo` consumes the upstream module directly (`main.go:188` `servertiming.ServerTimingMiddlewareWhen`) — the canonical direct-usage demo.
- No Server-Timing wiring in `setup`/`recommended_middleware.go` (verified: zero hits outside the files above).

## Capability inventory

| #   | Capability                                      | Status                               | Evidence / reason                                                                                                                            |
| --- | ----------------------------------------------- | ------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Server-Timing middleware + measurement spans    | 🟢 Fully leveraged (upstream-direct) | root `Config.ServerTiming` + `WrapServerTiming` on every Command/Query handler; alias file delegates 1:1; example shows the middleware form  |
| 2   | Direct upstream adoption inside library modules | ⚪ N/A (by design)                   | library code doesn't emit Server-Timing by default (never enforce headers consumers might not want); consumers opt in via the root predicate |

## Score derivation

Not scored: this is a managed deprecation, not an adoption question. The glide path (re-export → v5 removal, root App and example both import upstream directly) is the correct end state and is already mechanized — the root's own code never touches its aliases, so the v5 cut stays a pure file delete.

## Version currency

v1.0.3 pinned everywhere (root `go.mod:23` direct, `examples/admin-demo/go.mod:15` direct, v1.0.3 indirect in all other modules; the lone v0.12.0 in `scripts/testdata/…poisoned/` is intentional test data) = v1.0.3 latest (verified `git ls-remote --tags origin` + `git for-each-ref refs/tags`, 2026-11-14). Zero commits touch `server_timing/` past the tag (`git log server_timing/v1.0.3..HEAD -- server_timing/` = 0).

## Should cqrs-htmx use it (more)?

No — hold the course: keep the root predicate + example direct-importing, delete the aliases with v5.

## How to verify

```bash
cat server_timing_reexport.go
rg -n 'ServerTimingMiddleware|MeasureServerTiming' --type go -g '!*_test.go' . | grep -v server_timing_reexport.go   # only doc comment app.go:84 + qualified upstream name in admin-demo
rg -n 'servertiming\.' app.go   # root direct wiring: WrapServerTiming + Config predicate
git -C ~/projects/httputil for-each-ref --sort=-v:refname refs/tags | grep server_timing | head -1   # server_timing/v1.0.3
git -C ~/projects/httputil log --oneline server_timing/v1.0.3..HEAD -- server_timing/ | wc -l   # expect 0
```

## Prior reports

- Parent context: [httputil report](httputil.md).
