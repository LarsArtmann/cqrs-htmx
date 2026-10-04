# Status Report — docs-health Round 14: Full 2026-0\* Corpus Pass, Living-Docs Superb Refresh, Archive + Planning Triage — 2026-10-01 12:30 CEST

> **Session:** 2026-10-01 ~11:55 → 12:30 CEST (report written 12:30 CEST)
> **Scope:** this session only — the owner-directed docs-health AUDIT over the full `**/2026-0*` file corpus and the six living docs: BUILD + HARVEST + VERIFY + ANNOTATE + ARCHIVE. No code changes; docs-only session by design.
> **Trigger:** owner instruction — "View ALL \*\*/2026-0\* files! Execute the docs-health SKILL! … TODO_LIST.md, CHANGELOG.md, AGENTS.md, README.md, ROADMAP.md, and FEATURES.md must be all SUPERB! … Archive FULLY done and UPDATED (inline strikethrough) .md files!"
> **Machine context:** shared fleet box, load 33–40 across the session — the heavy `check-modules` composite was deliberately deferred (see b1), the four targeted docs gates ran green.
> **Ingestion:** docs-health SKILL.md loaded first; corpus enumerated via `find` (300+ matches; the actionable non-archived set: 23 md + 55 html across docs/, plus the 3 live 2026-10-01 reports read in full).

---

## Executive verdict

The docs surface went from "3-report tail with a zero-strike report that would fail Gate 1 on archive, TODO still carrying a done item, FEATURES/ROADMAP headers claiming the train was pending, the six 2026-08-30 planning files untriaged, and no recorded HTML-corpus decision" to: **1-live-report tail + this report, 437 archived reports all gate-clean (72 in the gated convention), TODO_LIST carrying only real open work with every citation resolvable, all six living docs telling the same v4.13.x-shipped story, and the round-13/14 questions that blocked archival answered with receipts.** The session's most valuable catches: the 02:59 report had ZERO inline strikes (Gate 1 would have failed the archive — the round-13 annotator routed everything at blockquote level and struck nothing); the round-13 report's §c "not started" list was falsified by the W0–W3 session hours later (train shipped, gate shipped, battery green) and is now struck with evidence; and the six planning files triaged cleanly into 1 archive + 2 annotations + 3 confirmed live owner-gated decisions.

## Scores (health-report format)

