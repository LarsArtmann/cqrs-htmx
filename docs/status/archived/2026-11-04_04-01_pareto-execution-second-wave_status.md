# Pareto execution, second wave — R3/R9/R10 shipped, R12 honest gap persists

> **ANNOTATED 2026-11-04 (docs-health round 46):** RESOLVED-AND-ROUTED — 61 items dispositioned
> inline below. §a receipts verified against the tree (R3 `823fcd7e`, R9 `72a0ae84`, R10
> `13d0f0e9`, L1 half `469a702f`); the §f25-31 memory items this report left "not started" were
> EXECUTED by round-46 (AGENTS/FEATURES/CHANGELOG/TODO trued — the drift this report warned
> about is closed); §f1-24/33-40 open remainders live in TODO_LIST rows (leverage + cqrs-lint +
> upstream-asks); §g1 answered by execution (push done, 17-28 report), §g2/§g3 routed (ROADMAP
> OQ20; TODO triage row). §d process ledger: acknowledged, no open action. Successor for the
> docs-health thread: [`2026-11-04_18-52_docs-health-round46-…`](2026-11-04_18-52_docs-health-round46-2026-0-corpus-six-docs-truth_status.md);
> the leverage plan itself stays ACTIVE. **ARCHIVED 2026-11-04 (round-46 docs-health pass).**

