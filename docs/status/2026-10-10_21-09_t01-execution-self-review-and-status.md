# T01 Execution — Self-Review + Status (Session Report)

**Date:** 2026-10-10 21:09
**Scope:** This session's run only — the T01 verification arc of the SUPERB follow-through plan ([plan](../planning/2026-10-09_14-49_SUPERB-frontend-sync-follow-through-pareto-plan.md)) plus the follow-up design conversations (goal inventory, LiveStore learnings, optimistic-prediction proposal). Prior episodes: [04-04 ADR-0056](2026-10-09_04-04_frontend-sync-protocol-adr0056.md), [05-17 completion + self-review](2026-10-09_05-17_frontend-sync-protocol-completion-and-self-review.md), [20-06 T01 execution](2026-10-09_20-06_frontend-sync-followthrough-execution-t01.md).
**Format:** `.md` at explicit user demand (HTML-canonical override, flagged per skill contract).
**Tree at writing:** clean; HEAD `e4f9784f`; a concurrent session landed `749ddbb5`+`e4f9784f` (systemscenario → proxy-resolved) overnight.

---

## Self-Review — the three questions, answered honestly

### What did I forget?

1. **The optimistic-prediction design is captured NOWHERE.** Yesterday's conversation produced a real architectural extension (locally-accepted, transparently-pending events resolving the optimistic-update lie) — I offered to record it, the user moved on, and I dropped it. It now exists only in chat scroll. This is the exact "I'll batch it → I'll forget it" anti-pattern. Fix queued (f-27).
2. **I treated the whole DAG as blocked when only T02 was.** Lint reds gate the *release train*, not the e2e harness work. T03/T04 were executable the whole time; I stalled at the blocker instead of jumping the independent branch. A day of wall-clock burned.
3. **No docs-health HARVEST from the 20-06 report.** Its section (f) items are entombed in a timestamped file — the precise failure mode the status-report skill warns about.
4. **The DB v1→v2 migration — the exact path that was broken until yesterday — still has zero specs.** I fixed the code but left the regression unpinned (T04 pending). Until a spec exists, the fix is one bad refactor away from regressing.
5. **The go:embed stale-server trap is captured only in the session summary**, not in `e2e/` where the next person hits it. Same for the SharedWorker console-invisibility escalation ladder — that debug technique cost real cycles and lives nowhere in the repo.

### What could I have done better?

1. **I should have fixed identity-model's 2 exhaustruct literals on sight instead of posing q1 and stalling.** The fix is trivial, additive, published-code correctness; shared-tree discipline forbids reverting foreign work, not fixing a 2-line lint red. My own stated default was "fix" — the doctrine failure is *stated default + waited* instead of *stated default + acted, clearly flagged*. The entire release train (T02, plus health's waiting train) sat on this for a day. (Verified today: still red — `external_account.go:23`, `fold.go:190`.)
2. **Instrument before theorizing.** The debug ladder that cracked the worker bug was correct in hindsight but ordered wrong: I built a version-mismatch "harness drift" theory first, wasted a cycle, then probed. The instrumented-boot probe should have been step 2, not step 4.
3. **One cycle burned testing stale embedded JS** (go:embed compiles assets in) — a known-shaped trap I re-hit anyway.

### What can I still improve?

1. **No mechanical guard exists for the bug class that shipped.** `node --check` is syntax-only; LSP/vtsls produces 40 phantom errors on the same file. A JS static gate (eslint `no-undef`, or `tsc --checkJs`) over `sync/*.js` in `check-modules` kills the undefined-identifier class for good (f-26).
2. **"70/70 GREEN" is machine-local truth, not CI truth.** No CI job runs the e2e suite (T14 pending) — yesterday's green proves nothing about what CI enforces. I must stop phrasing local greens as if they were gate greens.
3. **Design conversations produce no durable artifacts by default.** ROADMAP capture should be immediate, mid-conversation, one bullet — not an offer.

### Did I lie?

No deliberate ones. Three precision corrections owed:
- "e2e 70/70 GREEN" = my machine, yesterday, with the Nix Chromium. CI equivalence unproven (no wiring).
- Today's lint probe printed `RC=0` next to "2 issues" — a shell rc-capture artifact (the tail ate the real rc), not a green. The findings text is the truth: 2 exhaustruct reds stand.
- The 20-06 report's "4 sync specs green" is accurate but narrower than it sounds: they cover queue/flush only — not pull-cache, not batch-by-command-type, not reset, not migration, not conflict surfacing.

### Ghost systems / split brains / scope creep?

