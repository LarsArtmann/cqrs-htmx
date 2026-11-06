# Status: Round-46 M10–M13 executed — cqrs-lint triaged at v4.20.0, gotchas 51–53, the L1 tail, and two flake/e2e repairs the plan did not name

> **ANNOTATED 2026-11-06** — ~~this report~~ **SUPERSEDED** by [`2026-11-06_02-44_fleet-plan-execution-waves-1-2-status.md`](../2026-11-06_02-44_fleet-plan-execution-waves-1-2-status.md), which round-46 M10-M13 receipts fully absorbed into AGENTS/TODO/CHANGELOG and the fleet-plan execution; superseded by the waves-1-2 execution report. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-06 (round-23 (2026-11-06)).**

**Session window:** 2026-11-05 ~09:52 → ~11:00 active work (report written 18:15 after a pause; tree advanced by daemon + a 17-40 sibling meanwhile)
**Plan of record:** `docs/planning/2026-11-05_03-11_SUPERB-round46-tail-train-green-and-backlog-pareto-plan.md` (M10–M13 lane)
**Author:** this session (Crush). Parallel lanes observed: the 09-53 docs-health round-47 session, and a 17-40 session (waved-harvest/loginredirect/authztest — their lane, untouched).
**Verdict:** M10 ✅ green gate. M11 ✅. M13 ✅ (one F-item declined with reason). M12 ~90% (B55/B58/B59/B60-sync proven; ONE e2e red remains, root-cause isolated). Two unplanned repairs landed: the templ-generator flake split brain (real CI-red hazard, closed) and a WCAG a11y fix.

---

## a) FULLY DONE (receipts)

