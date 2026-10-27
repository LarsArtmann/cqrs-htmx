# Wave D lane closure: sync tails shipped as assets v1.6.0, battery run green-except-sibling-lint, push held

> **ANNOTATED 2026-10-25** — ~~this report~~ **SUPERSEDED** by [`2026-10-25_16-02_docs-lane-done-master-red-on-sibling-lint-self-review_status.md`](2026-10-25_16-02_docs-lane-done-master-red-on-sibling-lint-self-review_status.md), which Owner-demanded full status + self-review: same arc plus the sibling push/CI-red turn of events; supersedes the closure receipt. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-25 (round-23 (2026-10-25)).**

**Created:** 2026-10-25 15:57 CET · **Session:** continuation of the docs lane (10-42 plan + 10-43 addendum). **Supersedes:** [`2026-10-25_15-16_wave-d-lane-execution-master-healed-w13-w16-w17-quartet-closed_status.md`](2026-10-25_15-16_wave-d-lane-execution-master-healed-w13-w16-w17-quartet-closed_status.md) (archived same-commit; tail stays 2). **State at writing:** master CI green at origin tip `3ff9e0de` (run `40198848723`); 14 commits ahead (4 authored here: `47cf2308`, `dfc26bde`, `471ab2b2`, `5e0fb368`; the rest daemon-banked sibling W03 fragments + their 15-22 ritual); load avg 17–26 (down from 111–211).

## a) FULLY DONE (this session, receipts)

1. **Sync tails s.1–s.4 CLOSED — `5e0fb368` (7 files, squashed from a 3-commit daemon scatter; see d2).**
   - **s.1 spec:** `docs/guides/frontend-sync.md` gained "Local persistence: the IndexedDB schema and the v1→v2 upgrade" — the v1/v2 store table (v1 = `commands` only per ADR-0040; v2 adds `events` + `meta` per ADR-0056), the automatic upgrade path (`DB_VERSION = 2` open → `onupgradeneeded` creates the new stores, v1 pending commands preserved with retry counts, fresh DBs create all three), and the downgrade law (a stale v1 worker on a v2 DB gets `VersionError` → warns → degrades to in-memory; v2 data intact — per IndexedDB spec, not shim-testable).
   - **s.2 boot pins:** two new jstests — a v1-seeded FakeIndexedDB (version 1, commands store + one pending command) upgrades to version 2 with all three stores, the command counted and NOT evicted (including the documented hello-before-openDB race: the pre-open flush reports the in-memory 0; the test waits for the post-upgrade count-1 message); a fresh boot creates the full v2 schema. Harness generalized: `loadWorker({ idb })` accepts a pre-seeded shim, and `console.warn` is captured (`worker.warns`) so log lines can be pinned.
   - **s.3 monotonicity guard:** `cacheEvents` (sync-worker.js) now refuses to persist a cursor that would move backwards on the same backend (cursors are event-ID ULIDs — lexicographic compare IS the ordering, sync_pull.go:171); the stale batch's events still land (puts idempotent by `eventId`), the `sync:events` broadcast carries the KEPT cursor, and the refusal logs the pinned line `[sync-worker] cursor regression refused: kept …, rejected …` (asserted by the pin test). A `backendId` reset is the one legitimate rewind — pinned exempt in the same test. Pull correctness argument: the client's pull reads the cursor exclusively via `get-state`, so guarding the meta write fully protects the resume position.
   - **s.4 guide halves + verdict:** batch-budget table (`MaxSyncPushBatch` 100 — with the honest sharp edge: >100 queued commands = wholesale 400 `cqrshtmx.sync.push.too_large` → retry burn → `dead`; `PULL_PAGE_LIMIT` 500 × `PULL_MAX_PAGES` 20; `MAX_RETRIES` 10; `RETRY_TTL_MS` 24h), timeout posture (no asset-side deadlines — browser defaults; tuning = copied assets via the `With` handlers), and CSP wiring (`script-src` for both script URLs, `worker-src` for the SharedWorker load with the fallback chain, `connect-src` for pull/push/SSE; worker URL derivation + `data-sync-worker-url` override). **Family.IsZero() verdict: NO-OP, no referent** — `Family` is a JSON string field (`SyncPushError.Family`, sync_push.go), the client reads `result.error.family` as a string (sync-client.js:571); no Family type with methods exists in the sync surface or pinned upstream modules. Harvested micro-task falsified at referent check, as the 15-16 §e6 lesson predicted.
   - **v1.6.0 bump:** worker + client `VERSION` constants and `syncVersion` (sync_embed.go) moved together in one commit — `TestSyncVersionMatchesJSConstants` enforces the triple. CHANGELOG receipt under Unreleased/Fixed. 9/9 jstests green via `node --test` AND the Go wrapper (`TestSyncWorkerJSSuite`); root sync Go tests green.
