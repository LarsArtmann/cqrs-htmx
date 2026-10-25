# Round 28 — W01 landed (S37 closed end-to-end), CI-red fixed twice, W02 half done

> **ANNOTATED 2026-10-25** — ~~this report~~ **SUPERSEDED** by [`2026-10-25_15-22_w02-closed-w03-landed-one-receipt-missing_status.md`](2026-10-25_15-22_w02-closed-w03-landed-one-receipt-missing_status.md), which Superseded by the round-29 receipt: W02 closed, W03 landed, all three questions restated. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-25 (round-23 (2026-10-25)).**

**Written:** 2026-10-25 13:30 CET · **Author:** module-code session (10-42 plan executor, W-lane)
**Supersedes:** [`2026-10-25_10-37_wave-d-resume-d01-d03-landed-d05-in-analysis_status.md`](2026-10-25_10-37_wave-d-resume-d01-d03-landed-d05-in-analysis_status.md) (its §"What remains" §1 is DONE; its §a–g analysis was executed as W01)
**Plan of record:** [`../planning/2026-10-25_10-42_SUPERB-wave-d-completion-code-fidelity-to-closure-pareto-plan.md`](../../planning/2026-10-25_10-42_SUPERB-wave-d-completion-code-fidelity-to-closure-pareto-plan.md) + the sibling's lane addendum (`824b3dba`): module code W01–W08/W10/W11/W14/W15 = this session; W09/W13/W16 + W17 verdicts = sibling session.

---

## §0 What happened this round (short)

The continue-directive executed the 10-42 plan from W01. Landed: the CI-red fixture fix
(pushed `d19249bb`), the FULL S37 milestone (all six tails), and the loginpage half of S36
(W02). Two CI incidents occurred and were fixed same-session. W02's adminui half and the
module CHANGELOG receipts are the only started-but-unfinished code; W03+ untouched.
All gates green at last push (`a9e7d64c` through strict pre-push; watch in flight at report
time — see §c).

## §1 FULLY DONE (verified this round)

| Item | Verify | Receipt |
| --- | --- | --- |
| **CI-red #1** — `checks` job red on `8c51ec59`: the D03 hook template referenced `check-templ-codegen-staged.sh`, but the `test-install-git-hooks.sh` fixture never copied it into scratch repos (T6/T12 failed, 13/17) | self-test 17/17 local; CI `checks` green on `824b3dba` | `d19249bb` |
| **W01/S37a** — TenantDeleted casbin cascade: `removeAllRolesForDomain` enumerates the projection's own `GroupPolicies()` view and removes every `g` row in the tenant domain (member grants + mirrored hierarchy edges) via `RemoveAllRolesInDomain`; deleted tenants no longer authorize their former members | `TestService_DeleteTenant_CascadesCasbinPolicies` green (grants die in A, survive in B) | daemon commits from `bd71283a` |
| **W01/S37a-service** — `revokeMembershipsForTenantBestEffort` counts failures and emits ONE summary warning (`partial membership cleanup`) instead of N swallowed log lines; casbin cleanup proven independent of dispatch success | `TestService_DeleteTenant_PartialCleanupIsVisibleAndStillCascades` (phantom member: dispatch fails, summary logs, policy still cascades) | same |
| **W01/S37b** — rebuild recovery host: a Reset/create failure after Stop now starts a recovery host from existing checkpoints (idempotent replay converges); original error still returned; double-failure returns combined `usermgmt.rebuild.recovery_failed` | `TestService_RebuildProjection_UnknownProjection` — functional pin: Register+GetUser work after failed rebuild (pre-fix: read model permanently stale) | `cb1ca749` region |
| **W01/S37d** — lockout fixed-expiry (owner verdict 2026-10-25): failures during an active lock neither extend it nor count toward the next window | `TestAccountLockout_ActiveLockIsFixedExpiry` (probes leave `lockedAt` byte-identical; lock expires at original expiry) | `08771fea` |
| **W01/S37d** — `EvictStale` truth: sweeps abandoned attempt counters (last failure > Duration, never locked) via new `attemptAt` map; doc no longer lies | `TestAccountLockout_EvictStaleSweepsAbandonedAttempts` (abandoned evicted, fresh survives) | same |
| **W01/S37f** — empty-journal fallback warns: `journalFromStore` logs a startup/rebuild warning naming the store type when a non-Journal store yields the EMPTY in-memory journal | `TestJournalFromStore_WarnsOnFallback` | `d4f72c6a` region |
| **W01/S37c** — session-cookie Max-Age pin (behavior already correct): default = session TTL, explicit `SessionMaxAge` wins | `TestAuthHandler_SessionCookieMaxAge` | `1be4685a` region |
| **W01 receipts** — usermgmt CHANGELOG §Unreleased/Fixed: all five S37 fixes written in the same change | CHANGELOG committed | daemon commits |
| **W01 verify battery** — `go vet` OK · full usermgmt suite `-count=1` OK (18.5s) · `.#lint usermgmt(+totp+webauthn)` 0 issues | rc=0 each | this session |
| **W01 delivery** — pushed through strict pre-push gates (release-train strict 0/0/0 + version-drift strict) | push accepted `824b3dba..a9e7d64c` | this session |
| **W02/S36-loginpage** — Option B parity: `applyRequestScoped` extracted; `buildPageData(r != nil)` now populates Nonce (sanitized) + Theme; ServeHTTP uses the same helper (no drift) | loginpage suite green incl. new `TestNewPageData_PopulatesNonceAndTheme` | `c3f85b75` + uncommitted test (committed with this report) |
| **Table-view report** (owner directive items 3+4) — W01–W17 milestone table + 84-micro-task condensed table rendered in chat | delivered in chat | this session |

