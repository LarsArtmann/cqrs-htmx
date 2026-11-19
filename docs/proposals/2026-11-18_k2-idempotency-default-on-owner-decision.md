# Owner decision packet — K2: idempotency default-on for token-authenticated dispatch

> **Filed:** 2026-11-18 (the GET-SHIT-DONE session; pre-stages the recommendation the 14-26 report §g② named as missing) · **Plan:** [`docs/planning/2026-11-15_07-21_SUPERB-fleet-audit-follow-through-v2-pareto-plan.md`](../planning/2026-11-15_07-21_SUPERB-fleet-audit-follow-through-v2-pareto-plan.md) §202 D3 · **Related:** B2's shipped recipe (`docs/guides/leveraging-go-cqrs-lite.md` § Client-driven idempotency for token-authenticated bot traffic), the M2M packet ([`2026-11-07_m2m-owner-decisions-d1-d4.md`](2026-11-07_m2m-owner-decisions-d1-d4.md)) · **Status: ADOPTED — owner ticked GO 2026-11-19 (via the native decision tool; receipt §a1 of the [20-47 report](../status/2026-11-19_20-47_owner-verdicts-k2-wave-templ-1118-hook-fix_status.md)); the wave executed the same day, see [Executed](#executed-2026-11-19-owner-go-tick-via-the-native-decision-tool) below.**

## The decision in one line

Should `middleware.CommandIdempotency` be mounted BY DEFAULT on the token-authenticated (bot) dispatch surface — keyed by the request's `Idempotency-Key` metadata, pass-through when the header is absent — or stay recipe-only (B2's status quo)?

## The mechanism, verified at source (2026-11-18)

`middleware/v4/idempotency.go` (`NewIdempotency` / `CommandIdempotency`, go-cqrs-lite): on each message the middleware extracts a key; **an empty key is a pass-through** (no dedupe, no store write); a non-empty key is `store.CheckAndRecord(key, ttl)` — first occurrence proceeds, later occurrences within the TTL return `idempotency.ErrDuplicate` WITHOUT calling the handler. The DEFAULT key extractor is the command's own minted ID (`cmd.ID().String()`), which dedupes re-invocation of the SAME command instance but not client retries (a retry mints a new command). The B2 recipe's keyExtractor bridges the wire header into that position: `Idempotency-Key` header → `WithCustomMetadata` → `MetadataCarrier` keyExtractor → `ErrDuplicate` → 409 translate; `examples/admin-demo`'s `ackMiddleware` already honors the header as the bot twin of `X-Command-Id` (replay-409 + pass-through tests).

The decisive property: **keyed-pass-through framing makes "default-on" cheap and opt-in at the REQUEST level** — mounting the middleware by default costs one store instance and one chain slot; dedupe engages only for requests that CARRY a key. Bots that never send the header are byte-for-byte unaffected.

## Recommendation

**GO — default-on, keyed by `Idempotency-Key`, empty-key pass-through, TTL bounded and documented.** Concretely: `ServiceConfig` (or the token middleware factory) mounts `CommandIdempotency(store, defaultTTL, metadataKeyExtractor)` on the dispatcher for token-authenticated dispatch paths; the store is the same class admin-demo wires (in-memory default, consumer-replaceable for multi-instance deployments); `defaultTTL` documented (5–15 min covers HTTP retry windows; not a de-duplication guarantee beyond it — a replay after TTL expiry executes again, by design).

Wrong-answer costs:

- **Guessed default-on (this recommendation): LOW.** No-key requests pass through unchanged (verified pass-through branch); keyed requests get exactly the protection the client declared; a consumer who dislikes it replaces the store or unmounts via the existing config seam (additive, reversible — a later default-OFF is one line).
- **Guessed stay-recipe-only: MEDIUM and invisible.** Every future token-authenticated consumer re-derives B2 by hand (the guide exists, but adoption is opt-in-by-discovery — the audit's original finding was that the M2M bot path minted NO key and nobody noticed until a fleet audit looked). Replay of a retried POST double-executes a command — the exact class idempotency middleware exists to kill — and nothing in the product surfaces it.
- **The one real hazard is key EXTRACTION, not the default:** a mis-derived extractor that hands DIFFERENT commands the SAME key (constant key, shared-header bug) rejects legitimate traffic with 409s. That hazard exists identically in the recipe-only world — B2 already ships the extractor to consumers — so defaulting the MOUNT does not add it; the B2 tests pin the extractor's correctness.

## Why this is a packet and not a landed default

D3 of the v2 plan (§202) records the split deliberately: "defaulting `CommandIdempotency` on for token-authenticated dispatch is a library-opinion change entangled with M2M owner-decision D1" (if bot-token M2M were deprecated — D1's rejected branch — a default-on bot surface would be scaffolding for nothing). The B2 documentation half landed; the default is the remaining half, and it shapes a published surface (`ServiceConfig`).

## What the tick unlocks (one small wave, ~2–4 h)

1. `ServiceConfig` seam: mount the middleware by default when a token path is configured; opt-out stays a config field (name it in the ADR if the owner wants a formal record — one-page ADR-0067 candidate).
2. Default TTL + store: constants + doc comments; store injectable for multi-instance deployments.
3. Tests: replay-within-TTL → 409 (`ErrDuplicate` translate, the admin-demo assertions promoted to the module suite), no-key → pass-through byte-parity, TTL expiry → second execution executes (documented honesty).
4. Docs: the B2 guide section gains "mounted by default on token paths since vX.Y.Z; header still required per request"; CHANGELOG entry; the go-idempotency.md libs-report row trued (default-on receipt).

## Overrule path

Any row overruled (e.g., "stay recipe-only"): strike the recommendation, record the answer in the TODO_LIST K2 row (canonical copy) — no landed work is invalidated; the recipe (B2) is complete and stands alone.

## Executed (2026-11-19, owner GO tick via the native decision tool)

The wave landed the same day, with ONE deliberate divergence from the unlock list above: the guard does not wrap upstream `middleware.CommandIdempotency` — it **owns the claim** (a self-contained mirror of the `CheckAndRecord` discipline with the Conflict translation built in), because wrapper composition re-familied `ErrDuplicate` bubbling from a consumer-mounted idempotency middleware deeper in the chain and inverted the B2 sentinel contract (the pre-existing opt-in test reddened before any push; receipt §a7 of the 20-47 report — the first implementation shipped a real regression of exactly the class the wave exists to prevent). Landed: `ServiceConfig.IdempotencyStore` (injectable; nil = in-memory default) + `ServiceConfig.DisableCommandIdempotency` (`service_core.go`); `usermgmt.IdempotencyMetadataKey` + `DefaultCommandIdempotencyTTL` (10 min — inside the recommendation's 5–15 band, the admin-demo browser path's existing value) + the actor-scoped key extractor (`command_idempotency.go`); the default mount directly inside the audit chain, after audit and before lifecycle + consumer middleware (`service_factory.go`); four new tests (keyed retry → 409 across different commands, key-less pass-through parity, opt-out, TTL-expiry re-execution) plus the pre-existing consumer-sentinel test, all green with the full usermgmt suite (18.3s) and hermetic lint 0 issues. Docs: the B2 guide's default-mount note, the usermgmt CHANGELOG entry, the go-idempotency libs-report true. The ADR-0067 candidate was NOT filed — this packet plus the CHANGELOG entry are the decision record (per-row verdict style; no owner request for a formal ADR).