2. **Checkpoint battery RUN (load 17–26, under the ~30 refuse-line; verdicts from unpiped rc only):**
   - `.#test` **rc=0** — every module ok (root, adminui, auditlog, dashboardui+core, datastar, health, identity-model, integration_test, loginpage, setup, systemadapter, usermgmt + 3 auth strategies).
   - `.#coverage-gate` **rc=0** — auditlog + health at 100% (this resolves the sibling 15-22 §b2 coverage worry: their W03 additions are fully gated).
   - `.#check-modules -- --report` **58/59** — sole red is `branching-flow`: +22 findings vs baseline (was +12 at 15:16; the sibling's W03 code grew it), CI-exempt, owner-questioned (g1).
   - `.#lint` **rc=1 — 7 findings, ALL in the sibling's W03 landings, ZERO in this lane's files.** Inventory in b1.
3. **Docs gates green same-commit:** corpus 32137 (bumped with the guide growth), links 844, todo-hygiene 64 rows clean, tail ≤2 (this receipt + sibling 15-22 after the 15-16 archive). AGENTS.md root-module row updated to "sync assets v1.6.0".

## b) PARTIALLY DONE

1. **PUSH HELD on content-red, not noise — 14 commits await.** The battery falsified the sibling 15-22 §b3 expectation ("green"): their W03 verification ran suites + hermetic vet but NOT lint (their §b2 admits the battery gap). CI runs the same per-module lint matrix (2.15.0 everywhere, plan D04), so pushing now turns master red — undoing this lane's 15:16 master-healing. The fix list, all in their trees:
   - root `transport/serve.go` (w03.1): gci formatting (line 467 region), gocognit 32>25 on `ServeDomainEvents` (418), wsl_v5 missing-whitespace ×2 (502, 507); `transport/serve_test.go` wsl_v5 (1115).
   - `auditlog/auditlog.go:64` (w03.3): wrapcheck — the `Viewer.Shutdown(ctx)` passthrough returns an external error unwrapped.
   - usermgmt (w03.6): `oauth2.go:64` exhaustruct (`oauth2StateEntry` missing `nonce`); `service_oauth2_extracted.go:113` golines (the `BeginLoginWithNonce` signature line).
   Unblock sequence (mine, the moment their fix lands): re-run `.#lint` → rc=0 → push the 14-commit stack through the strict pre-push gates → `gh run watch` → `.#verify-remote-tags` → append the push receipt here or in the sibling's next report.

## c) NOT STARTED

- The push + CI watch + tag audit (blocked on b1, sequence written).
- Whole-Wave-D closure (sibling lanes W04–W15, W17-final receipt + CURRENT_FOCUS handoff) — their plan of record.

## d) TOTALLY FUCKED UP / went sideways (recovered or flagged)

1. **The PIPESTATUS trap bit me AGAIN despite the documented lesson** — the first battery-test invocation piped to `tail` and rendered `TEST_RC=` (empty). Caught it immediately (an empty rc is not a verdict), re-ran with redirect + inline `$?`. The pattern that works is now enforced by my own error history twice; d1 of the 15-16 report said it, e1 below rewrites it into the muscle.
2. **Daemon race #3:** the s.1–s.4 edits were banked as three `chore: auto-commit`s during the 20s jstest + Go-test verification window before my atomic commit. Clean separation this time (no sibling files mixed), recovered by `git reset --soft HEAD~3` + re-commit as `5e0fb368`. Protocol holds; the window to close is "verification runtime before commit".
3. **Q3's framing was wrong (load-timing), the blocker is content:** the battery ran fine under load 17–26; what it found is a real lint red. Honest escalation changed shape — not "when to run" but "whose fix gates the push".

## e) WHAT WE SHOULD IMPROVE (process)

1. **Commit at quick-green, battery on the committed state.** Long verifications are daemon windows; the atomic-commit rule needs the commit to precede any multi-minute gate run.
2. **A two-session tree makes the battery a shared good:** running MY lane-closure battery produced the sibling's missing W12 lint leg and resolved their coverage worry for free. Run it at every lane boundary, not only at W12.
3. **Verdict capture: redirect + `$?`, always.** PIPESTATUS is unreliable in this shell; stop re-learning it.

## f) NEXT (ordered; "mine" unless marked)

1. Sibling lane: fix the 7 lint findings (b1 inventory) + their still-missing usermgmt/CHANGELOG S39 receipt (observed absent at 15:40).
2. Mine, on their fix landing: `.#lint` rc=0 → push 14 commits → CI watch → `.#verify-remote-tags` → push receipt.
3. g1 verdict (branching-flow, now +22) — sibling fixes or owner re-pin with dated ledger row.
4. g2 verdict (nightly strict-lag treadmill workflow — packet §6.3; worst-lag-0 measured green).
5. AGENTS.md quick-ref verified post-s-tails: sync-assets row updated to v1.6.0; no other stale row found.
6. Packet follow-ups ON TICK (owner): D6 chain (cqrs-lint tag → fleet swap → CI gate), D14 httputil/server_timing tag, D17 gosec pin, U1/U2/E005 drafting batches, PapDashboard reply.
7. Watches: templ first release after 10-05; treefmt-nix#545; BF1 fixture pruning; /mnt/buildcache (69%); bench-spike at first load <6 window; loginpage coverage-gate re-pin (templ ep-4 quiet-window micro).
8. Post-closure (both lanes done): the 10-42 plan file gets its terminal DONE line; the sibling's W17-final receipt supersedes this one.

### v5-window restatements (parked rows, carried for the closure table)

| Thread | State (2026-10-25) | Gate to move |
| --- | --- | --- |
| V007 cluster 1 (usermgmt off v5-removed APIs) | 68 SQLViewStore-family findings; criterion re-verified unmet 2026-10-11 | Upstream declarative Hydrator in `system.New` |
| appkit as the setup server layer (ADR-0052) | v5-window; upstream master fix still unpushed (>82 commits) | Upstream appkit release + green master |
| `ProjectionLayer` removal | All prep done; accepted limitation documented | v5 removal bundle (or upstream checkpoint/DLQ option first) |
| v5 backwards auto-upgrade (M26–M28) | Codemod R1–R10 + v4-journal compat plan written | v5 window start |
| templ ep-4 / SidebarNav | SidebarNav re-verified against v1.24.0: `display.Drawer` still a nonce'd `showModal` IIFE — custom dark sidebar STAYS | Library ships a zero-JS mobile drawer |

## g) OWNER QUESTIONS (cannot be derived from the tree)

1. **Q1 (carried, number grew):** `branching-flow` is now +22 vs baseline (was +12 at 15:16), set-diff still entirely inside the sibling's W01/W03 surfaces; CI self-skips this gate. Sibling fixes in-lane, or first-checkpoint re-pin with a dated ledger row (packet §6.2 recommends MAY)?
2. **Q2 (carried):** treadmill — keep PR-time advisory only, or also add the nightly strict-lag 0 workflow (packet §6.3 recommends BOTH)?
3. **NEW — ratify the push-hold:** the 14-commit stack stays unpushed because 7 lint findings in the sibling's W03 code would turn master's CI red (b1). I hold until their lint is green — the same master-stays-green principle that justified the 13:26 healing push, applied in reverse. Alternatives on your instruction: push anyway (accept a red CI window), or direct the sibling's fix order.

**Done-when for this report's commit:** tail ≤2 (15-16 archived same-commit), corpus ratchet green with same-commit bump, links green, todo-hygiene green. Battery data above is the push-decision record.
