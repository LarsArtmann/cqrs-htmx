# go-idempotency Utilization Report

> Audit date: 2026-11-22 (prior: 2026-11-14, K2-trued) · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-idempotency` · Checkout: `/home/lars/projects/go-idempotency`

## Verdict at a glance

| Pin                  | Latest (verified 2026-11-22) | Prod import sites                                                                                                      | Test import sites                                                                                                              | Adoption                          | Score      |
| -------------------- | ---------------------------- | ---------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | --------------------------------- | ---------- |
| v0.6.0 (root direct) | v0.6.0 (= latest)            | 4 files (root alias layer `idempotency.go`, `usermgmt/command_idempotency.go` + `service_core.go` — the K2 default guard, `examples/admin-demo/main.go`) | 2 direct (`usermgmt/command_idempotency_test.go`, `examples/admin-demo/ack_idempotency_test.go`) + 1 alias-layer (root `idempotency_test.go`) | **Used — load-bearing by default** | **80/100** |

> Version jump since the last audit: the pin moved v0.3.2 → **v0.6.0** (three upstream releases 2026-11-17: v0.4.0 `Forever` permanent claims, v0.5.0 the `idemtest` consumer test kit as a nested module, v0.6.0 injectable `WithClock`) — inventory below extended accordingly.

## What the library is

A deliberately tiny command-idempotency kit. The **root package** is the `Store`/`MemoryStore`/`NewMemoryStore`/`ErrDuplicate`/`ErrInvalidTTL` core, now grown with `Forever` permanent claims (v0.4.0) and `WithClock` store options (v0.6.0, clock.go:29); the module also ships `middleware/` (`Command`, `NewCommand`, `HTTP` adapters for generic Go), `contract/` (`RunTests`/`RunTestsStrict`/`RunTestsContextAware` for Store implementors), and — since v0.5.0 — the **`idemtest/` nested module** (ADR-007): a contract-certified in-memory test store, failure injection (`Fail`), a store spy, execution counters, and dispatch drills (`RequireExactlyOnce`, `RequireOneWinner`, `RequireFailClosed`, `RequireAllowsAfterExpiry` + the pinned-clock `RequireAllowsAfterExpiryPinned` staged in `[Unreleased]`, already tagged `idemtest/v0.1.1`) plus in-process HTTP drills.

## How cqrs-htmx uses it today

- Root `idempotency.go` re-exports 4 symbols as **deprecated aliases** (`IdempotencyStore:18`, `MemoryIdempotencyStore:27`, `ErrDuplicateCommand:33`, `NewMemoryIdempotencyStore:42`) — v5 removes them (doc.go:12); every alias delegates 1:1 upstream.
- `examples/admin-demo/main.go:42,118,392,400-405` demonstrates direct upstream usage (memory store + `idempotency.Store`-backed `ackMiddleware` honoring BOTH `X-Command-Id` and `Idempotency-Key` → 409 on replay; `ack_idempotency_test.go` pins replay-409 + no-header pass-through — the B2 bot-twin receipt).
- `usermgmt/command_idempotency_test.go:29,49` proves the dispatch seam: `middleware.CommandIdempotency(store, …)` (go-cqrs-lite `middleware/v4`) short-circuits a replayed command with `idempotency.ErrDuplicate` before the handler runs.
- The _production_ idempotency story routes through go-cqrs-lite's `middleware/v4` factory on `ServiceConfig.CommandMiddleware` (`setup/config.go:517-523`; guide [§1](../../guides/leveraging-go-cqrs-lite.md) "Dispatch middleware", store decision at line 113) — the right layer, since dedup belongs at dispatch, not HTTP decode.
- **`usermgmt/command_idempotency.go` (K2, executed 2026-11-19; shipped in usermgmt v4.21.0 on the 2026-11-21 family train) — the guard is now a PRODUCT DEFAULT:** `NewService` mounts a replay guard by default inside the audit chain (`usermgmt/service_factory.go:253-263`), keyed by the actor-scoped `usermgmt.IdempotencyMetadataKey` metadata (`command_idempotency.go:37`, TTL `DefaultCommandIdempotencyTTL` 10 min at :22), store injectable via `ServiceConfig.IdempotencyStore` (nil = the upstream memory default with its `//nolint:staticcheck` deprecation receipt at `command_idempotency.go:89-91` — dev/test class by upstream decree), `ErrDuplicate` → Conflict 409 translation built in, opt-out `ServiceConfig.DisableCommandIdempotency`. The guard owns its claim — a self-contained `CheckAndRecord` mirror (`command_idempotency.go:105`) — so an `ErrDuplicate` from a consumer-mounted idempotency middleware passes through untouched (four tests + the opt-in sentinel test pin it).