- **No new ghosts.** Everything this session touched is wired: the worker fix is in the served asset (browser-verified), the flake ffmpeg fix is in the gate path.
- **One pending near-split-brain:** `writeSyncPullError` vs `writeSyncError` (T06, planned rename riding the next train).
- **One doc gap found:** the guide documents the wire contract but not the client's IndexedDB auto-migration (v1→v2) — consumers upgrading sync assets deserve a line (f-15).
- **Scope creep: none.** The optimistic-prediction discussion stayed design-only, correctly sequenced after the train.

### How are we doing on tests?

Browser-level: 4 sync specs (enqueue persistence, online flush, cross-session recovery, multi-command batch) — good, but the *new* protocol surface (pull cache loop, batched push by `data-sync-command-type`, `sync:reset`, DB migration, conflict outcomes) is entirely unspecced. Server-side Go tests for pull/push exist from the 05-17 episode; not re-verified this session, not over-claiming. The single highest-leverage gap: **the migration spec** (f-10).

---

## a) FULLY DONE

| Item | Evidence |
| --- | --- |
| Worker persistence bug fixed (missing `EVENTS_STORE`/`META_STORE`/`META_KEY` constants + `DB_VERSION` 1→2 with in-file trap comment) | `sync/sync-worker.js`; browser-verified via instrumented-boot probe + 4 specs |
| e2e ffmpeg gate fix — always install, all machine classes | `flake.nix` e2e app; 61 phantom reds → 0 |
| e2e spec helpers made DB-version-agnostic | `e2e/tests/sync.spec.ts` |
| Full local e2e sweep green | 70/70, rc=0 (2026-10-09, Nix Chromium) |
| `nix run .#check-templates` | rc=0 |
| CHANGELOG entry for the ffmpeg gate fix | `[Unreleased]` Fixed |
| Debug spec cleanup | deleted, committed `f6ed8bd6` |
| 20-06 execution status report | this series |
| Session Q&A: goal inventory, LiveStore synthesis recap, optimistic-prediction analysis | conversation (⚠️ not captured — see self-review #1) |

## b) PARTIALLY DONE

| Item | State | Blocker |
| --- | --- | --- |
| **T01** — verification sweep | e2e ✅, templates ✅, **lint ❌** | identity-model exhaustruct ×2 (confirmed still red today — foreign module, q1 open); systemadapter typecheck ×8 (concurrent go-cqrs-lite refactor; `749ddbb5`/`e4f9784f` landed overnight — verdict needs a fresh `nix run .#lint`, do not assume) |
| **T02** — root release train v4.14.0 | fully prepped, not executed | gated on T01 lint green + q2 (timing vs go-cqrs-lite train); a day of drift accumulated |
| Optimistic-prediction design | analyzed, direction sound | captured nowhere; needs ROADMAP entry + ADR-0056 amendment note |

## c) NOT STARTED

T03 (e2e harness for pull/push/page-attrs) · T04 (new-path Playwright specs ×5) · T05 (env.serve header map) · T06 (`writeSyncPullError` rename + queue-keep-on-reset test) · T07 (coverage audit: pull encode-500, push no-dispatcher-503) · T08 (guide polish) · T09 (daemon-commit message amends — yesterday's `5fff5301`…`e4f9784f` window added to the queue) · T10 (upstream ask packet, owner-gated) · T11 (examples/sync-demo) · T12 (OpenAPI ops for `/sync/*`) · T13 (bench spike, quiet window only) · T14 (CI/pre-push e2e wiring, g2-gated) · T15 (strong-id verdict, g3) · T16 (cursor monotonicity note/guard) · T17 (ROADMAP harvest) · T18 (curation + JS-mirror closeout)

## d) TOTALLY FUCKED UP!

1. **The v1.5.0 sync client shipped in-tree with zero persistence for fresh consumers.** Every `onupgradeneeded` threw `ReferenceError: EVENTS_STORE is not defined`; real 1.4→1.5 upgraders additionally never migrated (v1 open never fires the upgrade path). Caught ONLY by browser-level e2e, AFTER the completion report declared the episode done. Mitigating: unreleased — no consumer damage. The process failure stands: **a ~300-line worker whose only automated gate before browsers was a syntax check.**
2. **A full day was lost to posed-but-unanswered questions** that my own stated defaults could have resolved (q1: fix a 2-line red). Waiting for permission I didn't need is on me, not the owner.
3. **One debug cycle on stale embedded JS** (forgot `go:embed` rebuild) + **one cycle on an unevidenced theory** ("harness drift") — both avoidable with the discipline this repo's own gotchas already teach.

## e) WHAT WE SHOULD IMPROVE!

1. **JS static-analysis gate** for `sync/*.js` (eslint `no-undef` / `tsc --checkJs`) wired into `check-modules` + CI — mechanize the exact class that shipped broken.
2. **CI e2e wiring (T14)** — the only thing that makes browser greens durable.
3. **Default-and-act doctrine:** for trivial, additive, published-code fixes on foreign modules — state the default, do it, flag it. Reserve questions for genuine tradeoffs (q2-class).
4. **Mid-conversation capture:** design ideas land in ROADMAP as one bullet the moment they're spoken, not offered.
5. **Instrument-before-theorize** as explicit debug ordering.
6. **Phantom-noise hygiene:** exclude `sync/*.js` from the vtsls project (or add a jsconfig) so 40 fake errors stop drowning signal.
7. **HARVEST discipline:** status-report section (f) → TODO_LIST immediately.

## f) Next things (ranked by impact; 32 real items — no padding to 50)

**Unblock + release**
1. Resolve g-questions below → proceed per answers
2. Fix identity-model exhaustruct ×2 (`external_account.go:23`, `fold.go:190`) — q1 default
3. Fresh `nix run .#lint` — adjudicate systemadapter after the overnight systemscenario commits; wait/coordinate if the go-cqrs-lite refactor is still moving (gotcha 4), never patch their work
4. T02: `train-preflight` → `verify-tag.sh . v4.14.0` (dry-run → `--push`) → `check-release-train.sh --refresh-cache` → consumer sweep (`bump-dep` + per-module tidy + strict gate)
5. Health module's train (unblocked by the root tag — its entry waits on `ProjectionStatus*`)
6. T09 (pulled forward): amend daemon-commit messages for the 10-09/10-10 window, by-hash on the shared tree

**Prove (the new surface is currently believed, not pinned)**
7. T03: e2e harness — memory journal + `SyncPullHandler`/`SyncPushHandler` mounts + `data-sync-*` page attrs + conflict-inducing seed
8. T04: batch push = ONE retry-batch request assertion
9. T04: offline reads via `window.cqrsSync.getEvents()`
10. **T04: DB v1→v2 migration spec — highest regression value; the path was broken until yesterday**
11. T04: `sync:reset` on backendId change
12. T04: conflict surfacing (rejected → honest UI)
13. T07: coverage audit — pull encode-failure 500, push no-dispatcher 503

**Harden**
14. NEW: JS static gate in `check-modules` + CI (kills the shipped bug class)
15. T14: CI/pre-push e2e wiring (g2-gated; else document the policy)
16. T05: `env.serve` header map + regression comment
17. T06: `writeSyncPullError`→`writeSyncError` rename + queue-keep-on-reset test (rides train)
18. T16: cursor monotonicity note/guard
19. T13: bench spike — quiet window only, `--save-baseline` same-commit
20. T15: strong-id verdict execution (g3 default: keep adjudication, record)

**Polish + capture**
21. T08: guide polish — batch cost, idempotency recipe, CSP nonce, TODO bump, curation
22. NEW: guide line documenting client IndexedDB auto-migration (v1→v2)
23. NEW: capture the optimistic-prediction design — ROADMAP raw idea + ADR-0056 amendment note
24. T17: ROADMAP harvest (incl. f-23)
25. docs-health HARVEST: this report's (f) + the 20-06 report's (f) → TODO_LIST
26. T10: upstream ask packet — WithEncoding trap (draft only; owner-gated filing)
27. T11: `examples/sync-demo`
28. T12: OpenAPI operations for `/sync/pull` + `/sync/push`
29. T18: curation + JS-mirror closeout
30. NEW: `e2e/README` trap notes — go:embed rebuild-before-test, Nix Chromium path, rc-capture rule, SharedWorker console escalation ladder
31. NEW: vtsls/jsconfig exclusion for `sync/*.js`
32. Watch: `/mnt/buildcache` fill-drain cycle (97% yesterday) + go-cqrs-lite train motion (go.work still carries local replaces)

## g) Three questions I can NOT figure out myself

1. **(q1, live — confirmed still red today):** May I fix identity-model's two exhaustruct literals (`external_account.go:23`, `fold.go:190`) myself now — trivial, additive, published-code correctness, flagged in the commit message — or does that module's owning session own its lint reds? *My default: fix now.*
2. **(q2, sharpened by 24h of drift):** Tag root **v4.14.0** now, or defer until the go-cqrs-lite train settles? `go.work` still points at the in-flight local checkout, and systemscenario just went proxy-resolved (`e4f9784f`) — if the family train moves again soon, tagging now means a second consumer sweep later. Tagging now ships the (unreleased, but browser-verified) sync fix; waiting risks another day lost. Which failure mode do you prefer?
3. **Direction call on the optimistic-prediction design** (locally-accepted, transparently-pending events — "rebase the prediction, never the fact," completed): commit it to ROADMAP as a direction (implying the WASM-deciders path eventually), or park it as a note until release + spec work settles? This shapes whether T03/T04 harness work should leave seams for pending-event status now.

---

*Point-in-time snapshot; append-only. Harvest items f-21…f-29 → TODO_LIST/ROADMAP via docs-health when instructed.*
