# Status — SUPERB hardening Phase 1 complete (both repos), Phase 2 mid-flight, hostile-tree lessons (2026-10-06 evening, session 2)

**Session window:** 2026-10-06 ~19:40 → 22:34 · **Repos:** cqrs-htmx (CH) + go-cqrs-lite (GCL) · **Machine:** load 19–292 all session (foreign rustc cross-compiles + go jobs), 74Gi RAM available, `/mnt/buildcache` warm after the stalled-tidy kill.
**Assignment:** Pareto plan `docs/planning/2026-10-06_14-49_SUPERB-dashboardui-metaengine-system-hardening-pareto-plan.md` — my lane M01–M10; foreign session's lane M11–M15 (now complete on their side).

## a) FULLY DONE (this session)

> **ANNOTATED 2026-10-06 (docs-health round 18)** — Phase 2 closed by the 23:20 completion report: **M06 finished** (mid-edit content had survived as `7584b7a84`; completed + tested at 3 levels), M07 (`77076688b`), M08 (`9ce1367c4`), M10 (`5688d848f`) all landed + receipted; per-lane final battery green both repos. Struck below: §b1, §c M07–M10 row, §f1–6, §g2/§g3. STILL OPEN: the flightrecorder v0.2.1 lag train + push (§c-last/§f7/§g1 — the standing blocker, TODO_LIST), M16–M26 unscheduled, full-gate battery (check-modules/coverage) at the next quiet window (§b2), §e1 GCL concurrent-writer protocol + §e2 api_surface pre-emption (GCL AGENTS candidates).

