# go-idempotency Utilization Report

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-idempotency` · Checkout: `/home/lars/projects/go-idempotency`

## Verdict at a glance

| Pin                  | Latest (verified 2026-11-14) | Prod import sites                               | Test import sites                                                                             | Adoption                                           | Score      |
| -------------------- | ---------------------------- | ----------------------------------------------- | --------------------------------------------------------------------------------------------- | -------------------------------------------------- | ---------- |
| v0.3.2 (root direct) | v0.3.2 (= latest)            | 4 files (root alias layer, examples/admin-demo, `usermgmt/command_idempotency.go` + `service_core.go` — the K2 default guard, 2026-11-19) | 2 direct (`usermgmt/command_idempotency_test.go`, `examples/admin-demo/ack_idempotency_test.go`) + 1 alias-layer (root `idempotency_test.go`) | **Used — now load-bearing by default, not winding down** | **88/100** |

> 2026-11-19 refresh (K2): the counts and posture above were trued after the default-mount wave — the utilization audit was 2026-11-14; the K2 receipt rows below are dated and do not re-derive the score.

## What the library is

A deliberately tiny command-idempotency kit. The **root package** is five exported symbols (`Store`, `MemoryStore`, `NewMemoryStore`, `ErrDuplicate`, `ErrInvalidTTL`); the single module also ships two subpackages present since ≤ v0.3.0 (corrected this audit — the 2026-11-05 report's "five symbols" was root-only): `middleware/` (`Command`, `NewCommand`, `HTTP` adapters for generic Go) and `contract/` (`RunTests`/`RunTestsStrict`/`RunTestsContextAware` for Store implementors).

## How cqrs-htmx uses it today

- Root `idempotency.go` re-exports 4 symbols as **deprecated aliases** (`IdempotencyStore:18`, `MemoryIdempotencyStore:27`, `ErrDuplicateCommand:33`, `NewMemoryIdempotencyStore:42`) — v5 removes them (doc.go:12); every alias delegates 1:1 upstream.
- `examples/admin-demo/main.go:39,94,338` demonstrates direct upstream usage (memory store + `idempotency.Store`-backed `ackMiddleware` → 409 on replay).
- `usermgmt/command_idempotency_test.go:23-48` proves the dispatch seam: `middleware.CommandIdempotency(store, …)` short-circuits a replayed command with `idempotency.ErrDuplicate` before the handler runs.
- The _production_ idempotency story routes through go-cqrs-lite's `middleware/v4` factory on `ServiceConfig.CommandMiddleware` (`setup/config.go:517-523`, `usermgmt/service_core.go:195-204`; guide [§1](../../guides/leveraging-go-cqrs-lite.md) "Dispatch middleware", store decision at line 113) — the right layer, since dedup belongs at dispatch, not HTTP decode.
- **`usermgmt/command_idempotency.go` (K2, 2026-11-19) — the guard is now a PRODUCT DEFAULT:** `NewService` mounts a replay guard by default (`service_factory.go`, directly inside the audit chain), store injectable via `ServiceConfig.IdempotencyStore` (nil = the upstream memory default, dev/test class; `service_core.go` imports the type), key read from `usermgmt.IdempotencyMetadataKey` metadata scoped by the authenticated actor, duplicate → Conflict 409 translation built in, opt-out `ServiceConfig.DisableCommandIdempotency`. The guard owns its claim — a self-contained `CheckAndRecord` mirror — so an `ErrDuplicate` from a consumer-mounted idempotency middleware passes through untouched (four new tests + the pre-existing opt-in sentinel test pin it).

**Relationship with go-cqrs-lite (re-verified 2026-11-14):** the two libraries are complementary, not competing. go-cqrs-lite deleted its in-repo `idempotency/v4` shim in the ADR-0128 extraction (per cqrs-htmx `go.work` replace note), making **go-idempotency the Store contract behind the CQRS factory**: `go-cqrs-lite/middleware/v4` (train now **v4.7.2**, not v4.4.0) takes `store idempotency.Store` with a `command.Command`-native adapter + keyExtractor defaulting to `cmd.ID()` (`middleware/v4/idempotency.go:57-66`), and its durable option is the `idempotency/sqlstore/v4` submodule (no `kvstore` submodule exists). Both go-cqrs-lite modules pin go-idempotency **v0.3.2** (`middleware/v4/go.mod:17`, `idempotency/sqlstore/v4/go.mod:10`) — same version as cqrs-htmx's direct pin, no train skew.

## Capability inventory

| #   | Capability                    | Status             | Evidence / reason                                                                                                                                        |
| --- | ----------------------------- | ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `Store` contract              | 🟢 Fully leveraged | aliased at root; go-cqrs-lite `CommandIdempotency` consumes the upstream type; admin-demo `ackMiddleware` takes `idempotency.Store`                      |
| 2   | `MemoryStore` + sweeper       | 🟢 Fully leveraged | documented default (`NewMemoryIdempotencyStore` delegates; dev/test-only posture upstream, per guide line 113)                                           |
| 3   | `ErrDuplicate` classification | 🟢 Fully leveraged | maps to 409 Conflict in admin-demo (`MapError`); aliased as `ErrDuplicateCommand`; seam test asserts `errors.Is(…, idempotency.ErrDuplicate)`            |
| 4   | Durable store variants        | ⚪ N/A             | root library offers none (memory-only); durability arrives via go-cqrs-lite `idempotency/sqlstore/v4` implementing this `Store`                          |
| 5   | Direct-adoption posture       | 🟡 Partially used  | integration is a deprecated alias layer scheduled for v5 removal — deliberate exit, not neglect; the example + guide keep the upstream-first story alive |
| 6   | `middleware/` subpackage      | ⚪ N/A             | superseded here — go-cqrs-lite's `CommandIdempotency` wraps the same Store with a native `command.Command` adapter; cqrs-htmx's seam is that factory     |
| 7   | `contract/` subpackage        | ⚪ N/A             | store-implementor test kit; becomes applicable only when a custom (SQL) `Store` lands per the guide's production recommendation                          |

## Score derivation

7 rows − 3 N/A = **4 applicable**; 3 full ×1 + 1 partial ×0.5 = **3.5 → 88/100** (band: very high). Unchanged from 2026-11-05, but on a corrected 7-row inventory (rows 6-7 added; the old report undercounted the module surface).

## Version currency

v0.3.2 pinned = v0.3.2 latest (verified `git for-each-ref` + `ls-remote --tags origin`, 2026-11-14). v0.3.1..v0.3.2 (16 commits) was release-hardening bookkeeping — changelog/CI/docs + auto-commits, no exported-API change. Master sits 5 commits past the tag, all docs/style/test-split/chore.

## Should cqrs-htmx use it (more)?

- Keep `examples/admin-demo` as the canonical direct-usage demo through the v5 alias-removal cutover.
- **M2M W0 gap (2026-11-06):** bots now authenticate via `NewAPITokenMiddleware` (`usermgmt/api_token_middleware.go:45-73`) and dispatch attributed to `bot:<id>` — but **no idempotency key is minted or extracted anywhere on the bot path**, and `CommandIdempotency` is absent from the default factory list (`setup/config.go:519-520`). M2M clients are exactly the retry-happy consumers dedup protects; the cheap fix is documentation — steer bot-route consumers to wire `CommandIdempotency` with a metadata/`Idempotency-Key` keyExtractor (the guide's line-113 recipe). **RESOLVED 2026-11-15 (B2):** the documentation half shipped — `docs/guides/leveraging-go-cqrs-lite.md` § "Client-driven idempotency for token-authenticated bot traffic" carries the compile-verified three-piece recipe (decode the header into `WithCustomMetadata`, extract via the `MetadataCarrier` keyExtractor, translate `ErrDuplicate` to a Conflict-family 409), TTL + key-scoping guidance, and the demo wiring: `examples/admin-demo`'s `ackMiddleware` now honors `Idempotency-Key` as the bot twin of `X-Command-Id` (`ack_idempotency_test.go` pins replay-409 + no-header pass-through). ~~Defaulting it on for token dispatch stays owner-gated (plan K2).~~ **Executed 2026-11-19 (K2, owner GO):** the default mount landed in `usermgmt` — see the K2 bullet under "How cqrs-htmx uses it today"; the remaining owner question is the multi-instance follow-up (reference SQL/Redis `idempotency.Store` guide example vs the seam + docs suffice).
- The ADR-0056 sync push path already anticipates this: each envelope's `commandId` is stamped as `X-Command-Id` precisely so consumer-wired idempotency middleware dedups replays (`sync_push.go:157-162`) — same consumer-wires-it posture, nothing for this library to change.

## How to verify

```bash
cat idempotency.go   # every alias delegates 1:1 upstream
rg -n 'go-idempotency' --type go -g '!scripts/**' . | grep -v _test
git -C ~/projects/go-idempotency for-each-ref --sort=-v:refname --format='%(refname:short)' refs/tags | head -1
git -C ~/projects/go-cqrs-lite/middleware/v4 show HEAD:go.mod | grep go-idempotency   # v0.3.2, same pin
```

## Prior reports

- 2026-11-05 (v0.3.2, score 88) — superseded by this refresh: module-surface undercount fixed, go-cqrs-lite train corrected v4.4.0→v4.7.2, M2M W0 gap added.
