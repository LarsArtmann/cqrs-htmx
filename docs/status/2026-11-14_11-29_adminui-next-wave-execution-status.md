# Adminui tenant/user-management next-wave execution — session status report

**Date:** 2026-11-14 11:29 CET
**Author:** session agent (execution session for `docs/planning/2026-11-14_08-45_SUPERB-adminui-tenant-user-management-next-wave-pareto-plan.md`)
**Scope:** executed the SUPERB plan's 17 milestones in dependency order; this report records what landed, what partially landed, what never started, what went wrong, and what is next.
**Predecessor plan:** the plan named above (M00 was done before this session; its §9 execution log carries per-milestone receipts written as work progressed).

---

## a) FULLY DONE

| Milestone | What landed | Receipt |
| --- | --- | --- |
| **M13 — CHANGELOG archaeology** | Verified already complete; NO write was needed. All four adminui tags (v4.12.4, v4.13.0, v4.16.0, v4.17.0) have substantive CHANGELOG sections; the plan's assumed v4.14.0/v4.15.0 adminui tags were never cut. Also closed the stale TODO_LIST row "adminui CHANGELOG receipt gap". | §9 log entry; `git tag -l 'adminui/*'` |
| **M15 — upstream asks** | (T150) `AddMember` doc comment corrected in `usermgmt/service_membership.go` — now states the Conflict refusal (`usermgmt.membership.already_exists`) and points at `UpdateMemberRoles`; usermgmt CHANGELOG [Unreleased] entry added; hermetic build green. (T151) the four usermgmt API asks recorded as TODO_LIST upstream-asks item (h): bot listing, batched membership counts, tenant rename, session listing/revocation — all D2-gated. (T152) published-API-only decision recorded in the plan §9. | commit (daemon); plan §9 |
| **M16 — owner decision packet** | `docs/proposals/2026-11-14_adminui-next-wave-owner-decisions-d1-d3.md` filed: D1 impersonation GO behind opt-in config; D2 batch all four usermgmt asks into ONE tag train; D3 sessions+impersonation before bots. Cross-linked from a new canonical TODO_LIST row. **Erratum recorded:** published usermgmt v4.20.0 already exposes `BeginImpersonation`/`EndImpersonation` + the ADR-0060 forensics — D1 needs NO API change. Gated work stays blocked pending the owner tick. | the packet file; TODO_LIST row |
| **M02 — multi-role membership editing** | `roleCheckboxGroup` templ primitive (components.templ); both surfaces swapped (tenant roster rows + add-member form, user-page grant + per-row controls); `parseRolesForm` validates every posted value against `AssignableRoles()` and refuses none-checked/unknown sets; CSS bundle rebuilt same-change (adminui +`gap-x-3`, −2 select widths); canonical v0.3.1100 templ regen; hermetic + workspace builds, lint 0, route-postures green; tests pin set-two → reduce-to-one → clear-all-refused on BOTH surfaces + unchecked-start render pins. | `user_tenant_access_test.go`, `flows_test.go`, `coverage_gaps_test.go`, `tenant_ux_test.go`, `seed_render_test.go` |
| **M03 — edge states, a11y, sync targets** | (1) member-of-every-tenant state replaces the dead-end disabled select (`withoutTenantOptions` now drops the prompt-only residue); (2) true-empty users directory carries a **Create the first user** CTA; (3) unknown-email banner switched `FeedbackWarning`→`FeedbackError` so the library's `role="alert"` semantics announce it (the plan's assumption that warning already carried it was WRONG — pinned by test); (4) last-admin removal guard escalates the confirm dialog on BOTH removal surfaces ("the tenant would be left without an admin"), backed by a bounded per-tenant admin count computed for the visible rows; `data-sync-target` markers on the tenant identity card + tenant access card. Pinned by `edge_states_test.go` (5 tests). | `edge_states_test.go`; stale `handler_test.go`/`user_tenant_access_test.go` expectations trued |
| **M06 — import/export users** | Super-admin routes `GET/POST /users/import` + `GET /users/export.csv` + `GET /users/export.json` (literal routes — Go ServeMux forbids partial-segment wildcards, so the plan's `export.{ext}` shape was corrected); multipart upload through the published `ImportUsersFromJSON/CSV`; `http.MaxBytesReader` caps uploads (413); CSV header must declare an `email` column or the import is refused up front (the upstream importer maps by name and would silently land nothing); result panel with exact counts + bounded error list; users-list entry points; manifest golden + route rows with Why. Pinned by `import_export_test.go` (8 tests incl. round-trip, duplicate re-run, malformed/oversized refusals, service-level end-to-end). | `handler_import.go`, `import_export_test.go`, routes golden |
| **M07 — last-seen column** | "Last seen" column on the users directory: latest audit-window entry per user's stream, relative time + absolute hover title, honest dash when the bounded window records nothing. ONE scan of the audit window per page render (no per-row queries, no unbounded scan). Published AuditLog API only. Pinned by `TestLastSeenColumn`. | `handler_users.go` `lastSeenIndex`, users.templ |
| **M08 — member attribution** | IMPLEMENTED (the plan's implement-vs-verdict decision went to implement): the membership's own stream's first event carries `Metadata().ActorID` (go-cqrs-lite ActorEnricher, wired by NewService with zero consumer code); one bounded stream Load per visible row renders "by \<email\>" on the roster (bot/system actors keep the prefixed id; unattributed events render nothing). Pinned by `TestAddedByAttribution` through the real dispatch context (`usermgmt.WithUser`). | `attribution.go`; members.templ Added cell |
| **M01 partial — gate battery** | cqrs-lint strict **14/14 GREEN**; erraudit strict advisory **TOTAL=330 GREEN** (was 332); coverage gate **PASSED** — all 15 modules above threshold, adminui 68.2% ≥ 66; docs-gates **4/5** (audit-all PASS at 267≤268, annotate/routing/cadence CLEAN; freshness RED — expected, resolves as milestone CHANGELOG entries harvest the predecessor §f tokens); branching-flow re-pinned **211→211 net-zero** (M02 line-shift drift on frozen mixin pairs + one sibling loginpage large-struct; amendment row in triage-decisions.md); lint-sweep adminui-clean; tidy-sweep green (root go.sum tidied). | §9 log; `docs/analysis/triage-decisions.md` amendment |
| **M04/M05 partial — e2e** | New `e2e/tests/admin-tenant-access.spec.ts` (4 specs, self-seeding tenant+user per test, CSRF from the page meta): grant-from-user-page, checkbox-group role change + revoke with per-step state assertions + the confirm escalation, add-member datalist (drops joined members; unknown-email banner preserves the typed address), access column + id search + Administrators chip. Chromium battery ran at **110/111** then **109/110** — every failure was in MY new spec, each fixed (303-with-maxRedirects-0, `option.value` vs label, datalist innerText vs innerHTML, row-text vs checkbox-state for absence assertions); the loginpage golden drift found by the battery was whitespace-only (2→3 spaces from the templ-1100 era) and was refreshed per the spec's WRITE_GOLDEN protocol after a hand diff. One rerun is still in flight. | e2e/test-results; golden commit |

## b) PARTIALLY DONE

1. **M17 — family train alignment.** The lag was enumerated (23× httputil v1.9.0 + 13× catalog v4.15.0 = the 36 pre-existing push-gate entries). The first httputil sweep correctly **ABORTED**: upstream cut a PARTIAL release — root `httputil` v1.9.0 is published but the same-family `httputil/server_timing` submodule has NO v1.9.0 tag (proxy 404), so a naive sweep would break every consumer's server_timing require. A retry with an anchored pattern (`larsartmann/httputil$`) is running as this report is written; **the catalog sweep has not started**. The strict `--strict-lag 0` gate is NOT green yet.
2. **M05 — browser battery run + triage.** Chromium mostly green with my spec's failures fixed one by one, but the FINAL clean full-battery run (all 14 specs, the ran-count gate, and the firefox advisory verdict) has not been recorded. One targeted rerun is in flight.
3. **M01 — check-modules full green.** The aggregate's rerun raced my in-flight edits and showed a mixed PASS/FAIL list; the owned legs were then fixed individually and verified standalone (branching-flow, route-postures, version-drift, lint-sweep, tidy-sweep), but ONE final settled-tree `check-modules` end-to-end pass has not been recorded.
4. **M12 — docs truth (T121 README route table, T122 screenshots).** The gotcha entry (T120) and README route-table update were NOT yet written; screenshots not refreshed. The CHANGELOG entries exist.

## c) NOT STARTED

- **M09 — tenant switcher** (super-admin header control).
- **M10 — tenant list/detail polish** (chips, sort, delete preflight, suspension impact, audit shortcut).
- **M11 — accessibility sweep** (axe spec, keyboard paths).
- **M14 — archive pass + baseline re-true** (corpus now at 58,926 vs the pinned baseline — the ratchet is red and needs the same-change ledger sentence; T142).
- **M15 filing** — the four upstream asks are RECORDED in TODO_LIST but not FILED upstream (owner-gated per repo norm; verify-before-filing applies).
- **The owner decisions D1–D3 themselves** (packet filed; answers block the follow-on waves only).

## d) TOTALLY FUCKED UP (honest list)

1. **Three careless `edit`/`multiedit` operations that broke files mid-flight**, each caught and repaired before commit but each a real bug I introduced: (a) a stray `-` prefix in front of `email := ...` in handler_members.go from a malformed old_string; (b) a users.templ edit that DELETED `userNewContent`'s opening div/form lines (restored two steps later); (c) a users.templ status-cell edit that DELETED the MFA + external-provider badges block (restored in the next edit). Root cause: composing edits from memory instead of re-reading after intervening modifications; the tool even warned "modified since last read" once and I proceeded anyway on a stale read.
2. **The branching-flow +7 scare** — my first reading treated the new findings as possibly-mine and burned a full re-run cycle before establishing they were pure anchor drift on frozen verdicts (the +7 list never referenced my new code). Better: read the finding list FIRST, then decide.
3. **The e2e spec failed four separate times for four separate rookie reasons** (maxRedirects:0 + ok(), option-value vs option-label, datalist innerText, row innerText vs checkbox state) — each fixed, but the plan's own T047 budgeted exactly this fallout and I could have avoided two of the four by checking the existing admin-behavior spec's postForm recipe before writing mine.
4. **The commit-message race**: twice the auto-commit daemon took my staged files mid-`git commit`, so two milestone commits (M02, M13-adjacent) exist only as anonymous "chore: auto-commit N files" entries; the narrative lives in the plan §9 instead of git history. One git commit also failed outright on a moved HEAD ref.
5. **A `git stash push` fired on the fresh CSS bundle** for no reason (stray command), requiring a pop to recover. Zero damage, zero excuse.
6. **Workspace-mode lint said "0 issues" three times while the hermetic lint had findings** (SA4006, exhaustruct, golines 120-col, G120) — I kept declaring victory on the workspace leg. The hermetic leg is the CI-parity leg; AGENTS says so and I still ran the wrong leg first, three times.

## e) WHAT WE SHOULD IMPROVE

1. **Re-read before every edit, no exceptions** — the tool's "modified since read" warning must hard-stop the edit, not be worked around.
2. **Write e2e specs against the existing recipe first** (admin-behavior's helpers exist precisely for this).
3. **Treat hermetic legs as the default verification**, not a follow-up.
4. **Run the finding list before the fix** on ratchet gates (branching-flow).
5. **Batch `bump-dep` awareness**: check upstream submodule publication BEFORE starting a sweep (the server_timing 404 cost a full pre-flight cycle).
6. **Verify the probe cleanup**: two `zz_probe*_test.go` files survived into a lint sweep before deletion.
7. **The corpus ratchet is red right now** (58,926 lines vs the pinned baseline) and must be re-trued WITH the ledger sentence in the same change as the archive pass (M14) — not silently bumped.

## f) NEXT 50 (ordered by the plan's own dependency graph + discovered blockers)

**Immediate (blocks pushes / gates):**
1. Land the M17 httputil root-module sweep (in flight) — verify PASS table + hermetic lint leg.
2. Decide + record the `server_timing` submodule ask (upstream must cut v1.9.0 for it; the treadmill will keep reporting until then).
3. Sweep go-cqrs-lite catalog to v4.15.0; verify the docserver templ era stays 1100-compatible.
4. Re-run `check-release-train --refresh-cache --strict-lag 0` → must report 0 lag.
5. Full workspace build + affected module tests post-bumps (T176).
6. Final clean `check-modules` run on the settled tree; record the aggregate verdict.
7. Re-true the corpus baseline WITH the ledger sentence (M14/T142) — it is red now.
8. Record the final full chromium battery verdict + `check-e2e-ran-count.sh` (T054).
9. Firefox advisory run + divergence verdict (T053).
10. Commit the e2e spec fixes + golden if the in-flight rerun is green.

**Milestone completion:**
11. M12: gotcha entry — "a Service doc comment is not the decider's contract" (T120).
12. M12: adminui README route table incl. the four new routes (T121).
13. M12: refresh demo screenshots for the new cards (T122).
14. M09: tenant switcher in the header (super-admin only; tenant-admin must not render it).
15. M10: suspended/deleted chips + status filter on the tenant list.
16. M10: sort tenants by display name and member count.
17. M10: delete preflight stating how many memberships will be revoked.
18. M10: suspension impact stating how many members lose access.
19. M10: per-tenant audit shortcut link.
20. M11: axe spec over tenant detail + user detail.
21. M11: keyboard-path test for the role checkbox groups + grant form.
22. M11: record axe results (green or documented exceptions).

**Debt/truth:**
23. M14: dry-run `archive-status` with the settled-report spec.
24. M14: apply the archive (blockquote + SUPERSEDED + manifest rows).
25. M14: docs-gates battery all five PASS after the drain.
26. M13 erratum: strike the plan's T132–T134 rows as moot (tags never existed) — recorded in §9, plan table not yet annotated.
27. Add the loginpage `buildPageData` cyclop finding to the triage ledger (sibling red, K2-recorded, not fixed here).
28. Surface the two pre-existing `dashboard-live.spec.ts` TS errors (`HTMLElement.value`) to their owning wave — flagged by BuildFlow type-check twice this session.
29. Verify `adminui` coverage ≥ 66% after ALL waves land (currently 68.2%; each wave adds tests but the threshold ratchet must be re-run at the end).
30. Re-run erraudit strict + record the new TOTAL after all waves (last recorded 330).
31. Re-run cqrs-lint strict after the catalog bump (the catalog/docserver templ-era coupling, gotcha 10-class).
32. Re-run branching-flow after M09/M10 land (more line shifts are coming).
33. Check-templates gate after any SQL/templ changes.
34. Verify the `check-skill-snippets` gate still passes (the skill's fences parse-check against the repo).

**Owner-gated (D1–D3 answers unlock):**
35. D1=GO → adminui `Config.Impersonation` + begin/end routes + banner + audit echo (zero usermgmt change).
36. D2=GO → the ONE usermgmt tag train carrying the four asks (bot listing, batched counts, tenant rename, session list/revoke).
37. D2=GO + rename → owner signs off the rename payload field-set (display name only vs slug+display) BEFORE the command lands.
38. D3=(b) → the sessions+impersonation wave plan.
39. File the four upstream asks after D2 (verify-before-filing gate applies).
40. File the `server_timing` v1.9.0 submodule publication ask upstream.
41. File the M2M packet rows if the adminui bots wave depends on them (D3=(a) path).

**Smaller improvements noticed this session:**
42. `tenantAccessRows` on the user page could carry the same "added by" attribution (M08 landed roster-only by scope discipline).
43. The import result panel's non-HTMX fallback answers with a redirect + toast only — a full-page result rendering would help no-JS admins.
44. `parseRolesForm` refuses duplicated values by dedup silently — document the choice in the doc comment (currently only the code shows it).
45. The roleCheckboxGroup legend is `aria-label` only — a visible group label would help sighted keyboard users (M11 territory).
46. `maxRenderedImportErrors` window (20) is arbitrary — a consumer knob could be justified after real imports happen.
47. The e2e helper `postForm` now follows redirects — consider asserting `res.url()` paths for the other POST assertions too.
48. `lastSeenIndex` scans the whole window per render — if directories grow past thousands of users, an indexed projection becomes worth it (verdict-level note, not planned).
49. The plan's §9 log has milestone rows but M03/M04/M07/M08 receipts land only in this report — append one-line §9 pointers for them.
50. TODO_LIST "Updated:" line still says 2026-11-08 — the freshness gate and honest docs want it trued with this wave's summary.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **D2's rename payload shape**: when the usermgmt rename command is designed, is the owner-signed field set **display-name only** (my recommendation, immutable slug stays) or **slug + display name**? This is the one irreversible event-shape decision in the gated work.
2. **The upstream `httputil/server_timing` v1.9.0 gap**: do you want me to **file the missing-submodule-tag ask upstream now** (it blocks every future root-module train bump for all consumers, not just this repo), or record it and wait for the next upstream release to self-heal?
3. **Firefox advisory depth for M05**: the plan says record the firefox verdict advisory-only. The known 5 deterministic firefox divergences are still open in TODO_LIST — should my new admin-tenant-access specs be **scoped off firefox like sync.spec** (they are chromium-first, `dialist`/checkbox assertions), or left running under firefox with expected failures recorded?

---

*Awaiting instructions. The in-flight background work (httputil sweep retry; admin-tenant-access e2e rerun) will be collected and recorded when it completes; nothing else is running.*

---

## Addendum (11:40, written immediately after the report — the two in-flight jobs completed)

1. **The anchored httputil sweep (root module only) ran but FAILED its hermetic lint leg on at least one module**, leaving the tree in a MIXED state: train lag dropped 36 → 13 (the 23 httputil entries consumed), but go.mod/go.sum across ~57 files are modified and one module's lint leg is red. The daemon has already committed the partial sweep. **The tree is currently NOT push-clean.** Next session must: find the failing module in the sweep output, fix or --no-lint with justification per the runbook, then run the catalog sweep, then `--strict-lag 0`.
2. **The final e2e rerun: 110 passed, 2 failed — both attributable to environment noise I created**: the battery ran CONCURRENTLY with the bump-dep sweep mutating examples' go.mod files, so the admin-demo server built against half-swept module state (the new failure is the id-search assertion finding 0 rows, and a sync.spec regression appeared that was green 20 minutes earlier). Do not trust these two reds as product regressions; rerun the full battery on the settled, post-sweep tree.
3. **Lesson, recorded for (e): never run the e2e battery concurrently with a dependency sweep** — the server binary and the go.mod graph it builds from are the sweep's own mutation surface. This is item 51 for the next list.

The session stops here per instruction; the three owner questions in §g stand.
