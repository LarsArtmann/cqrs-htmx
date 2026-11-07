# Status: full-backlog pareto execution — v4.18.0 + QUERY W2/W4 + docs-drain + upstream filings

**Session:** 2026-11-07 ~09:00–11:00 CET · **Repo:** `cqrs-htmx`, branch `master` (all work pushed through `5661c288`; later commits daemon-held locally) · **Plan of record:** [`docs/planning/2026-11-07_08-42_SUPERB-full-backlog-pareto-rank-and-execution-plan.md`](../planning/2026-11-07_08-42_SUPERB-full-backlog-pareto-rank-and-execution-plan.md), executed under the owner's blanket directive ("NOW GET SHIT DONE! The WHOLE TODO LIST!"), recorded as decision **D18**.

## §a Accomplished (all receipts in CHANGELOG [Unreleased] / module trees / git)

1. **Conflict-marker incident resolved (gotcha 47, strike three).** A sibling's stash-pop collided with the committed QUERY plan; the daemon committed 10 raw conflict blocks in the plan of record + a duplicate QUERY row in TODO_LIST. Resolved keeping the upstream (post-execution) side; `check-conflict-markers` + todo-hygiene green (`28a5d23d`). The gate later proved itself again live (pre-commit staged scan).
2. **1% tier — publish + prove (C01–C04).** Root **v4.18.0** tagged (`6b804a01`, annotated, peeled-ref verified on origin), CHANGELOG section cut (QUERY W1 + Go 1.27.3 floor), consumer sweep to v4.18.0 (42 files, tidy+verify+build+vet+lint green, `2c077269`), then go-appkit v0.12.0 swept when upstream published mid-session (`5661c288` — gotcha 39 live). Receipts: coverage gate PASSED (15 modules), cqrs-lint 14/14 strict, erraudit strict TOTAL=326 (advisory), pre-push CI-parity gates green (950→951 requires, 0 unpublished, 0 lag). Two CI reds on the way were triaged and healed: root-clusters row missing for `method_override.go` (`ed92944a`) + a one-line corpus race.
3. **4% tier — QUERY spec-truth (C05–C11).** `openapi.Query()` with auto-bump to OpenAPI 3.2.0 + `Validate` invariant (a query op on any other version fails); `ETagResults()` (strong FNV tag over rendered bytes, 304 on If-None-Match, wildcard, misorder = logged no-op); huma verdict **NO** pinned by a verdict test that reddens the day upstream huma ships a `query` Path Item case (v2.40.0 panics `unknown method QUERY` — verified at source); **ADR-0064** (the fallback ladder as library contract) accepted + indexed; README QUERY row + OpenAPI 3.1/3.2 note; hermetic suites green (samber-do-demo, loginpage, GOWORK=off). C11's branch-guard + self-test found ALREADY LANDED (round-45 M17) — stale row, verified green.
4. **20% tier — the superb integrations (C12–C26).** **htmx 4.0.1 embedded alongside 2.0.10** (`htmx.ScriptHandler4()`, unpkg dist 37,448 B, pin verified: bundle carries `hx-query` + `"query"`; QUERY-support bytes test-pinned so a bad re-vendor fails the suite); **the `cqrs-query` extension v1.0.0** (the library's own: `hx-query` upgrades an `hx-post` to native QUERY, first 405/501 remembers the origin for 1h in sessionStorage and retries as the override POST; registered as `htmx.ExtCQRSQuery`, served+bundled through the existing extension handler); SA1019 sweep (6 modules → `htmx.ScriptHandler`); **ExtraNav (c)(d)(e)** — `docs/guides/dashboard-embedding.md` (Layout contract verified against source; Verbatim section), the row-23 table ask resolved-by-execution (0 unpublished/0 lag today), `renderNavForTest` probe + both body-assertion tests migrated; **StateShape audit (C24)** — VERDICT: the SQL snapshot store DROPS the stamp (no column/INSERT/scan at storage/v4 v4.11.1; empty stamp = accepted-as-legacy → bites on shape bumps); pinned by `TestSnapshotShape_SQLStoreDropsStampToday` (flips when upstream fixes); **filed as [go-cqrs-lite#117](https://github.com/LarsArtmann/go-cqrs-lite/issues/117)**; **httputil CSRF ask filed as [LarsArtmann/httputil#48](https://github.com/LarsArtmann/httputil/issues/48)** (QUERY is spec-safe; `isUnsafeCSRFMethod` at v1.8.0 safe-lists only GET/HEAD/OPTIONS/TRACE — verified at source); round-42 third test case settled as a reachability verdict (the `aggIDFromUser` failure branch is defensive dead code for typed UserIDs — tripwire test shipped); cqrs-lint advisory batch adjudicated STALE (A013/A027/P008/F005 = zero live findings at the rebuilt 4.20.0 binary; A027 improved by the shared `cborEventCodec`); coverage self-test + templ-chain rows verified STALE (already shipped, green); **proxy compatibility table** added to the guide (Envoy v1.40+/Caddy/nginx = verified YES with official citations; Cloudflare edge + AWS ALB = honestly UNVERIFIED); ROADMAP question 35 (the three QUERY v5 flags); loginpage v4.14.0 release notes pre-staged (verified it did NOT ride v4.17.0); crush-config lessons F11/F12 committed (`c033659`); **migration plan drafted** (see §a5).
5. **usermgmt→system/metaengine migration plan (C42).** [`docs/planning/2026-11-07_10-45_SUPERB-usermgmt-system-metaengine-migration-plan.md`](../planning/2026-11-07_10-45_SUPERB-usermgmt-system-metaengine-migration-plan.md) (6 waves, 5 owner decisions). Criterion verdict **PARTIALLY MET** at the current pins — and two long-standing TODO claims proven STALE at source: `system.New` does NOT default to an in-memory checkpoint store anymore (consumer → engine-backed `system_checkpoints` → memory fallback), and the hydration contract is met by construction on persistent engines (`ResetProjection` is published). Rows 107/108 + `docs/guides/v5-removal-inventory.md` §5c corrected. V007 recommendation: evaporate the SQL view tier at v5.
6. **Docs-drain wave (C19).** Annotate gate: 1 rule-F finding fixed (`HandlerOption` plural-span) → CLEAN. Cadence: 3 oldest reports archived via the ritual (tail 18→15, links re-based, TAIL_BASELINE lowered same-change). Audit-all: **four drain agents applied ~300 evidence-backed verdicts** across the 12-item live tail (439 → ~193 live; annotate gate CLEAN after normalizing the tool's kind-letters to the marker vocabulary and strike-wrapping routed items — bare routed notes do NOT reduce the count, learned live). The remaining ~221 items live in ARCHIVED pre-canon October reports (the tool replays them despite its --help claiming "unarchived corpus"); a second wave (2 agents) is mid-flight/retry on the top-16 archived files. **The audit-all pin (322) is still exceeded (≈420 at last measure) — the drain-or-repin decision point is NOW** (see §g).
7. **TODO harvest + truth pass.** Duplicate QUERY row deleted (conflict artifact); stale rows deleted with CHANGELOG receipts (TOC-merge row — branch is an ANCESTOR of master, nothing to merge; coverage self-test row); QUERY/ExtraNav/cqrs-lint/migration/sync/erraudit rows trued in place with today's receipts; hygiene gate green (76 rows, 0 violations).

## §b Not done / open (per plan tier)

- **C27/C28 consumer waves** (browser-history, Swetty, crm, InboxClean, github-local-sync, webphone + dispatch-gate adoption): THEIR repos — not started, untouched (multi-repo scope, sibling-coordination).
- **C29 M2M W1/W2**: still blocked on the four M2M owner decisions (row 35) — deprecating/reshaping a published surface is not a directive-executable call.
- **C14/C15 browser e2e** (extension + hx-query, NV4 SSE/idiomorph compat): deliberately left to the sibling session's ACTIVE browser-e2e battery wave (their 09-13 report; port/spec collision risk).
- **C17/C18 DataStar bundle bump**: sibling's active lane (their datastar deep-dive + e2e reports) — untouched to avoid the two-writers class.
- **F020 PR-#40 receipt comment**: draft exists (`docs/drafts/f020-pr40-receipt-comment.md`, voice-checked); the precondition (a fully green CI run on the final pushed head) is not yet met — the last pushed run reddened on the one-line corpus race; the final push of this session's commits carries the corrected baseline.
- **bench-spike re-pin**: refused-by-rule (machine not quiet; standing row).
- **humans/UX**: nothing.

## §c Verification receipts (the honest gate state at ~10:55)

- GOEXPERIMENT=jsonv2 `go test` -race GREEN: root (23.4s), openapi, htmx, usermgmt (22.2s incl. the new tests), dashboardui, examples' modules via the sweep's per-module verify.
- `buildflow -s golangci-lint` across modules: **0 findings** (the earlier 12 in my new code fixed: goconst→`methodQuery`, wrapcheck→errorfamily wrap, exhaustruct, wsl/golines/gofumpt churn).
- `nix run .#check-cqrs-lint` 14/14 strict; coverage gate PASSED; todo-hygiene green; docs-links green (1,506 links); skill gates green (45 fences, 222 symbols); conflict-markers green; corpus/tail ratchets bumped same-change throughout; docs-gates battery **4/5 PASS** (audit-all red at ~420 vs pin 322 — the drain'sremainder).
- Local-only gates not yet re-run on the final tree: full `nix run .#check-modules` (deferred to the post-drain settle), branching-flow (+13/−13 vs baseline — pre-stage recorded; the re-pin verdict refresh still owner-gated), bench-spike (quiet-window rule).

## §d What was fucked up and fixed (mistakes, self-caught)

1. My first ETAG/probe tests mis-registered query types + asserted pretty-printed JSON as compact — fixed by matching the dispatcher's registered name and dropping the redundant assertion.
2. My first dashboard-embedding guide WRONGLY described the `LayoutFunc` signature (`(w io.Writer, meta PageMeta) error`); caught verifying against `dashboardui/layoutfunc.go` and rewritten to the real `(meta PageMeta, content templ.Component) templ.Component` contract before commit.
3. The initial proxy-guide example carried a nonexistent `hx-ext-encode` attribute — fixed to `hx-ext="cqrs-query, json-enc"`.
4. The daemon raced/absorbed several commits (expected churn; every commit's content verified in HEAD afterward per gotcha 51's lesson).

## §e Improvements identified

- The annotate tool's `status-items` kind-letter suffixes (`h`/`v`/`r` + TAB) contradict the annotate gate's marker vocabulary — every agent needed a python normalization pass. File upstream (docs-health assets): emit vocabulary words directly.
- `docs-gates --help` says audit-all replays the "unarchived corpus" — it demonstrably replays archived files too. File upstream (docs-gates) or add an explicit scope knob.
- Routed-bare verdicts don't reduce audit-all (only strikes do) — either the tool's `r` kind should strike (as the gate wants), or the gate should credit routed-bare lines; they disagree today.

## §f Next ~50 tasks (ordered; the plan's C-ids)

1. Re-run/finish the archived drain wave: mindy (arch-A, 8 files) FAILED rate-limited with zero writes — relaunch; betty (arch-B, 8 files) mid-flight.
2. After the drain settles: re-measure audit-all → EITHER ≤322 (pin drops, policy-clean) OR record the drain-or-repin call with the owner (raising 322→~420 needs the explicit tick; the blanket directive arguably covers it but this pin is the one policy that says never raise without a recorded owner decision).
3. Push the session's remaining commits; watch CI (`.#push-and-watch`); on a fully green run post the F020 PR-#40 receipt comment (draft ready) and close row 42.
4. Full `nix run .#check-modules` on the settled tree (expect green except the documented catalog-poison legs + branching-flow +13 pending the owner's re-pin tick).
5. Write the session report (this file) — DONE; tail/corpus baselines bumped same-change.
6. Relaunch `drain-arch-a` (rate-limit retry) with the same prompt.
7. When betty reports: merge counts, re-measure, re-pin audit_max.
8. Post-drain: `docs-gates run .` must show 5/5.
9. examples/basic: serve `/htmx4.js` + a demo button using the `cqrs-query` extension (dogfood; browser check with the sibling's wave).
10. e2e spec: extension upgrade/downgrade case + htmx-4 `hx-query` case (once the sibling's battery wave lands).
11. NV4 verdict: load ExtSSE/idiomorph under htmx 4 (probe page in e2e/server).
12. NV1: identify the DataStar release adding `@query()` (upstream tags watch) → then the fork-bump (C17, sibling-coordinated).
13. httputil#48 follow-up: when upstream ships the safe-list, bump + drop the guide's CSRF caveat row.
14. go-cqrs-lite#117 follow-up: when upstream persists StateShape, flip `TestSnapshotShape_SQLStoreDropsStampToday` to assert preservation + ride the family train.
15. kv v4.5 watch (#113 closed-resolved): bump + delete usermgmt's view-cache adapter.
16. loginpage v4.14.0 train (wave-tag; notes pre-staged).
17. huma/v4.0.0 tag (row: no tag exists; unblocks `go get`).
18. M06-M09 consumer waves (their repos; C27/C28).
19. M2M W1/W2 after the four owner decisions (row 35).
20. The migration plan's W0 (truth-on-dust errata: systemadapter doc-comment counts "21 commands/27 events" fix) + W1 probes.
21. ADR-0064 already references ROADMAP Q35 — done; keep the v5 flip list alive at the v5 window (direct-imports W1-W3, 422, htmx-4 default, extension retirement).
22. audit_all pin + `.docs-gates.json` knob decision (see §e).
23. branching-flow re-pin tick (owner) — pre-stage numbers recorded (+13/−13/~164).
24. bench-spike re-pin at the next quiet window.
25. round-42 (b) `buildflow format` quiet-machine re-run.
26. ExtraNav (a) scheme-validation + (b) aria-current owner decisions (row 43).
27. proxy table: re-verify Cloudflare/ALB rows quarterly or on RFC-anniversary (UNVERIFIED discipline).
28. docs-gates + annotate-tool upstream asks from §e (three small filings).
29. Todo: delete the drained-archived follow-ups once counts settle (this §f is the harvest source).
30-50. The 08-42 plan's 80%-tier C27-C44 items, each still owned by its row.

## §g Owner questions (≤3)

1. **audit-all pin (the drain-or-repin call, second ask):** the live tail is drained with evidence (~193 left, all deliberate with reasons); the residue ~221 sits in archived pre-canon October reports, and the finishing drain wave is mid-flight. If the final count stays >322: **raise `audit_max` to the measured floor (~420)** — the blanket directive covers it, but this specific pin's policy says "never raise without an owner tick", so it is the one decision I will not take alone. Alternatives: keep grinding the archived files (another ~1-2 waves), or exclude `archived/` from the replay (an upstream docs-gates ask — arguably the doctrinally correct fix since archived = resolved by definition).
2. **F020 posting authority:** the draft is voice-checked and the precondition is "newest CI run fully green" — after the final push, may I post it with the unticked provenance banner (you tick after review), or do you want to review the draft first?
3. **Sibling-lane ownership for the remainder of the 20% tier:** the browser-e2e battery + DataStar lanes belong to a concurrent session (active reports at 09:13). Confirm I should keep hands off C14/C15/C17/C18 and leave them to that wave (my recommendation), so the QUERY row's remaining W2/W3 sub-items carry their name, not a second writer.
