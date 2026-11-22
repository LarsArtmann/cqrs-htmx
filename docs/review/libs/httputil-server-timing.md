# httputil/server_timing Utilization Report

> Audit date: 2026-11-22 (third refresh; prior: 2026-11-14) · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/httputil/server_timing` (submodule of [httputil](httputil.md)) · Checkout: `/home/lars/projects/httputil`

## Verdict at a glance

| Pin                                     | Latest (verified 2026-11-22) | Prod import sites                    | Adoption                                                                                  | Score                                  |
| --------------------------------------- | ---------------------------- | ------------------------------------ | ----------------------------------------------------------------------------------------- | -------------------------------------- |
| v1.0.3 (root direct; admin-demo direct) | v1.0.3 (= latest)            | 3 files (re-export, app.go, example) | **Used — deprecated re-export on the v5 glide path; root App consumes upstream directly** | **n/a — wind-down, correctly managed** |

## What the library is

The Server-Timing middleware suite (W3C Server Timing API: middleware + `MeasureServerTiming` spans emitting `Server-Timing` response headers, plus `WrapServerTiming`/`ServerTimingMiddlewareWhen`), extracted from httputil into its own module so the root stays lighter.

## How cqrs-htmx uses it today

- **Root App wiring** (`app.go:17` direct import): `Config.ServerTiming` predicate (`app.go:93`; nil = disabled, zero overhead) wraps every `App.Command()`/`App.Query()` handler via `applyServerTiming` → `servertiming.WrapServerTiming` (`app.go:370-380`, wired at `:254,:280,:308,:335`; all receipts re-verified 2026-11-22 — unchanged through the 412-commit drift window). The root module imports upstream directly and uses **zero** of its own deprecated aliases (re-verified 2026-11-22; the only mention is a doc comment at `app.go:84`). Present since 2026-07-30 (`eefb0b8a`).
- Root `server_timing_reexport.go` re-exports 8 symbols as deprecated aliases (`ServerTiming` type at `:16`, 6 middleware/context vars at `:18-47`, 1 const mirror at `:48`) — v5 removes them (wind-down class 1b, ADR-0062 `docs/adr/0062-direct-imports-only.md:80`; `doc.go:12` corroborates the v5 deletion).
- `examples/admin-demo` consumes the upstream module directly (`main.go:242` `servertiming.ServerTimingMiddlewareWhen`, import at `:45`; moved from `main.go:188` in the 11-14 file during the admin-demo QUERY-ladder/security rework) — the canonical direct-usage demo.
- No Server-Timing wiring in `setup`/`recommended_middleware.go` (re-verified 2026-11-22: zero hits outside the files above).

## Capability inventory

| #   | Capability                                      | Status                               | Evidence / reason                                                                                                                            |
| --- | ----------------------------------------------- | ------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Server-Timing middleware + measurement spans    | 🟢 Fully leveraged (upstream-direct) | root `Config.ServerTiming` + `WrapServerTiming` on every Command/Query handler; alias file delegates 1:1; example shows the middleware form  |
| 2   | Direct upstream adoption inside library modules | ⚪ N/A (by design)                   | library code doesn't emit Server-Timing by default (never enforce headers consumers might not want); consumers opt in via the root predicate |

## Score derivation

Not scored: this is a managed deprecation, not an adoption question. The glide path (re-export → v5 removal, root App and example both import upstream directly) is the correct end state and is already mechanized — the root's own code never touches its aliases (re-verified 2026-11-22), so the v5 cut stays a pure file delete.

## Version currency

v1.0.3 pinned everywhere (root `go.mod:23` direct, `examples/admin-demo/go.mod:16` direct, v1.0.3 indirect in all other modules; the lone v0.12.0 in `scripts/testdata/…poisoned/` is intentional test data) = v1.0.3 latest (verified `git ls-remote --tags origin` + `git for-each-ref refs/tags`, 2026-11-22; the remote's pre-split `server_timing/v0.9.x–v0.12.0` tags are legacy). Zero commits touch `server_timing/` past the tag (`git log server_timing/v1.0.3..HEAD -- server_timing/` = 0, re-verified 2026-11-22).

## Should cqrs-htmx use it (more)?

No — hold the course: keep the root predicate + example direct-importing, delete the aliases with v5.

## How to verify

```bash
cat server_timing_reexport.go                                  # 8 deprecated aliases (type :16, 6 vars :18-47, const mirror :48), each delegating 1:1 upstream
rg -n 'ServerTimingMiddleware|MeasureServerTiming' --type go -g '!*_test.go' . | grep -v server_timing_reexport.go   # exactly 2 hits: doc comment app.go:84 + qualified upstream call examples/admin-demo/main.go:242
rg -n 'servertiming\.' app.go                                  # :380 WrapServerTiming inside applyServerTiming (the :17 import line has no dot)
git -C ~/projects/httputil for-each-ref --sort=-v:refname refs/tags | grep server_timing | head -1   # server_timing/v1.0.3 (local)
git -C ~/projects/httputil log --oneline server_timing/v1.0.3..HEAD -- server_timing/ | wc -l        # 0 — no submodule drift past the tag
git -C ~/projects/httputil ls-remote --tags origin 'refs/tags/server_timing/*' | grep -v '\^{}' | awk '{print $2}' | sort -t/ -k3 -V | tail -1   # refs/tags/server_timing/v1.0.3 (remote = local; older v0.x entries are pre-split legacy — sort the ref column, not the hash-first line)
```

## Prior reports

- 2026-11-14 (prior state of this file): n/a (wind-down) · no drift across 412 consumer commits — v1.0.3 still latest, alias-use still 0, glide path intact; receipts re-pinned only (admin-demo `ServerTimingMiddlewareWhen` main.go:188 → :242, admin-demo pin go.mod:15 → :16).
- Parent context: [httputil report](httputil.md).
