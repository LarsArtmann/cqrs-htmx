# Frontend Sync Follow-Through — Execution Session (T01 of the SUPERB Pareto Plan)

**Date:** 2026-10-09 20:06 CEST
**Scope:** This session only — executing plan tasks T01+ from `docs/planning/2026-10-09_14-49_SUPERB-frontend-sync-follow-through-pareto-plan.md`.
**Series:** episode 3 of the offline-sync series (episode 2 = `2026-10-09_05-17_frontend-sync-protocol-completion-and-self-review.md`).

---

## a) FULLY DONE (verified green)

1. **T01.1–T01.4: e2e sweep against the v1.5.0 client — FULL SUITE GREEN (70/70, rc=0) after two fixes.** The plan's stated "largest open risk" (backward-compat untested) was REAL: the first run failed 61/70 (ffmpeg missing — environment) and after that fix exposed a genuine v1.5.0 client bug (below). Final state: `nix run .#e2e` rc=0, all 4 v1.4-path sync specs green against the 1.5.0 worker.
2. **Environment fix (gotcha-12 class): Playwright ffmpeg-1013 provisioned + flake self-heal.** Cache held `ffmpeg-1011` while Playwright 1.64 wants `ffmpeg-1013`; every spec failed at `newPage` (video contexts require the ffmpeg binary). Fixed twice: (a) manual `bun x playwright install ffmpeg` into `/mnt/buildcache/playwright`; (b) **flake.nix e2e app now ALWAYS installs ffmpeg** — previously the install ran only when no system Chromium existed, but E2E_BROWSER_PATH covers only the browser, never ffmpeg, so Nix-Chromium machines silently shipped no ffmpeg. CHANGELOG `[Unreleased]` Fixed entry written.
3. **REAL BUG FOUND+FIXED: the 1.5.0 sync worker could not persist ANYTHING (ADR-0056 regression, would have shipped broken).** `sync/sync-worker.js` used `EVENTS_STORE`/`META_STORE`/`META_KEY` (14 references) but **never defined them** — every fresh-DB `onupgradeneeded` threw `ReferenceError: EVENTS_STORE is not defined`, the DB open died, and the worker silently degraded to an in-memory queue: no persistence, no cross-session recovery, no offline reads — for EVERY consumer. `node --check` (syntax-only) and all Go tests (never execute the JS) were blind to it; only browser truth caught it. Second latent bug fixed in the same edit: `DB_VERSION` stayed 1 while the upgrade handler creates the v2 stores — opening an EXISTING v1 database at version 1 never fires `onupgradeneeded`, so real 1.4.0→1.5.0 upgraders would have gotten no `events`/`meta` stores and every cache transaction would throw NotFoundError. Now `DB_VERSION = 2` with a comment explaining exactly that trap. Both fixed before any release — T01 did its job.
4. **Root-cause method (reusable):** page/worker console was silent (SharedWorker errors don't reach Playwright's page console), so diagnosis went: failing-values diff → IndexedDB probe (`indexedDB.databases()` = empty after boot) → blob-URL SharedWorker probe (IDB works in this environment, worker MIME fine) → **instrumented worker boot** (fetched worker source wrapped in a Blob with an `error` listener that reports the first uncaught error back through the port) → exact `ReferenceError` + line. Scratch spec deleted after use (commit `f6ed8bd6`).
5. **T01.6: `nix run .#check-templates` rc=0.**

## b) PARTIALLY DONE

