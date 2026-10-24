# Tag Wave Staged, Sync E2E Green, Two v1.5.0 Sync Defects Fixed — Session Status

> **ANNOTATED 2026-10-24** — ~~this report~~ **SUPERSEDED** by [`2026-10-12_05-23_train-lag-unblock-atomic-wave-ci-repair.md`](../2026-10-12_05-23_train-lag-unblock-atomic-wave-ci-repair.md) (same series, later in the
> session). The successor re-covers every section with the outcome, so nothing here is a
> unique open item. Strike depth: supersession verdict (round-18 precedent for
> series-internal reports). **ARCHIVED 2026-10-24 (round 22).**


**Date:** 2026-10-11 10:58 CEST
**Scope:** resuming from `2026-10-11_06-39_postwave-alignment-push-ci-fix-toolchain-battery_status.md` (same SUPERB plan). This session: owner answered the 3 parked questions → executed the release chain to the wave boundary.
**Baseline at start:** HEAD `3377b818`, tree clean, v4.14.0 tag NOT cut (owner Q1 pending).
**Tree at writing:** HEAD `d06b17d8`, clean, pushed; CI running (expect green — the two red classes from the previous push are fixed in `d06b17d8`). Family wave STAGED (receipts cut) but tags NOT cut — blocked on the foreign go-cqrs-lite session (details §b).

---

## a) FULLY DONE

