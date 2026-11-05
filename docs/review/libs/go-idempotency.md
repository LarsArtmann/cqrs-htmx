# go-idempotency Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-idempotency` · Checkout: `/home/lars/projects/go-idempotency`

## Verdict at a glance

| Pin | Latest (verified 2026-11-05) | Prod import sites | Test import sites | Adoption | Score |
| --- | --- | --- | --- | --- | --- |
| v0.3.2 (root direct) | v0.3.2 (= latest) | 3 files (root alias layer, examples/admin-demo) | 2 files | **Used — fully delegated, winding down by design** | **88/100** |

## What the library is

A deliberately tiny command-idempotency kit: the `Store` interface, `MemoryStore` (TTL entries + background sweeper), `ErrDuplicate`, and their constructors/check methods. The whole non-test surface is five exported symbols.

## How cqrs-htmx uses it today

- Root `idempotency.go` re-exports the entire surface as **deprecated aliases** (`IdempotencyStore`, `MemoryIdempotencyStore`, `NewMemoryIdempotencyStore`, `ErrDuplicateCommand`) — v5 removes them; every alias delegates 1:1 to the upstream module.
- `examples/admin-demo/main.go` demonstrates direct upstream usage (memory store + `idempotency.ErrDuplicate` → 409 via `MapError`).
- `usermgmt/command_idempotency_test.go` proves the dispatch-level seam. The *production* idempotency story for commands routes through go-cqrs-lite's `middleware` idempotency factory on `ServiceConfig.CommandMiddleware` (documented in [docs/guides/leveraging-go-cqrs-lite.md](../../guides/leveraging-go-cqrs-lite.md) §1) — the right layer, since dedup belongs at dispatch, not HTTP decode.

## Capability inventory

| # | Capability | Status | Evidence / reason |
| --- | --- | --- | --- |
| 1 | `Store` contract | 🟢 Fully leveraged | aliased at root; consumers + middleware path consume the upstream type |
| 2 | `MemoryStore` + sweeper | 🟢 Fully leveraged | documented default (`NewMemoryIdempotencyStore` delegates; library principle: never force persistence) |
| 3 | `ErrDuplicate` classification | 🟢 Fully leveraged | maps to 409 Conflict through `MapError`; aliased as `ErrDuplicateCommand` |
| 4 | Durable store variants | ⚪ N/A | the library offers none (v0.3.x is memory-only) — durability arrives via go-cqrs-lite middleware idempotency over its own stores |
| 5 | Direct-adoption posture | 🟡 Partially used | integration is a deprecated alias layer scheduled for v5 removal — deliberate exit, not neglect; the example + guide keep the upstream-first story alive |

## Score derivation

5 rows − 1 N/A = **4 applicable**; 3 full ×1 + 1 partial ×0.5 = **3.5 → 88/100** (band: very high). A five-symbol library cannot hide gaps; there are none.

## Version currency

v0.3.2 pinned = v0.3.2 latest (verified `git ls-remote --tags origin`, 2026-11-05).

## Should cqrs-htmx use it (more)?

It already uses all of it. The only forward motion is the planned v5 alias removal — keep `examples/admin-demo` as the canonical direct-usage demo through that cutover.

## How to verify

```bash
cat idempotency.go   # every alias delegates 1:1 upstream
rg -n 'go-idempotency' --type go -g '!scripts/**' . | grep -v _test
git -C ~/projects/go-idempotency ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- None specific.