## §2 PARTIALLY DONE

| Item | Done | Missing |
| --- | --- | --- |
| **W02/S36** | loginpage Nonce/Theme fix + pin test green | w02.3 adminui AccentColor grammar validation · w02.4 adminui partial-render errors → non-200 · w02.5 CHANGELOG receipts (loginpage + adminui) + full module suites |
| **CI watch on `a9e7d64c`** | watch running since push (bg shell 1FE, log `/tmp/ci-watch-a9e7d64c.log`) | terminal verdict — all observed jobs green so far; next session reads the `CI-VERDICT` line first |
| **Daemon co-mingling** | all W01+W02 content IS committed (content never lost) | attribution is mixed across 6 daemon commits (`bd71283a`..`c3f85b75`), interleaved with the sibling's W09 treadmill/wave-tag files — reconcile with `git log --stat` at W12 per guardrail 2 |

## §3 NOT STARTED (from the 10-42 plan, in order)

W03 (S39 six module tails) · W04 (S38 identity-model a–g) · W05 (S42 declarative fidelity) ·
W06 (S46+S49 seams/StateShape) · W07 (S58 stamping+mirrors) · W08 (sibling D02 verify) ·
W10 (loginpage E2E) · W11 (five file splits) · W12 CHECKPOINT (battery+push) ·
W14/W15 (harvested backlog I/II) · shared W17 closure. (W09/W13/W16 + W17-verdicts are the
sibling's lane per the addendum — this session verifies only.)

## §4 TOTALLY FUCKED UP (and what it cost)

Nothing unrecovered. Two self-inflicted classes, both fixed in-session:

1. **The D03 hook-fixture gap (CI-red #1)** — my own D03 change edited the hook template
   without re-running `test-install-git-hooks.sh`; CI caught it (`40181909642`, checks 13/17).
   Cost: one red CI cycle + one fix commit (`d19249bb`). Lesson re-confirmed: a hook/template
   edit is not done until EVERY fixture self-test that installs it is re-run.
2. **A projection-status misread** — my first S37b test asserted workers must be "live",
   but in this wiring "stopped" is the drain-then-exit steady state (probe proved the FRESH
   service also reports all-stopped while events flow). Cost: ~15 min. The honest pin is
   functional (new events still consumed), which is what shipped.

## §5 IMPROVEMENTS (made this round)

- `applyRequestScoped` — one shared request-scoped populator for both loginpage render
  paths (Option A/B); the drift that caused S36 is now structurally impossible.
- `recoverProjectionHost` — the rebuild failure path now has a named, testable recovery
  contract (`usermgmt.rebuild.recovery_failed` names both failures when even recovery dies).
- Lockout `attemptAt` bookkeeping — abandoned counters are now sweepable AND probe-immune
  (fixed expiry), closing the unbounded-map + infinite-lockout pair in one coherent model.
- The cascade is enumerated from the projection's own view — no read-model dependency,
  so authorization cleanup cannot be defeated by membership read-model staleness.

## §6 NEXT THINGS (execute-verify ordered, top 50 — plan §3/§4 IDs)

1. Read `/tmp/ci-watch-a9e7d64c.log` `CI-VERDICT` line; if red: name job + triage before ANY new code.
2. Commit the uncommitted loginpage pin test with this report (done as part of this commit).
3. w02.3 — adminui: validate `AccentColor` against the shared grammar (dashboardui parity), reject invalid at `New` (test: accept `#0ea5e9`, reject `red; background:url(...)`).
4. w02.4 — adminui: partial-render (HTMX) error paths answer non-200, not 200-with-error-HTML.
5. w02.5 — CHANGELOG receipts (loginpage v-next + adminui) + `.#test-loginpage` + `.#test-adminui` + goldens; commit W02.
6. w03.1 — transport: ID dedup at the replay/live seam (no-dup test).
7. w03.2 — transport: unknown-cursor semantics documented or probed (decision + test).
8. w03.3 — auditlog: shutdown story for the mounted SSE viewer (test/doc).
9. w03.4 — health: `Recorder(nil)` reject + unknown-status wording (tests).
10. w03.5 — datastar: `NewEventBridge(nil)` guard + `MappedEventTypes` sort (tests).
11. w03.6 — oauth2: empty-`sub` rejection + OIDC nonce (tests).
12. w03.7 — totp: negative `Window` clamp (test).
13. w03.8 — W03 receipts + module suites; commit W03.
14. w04.1 — `GenerateUserID` → `brandid.NewID` (ordering test).
15. w04.2 — strict `ParseTenantID`/`ParseBotID`; folds reject ambiguous forms.
16. w04.3 — `Exists()`/`IsActive` split per state struct (impossible states).
17. w04.4 — `NewAuthz` must not seed `g2` under a custom model (test).
18. w04.5 — `NotConfigured` → permanent error family (tests).
19. w04.6 — command accessors return copies (mutation test).
20. w04.7 — kill internal deprecated `NewUserID` uses (grep clean).
21. w04.8 — idm CHANGELOG + `.#test-identity-model`; commit W04.
22. w05.1 — declarative vs legacy Enforce ladder diff table.
23. w05.2 — align ladder or document divergence (declarative test).
24. w05.3 — audit Action vocabulary parity (golden).
25. w05.4 — ActorID into declarative audit entries (test).
26. w05.5 — systemadapter CHANGELOG + suite; commit W05.
27. w06.1 — DocsHandler/DocsPath end-to-end mount test.
28. w06.2 — `RecorderChain(Recorder, auditPlugin)` chain proof.
29. w06.3 — SQL snapshot store `StateShape` round-trip test.
30. w06.4 — codec round-trip; file upstream if the stamp drops.
31. w06.5 — integration_test suite; commit W06.
32. w07.1 — `WithSchemaVersion(1)` on the 21 idm constructors.
33. w07.2 — stamp the 8 e2e/demo event sites.
34. w07.3 — delete health status mirrors + root require bump.
35. w07.4 — integration_test half + CHANGELOGs (idm + health); commit W07.
36. w08.1 — sibling D02 receipts presence checklist (S45/S52/S53/S54).
37. w08.2 — docs gates (links/freshness/corpus/tail) rc=0.
38. w08.3 — AGENTS quick-ref truth-pass on sibling rows.
39. w09 — VERIFY sibling's treadmill + wave-tag (self-tests green, flake wiring) — their lane.
40. w10.1–w10.3 — Playwright WebAuthn virtual-authenticator E2E + full-page render golden.
41. w11.1–w11.5 — the five file splits + touched-module test/lint.
42. w12 CHECKPOINT — `.#test` · `.#lint` · `.#coverage-gate` · `.#check-modules` (4× rc=0, two attempts max) + push + CI watch + `.#verify-remote-tags` + ahead 0.
43. w13 — verify sibling's owner packet + watch table.
44. w14.1–w14.6 — backlog I rows (AutoRedirect, dashboardui tails a/b/c, biome.json verdict, sweep M18/M20/M21, RequireSession).
45. w15.1–w15.6 — backlog II rows (art-dupl r3, skill gates, canaries a–c/d–g, upstream drafts).
46. w16 — verify sibling's BuildFlow filing + bench verdict.
47. w17.1–w17.2 — verify sibling's 8 parked-row verdicts + TODO hygiene gate.
48. w17.3–w17.4 — FINAL battery + push + CI watch (mine).
49. w17.5 — final receipt (supersede THIS file; archive ritual; TAIL ≤ 2; corpus after mutations).
50. w17.6 — CURRENT_FOCUS rewrite (Wave D done; next thread named).

## §7 QUESTIONS I CANNOT ANSWER MYSELF (max 3)

1. **S37b steady-state naming:** upstream projectionhost v4.7.1 workers report `stopped`
   as their DRAIN-COMPLETE steady state (drain-then-exit + bus tail), which reads alarming
   in `ProjectionStatuses()`/dashboardui health panels. Is a UX rename/interpretation
   upstream (`drained`?) wanted, or do we document `stopped == healthy-idle` in the
   projection-health guide? (Upstream call, not mine to make.)
2. **Lockout semantics under load (follow-up from the fixed-expiry verdict):** during an
   active lock the counter no longer increments — meaning a probe DURING a lock is
   invisible to `EvictStale` age-sweeping until the lock row itself expires. I believe
   this is exactly the ratified semantics (probe-immunity), but confirming the intent:
   yes?
3. **adminui AccentColor grammar:** dashboardui pins a shared color grammar — should
   adminui REJECT-at-`New` (breaking for consumers passing junk that previously
   "worked") or sanitize-to-default + warn? I lean reject-at-`New` (fail fast, library
   principle), but it is a consumer-visible behavior change mid-v4.

---

*Report ritual: this file supersedes the 10-37 report (archived in this commit); tail
returns to 2; corpus baseline measured after mutations.*