6. **T01.5: `nix run .#lint` RED — but on two modules this session never touched.** Root module: 0 issues. Failures: (a) **identity-model ×2 exhaustruct** (`external_account.go:23`, `fold.go:190` — `ExternalAccount` literals missing `ExternalAccountCore`); (b) **systemadapter ×8 typecheck** — `systemscenario/v4` fails to compile against the LOCAL go-cqrs-lite checkout (`undefined: system.Clock`, `system.New` arity) — a concurrent session is mid-refactor in `/home/lars/projects/go-cqrs-lite` and the go.work replace exposes their in-flight state. Both look foreign (identity-model's findings are on published v4.12.2-era code; systemadapter's on the cross-repo replace). Not triaged to origin yet — deliberately not "fixed" blind (gotcha: never revert/patch concurrent sessions' work).
7. **T01.7 phase-boundary commit:** content landed via daemon heuristic commits (spec fix, worker fix, flake fix, debug-spec removal); descriptive-message amends queued into T09 (which exists for exactly this).

## c) NOT STARTED

8. T02 release train (g1 = minor → root **v4.14.0**; `[Unreleased]` carries sync protocol + ProjectionStatus constants + cursor-context fix + e2e tooling receipts; **blocked** on lint green / concurrent go-cqrs-lite settling — see d/g).
9. T03 e2e harness for pull/push paths; T04 new-path Playwright specs (T04.5's DB v1→v2 migration spec just became MORE valuable — the migration path was broken until today's fix).
10. T05–T18 per plan (hardening, docs polish, ledger hygiene, upstream packet, sync-demo, OpenAPI, bench, gated items, ROADMAP harvest, curation).

## d) TOTALLY FUCKED UP (honest)

11. **One wrong early diagnosis, corrected by evidence:** I first attributed the spec failures to "harness drift" (spec pins `indexedDB.open(name, 1)` vs worker DB version) and made the spec helpers version-agnostic. That edit turned out necessary-but-insufficient: the worker was actually still at DB version 1 (the bug!), so no VersionError ever occurred — the real cause was the missing constants. The version-agnostic helpers are still correct (required now that DB_VERSION=2) but my initial confidence was misplaced. Lesson: same as the UserFromContext one — verify the mechanism, not the coincidence.
12. **Wasted a full e2e cycle on a stale server:** re-ran the suite after editing the worker, forgot the e2e server embeds the JS at COMPILE time — the run tested the old worker. 4 specs still red → rebuilt → green. (Cheap, but avoidable: embed = compile-time.)
13. **First commit attempt blocked by hook under load** (samber-linter context-deadline + test-compile spawn-hang — machine-load class per gotcha 8); the daemon landed the content minutes later. No damage, but I should have used `preflight-tree-check` discipline before the boundary commit anyway.

## e) WHAT WE SHOULD IMPROVE

14. **JS assets have no execution-path test below the browser.** Two consecutive sessions shipped worker bugs (encoding lie → this session; missing constants → last session) that every Go gate was structurally blind to. The plan's T06.2 node-harness idea should also pin the "worker boots without ReferenceError + DB upgrade succeeds" path (a tiny fake-`self`+fake-`indexedDB` smoke would have caught both classes); until then, `nix run .#e2e` is the only real pin — which is exactly why it must run before every sync-asset release (plan g2).
15. **The pre-commit hook's `--no-verify` escape hatch is now well-gated but load-sensitive:** deadline-exceeded failures under concurrent-session load cost a retry cycle; consider documenting "retry once after `wait-tree-quiet`" as the canonical response (it's in gotcha 8 already — this session is just another data point).

## f) NEXT (priority order)

1. Triage the two foreign lint reds: confirm identity-model exhaustruct origin (pre-existing vs concurrent session; if trivial and on published code, fix on sight per doctrine) and wait for / coordinate with the go-cqrs-lite session (systemadapter typecheck should self-heal when their refactor lands or the replace re-pins).
2. T02 release train once lint is green: `train-preflight` → `verify-tag.sh . v4.14.0 --push` → `check-release-train --refresh-cache` → consumer sweep (`bump-dep 'larsartmann/cqrs-htmx/v4$'`) — expect health module's consumer bump to unlock its ProjectionStatus train (its CHANGELOG entry is waiting on exactly this tag).
3. T03+T04 (harness + specs; DB migration spec first-class).
4. T05–T18 per the plan's DAG.

## g) QUESTIONS (cannot figure out myself)

1. **Foreign lint reds — fix or wait?** identity-model's 2 exhaustruct findings are on published module code I didn't touch this session; the fix is trivial (add the embedded core field to two literals) but the origin is a concurrent/prior session's change. The systemadapter typecheck red depends entirely on the go-cqrs-lite session's in-flight state. My default: fix the former (published-code correctness, on-sight doctrine), wait on the latter. Confirm or override.
2. **Release timing under a concurrent cross-repo train:** if go-cqrs-lite is mid-release, cutting root v4.14.0 now risks version-drift churn on the replace pins. Alternative: cut the tag but defer the consumer sweep until their train settles. Owner call.

---

**Bottom line:** T01 found exactly what it existed to find — the 1.5.0 worker was fundamentally broken (missing constants + wrong DB version; nothing persisted, upgraders would break too) and is now fixed with the full 70/70 e2e as proof. Lint red on two foreign modules is the only thing standing between this session and the v4.14.0 release train.
