# go-etag Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-etag` (root + `server`/`entitytag` submodules, single tag stream) · Checkout: `/home/lars/projects/go-etag`

## Verdict at a glance

| Pin | Latest (verified 2026-11-05) | Prod import sites | Adoption | Score |
| --- | --- | --- | --- | --- |
| `go-etag/server` v0.6.1 (entitytag v0.6.1 indirect) | **v0.6.1 = latest for the consumed submodules** (see § Version currency — the v0.7.0 root tag is a deprecated shim, corrected 2026-11-06) | 4 files | **Used — conditional requests for immutable JSON** | **83/100 · at latest** |

## What the library is

Entity-tag plumbing: `entitytag` (ETag values/computation) and `server` (conditional-request middleware + primitives like `ParseETag`/`MatchesIfNoneMatch`). Single tag stream versions all three modules.

## How cqrs-htmx uses it today

- `internal/hx/immutable.go:62,90` — immutable-JSON serving with FNV-1a `HashTag` + `etag.ParseETag`/`etag.MatchesIfNoneMatch` for `If-None-Match` → 304. This underpins `EventCatalogHandler` (1-year cache) and `OpenAPISpecHandler`.
- `projection_status_handler.go:8` — per-request ETag on projection health JSON.
- `sync_pull.go:16` — ETag on the ADR-0056 sync pull response.
- `examples/middleware-showcase` — the ETag middleware demo (all-8-middleware stack).

## Capability inventory

| # | Capability | Status | Evidence / reason |
| --- | --- | --- | --- |
| 1 | Conditional-request primitives (`ParseETag`, `MatchesIfNoneMatch`) | 🟢 Fully leveraged | the internal/hx immutable cache path |
| 2 | Immutable-asset serving pattern (ETag + long cache + 304) | 🟢 Fully leveraged | event catalog, OpenAPI, projection status, sync pull |
| 3 | Middleware form (`server` middleware) | 🟡 Partially used | examples only — correct for a library (never force response caching semantics on consumer routes); the hx layer composes primitives where the contract demands them |
| 4 | `entitytag` direct use | ⚪ N/A | arrives as a transitive pin of `server`; no direct need has surfaced (hashes are FNV-1a over canonical bytes by design) |

## Score derivation

4 rows − 1 N/A = **3 applicable**; 2 full ×1 + 1 partial ×0.5 = **2.5 → 83/100** (band: very high for its narrow role).

## Version currency — CORRECTED 2026-11-06: no action item

The audit's original claim ("pinned v0.6.1 vs v0.7.0 — the fleet's only lag") was an **artifact**: the `v0.7.0` tag releases only the repo-ROOT module, which the upstream README declares a deprecated alias shim for the server package (removed in v1.0.0) — NOT the consumed submodules. Verified: no `server/v0.7.0` or `entitytag/v0.7.0` submodule tags exist (`go-etag/server@v0.6.1` + `go-etag/entitytag@v0.6.1` are the latest), and the mechanical treadmill gate (20 repos · 78 prefixes) reports worst lag **0**. A bump sweep pre-flight correctly refused the nonexistent `entitytag@v0.7.0` (proxy 404). Nothing to do — ride the submodule tags.

## Should cqrs-htmx use it (more)?

No expansion needed — the conditional-request story is exactly as deep as the contracts require. Just ride the tag.

## How to verify

```bash
rg -n 'go-etag' --type go -g '!scripts/**' . | grep -v _test
git -C ~/projects/go-etag log --oneline v0.6.1..v0.7.0 | wc -l        # 11, all chore/docs
git -C ~/projects/go-etag ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
nix run .#check-upstream-treadmill
```

## Prior reports

- None specific; context in [docs/guides/event-catalog-guide.md](../../guides/event-catalog-guide.md) and ADR-0056.