**Date:** 2026-11-04 04:01 CET · **Session:** ~01:00 → 04:01 · **Thread:** [leverage plan](../../planning/2026-11-03_20-02_SUPERB-gocqrslite-leverage-maximization-pareto-plan.md) (supersedes [the first-wave report](2026-11-04_00-12_pareto-execution-first-wave_status.md))
**Commits this session:** `823fcd7e` (R3) · `72a0ae84` (R9) · `13d0f0e9` (R10) · `13d0f0e9..HEAD` (L1 half) · upstream [go-cqrs-lite#113](https://github.com/LarsArtmann/go-cqrs-lite/issues/113)
**Tree at report time:** master ahead 20→21, working tree clean after the L1-half commit.

## a) Fully done

~~1. **Session recon.** The concurrent session ended at 00:29 (`9f452f43`,~~ v:shipped — receipts verified 2026-11-04 round-46
   "standing by"). Tree was clean; their R3 work sat in `stash@{0}` (serve.go
   ring swap + root go.mod dedup v4.3.0 + the id/v4 v4.7.4 root completion the
   24-module sweep deliberately excluded). Took over the R3 lane per the
   standing directive; attribution carried in the commit message.
~~2. **R3 — dedup.Ring at the SSE seam (`823fcd7e`).** Stash popped cleanly;~~ v:shipped 823fcd7e — verified round-46 (FEATURES/CHANGELOG receipts)
   Ring nil-safety verified at `dedup/v4@v4.3.0` source (the stash's removal of
   the explicit nil check is the documented upstream contract). B24 shipped as
   `transport/replay_seam_test.go`: bounded/tail-exact eviction forced with a
   tiny ring through BOTH store adapters, the nil-ring contract (so an upstream
   regression fails here, not on a customer connection), and a handler-level
   no-duplicate-delivery test through the replay/live seam. B23 verified: C035
   gone. B25: S39 cross-link in the message. Transport suite green under
   `-race`, golangci 0 issues. Root id/v4 v4.7.4 alignment completed.
~~3. **R9 — opt-in FindByID write-through view cache (`72a0ae84`).** Upstream~~ v:shipped 72a0ae84 — verified round-46 (FEATURES/CHANGELOG receipts)
   `kv.NewCache` cannot wrap `SQLViewStore` (concrete `*TypedStore` param —
   verified at v4.3.5 AND master), so `usermgmt/view_cache.go` reproduces the
   ADR-0032 contract over the `kv.ViewStore` interface on otter — pinned
   v2.3.0 (kv's own pin; tidy's v2.4.0 bump broke the API) using kv's exact
   `GetIfPresent`/`Set`/`Invalidate` idiom. Opt-in only: constructor options +
   `UserViewCache *ViewCacheConfig` threaded ServiceConfig →
   EventSourcedConfig → dialect factory → setup.Config. FindByEmailSQL
   deliberately never cached. 8 tests: write-through-on-sync (post-event reads
   are pure cache hits), EmailChanged replacement, UserDeleted eviction +
   negatives-not-cached, disabled-by-default passthrough, copy isolation,
   FindByEmail-never-cached, capacity validation, config translation. Plan
   erratum found: Tenant/Bot have NO FindByID path (B50 vacuous — the adapter
   is generic for when they gain one).
~~4. **Upstream ask filed: [go-cqrs-lite#113](https://github.com/LarsArtmann/go-cqrs-lite/issues/113).**~~ v:filed; AGENTS + TODO carry the #113 pointer (round-46)
   The kv.NewCache concrete-type wall, with the codec-coupling root cause
   (`Cache.copy` round-trips through the TypedStore codec; `Has` is not in the
   ViewStore interface). All five verify-before-filing gates passed (master
   `6e117971a` checked, no duplicate issues); voice-checker green; provenance
   banner shipped unchecked for manual review. cqrs-htmx's adapter deletes
   when an interface-level constructor ships.
~~5. **R10 — snapshot shape guarding + v5 tail (`13d0f0e9`).**~~ v:shipped 13d0f0e9 — verified round-46 (FEATURES/CHANGELOG receipts)
   `repositoryOptions` (the single seam both repo builders funnel through) now
   applies `decider.WithSnapshotStateVersion("v1")` to all four aggregates.
   B39 test pins the full contract: fresh stamps, stale-stamp discard with
   journal rebuild, command success on rebuilt truth, self-healing rewrite.
   B33-35 recorded as a **plan erratum**: `record.Cause` exists as a type but
   `metadata.Metadata` carries no Cause field in ANY v4 release — the three
   dashboardui reads migrate at the v5 cut, not before. B40: cqrs-upgrade
   dry-run at master (raw artifact committed) — **41 V007, all in usermgmt,
   stable vs §5c**; 5 root-tag-gap modules fail the hermetic scan until the
   train tags HEAD. B41: reconciliation note added to the v5-removal-inventory.
~~6. **R4 debt fixed.** Last session's durable-DLQ commit placed its C017~~ v:shipped — C017 suppressions honest; L1-half receipt 469a702f
   suppression on the wrong line; both `NewMemoryDeadLetterStore` fallbacks now
   carry honest suppressions. usermgmt cqrs-lint is red on exactly ONE
   pre-existing finding (see d/3).
~~7. **Binary drift discovered.** Installed cqrs-lint is **4.19.1 pseudo~~ v:verified round-46 — AGENTS cqrs-lint row trued to 4.19.1; triage rowed
   (2026-11-03)**, not the 4.16.1 AGENTS.md documents — the new E-class
   adoption rules explain the 8-module gate red; it predates this session's
   commits.

## b) Partially done

~~1. **L1 (lint correctness) — first half committed** (B54 decode-outside-lock~~ r:B55/B58/B59/B60 remainder open — TODO_LIST leverage row L1 tail (round-46)
   in `audit_log.go`; B56/B57 C042 verified false positive + reasoned
   suppression in `auth_audit.go`; audit tests green under `-race`). **B55**
   (dedicated concurrent -race test) and **B58/B59/B60** (A013 value-embed ×2
   + e2e/sync-demo suites) are not started.
~~2. **R12 — triage only.** cqrs-lint drift investigated for usermgmt alone;~~ → routed — R12 remainder rides the TODO_LIST cqrs-lint 4.19.1 triage row (round-46)
   the full `check-modules` aggregate has still never run this session
   (the honest gap, second session running).

## c) Not started

L2 (B61-B64) · L3 C044 sweep (B65-B69) · T1 testutil (B70-B72) · W1 waivers
(B73-B74) · U1 systemcatalog tag ask (B75-B76) · F1 dashboardui FailureLog
(B77-B80, dependency R7 met) · X1 (B81) · X2 (B82-B83) · X3 (B84) · memory
updates (AGENTS.md rows for otter/4.19.1/#113, FEATURES.md, CHANGELOG,
TODO_LIST refresh) · push/train decisions.

## d) Totally fucked up (process friction; no lasting damage)

~~1. `rg -rn` hook trap hit **twice more** (strikes 4+5) after the handover~~ v:acknowledged — no further action
   summary explicitly said "never rg -rn". grep -n from now on, unconditionally.
~~2. LSP diagnostics half-trusted all session: stale golines/gocognit/nolintlint~~ v:acknowledged (gotcha 14) — no further action
   warnings disproven by real runs at least four times. Gotcha 14 is not a
   suggestion on this host — it is the rule.
~~3. The one usermgmt cqrs-lint ERROR left standing (audit_context.go:8 "never~~ v:acknowledged — carried into the 4.19.1 triage row
   imports go-cqrs-lite/catalog") is pre-existing binary drift, but I spent a
   fix cycle on MY suppressions before realizing the binary had changed under
   everyone. `cqrs-lint version` should be step 1 of any lint work.
~~4. One wasted cqrs-lint background gate: launched while R9 files were mid-write,~~ v:acknowledged — no further action
   results untrustworthy, re-run after the tree settled.
~~5. First B39 test draft mutated a map-value copy (classic Go aliasing) —~~ v:fixed in-session — no further action
   caught by the test failing, fixed by reassigning the map entry.
~~6. `go mod tidy` bumped otter to v2.4.0 whose in-v2 API drift broke the build;~~ v:fixed in-session (re-pin v2.3.0) — no further action
   re-pinned v2.3.0 and switched to kv's exact idiom. Two wasted build cycles.
~~7. Minor: one edit rejected for editing without a view-tool read first; two~~ v:acknowledged — no further action
   `nix develop` cd-path fumbles (cd inside an already-set working_dir).

## e) What to improve

- **Run `check-modules` (or at least `cqrs-lint version` + one gate module)
  FIRST when resuming a session** — the 4.19.1 drift would have surfaced in
  minutes instead of mid-R9.
- The plan's API references (B33 `record.Cause`) need pin-level existence
  checks during planning, not execution — this is the third plan erratum of
  this kind (fifth CBOR payload, "three SQL setup paths", now Cause).
- Memory rows (AGENTS.md) should ride the SAME commit as the work; deferring
  them to a wrap-up commit is how the 4.16.1 claim went stale in the first
  place.
- The kv issue draft→file pipeline worked well (draft in repo, checker, real
  file for --body-file, verify after) — keep that shape.

## f) Next (ordered)

~~1. B55 concurrent audit-log -race test (completes L1's first half)~~ r:TODO_LIST leverage row L1 tail (round-46)
~~2. B58 A013: `syncAddItemCmd` embed `*command.BasicCommand` (e2e/server/sync_types.go:19)~~ r:TODO_LIST leverage row L1 tail (round-46)
~~3. B59 A013: `createNoteCmd` (examples/sync-demo/main.go:44)~~ r:TODO_LIST leverage row L1 tail (round-46)
~~4. B60: e2e + sync-demo suites; commit L1 complete~~ r:TODO_LIST leverage row L1 tail (round-46)
~~5. R12: full `nix run .#check-modules`~~ v:done — round-46 ran the composite: 83/86, the 3 reds are documented owner-gated classes
~~6. R12: branching-flow vs re-pinned baseline; adjudicate new findings~~ r:owner-held baseline refresh (round-37 §g2 posture; +5 findings await R12 adjudication)
~~7. R12: annotate plan + audit report gate-green caveat~~ r:plan 2026-11-03_20-02 stays ACTIVE until the leverage plan closes
~~8. cqrs-lint 4.19.1 triage for the other 7 red modules (root, identity-model, adminui, loginpage, dashboardui, datastar, systemadapter)~~ r:TODO_LIST cqrs-lint 4.19.1 triage row (round-46)
~~9. Decide: usermgmt upstream-catalog wiring (R2 follow-up) vs reasoned suppression for audit_context.go:8~~ r:TODO_LIST cqrs-lint row + leverage R2 (systemcatalog-derived catalog)
~~10. L2 B61: remove stale S001 suppression (events.go:171)~~ r:TODO_LIST P2 cqrs-lint advisory batch row
~~11. L2 B62: A027 codec-once repository option (retire 3 WithCodec sites)~~ r:TODO_LIST P2 cqrs-lint advisory batch row
~~12. L2 B63: P008 projectionhost.WithBatchSize in e2e/server~~ r:TODO_LIST P2 cqrs-lint advisory batch row
~~13. L2 B64: F005 upcaster adjudication note~~ r:TODO_LIST P2 cqrs-lint advisory batch row
~~14. L3 B65: C044 inventory — KEY/CURSOR vs DISPLAY classification~~ r:TODO_LIST P2 cqrs-lint advisory batch row
~~15. L3 B66-B68: fix KEY/CURSOR `.String()` sites in dashboardui~~ r:TODO_LIST P2 cqrs-lint advisory batch row
~~16. L3 B69: cursor round-trip tests~~ r:TODO_LIST P2 cqrs-lint advisory batch row
~~17. T1 B70-B71: testutil.MustNewCmd / querytest swaps~~ r:TODO_LIST leverage row (T1)
~~18. T1 B72: testing-guide convention line~~ r:TODO_LIST leverage row (T1)
~~19. W1 B73-B74: `.cqrs-lint.json` scorecard waivers with triggers~~ r:TODO_LIST leverage row (W1)
~~20. U1 B75-B76: systemcatalog tag ask (verify no tag, search issues, file)~~ r:TODO_LIST upstream-asks row (b) systemcatalog tag ask
~~21. F1 B77-B80: dashboardui FailureLog/RetryCount views over commandlifecycle~~ r:TODO_LIST leverage row (F1)
~~22. X1 B81: ARCHIVE_EVICTED row for the 09-17 deep-dive~~ r:TODO_LIST leverage row (X1)
~~23. X2 B82-B83: headless render trace beside the audit report~~ r:TODO_LIST leverage row (X2)
~~24. X3 B84: cqrs-gen spike verdict (examples only)~~ r:TODO_LIST leverage row (X3)
~~25. AGENTS.md: cqrs-lint binary 4.19.1 + the drift story + gate-red triage state~~ v:done — AGENTS row landed round-46
~~26. AGENTS.md: otter v2.3.0 direct dep + viewCache row (R9)~~ v:done — AGENTS key-deps + FEATURES View Cache row round-46
~~27. AGENTS.md: issue #113 pointer — adapter deletes on upstream release~~ v:done — AGENTS utilization row round-46
~~28. AGENTS.md: R10 shape-stamp convention (bump with State shape changes)~~ v:done — AGENTS + FEATURES shape-stamp rows round-46
~~29. FEATURES.md: rows for R3/R7/R8/R9/R10 capabilities~~ r:R3/R9/R10 rows landed round-46; R7/R8 rows pending their ship (TODO leverage row)
~~30. CHANGELOG: entries for this wave~~ v:done — [Unreleased] leverage-wave + sweep entries round-46
~~31. TODO_LIST: leverage row — mark R1-R10 progress, keep U1/F1 open items~~ v:done — leverage row rewritten round-46 (R3/R9/R10 closed, remainder open)
~~32. Push decision: 20+ commits ahead (see question 1)~~ v:done — push landed; all eight session commits on origin (17-28 report)
~~33. Family train (root/usermgmt/dashboardui/adminui/loginpage/setup tags) per answer~~ r:owner GO pending — CURRENT_FOCUS round 46
~~34. Post-train: re-run cqrs-upgrade — the 5 root-tag-gap modules must scan green~~ r:post-train step (CURRENT_FOCUS done-when)
~~35. Watch #113; migrate viewCache → upstream NewCacheForViewStore when tagged~~ r:TODO_LIST upstream-asks row (c)
~~36. R9 B53: optional latency bench note (dispatch_bench class)~~ r:TODO_LIST leverage row (B53 bench note)
~~37. Guide note: UserViewCache in production-readiness or leveraging-go-cqrs-lite~~ r:open — production-readiness / leveraging-go-cqrs-lite guide note
~~38. TODO_LIST S39 row: note the dedup slice done (cursor-semantics remainder open)~~ r:dedup slice done via R3; cursor-semantics remainder open (TODO)
~~39. erraudit advisory re-baseline (TOTAL=303 claim may have moved with new code)~~ r:TODO_LIST erraudit row (re-baseline note)
~~40. Coverage gate after the wave (`nix run .#coverage-gate`)~~ r:post-wave step
~~41. `.#check-readme-quickstart` + skill-snippets after any doc edits~~ v:done — round-46 composite ran both gates green
~~42. Re-pin bench baseline only if bench paths changed (they did not this wave)~~ v:verified — bench paths unchanged this wave; no re-pin needed

## g) Questions I cannot figure out myself

~~1. **Push vs train.** Master is 20+ ahead with both sessions' waves interleaved.~~ v:answered by execution — push done (17-28 report); train remains owner GO
   Push straight (CI stays red on the root-tag-gap class until a train tags
   root/usermgmt/dashboardui/adminui/loginpage/setup) — or run the six-tag
   family train first so CI goes green on push? The concurrent session's
   §g.1 asked the same; both of us have been deadlocked on your call.
~~2. **Auto-git daemon.** It races multi-session work (it ate one commit last~~ r:ROADMAP OQ20 extended with the push-authority evidence (round-46)
   session; one index.lock failure on theirs). Pause it during train windows,
   or keep out-racing it with pathspec commits?
~~3. **cqrs-lint 4.19.1 posture.** The new E-class adoption findings fire at~~ r:TODO_LIST cqrs-lint 4.19.1 triage row (round-46)
   ERROR severity on 8 modules (catalog wiring, metaengine pushdown
   advisories). Fix-forward (wire upstream catalog in usermgmt — an R2
   follow-up with real design weight), reasoned suppressions, or downgrade the
   gate posture until the binary's rule set is re-baselined?