**Relationship with go-cqrs-lite (re-verified 2026-11-22):** the two libraries are complementary, not competing. go-cqrs-lite deleted its in-repo `idempotency/v4` shim in the ADR-0128 extraction, making **go-idempotency the Store contract behind the CQRS factory**: `go-cqrs-lite/middleware/v4` now pins go-idempotency **v0.6.0** (`middleware/v4/go.mod:17` — moved off v0.3.2 with this repo) and takes `store idempotency.Store` with a `command.Command`-native adapter + keyExtractor defaulting to `cmd.ID()` (`middleware/v4/idempotency.go:55-62`). Its durable option is the `idempotency/sqlstore/v4` submodule, which still pins **v0.3.2** (`idempotency/sqlstore/v4/go.mod:10`) — an intra-train skew this repo does not feel (no sqlstore consumer here) but the next cqrs-lite train owes a reconcile.

## Capability inventory

| #   | Capability                    | Status             | Evidence / reason                                                                                                                                                  |
| --- | ----------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | `Store` contract              | 🟢 Fully leveraged | aliased at root; the K2 guard consumes the upstream type (`service_core.go:216`, `command_idempotency.go:89,105`); admin-demo `ackMiddleware` takes `idempotency.Store` (main.go:392) |
| 2   | `MemoryStore` + sweeper       | 🟢 Fully leveraged | documented default (`NewMemoryIdempotencyStore` delegates); upstream now marks `MemoryStore` **Deprecated for production** (store.go:150) — the K2 default mount cites exactly that class with its nolint receipt; `WithClock` (v0.6.0) unused here (no clock-dependent store tests yet) |
| 3   | `ErrDuplicate` classification | 🟢 Fully leveraged | maps to 409 Conflict in admin-demo (`MapError`) and in the K2 guard's built-in translation; aliased as `ErrDuplicateCommand`; seam test asserts `errors.Is(…, idempotency.ErrDuplicate)` (command_idempotency_test.go:49) |
| 4   | Durable store variants        | ⚪ N/A             | root library offers none (memory-only, no-backends stance); durability arrives via go-cqrs-lite `idempotency/sqlstore/v4` implementing this `Store`                     |
| 5   | Direct-adoption posture       | 🟢 Fully leveraged | **FLIPPED 🟡→🟢 this refresh:** the integration is no longer just the deprecated alias layer — `usermgmt` ships prod code importing the upstream root directly (command_idempotency.go:11, service_core.go:15) behind a default-on guard; aliases remain scheduled for v5 removal |
| 6   | `middleware/` subpackage      | ⚪ N/A             | superseded here — go-cqrs-lite's `CommandIdempotency` wraps the same Store with a native `command.Command` adapter; cqrs-htmx's seam is that factory (verified again in the K2-era test) |
| 7   | `contract/` subpackage        | ⚪ N/A             | store-implementor test kit; becomes applicable only when a custom (SQL) `Store` lands per the guide's production recommendation (line 113)                             |
| 8   | `idemtest` consumer test kit (v0.5.0+) | 🔴 Missed  | 0 imports anywhere in the workspace — while usermgmt hand-rolls its guard tests. The fleet's own **library-policy companion rule** (`library-policy/policies/ecosystem.go:102-115`, SeverityModerate, 2026-11-17) demands `go-idempotency/idemtest` in imports or go.mod of any go-idempotency consumer: "a project that uses go-idempotency must drill-test at-most-once behavior with idemtest". cqrs-htmx is currently NON-COMPLIANT with that policy |
| 9   | `Forever` permanent claims (v0.4.0) | ⚪ N/A     | no permanent-claim use here: the K2 guard is TTL-bounded by design (10-min replay window, sized to client retry budgets); event dedup routes through metaengine `DedupStore`, not this library |

## Score derivation

