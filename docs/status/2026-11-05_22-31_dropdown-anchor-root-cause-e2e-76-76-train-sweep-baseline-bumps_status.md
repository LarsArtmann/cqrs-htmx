# Dropdown anchor root cause, e2e 76/76, train sweep + baseline bumps — round-46 M12 closure session

**Date:** 2026-11-05 evening (session start ~18:42, report 22:31) · **Session:** continuation of the M10–M13 lane (prior report: [18-15](archived/2026-11-05_18-15_round46-m10-m13-cqrslint-triage-gotchas-l1-tail-templgen-repair_status.md)) · **Tree:** clean at `ecb65976`, nothing pushed.

**TL;DR:** The one deterministic e2e red (`admin-behavior.spec.ts:126`, dropdown popover positioning) is root-caused and healed — **e2e 76/76 green**. The root cause is NOT in this repo: templ-components v1.27.0's popover CSS-anchor layer is broken on Chromium (verified empirically on nix Chromium 156); adminui now vendors a corrected layer. The post-repair battery ran in full: branching-flow re-pinned 203→204, release-train strict back to 0 lag (go-health-dashboard v0.13.2 swept), corpus + root-files baselines trued, lint clean, coverage gate PASSED, and `check-modules` carries ZERO non-catalog reds. Owner questions remain unanswered; owner-gated scope untouched.

---

## a) FULLY DONE (this session, receipts verified)

