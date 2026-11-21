# Fleet Presentation Audit Session — Status Report

> **ANNOTATED 2026-11-21** — ~~every open item in this report is resolved~~ **ARCHIVED 2026-11-21 (round-23 (2026-11-21)).**
> The next session in the same arc is [`2026-11-18_07-01_fleet-plan-and-train-alignment_status.md`](2026-11-18_07-01_fleet-plan-and-train-alignment_status.md), which fleet-plan lineage continues; audit actions executed and ledgered.
> Nothing here is a live work item: it is kept as the session's evidence.

> **2026-11-17 18:05 CET · scope-first, self-critical.** Subject: the session
> that read all 21 fleet `cqrs-htmx.md` reviews and produced the docs /
> marketing / AI-skill improvement brief. Nothing else researched (per owner
> instruction). This report inventories THAT work only.

**Scope line — what this report VERIFIES (all fresh, this session):**

- All 21 review files read end-to-end (`~/projects/*/docs/review/libs/cqrs-htmx.md`, 1,930 lines; 20 consumer audits + the in-repo self-audit). File list and per-review verdict table are in the deliverable's Appendix.
- Every repo-claim re-verified with named commands, outputs recorded in the deliverable's "How to verify" block: `LICENSE` head (MIT), `sed -n '77p' README.md` (fit-test), `git show v4.18.1:htmx/htmx.go | grep HeaderRetarget` (present, added `e1bc0095` 2026-11-04), `git tag -l 'usermgmt/oauth2/*'` (v4.13.0 < go-jose patch `7fd1db43` 2026-10-28), `grep -c usermgmt dashboardui/go.mod` (0), `diff` repo-vs-fanout SKILL.md (byte-identical), skill-recipe greps (standalone-oauth2 / flash-contract / headless-dashboardui recipes absent).
- Deliverable rendered in headless Chromium (Playwright + Nix Chromium 152): structure probe returned h1 + 10 issue cards + 8 strengths + 6 sections; render receipt PNG saved beside the report.
- Gates run after the doc edits: corpus ratchet ✓ (61,035 / baseline 61,035), docs-links ✓ (1,859 links resolve), tail ratchet ✓ (budget 3, at time of run).

**Scope line — what this report does NOT verify:**

- No Go code was touched, so no build/test run happened (nothing to run; `check-modules` aggregate NOT executed — the three doc-relevant gates were run individually instead).
- Consumer-repo claims (e.g. dnsblockd's go-jose exposure, go-taskqueue's license reasoning) are taken from those repos' review files as **evidence text**, not re-derived inside the consumer repos this session.
- The +2 corpus drift pinned into the ledger was confirmed uncommitted-change-free (`git status` showed only my row) but the exact daemon-committed lines were **not** identified file-by-file.
- The aborted cmdguard interpretation (owner corrected the subject mid-session): its research (repo listing, fleet `cmdguard.md` review) is NOT reflected in the deliverable.

**Deliverable:** `docs/research/2026-11-17_docs-marketing-skill-fleet-audit.html` (+ `.render.png` receipt) — 8 presentation strengths, 10 findings (3 critical / 5 important / 2 nice), 10-step prioritized fix list, 20-row evidence table, replayable verify block. Wired: series row in `docs/research/README.md`, corpus ledger sentence + baseline true-up in the docs/status manifest.

---

## a) FULLY DONE

1. **All 21 fleet reviews read in full.** Evidence: the deliverable's Appendix table carries all 20 consumer verdicts + scores (adopters: audiblesync 77, dnsblockd 100, journal 86, browser-history 80, Code-To-CV-Agent 92, quotaboard 75, webphone 100, pbx-lens 78, PapDashboard 6; non-adopters: 11) plus the self-audit (87/95). Scope: read-only across `~/projects/*`.
2. **HTML improvement brief written and shipped in-repo.** `docs/research/2026-11-17_docs-marketing-skill-fleet-audit.html` (1,600 lines, kit-css intact — copied from the editorial template, never transcribed). Evidence: HTML tag-balance check (only the template's two self-closing `<meta/>` parser artifacts), Chromium render probe, receipt PNG (142 KB).
3. **Three critical findings verified to the commit/tag level:** (F1) LICENSE is MIT while two consumer reviews recorded "proprietary"; (F2) go-jose JWK DoS patch `7fd1db43` (2026-10-28) sits above the oauth2 train's last tag v4.13.0 (2026-10-11) — live today; (F3) `htmx.HeaderRetarget`/`HeaderReswap` exist at `htmx/htmx.go:27-28` in v4.18.1, contradicting audiblesync's `go doc`-based "no constants" verification.
4. **Series wiring complete:** `docs/research/README.md` gained the "Fleet presentation audit" row; corpus ledger sentence appended to the docs/status manifest; `CORPUS_BASELINE` re-trued 61,031 → 61,035. Evidence: gate outputs quoted in the scope line.
5. **One shipped-error correction executed during THIS review pass:** the report's summary sentence "Nine adopters run the library at 75–100/100" was wrong (PapDashboard = 6/100); fixed to "eight of nine at 75–100, the ninth a deliberate single-symbol consumer at 6/100" and the render receipt re-captured. Evidence: line 926 edit + fresh Chromium probe after the edit.

## b) PARTIALLY DONE

1. **The 10-step fix list is authored, not executed** (0/10 landed). What exists: each step has impact × effort × closes-what in the deliverable's fix table. What remains: every step. Blocker: awaiting owner instruction (this session was report-scoped). Effort per step S–L, listed in the table.
2. **Errata on consumer reviews (F1/F3) are specified, not written.** They live in OTHER repos (go-taskqueue, business-networking, audiblesync); cross-repo edits were deliberately not started without instruction. Effort: S each.
3. **HARVEST of §f into TODO_LIST.md is pending.** The status-report skill mandates closing the loop; `TODO_LIST.md` is currently showing uncommitted modifications from another session (collision risk, gotcha 47), so harvest waits on an owner call (§g Q3). Effort: M.
4. **The F2 consumer side** (dnsblockd raising its go-jose indirect pin) is dnsblockd's row, documented in its review; not actionable from this repo beyond the upstream re-tag.

## c) NOT STARTED

1. All ten in-repo fix-list items as actual changes: `usermgmt/oauth2/v4.13.1` cut (release pass); README license line; README dependency-footprint table; README release-train table; dashboardui README "runs without usermgmt" receipt + headless example; skill "Standalone module recipes" section; skill Discoverability-notes constants addition; changelog receipt rule in the release checklist; `httperr.FlashErrorHandler`; metaengine→dashboardui bridge guide/example. Why: report-first session; owner instruction to stop after the status report. Priority: steps 1–3 marked next-session material in the report.
2. TODO_LIST/ROADMAP routing rows for the §f brainstorm tail (items beyond the top 10). Why: HARVEST not yet run. Priority: next docs-health round.
3. A cmdguard / other-library variant of this presentation audit. Why: the owner corrected the subject away from cmdguard mid-session; dropped by design. Priority: unknown — owner question implicit.

## d) TOTALLY FUCKED UP

1. **Wrong subject on the first pass (mine).** I read "cmdguard's docs and marketing" literally and started researching the cmdguard repo before the owner corrected: the subject was cqrs-htmx itself. Severity: wasted 2–3 tool calls; no wrong artifact shipped (correction landed before any write). Root cause: the named subject contradicted the evidence base (21 reviews are all about cqrs-htmx) and I resolved the contradiction by following the noun instead of flagging it. Mitigation: none needed downstream; the lesson is recorded in §e-1.
2. **A wrong arithmetic claim shipped inside the deliverable** ("nine adopters at 75–100/100" — PapDashboard is 6/100). Severity: low (one summary sentence, prose-only), but it was MY OWN number, derivable from my own Appendix table, and it survived my write-time self-check. Fixed during THIS status pass (a-5). Root cause: prose numbers were not re-derived from the table they summarize.
3. **The first `edit` to the HTML failed on a stale-read guard** (file mtime moved between my view and my edit — the daemon touches the tree continuously; gotcha 47's ambient churn). Severity: none (re-read + retry succeeded). Noted so the next session doesn't misread it as a content conflict.
4. **Unidentified corpus drift (+2 lines) pinned into the baseline.** The ledger records it as daemon-committed sibling drift, honestly, but without naming the files. Severity: hygiene only — the ratchet is green either way. Mitigation: none; a `--files` before/after diff at pin time would name them.

## e) WHAT WE SHOULD IMPROVE

1. **One-line subject confirmation when noun and evidence base disagree.** Cost of skipping it this session: ~3 wasted calls and a mid-flight correction. Fix: if the user names a subject and the requested evidence clearly points elsewhere, state the mismatch and proceed with the evidence base (or ask) BEFORE researching.
2. **Re-derive prose numbers from their tables before shipping.** The §d-2 slip is a class, not an incident. Fix: a personal write-time checklist step — every count in a summary sentence gets re-derived from the table it summarizes.
3. **Receipt-rendering should be a flake app, not a temp script.** This session hand-rolled `render-receipt.tmp.mjs` in `e2e/` (Playwright + Nix Chromium path discovery included). The fleet already treats render PNGs as receipts (`index.render.png`). Fix: a small `.#render-receipt` flake app wrapping the Playwright call; deletes the temp-script class.
4. **Cross-repo errata have no documented path.** F1/F3 corrections belong in three foreign repos; the fleet's erratum culture is per-repo only. Fix: one paragraph in `docs/review/libs/FORMAT.md` ("errata to consumer reviews are written in the consumer repo, cited back here").
5. **Summary claims in reports should carry their denominator.** "11 of 20" survived because it carried one; "nine adopters at 75–100" failed because it didn't. Same fix as e-2, stated as the rule.
6. **Heavy verification gates vs docs-only changes.** Full `check-modules` was skipped for a docs-only change set; the three relevant gates ran individually. That was the right call, but it's tribal knowledge — noting it here so the choice is visible and repeatable.

## f) Top 50 things to get done next

Ranked by impact; effort S (<30min) / M (30min–2h) / L (>2h). Items 1–10 are the deliverable's fix list (repeated here so this section is the complete harvest input); 11–50 are session-observed follow-ons and are **brainstorm/ROADMAP fuel** per the status-report skill (HARVEST must route them with extra rigor, not auto-commit them to TODO_LIST).

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Cut `usermgmt/oauth2/v4.13.1` carrying the go-jose DoS patch; publish the consumer sweep recipe in its changelog entry | Critical | M | Release |
| 2 | Add the license disambiguation line (MIT; "each sibling repo's LICENSE is authoritative") to README first screen + the policy tables consumers consult | Critical | S | Documentation |
| 3 | Add `htmx.HeaderRetarget`/`HeaderReswap` to the skill's Discoverability notes + cross-link from the Response-builder `Retarget()` bullet | Critical | S | Documentation |
| 4 | Write errata: audiblesync review (constants row), go-taskqueue review (license row), business-networking review (license row) | High | S | Documentation |
| 5 | dashboardui README: "Runs without usermgmt" receipt (go.mod citation) + headless/API-key mounting example | High | S | Documentation |
| 6 | README dependency-footprint table (per module: direct requires, closure size, Go floor), generated in the release checklist | High | M | Documentation |
| 7 | Consumer-facing release-train table in README § Versioning (nested→root tag mapping, flake-input rule, security-lag escalation path); link from nested READMEs | High | M | Documentation |
| 8 | Skill "Standalone module recipes" section: oauth2-without-usermgmt (credit dnsblockd), HTMX flash error contract (credit audiblesync), dashboardui headless (credit index), metaengine→dashboardui (credit invoices/CV) | High | M | Documentation |
| 9 | Changelog receipt rule (diff-verify every named symbol at tag time) in the release checklist / release-train gate | Medium | M | Quality |
| 10 | `httperr.FlashErrorHandler` (200 + retarget-to-flash for HTMX clients) or documented Response-builder recipe | Medium | M | Feature |
| 11 | Sweep webauthn/totp trains for the same go-jose closure while doing #1 | High | S | Release |
| 12 | Metaengine-facts → dashboardui bridge guide or `examples/` spike | Medium | L | Feature |
| 13 | HARVEST this report's §f into TODO_LIST (top ~10) and ROADMAP (brainstorm tail) | High | M | Process |
| 14 | FORMAT.md: cross-repo errata convention (e-4) | Medium | S | Documentation |
| 15 | `.#render-receipt` flake app (e-3) | Low | M | Tooling |
| 16 | Re-run corpus/links/tail gates after the daemon commits this report's wiring (the +drift classes keep moving) | Medium | S | Process |
| 17 | Verify skill fan-out sync after any skill edit (rule 2 of the skill-authoring norm) — ride along with #3/#8 | Medium | S | Process |
| 18 | Add per-module Go floor to the README module table while doing #6 | Medium | S | Documentation |
| 19 | CHANGELOG note on the next train correcting the v4.13.0 overstated receipts (dnsblockd erratum upstream-facing half) | Medium | S | Documentation |
| 20 | Release-train gate extension: security-dep-bump-in-nested-closure → nested re-tag reminder (mechanize #1's rule) | Medium | M | Quality |
| 21 | Re-check `usermgmt/oauth2` tag state at next session start (if #1 landed meanwhile, close F2 in the deliverable with a receipt) | Medium | S | Process |
| 22 | Fix the deliverable's F1 wording once Q1 (license intent) is answered | High | S | Documentation |
| 23 | Route #10/#12 verdicts: v4 minor vs v5 window (one-line ROADMAP decision) | Low | S | Process |
| 24 | Annotate the self-audit's "prior reports" with the presentation-audit link | Low | S | Documentation |
| 25 | Consider a `docs/review/libs/` index row/column noting "presentation findings open" so the fleet audits see F1–F3 status | Low | S | Documentation |
| 26 | Decide whether PapDashboard-class single-symbol consumers deserve a dedicated "asset-only" install path in v5 (the one-symbol-closure complaint has now been recorded by two audits) | Medium | M | Feature |
| 27 | Re-derive "how many fleet reviews quote the fit-test" with the exact grep in the deliverable and pin the count next to README:77 | Low | S | Documentation |
| 28 | Check whether `go doc` output ordering/cache explains the audiblesync false-negative (would upgrade F3's root cause from hypothesis to fact) | Low | S | Quality |
| 29 | Give the deliverable's evidence table stable anchors (#review-<repo>) so future errata can link precisely | Low | S | Documentation |
| 30 | Mirror the "correctly-not-adopted" outcomes into FEATURES.md's WORTH CONSIDERING rows where the fix list changes them (dashboardui identity-free story) | Low | S | Documentation |
| 31 | Skill: state the section-title stability rule explicitly (consumers cite sections by name — additive-only edits) | Medium | S | Process |
| 32 | Skill: add the flash/retarget recipe BEFORE #10 ships so the API and the recipe land in one train | Medium | S | Documentation |
| 33 | README: link the presentation-audit report from the Documentation section (fleet-facing proof of the audit culture) | Low | S | Documentation |
| 34 | Verify the oauth2 sweep recipe actually resolves v4.1.6+ in a throwaway consumer module after #1 lands | High | S | Quality |
| 35 | Draft the dnsblockd-facing erratum text for its review's F2 row once v4.13.1 exists | Medium | S | Documentation |
| 36 | Check whether nested modules usermgmt/webauthn or usermgmt/totp carry go-jose at all before writing #11's sweep (may be a no-op) | Medium | S | Quality |
| 37 | Capture a fresh full-page render receipt (current PNG is viewport-only; fullPage would show all findings) | Low | S | Tooling | <!-- resolved 2026-11-21: done — P16 regenerated the receipt as a 1440x9517 fullPage PNG (palette-compressed 1.6 MB -> 654 KB under the binary guard), and the recipe is codified as the .#render-receipt flake app (docs/runbooks/render-receipt.md); also resolves §f62 below |
| 38 | Add the report's "How to verify" block to a replay schedule (the B10 lesson: replay catches rot within a day) | Medium | S | Process |
| 39 | Consider recording e-1/e-2 in `references/lessons.md` (crush-config repo) as cross-project lessons — they generalize | Low | S | Process |
| 40 | Next presentation-audit refresh trigger: after the next family train lands, re-verify F2/F3 receipts and move the audit date | Medium | S | Process |
| 41 | Check that `check-docs-links` covers HTML files or document that it doesn't (this session's internal links were verified by hand) | Low | S | Tooling |
| 42 | Evaluate whether the skill's Path decision tree should gain a Path for "dashboardui-only" now that F5 makes it a real adoption shape | Medium | S | Documentation |
| 43 | De-duplicate the two CV-family reviews (cv vs CV) at the next fleet refresh — same verdict, two files, one project family | Low | S | Cleanup |
| 44 | Add "audit-date stamps must move" as an explicit checklist line for research-series HTML reports (FORMAT.md covers md reports; HTML series has no such rule written) | Low | S | Documentation |
| 45 | Prove the release-train gate catches a stale nested tag TODAY (fixture or dry-run) before relying on it for #1 | Medium | M | Quality |
| 46 | Consider surfacing the dependency-footprint table in `setup` docs too (setup consumers inherit the whole closure — the strongest footprint case) | Low | S | Documentation |
| 47 | Read-back check: next session re-runs the deliverable's verify block literally (the fleet's replay rule) before trusting its receipts | Medium | S | Process |
| 48 | Record the render-receipt Chromium path (nix store hash changes on bump — read it from `nix build .#e2e-chromium` if it exists, else re-derive) in the runbook, not a temp script | Low | S | Documentation |
| 49 | Ask whether the fleet's non-adopters should see the fix list at all (a one-line pointer in FORMAT.md: "presentation findings live here") | Low | S | Documentation |
| 50 | Book the follow-through session: steps 1–3 of the fix list are ≤1 day combined and close every critical finding | High | M | Process |

**HARVEST note (skill mandate):** items 1–10 + 13 + 50 are the TODO_LIST-class rows; the rest route to ROADMAP or stay in this file as brainstorm. Harvest itself is pending the owner call in §g-Q3.

## g) Questions I cannot answer myself

1. **License intent (drives F1's wording):** Is cqrs-htmx deliberately MIT while go-cqrs-lite is proprietary (as two consumer reviews claim), and do you want README/policy surfaces to state that divergence explicitly? I verified cqrs-htmx's LICENSE is MIT; I cannot know the intent, and I deliberately did not research go-cqrs-lite's license this session (out of scope).
2. **Release policy (drives F2's shape):** Should the oauth2 security re-tag (`v4.13.1`) be cut as a standalone hotfix now, or ride the next regular family train? I know the mechanics and the consumer recipe; the hotfix-vs-train call is release policy only you hold.
3. **Harvest timing (drives §b-3):** `TODO_LIST.md` currently carries uncommitted edits from another session — do you want §f harvested now (collision risk, gotcha 47), or should I wait for the next docs-health round?

---

*Generated 2026-11-17 18:05 CET · session scope: fleet presentation audit only · no commits made (harness forbids; auto-commit daemon picks this file up) · WAITING FOR INSTRUCTIONS.*