9 rows − 4 N/A = **5 applicable**; 4 full ×1 + 0 partial + 1 missed ×0 = **4.0 → 4.0/5 = 80/100** (band: high). Movement from the prior 88 (then 4 applicable, 3 full + 1 partial): row 5 🟡→🟢 (K2 made the upstream import load-bearing prod code) adds +0.5-worth of posture, but the new row 8 (`idemtest`, 🔴 — and policy-violating, not merely optional) grows the applicable set faster than the numerator. The drop is the honest cost of the fleet having formalized the companion rule after 2026-11-14; closing it (adopting `idemtest` drills in `usermgmt`) returns the score to 4.5/5 = 90.

## Version currency

v0.6.0 pinned (root go.mod:19 direct; huma/setup/loginpage/systemadapter/e2e indirect) = v0.6.0 latest (verified `git ls-remote --tags origin`, 2026-11-22). The library's `[Unreleased]` is `idemtest`-side only (pinned-clock expiry drill — already released as the nested `idemtest/v0.1.1` tag; fuzz-battery and drill-tightening entries) — no root-module release pending, nothing to bump.

## Should cqrs-htmx use it (more)?

- **CLOSED — M2M bot-path gap (2026-11-06 → resolved 2026-11-15/19):** the documentation half (B2: guide recipe "Client-driven idempotency for token-authenticated bot traffic", `docs/guides/leveraging-go-cqrs-lite.md:117-185`, decode `Idempotency-Key` → `WithCustomMetadata`, extract via the metadata keyExtractor, translate `ErrDuplicate` → Conflict 409) AND the default half (K2: `usermgmt.NewService` mounts the guard by default, keyed on the same metadata — recipe followers are deduped with zero extra wiring; shipped in usermgmt v4.21.0, 2026-11-21 train) have both landed. Bots that send `Idempotency-Key` are deduped by default; key-less traffic passes byte-for-byte unchanged.
- **OPEN (this refresh, Important) — idemtest adoption, owning row: fleet library-policy companion rule.** Trigger: the next usermgmt test pass (or the policy scan reaching this repo). Shape: add `github.com/larsartmann/go-idempotency/idemtest` to `usermgmt/go.mod` (test-only) and replace/augment the hand-rolled guard tests with `RequireExactlyOnce`/`RequireOneWinner`/`RequireFailClosed` drills over the default guard's `func(ctx, key) error` seam — one session, closes the policy finding, +10 score.
- **OPEN (owner question, unchanged from K2) — multi-instance follow-up:** reference SQL/Redis `idempotency.Store` guide example vs the seam + docs suffice (`ServiceConfig.IdempotencyStore` is the injection point; guide line 113 carries the production guidance).
- The ADR-0056 sync push path already anticipates consumer-wired idempotency: each envelope's `commandId` is stamped as `X-Command-Id` precisely so consumer-wired middleware dedups replays (`sync_push.go:160-162`) — same consumer-wires-it posture, nothing for this library to change.

## How to verify

```bash
cat idempotency.go   # every alias delegates 1:1 upstream
rg -n 'go-idempotency' --type go -g '!scripts/**' . | grep -v _test    # 9 lines: idempotency.go:6,9,16,23,39 · doc.go:10 · admin-demo/main.go:42 · usermgmt/service_core.go:15 · usermgmt/command_idempotency.go:11
rg -c 'idemtest' --type go . | wc -l                                    # 0 — row 8, the policy gap (library-policy/policies/ecosystem.go:102 demands it)
rg -n 'DisableCommandIdempotency|IdempotencyStore' usermgmt/service_factory.go | head -2   # :254 (default mount), :259 (store injection)
git -C ~/projects/go-idempotency ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1                     # v0.6.0
rg -n 'go-idempotency' ~/projects/go-cqrs-lite/middleware/v4/go.mod ~/projects/go-cqrs-lite/idempotency/sqlstore/v4/go.mod  # v0.6.0 / v0.3.2 (sqlstore skew, not felt here)
```

## Prior reports

- 2026-11-05 (v0.3.2, score 88) — superseded by this refresh: module-surface undercount fixed, go-cqrs-lite train corrected v4.4.0→v4.7.2, M2M W0 gap added.
- 2026-11-14 (prior state of this file): 88 (K2-trued posture, score not re-derived) · delta: pin v0.3.2→v0.6.0 (three upstream releases), row 5 🟡→🟢 (K2 default guard = prod upstream imports), rows 8 (idemtest 🔴, policy companion rule unmet) and 9 (Forever ⚪) added, M2M bot-path closed (B2 docs + K2 default mount), MemoryStore upstream-deprecated, cqrs-lite middleware/v4 pin moved to v0.6.0.