1. **Dropdown popover root cause — found, proven, fixed, e2e 76/76.** The failing chain, established by reading the library source then probing the LIVE page and minimal repros against nix Chromium 156 (`E2E_BROWSER_PATH`, the flake app's resolution):
   - Since templ-components v1.27.0, `tcPopoverPosition` (the getBoundingClientRect JS singleton) **exits early** on engines where `CSS.supports("anchor-name: --x")` — deferring to the library `custom.css` anchor layer. adminui had only the ANIMATION CSS vendored, so the popover kept the UA default and parked at the viewport corner (test saw `box.x = 0`).
   - Vendoring the library's anchor block verbatim was NOT enough (first attempt, still red at `box.x = 8`): the library's own layer is broken. Empirical matrix on Chromium 156: the compound `position-area` values the library uses (`bottom span-inline-start/end`, `bottom start|end`, `span-x`) are **rejected by the engine** — the declaration drops and the popover falls to its static position; and `anchor()` **cannot see the implicit `popovertarget` invoker** (only explicit `anchor-name` declarations resolve; single-keyword bands like `position-area: bottom` are the only valid generic form, self-aligned anchor-centered by default; `justify-self` aligns within the CB-width band, not the anchor).
   - Fix: corrected vendored layer in `adminui/tailwind.css` (single-keyword bands per `data-tc-position` — valid syntax, safe default for any future anchored popover — plus explicit `anchor-name: --tc-admin-user-menu` / `position-anchor` refining the one known instance, `#admin-user-menu-*` align=end, to the documented flush-end; stylesheet-declared so strict CSP holds). Bundle rebuilt via `.#build-adminui-css`, both CSS gates green (class set exact, 1069 tokens).
   - Live verification: menu right edge = trigger right − 8px, top = trigger bottom + 8px. Full `nix run .#e2e`: **76 passed (1.3m)** — was 71/1 fail/1 skip/3 did-not-run.
   - Receipts: CHANGELOG [Unreleased] Fixed; AGENTS templ-components bullet ("engine-validate the VALUES, not just the @supports gate"); TODO upstream-asks row (f). Code landed via daemon sweep `889d0ab4` (content verified in HEAD: 51-line tailwind.css block + `--tc-admin-user-menu` in the compiled bundle).
2. **Post-repair battery — every leg green or trued:**
   - `.#check-css-bundles` + `.#check-css-bundle-classes`: green (both bundles canonical + class sets exact).
   - `.#check-branching-flow`: +1 new finding adjudicated and baseline re-pinned **203 → 204** (`bbd5d943`): `COMPOSITION_mixin examples/sync-demo/main.go:44` is the B59 pointer-embed fix itself — the structurally identical twin of the already-frozen `sync_types.go:19` (B58). Rejected under the standing command-plumbing mixin verdict; triage row appended to `docs/analysis/triage-decisions.md`. Gate green after commit.
   - `.#check-release-train --refresh-cache --strict-lag 0`: **0 unpublished / 0 lag** after sweeping the freshly published `go-health-dashboard v0.13.2` into health + integration_test + examples/samar-do-demo (bump-dep exact-anchor sweep; the daemon landed the files — content verified by grep; health suite green, demo hermetic build+vet green; the sweep's lint leg failed ONLY on integration_test's vet via the documented `catalog/v4@v4.12.0` docserver templ poison — the known CI red class, not a regression).
   - Corpus baseline 44372 → 44373 and root-files baseline 31 → 33 (`ecb65976`): the docs growth from this session's receipts, and `templ-hash.nix`/`templ-vendorHash.nix` (the templGen derivation files from `8710b96d`, prior session) which had never been baselined. Ledger row in `docs/status/README.md`.
   - Lint: adminui + dashboardui `golangci-lint run` — 0 issues each.
   - `.#coverage-gate`: **PASSED** (all gated modules above threshold; first full re-measure post-train).
   - `.#check-modules` final: 4 red legs — module-isolation ×2, workspace-build, train-consumers-hermetic — **all one root cause**: the upstream `catalog/v4@v4.12.0` docserver templ poison (CI's unchanged red set). Zero non-catalog reds.
3. **Catalog watch (armed, checked):** `git ls-remote` — catalog/v4 still ends at v4.12.0; the upstream release has not shipped. No action possible from here.
4. **Owner-answers check:** TODO_LIST / CURRENT_FOCUS / tree scanned — no answers to the three gating questions (§g below). Owner-gated scope (M06–M09, pre-push hook leg, templGen pattern redirect) untouched, as the protocol requires.
5. **Housekeeping:** 9 throwaway Playwright probe scripts removed after use (the e2e dropdown spec IS the regression test); stray servers (admin-demo, e2e probe server) killed; tree left clean.

## b) PARTIALLY DONE

1. **templ-components upstream anchor-bug filing** — evidence is COMPLETE (repro matrix, corrected CSS, engine semantics table) and recorded in TODO row (f), but the ISSUE has not been filed: upstream filings are owner-gated here (W17 verdicts). The fix therefore exists only in adminui's vendored CSS; any OTHER consumer of templ-components v1.27.0 `Popover`/`HoverCard`/`Dropdown` (align ≠ default) silently gets viewport-corner popovers on Chromium 125+. In-tree exposure verified zero (only adminui renders anchored popovers); external consumers unassessed.
2. **CI un-red** — everything locally provable is green; the 3 CI red jobs (build/test/module-architecture) + the composite's 4 red legs all wait on the ONE upstream catalog release. The bump-and-watch recipe is staged in the TODO P1 row (close row + PR-#40 receipt after).

## c) NOT STARTED (deliberately — owner-gated or other lanes)

1. M06–M09 consumer waves (browser-history, InboxClean, SwettySwipperWeb, crm + posture declarations) — blocked on Q1.
2. Pre-push hook hermetic isolation leg — blocked on Q2 (new supporting evidence: today's templ-downgrade window is exactly the class it would have caught at push time).
3. templGen pattern confirmation (in-tree buildGoModule pin stands unless redirected — Q3's predecessor; folded into Q3's decision space).
4. The 18-15 report §f backlog beyond this session's lane: R2 systemcatalog-derived catalog, L2/L3 remainder (A027, P008, F005, 135-suppression tuning, scorecard waivers), tail archive pass, bench-spike re-pin, go-sse v0.5.1 straggler, templ v0.3.1114 root-cause, audit_max drain decision, sibling 17-40 lane continuation, AGENTS S17 split.

## d) TOTALLY FUCKED UP (process slips — all caught, all recovered, honestly listed)

1. **Blind first vendor:** I vendored the library's anchor block VERBATIM and ran a full e2e (~1.4m) before engine-validating the values — despite writing a comment claiming things I had not yet verified. The library's CSS was itself the bug. Cost: one wasted e2e cycle + an intermediate broken state swept by the daemon. The lesson is now the AGENTS bullet, but I should have probed first.
2. **Pipe-masked exit code on a gate verification:** `bash …check-root-files.sh --strict 2>&1 | tail -1 && git add …` — the pipe made the chain succeed on TAIL's rc (0) regardless of the gate's failure, and I committed a drifted (wrongly ordered) baseline. Caught by a follow-up plain run; fixed via the gate's own `--update-baseline` + commit amend. The `| tail` rc-masking class is a shell trap I know — I still walked into it.
3. **Baseline discipline two times over:** I authored CHANGELOG/TODO/AGENTS growth without bumping CORPUS_BASELINE in the same commit (rule known, in AGENTS), reddening the aggregate; and the templGen session (mine, prior) never baselined its two root files. Both trued in `ecb65976`, but a gate should never have had to tell me.
4. **bump-dep sweep shredded by the daemon anyway:** the recipe's whole point of `--commit` is ONE reviewable commit; the tool failed on integration_test's documented-red lint leg BEFORE committing, and the daemon shredded the sweep across `497d0cf6`/`21201f51`. End state verified (grep + health tests + demo build), but the reviewable-commit intent was lost — with a documented-red lint leg I should have pre-decided `--no-lint` for that module or expected the shred.
5. **Overwrote a prior session's untracked file:** my probe write clobbered the previous session's untracked `e2e/probe-explicit.mjs` (a stale mid-diagnosis probe) before flagging it. Low damage (it was superseded, all probes removed at end), but the "don't touch changes you didn't author" rule deserved a flag-first.

## e) WHAT WE SHOULD IMPROVE (session-derived, concrete)

1. **Engine-validate vendored CSS values, not just the @supports gate** — `@supports (anchor-name: --x)` passes on Chromium while the compound value syntax silently drops. Any future vendored-library CSS block should ship with a minimal live-repro probe FIRST (the 9-probe pattern this session used; ~2min each).
2. **A gate-output-rc guard for my own shell habits:** never chain `gate | tail && commit`; capture rc explicitly (`set -o pipefail` at minimum). Candidate gotcha 54.
3. **Baseline-bump checklist at AUTHOR time:** any edit under `docs/` → corpus ratchet; any new tracked root file → root-files baseline; any new status report → TAIL_BASELINE + corpus, same commit. I ran the aggregate to DISCOVER these instead of remembering them.
4. **bump-dep on a documented-red module:** pre-decide the lint waiver (`--no-lint` for that module) so the sweep can land as one commit, or explicitly accept the daemon shred + content-verify. Decide it BEFORE running, not after the tool fails.
5. **Upstream templ-components version check was shallow:** I checked the module cache's v1.27.1-pseudo for a fix but never ran `git ls-remote --tags` on templ-components for newer releases. TODO row (f) should be trued against the real tag list before any filing (maybe it is already fixed upstream and we simply pin v1.27.0).
6. **e2e receipt timing:** the 76/76 run predates the go-health-dashboard sweep and the baseline commits (both provably unrelated to e2e — no CSS/adminui changes after, no go-health-dashboard edge in the e2e webServers — but strictly the receipt is not "final tree"). A final one-command re-run would have made it airtight; do that next session start.

## f) NEXT (priority order; ★ = owner-gated)

1. Catalog watch: when go-cqrs-lite tags the next `catalog/v4` release, bump `examples/catalog-demo` + `integration_test` (`bump-dep` recipe) → CI's 3 red jobs + composite's 4 red legs flip together → post the PR-#40 aggregate-battery receipt (F020) → close the TODO P1 CI row → archive per docs-health.
2. ★ templ-components anchor-bug filing (Q3): verify the upstream tag list first (see §e5), then file with this session's evidence or hold.
3. ★ M06–M09 consumer waves (Q1).
4. ★ Pre-push hook hermetic leg (Q2).
5. Re-run `nix run .#e2e` once on the final tree (receipt airtightness, §e6) — cheap.
6. Root-cause the two mysterious `cd examples/samar-do-demo` ENOENT failures from this session (directory existed; mvdan/sh quirk suspected; worked around via working_dir — worth 5 minutes before someone else loses an hour).
7. Tail archive pass: now 6 live reports (this one included) vs TAIL_BASELINE 6 — the next docs-health round should annotate + archive 06-41/07-24/09-53/17-40/18-15 (open §g items answered or restated here).
8. R2: systemcatalog-derived event catalog (`catalog.SchemaFromType[T]` + `AssertInventory`) — unblocked since `systemcatalog/v4.0.0`; also the remove-when trigger on the F032 disables.
9. L2/L3 leverage remainder: A027 codec-once, P008 e2e batch size, F005 upcaster note, 135-suppression heuristic tuning (C028×22, C023×9, A032×8), scorecard waivers.
10. bench-spike re-pin (idle-machine window; same-change rule for any bench-path edit).
11. go-sse v0.5.1 straggler bump; templ v0.3.1114 root-cause; audit_max 322 drain-or-scope decision.
12. Sibling 17-40 lane continuation (waved-harvest truth pass) — their TODO rows.
13. AGENTS S17 split (owner-parked; file grew again this session).
14. Gotcha 54 candidate (§e2: pipe-masked gate rc) — write at WRITE time per protocol.
15. NEXT-WAVE adminui: if another anchored popover is ever added, extend the explicit anchor-name refinement then (the generic layer keeps it merely centered, never corner-parked).

## g) QUESTIONS I cannot figure out myself

1. **M06–M09 consumer-wave ownership (re-asked, third time — 07-24 §g2, 18-15 §g1):** the waves edit OTHER repos (browser-history, InboxClean, SwettySwipperWeb, crm). May cqrs-htmx sessions execute them, or do the sibling sessions own those repos?
2. **Pre-push hook hermetic isolation leg (re-asked — 07-24 §g1):** add it to `.githooks/pre-push`? Today's templ-downgrade window is a fresh, concrete case it would have caught at push time; the standing counter-argument (it blocks every push during a documented-red window, like the current catalog class) is unchanged and now live — the tree cannot go fully green until the upstream release ships.
3. **templ-components upstream handling (supersedes the templGen-pattern question — the in-tree pin stands unless redirected):** file the popover-anchor-layer bug upstream now (evidence complete; consumers beyond this repo are exposed on Chromium — extent unknown), or hold it for the next templ-components family bump and keep the vendored correction as the only fix? If filing: in my voice per github-voice, from the verify-before-filing-gated evidence pack.

---

**Standing state at report time:** tree clean at `ecb65976` (this session: hand commits `a78f3b73` docs, `bbd5d943` branching-flow re-pin, `ecb65976` gate baselines; daemon sweeps `889d0ab4` CSS fix + `497d0cf6`/`21201f51` dep sweep). Nothing pushed. Composite: 85/89 with all 4 reds the single upstream catalog class. e2e 76/76. cqrs-lint 14/14 (unchanged from M10). This report raises TAIL_BASELINE 5 → 6 and CORPUS_BASELINE 44373 → (measured) in the same commit.
