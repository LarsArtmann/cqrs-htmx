# Adminui next-wave M05–M17 completion — session status report

> **ANNOTATED 2026-11-16** — ~~this report~~ **SUPERSEDED** by [`2026-11-15_14-37_comprehensive-status-update-plan-tail-execution_status.md`](2026-11-15_14-37_comprehensive-status-update-plan-tail-execution_status.md), which M05-M17 exit set: the wave shipped and its distinctive tails are harvested as TODO_LIST P2 rows; the comprehensive 11-15 ledger is the current execution record. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-16 (round-23 (2026-11-16)).**

**Date:** 2026-11-14 16:2x CET
**Author:** session agent (continuation execution session for `docs/planning/2026-11-14_08-45_SUPERB-adminui-tenant-user-management-next-wave-pareto-plan.md`; predecessor report: [`2026-11-14_11-29_adminui-next-wave-execution-status.md`](2026-11-14_11-29_adminui-next-wave-execution-status.md) — its §b/§c lists are this session's work order)
**Scope:** finished the plan's remaining milestones on the settled tree — M17 completion, M05 verdicts, M14 archive, M09/M10/M11/M12 feature waves, and the true-ups — then ran the final aggregate.
**Directive:** "execute the WHOLE todo list … keep going until everything works."

---

## a) FULLY DONE

| Item | What landed | Receipt |
| --- | --- | --- |
| **M17 — family train alignment (completed)** | (1) The httputil sweep's lint-leg failure root-caused: loginpage `buildPageData` cyclop 13>12 (the K2-recorded sibling red was real and blocked every bump-dep lint leg) — fixed by extracting `resolveOAuth2Buttons` (behavior identical, tests green); (2) admin-demo's idempotency nolint had drifted off the line staticcheck reports (golines rewraps trailing-comment nolints) — moved onto the call's first line where formatting is stable; (3) catalog swept 13 modules v4.14.0→v4.15.0 via `bump-dep` (24/25 verification legs PASS first run, the one FAIL fixed, then green); (4) **`check-release-train --refresh-cache --strict-lag 0`: 0 unpublished, 0 train lag — the strict push gate is GREEN**; (5) full workspace build + tests green post-sweep. Bonus: catalog v4.15.0's docserver is templ-1100-era — `examples/catalog-demo` builds in workspace mode again (the A25 upstream red healed by the bump; TODO row 51b satisfied). `httputil/server_timing` stays at v1.0.3 by design (upstream never cut v1.9.0 for the submodule — the unanchored sweep correctly aborts; the anchored `'larsartmann/httputil$'` pattern is the recorded recipe). | `/tmp` sweep logs (transient), plan §9 M17 row, `check-release-train` output |
| **T176 — settled-tree build + tests** | Workspace `go build ./...` + `go test ./...` green after both dependency bumps. | build/test logs |
| **M05 — browser battery verdicts** | **Chromium (blocking engine): green.** Full battery via `nix run .#e2e-firefox` on the settled tree: 117 passed, 0 chromium failures (the two failures of the EARLIER run were triaged — see §d1 — and fixed). **Ran-count gate green** on a fresh chromium JSON report: "112 listed specs, 0 produced no result". **Firefox (ADVISORY): 66 failures, all one environment class** — `browser.newContext: Protocol error (Browser.setDownloadOptions)`: Playwright 1.65's download-options protocol call against the pinned Nix firefox binary that does not implement it; zero product assertions executed; recorded as the standing advisory posture (needs the browser-stack fix or CI's registry firefox before promotion). | `/tmp/e2e-battery-m05-v2.log`, `e2e/pw-chromium-report.json` gate output |
| **M14 — archive pass + baselines** | 11 settled 2026-11-06/07/08 reports archived via `archive-status.sh --apply --update-baseline` (row gates BEFORE the move, dated SUPERSEDED blockquotes, 19 links re-based, manifest + per-file deciding-evidence table appended to `docs/status/README.md`). Live tail 14→3 (budget 3), `TAIL_BASELINE` 13→3, `CORPUS_BASELINE` re-trued to the measured 59,141 same-change. docs-links green (1,621 links). docs-gates battery: annotate/routing/cadence PASS; freshness + audit-all RED (both owner-tracked/self-healing — see §b). | `docs/status/README.md` M14 manifest; `docs/status/TAIL_BASELINE`; `docs/CORPUS_BASELINE` |
| **M09 — header tenant switcher** | Super-admin-only header Dropdown (same library component as the user menu — popover keyboard-nav, light-dismiss, Esc for free): live tenants in display-name order, suspended/deleted excluded (not navigable targets), links to each tenant's detail page, and the viewed tenant marked (check icon + the trigger names it). Tenant-admin mode renders nothing (scoped to one tenant by construction). Tests `tenant_switcher_test.go` (3: live-listing + suspended exclusion via a distinct fixture name, mode absence via `newTenantAdminPanel`, current-marking on the detail page). CSS bundle rebuilt same-change (`md:inline` added; both class-set gates green). CHANGELOG entry. | `handler.go` `tenantSwitchOptions`, `layout.templ` `tenantSwitchTrigger`, `models.go` `tenantSwitchItems`/`tenantSwitchCurrentName`, `tenant_switcher_test.go` |
| **M10 — tenant list/detail polish** | T100: status filter (`?status=active|suspended|deleted`) with badge-matched precedence (unknown values degrade to unfiltered — a hand-edited URL must not read as "no tenants exist"); chip counts are the search-scoped mix. T101: Name/Members sort (`?sort=members`, counts computed for every MATCHED tenant pre-pagination, name tiebreak). T102/T103: suspend/delete confirms state the exact blast radius ("Its N member(s) lose access…" / "revoke its N membership(s)?"). T105: `tenant_polish_test.go` (filter narrowing + degradation through the HTMX partial, member sort + tiebreak, impact confirms) + CHANGELOG. **T104 is the honest exception — see §b2.** | `handler_tenants.go`, `util.go` (`parseTenantStatusFilter`/`matchesTenantStatus`/`tenantStatusCounts`/`parseTenantSort`/`sortTenants`/`tenantListURL`), `tenants.templ` chips + sort links, `models.go` confirms, `tenant_polish_test.go` |
| **M11 — accessibility sweep** | T110: `admin-a11y.spec.ts` — axe sweeps over the tenants/users lists AND tenant/user detail (critical/serious, reasoned-acceptance contract mirroring the dashboard spec). T112: keyboard path — Tab reaches select → each role checkbox → submit, Space toggles (pinned `before != after`), Enter submits; the grant completes without a mouse and lands the row. T111 findings FIXED: user-menu trigger label now names the full visible text (initials + email — WCAG 2.5.3 Label in Name; the library renders the trigger aria-label from `DropdownProps.Label`, discovered by probing the generated component), the tenant switcher's label names the CURRENT tenant on detail pages, users-list filter chips dropped link-invalid `aria-pressed` for `aria-current`, two icon-only remove buttons gained per-target accessible names, and the wave's own chip grays bumped to gray-600 (Tailwind v4's gray-500 misses 4.5:1). Spec family (admin-behavior + admin-tenant-access + admin-a11y): **18/18 chromium green**. T113: results recorded in the CHANGELOG + §9. | `admin-a11y.spec.ts`, `layout.templ`, `users.templ`, `members.templ`, `tenants.templ`, CHANGELOG [Unreleased] a11y entry |
| **M12 — docs truth** | T120: **gotcha 60** — "a Service doc comment is not the decider's contract" (full text in `docs/gotchas.md` + AGENTS.md index row, count 55→60). T121: adminui README documents the new surfaces (tenant switcher, lifecycle chips + sort, blast-radius confirms, bulk import/export routes) — no route table pre-existed, so the "What you get" list carries it. T122: `admin-screenshots.spec.ts` harness (light+dark × tenants/users lists + tenant/user detail → `docs/screenshots/{light,dark}/admin-*.png`, 8 committed captures; `waitForLoadState("networkidle")` never settles on SSE-connected panel pages — fixed settle wait). T123 (member-row dedupe) deliberately not attempted: the renderers diverge by surface contract, not mechanically. | `docs/gotchas.md` gotcha-60, AGENTS.md, adminui/README.md, `admin-screenshots.spec.ts`, `docs/screenshots/{light,dark}/admin-*.png` |
| **M09–M12 browser proof** | The full admin spec family ran green together AFTER the feature waves: admin-behavior + admin-tenant-access + admin-a11y = **18/18 chromium** on a fresh server carrying every M09/M10/M11 change. | playwright run output |
| **e2e typecheck + TS debt** | `tsc --noEmit` green across the e2e tree — including the two PRE-EXISTING dashboard-live TS2339 errors (HTMLElement casts) that predated this session and the new specs' types. | tsc runs |
| **Docs true-ups** | Plan §9 execution-log rows for M02–M17 (per-milestone receipts); TODO_LIST `Updated:` line trued to 2026-11-14; upstream-asks item (h) extended with ask (5) tenant attribution in audit entries; CURRENT_FOCUS.md rewritten to the new thread; FEATURES.md adminui rows trued (switcher, filter/sort, import/export, last-seen, multi-role, attribution). | the files themselves |
| **Local tooling hygiene** | The stale `/tmp/cqrs-htmx-e2e-server` (root cause of §d2) and stale admin-demo binary replaced with current-tree builds so future isolated runs cannot silently test week-old code. | rebuild receipts |

## b) PARTIALLY DONE

1. **M01 — final aggregate GREEN (104/104) — RESOLVED same-session.** The first settled-tree `check-modules` rerun reported **2 of 104 checks FAILED**: (a) `check-docs-corpus-budget --ratchet` — docs grew +17 lines after the M14 re-true (gotcha 60, plan §9 rows, FEATURES/README/CURRENT_FOCUS edits) without the ledger sentence — exactly the drift the ritual says to ledger same-change, and I batched it to the end instead — repaired by this report + the same-change re-true (commit 82876e9f); (b) `check-branching-flow` (+6 added), identified from the in-flight rerun log (its first log was lost to /tmp cleanup — itself a lesson, §d6). The branching-flow red was triaged and re-pinned same-session (commits ffb607d2 + 12ba3e47): all six gate-adds decompose to the already-frozen adminui view-model mixin family — ONE pairing flip (`usersListData`↔`tenantsListData` → `usersListData`↔`auditData`, same 4 shared fields, the analyzer's reported counterpart flipped as the file grew) + five fix-snippet re-anchors of pairs named in the M02/M03 amendment; per-rule counts identical 211 = 211; zero findings on new code (amendment row in docs/analysis/triage-decisions.md). The final aggregate on the repaired tree: **104/104 GREEN, rc=0** (`/var/tmp/check-modules-final3.log`). §f34 resolved.
2. **T104 — per-tenant audit shortcut: built, then REVERTED.** The audit-filter extension (tenant-name matching) was implemented, then discovered unverifiable: the audit log records **zero** tenant lifecycle entries (verified empirically — a CreateTenant dispatch produces no entries; `commandAuditMiddleware` covers user/session commands only). A shortcut landing on an empty search is a dead-end link — the plan's philosophy forbids shipping it. The filter extension and the link were reverted in the same change; the gap is recorded as TODO_LIST item (h)(5) (tenant attribution in audit entries, D2-gated) and the test documents it with a reasoned skip.
3. **docs-gates freshness** — still RED by design until this report lands (the newest report must be fresher than the newest change); it greens with this very file. audit-all remains RED (owner-gated drain-or-re-pin; policy forbids raising the pin without an owner tick — untouched, per instruction).
4. **Firefox advisory promotion** — not attempted (66 environment-class failures; needs the browser-stack decision, not test fixes).
5. **M12 T123** — the optional member-row renderer dedupe: deliberately skipped (contract divergence, not mechanical duplication).

## c) NOT STARTED

- **Owner decisions D1–D3** (adminui packet) — filed with pre-staged recommendations; every follow-on wave (impersonation, bots, sessions, rename) stays blocked on the ticks. Same for the M2M D1–D4 packet.
- **Filing the usermgmt asks upstream** — item (h) now has FIVE asks, all recorded in TODO_LIST only (owner-gated; verify-before-filing applies before any GitHub issue).
- **The `httputil/server_timing` v1.9.0 upstream ask** — the submodule tag was never cut upstream; the ask should be filed so the treadmill stops reporting the family split.
- **Push + CI watch** — the tree is push-gated GREEN (strict train gate 0 lag) but NOT pushed (never authorized); no CI run has validated this session's work.
- **The color-contrast design pass** — ~115 accepted pre-existing contrast findings (accent brand chrome + gray-400 metadata ramp) documented in the spec's ACCEPTED set; the fix is a design decision, not mechanical.

## d) TOTALLY FUCKED UP (honest list)

1. **The earlier battery's "2 failures = sweep noise" judgment was WRONG, and I inherited it without reproving it.** One of those failures was a deterministic spec bug (the users-list assertions against an unscoped paginated page). The plan's M05 explicitly existed to re-verify on a settled tree; the summary I resumed from said "judged environment noise" and I initially planned to just re-run — the real triage (reproduce in isolation, read the seed, find the 60-filler pagination trap) only happened because the rerun failed again. Gotcha 40's lesson ("a documented red retires the evidentiary value of every later run") applied to MY OWN inherited judgment.
2. **I chased a wrong hypothesis for a full debug cycle and edited a spec on an unproven theory.** The time-travel slider failure: I built an SSE-race theory, added a `page.route` block to the spec, reran, watched it fail anyway, and only THEN instrumented the page properly — the real cause was the stale `/tmp` e2e-server binary resolving dashboardui **v4.17.0 from the module proxy** (that tag predates the delegated-slider fix; NO dashboardui tag contains it — only the workspace). The route-block was reverted. The wasted cycle was avoidable: `go version -m` on the binary would have named the resolved dashboardui version in one command.
3. **The T104 premise was never verified before building.** I implemented the tenant-name audit filter and the shortcut link FIRST and checked the audit log's actual contents AFTER — the build-then-revert cycle is pure waste and violates verify-before-building (the outbound cousin of verify-before-filing).
4. **The sort assertion was written inverted** — I flagged the CORRECT display-name order as the error, then misread two consecutive reruns as product bugs before finally reading my own assertion. `strings.Index` comparisons deserve the same care as product code.
5. **Daemon commit races all session.** My explicit `git commit` failed on moved-HEAD refs 3× ("cannot lock ref"), and several milestone commits exist only as anonymous "chore: auto-commit N file(s)" entries — the §9 log is the narrative because git history is not. One near-miss: a daemon batch shredded my e2e spec fix together with unrelated sweep go.mods.
6. **The check-modules verdict log was written to /tmp and cleaned before I read which 2 of 104 checks failed.** Long gate logs belong in /var/tmp or the repo; I re-ran the aggregate to get the names.
7. **Corpus growth was batched instead of ledgered.** Gotcha 60, §9 rows, FEATURES/README/CURRENT_FOCUS edits all grew docs/ across hours without the same-change ledger + re-true — the exact ritual violation the corpus gate exists to catch, caught by the aggregate at the end. The `+2` erraudit drift (330→332) is the same shape in miniature.

## e) WHAT WE SHOULD IMPROVE

1. **Verify the premise before building the feature** (T104): one `Entries()` dump would have saved the entire build-revert cycle. The same discipline the runbook demands for upstream claims applies to OUR OWN service behavior.
2. **Never inherit a "judged noise" verdict** — reproduce on the settled tree or it is not triaged (gotcha 40, self-applied).
3. **Read the binary's build info before blaming code**: `go version -m` names resolved module versions instantly; the stale-proxy-resolution class (workspace source vs published tag) deserves a gotcha of its own, alongside the `reuseExistingServer` trap (a "fresh" playwright run silently reusing a stale server on :18930 — both bit this session).
4. **Ledger corpus growth as it lands** — every docs-touching milestone should carry its ledger sentence in the same commit, not a batch at session end.
5. **Long gate logs go to /var/tmp** (or the repo); /tmp is cleaned under this host.
6. **`nolint` placement rule**: a directive must sit on the line staticcheck reports — golines rewraps trailing-comment directives off multiline calls; put the directive on the call's first line (under 120 cols) so formatting is stable.
7. **ACCEPTED-set granularity**: accepting `color-contrast` by ID masks future regressions on the swept pages; the follow-up design pass should shrink it to per-node scoping or delete it (the spec comment says so).

## f) NEXT 50 (ordered: blockers → waves → hygiene)

**Immediate (this change + the aggregate):**
1. Land this report with the same-change corpus ledger + `CORPUS_BASELINE` re-true (+ `TAIL_BASELINE` 3→4) — repairs aggregate failure (a).
2. Read the in-flight aggregate rerun's two failing check names; fix the second (if not corpus) and record the final 104-check verdict.
3. Re-run docs-gates battery post-report: freshness must flip GREEN; annotate/routing/cadence stay PASS.
4. Re-true the plan §9 M01 row (aggregate verdict) — the table still carries a stale "M01 not started" tail row from the predecessor session.

**Push gate:**
5. Owner authorization to push; then `train-preflight` → push → `push-and-watch` (workflow-filtered CI watch).
6. First CI verdict on the session's work: the loginpage lint leg, the admin-demo lint leg, the e2e chromium job, and the docs gates are the four legs my changes touch.
7. If CI's e2e job reddens on firefox: decide the `setDownloadOptions` mitigation (pin an older playwright, or the registry firefox path CI already uses — locally the pinned Nix firefox is the mismatch).

**Owner-gated waves (blocked on ticks, all pre-staged):**
8. D1 tick → impersonation panel surface behind opt-in config (no usermgmt change needed — the erratum stands).
9. D2 tick → ONE usermgmt tag train carrying the five (h) asks: bot listing, batched membership counts, tenant rename, session listing/revocation, audit tenant attribution.
10. D3 tick → wave ordering (sessions+impersonation before bots).
11. (h)(5) landing unblocks the M10 T104 audit shortcut — rebuild the link + the tenant-name filter against real entries.
12. M2M D1–D4 ticks (separate packet, unchanged).
13. usermgmt→metaengine D1–D5: W1/W2 probes already GO.

**Feature/hardening follow-ups surfaced by this session:**
14. color-contrast design pass (the ACCEPTED entry's debt: accent brand chrome + metadata gray ramp).
15. Shrink the a11y ACCEPTED entry to per-node scoping after that pass.
16. Firefox battery stack: fix the `setDownloadOptions` mismatch so the advisory engine produces real verdicts again; then the 3-clean-runs promotion clock.
17. Adminui audit-log attribution wave (needs (h)(5)): surface tenant + membership actions in the audit trail.
18. `aria-current` audit across OTHER chip/link toggles the sweep did not cover (loginpage, dashboardui).
19. axe sweeps for the remaining adminui pages (audit, dashboard, import form) — the spec covers lists + details only. <!-- fresh:skip: routed — harvested as the "axe sweeps for the remaining adminui pages" sub-row of the M05–M17 §f-tails batch in TODO_LIST P2; every >=4-char token in this item (sweeps/remaining/adminui/pages/audit/dashboard/import/form/spec/covers/lists/details) is generic across TODO_LIST (DF >= 8) and "axe" is below the token floor, so the match cannot be proven mechanically -->
20. Mobile-viewport admin screenshots (the harness is light/dark; the dashboard harness has mobile).
21. Keyboard-path tests for the REMAINING forms (import upload, new-user, new-tenant, member add).
22. The `reuseExistingServer` trap → candidate gotcha 61 (stale server masquerading as a fresh run; kill-strays is not enough if a manual server holds the port with old code).
23. The stale-proxy-resolution class → candidate gotcha 62 (`go version -m` the binary; workspace source vs published tag divergence).
24. `waitForLoadState("networkidle")` never settles on SSE pages → candidate gotcha 63 (use fixed settle waits on SSE-connected panels).
25. Import/export depth (D2-adjacent): dry-run mode, per-row error export, larger batch guidance.
26. Tenant rename UI (rides (h)(3)'s rename command). <!-- fresh:skip: owner-gated — the panel surface of upstream-ask (h)(3), already routed by the D1–D3 decision row and the (h) upstream-asks row in TODO_LIST -->
27. Bot management surface (rides (h)(1) + D3 ordering).
28. Sessions admin surface ("sign out everywhere", rides (h)(4)). <!-- fresh:skip: owner-gated — the panel surface of upstream-ask (h)(4), already routed by the D1–D3 decision row and the (h) upstream-asks row in TODO_LIST -->
29. Impersonation forensics view (rides ADR-0060's journal, post-D1).
30. Batched membership counts adoption (rides (h)(2)) — retires the N+1 in `tenantsIndex` AND `tenantSwitchOptions` (the switcher added a second per-tenant-count scan on detail pages).
31. `tenantSwitchOptions` cost note: it runs `AllTenants()` on EVERY super-admin page render — fine at demo scale, worth the batched API at fleet scale.
32. Route-posture manifest: verify the four import/export rows + switcher landed postures (golden covers them; spot-check the CI hermetic leg after push).
33. erraudit +2 drift (330→332): triage the two new findings (loginpage `resolveOAuth2Buttons` extraction is the likely source) at the next advisory sweep.
34. branching-flow ratchet: re-pin after the next code-settling commit if the extraction shifted anchors (211 baseline held at last check). **RESOLVED 16-4x same-session:** the rerun's +6 were the M09–M12 line-shift bill inside the frozen view-model mixin family (one pairing flip + five fix-snippet re-anchors); re-pinned 211 → 211 net-zero (commit ffb607d2), amendment row in docs/analysis/triage-decisions.md, gate re-run +0 added — leg of the final 104/104 aggregate.
35. Coverage: adminui threshold held at 68.2% pre-wave; re-run `coverage-gate` post-M09–M12 (new files: attribution, import, switcher, polish paths). **RESOLVED 16-5x same-session:** gate PASSED, all 16 modules above threshold — adminui 69.7% vs 66% (UP from 68.2 with the new M09–M12 files), loginpage 81.1/79, usermgmt 83.0/74, root 90.6/90.
36. `dashboard-live` slider: consider pinning the e2e server build to refuse stale-proxy resolution (a `go.work` presence assertion in the battery bootstrap).
37. Document the anchored-pattern recipe (`'larsartmann/httputil$'`) + the server_timing publication check as a bump-dep pre-flight step in the runbook.
38. Upstream ask: `httputil/server_timing` v1.9.0 tag (the family split that aborts unanchored sweeps).
39. Upstream ask: dashboardui family train carrying the delegated-slider fix into a TAG (no dashboardui tag contains it — every hermetic consumer build of the demo servers serves the dead slider).
40. Consider a `go.work`-vs-published drift gate: assert hermetic demo builds serve the same JS the workspace serves (the class behind §d2).
41. The T132–T134 plan erratum (adminui v4.14/v4.15 tags never cut) is recorded in §9 — mirror one sentence into the adminui CHANGELOG's archaeology note for consumers reading the tag history.
42. Screenshots: add the import page + the switcher open-state to the harness (the two most distinctive new UIs).
43. Consider `axe` on the ERROR pages (404/errorpage cards) — the sweep skips them.
44. Keyboard path for the tenant switcher itself (the library Dropdown ships keyboard nav; pin it once in the a11y spec).
45. Consider surfacing "added by" on the user-detail access rows too (M08 data exists; only the roster renders it today).
46. The seed's 60-filler pagination trap bit two specs — consider an admin-demo env knob (smaller filler count in e2e) OR a spec helper that always searches.
47. Harvest the predecessor report's §f list against this session's landed items (the docs-health freshness tokens) — several of its 50 items are now done and annotated.
48. Re-run `check-skill-claims`/`check-skill-snippets` attention: the SKILL.md's adminui section may deserve the switcher/import mentions (consumer-facing surface truth).
49. The adminui CHANGELOG [Unreleased] is now four substantial entries — evaluate cutting an adminui tag train (the release playbook) once the owner ticks land.
50. Update `docs/runbooks/session-start.md` if the exit-report ritual should mention the corpus-ledger-with-report pairing (this session re-derived it twice).

## g) QUESTIONS FOR THE OWNER (not self-answerable)

1. **D1–D3 (adminui packet) + the five usermgmt asks (h): do the ticks land now, and does D2's "ONE tag train" get scheduled?** Everything gated (impersonation UI, bots, sessions, rename, audit attribution → the T104 shortcut) waits on this one answer set.
2. **Push authorization: is the tree (strict train gate 0 lag, aggregate pending the corpus repair in this same change) cleared for `train-preflight` → push → CI watch, or does it wait for the aggregate's final green + your review of the a11y ACCEPTED set?**
3. **The ~115 pre-existing color-contrast findings: commission the design pass now (accent palette + metadata gray ramp across adminui chrome), or keep them accepted-with-reason until a dedicated adminui polish wave?** It changes the ACCEPTED entry's lifetime and whether the next a11y sweep tightens.

---

## Verification receipts (spot checks)

- `check-release-train --refresh-cache --strict-lag 0` → RC=0, "0 unpublished, 0 train lag".
- Chromium battery (settled tree): 117 passed / 0 failed; `check-e2e-ran-count.sh e2e/pw-chromium-report.json` → "112 listed specs, 0 produced no result".
- Admin spec family post-M11: 18/18 chromium (admin-behavior + admin-tenant-access + admin-a11y).
- adminui hermetic: `go test .` ok (7.8s), `golangci-lint run` 0 issues; loginpage hermetic test ok + lint 0 after the cyclop fix.
- `check-css-bundles` + `check-css-bundle-classes` green after every templ change (3 rebuilds this session).
- e2e `tsc --noEmit` RC=0.
- Archive pass: tail 3, links green (1,621), manifest + ledger landed.
- docs-gates: annotate/routing/cadence PASS; freshness self-heals with this report; audit-all owner-gated RED (unchanged posture).