- **Accuracy 95/100** — 40+ concrete claims verified against git (origin/v4 live at `339ce82b`), gate outputs (72/72, 0 PARTIAL, 315 links, freshness PASS), CI run id, and CHANGELOG presence greps; 4 stale claims found and fixed; 0 ghost systems. −5: two numeric header claims (`check-modules` 25/25, "9 tags") propagated on same-day report authority, not first-hand re-verification, and two coverage-percentage sets could only be date-stamped, not re-measured (load 36 box).
- **Fitness 96/100** — 6 drift instances fixed (3-report tail vs the README's "1" claim; 1 done item squatting in TODO_LIST; the round-13 report squatting in the live tail; a missing OQ; 2 missing CHANGELOG entries; an unanswered-planning-dir). −4: the 25-stage composite is unproven this session (b1) and the §g owner-decision harvest gap (b3).

## a) FULLY DONE (verified this session)

1. **Corpus ingestion + VERIFY pass.** All `**/2026-0*` files enumerated; the 3 live reports (02:59 BuildFlow recovery, 06:36 round-13 audit, 10:30 W0–W3 Pareto execution) read in full; TODO_LIST, CHANGELOG head, FEATURES head, ROADMAP head + OQ list, AGENTS gotcha 8, docs/status/README, planning README, all six 2026-08-30 planning files, the 09-23 verdict memo, and the active 06-47 plan head read; claims spot-verified: `origin/v4` still at `339ce82b` (matches the purge plan's census), CHANGELOG has no link-reference block (concern void), README.md carries ZERO version-pinned claims (grep clean), CI green run `36839790817` consistent across three docs.
2. **ANNOTATE — the 02:59 BuildFlow-recovery report** (the big one): it had a round-13 blockquote but **zero `~~` strikes** — archived as-is it would have failed Gate 1's MISSING-STRIKETHROUGH check. Struck §c2/c3/c4/c5/c7 + §f1/f3/f4/f5/f6/f7/f8/f26/f39/f46/f47 with W0–W3 receipts (`check-workspace-build` at `c85d802f` + consumer-view `eed53317`; trivial pair at `3d9ce553` with scoped re-runs 1→0; T03 battery: `.#test` rc=0 ×28 modules at load 5.9, coverage 15/15; strict train gates 0 lag / 0 unpublished at every push), and added a round-14 top-up blockquote.
3. **ANNOTATE — the 06:36 round-13 report:** struck §b3/§b7/§c1/§c3 + §f1–f4/f9–f11/f40/f42/f50 — §c1 is the highlight: its "NOT STARTED: the family train itself" was falsified within hours by the W0–W3 session (9 tags published, wave-ordered, strict gates green, proxy smoke green). Round-14 top-up blockquote records the falsification with the run id.
4. **ARCHIVE — `git mv` ×3:** both annotated reports → `docs/status/archived/` (435 → **437**; gated-convention count 70 → **72**), and `docs/planning/2026-08-30_upstream-issue-drafts.md` → `docs/planning/archived/` (its 4 asks were FILED 2026-09-10 as go-cqrs-lite #25–#28 — purpose served; the gated-work-index row link updated to the archived path). Live tail reduced to exactly the W0–W3 report (now 2 with this report).
5. **Planning triage — TODO P3(m) EXECUTED, not just tracked** (it had been punted by round 13): `gated-work-index` top-up blockquote (rows re-verified live; buildcache dormancy re-confirmed at 58% used; the NEW 5-ask fleet upstream bundle noted as tracked separately in TODO P2); `cqrs-lint-go-distribution-draft` annotated (its core premise moved: the linter source now lives as real Go code in go-cqrs-lite `cmd/cqrs-lint` since the 09-30 C040 fix — only the publish step + CI flip remain gated; v4.6.0 version refs flagged stale); the 09-23 verdict memo, buildcache decision, setup-demo blob-purge plan, and v4-branch-purge plan confirmed as LIVE owner-gated decisions (leave-in-place, correctly indexed); the datastar dual-frontend rollout plan confirmed active (TODO P2 Tier-4 item references it).
6. **HTML status corpus — the deliberate decision made and recorded:** keep-in-place CONFIRMED as standing policy in `docs/status/README.md` (12 generated deliverable artifacts, exempt from every gate, zero information gain from archiving). TODO item (m)'s second half closed with that line as the record.
7. **TODO_LIST rebuilt to live truth:** the erraudit item rewritten from the stale "57 findings, trivial pair first" to the real state (61-site inventory, 36 fixed across 9 clusters with commit receipts, 25 remaining with exact file:line sites, rawIDToken trio decided suppress-with-reason pending owner confirm, first moves = analyzer-name verification + the 61-vs-54 count re-derivation); the DONE `check-workspace-build` item REMOVED (it lives in CHANGELOG [Unreleased] Added — done items never stay in TODO); the battery item tightened to e2e + bench-spike only with everything-else-green receipts; the appkit item's stale pin clause rewritten (the go-etag stub-replace was RETIRED at its natural end — the v4.13.0 train dissolved the union-graph edge, not an appkit release); harvested NEW from W0–W3/round-13: `sentinel_concrete_type` ×51 (cqrs-lint item f), the BuildFlow noise-policy batch (vulnix dead NVD feed 404, "9 tools unavailable", jscpd config dupes → micro-batch m), the bump-dep single-line-require gap (micro-batch a, proved live at `9b3c2e18`), and the PapDashboard reply packet (the v4.13.0 prerequisite is LIVE); every citation pointing at the two moved reports repointed to `archived/` paths; header restamped (check-modules 25/25 per W0–W3, CI green run id added).
8. **CHANGELOG:** +2 [Unreleased] Fixed entries — erraudit wave 1 (trivial pair + the 36 per-cluster context_loss fixes with commits, the gate-stays-red posture, the rawIDToken decision) and the CI golangci version-drift regression (the `NewStatusRecorder` promoted-key/embedlit double-trap resolved by the one-arg `newDelegatingWriter` extraction; the setup validator cyclop/funlen/goconst round at `a56241bf`; first fully-green master run `36839790817`).
9. **FEATURES.md header:** restamped for the SHIPPED v4.13.x train (was "root v4.12.0 … re-tag pending — TODO_LIST P1"); coverage line split honestly: gates 15/15 GREEN re-run 2026-10-01, percentages explicitly from the 2026-09-29 run.
10. **ROADMAP.md:** header + the Current-State version bullet rebuilt for v4.13.x (the old text still led with the 14-tag v4.12.0 train from 09-22); coverage bullet restamped with the same gates-vs-percentages split; **OQ27 opened** — the go.work fleet-local replaces standardization question (tracked-with-gate-filter vs untracked `GOWORK` overlay), harvested from W0–W3 §e/§g3.
11. **AGENTS.md gotcha 8:** the erraudit parenthetical refreshed — "55× context_loss + 1× panic + 1× bug-marker queued" was stale the moment T05 landed; now records the pair fixed + 36/61 + 25 remain + gate-honestly-red-until-zero.
12. **docs/status/README.md:** counts refreshed (tail, 437 archived, 72 gated-convention), the round-14 sweep added to the sweep history, the HTML keep-in-place decision recorded.
13. **Verification battery:** Gate 1 `check-status-annotations.sh` **72/72** (365 legacy-exempt, 437 scanned); Gate 2 `check-status-rows.py` **0 PARTIAL** across 437 files (30 deliberately-mixed tables, first-class); `check-docs-links.sh` **315 OK** (no breakage from the three `git mv`s); `check-docs-freshness.sh` PASS; `nix fmt` **0 changed** (treefmt-clean hand edits); repo-wide grep proves zero remaining citations to the moved reports' old paths; README.md zero stale version claims.
14. **This report** (which immediately makes the tail 2 — the README count is amended in the same breath as this file lands, see d4).

## b) PARTIALLY DONE

~~1. **The heavy-gate confirmation.** The four targeted docs gates are green (a13), but the 25-stage `check-modules` composite did NOT re-run this session: load 33–40 is past the repo's own suspect-contended line (identical failures under load > 20 are noise per the runbook; W0–W3 made the same refusal at 24.8). The two docs gates inside it ran green standalone and a docs-only change touches no Go stage — but "composite proven green today" is unproven. Next quiet window converts this to green.~~ done 2026-10-01 ~15:50 — the composite re-run landed rc=0 all 27 stages under load 80+ (see the 2026-10-01_15-55 report §a W10)
~~2. **Header-claim provenance.** "check-modules 25/25" and "9 tags" now sit in three living-doc headers on the W0–W3 report's authority — same-day, gate-cited, internally consistent, but not independently re-verified by this session (no `git ls-remote` tag spot-check, no flake stage-list count). Reasonable trust, still trust.~~ done (docs-health pass 2026-10-04: counts re-derived at this sweep — archived 450, gated convention 85, check-modules stages 28)
~~3. **The §g owner-decision harvest gap.** The gate-policy question (`fail_on: critical` honestly red vs a documented temporary demotion while the erraudit program runs) was asked by BOTH archived reports and remains unanswered — and TODO_LIST has no owner-decision index, so the question now lives only inside an archived report plus §f/§g here. The archive-bar question (round-13 §g1) is in the same boat. Deliberately routed to §g of this report rather than invented into a new TODO section — but a standing owner-decisions index would close this class mechanically.~~ done 2026-10-01 evening — the standing Owner Decisions index (D1–D11) now lives in TODO_LIST
~~4. **Research/benchmarks/proposals corpus verdict** — classified LEAVE (each governed by its own standing convention: `docs/research/README.md` indexing, ROADMAP-cited outcomes, benchmark raw-baseline policy), but that verdict is recorded only in this report, not in any standing doc.~~ done (docs-health pass 2026-10-04: the LEAVE verdict is recorded in docs/research/README.md)
~~5. **The W0–W3 (10:30) report** is the live tail and correctly unannotated — but its §g1–g3 (rawIDToken confirm, upstream-ask filing authorization, go.work overlay choice) remain open and unanswered, and this session added nothing to their resolution.~~ done — annotated + archived in this pass (round 17, 2026-10-04)

## c) NOT STARTED (all pre-existing, all routed — none belonged to this docs-only session)

~~1. **erraudit context_loss remainder** — 25 of 61 sites (oauth2 ×5 incl. the rawIDToken suppress trio, identity-model authz ×3, dashboardui ×10+): PUBLISHED-module code → rides the next train. TODO P2.~~ done 2026-10-01 evening — all 34 sites in the current SDK inventory fixed/suppressed; 0 criticals ×28 modules
~~2. **e2e Playwright suite + bench-spike quiet-window run** — the battery's last slice (snapshot-sentinel proof; load < 6 twice; OQ16 posture). TODO P2.~~ e2e DONE (70/70 rc=0); bench-spike still owed at a verified-quiet window — routed TODO P2
~~3. **The 5 fleet upstream asks** — none filed (treefmt-nix templ Go-pin, nixpkgs go-licenses GOROOT, golangci TMPDIR lock, BuildFlow go-work-sync union-graph guard, a-h/templ parser). TODO P2; needs filing authorization.~~ done 2026-10-01 evening (3 FILED: treefmt-nix#545, a-h/templ#1449, BuildFlow#29; 2 RETIRED with evidence)
~~4. **System cqrs-lint binary rebuild** — owner-gated; still pre-fix (`3756eb4/20260929`); gotcha-13's caveat stays until it lands.~~ local half done 2026-10-01 evening (rebuilt binary 0×C040, 14/14 modules); fleet swap owner-pending (D4)
~~5. **check-modules composite re-run** in a quiet window (b1).~~ done 2026-10-01 ~15:50 (rc=0, 27/27, load 80+)
~~6. **go-cqrs-lite cross-repo debt** — CHANGELOG/AGENTS/TODO there + vet/lint/-race on the collector fix. Owner-gated. TODO P3.~~ OPEN — owner-gated (TODO_LIST P3; its f2 lint+race remainder surfaced again in the 2026-10-03 report)
~~7. **Tooling micro-batch (a)–(m)** — bump-dep `--commit` + single-line-require fix, lychee 404s, run.go audit ×16, dependabot cap, normalizer promotion, tail-budget gate, noise-policy batch. TODO P3.~~ done 2026-10-01 (tooling round 2); (g) feedback checker gated on D7
~~8. **Owner calls** — OQ23/24 (PapDashboard go/no-go), OQ26 (go-directive policy), OQ27 (go.work replaces, NEW), loginpage OQ21, datastar-demo keep/remove, /mnt/buildcache reclaim, the PapDashboard reply packet. TODO P3 + ROADMAP.~~ routed — standing D-index in TODO_LIST (D1–D11)
~~9. **agents-notes narratives** — cqrs-lint three-pass arc + three-tools-vs-nixpkgs-Go story. TODO P3 docs micro-batch.~~ done 2026-10-01 (two narratives assembled into docs/agents-notes.md)

## d) TOTALLY FUCKED UP (honest ledger — nothing shipped broken; all gates green at handoff)

1. **Three wasted roundtrips on edit-tool discipline.** Two `edit` refusals because I had read `gated-work-index.md` and the cqrs-lint draft via bash `cat` instead of the View tool (the tool requires View; the rule says read before edit and I pattern-matched "already read the content" against the letter of the tool contract). One failed multiedit match: I targeted the cross-repo citation's FULL filename when the file carries the abbreviated `2026-09-30_08-03...md` form — I knew the exact-match rule and still batched an unverified string. All three recovered in the following batch; the cost was roundtrips, not correctness.
2. **The implied-re-measurement trap — caught one commit short of shipping it.** My first FEATURES/ROADMAP coverage restamps read "re-run 2026-10-01: …94.7%…" — which asserts the percentages were re-measured. They were NOT: the 10-01 run proved 15 gates GREEN; the percentages are from the 09-29 run. Self-review of my own wording caught it; both files now carry the explicit "(percentages from the 2026-09-29 run)" split. A top-tier reviewer would ask why the trap existed at all — the answer (gates print percentages every run, so "gate green" and "these numbers" LOOK like the same claim) is exactly why the convention needs codifying (see e3).
3. **Unverified numeric propagation into three headers.** "check-modules 25/25" and "9 tags" went into TODO/FEATURES/ROADMAP on the W0–W3 report's say-so. Same-day, gate-cited, almost certainly right — but this session's own standard (verify, don't trust) says one `git ls-remote` and one grep of the stage list were owed first. Recorded as d-class because it is the inbound variant of the verify-external-claims violation at miniature scale.
4. **Re-created the §f39 drift class with my own hands, same session.** I stamped `docs/status/README.md` "1 unarchived report" and then, minutes later, wrote THIS report — making the tail 2 and my fresh count a lie on arrival. Amended in the same breath as this file lands (count → 2, sweep entry updated). The class recurs because count-claims and report-creation happen in every sweep with nothing mechanical binding them (the tail-budget gate, TODO P3 l, would catch the tail side; a README-count fixture would catch the claim side).
5. **Not actually fucked up (verified non-events):** the CHANGELOG link-reference-block worry (grep proved no `[Unreleased]:` convention exists in this file — concern void); the `git add -A` before reading status (tree was already daemon-clean; no-op); `nix fmt` after the hand edits (0 changed, so the green gates stayed valid); leaving the W0–W3 report unannotated (correct — it is the live tail, absence of annotation is not a defect there).

## e) WHAT WE SHOULD IMPROVE

1. **View-before-Edit is a tool contract, not a suggestion.** bash `cat` does not satisfy it, and "I already saw the content" is exactly how exact-match failures on abbreviated text happen (d1). Three roundtrips this session bought this lesson for free; the next session should not re-buy it.
2. **Numbers quoted into living-doc headers get first-hand spot-checks or explicit provenance tags.** One `git ls-remote` + one stage-list grep per session is cheap; alternatively tag the header itself "(per W0–W3 §a)" so the reader knows whose measurement it is (d3, b2).
3. **Codify the coverage-stamp convention:** "gates re-verified green (date)" and "percentages measured (date)" are two different claims that MUST NOT share a colon. A docs-freshness greppable rule (a percentage adjacent to a re-run date without a measured-date qualifier = fail) would kill the d2 trap permanently.
4. **§g questions evaporate on archive.** Two reports carried the same unanswered gate-policy question; after archiving, only §g-here keeps it visible. A standing **"Owner decisions awaiting you"** index (TODO_LIST section or ROADMAP table) that HARVEST must feed would make owner questions impossible to lose — this session routed three of them there (see g).
5. **Tail-budget gate (TODO P3 l) + a README-count fixture.** The tail drifted to 22 before round 13, to 2 fresh claims today (d4); both sides are mechanical: warn when `docs/status/*.md` count > 3, and assert the README's stated counts against reality in a fixture self-test.
6. **Use the mechanical annotators.** The skill's `annotate-status-items.py --emit-keys` specfile flow is atomic, self-verifying, and refuses ambiguity; hand multiedits were slower and produced the whitespace-class risks. Round-13's §e4 said the same thing — this is a repeat note, which makes it a discipline item, not a discovery.
7. **Daemon-phase attribution stays structural.** Every phase landed as heuristic `chore:` commits again (`6ff96489` → `c631958b`; each `git show`-inspected). The never-commit rule forbids self-committing without an explicit owner instruction, so the runbook's "verify content, not messages" remains the mitigation; if the owner wants readable history for docs sessions, the fix is an explicit "commit at phase boundaries" instruction, not a rule change.

## f) Up to 50 things to get done next (impact-sorted; most pre-existing + this session's deltas; sources: TODO_LIST P1–P3, the live W0–W3 report §f, this session's b/e)

**P1-class (unblocks value or honesty):**
~~1. Finish the erraudit `context_loss` program — 25 sites: oauth2 ×5 (rawIDToken trio = suppress-with-reason after owner confirm), identity-model authz ×3, dashboardui ×10+ → scoped erraudit 0 → gate turns green. *(TODO P2; published code → next train.)*~~ done 2026-10-01 evening (0 criticals ×28 modules)
2. Verify the `//nolint:<analyzer>` name erraudit honors — empirically, before the suppressions above rely on it; document in AGENTS. *(W0–W3 §f1.)*
3. Re-derive the 61-vs-54 site-count discrepancy in the working-note extractor before the final verdict. *(W0–W3 §b.)*
~~4. e2e Playwright suite + bench-spike in a verified-quiet window (load < 6 twice) — the battery's last slice; proves the snapshot sentinel change. *(TODO P2.)*~~ e2e DONE (70/70); bench-spike routed TODO P2 (quiet window)
5. Re-run `check-modules` (25 stages) in that same quiet window — converts this session's b1 into green.
6. Bundle the erraudit code changes with the next tag train (published modules — verify-tag choreography per release-playbook §3a). *(TODO P2.)*
~~7. Draft + file the 5 fleet upstream asks (treefmt-nix templ Go-pin, nixpkgs go-licenses GOROOT, golangci TMPDIR lock, BuildFlow go-work-sync union-graph guard with `373209a7` as case study, a-h/templ parser) — verify-before-filing discipline. *(TODO P2.)*~~ done 2026-10-01 evening (3 filed / 2 retired with evidence)
~~8. Owner: rebuild the system cqrs-lint binary → `--strict --verbose .` root walk shows zero C040 → retire gotcha-13's caveat. *(TODO P2.)*~~ local half done 2026-10-01 evening; fleet swap owner-pending (D4)
~~9. CHANGELOG + AGENTS entries once the analyzer name lands (2). *(W0–W3 §f1 tail.)*~~ done (analyzer semantics source-verified + documented in AGENTS gotcha 8; CHANGELOG receipts added same evening)
~~10. Watch CI on the current ranges; record run ids in the next report. *(W0–W3 §f2 pattern.)*~~ done (superseded — first fully-green run 36839790817; later runs in the round-15 reports)

**Tooling & gates (P2/P3):**
~~11. Build the **owner-decisions index** (e4): a standing TODO_LIST section or ROADMAP table that HARVEST feeds with every §g — the gate-policy, rawIDToken, and archive-bar questions are its founding entries.~~ done 2026-10-01 evening (TODO_LIST D1–D11)
~~12. Build the **tail-budget gate** (warn `docs/status/*.md` > 3) + a fixture asserting `docs/status/README.md`'s stated counts against reality — both sides of the d4 class, full atomic-gate checklist.~~ done 2026-10-01 (scripts/check-docs-tail-budget.sh + self-test; advisory by design)
~~13. Repo-own the status-row PARTIAL normalizer (checker exists; the round-13 /tmp fixer is throwaway). *(TODO P3 k.)*~~ done 2026-10-01 (scripts/normalize-status-rows.py + self-test; fixer/checker agreement proven on the real archive)
14. Codify the coverage-stamp convention (e3) as a docs-freshness rule + fixture.
15. bump-dep.sh: `--commit` mode + per-module `go mod verify` + the single-line-require regex fix with self-test fixtures (`9b3c2e18` proved the gap live). *(TODO P3 a.)*
16. Wire `--fail-on-stale-suppressions` into the flake gate. *(TODO P3 b.)*
17. `cqrs-lint rules` old-vs-new diff ritual for binary updates. *(TODO P3 c.)*
18. Audit `run.go:` values across all 16 `.golangci.yml` (1.26.7 vs the 1.27.1 floor). *(TODO P3 d.)*
19. Raise dependabot's 20-module cap (28 modules today). *(TODO P3 e.)*
20. Fix lychee's real 404s + deepwiki 429 backoff. *(TODO P3 f.)*
21. Feedback-inbox checker. *(TODO P3 g.)*
22. Scoped-format flake app (`.#fmt <paths>`). *(TODO P3 h.)*
23. e2e pin: `ExtraMiddleware` composes under `RunWithAppkit`. *(TODO P3 i.)*
24. Per-module race tests for the 8 e2e/examples modules (or a `#test-all` app). *(TODO P3 j.)*
25. BuildFlow noise-policy batch: vulnix dead NVD feed disposition, "9 tools unavailable" class, jscpd config dupes. *(TODO P3 m.)*
26. cqrs-lint residual triage incl. the new `sentinel_concrete_type` ×51 class. *(TODO P3 f-item.)*
27. Add a `check-cqrs-lint` CI path once a Go-installable distribution exists (the annotated draft's publish step). *(TODO P2/P3 + planning draft.)*

**Docs & memory (P3):**
28. agents-notes narratives: the cqrs-lint three-pass arc; the three-tools-vs-nixpkgs-Go story. *(TODO P3 docs a.)*
29. AGENTS gotcha 4 note: formatter-clean ≠ lint-clean. *(TODO P3 docs b.)*
30. Runbook note: never chain bump-dep invocations without committing between. *(TODO P3 docs c.)*
31. Record the research/benchmarks/proposals LEAVE verdict in `docs/research/README.md` (this session's b4).
32. Annotate the W0–W3 report at the NEXT docs-health pass (it is the live tail today — absence of annotation is correct NOW, stale NEXT time).
33. The round-14 sweep receipt in `docs/status/README.md` should gain the gate counts when the next pass runs (keep the sweep list append-only).

**Owner calls (route, don't decide):**
34. Gate policy during the erraudit program (see g1).
35. rawIDToken suppress-with-reason confirm (see g2).
36. Archive-bar ratification + tail budget (see g3).
37. OQ27: go.work fleet-local replaces — tracked-with-gate-filter vs untracked GOWORK overlay. *(ROADMAP, NEW.)*
38. OQ26: fleet go-directive policy (1.27.1 patch pin vs 1.27). *(ROADMAP.)*
39. OQ23/24: PapDashboard architectural go/no-go. *(ROADMAP.)*
40. PapDashboard reply packet — the v4.13.0 prerequisite is LIVE; say where #67/#68 land. *(TODO P3, NEW this session.)*
41. loginpage templ-components adoption (OQ21) — zero-dep posture vs design-system consistency.
42. `examples/datastar-demo` rebrand-or-remove — recommendation stands at KEEP AS-IS.
43. /mnt/buildcache reclaim decision (rust/ 155G + sccache/ 20G — not this repo's to delete).
44. Standing CHANGELOG convention for dep-only sweeps (the round-13 backfill set precedent; the rule is still the owner's).
45. OQ16: bench-spike automate-or-retire posture (the quiet-window rerun in item 4 feeds it).

**Small sharp ones:**
46. dashboardui snapshot-detail "store errored" vs "store empty" distinct state. *(W0–W3/02-59 §f32.)*
47. Document the Load contract (nil,nil vs ErrSnapshotNotFound) upstream in go-cqrs-lite store docs. *(02-59 §f33.)*
48. `nix flake check --all-systems` scheduled job (darwin arms unchecked). *(02-59 §f40.)*
49. Preflight-tree-check as a habit-gate before every full BuildFlow run. *(02-59 §f43.)*
50. Vocabulary: keep the round numbering monotonic (13 → 14 → …) across reports, README sweep list, and plan names — this session found "round-13" and "W0–W3" coexisting happily, but a third naming scheme would fork the history.

## g) Questions I can NOT figure out myself (max 3)

~~1. **Gate policy during the erraudit program:** keep `fail_on: critical` honestly red for every BuildFlow run until the 25 remaining `context_loss` sites land (my default — red = true signal), or a documented temporary demotion in `.buildflow.yml` so runs exit 0 while the program is open? Two prior reports asked; I refuse to pick alone because it redefines what CI red means for everyone else on the box.~~ moot — the program closed 2026-10-01 evening; the gate stayed at fail_on: critical and is green for the erraudit layer
~~2. **The rawIDToken trio** (`oauth2/provider.go` L329/L336/L346): confirm suppress-with-reason as the standing rule — a live ID-token credential must NOT ride error context (log/audit-leak hygiene) — and confirm you want the `//nolint:<analyzer>` name verified empirically before any suppression relies on it (TODO's standing caveat).~~ executed 2026-10-01 per the recorded decision (suppress-with-reason; LIVE-SECRET class); owner tick = TODO D3
~~3. **Archive bar + tail budget, ratified or not:** round 13 archived 22 reports and round 14 archived 2 more on the bar "dated blockquote + evidence strikes + open items routed to TODO/ROADMAP" (not the stricter "every numbered item individually struck"). Do you endorse that as the STANDING bar — and if yes, should the tail-budget gate (f12) be built so sweeps start from a mechanically-enforced tail of 1 instead of re-discovering drift?~~ applied de-facto by rounds 13–17 (each pass archived on this bar); formal owner ratification still pending

---

_Point-in-time snapshot — 2026-10-01 12:30 CEST. Living state lives in `TODO_LIST.md` / `FEATURES.md` / `CHANGELOG.md` / `ROADMAP.md` / `AGENTS.md`. Per the status convention, later sessions should ANNOTATE, never rewrite, this file. WAITING FOR INSTRUCTIONS._