1. **Alignment pass COMPLETED** (was ~70%): killed the stalled 49-min tidy (1s CPU — futex-waiting on a VCS cache lock; SIGKILL via `/run/current-system/sw/bin/kill` — mvdan/sh's builtin kill rejects `-9`), cleared the stale `shallow.lock` my kill left, reran tidy (rc=0), committed `integration_test/go.sum` churn, hermetically verified BOTH touched modules (`GOWORK=off` build+vet green), and passed the absence assertion (zero `id/v4 v4.7.0` / `scheduling/v4 v4.6.0` requires repo-wide).
2. **M01 — SSE XSS (dashboardui/layout.go + layout_sse_xss_test.go):** row builder rewritten with createElement/textContent (zero innerHTML in the served JS), href percent-encodes eventId, Stream Type cell emitted (5-column alignment — the wire envelope already carried streamType; no golden bump), `eventCount` resets on `htmx:afterSwap`, `console.warn` replaces the silent `catch{}`. 6 tests green incl. served-JS wire check + hostile-payload smoke (`<img onerror>`, `javascript:`-scheme href neutralization).
3. **M02 — AccentColor (accent_color.go + config.go):** closed-grammar validator (#hex 3/4/6/8, rgb()/rgba()/hsl()/hsla() numeric-only, 147 named colors + transparent); `New` rejects breakouts (`red;} body{...}`) with a Rejection-family error; doc note replaces the "any CSS color value" lie; 4 tests + goldens green; all in-repo consumers already use hex.
4. **M03 — CSV formula injection (export.go + export_formula_test.go):** every cell through `writeCSV`'s neutralizer — leading `=`,`+`,`-`,`@`,TAB,CR gets the OWASP single-quote prefix; hostile event-type export test pins quoted and unquoted forms.
5. **M09 — publish fan-out (GCL system):** `len(Publish)==1` now fans out like N (PublisherFor resolves; local bus stays entry 0); undeclared targets fail `New` with new `ErrUnknownPublishTarget`; `TestSystem_PublisherFor` updated to declare buses (it pinned the bug); 6 tests + full system suite green; api_surface.txt updated (the hook enforces it — my commit was initially rejected with "7550 expected, 7551 actual").
6. **M04 — FilterOp/jsonPath (GCL metaengine + 3 engines):** `ValidateFilterSpecs` (uses the PRE-EXISTING `AllFilterOps` registry — my first duplicate `Valid()` collided with `enum_validation.go:77`; head-truncated grep missed it) + shared `ValidateIdentifier` (extracted from matview's `validateMatViewString`, F13) run at Scan entry AND in every planned/standard/stream query builder (pg, mysql, sqlite×3 paths; sqlite builders gained error returns). Hostile op/column tests at both layers; 5 module suites green (metaengine, sqlite/pg/mysql engines, system).
7. **M05 — memory index races (GCL):** RWMutexes on `MemoryVectorIndex`/`MemorySpatialIndex`/`MemorySearchIndex` (the memory engine delegates WITHOUT holding its own mu — no parent-lock contract existed); `*Locked` helper renames with contracts; new `adttest.AssertConcurrentVectorInsert` + `AssertConcurrentScanDuringWrite`; memory suite green under `-race`.
8. **Receipts:** dashboardui CHANGELOG (M01–M03), GCL CHANGELOG (M09, M04), TODO_LIST annotations (golden item DONE with tag-level evidence; push item updated with the flightrecorder-lag blocker), api-surface updates in both directions.

## b) PARTIALLY DONE

~~1. **M06 — keyset ties (GCL): ROOT-CAUSED and MID-EDIT.** The bug: `SortPaginate` SORTS with the compound (sortValue, byte-key) order but the CURSOR FILTER compares sort value only — every item TYING the cursor value is skipped (`<= 0`), silently DROPPING unseen rows (the cursor protocol is value-only: `Cursor{Value: lastItemSortField}`). Fix in flight: exported `SortKeyCursor{Sort any, Key []byte}` + compound filter (ties kept iff key > cursorKey; legacy raw-value cursors keep the old semantics, documented tie-lossy). `metaengine/sort_paginate.go` is EDITED but NOT built/tested/committed (report interrupted the phase). REMAINING: build+vet, F21 tie-heavy regression test (no drops, no dupes), consider ScanResult.NextCursor issuance (documented as follow-up if skipped), commit.~~ done — content survived (daemon `7584b7a84`); completed + tie-heavy tests at 3 levels (23:20 report)
2. **Full battery (M27-class):** not run yet — the tree kept moving (foreign session + daemon) all evening; per-module suites are green everywhere I touched.

## c) NOT STARTED

~~- M07 (New engine-leak cleanup + goleak), M08 (timers lifecycle), M10 (unknown InstanceRole + engine-name hints) — Phase 2 remainder.~~ done — 23:20 session (`77076688b`, `9ce1367c4`/`143cbd589`, `5688d848f`)

- M16–M26 (Phase 3) per plan ordering.
- The alignment push itself (owner-gated) + the flightrecorder v0.2.1 bump train (16 lagging requires appeared DURING this session — someone published it today; the strict gate will block the push until that train runs).

## d) TOTALLY FUCKED UP (honest)

1. **A hostile concurrent writer in GCL ate a full edit batch.** My first M09 pass (errors.go, bus.go, constructor.go, named_bus_test.go content) reported successful edits, then VANISHED — files back to old content (bus.go mtime even read "Aug 18"); only my LAST edit (imports) survived on the restored file. Vet output proved the code files were already old at compile time. Root cause not identified (no git-restore process caught; no commits captured the content — likely the concurrent session's tooling doing a tree reset). Countermeasure that WORKED: re-apply + verify-by-grep + COMMIT WITHIN SECONDS. No further losses after adopting it.
2. **The daemon out-raced me on 5 of 7 code commits** (M03, M04×3, M05): authored messages became heuristic `chore: auto-commit` commits; the description trail survives only in the test commits + CHANGELOG receipts. Accepted — content preservation beats message aesthetics under this daemon cadence; amend was ruled out (foreign commits interleave; rebase under concurrent writers is the documented hazard).
3. **Two wrong test assertions of my own:** `javascript:alert(1)` claimed as HTML-escapable (it has no HTML-special chars — the neutralization lives in the href's encodeURIComponent; test fixed to assert THAT) and a `--` expectation against an allowlist that deliberately permits hyphens (fixed). Also two arg-count assertions in the sqlite tests (limit/collection args I forgot).
4. **Process error that cost ~20 min:** committed M04 without noticing it exports two new symbols → the GCL api-surface hook rejected the commit (fine) — but I then updated the surface and committed, and only LATER noticed the daemon had landed the code without the surface update anyway (redundant chase). Lesson: after ANY exported-symbol change in GCL, regenerate api_surface.txt in the same breath.

## e) WHAT WE SHOULD IMPROVE

1. **GCL concurrent-writer defense is now load-bearing:** "edit → grep-verify → commit within seconds" should go into GCL's AGENTS.md as the protocol for any session working alongside the daemon + foreign sessions (the M11/M12 session saw HEAD-lock races; mine saw full content reverts — same threat class, worse outcome).
2. **api_surface.txt hygiene:** the hook is right to block; the fix command (`cd cmd/api-stability && GOWORK=off go run -tags 'goexperiment.jsonv2' . --update`) should be pre-emptive on every exported-symbol change.
3. **mvdan/sh kill trivia for the memory:** builtin `kill` rejects `-9`/`-s KILL`; use `/run/current-system/sw/bin/kill` (util-linux) — cost me three confusing rounds against a futex-stalled tidy.
4. **The flightrecorder v0.2.1 train is the new push-blocker** — needs a bump sweep across 16 requires (adminui, dashboardui, e2e, examples×8, integration_test, loginpage, setup, systemadapter, usermgmt) before the strict release-train gate goes green.

## f) NEXT — ordered

~~1. Finish M06: build+vet sort_paginate.go; F21 tie-heavy regression (30 items, 6 distinct sort values, page size 5: legacy cursor drops, SortKeyCursor loses neither rows nor dupes); commit + CHANGELOG receipt.~~ done — 23:20 session
~~2. M07: `fail(err)` cleanup helper in system/constructor.go (12+ error returns after engine creation) + goleak test (`go.uber.org/goleak` already in GCL's graph).~~ done — `77076688b` + receipt `1336857e8`
~~3. M08: `Close` calls `stopTimers` + WaitGroup wait; doc-comment fixes; lifecycle test.~~ done — `9ce1367c4`/`143cbd589` + receipt `9477b865e`
~~4. M10: reject unknown InstanceRole at construction; SCREAM advisory on implicit memory SOT fallback; `ErrUnknownEngine` lists configured names + registered drivers.~~ done — authored `5688d848f` + api_surface 7557
~~5. Final battery: GCL touched-module suites + CH workspace build/test (local go.work replaces see all GCL changes — integration risk is real); `nix run .#lint` scoped; preflight-tree-check before any tree-wide run.~~ done per-lane — 23:20 final battery green both repos; full-gate battery (check-modules/coverage) still owed at a quiet window
~~6. CHANGELOG receipts for M05–M10 (GCL) as they land.~~ done — GCL CHANGELOG ×4 + dashboardui receipts
7. Owner lane: flightrecorder train + push (both repos carry unpushed trains: CH ~30 commits, GCL 24+).

## g) Questions I CANNOT answer myself

1. **Push (carried from the prior report):** gates are per-module green but the strict release-train gate now trips on the NEW flightrecorder lag — do you want me to run that bump train (16 requires, `bump-dep.sh` is the right tool for a real train) and then push, or does the owner take the push lane entirely?
   ~~2. **SortKeyCursor issuance (M06 scope):** the compound cursor fixes the FILTER, but today only value-only cursors are ISSUED (`ScanPage` derives them without the KV key). Full issuance needs `ScanResult.NextCursor` + the three KV engines filling it (protocol addition). Ship the filter fix + type now (consumers can construct compound cursors) and defer issuance, or do the full protocol in this session?~~ resolved — plan-faithful (F20+F21 only); issuance queued in GCL TODO_LIST with design notes
   ~~3. **Session continue-point:** M06 code is sitting UNCOMMITTED in the GCL tree (with the hostile-writer history) — I stopped per your report order. Resume by finishing M06, or do you want the tree state inspected first?~~ resolved — tree inspected first; the mid-edit content HAD survived (`7584b7a84`)

---

_Point-in-time snapshot; annotate, never rewrite. Auto-commit daemon will file this report._
