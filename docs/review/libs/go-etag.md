# go-etag Utilization Report

> Audit date: 2026-11-22 (third refresh; prior: 2026-11-14) · Method: library-deep-dive (fleet series, [index](index.html)) · Series conventions: [FORMAT.md](FORMAT.md)
> Library: `github.com/larsartmann/go-etag` (five modules, two tag streams: the consumed `server`/`entitytag` submodules ride the nested train; `client`/`metrics` submodules and the deprecated root shim round out the repo) · Checkout: `/home/lars/projects/go-etag`

## Verdict at a glance

| Pin                                                 | Latest (verified 2026-11-22)                                                                                          | Prod import sites | Adoption                                           | Score                  |
| --------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- | ----------------- | -------------------------------------------------- | ---------------------- |
| `go-etag/server` v0.6.2 (root `go.mod:18`; entitytag v0.6.2 indirect `:48`) | **v0.6.2 = latest for the consumed submodules** (nested train v0.6.2; the v0.7.0/v0.7.1 root tags are the deprecated shim's own stream — re-verified 2026-11-22, see § Version currency) | 5 files           | **Used — conditional requests for immutable JSON** | **63/100 · at latest** |

## What the library is

Entity-tag plumbing in five modules: `entitytag` (ETag values/computation), `server` (conditional-request middleware + primitives like `ParseETag`/`MatchesIfNoneMatch`), `client` (RFC 9111 stored-response caching for Go HTTP clients), `metrics` (hit-ratio counters attached over the server hooks), and the repo-root module — a deprecated alias shim for `server`, removed in v1.0.0. Two tag streams: the nested train versions all four submodules together (v0.6.2), while the root shim tag runs one minor ahead by design until v1.0.0 deletes it.

## How cqrs-htmx uses it today

- `internal/hx/immutable.go:62,90` — immutable-JSON serving with FNV-64a `HashTag` (`:39`, defined at `internal/hx/response.go:75`; the library's own `defaultHashFunc` is unexported, server/config.go:111, so the hash half is necessarily local) + `etag.ParseETag`/`etag.MatchesIfNoneMatch` for `If-None-Match` → 304, under a 1-year immutable `Cache-Control` (`:87`). This underpins `EventCatalogHandler` ([eventcatalog_reexport.go:54](../../../eventcatalog_reexport.go)) and `OpenAPISpecHandler` ([options_openapi.go:37](../../../options_openapi.go)).
- `options_render.go:162-163` — per-render ETag on partial-render responses (`RenderPartialOrFull` honors `If-None-Match`; landed with the 2026-11-06 partial-render showcase).
- `projection_status_handler.go:8` — per-request ETag on projection health JSON.
- `sync_pull.go:16` — ETag on the ADR-0056 sync pull response.
- `examples/middleware-showcase` — the ETag middleware demo (all-8-middleware stack; direct pin at `examples/middleware-showcase/go.mod:7`).
- The consumer swept v0.6.1 → v0.6.2 on 2026-11-18, same-day with the upstream staircase (the library's own docs commit `9b19565` records the sweep).

## Capability inventory

| #   | Capability                                                         | Status             | Evidence / reason                                                                                                                                                   |
| --- | ------------------------------------------------------------------ | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Conditional-request primitives (`ParseETag`, `MatchesIfNoneMatch`) | 🟢 Fully leveraged | the internal/hx immutable cache path (server/entity_tag.go:38,52 at the tag)                                                                                        |
| 2   | Immutable-asset serving pattern (ETag + long cache + 304)          | 🟢 Fully leveraged | event catalog, OpenAPI, projection status, sync pull — now also spec-pinned end-to-end via `httpspec.ExpectNotModifiedDynamic` ([httpspec_etag_test.go:140](../../../httpspec_etag_test.go), B5 2026-11-15) |
| 3   | Middleware form (`server` middleware)                              | 🟡 Partially used  | examples only — correct for a library (never force response caching semantics on consumer routes); the hx layer composes primitives where the contract demands them |
| 4   | `entitytag` direct use                                             | ⚪ N/A             | arrives as a transitive pin of `server` (`go.mod:48`); no direct need has surfaced (hashes are FNV-64a over canonical bytes by design, mirrored locally because the library default is unexported) |
| 5   | `client` module (RFC 9111 stored-response caching for Go clients)  | ⚪ N/A             | cqrs-htmx ships servers — its revalidating consumers are browsers, and its Go-client tests assert 304 semantics directly (`httpspec.ExpectNotModifiedDynamic` + manual `If-None-Match`), so a stored-response cache would test the cache, not the server. (Module since 2026-09-23; missing from both prior editions' module maps — corrected 2026-11-22.) |
| 6   | `metrics` module (`Attach`/`Counters`/`HitRatio`/`Snapshot` over the server hooks, metrics/metrics.go:38,72,87) | 🔴 Not used | zero import sites and zero pins (verified 2026-11-22); hit-ratio counters would quantify the immutable-JSON 304 story — same observe-first posture as go-sse's drops-as-counters. Flip trigger below. (Module since 2026-09-23; corrected into the map 2026-11-22.) |

## Score derivation

6 rows − 2 N/A = **4 applicable**; 2 full ×1 + 1 partial ×0.5 + 1 not-used ×0 = **2.5 → 2.5/4 = 63/100** (band: partially leveraged by count). The drop from 83 is an **inventory correction, not a consumer regression**: rows 5–6 entered the map this session (both modules existed since 2026-09-23 but were missing from the 11-05/11-14 module maps), and the derivation moved with the inventory per FORMAT.md. The consumed core (rows 1–3) is unchanged and very high for its narrow role.

## Version currency

Pins `server` v0.6.2 + `entitytag` v0.6.2 = latest (verified `git ls-remote --tags origin`, 2026-11-22: the nested train — `entitytag/`, `server/`, `client/`, `metrics/` — is uniformly v0.6.2, tagged 2026-11-18). Nothing to do; ride the submodule tags.

**Root tags v0.7.0/v0.7.1 re-verified 2026-11-22 — the 2026-11-06 artifact verdict HOLDS, and is now upstream-documented.** v0.7.0 (89a6bf3, 2026-11-03) refreshed only the shim's dependency graph — the upstream CHANGELOG's own [0.7.0] entry: "Root-module-only release … The nested modules were not re-tagged (they stayed at v0.6.1)". v0.7.1 (24ae842, 2026-11-18) is the shim's `go mod tidy` against `server` v0.6.2 — [0.7.1]: "the deprecated shim rides the nested train and stays one minor ahead of it until v1.0.0 deletes it". Receipts: the root `go.mod` requires `go-etag/server v0.6.2`; the root's entire Go surface is the alias shim (`deprecated.go:9`) + `doc.go`; README.md:15 declares the root "Deprecated alias shim for the server package; removed in v1.0.0". Neither root tag releases anything the consumer imports.

The v0.6.1 → v0.6.2 delta (the pinned train): `server`/`metrics` pick up go-error-family v0.11.0 → v0.12.0 (nil-receiver-safe `(*Error).Unwrap` — a panic-class hardening inherited free); `entitytag`/`client` are internal-layout re-tags with zero public-API change (per-topic test suites, file splits under the 350-line gate). Library HEAD 39f9f37 (2026-11-18) sits at the staircase: zero `.go` drift past the train in any module dir (only `client/go.sum`/`metrics/go.sum` finalization + docs), and CHANGELOG [Unreleased] reads "Nothing yet."

## Should cqrs-htmx use it (more)?

No expansion needed for the conditional-request story — it is exactly as deep as the contracts require, now spec-pinned end-to-end. The one open (Minor) item is row 6 below; just ride the tag otherwise.

## Issues / open items

- **Minor — `metrics` module unadopted (row 6).** `metrics.Attach` over the existing `internal/hx` ETag configs would surface `NotModified`/`Generated`/hit-ratio counters for the immutable-JSON surface with ~zero plumbing. Severity Minor because drops/304s already surface as counts/logs (the DropStats precedent) and nothing scrapes metrics today. Flip trigger: a metrics scraper / OTel ask lands (cf. the go-sse guide's OTel recipe, `docs/guides/sse-and-datastar.md`) or an SLO on cache effectiveness. No owning TODO_LIST row exists (checked 2026-11-22); fold into any future OTel ask rather than filing a new row.

## How to verify

```bash
rg -n 'larsartmann/go-etag/(server|entitytag)' go.mod   # :18 server v0.6.2 direct · :48 entitytag v0.6.2 indirect
rg -l 'larsartmann/go-etag' --type go -g '!*_test.go' -g '!scripts/**' . | wc -l   # 5 prod import files
git -C ~/projects/go-etag ls-remote --tags origin 'refs/tags/server/*'    | grep -v '\^{}' | awk '{print $2}' | sort -V | tail -1   # refs/tags/server/v0.6.2 = latest
git -C ~/projects/go-etag ls-remote --tags origin 'refs/tags/entitytag/*' | grep -v '\^{}' | awk '{print $2}' | sort -V | tail -1   # refs/tags/entitytag/v0.6.2 = latest
git -C ~/projects/go-etag show v0.7.1:go.mod | grep 'server'   # "require github.com/larsartmann/go-etag/server v0.6.2" — the root tag is the shim's own stream
git -C ~/projects/go-etag show v0.7.1:README.md | sed -n '15p'  # root = "Deprecated alias shim for the server package; removed in v1.0.0"
git -C ~/projects/go-etag diff server/v0.6.2..HEAD --name-only -- server/ entitytag/ client/ metrics/   # client/go.sum · metrics/go.sum only — no .go drift past the train
```

## Prior reports

- 2026-11-14 (prior state of this file): 83/100 · at latest · module map corrected 4 → 6 rows this session (`client`/`metrics` existed since 2026-09-23 but were missing from both prior editions), so the derivation moved 83 → 63 with no consumer-side change; consumer swept v0.6.1 → v0.6.2 on the 2026-11-18 staircase; root v0.7.0/v0.7.1 re-verified shim-only — the 2026-11-06 artifact verdict now corroborated by upstream's own CHANGELOG entries.
- None earlier — this file's first edition was 2026-11-14 (the refresh that nearly dropped it; FORMAT.md cites that index-pass catch). Context in [docs/guides/event-catalog-guide.md](../../guides/event-catalog-guide.md) and ADR-0056.
