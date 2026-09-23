# Status: templ-components Deep-Dive Session — dashboardui Audit

> **Timestamp:** 2026-09-23 01:38 CEST
> **Session scope:** library-deep-dive audit of templ-components usage in `dashboardui/`
> **Deliverable produced:** [`docs/research/2026-09-23_templ-components-deep-dive.html`](../research/2026-09-23_templ-components-deep-dive.html) (corrected same session, see §d)
> **Format note:** user explicitly requested `.md`; skill default is HTML — override honored, not propagated.

> ANNOTATED 2026-09-23 (same-day execution session): the §f plan (24 tasks / 66 micro-tasks) was executed end-to-end after the owner's go. Resolved: #1 (`4b1e82aa`), #2 (`ee14dba9`/`e722ddcb`), #3–#18, #20–#22, #24–#26, #28–#34 (each struck inline below; evidence in the audit report §06/§07, ADR-0053, `docs/research/README.md`, loginpage mini-report). Open: #19 (axe pass), #27 (dark-token audit) — routed to TODO_LIST P3; #23 — options memo drafted, owner decision pending. §g answers: Q2 answered by events (toggle shipped, adminui pattern); Q3 remains owner policy (verdicts documented as divergences in ADR-0053 with reopen triggers).

---

## Self-Review (brutal, first — because it reshapes the sections below)

**What did you forget?**
1. **The prior art.** `docs/research/` already contained `2026-09-17_templ-components-deep-dive.html` (adminui, 68/100), `2026-09-17_templ-components-dashboardui-deep-dive.html` (dashboardui, **14/100** pre-migration), and `2026-09-20_theme-toggle-strategy-spike.md`. I ran a fresh audit without listing the directory. My "85/100" lands out of context — the real headline is **14 → 85 in six days** (post full-templ migration), and my ThemeToggle "finding" was already spiked three days earlier. Fixed: series-context paragraph added to the report.
2. **Claim verification before printing.** The report shipped saying "17 non-test files" — I never counted. True count: 12 handwritten + 10 generated. Also said "five imported subpackages" then listed seven packages (five go.mod module paths span seven import packages). Both corrected.
3. **The library skill's consumer tip** — track adoption in a grep-able table in the consumer's docs. Read it, didn't do it.
4. **Rubric for the score.** 85/100 has no written rubric; the 2026-09-17 series at least enumerated capabilities (1/22). Mine is a judgment, not a measurement. Disclosed, not yet fixed.

**What could you have done better?**
- Run `go test ./dashboardui/` (or at least build/vet) as audit evidence — the audit verified claims by reading, never ran the module.
- Skim the prior reports FIRST (the deep-dive skill says cross-reference the series; I executed phases 1–7 as if episode one).
- Answered in chat earlier that `config.go` "imports" templ-components — it only mentions it in a comment. Minor, but it was wrong.

**What could you still improve?**
- Anchor adoption scores to an explicit capability rubric so series scores are comparable (14 → 68 → 85 currently use different rubrics).
- Fold the theme-spike's decided strategy into finding #2 instead of re-recommending from scratch.
- Verify counts mechanically in the report-writing step, not from memory of grep output.

**Did you lie?** No — but two claims (file count, "available at upgrade time") were printed without evidence, which is the same failure class. Both now corrected or softened in the report.

---

## a) FULLY DONE

| Item | Evidence |
|---|---|
| Answered "does dashboardui use templ-components?" — yes, 5 module paths @ v1.19.2 | `dashboardui/go.mod:23-27`; `git -C templ-components tag` head = v1.19.2 |
| Full utilization audit: ~90 distinct library identifiers censused across 12 handwritten + 10 generated files, 7 import packages | `grep -rhoE \| sort -u \| wc -l` = 90 (session log) |
| Gap analysis with evidence: 3 partial adoptions, 2 optional candidates, 3 justified divergences, 0 anti-patterns | Report findings §01, each with file:line |
| CSS bundle health verified | `nix run .#check-css-bundles` exit 0; 76,984 bytes, canaries present |
| Version currency verified (0 behind) | tag comparison, report §02 table |
| HTML report written + spliced from canonical template + parses | `docs/research/2026-09-23_templ-components-deep-dive.html`, 50 KB, html.parser OK |
| Report self-corrected after review: 3 claim fixes + series-context paragraph | edits at report lines ~930, ~1215, ~1348, summary section |