1. **M10 — cqrs-lint triaged at the published `cmd/cqrs-lint/v4.20.0` tag; gate green 14/14** (commit `3c544dc9`). The system binary was a 4.19.1 pseudo build; `go install @version` is permission-blocked on this host, so the working recipe was `git archive` of the tag into /tmp + `go build -ldflags "-X main.commitHash=18eeb6f"` (version line reads `dev (commit: 18eeb6f, …)` — the Nix-anticipated stamp form). Rules diff 4.16→4.20: **C044** branded-ID `.String()` (warning), **E023** satellite-owned-SQL (info), **F032** no-catalog-import (ERROR), **T009** hand-booted-system (info, 4.20.0-only). Red inventory at v4.20.0 = 6 modules, not 8: 5× F032 (identity-model, usermgmt, dashboardui, datastar, systemadapter) + one stale S001 directive shared by root/identity-model walks. Adjudication: F032 is a consumer-adoption rule firing on a library — **disabled-with-reason per module `.cqrs-lint.json`** (3 created, 2 extended), with usermgmt + systemadapter carrying an explicit remove-when trigger (leverage-plan R2, now unblocked by the published `systemcatalog/v4.0.0`). Stale directive removed (`identity-model/events.go:171`, tool-marked safe-to-remove). **C044 sweep: every fired site is a display context or a self-consistent single-rendering key/cursor** (incl. the golden-pinned SSE envelope at `transport/event_sse.go:40` and usermgmt's ActorID map-key/SQL-view sites) — zero real bugs; kept as a non-blocking diagnostic. `nix run .#check-cqrs-lint`: all 14 green under `--strict --fail-on-stale-suppressions`. Receipts: AGENTS cqrs-lint row trued, TODO row 35 resolved + row 33 hygiene items struck, CHANGELOG [Unreleased] entry.
2. **M11 — gotchas 51, 52, 53 written + AGENTS index trued to 53** (commit `d75bc95a`). 51 = BuildFlow spawn starvation under load>30 (`BUILDFLOW_MAX_CONCURRENCY=1` on the FIRST commit attempt; 5 steps lost at load 35.7; daemon committed through the starved window — 17-28 §d1). 52 = phantom Tailwind token (`.shadow` harvested from the comment "would shadow every route"; reword-vs-rebuild decision table). 53 = fmt-scoping on shared trees (tree-wide `nix fmt` rewrote a parallel session's in-flight files — the 09-53 session's own §d2). Anchor count 53 verified; tail-budget strict + todo-hygiene green. TODO row 36 closed (all candidates written; the plan's 49/50 content had already landed as the sibling's 49/50).
3. **M13 — docs-truth tail** (commit `b7f32589` + daemon sweeps `310fa5a8`/`e0ef3302`; all content verified in HEAD): **F070** verified already-true (the Gates-row coverage note landed in `1464f60c`; gotcha 50(d) carries the nuance). **F071** `docs/research/README.md` now documents the `archived/` convention (in-file `ARCHIVED` blockquote naming the successor, links re-based, series table keeps the entry point) and the theme-toggle spike carries its marker. **F072** route-posture guide gains the owning-package paragraph — every claim source-verified (`dashboardui/routes.go:49` Posture field; `authztest.go:41` type alias). **F073** skill gates green (237 symbols resolve, 41 fences parse). **F074** event-catalog-guide's file list now points at `eventcatalog/{catalog,handler}.go` with the re-export names (`cqrshtmx.EventCatalog = eventcatalog.Catalog`). **F075 deliberately NOT implemented**: the links gate is a custom find-based script, green INCLUDING both archived dirs (1064 links) — excluding them would weaken the gate for zero present gain (the plan assumed lychee).
4. **Unplanned repair 1 — the templ-generator flake split brain, closed** (commits `2a5a9abf` restoration + `8710b96d` templGen). `nix run .#gen`/`.#check-codegen` wrapped `pkgs.templ` (v0.3.1020) while CI canonicalizes v0.3.1100 — a local regen downgraded all 19 `_templ.go` files and the daemon committed the damage; CI would have red'd. Restored by building templ v0.3.1100 from the module-cache zip (CI-canonical invocation per module), proven by a filtered `git diff` against the pre-downgrade commit showing EXACTLY the intended change. Durable fix: **`templGen`** — a benchstat-pattern buildGoModule pinned to the v0.3.1100 tag (`templ-hash.nix`/`templ-vendorHash.nix`), now backing `gen`, `check-codegen`, the devShell, and the ci shell; treefmt's fmt-only wrapper intentionally keeps pkgs.templ (gotcha 22 machinery). `nix run .#check-codegen` PASSED with updates=0 on the restored tree.
5. **Unplanned repair 2 — WCAG 2.5.3 label-in-name fix** (rides `2a5a9abf`): the dashboardui projections DLQ link's aria-label now contains its visible text (`DLQ (N), dead letters for X`), healing BOTH axe reds (light + dark) from the first post-train e2e measurement.
6. **Owner-questions check:** no answers found in-tree to the 07-24 §g questions → Q1 (pre-push hook) and Q2 (M06–M09 ownership) were left untouched and are re-asked below.

## b) PARTIALLY DONE

1. **M12 — L1 tail ~90%** (files in HEAD via daemon sweeps `e0ef3302`/`5d7349e5`/`b1f1322c`, content verified):
   - **B55** ✅ `TestAuditLog_ConcurrentAppendsUnderRace` (8 writers × 64 events, ring wraps at limit 64, 4 concurrent readers, count/dropped/distinct-aggregate invariants) — green under `-race`.
   - **B58/B59** ✅ `syncAddItemCmd` + `createNoteCmd` now embed `*command.BasicCommand` (A013: value-embed never satisfied `command.Command`); hermetic builds green, and **all 8 sync e2e specs green** — the changed command path proven end-to-end through a real browser (incl. offline flush → `/sync/push` delivery).
   - **B60** ~75%: sync-demo suite green; the full e2e battery went **71 passed / 1 failed / 1 skipped / 3 did-not-run** (first full post-train measurement since 2026-10-01). Fixed in-flight: axe ×2 (see a5), loginpage golden regenerated via `WRITE_GOLDEN=1 nix run .#e2e` after the diff was adjudicated as exactly the family-train render drift (autocorrect/autocapitalize/spellcheck attrs + spinner-span whitespace; golden landed in HEAD via daemon sweep, attrs verified present). Remaining: **one deterministic red** — `admin-behavior.spec.ts:126` "user menu dropdown: opens anchored" (menu opens `:popover-open` but sits at x=0; expects >500).
2. **E2E first-post-train drift classification (the 4 reds):** 3 of 4 fixed (axe ×2, golden); the dropdown is root-cause-isolated but unfixed: templ-components v1.27.0's Dropdown anchors via a singleton positioning SCRIPT (popover API + `getBoundingClientRect`, ADR-0017), and the captured error-context snapshot contains NO script tags — mid-diagnosis between "script not rendered (nonce/CSP)" and "script rendered but blocked". Investigation state preserved here; not chased further this session.

## c) NOT STARTED (this session's lane)

1. The final 76/76 e2e confirmation run (blocked on the dropdown fix).
2. The dropdown root-cause fix itself (vendor the positioning script path or fix the nonce wiring in adminui's layout).
3. The catalog-release watch never fired: the go-cqrs-lite sibling's M07 tag wave was "NOT STARTED" at their 09-46 report; no new `catalog/v4` tag existed at session end. The bump-when-tagged work (examples/catalog-demo + integration_test → CI 3 red jobs + composite 2 legs flip green → F020 receipt → TODO row 21 closure) stays armed.

## d) TOTALLY FUCKED UP (owned, brutally)

1. **I ran `nix run .#gen` without checking WHICH templ it wraps — and it downgraded 19 generated files, with the daemon committing the damage** (`c31d9465`, `b1f1322c`). Gotcha 10 documented the ambient-templ trap; I verified the trap's EXISTENCE and still invoked the sanctioned app, whose own generator pin was the stale one. A push in that window would have shipped a CI-red. Recovered same-session (restoration + templGen pin), but the check should have preceded the first invocation: `templ version` before `.#gen`, exactly like `cqrs-lint version` before lint work.
2. **Two wasted Playwright runs from a bare shell, including a 120 MB browser download that can never run on this host.** Run 1: missing browser → installed `chromium_headless_shell` to /mnt/buildcache. Run 2 (serialized): died with `libglib-2.0.so.0: cannot open shared object file` — NixOS has no FHS libs, so Playwright-downloaded Chromium is a documented dead end, and I then misread the earlier `kill ESRCH` noise as load-class spawn starvation before reading the actual browser log line. The canonical runner (`nix run .#e2e`, nix chromium via `E2E_BROWSER_PATH`) was in `e2e/README.md` the whole time.
3. **M11's gotchas commit grew the docs corpus +16 lines without the same-commit baseline bump** — caught at M13 when the ratchet red'd at +31; ledgered retroactively with per-task attribution in the M13 commit. The bump discipline (measure before commit) was skipped, not unknown.
4. **B55 test first draft used a nonexistent API and illegal assertions** (`sync.WaitGroup.Len()`; `t.Fatalf` from non-test goroutines) — caught by self-review before the first run, but both are basics.
5. **One burned toolchain build:** `GOTOOLCHAIN=local` against the ambient go 1.26.8 while building cqrs-lint (the floor is 1.27.2; go-cache-env's empty-GOTOOLCHAIN auto-resolution is the correct path). Small, avoidable.

## e) WHAT WE SHOULD IMPROVE (durable)

1. **Pinned-tool version asserts before every flake app invocation.** Three pins have now drifted under sessions (cqrs-lint, templ, and the ambient go): make "print the version the app wraps, compare to CI's pin" the standing pre-invocation ritual — and better, make the APPS assert it (check-codegen could refuse when `templ version` ≠ the CI pin; the templGen derivation makes drift structurally harder, an assert makes it loud).
2. **e2e/README should lead with `nix run .#e2e` on NixOS and name the bare-shell dead end** (downloaded Chromium lacks FHS libs; `libglib` is the signature). One warning line would have saved this session ~25 minutes and 120 MB.
3. **Corpus ratchet BEFORE docs commits, not at the next task** — run `check-docs-corpus-budget --ratchet` in the same tool-call sequence as the commit when the change touches docs/ line counts.
4. **Daemon-swept work needs a content-verification receipt in the NEXT commit's message**, not just a mental note — this session verified M12 files in HEAD but three of the four receipts live only in this report.

## f) NEXT (priority order; ★ = owner-gated)

1. Dropdown root cause: inspect adminui's rendered shell for the `popoverPositionScriptComponent` `<script nonce>` (CSP? empty Nonce?) → vendor/fix → re-run `tests/admin-behavior.spec.ts`.
2. Final full e2e run expecting 76/76; then the suite is re-baselined post-train.
3. ★ Catalog watch: on the go-cqrs-lite catalog tag → bump `examples/catalog-demo` + `integration_test` (`bump-dep` recipe) → CI build/test/module-architecture + composite module-isolation/workspace-build flip green together → post the PR-#40 battery receipt (F020) → close TODO row 21 + archive per docs-health.
4. R2: systemcatalog-derived `DefaultEventCatalog` (systemcatalog/v4.0.0 published 2026-11-05) — lands the real F032 fix-forward and retires two `.cqrs-lint.json` disables.
5. L2/L3 remainder (TODO row 33): A027 codec-once, P008 e2e batch size, F005 upcaster note, 135-suppression heuristic review, scorecard waivers with triggers.
6. ★ M06–M09 consumer waves (browser-history, InboxClean, SwettySwipperWeb, crm + two posture declarations) — ownership question below.
7. Post-repair verification battery: `nix run .#check-modules` (expect 86/88, both reds the catalog class), `.#check-branching-flow` (templ churn since the re-pin), `.#lint` on dashboardui/adminui (aria-label + embed changes).
8. `.#coverage-gate` re-run (M02 leg never re-measured post-train).
9. Tail pass: 5 live reports now — the next docs-health round should annotate + archive 06-41/07-24/09-53 (their open §g items are answered or restated here).
10. bench-spike re-pin at a quiet window (TODO row 54; load was 17–82 all session).
11. go-sse v0.5.1 straggler pin → treadmill sweep (09-53 §f8).
12. templ v0.3.1114 generator root-cause ask (09-53 §f6; templGen makes future bumps one-hash-file edits now).
13. `audit_max: 325` drain-or-scope posture (09-53 §g1).
14. Sibling 17-40 lane continuation: authztest `Suite` wiring into consumer suites + loginredirect verification (their report's §f).
15. erraudit-strict sweep re-baseline after today's Go changes (TOTAL should hold at ~303).
16. AGENTS S17 split (owner-parked; 73.5 KB → larger today).

## g) QUESTIONS I cannot figure out myself

1. **M06–M09 consumer-wave ownership (re-asked from 07-24 §g2):** the waves edit OTHER repos (browser-history, InboxClean, SwettySwipperWeb, crm). May cqrs-htmx sessions execute them, or do the still-active sibling sessions own those repos? (Two sessions were active on this tree today; the repos' own AGENTS files may claim lanes I cannot see.)
2. **Pre-push hook hermetic leg (re-asked from 07-24 §g1):** add the hermetic isolation/examples leg to `.githooks/pre-push`? It would have caught today's templ-downgrade window at push time — but it also blocks every push during documented-red windows (the catalog class today), which is exactly the trade you deferred twice.
3. **templGen standing pattern:** the in-tree buildGoModule pin (benchstat-style; two hash files to maintain per templ bump) vs bumping nixpkgs' templ via the flake's nixpkgs input vs upstreaming the version bump to nixpkgs. I chose the in-tree pin (self-contained, precedent); confirm or redirect.

---

**Standing state at report time:** tree clean, all work in HEAD (six daemon sweeps + five hand commits: `3c544dc9`, `d75bc95a`, `b7f32589`, `2a5a9abf`, `8710b96d`). Composite expectation unchanged: 86/88 green, both reds the upstream catalog class. CI red set unchanged (build/test/module-architecture on catalog-docserver templ poison). cqrs-lint gate green 14/14 at v4.20.0. Nothing pushed this session.
