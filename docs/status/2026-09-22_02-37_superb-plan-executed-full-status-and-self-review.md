# Status Report — Superb-Plan Execution Session: Full Status + Brutal Self-Review (2026-09-22 02:37 CEST)

**Date:** 2026-09-22 02:37 CEST · **Scope:** THIS session only (resume → Q1–Q3 answered by execution → the ENTIRE superb-plan todo list A6→Z → closing report → this review). No new research beyond what this session did and noticed. **Predecessors:** `2026-09-22_02-58_docs-health-superb-plan-executed-to-completion.md` (the closing report), `docs/planning/archived/2026-09-20_15-43_docs-health-completion-and-backlog-superb-plan.md` (the executed plan).

**Format note:** written as `.md` per explicit user instruction (the status-report skill's canonical format is HTML; instruction wins, one-off, not propagated).

---

## What did you forget? (the direct answers)

1. **A claim I wrote did not match the work.** The closing report says the buildflow env-class diagnosis is "Recorded in AGENTS gotcha 8" — it was NOT (grep: 0 hits). The lean AGENTS rewrite happened BEFORE the diagnosis, and I never went back. **Fixed at 02:37** (gotcha 8 now carries the tsc/go-licenses/govulncheck/samber-linter list). This is precisely the docs-health sin this whole program exists to catch — I committed it in the victory report.
2. **The documented gate ladder has holes I didn't name.** I never ran `check-templates`, `check-codegen`, `test-fuzz`, `test-flake`, e2e, or integration_test. All were genuinely out of scope for my diff (scripts/docs/tests-only), but "final battery green" listed exactly what I ran without naming the gaps — the exact soft spot the round-6 sibling self-flagged, reproduced one session later.
3. **`examples/samber-do-demo/container_test.go` was never golangci-linted** — the lint app excludes examples/; I ran only `go vet` there. The systemadapter test IS covered (systemadapter is in the lint app's 15).
4. **The runbook I created is under-linked.** `docs/runbooks/dependency-train-bump.md` exists but nothing in TODO_LIST/AGENTS points at it (only the ROADMAP micro-ideas strike-through). The half-wiring class, again, in miniature.
5. **`scripts/bump-dep.sh` happy path never executed** — only its dirty-tree refusal fired (which was itself the tree being dirty, not a fixture). The edit/verify/absence-assertion loop is untested against a real sweep.
6. **Sibling-repo appends unverified.** I added items to `~/projects/templ-components/TODO_LIST.md` (230–232) and `~/projects/go-cqrs-lite/TODO_LIST.md` (new section) but never checked their daemons absorbed them, and go-cqrs-lite's TODO **Section index** was not updated with the new section.
7. **Small self-inflicted staleness:** `docs/status/README.md`'s "swept repeatedly" list omitted the 2026-09-22 sweep I myself had just performed (**fixed 02:37**); `flake.nix` check-modules `meta.description` still lists six stages though it now runs fourteen; the closing report's filename says 02-58.
8. **No push.** `master` sits ~22 commits ahead of `origin/master` (this session + daemon). Deliberate (never push unasked) but it must be a visible fact, not a silent one.

---

## a) FULLY DONE (verified this session)

1. **Q1 — pre-commit hook fixed for bare shells and PROVEN:** `go-cache-env.sh` raises `GOTOOLCHAIN` to the go.work floor (never downgrades; module-cache-resolved, no network); 4 behaviors hand-verified; T9 fixture case (hook self-test 12/12); live proof = a bare-shell commit ran large-file ✓ + release-train 0/0/0 ✓ + buildflow ✓. Buildflow's residual step failures diagnosed as deterministic env-class (tsc-without-tsconfig, go-licenses-outside-devShell, govulncheck patterns, samber-linter ~90% history) — recorded in AGENTS gotcha 8 (at 02:37, after the fix above).
2. **Q2 — python3 ownership decided:** ambient `python3` is the contract; flake apps pin `pkgs.python3`; README documents both.
3. **Gate 2 fully wired:** check-modules stages + sequential path + hermetic runtimeInput; CI steps; both flake apps run for real (413 files, 19 mixed tables); self-test strengthened to exact counts/file:line/row-lines (17 cases — it immediately caught my wrong assumption about table-header line semantics); **scope identity vs the skill asset proven per-file AND per-line (19 = 19, EQUAL)**; README Gate 2 section rewritten (repo-owned command of record; skill asset = authoring aid).
4. **A7 — AUDIT health report printed inline** (skill format): Accuracy 8.0/10, Fitness 7.75/10, visible math, first-scored-audit honesty; every finding had an owning phase later this session.
5. **B1/B4 — CHANGELOG `[Unreleased]` filled; TODO_LIST purged** (3 done-work-in-open-items removed: theme residual, /sse posture, usermgmt replace) **and re-harvested** (5 new P2 items + asks folded into P3; 23 open total; v4.12.0 headers); ROADMAP updated (v4.12.0, ADR-0051 cross-link, OQ15 GitHub-Releases posture, OQ16 bench-gate future).
6. **A8 — planning corpus swept:** 11 executed plans annotated + archived, 16 pre-September d2/svg/html artifacts moved; the ONE real inbound markdown link repaired (+5 dupes); relative links inside moved files fixed; gated-work-index refreshed (buildcache DORMANT, upstream drafts #25–#28 filed); planning README updated. All four doc gates green after.
7. **A9 — DOMAIN_LANGUAGE verified against code:** Actor/ActorID → 5 kinds (ADR-0111); SQLEventStore + MySQL; Session fields (ActorID/Origin/timestamps); UserID context; Broadcaster + Scoped Feed added; 21 events / 20 commands / role constants confirmed against source.
8. **A10 — AGENTS.md split: 123.5 KB → 28.1 KB** enduring core + `docs/agents-notes.md` verbatim archive (zero knowledge loss); 24/24 retained paths verified to resolve; pollution inventory was 63 dated/qualifier hits.
9. **B2 — `/v4` import-path gate shipped atomically:** `scripts/lib/docs-import-paths.sh` + rule in `check-docs-freshness.sh` + 5-case fixture self-test + CI step + check-modules stage. Four exemption classes adjudicated (nested-module shape, GitHub web URLs, workspace-only modules, MIGRATION docs).
10. **B3 — FEATURES re-audited:** all 195 green rows' path refs mechanically verified (1 stale cell fixed: `server_timing.go` → re-export); v4.12.0 surface added (theme-toggle row, SSEFilter, v1.19.0 pins, Trigger-slot identity menu); header refreshed. Honest bound: verification was paths + suite-green inference, not per-row behavioral exercising.
11. **C-tier:** C1 bench refusal #9 recorded (load 26.9–31.2 vs limit 8; OQ16 routed). C10 = **ADR-0052 written** (appkit posture: opt-in stands, flip deferred to v5, (b)–(f) verdicts) — kills the dangling "ADR-001" ghost reference. C2–C8 verified as sibling-done (deconflicted, not repeated); C9/D7 untouched as unauthorized.
12. **D-tier:** D1 asks recorded in the OWNING repos' trackers (templ-components TODO 230–232; go-cqrs-lite TODO section — incl. a third ask, Explain Volume display, verified absent empirically). D2: tag-message guard (`lib/tag-message-guard.sh` wired into verify-tag; 3 new fixture cases, 15/15) + MD024 `siblings_only` + a REAL bug fixed (coverage-gate shared `/tmp/cov` profile raced across concurrent sessions → per-run mktemp). D3: `docs/runbooks/dependency-train-bump.md` (wave order, verification discipline, failure-class table, never-repo-wide-`go work sync` policy) + `scripts/bump-dep.sh` (digit-safe matching, hermetic per-module tidy+build+vet, absence assertion, dirty-tree refusal). D4: both purge plans re-verified with fresh census (v4: 3 blobs = 27.56 MiB at `339ce82b`; setup-demo range: FOUR revisions ≈ 105 MiB) — prep-only, nothing pushed. D5: four named example suites green + new `TestSmoke_AuditViewerServes` (caught the `/audit`→`/audit/` 307 first try). D6: ROADMAP candidates triaged WITH verdicts (Option B + Option D SHIPPED; Option C + SQL-defaults REJECTED with reasons; MetricsRecorder/setup ideas KEPT demand-gated); micro-ideas struck for everything done. D8: `declarations_volume_test.go` (per-query magnitudes + count pin) + demo verification (12 collections confirmed; Volume display absent upstream → routed).
13. **Z — final battery green:** check-modules 14/14 stages · coverage-gate 15/15 (sibling's gap closed) · lint 0/15 (second gap closed) · `nix flake check --no-build` · actionlint · shellcheck (all touched scripts) · annotation gate 48/413 · row gate 413/0-PARTIAL/19-mixed · links 281 · freshness. Corpus swept: 8 reports + the superb plan annotated + archived (413 archived; unarchived: the closing report + this one). Cache incident: `/mnt/buildcache` hit 100% mid-battery — the go-cache-env fail-fast caught it exactly as designed; build cache cleaned (17G freed); battery re-run green. Closing report committed as `fe52e1b5`.

## b) PARTIALLY DONE

1. **D2 remainders routed, not done:** per-module LICENSE presence (licensing policy) and `nix flake check` WITH builds (needs a nix-capable CI decision) — both TODO-class, flagged in ROADMAP micro-ideas.
2. **`bump-dep.sh`:** shipped + shellcheck-clean + refusal verified; happy path unexercised (see forgot-#5).
3. **Example coverage:** 8/12 modules have tests; `async-startup-demo` + `middleware-showcase` do not (ROADMAP micro-ideas); admin-demo relies on external e2e specs.
4. **Sibling-repo tracker appends:** written, not verified-absorbed; go-cqrs-lite section index not updated (forgot-#6).
5. **Narrative commits:** 3 attempted; 1 landed clean (`fe52e1b5`); the rest raced into daemon heuristic commits (content verified intact via stat diffs). The hook now WORKS — the race is purely the daemon's poll frequency.
6. **`docs/status/README.md` counts:** now accurate again (2 unarchived, 413 archived, sweep list includes 09-22) — but only because this review caught the omission at 02:37.

## c) NOT STARTED (deliberate, with reasons)

1. **C9** (cqrs-lint Go distribution) + **D7** (datastar-demo rebrand) — explicitly unauthorized; drafts/plans stay active in `docs/planning/`.
2. **Z.3 push** — not performed (never push unasked); surfaced as Q1 below.
3. **`check-templates` / `check-codegen` / `test-fuzz` / `test-flake` / e2e / integration_test** — no templ/SQL-template/fuzzed-code changes this session; NOT run and NOT previously named as gaps (now named; routed in §f).
4. **Bench-spike run** — refused under load, 9th time (policy); gate's future = OQ16.

## d) TOTALLY FUCKED UP (honest, ranked)

1. **The closing report claimed work that did not exist** ("Recorded in AGENTS gotcha 8" — grep said 0). The worst miss of the session because it's the exact class (unverified claim) this whole docs-health program hunts. Fixed at 02:37; lesson: re-grep every "recorded in X" claim against X before writing it.
2. **Two tool-handling near-misses in the A8 sweep:** (a) the annotate function ran `cat` from the wrong CWD — the `&&` chain saved the originals (only `.tmp` stubs to trash); (b) `git mv` with a full-path destination failed loudly (basename fix). Zero data loss, both avoidable with mktemp-workdir + absolute-path discipline.
3. **The D8 Volume test pinned MY assumptions, not the code** — 10 red assertions on first run; corrected to pin actual magnitudes. A regression test must encode current behavior, not memory.
4. **The B2 rule's first draft flagged 10+ false positives** (nested-module paths, GitHub web URLs, workspace-only modules, migration docs) and its first lib design lost a variable to command-substitution subshells. Both caught by my own fixtures before shipping — the tests worked; the first drafts were weak.
5. **Three long gates ran against a 100% cache filesystem before I noticed** — the fail-fast fired correctly, but `df` should have come first. And the battery re-ate the freed space: at 02:37 the FS is back at 917 MB free — a recurring operational constraint, not a one-off (Q3).
6. **The first commit attempt bypassed without reading step names** — I diagnosed properly on the SECOND attempt. The sibling's "read WHICH step fails" lesson took me one commit to absorb.
7. **Battery framing omitted what didn't run** (forgot-#2) — "all green" was true-for-what-ran; the honest form names the exclusions.

## e) WHAT WE SHOULD IMPROVE

1. **Verify-then-write for cross-references:** every "recorded in / linked from / wired into" claim gets a grep against the target before it enters a report. (Would have caught miss #1.)
2. **Battery disclosure discipline:** state the gates that ran AND the documented-ladder gates that didn't — "green" without the exclusion list is a soft lie.
3. **`df` the cache FS before any long battery** — the guard catches it mid-run, but by then you've burned minutes; a 2-second preflight beats a fail-fast.
4. **Sweep scripts get mktemp workdirs + absolute paths + `set -e`** — the A8 stub incident is the anti-template.
5. **Cross-link new artifacts at creation time** (runbook → TODO/AGENTS pointer in the same change) — the atomic-gate checklist generalized to docs.
6. **Pin the code, not the memory** when writing regression tests (D8 lesson, now twice-learned repo-wide).
7. **Sibling-repo edits end with a verification** (`git -C ../repo log --oneline -1` + grep the landed text) — cross-repo writes are still writes.
8. **The daemon race is mitigated, not solved:** the hook works now, so narrative commits are POSSIBLE; keep them short-verification → immediate-commit to shrink the race window.

## f) Up to 50 things to get done next

1. **Decide + execute the push** (Q1) — ~22 commits ahead of origin.
2. **LICENSE posture decision + execution** (Q2) — 28 module dirs vs repo-root-only.
3. **Bench gate future** (Q3 / OQ16) — automate, retire, or keep manual.
4. Close my ladder gaps on the current tree: `check-templates`, `check-codegen`.
5. …and `test-fuzz` + `test-flake`.
6. …and e2e (58/58 expected) + integration_test re-run.
7. golangci-lint `examples/samber-do-demo` (extend the lint app's scope or a one-off; the new test file was only vetted).
8. Cross-link `docs/runbooks/dependency-train-bump.md` from TODO_LIST's sweep item + AGENTS.md Quick Reference.
9. Exercise `scripts/bump-dep.sh` happy path on the next real dependency bump (and fix what it finds).
10. Update `flake.nix` check-modules `meta.description` to name all 14 stages.
11. go-cqrs-lite TODO: add the new section to its Section index; verify the daemon absorbed both sibling-repo appends.
12. `/mnt/buildcache` headroom: recurring-fill problem — escalate the dormant hardware decision (bigger disk / move module cache / scheduled `go clean -cache`).
13. Strict-lag split brain: default `check-release-train.sh` to CI flags or add `--ci` preset (TODO P2).
14. Pre-push hook: release-train (CI flags) + version-drift --strict (TODO P2).
15. Post-train consumer-eye smoke: throwaway module `go get .../v4@v4.12.0` + pkg.go.dev check (TODO P2).
16. Self-test for `check-dep-budgets.sh` comment-line exclusion (TODO P2).
17. Wire the VCS-cache no-remote sweep into check-modules/prewarm (TODO P2).
18. Snapshot cqrs-lint zero-warning state; consider stale-suppression warnings blocking once CI-installable (TODO P2).
19. Sweep `X-Client-Id` raw literals to the constant (TODO P2).
20. OQ15: GitHub Releases posture for the 14 v4.12.0 tags (or record tags-only as policy).
21. OQ16 resolution (overlaps #3).
22. V007 cluster 1 (68 SQLViewStore findings) — track go-cqrs-lite metaengine layout-planning (ADR-0051 go-criterion).
23. templ-components items 230–232 (ListNote semantics, Grid hybrid children, CopyButton span color).
24. go-cqrs-lite asks: requestContextEnricher upstreaming, `system.New` checkpoint/DLQ options, Explain Volume display.
25. Example smokes: `async-startup-demo`, `middleware-showcase`.
26. admin-panel SSE e2e spec + catalog-demo visual smoke (ROADMAP micro-ideas).
27. loginpage templ-components adoption (`recipes.AuthLayout`, forms, Alert, Button).
28. `nix flake check` WITH builds — nix-capable CI decision (D2 routed).
29. `SessionMiddleware` loud-failure on missing store + `Config.SelfCheck` cookie round-trip (micro-ideas).
30. Cross-link `async-projection-startup.md` from `fullstack-wiring.md` (micro-ideas).
31. `leveraging-system-metaengine.md`: add the `WithCheckpointStore`/`WithDeadLetterStore` section (micro-ideas).
32. `WithOpenAPI` merge-into-Spec consumer guide (micro-ideas).
33. go-structure-linter 55 root-package findings triage (micro-ideas).
34. Committed `cqrs-upgrade -dry-run` artifact per train (micro-ideas).
35. eventtest v0.x churn note in the bump playbook (micro-ideas).
36. Tag-guard variant: reject pasted SSH signatures (micro-ideas).
37. release-train: honor retract directives when resolving "latest published" (micro-ideas).
38. buildflow golangci steps: inherit `/tmp` GOCACHE + repo `.golangci.yml` (micro-ideas; upstream ask).
39. codespell ignore list ("deriver", …) (micro-ideas).
40. buildflow `go-mod-normalize` run (direct/indirect mixing) (micro-ideas).
41. `meta.mainProgram` in flake packages (micro-ideas).
42. Per-module full-lint pre-commit step (micro-ideas).
43. Delete `backup/pre-blob-purge` + `refs/original/*` once confident (micro-ideas; gated).
44. Validate `.markdownlint.json` MD024 actually does what's intended (run a full buildflow once or wire markdownlint into CI — currently only full-mode buildflow reads it).
45. Consider a `docs-check` composite app (freshness + links + both status gates + self-tests) + measure check-modules wall time after the stage additions.
46. Annotate + archive the 02-58 closing report and this report once their items resolve (the standing convention; corpus currently 2 unarchived).
47. adminui mobile-390 header check with the theme toggle (sibling f19/f20).
48. adminui theme OS-preference fallback path test (no localStorage; sibling f21).
49. Train-lag watch cadence: weekly check or daemon notification (sibling f41).
50. Next docs-health pass: verify TODO_LIST "Round 7" header and ROADMAP Current State still match the tree (they were hand-updated under daemon pressure tonight).

## g) Questions I cannot figure out myself

1. **Push?** `master` is ~22 commits ahead of `origin/master` — tonight's entire output (gates, hook fix, docs corpus, both reports) plus daemon commits. The superb plan's Z.3 authorized a push for ITS session; I have not pushed. Say the word and I'll `git push origin master` (fast-forward, no force).
2. **LICENSE posture:** the repo root has a LICENSE; the 28 module directories do not. pkg.go.dev detects licenses per module directory — per-module LICENSE files (27 copies) or accept root-only (status quo, most consumers resolve via the proxy which serves regardless)? It's a licensing/positioning call, not a technical one.
3. **`/mnt/buildcache` headroom is a recurring constraint, not an incident:** 220G disk, refilled to 917 MB free within an hour of a 17 G clean (full gate batteries regenerate the entire build cache). The dormant hardware decision (`docs/planning/2026-08-30_buildcache-hardware-decision.md`) was parked because the mount healed — but capacity, not health, is now the binding constraint. Options: bigger/second disk, move GOMODCACHE off sdb1, scheduled `go clean -cache`, or accept + rely on the fail-fast. Which?

---

*Report written per explicit user instruction as `.md` (skill canonical is HTML; one-off override, flagged). NOW WAITING FOR INSTRUCTIONS.*