## b) PARTIALLY DONE

| Item | Works | Open | Effort |
|---|---|---|---|
| Audit actionability | Findings + prioritized fixes exist | None of the 4 recommendations implemented; no rubric; theme finding not reconciled with the 2026-09-20 spike | S–M |
| Series reconciliation | Prior reports located, overlap characterized, cross-links added | Still-open items from the 2026-09-17 report's 17 missed opportunities not individually re-verified | M |
| Score methodology | Score produced with rationale | No capability rubric; not comparable to series' 14/68 baselines | S |

## c) NOT STARTED

- Implementing `htmx.CSRFToken` swap (7 sites: dlq ×4, projections ×2, snapshots ×1).
- `ThemeScript`/`ThemeToggle` + `@custom-variant dark` adoption (gated on spike reconciliation + want-decision).
- `forms.Slider` and `navigation.SidebarNav` evaluations (prototype + visual verdict).
- Adoption table in dashboardui docs (library skill's consumer tip).
- HARVEST of §f into `TODO_LIST.md`/`ROADMAP.md` (waiting for instructions per session contract).
- Equivalent audits are DONE for adminui (68/100, 2026-09-17) — no loginpage audit exists.

## d) TOTALLY FUCKED UP

Nothing in the repo is broken by this session (read-only audit + new files; CSS gate green). The fucked-up list is about the session's own work quality:

1. **Duplicated an existing research series without checking.** Severity: wasted-context class; produced a report that reads as episode one when it is episode three. Root cause: skipped the skill's "read prior reports in the series" step. Mitigation: shipped (cross-links + series paragraph added same session).
2. **Printed two unverified claims in a report whose entire premise is "every claim verified".** Severity: credibility of the deliverable. Root cause: wrote from memory of grep output instead of re-grepping. Mitigation: corrected within the same session (see §a last row).
3. **Chat-level inaccuracy:** cited `config.go` as an importer; it is comment-only. No file fix needed (report never made this claim); noted for honesty.

## e) WHAT WE SHOULD IMPROVE

1. **Deep-dive series protocol:** always `ls docs/research/ | grep <library>` before Phase 1. Cheap gate, prevents episode duplication. (Candidate: one line in the library-deep-dive skill or project AGENTS.md.)
2. **Evidence discipline for reports:** any count/number in a written deliverable gets re-run at write time, not quoted from scrollback.
3. **Score rubrics:** define the capability list once (the 2026-09-17 report's 22-capability ladder is a good base) and reuse it so series scores trend comparably.
4. **Audit = read + run:** add module build/test to the audit evidence checklist.
5. **Consumer adoption tables** (per-package adopted/custom/hand-rolled) in each UI module's docs — the library skill recommends it; would have made this audit 10× faster.

## f) NEXT TASKS (up to 50 — honest version: ~35 real, rest would be padding)

Impact: C=Critical, H=High, M=Medium, L=Low. Effort: S<30min, M=30m–2h, L>2h. Route: T=TODO_LIST, R=ROADMAP.

| # | Task | Impact | Effort | Cat | Route |
|---|---|---|---|---|---|
| 1 | ~~Swap 7 hidden CSRF inputs → `htmx.CSRFToken` (dlq/projections/snapshots)~~ done at `4b1e82aa` (also fixed the `_csrf` vs `csrf_token` field-name mismatch) | M | S | Quality | T |
| 2 | ~~Reconcile ThemeToggle finding with `2026-09-20_theme-toggle-strategy-spike.md`; then land ThemeScript+Toggle+`@custom-variant dark` + CSS rebuild~~ done at `ee14dba9`/`e722ddcb` (shipped adminui class-driven pattern; spike folded in report §06) | H | M | Feature | T |
| 3 | ~~Harvest this §f into TODO_LIST/ROADMAP (docs-health)~~ done (survivors only: #19/#27 + RelativeTime swap + PageHeader gap; OQ22 added) | H | S | Docs | T |
| 4 | ~~Add adoption table (adopted/custom/hand-rolled) to dashboardui docs~~ done (README table + AGENTS.md pointer) | M | S | Docs | T |
| 5 | ~~Re-verify the 2026-09-17 report's 17 missed opportunities; strike resolved, carry open~~ done (inline ANNOTATED outcome note; 6 resolved + theme same-day, 3 partial/deliberate) | M | M | Docs | T |
| 6 | ~~Define shared adoption-score rubric from the 2026-09-17 capability ladder; re-score 14/68/85 on it~~ done (rubric v1 + series re-score 14→79→83→94, report §06; provenance correction: "85" was never computed) | M | M | Quality | T |
| 7 | ~~Run dashboardui build+tests as audit evidence; attach to report appendix~~ done (report §06 evidence table) | M | S | Quality | T |
| 8 | ~~Prototype `forms.Slider` for time-travel scrubber; adopt or document rejection~~ done (rejected: labeled wrapper vs compact scrubber row, §07) | L | S | Feature | T |
| 9 | ~~Prototype `navigation.SidebarNav` in aside; adopt or document rejection~~ done (kept per standing TODO_LIST P2 criteria; divergence documented in ADR-0053 + §07) | L | M | Feature | R |
| 10 | ~~Evaluate `forms.FilterInput`/`FilterDropdown` vs hand-rolled `filterInput` (events.templ)~~ done (rejected: single-field form semantics vs 3-field AND bar, §07) | M | S | Feature | T |
| 11 | ~~Evaluate `display.RelativeTime` vs Go `relativeTime()` helper in table rows~~ done (adopt-candidate, narrow server-rendered swap queued to TODO_LIST; live JS rows excluded) | L | S | Cleanup | T |
| 12 | ~~Evaluate `display.Grid` for overview stat-card row (currently custom CSS grid)~~ done (kept `.panel`/stat-grid token styling, §07 Card/Grid verdict) | L | S | Cleanup | R |
| 13 | ~~Evaluate `display.Card`/`SimpleCard` vs `.panel`/`.panel-title` CSS~~ done (keep `.panel` — 4 usages, token-driven, zero-gain swap risk, §07) | M | M | Cleanup | R |
| 14 | ~~Evaluate `htmx.ConfirmDelete` vs raw `data-confirm` forms~~ done (rejected: one delegated listener beats per-form pairing, §07) | L | S | Feature | T |
| 15 | ~~Check `paginationInfoText` renders via `ListNote`/`ListNoteCount` where applicable~~ done (blocked on upstream X–Y range variant; ask already filed, §07) | L | S | Quality | T |
| 16 | ~~Consider `errorpage.ErrorDetail` for the HTMX-swap bare error card (render.go)~~ done (no fit — bare ErrorPage fragment is correct for swaps, §07) | L | S | Feature | R |
| 17 | ~~Sweep for bare "No X" text blocks not using `emptyStatePanel`~~ done (clean — all surfaces route through emptyStatePanel) | L | S | Quality | T |
| 18 | ~~Add goldens for any newly adopted components after #1/#2~~ done (no new consumer goldens required — both components' markup pinned upstream; module goldens + a11y suite green) | M | S | Quality | T |
| 19 | axe/a11y smoke pass once theme toggle lands | M | M | Quality | R |
| 20 | ~~Link the deep-dive report from `dashboardui/README.md` (docs section)~~ done (README links ADR-0053 + audit series; `docs/research/README.md` indexes the series) | L | S | Docs | T |
| 21 | ~~Run loginpage templ-components audit (AGENTS.md lists 4 adoption opportunities)~~ done (`docs/research/2026-09-23_loginpage-templ-components-audit.md` — keep zero-dep; OQ21 triggers unchanged) | M | M | Docs | T |
| 22 | ~~Re-run dashboardui audit post-#1/#2; ANNOTATE this report (docs-health)~~ done (census re-run + this annotation block) | M | S | Docs | T |
| 23 | Decide publish-vs-internal for audit verdicts on the public repo README — options memo drafted (`docs/planning/2026-09-23_audit-verdict-publishing-memo.md`, recommends A-with-exceptions); OWNER DECISION PENDING | M | S | Docs | — (needs §g) |
| 24 | ~~Add `ls docs/research/ first` step to session protocol / AGENTS.md note~~ done (AGENTS.md gotcha 21) | M | S | Process | T |
| 25 | ~~Check `icons.Name` constants vs string literals drift in navIcon (layout.go:22 comment)~~ done (no drift — all names flow through mapNavIconName → typed constants with Question fallback) | L | S | Quality | T |
| 26 | ~~Verify `check-templates` still green after any templ edits from #1~~ done (green post-swap) | M | S | Quality | T |
| 27 | Dark-mode audit of dashboardCSS tokens once class strategy lands (#2) | M | M | Quality | R |
| 28 | ~~Consider vendoring `.tc-*` utilities need-assessment for dashboard (library custom.css)~~ done (no vendoring needed — no Modal/Drawer/Dropdown/Combobox/Textarea adopted; tc-copy/toast hooks styled by utilities + dashboardCSS bridge) | L | S | Quality | R |
| 29 | ~~Confirm `RecommendedSecurityMiddleware` coverage is complete for all dashboard handlers~~ done (`Dashboard.Middleware()` delegates to it; opt-in by library principle; setup + demo wire it; README documents the consumer Chain) | M | S | Quality | T |
| 30 | ~~Add Scrollback component evaluation for event/payload viewers (fit?)~~ done (no current fit — payload is a blob, live feed is a table; revisit on a raw-log page, §07) | L | S | Feature | R |
| 31 | ~~Consider `display.DataTable` (integrated sort+pagination) vs current Table+sortState composition~~ done (rejected: data-driven shape fights inline templ cells; sortable headers already shipped, §07) | M | M | Feature | R |
| 32 | ~~Document the 3 justified divergences (shell/pagination/error-card) as ADR or README note~~ done (ADR-0053 + INDEX + README link) | M | S | Docs | T |
| 33 | ~~Annotate the 2026-09-17 dashboardui report: superseded-by link to today's 85/100~~ done (inline ANNOTATED outcome note) | M | S | Docs | T |
| 34 | ~~Update repo CHANGELOG/docs index if reports are meant to be discoverable (status README lists them?)~~ done (`docs/research/README.md` convention + load-bearing series index; CHANGELOG entries this train) | L | S | Docs | T |
| 35 | ~~Decide whether audits of all UI consumers become a recurring gate (post-migration regression check)~~ done (proposed as ROADMAP OQ22 — lightweight post-wave ritual, not a CI gate; owner call) | L | S | Process | R |

(Honest cut: items 36–50 would be filler — the real backlog from this session is the 35 above; #23 and §g gate a few.)

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Wave or single-module?** Should the dashboardui fixes (#1, #2) land now as a dashboardui-only change, or do you want one combined adoption wave across dashboardui + loginpage (adminui already audited) in a single train?
2. **Is a user-facing theme toggle actually wanted for dashboardui?** The spike (2026-09-20) and the tailwind.css comment say "planned", but for an embedded observability dashboard, OS-follow-only may be the intended product decision. Which is it?
3. **Is visual continuity with the dashboard's own token stylesheet a hard requirement?** It decides items #9/#12/#13/#31: if library classes may restyle these surfaces, several hand-rolled patterns collapse into library components; if not, they stay custom by policy.

---

*Point-in-time snapshot. §f executed same day (see the ANNOTATED block); survivors harvested to TODO_LIST/ROADMAP.*
