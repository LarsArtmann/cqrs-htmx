# Status Report — docs-health Round 13: Full Audit, TODO Upgrade, 22-Report Annotate+Archive Sweep

> ANNOTATED 2026-10-01 (self-check at write time): this is the round-13 pass's own report — its §a claims are gate-verified in the same session (annotations 70/70, rows 0 PARTIAL, links 315 OK, freshness OK, check-modules 17/17 incl. strict release-train), its §b/c open items are routed in TODO_LIST P1–P3 as cited, and §f is the harvest receipt. The live tail is THIS file plus the 10-01 02:59 BuildFlow-recovery report.
>
> ANNOTATED 2026-10-01 (docs-health round 14 — same-day archive pass): §c1 DONE hours later — the family train shipped (9 tags: usermgmt v4.13.0 codec migration + v4.13.1, root v4.13.0, setup v4.13.2, adminui/dashboardui/health/loginpage/systemadapter v4.12.1; strict gates 0 lag / 0 unpublished at every push; proxy smoke green; CI green run 36839790817). §c3 DONE at `c85d802f` (T07, full atomic checklist; the gate's first CI run then caught the machine-local-replace class BY DESIGN, fixed via the consumer-view build `eed53317`). §c7's test+coverage half DONE (T03 quiet window: race suite rc=0 ×28 modules at load 5.9, coverage 15/15) — e2e + bench-spike remain the TODO P2 battery remainder. §f1–f4, f9–f11, f50 struck with receipts below. §b3/§b7 and §f40/f42 resolved by THIS pass: the six 2026-08-30 planning files triaged (upstream-issue-drafts archived — its 4 asks were filed 09-10; the gated-work index + cqrs-lint distribution draft annotated; the rest confirmed live gated decisions, leave-in-place) and the 12-file HTML status corpus keep-in-place decision confirmed in `docs/status/README.md`. §c2's erraudit program is IN PROGRESS: trivial pair done at `3d9ce553`, the gate counts 61 unique context_loss sites with 36 fixed; TODO P2 carries the 25-site remainder. §g1–g3 remain owner questions; this archive proceeds on the same bar round 13 used (§g1).

> **Session:** 2026-10-01 ~03:40 → 04:35 CEST (report written 06:36 CEST)
> **Scope:** this session only — the full docs-health AUDIT (BUILD + HARVEST + VERIFY + ANNOTATE + ARCHIVE) over the living docs and the entire unarchived status tail (2026-09-21 → 2026-10-01). No unrelated research; no code changes (docs-only session by design).
> **Trigger:** owner instruction — "View ALL \*\*/2026-0\* files! Execute the docs-health SKILL! … ESPECIALLY the TODO_LIST needs an upgrade."
> **Machine context:** shared fleet box under EXTREME load (194 → 51 across the session); the heavy `.#test`/`#coverage-gate`/e2e battery was deliberately deferred rather than gambled — `check-modules` ran at load ~50–85 and passed.

---

## Executive verdict

The docs surface went from "22 unarchived reports, 4-day-stale TODO header, 4 missing CHANGELOG entries, 3 executed plans still in the active planning dir" to: **1-report tail, 435 archived reports all gate-clean, TODO_LIST rebuilt with evidence-cited items, CHANGELOG backfilled, all doc gates + check-modules green.** The session's most important catches: the M18 P1 item had been DONE for 4 days while still topping the TODO; the CHANGELOG claimed a go.work removal-condition document that did not exist (now it does); four completed dependency sweeps were entirely unrecorded in CHANGELOG; and the system cqrs-lint binary is still pre-fix (TODO item verified still-open, not assumed).

## a) FULLY DONE (verified this session)

1. **Skill + corpus ingestion.** docs-health SKILL.md loaded; living docs read (TODO_LIST, CHANGELOG [Unreleased], docs/status/README); 10 recent reports read in full (2026-09-26 → 10-01 arc); 4 parallel agent sweeps covered the remaining 14 unharvested reports (line-numbered open-item maps) and the repo-state verification (tags, ROADMAP OQs, CHANGELOG presence greps, agents-notes gaps, go.work pin).
2. **TODO_LIST upgraded (the owner's emphasis).** Header date-stamped honestly: lint re-verified 2026-10-01 (all per-module golangci steps green in the BuildFlow 601/601 run), coverage explicitly "NOT re-run since 2026-09-29", check-modules green (re-run this session, incl. strict release-train 0 lag / 0 unpublished). The M18 item — done 2026-09-27 but still topping P1 — removed. The erraudit item restructured (the 2 trivial fixes split out as first moves). **9 new evidence-cited items added**: the usermgmt re-tag train (new P1 — `usermgmt/v4.12.0` is still the newest tag while setup shipped v4.13.0+v4.13.1; the codec migration is PapDashboard's recorded adoption prerequisite), `check-workspace-build` gate, the post-change verification battery, the 4-ask fleet upstream bundle, the consolidated cqrs-lint residual/noise triage, the go-cqrs-lite cross-repo debt (owner-gated), and a 10-item tooling micro-batch. Every item cites file:line / report / tag evidence.
3. **CHANGELOG backfilled (4 missing entries, all [Unreleased] Changed/Fixed):** the 2026-10-01 BuildFlow 14-step recovery + erraudit surfacing (templ wrapper, go-licenses shim, allow-serial-runners ×16, go-etag pin restore + 19 dead-replace prune, ErrSnapshotNotFound, datastar-demo broadcast migration); the 09-30 release-train alignment (8 family sweeps, 21 lags); the 09-27 templ-components v1.19.4 + go-health v0.4.1 sweeps; and the never-recorded 2026-09-22 dedup-to-zero sweep (9 groups → 0 harmful, −147 lines).
4. **FEATURES.md:** two new FULLY_FUNCTIONAL rows (setup Identity-External Shell per ADR-0054; Ownership Seams — ExtraMiddleware/DisableSecurityMiddleware/HealthChecks); header refreshed (version state: root v4.12.0 + setup/v4.13.1 live, root/usermgmt re-tag pending; setup coverage corrected 86.8% → 87.9% per the 09-29 run).
5. **ROADMAP:** OQ26 opened — fleet go-directive policy (`go 1.27.1` patch pin vs `1.27`, the go-health-v0.4.1 anti-pattern disagreement, go-line-flipflop warnings). OQ16/21/23/24/25 + the three 09-28 raw exploitation ideas verified present.
6. **go.work:** the load-bearing `go-etag => v0.6.0` pin now carries its removal-condition comment in place (AGENTS gotcha 22c pointer) — making the existing CHANGELOG claim true; parse-verified via `go work edit -json`.
7. **ANNOTATE pass — 22 reports.** Every unarchived 2026-09-21→09-30 report got a dated `> ANNOTATED 2026-10-01 (docs-health round 13)` blockquote with per-section verdicts, plus ~120 evidence-backed inline strikes (pushes → CI run 36382941958 receipts; bench-spike → round-12 quiet-window pass; M18 → the 09-27 sweeps; C040 → the linter-source fix; WithChildren → round-12 production adoption; vendorHash alarm → disproven; …). The two mid-session cqrs-lint reports marked SUPERSEDED-BY the 08-03 close-out. The live 10-01 report got a same-day follow-through blockquote (§c7 CHANGELOG closed by this session; §f items routed).
8. **ARCHIVE pass.** 22 annotated reports `git mv`'d to `docs/status/archived/` (413 → 435); the round-6 stub anomaly (2-line duplicate pointer of the archived twin) `git rm`'ed; 3 executed plans archived (`2026-09-27_22-35` wave plan — already annotated EXECUTED, the round-12 plan HTML, the 09-23 dashboardui adoption plan). Tail is now exactly 1 report + README.
9. **72 pre-existing PARTIAL table rows normalized** (8 files): the 09-22/23 annotators had struck item-cells only, leaving Effort/Cat/Route cells unstruck — invisible while those files lived outside `archived/` (the row gate scans the archive dir only), surfaced by my moves, fixed by a mechanical whole-row-strike normalizer.
10. **11 broken relative links fixed** — all created by my own archive moves (`../../TODO_LIST.md` classes and two superseded-by links); `check-docs-links` 315 OK after.
11. **docs/status/README.md refreshed:** tail count 1, archive 435 (2026-05-03 → 2026-10-01), sweep list + 70 gated-convention count, stub-anomaly note.
12. **Verification battery (docs surface):** preflight-tree-check OK at start; check-status-annotations 70/70 gated reports annotated (365 legacy-exempt); check-status-rows 0 PARTIAL / 435 files (30 deliberately-mixed tables, first-class); check-docs-freshness PASS; check-docs-links 315 OK; **`nix run .#check-modules` 17/17 stages green** — including the strict release-train stage (0 lag / 0 unpublished) and both CSS class-set gates.
13. **This report.**

## b) PARTIALLY DONE

1. **The verification battery's heavy half.** `nix run .#test` (workspace race suite), `.#coverage-gate`, and the e2e Playwright suite (proves the 10-01 snapshot-sentinel change end-to-end) were NOT run — load 51–194 all session; per the repo's own doctrine (identical failures under load > 20 are suspect-contended), running them would have produced noise, not evidence. Tracked as TODO_LIST P2 with the honest header stamp. `check-modules` (the composite) DID run green, which retires its slice of the debt.
2. **§f mega-table adjudication depth.** The six unharvested 2026-09-22/23 reports carry ~250-row §f brainstorm tables. I struck rows with crisp receipts and adjudicated the remainder at BLOCKQUOTE level ("unadopted brainstorm superseded by the round-12 TODO rewrite") rather than per-row — matching the repo's archived-twin precedent, but short of the skill's per-item ideal. The genuinely-open rows I could name are routed in TODO_LIST P2/P3.
3. ~~**The six 2026-08-30 planning .md files** (buildcache-hardware-decision, cqrs-lint-go-distribution-draft, gated-work-index, setup-demo-blob-purge-plan, upstream-issue-drafts, v4-branch-purge-plan) — in scope by the "2026-0\*" glob, not triaged (annotate/archive/leave undecided). They predate the annotation epoch but are active-dir clutter.~~ resolved (docs-health round 14, 2026-10-01): upstream-issue-drafts ARCHIVED (asks filed 09-10, purpose served); gated-work-index top-up annotated (origin/v4 re-verified live at `339ce82b`); cqrs-lint-go-distribution-draft annotated (the linter source now lives in go-cqrs-lite `cmd/cqrs-lint` — what remains gated is only the Go-installable artifact); buildcache/setup-demo-purge/v4-branch-purge confirmed LIVE owner-gated decisions, leave-in-place.
4. ~~**agents-notes long-form narratives.** The 09-30/10-01 arc narratives (cqrs-lint three-pass story; the three-tools-vs-nixpkgs-Go story) are indexed as TODO_LIST P3 items, not written — I did not witness those sessions and refused to fabricate history prose beyond what the reports state.~~ RESOLVED-BY-ROUTING (docs-health pass 2026-10-01): the distilled layer (AGENTS gotchas 8/13/22) already carries the durable content; the narratives stay a TODO item, which is the honest state.
5. **Formatting provenance.** `nix fmt` reported "formatted 0 files" over my hand-edits (treefmt-clean), and two multiedit operations applied with "whitespace-equivalent re-indentation" warnings I did not individually re-inspect — covered by the final gate battery, not by per-hunk review.
6. **Commit attribution.** Per the never-commit rule I left every phase to the auto-commit daemon; several phases landed as heuristic `chore: auto-commit` commits (the daemon was faster than my phase boundaries in two windows — the exact gotcha-4 loss the repo documents). Content is gate-verified; history readability degraded as usual. ~6 files were still dirty at handoff (final TODO/10-01-addendum edits).
7. ~~**The HTML status corpus** (12 files in docs/status/) — exempt per policy, untouched; no policy decision requested or made.~~ resolved (docs-health round 14, 2026-10-01): keep-in-place CONFIRMED as the standing decision — they are generated deliverable artifacts superseded by the living docs, the README policy already exempts them, and archiving would be churn without information gain; decision recorded in `docs/status/README.md`.

## c) NOT STARTED (deliberately — all routed, none actionable this session)

1. ~~**The family train itself** — the usermgmt re-tag + root publish (TODO P1). This session wrote the item; cutting tags is a separate, verify-tag-gated action.~~ done (W0–W3 §a: 9 tags published 2026-10-01, wave-ordered, strict gates green at every push, proxy smoke green)
2. **erraudit triage** — zero of the 57 findings addressed (TODO P2; published-module constraint → rides the next train).
3. ~~**`check-workspace-build` gate** — not implemented (TODO P2; full atomic-gate checklist applies).~~ done at `c85d802f` (T07: checker + 4-case fixture + flake apps + check-modules stages + CI steps + AGENTS quick-ref row)
4. **The 4 fleet upstream asks** — none filed (TODO P2).
5. **go-cqrs-lite cross-repo docs/gates debt** — untouched, owner-gated (TODO P3).
6. **System cqrs-lint binary rebuild** — owner-side; re-verified still pre-fix (`3756eb4/20260929`) via `cqrs-lint version`.
7. **`nix run .#test` / coverage-gate / e2e / bench-spike** — see b1.
8. **All code work** — none; docs-only session by design.

## d) TOTALLY FUCKED UP (honest ledger — nothing shipped broken, all gates green at handoff)

1. **Edit-anchor guessing.** 6–7 multiedit operations failed on title/section anchors I inferred instead of viewed first (three different report titles were guessed wrong; one §f list assumed table format when it was plain numbered). Every failure cost a full roundtrip and was recovered by viewing — the exact read-before-edit discipline the global config mandates, violated reflexively under time pressure.
2. **The first CHANGELOG multiedit was structurally muddled.** My new_string designed a stray second `### Changed (2026-09-27/30 …)` header while the sibling edit targeted the real one — had BOTH applied, the file would have carried a duplicate Changed section (the round-12 report's §d1 incident class). It "worked" only because the flawed edit failed and the clean one applied. Luck, not process. Lesson: design multiedit batches as ONE coherent insertion plan, then verify the section skeleton (`grep -n '^### '`) immediately after.
3. **Archive-before-gate-scope-check.** I moved 22 files into `archived/` before checking that the row gate scans the ARCHIVE dir only — surfacing 72 pre-existing PARTIAL rows that had been invisible (and gate-green) in place. The moves were correct and the fix mechanical, but a 30-second scope read of `check-status-rows.py` (DEFAULT_DIR) would have predicted it. Same class: the annotation gate's MISSING-STRIKETHROUGH requirement for the moved 11-46 file — its 09-29 annotation used sub-blockquotes, not `~~` markers, and only failed AFTER archiving.
4. **Heavy gate under load.** I launched `check-modules` at load ~50–85 on a 69-user box. It passed (RC=0), so no damage — but the repo's own runbook says treat contention-window results as suspect; I spent the risk to save wall-clock and got away with it. The test/coverage/e2e deferral was the correct half of the same judgment; the check-modules launch was the inconsistent half.
5. **Scratch tooling left in /tmp.** The partial-row normalizer (`/tmp/normalize_rows.py`) mechanized a real, recurring class (the 2026-08-05 precedent fixed its two PARTIAL rows BY HAND) and is now throwaway — the next partial-row batch starts from zero. In a repo whose culture is "new gates ship atomically: checker + fixture + flake app + stage + CI," leaving the checker as a /tmp file is the half-done version.
6. **Inconsistent early verification.** `cqrs-lint version` (which discovered the still-pre-fix system binary) ran mid-session; it belonged in the first verification batch alongside load and tag checks — it materially shaped a TODO item's wording.
7. **Not actually fucked up (verified non-events):** the two whitespace-equivalent edit warnings (final treefmt + gate battery clean), the `git rm` of the stub (tracked file, fully recoverable from history), and the 09-29_11-46 top-up blockquote initially failing Gate 1 (fixed same-session with two real strikes).

## e) WHAT WE SHOULD IMPROVE

1. **Repo-own the partial-row normalizer.** `scripts/normalize-status-rows.py` + fixture self-test + flake app + check-modules stage + CI step (the full checklist). The class it fixes has now occurred twice by hand (2026-08-05) and once at scale (this session, 72 rows); the checker-side gate exists, the fixer-side doesn't.
2. **Run the status gates BEFORE `git mv` archive batches.** A pre-move `check-status-annotations.sh` + `check-status-rows.py` pass over the staged file list would surface MISSING-STRIKETHROUGH and PARTIAL-row surprises while the files are still cheap to edit — the archive move should be the last step, not the discovery step.
3. **Read the checker's scope lines before assuming them.** Both gate surprises this session came from scope facts (DEFAULT_DIR; strikethrough-presence requirement) that were one grep away.
4. **Use the skill's mechanical annotator for bulk tables.** `annotate-status-items.py --emit-keys` + specfiles are atomic, self-verifying, and refuse ambiguity — the right tool for §f tables; my hand multiedits were slower and produced the whitespace-equivalent warnings.
5. **Multiedit batches get one coherent insertion plan + a post-append skeleton check** (`grep -n '^### '` after CHANGELOG-scale edits — the round-12 lesson, nearly repeated in d2).
6. **Verification ordering discipline:** tool-state checks (`cqrs-lint version`, `which buildflow`, load, `git tag`) belong in the FIRST batch — they're cheap and they shape everything downstream.
7. **The honest-header pattern worked and should stay canonical:** "Coverage: … (NOT re-run after X; re-run queued in P2)" is the exact shape the round-12 self-review demanded; gates cannot catch a stale header, only discipline can.
8. **Tail-budget rule for docs-health:** the repo convention says "most recent 1–3" but the tail had drifted to 22. A tail-budget gate candidate (docs/status/*.md count > N → warn) would keep the next sweep small; candidate for the micro-batch.

## f) Up to 50 things to get done next (session-derived; impact-sorted; most already tracked in TODO_LIST — listed here as the harvest receipt)

**Release / P1:**
1. ~~Cut the family train, wave-ordered: usermgmt re-tag (codec migration — PapDashboard's adoption prerequisite; setup v4.13.x already live), then root + consumers for the Unreleased set.~~ done (W0–W3 §a: 9 tags — usermgmt v4.13.0/v4.13.1, root v4.13.0, setup v4.13.2, five consumer rides v4.12.1)
2. ~~Post-re-tag absence sweep: `rg 'codec/v4' -g go.mod` expecting zero.~~ done (W1 consumer sweeps: all codec/v4 indirects dropped, absence sweep zero)
3. ~~Push the backlog + watch CI to green on the new ranges.~~ done (CI fully green — run 36839790817, first green master after the lint-drift rounds `b66b26f0`/`a56241bf` fixed)
**erraudit program (P2):**
4. ~~Fix the trivial pair first: `examples/samber-do-demo/container.go:61` panic; `e2e/playwright.config.ts:20` bug-marker comment.~~ done at `3d9ce553` (T05; scoped erraudit re-runs 1→0 both)
5. Verify the `//nolint:<analyzer>` name erraudit honors; document in AGENTS once known.
6. The 55-site `context_loss` program, per-finding (go-error-modernization flow); bundle with the P1 train.
7. Decide §g1 of the 10-01 report: gate stays honestly red vs temporary documented demotion while the program runs.
8. Decide the intended error-context policy for read-model decode errors (§g2) — fixes the program's shape (fix sweep vs suppression sweep).
**Gates/tooling (P2):**
9. ~~`check-workspace-build` gate (plain workspace `go build ./...`) — full atomic checklist.~~ done at `c85d802f`/`eed53317` (T07)
10. ~~`nix run .#test` post go.work/dep changes (quiet window).~~ done (T03: rc=0 ×28 modules, load 5.9)
11. ~~`nix run .#coverage-gate` post the 09-30 sweeps + 10-01 changes.~~ done (T03: 15/15 PASSED post-train)
12. e2e Playwright run (proves the snapshot sentinel change).
13. bench-spike quiet-window look after the 09-30 dep bumps.
14. File the 4 fleet upstream asks: treefmt-nix templ Go-pin; nixpkgs go-licenses GOROOT; golangci-lint TMPDIR lock opt-out; BuildFlow go-work-sync union-graph guard (373209a7 case study).
15. File the a-h/templ parser issue (silent literal-text render of text-position component calls; round-12 leftover, still unfiled).
16. Rebuild the system cqrs-lint binary → verify `cqrs-lint --strict --verbose .` shows zero C040 on a root walk.
17. Repo-own the partial-row normalizer (checker + fixture + flake app + stage + CI) — this session's e1.
18. Tail-budget gate candidate (docs/status/*.md count warn) — this session's e8.
19. bump-dep.sh `--commit` mode + per-module `go mod verify`.
20. Wire `--fail-on-stale-suppressions` into the flake gate.
21. Adopt a `cqrs-lint rules` old-vs-new diff ritual for binary updates.
22. Audit `run.go:` values across all 16 `.golangci.yml` (1.26.7 vs the 1.27.1 floor).
23. Raise dependabot's 20-module cap (28 modules today).
24. Fix lychee's real 404s (golangci-lint install URL, go-datastar/broadcast path, DiscordSync/overview/sec links) + deepwiki 429 backoff.
25. vulnix 65-advisory policy (ignore-file vs nixpkgs bumps).
26. Resolve the "9 tools unavailable" health-check class (devShell adds or accept+document).
27. jscpd `.golangci.yml` duplication findings policy (fleet-managed configs).
28. go-auto-upgrade noise policy (~500 suggestion findings).
29. Feedback-inbox checker (`scripts/check-feedback-inbox.sh`, full checklist if built).
30. Scoped-format flake app (`.#fmt <paths>`).
31. e2e pin: ExtraMiddleware composes under RunWithAppkit.
32. Per-module race tests for the 8 e2e/examples modules (or a `#test-all` app; ROADMAP decision).
**Cross-repo / docs (P3):**
33. go-cqrs-lite: CHANGELOG/AGENTS/TODO entries for the linter fix (owner-gated).
34. go-cqrs-lite: `cmd/cqrs-lint` vet + golangci-lint + `-race`.
35. go-cqrs-lite: RULES.md C040 metadata-coverage touch.
36. cqrs-lint residual triage: E005×16 (candidate linter fail-open fix), V007×41 (ADR-0051 posture), A016+V006×4 (suppress-with-reason), ~130 further untriaged.
37. agents-notes narratives: the cqrs-lint three-pass arc; the three-tools-vs-nixpkgs-Go story.
38. AGENTS gotcha 4 note: formatter-clean ≠ lint-clean (pre-commit bar is `golangci-lint run`).
39. Runbook note: never chain bump-dep invocations without committing between.
40. ~~Triage the six 2026-08-30 planning .md files (annotate/archive/skip).~~ done (docs-health round 14, 2026-10-01: 1 archived, 2 annotated, 3 confirmed live gated decisions — see §b3)
41. Decide the standing CHANGELOG convention for future dep-only sweeps (this pass set precedent by backfilling; the standing rule is still the owner's — 09-30 g1).
42. ~~HTML status corpus: confirm the keep-in-place policy or archive the 12 files deliberately.~~ done (round 14: keep-in-place CONFIRMED, recorded in `docs/status/README.md`)
**Owner calls / parked (P3):**
43. OQ26: fleet go-directive policy decision.
44. OQ23/24 go/no-go (the PapDashboard architectural two).
45. Answer PapDashboard: codec migration shipped; where do #67/#68 live?
46. datastar-demo KEEP-as-is confirmation; loginpage OQ21 posture.
47. /mnt/buildcache reclaim decision (rust/ 155G + sccache/ 20G).
48. Dashboardui snapshot-detail "store errored vs store empty" distinct state (10-01 §f32).
49. Document the Load contract (nil,nil vs ErrSnapshotNotFound) upstream in go-cqrs-lite store docs.
50. ~~Verify the daemon swept this session's final ~6 dirty files + spot-check CI on those ranges.~~ done (tree clean at `d397d429`; CI fully green — run 36839790817)

## g) Questions I can NOT figure out myself (max 3)

1. **Archive bar:** I archived all 22 reports on the "dated blockquote + evidence strikes + open items routed" bar (repo precedent — archived twins legitimately carry unstruck §f bodies and routed residuals), keeping only the 10-01 report as the live tail. Do you endorse that as the standing bar, or do you want the stricter "every numbered item individually struck or explicitly Won't-implement" bar before archiving (which would have kept ~8 of the 22 in the tail and roughly doubled the session)?
2. **§f mega-table depth:** for the ~250-row brainstorm tables in the 09-22/23 reports I adjudicated at blockquote level ("unadopted brainstorm, superseded by the round-12 TODO rewrite") with selective strikes. Acceptable as the permanent precedent, or do you want a future mechanical per-row pass (annotate-status-items.py specs) for those six files?
3. **Gate policy during the erraudit program** (the 10-01 report's g1, still unanswered): keep `fail_on: critical` honestly red for every BuildFlow run until the 57 criticals are triaged, or a temporary documented demotion in `.buildflow.yml` so runs exit 0 while the program is open?

---

_Point-in-time snapshot — 2026-10-01 06:36 CEST. Living state lives in `TODO_LIST.md` / `FEATURES.md` / `CHANGELOG.md` / `ROADMAP.md`. Per the status convention, later sessions should ANNOTATE, never rewrite, this file. WAITING FOR INSTRUCTIONS._