| Item | Evidence |
| --- | --- |
| **Owner decisions landed** (06-39 report §g): (1) cut v4.14.0 — YES; (2) train cadence — WAVE-ordered, one sitting, after e2e green; (3) e2e rotate endpoint — KEEP + document test-only | question-tool answers, 09:05 |
| **root v4.14.0 TAGGED + PUSHED** — verify-tag dry-run (content assertions + message guard) → `git push origin v4.14.0` → ls-remote `bfe86081` | tag on origin |
| **go-sse/ssetest v0.4.0→v0.5.0 alignment** — strict gate caught the fresh cross-repo lag (gotcha 27e live); tag range audited FIRST (breaking `WithLastEventID(string→sse.EventID)` — zero call sites here; transitive floors already satisfied); swept root+auditlog | `a81a3f3b` |
| **Root-require sweep to v4.14.0** — all consumers (adminui, dashboardui, e2e/server, examples ×12, health, integration_test, loginpage, setup, systemadapter, usermgmt), bump-dep PASS × all, strict gate back to 0/0/0; daemon mid-sweep capture squashed into one clean commit | `cdf84122` |
| **P6 Sync E2E — DONE, 73/73 GREEN** (4 existing + 3 new specs). The run FOUND AND FIXED two real defects in the shipped sync assets v1.5.0 (see §a last rows) plus two spec-side bugs (§d) and one harness bug (§d) | battery logs (5 runs: 71→71→72→72→73), `8c29f147` |
| **Sync defect 1 — worker keyless IDB put (DataError)**: `metaStore.put(value)` on the keyPath-less META_STORE throws synchronously; the catch swallowed it while queued event puts still auto-committed → events cached, cursor/backendId NEVER persisted (every pull re-read the whole journal; reset detection had nothing to compare). Fix: `metaStore.put(value, META_KEY)` + explanatory comment | `ff36c975`; live-proven via debug spec (cursor+backendId persisted, sync:events broadcasts firing) |
| **Sync defect 2 — client drops caught-up updates**: `pullEvents` only posted `cache-events` when new events or a moved cursor existed — exactly the backend-rebuild shape (0 new events, CHANGED backendId) → `sync:reset` never fired, stale cache survived forever. Fix: also post when `backendId !== state.backendId` | `ff36c975`; pinned by the rotate spec |
| **P5 Health rider — CODE DONE**: mirrors at `health/probe.go` deleted, `checkError` switches to `cqrshtmx.ProjectionStatusLive/Stopped/Failed`, comment updated; hermetic GOWORK=off build+vet+race GREEN. (The `RecorderChain` integration_test half still waits for health's tag — gotcha 37) | `31d4e009` |
| **e2e harness floor-proofed**: `playwright.config.ts` derived `GOTOOLCHAIN` from go.work's `go` directive — the stale `go1.27.1` literal (missed by the floor sweep) killed BOTH webServers (`go.work requires go >= 1.27.2`); now single-source-of-truth | `8c29f147` |
| **Q3 follow-through**: rotate-backend endpoint documented TEST-ONLY in the file header comment (never ship in a consumer app) | `8c29f147` |
| **CI red triage + fix**: previous push's CI (run 38121454196) red'd 3 jobs — (a) `lint (usermgmt)` + `lint (datastar)`: golangci **v2.13.2 cannot decode Go 1.27.2 export data** ("version 5 > maximum supported 4"), surfaced only when the require sweep invalidated the result cache; CI pin → **v2.14.0** (the local fleet build parses 1.27.2 — verified empirically); (b) `checks / Markdown link check`: 7 links — archive-move relative-path rot from round-20's P9 (planning links need `../../`, sibling status links need `../` from `archived/`); fixed, local checker 443 links green | `d06b17d8` |
| **Wave receipts CUT for all 14 tags** (CHANGELOG-before-tag): root v4.14.1 section (sync fixes + 1.27.2 floor); per-module sections for identity-model v4.12.3, usermgmt v4.14.3, totp/webauthn v4.12.1, oauth2 **v4.13.0** (minor: public PKCE clients + preferred_username + FinishLoginWithToken), datastar v4.12.2, loginpage v4.12.3 (+node:test JS suite receipt), dashboardui v4.13.3, adminui v4.12.4 (retro ServeAsset/ETag receipts), systemadapter v4.12.4 (retro M22/M23 + systemscenario migration); central receipts for setup v4.14.3 / health **v4.13.0** (RecorderChain receipt already under v4.14.0's heading — module-CHANGELOG-less convention, 2026-10-08 precedent); wave-summary line under [Unreleased]. All structures validated (Unreleased first, no dupes) | `a4f28aae` |
| **Version-pick audit done per module** — tag dates vs receipt content checked (adminui/systemadapter receipts are RETRO for content already inside v4.12.3, precedent: usermgmt v4.14.2 retro receipt; oauth2's PKCE features are genuinely post-v4.12.0 (tag 2026-09-22) → minor) | session analysis |
| **06-39 report gate-verified** (row gate + annotation gate green over 470 files) — was written but unverified at park | gate outputs 09:06 |
| Housekeeping: `check-cqrs-lint.sh` stale goPkg comment de-versioned; remaining 1.27.1 literals audited (selftest fixtures are deliberate; flake/doc mentions are dated history) | tree |

## b) PARTIALLY DONE — the family wave (STAGED at the tag boundary, NOT cut)

| State | Detail |
| --- | --- |
| **Ready** | Receipts committed (`a4f28aae`) and pushed; versions picked (14 tags); wave order defined per playbook §4a: identity-model → totp/webauthn/oauth2/datastar/auditlog (leaves) → root v4.14.1 → usermgmt → loginpage/dashboardui/adminui/health/systemadapter → setup → final consumer sweep (root v4.14.1 everywhere + per-module) → strict gate → push. No replaces anywhere (systemadapter's filesystem replace is gone since systemscenario v4.0.0 published — gotcha 36 resolved). |
| **BLOCKED** | `train-preflight` lint stage reds in WORKSPACE mode: usermgmt typecheck fails against the go-cqrs-lite workspace replace — the foreign session's in-flight `dedup.NewRing[K]` generics change requires explicit instantiation (`NewRing[string](cap)`), and our `es_dedup_benchmark_test.go:29` bare call no longer infers. Hermetic GOWORK=off build+vet GREEN (published dedup v4.2.4 unaffected). Per gotcha 4/36 doctrine: do NOT touch the foreign checkout mid-flight; fixing our call site NOW would break hermetic builds until the new dedup publishes (explicit instantiation is invalid against the old non-generic… signature). The wave cuts the moment the foreign session settles/publishes dedup — then: adapt the bench call in the same sweep, preflight, 14× verify-tag in order, final sweep, push. |

## c) NOT STARTED (this session)

P5's `RecorderChain` integration_test half (needs health v4.13.0 published — it is IN this wave, so immediately after), P19 (systemscenario narrative), P11 (sync OpenAPI), P10 (sync-demo), P12/P13 (v5 codemod A / goldens B), P14 (loginpage Playwright E2E), P8/P15/P16 (GCL coordination), P20 (owner sheet — the 3 parked questions are now ANSWERED, fold in), P21 (D4/D6 tooling), P22/P23 (watch sweeps), round-21 full-tail archive (26 live reports).

## d) TOTALLY FUCKED UP!

1. **The stale-server debug trap (a full debug cycle wasted):** after editing the worker asset, my "restart" (`pkill -f "go run \."` + new `go run`) silently FAILED — the pkill killed the `go run` parent but not its built child holding :18923, the new server died with `bind: address already in use`, and Playwright's `reuseExistingServer` happily reused the STALE one — so the fix "didn't work" and I nearly re-diagnosed the wrong code. Lesson: after any server restart, verify the SERVED BYTES (fetch the asset and grep for your change), never trust process logs or port liveness alone.
2. **Spec bug, iteration 2 (after the handoff's noted iteration 1):** `page.evaluate(cqrsSync)` serialized the API OBJECT — functions do not survive structured clone, so `getEvents` arrived as undefined. The fix pattern (inline async expression returning plain data) was already in the file as `CACHED_EVENTS`; I should have reused it for every page-side API touch.
3. **Race in my first poll design:** asserted cache contents immediately after the status indicator turned "Synced" — but the indicator reflects the push/SSE connection, NOT the boot-time catch-up pull. Fixed with `expect.poll` on the cache itself; the wait-for-indicator-then-assert pattern is only safe for what the indicator actually gates.
4. **rc-capture confusion on the lint app:** printed `rc=0` from one shell while the log's tail said FAIL — I chased the mismatch for two tool calls before running per-module golangci with explicit rc capture (found the usermgmt typecheck). Trust per-module rc over aggregated app rc when the app's verdict line looks inconsistent.

## e) WHAT WE SHOULD IMPROVE!

1. **A node:test seam for the sync worker/client JS** (loginpage's P7 pattern): both v1.5.0 defects were pure-JS logic bugs (keyless IDB put; conditional update drop) that only Playwright caught — 5 battery runs. A `fake-indexeddb`-backed node:test suite for `cacheEvents`/`pullEvents` would catch this class in seconds. Candidate: P10/P11 time-boxed rider.
2. **The archive-move link-rot class is mechanizable:** round-20's archive step moved a report without rewriting its relative links; the link checker caught it only on the NEXT CI run. Cheap fix: `git mv` to archived/ should trigger a link fix-pass (or the checker gains a pre-commit scope for moved files).
3. **The toolchain-floor touch-list gains one more entry** (06-39 e-1): `e2e/playwright.config.ts` had a version literal; now derived from go.work — but the runbook page still doesn't exist. When written, include "grep the repo for the old floor literal, excluding dated history + selftest fixtures" as a step (this session's audit found exactly one live straggler).
4. **Served-bytes verification in the e2e debug loop** (§d-1): a one-liner habit — `fetch /sync-worker.js | grep <new marker>` after every server rebuild.

## f) Next things (ranked by impact)

1. **Watch go-cqrs-lite settlement** (the dedup generics commit is daemon-carried there) → when published: adapt `es_dedup_benchmark_test.go:29` to explicit instantiation IN THE SWEEP → `nix run .#train-preflight` → cut the 14-tag wave in §b's order (verify-tag dry-run → --push each) → refresh tag cache → final consumer sweep → strict gate 0/0/0 → push → CI green.
2. **P5 completion**: after health v4.13.0 publishes, add the `RecorderChain(Recorder, auditPlugin)` integration_test half (gotcha 37 satisfied).
3. **P19** systemscenario narrative (facts: gotcha 36 + the replace-strip in this wave's systemadapter receipt).
4. **P20** owner-call sheet — fold in the three answered questions (tag YES / wave cadence / keep rotate endpoint).
5. P11 sync OpenAPI → P10 sync-demo (with §e-1's JS test seam rider) → P12/P13 v5 tracks → P14 loginpage E2E.
6. P8/P15/P16 GCL coordination; P21/P22/P23; round-21 archive (26 live reports).
7. CI watch for `d06b17d8` (expect green; if the lint jobs still red with v2.14.0, the export-data claim needs re-verification against the action's binary provenance).

## g) Questions I can NOT answer myself

None new. The three from 06-39 §g are answered and executed (§a row 1). The wave's remaining blocker (§b) is a foreign-session timing question, not an owner decision — the wave proceeds automatically on settlement.

---

**Truth anchors:** tag `v4.14.0` = `bfe86081` on origin (ls-remote verified); strict gate 846 requires 0/0/0 after each sweep (`a81a3f3b`, `cdf84122`); e2e battery 73/73 (6th run, `/tmp/cqrs-e2e6-*`); health hermetic build+vet+race rc=0; link checker 443 green; pushed HEAD `d06b17d8` through strict pre-push gates; receipts commit `a4f28aae`; previous CI red run 38121454196 (3 jobs, all root-caused + fixed in `d06b17d8`).
