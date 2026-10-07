# Status Report — Tooling Round 2: scoped format, status-row normalizer, tail-budget, test-all, bump-dep hardening (shared-tree session)

> **ANNOTATED 2026-10-04 (docs-health round 17):** the honest b-section gaps all closed 2026-10-01/02 — heavy batteries ran (`.#test` 15/15, `.#test-all` 28/28, composite 27/27, lint 0, coverage 15/15), the e2e leg was already green when this report shipped, and the ARCHIVE pass this report called for executed today (the 13-report tail annotated + archived). T08/T10/T16/T24/T25 resolved to their non-owner halves; T09 answered (3 filed / 2 retired). Still open (routed): bench-spike, T11/T12 owner debt, D6/D7 ticks, the check-docs-counts mechanical gate, the load-policy runbook line. The stage-count drift class this report coined recurred exactly as predicted — 28 today. Archived this pass.

> **Session:** 2026-10-01 ~14:30 → 17:36 CEST (report written 17:36 CEST)
> **Scope:** this session only — the Pareto round-13 plan's remaining repo-local tooling items (T13/T14/T15/T17/T18-half/T21/T22-half/T19-partial), executed against the **shared** `cqrs-htmx` tree.
> **Trigger:** owner pasted `TODO_LIST.md` + "Break this down … execute and verify … get the whole list done", then "write a full status report".
> **Format note (explicit override):** the status-report skill's canonical output is a styled HTML dashboard; the owner explicitly asked for `.md`. This file honors the `.md` request and flags the divergence per the skill.
> **Machine context:** shared fleet box, **load 20–51** the whole session (32 cores) — every heavy gate (`#test`, `#coverage-gate`, `check-modules` composite, `golangci-lint` at scale) was deliberately **refused** to avoid load-noise reds (the repo's own runbook rule). `/mnt/buildcache` at 90% used.
> **Ingestion:** read `TODO_LIST.md`, AGENTS.md, the 10:30 W0–W3 report, the 12:30 round-14 report, the 06:47 round-13 pareto plan, the round-13 archived report, and the relevant scripts/flake/configs in full.
> **SHARED-TREE WARNING (read first):** a **concurrent session** edited this tree the entire time (verified via `git log 818bee0a..HEAD`). Several committed changes under "since session start" are **not mine** — see §a preamble and §d3. Attribution below is per-file, not per-commit (the auto-commit daemon batches multi-session edits into heuristic `chore:` commits).

---

## Executive verdict

The repo-local tail of the round-13 plan is largely **shipped with fixture coverage**: five new tools/gates (`normalize-status-rows`, `check-docs-tail-budget`, `#fmt`, `#test-all`, `#test-bump-dep`), `bump-dep` hardened (`--commit`/`--verify`/single-line-require fix), `run.go` audited ×16 → `1.27.1`, real docs 404s fixed, and the AGENTS/runbook/CHANGELOG/TODO/plan docs updated. Every new script passed its own fixture self-test; every offline docs/config gate is green. **What is NOT proven is the heavy composite** — I could not honestly re-run `check-modules`/`#test`/lint under load 20–51, so my "green" claims are offline/config-scoped, not composite-scoped. The most valuable honest catch: the concurrent session had already done **T22(i)** (the e2e `ExtraMiddleware`/`RunWithAppkit` pin) and **T06** (erraudit), so my plan's remaining-work framing was partly stale on arrival.

## Scores (health-report format)

- **Accuracy 82/100** — every claim I make is backed by a command I ran this session (per-file commit hashes, self-test outputs, gate rc). −18: I wrote a **stage count (27) that was stale within the hour** (a concurrent session added `erraudit-inventory-self-test` → 28) and I could not re-verify the composite, so the "check-modules ✅" class is trust, not proof (see §d3).
- **Fitness 85/100** — the tools are exactly the mechanizations the round-13/14 reports asked for (normalizer, tail-budget, scoped fmt, bump-dep daemon-race fix). −15: the advisory tail-budget gate is **already firing** (4 reports in the tail > budget 3) partly _because of this very report_, and I left the deferred feedback-inbox item and two docs items unfinished.

---

## a) FULLY DONE (verified this session)

> **Preamble — attribution.** My changed files (verified with `git log -1 -- <path>`): `CONTRIBUTING.md`, `datastar/README.md`, `docs/guides/leveraging-system-metaengine.md`, the 4 consumer-feedback `.md` files, all 16 `.golangci.yml`, `flake.nix` (my apps only), `.github/workflows/ci.yml` (my steps), `scripts/bump-dep.sh`, `scripts/test-bump-dep.sh`, `scripts/normalize-status-rows.py`, `scripts/test-normalize-status-rows.sh`, `scripts/check-docs-tail-budget.sh`, `scripts/test-check-docs-tail-budget.sh`, `AGENTS.md`, `TODO_LIST.md`, `CHANGELOG.md`, `docs/status/README.md`, `docs/runbooks/dependency-train-bump.md`, and the round-13 plan. **Not mine** (concurrent session): `flake.lock`, `scripts/erraudit-inventory.sh` (+self-test), `identity-model/authz_roles.go`, `usermgmt/oauth2/provider.go`, `usermgmt/service_oauth2_extracted.go`, `usermgmt/sql_readmodel*.go`, `usermgmt/totp.go`, `usermgmt/webauthn_service.go`, `usermgmt/es_readmodel.go`, `setup/run_appkit_test.go`, screenshots, the 15:55/17:01 reports, the round-14 plan.

1. **T14 — real documentation 404s fixed.** `CONTRIBUTING.md` golangci-lint install URL `…/usage/install/` → `…/docs/welcome/install/` (verified via the site sitemap); `datastar/README.md` + `docs/guides/leveraging-system-metaengine.md` `tree/main` → `tree/master` (verified default branch `master` via `raw.githubusercontent`); the DiscordSync/overview/sec consumer-feedback links (private repos, 404 anonymously) → inline code. Files at `a359bb5d`. Evidence: `bash scripts/check-docs-links.sh` → **315 links OK, rc=0**.
2. **T15(d) — `run.go` audited across all 16 `.golangci.yml`.** Stale `1.26.4/1.26.5/1.26.7` → **`1.27.1`** (the go.work floor). Evidence: `golangci-lint config verify` **rc=0 on every module dir**; a sampled module (`health`) lints **0 issues** at the new value; discovered and documented the trap that `1.27` (no patch) is rejected because YAML parses it as a number. Files at `2b5a3ec2`.
3. **T15(e) — dependabot "20-module cap" closed as superseded.** Verified `.github/dependabot.yml` deliberately watches only root/usermgmt/integration_test (2026-09-20 posture comment); raising it would duplicate every train bump as ~24 PRs. **No change made — the item's premise was stale.** Recorded in CHANGELOG [Unreleased] Changed.
4. **T13 — `bump-dep.sh` hardened.** Added `--commit` (stages + commits the sweep in-process, beating the auto-commit daemon), `--no-verify`, a per-module `GOMODWORK=off go mod verify` step, and the **single-line-`require` regex fix** (`^[[:space:]]*` → `^([[:space:]]*|require[[:space:]]+)` + field-aware path extraction) — the `9b3c2e18` gap. Files at `d5c00773`. Evidence: `scripts/test-bump-dep.sh` **3/3 cases green** (block form, single-line form, `$` anchor, testdata skip, no-match no-op); `nix run .#test-bump-dep` green.
5. **T17 — status-row PARTIAL normalizer (the fixer-side gap).** `scripts/normalize-status-rows.py` completes a PARTIAL row by whole-row-strike (the 2026-10-01 round-13 policy: 72 rows), idempotent, `--dry-run`, code-span-aware. Screenshot: `75cb8331`. Evidence: `scripts/test-normalize-status-rows.sh` **7/7 green**; on the real archive → `nothing to fix across 437 file(s)`, proving fixer and checker agree. Ships the full atomic checklist: script + self-test + 2 flake apps (`.#normalize-status-rows` / `.#test-normalize-status-rows`) + check-modules stage (`status-rows-normalize-self-test`) + CI step + `docs/status/README.md` command.
6. **T18(l) — docs-status tail-budget advisory gate.** `scripts/check-docs-tail-budget.sh` warns (exit 0) when the live `docs/status/*.md` tail (excluding README) exceeds 3; `--strict` exits 1. Portable (no `find -printf`). Files at `16732e40`. Evidence: `scripts/test-check-docs-tail-budget.sh` **5/5 green**; on today's tree → **warns "4 reports exceed budget 3"** (see §d3). Flake apps + CI self-test step + README.
7. **T21 — scoped-format flake app.** `nix run .#fmt -- <paths>` forwards paths to the treefmt wrapper (`config.treefmt.build.wrapper`, config + tree-root baked in), ending the bare-`treefmt <paths>` failure. Evidence: verified it traversed **only the 2 named files**, not the tree; also used it to format my own scripts (shfmt + shellcheck).
8. **T22(j) — `#test-all` app.** `nix run .#test-all` runs race tests over **every** workspace module incl. the 8 e2e/examples modules `#test`/`#test-race` exclude. Evidence: `nix eval …test-all.meta.description` resolves; app builds.
9. **T19(b)(c) + T16(c) — docs/memory.** AGENTS **gotcha 10** gained the _formatter-clean ≠ lint-clean_ bar + the scoped-`.#fmt` rule; **gotcha 23** (new) records the `run.go` pin, the `cqrs-lint rules` before/after-rebuild diff ritual, the `--fail-on-stale-suppressions` block, and the new tooling surface; `docs/runbooks/dependency-train-bump.md` gained the never-chain-bump-dep / MVS-carries-siblings note + `--commit`/`--verify`/`BUMP_DEP_ROOT`. Files at `20e81d9f` / `519dd929`.
10. **Bookkeeping.** `TODO_LIST.md` tooling item rewritten to done/closed/remaining; header + battery count restamped; `CHANGELOG.md` [Unreleased] Added (tooling round) + Changed (`run.go`, 404s, dependabot) entries; the round-13 plan got a dated **OUTCOME** blockquote (ANNOTATE, non-destructive).

## b) PARTIALLY DONE

~~1. **The heavy-gate verification (the honest big one).** All _offline_ gates pass: `check-docs-links.sh` (315 OK), `check-status-rows.py` (0 PARTIAL/437), `check-status-annotations.sh` (72/72), `check-docs-freshness.sh` (PASS), and 5 self-tests. **Not run:** `nix run .#test`, `.#coverage-gate`, `.#lint`, and the `check-modules` composite — refused under load 20–51 per the repo's own rule. My changes are docs/config/scripts only (zero `.go` edits) so build-regression risk is near-zero, but the composite is **unproven**. Effort to finish: **S** (one quiet-window run).~~ done 2026-10-01 evening/overnight — .#test 15/15 + .#test-all 28/28 race rc=0, lint 0, coverage 15/15, check-modules 27/27 composite (round-15 reports)
~~2. **The check-modules stage count is a moving target.** I added 2 stages (25→27) and wrote "27" into TODO/CHANGELOG/AGENTS; a concurrent session then added `erraudit-inventory-self-test` → **28**. My numbers are already stale (shared tree). Effort to reconcile: **S**, but the class recurs (see §e1).~~ superseded — the suite is 28 stages today; the check-docs-counts mechanical gate remains unbuilt (TODO)
~~3. **`#test-all` is built but never executed end-to-end.** The app evaluates and builds, but I did not run it (examples bind ports; load high). So "the 8 excluded modules now have a race path" is a capability claim, not a result. Effort: **S–M** (one run).~~ done 2026-10-01/02 — executed 28/28 modules rc=0
~~4. **T19(a) agents-notes narratives** — not written (prose assembly of already-recorded facts). Deliberately low priority.~~ done 2026-10-01 (two narratives in docs/agents-notes.md)
5. **Feedback-inbox checker (T18 second half)** — deferred with a concrete reason, not abandoned (see §c4).

## c) NOT STARTED (routed; not this session's scope, or blocked)

~~1. **T08 — system `cqrs-lint` binary rebuild.** BLOCKED (owner): still the pre-fix build `3756eb4/20260929`; until rebuilt, manual root walks print 21 stale C040 phantoms. Priority: still wanted (it gates T10/T16).~~ local half done 2026-10-01 evening (rebuilt binary 0×C040, 14/14 modules); fleet swap owner-pending (TODO D4)
~~2. **T10 / T16 — cqrs-lint residual triage + `--fail-on-stale-suppressions` wiring.** BLOCKED on T08 by design (wiring the flag now risks false-failures from the pre-fix binary). Priority: wanted once T08 lands.~~ done 2026-10-01 evening (residual triage note + --fail-on-stale-suppressions wired with fixture self-test)
~~3. **T09 — file the 5 fleet upstream asks.** NOT STARTED — needs filing authorization (owner).~~ done 2026-10-01 evening (3 filed / 2 retired at the verification gates)
4. **T18 feedback-inbox checker.** DEFERRED: 7 legacy `processed/` files predate the `> **PROCESSED**` marker and `docs/feedback/sec-consumer-feedback.md` sits at the feedback root (not `new/`) — a gate would need an epoch exemption + an owner disposition for the stray file. Priority: medium; needs a decision first.
~~5. **T24 — BuildFlow noise-policy batch** (vulnix dead NVD feed, "9 tools unavailable", jscpd config dupes). NOT STARTED.~~ done (docs/runbooks/buildflow-noise-policy.md)
6. **T11/T12 — go-cqrs-lite cross-repo docs + vet/lint/-race.** Owner-gated.
7. **T23 — cqrs-lint Go-installable distribution / strict CI gate.** Blocked on the Nix-only binary.
~~8. **T03/T04 remainder — e2e Playwright + bench-spike.** Quiet-window only; NOT run this session.~~ e2e DONE 2026-10-01 (70/70 rc=0); bench-spike still owed at a verified-quiet window — routed TODO P2
~~9. **T25/T26 — owner-call packets + v5-window watches.** Owner calls; untouched.~~ done (owner decision packets in docs/proposals/; v5-window watches refreshed — appkit, ProjectionLayer, DataStar T4, buildcache)

## d) TOTALLY FUCKED UP (radical honesty — nothing shipped broken this session; all my gates are green)

1. **I published a number that was stale within the hour.** I wrote "check-modules 25→27" into TODO/CHANGELOG/AGENTS; a concurrent session added a stage → 28 before I finished the report. Severity: low (no runtime impact) but it is **exactly the round-14 d4 class ("count-claims drift on arrival")** recurring under my own hands — the tail-budget gate I just built does not cover _stage_ counts. Mitigation: the report and TODO now say "28" or "moving target".
2. **No composite/heavy verification.** I cannot say "the gates are green" without the qualifier "the offline ones". Severity: low-risk (no Go code touched) but it means my integration claim is trust-based. Mitigation: a single quiet-window `nix run .#check-modules -- --report` closes it.
3. **My tail-budget gate is already firing — because of the shared tree, and partly because of THIS report.** At write time the live tail is 4 (`10:30`, `12:30`, `15:55`, `17:01` + this = 5), over the budget of 3. Severity: advisory by design (exit 0) — but it means a docs-health ARCHIVE pass is owed, and I am _adding_ to the pile by writing a report into a non-empty tail. That is the gate working as intended, not a bug — but it is a live signal.
4. **I did not resolve the stray `docs/feedback/sec-consumer-feedback.md`** (should be in `new/` or `processed/` per gotcha 20). Severity: low hygiene. Mitigation: documented in TODO + this report; needs a disposition.
5. **Two "fuck-ups" that turned out to be non-events (verified):** the `find -printf` non-portability in my tail-budget script (fixed before commit → `find | sed`); and my initial `run.go: 1.27` (caught by `config verify` before commit → `1.27.1`).

## e) WHAT WE SHOULD IMPROVE

1. **Counts in living-doc headers drift every time the suite grows.** The stage count, coverage %, and "N/25" claims are hand-maintained and go stale on any concurrent change (this session proved it twice). _Fix:_ have `flake.nix` expose a `.#stage-counts`/`check-docs-counts` app that asserts header numbers against reality (the round-14 report already proposed a README-count fixture for the tail side — extend it to stages). Impact: kills the d4 class permanently.
2. **Shared-tree sessions must re-verify counts/claims immediately before writing them**, not only compute-then-write (my "27" was true when computed, false when written). _Fix:_ a "re-read the number at write time" habit + `nix run .#preflight-tree-check` before doc-writing phases.
3. **Advisory gates need an explicit "who runs me" line.** My tail-budget gate is advisory and unwired by design, which means nothing runs it in CI except its self-test. _Fix:_ either wire the advisory check (exit 0) into the `checks` CI job so it _prints_ each run, or accept that it is a local sweep tool and say so in the README.
4. **`#test-all` (and the examples' race path) should be exercised at least once** so the capability is proven, not just declared.
5. **`bump-dep --commit` deserves a real-world proof run** (one harmless sweep) before it is trusted in a train — I did not run one (correctly: no sweep was needed, and the runbook forbids chaining).

## f) Up to 50 things to get done next (impact-sorted; the top ~15 are this session's harvest)

**P1 — unblocks value/honesty:**
~~1. Rebuild the system `cqrs-lint` binary from the fixed `~/projects/go-cqrs-lite` source; verify `cqrs-lint --strict --verbose .` shows 0 C040. _(TODO P2; owner.)_~~ local half done 2026-10-01 evening; fleet swap owner-pending (D4)
~~2. Quiet-window `nix run .#check-modules -- --report` (now 28 stages) + `#test` race suite + `#coverage-gate` — convert my "offline-green" into composite-green. _(this report §b1.)_~~ done 2026-10-01 ~15:50 (27/27 under load 80+ — the 15-55 report §a W10)
~~3. Reconcile the check-modules stage count everywhere it is written (28), or replace the hand-count with a derived `check-docs-counts` gate. _(§e1.)_~~ superseded — 28 stages today; the mechanical check-docs-counts gate remains unbuilt (TODO)
~~4. Run `nix run .#test-all` once (prove the 8 e2e/examples modules' race path). _(§b3.)_~~ done — 28/28 modules rc=0 (2026-10-01/02)
~~5. Run the e2e Playwright suite in a quiet window (the snapshot-sentinel proof still owed from W0–W3). _(TODO P2 battery.)_~~ stale on arrival — the 70/70 run landed 2026-10-01 ~15:00 (see the 17-01 report §a T04)
6. Re-run `bench-spike` after the 09-30/10-01 dep bumps (quiet window; OQ16). _(TODO P2 battery.)_
~~7. **HARVEST this report's §f into TODO_LIST/ROADMAP** (docs-health HARVEST) — otherwise these die in a timestamped file. _(skill requirement.)_~~ done (round-15 HARVEST + the round-17 archive pass)
8. Owner: decide the feedback-inbox convention (epoch for the `> **PROCESSED**` marker + disposition of the stray root file) so the checker (item 12) can be built unambiguously.
~~9. File the 5 fleet upstream asks (filing authorization needed). _(TODO P2.)_~~ done 2026-10-01 evening (3 filed / 2 retired)
10. Bundle the erraudit remainder with the next train (published-module code). _(TODO P2.)_
~~11. Wire `--fail-on-stale-suppressions` into the `check-cqrs-lint` flake app **after** T08 lands; add the stale-suppression fixture. _(T16.)_~~ done 2026-10-01 evening — wired EARLY (the flag predates the C040 fix; caught a real stale B024 at usermgmt/es_setup.go:221)
12. Build `scripts/check-feedback-inbox.sh` (after item 8): `new/` empty at train time + `processed/` files carry an outcome marker (epoch-exempt for legacy). _(T18 half.)_
~~13. Run an ARCHIVE pass on the `docs/status` tail (now 5, over budget 3) before the tail drifts further. _(§d3.)_~~ done — this pass (round 17, 2026-10-04): the full 13-report tail annotated + archived
~~14. Add AGENTS/runbook notes for the new gates' _who-runs-me_ status (advisory vs blocking). _(§e3.)_~~ done (docs/status/README.md documents the advisory tail-budget; the blocking gates are named in AGENTS quick-ref)
15. Add a `check-docs-counts` app asserting `docs/status/README.md`'s stated counts (tail/archived/gated) against reality. _(round-14 f12.)_

**Tooling & gates (P2/P3):**
16. Real-world proof of `bump-dep --commit` on one harmless sweep (rides the next train). _(§e5.)_
17. `cqrs-lint rules` diff ritual → make it a script (`scripts/cqrs-lint-rules-diff.sh`) instead of prose. _(T16(c) follow-up.)_
18. Coverage-stamp convention rule (round-14 f14): a docs-freshness greppable rule that forbids a percentage adjacent to a re-run date without a measured date.
19. Owner-decisions index (round-14 f11): a standing TODO/ROADMAP table HARVEST feeds (the gate-policy, rawIDToken, archive-bar questions are its founding entries).
20. Raise dependabot coverage ONLY if a real third-party-drift incident appears (currently superseded — keep as a documented decision, not a task). _(T15(e).)_
21. Audit the remaining `docs/feedback/` root-file convention (why `sec-consumer-feedback.md` is not in `new/`).
22. `check-css-bundle-classes.sh` on the concurrent session's screenshot/style changes (they touched dashboard assets). _(shared-tree hygiene.)_
23. Confirm the concurrent session's `flake.lock` `systems`-input removal still leaves `nix flake check` green on all systems. _(shared-tree hygiene.)_
24. Wire `#test-all` into a scheduled (not per-push) CI job so the examples' race path runs without blocking.
25. Add a fixture for `normalize-status-rows.py` covering a table with a `|` inside a code span (current fixture avoids it).

**Docs & memory (P3):**
26. T19(a) agents-notes narratives (cqrs-lint three-pass arc; three-tools-vs-nixpkgs-Go). _(TODO P3.)_
27. De-duplicate the "atomic-gate checklist" prose now that it appears in AGENTS gotcha 19, the round-13 report, and CHANGELOG — one canonical location + links.
28. Record the "shared-tree concurrent session" mechanics (my session's reality) as an agents-notes narrative.

**Owner calls (route, don't decide):**
29. Gate policy during the erraudit program (round-14 g1).
30. rawIDToken suppress-with-reason confirm (round-14 g2).
31. Archive-bar + tail-budget ratification (round-14 g3).
32. OQ23/24 PapDashboard go/no-go.
33. OQ26 fleet go-directive policy.
34. OQ27 go.work fleet-local replaces.
35. PapDashboard reply packet.
36. loginpage templ-components adoption (OQ21).
37. `examples/datastar-demo` rebrand-or-remove.
38. `/mnt/buildcache` reclaim decision.

**Small sharp ones:**
39. Delete the orphan `examples/*/<binary>` build artifacts if they are not gitignored (they are — no action, but confirm).
40. Verify `#fmt` behaves on a directory arg (`nix run .#fmt -- scripts/`).
41. Add `--strict` guidance for `check-docs-tail-budget` into the release runbook (train-time check).
42. Cross-link `docs/status/README.md` ↔ the new `.#normalize-status-rows` apps in the flake README.
43. Re-derive the erraudit 61-vs-54 extractor discrepancy (concurrent session's inventory script may have changed the number).
44. Confirm `setup/run_appkit_test.go` (concurrent T22(i)) covers the `RunHandler`/`Run` paths too, not just `RunWithAppkit`.
45. Keep the round numbering monotonic (13 → 14 → …) — this session informally adds "round 2"; avoid forking naming.
46. Add a CI actionlint run for the new steps (actionlint already runs; verify it does not warn on my additions — needs docker, not run here).
47. Document `BUMP_DEP_ROOT` in the runbook's env section (done inline; verify wording).
48. Add the `#fmt`, `#bump-dep`, `#test-all` apps to the flake README app list if one exists.
49. Prune `/tmp/health-golangci.bak` (my scratch) and old `/tmp/bump-dep-*.log` files.
50. Re-check AGENTS gotcha 23's "28 vibe" against the actual suite after the concurrent stage landed.

## g) Questions I can NOT figure out myself (max 3)

~~1. **Concurrent-session authority over "my" numbers and files.** A concurrent session edited the same tree for the whole session (it did T06 erraudit, T22(i), added an `erraudit-inventory` gate/plan/reports, and struck (i) inside _my_ TODO line). When our claims collide (e.g., the stage count, or whether T22(i) is "mine to close"), **whose framing wins?** I default to "merge on evidence, never revert theirs", but I need to know if you want the two sessions' reports reconciled now, or left as independent snapshots.~~ routed — ROADMAP OQ18 (push/sync policy under concurrent sessions) already carries the owner question
~~2. **The feedback-inbox convention (blocking T18's second half).** Do you want (a) an epoch-exempt checker that accepts any top-of-file resolution marker (`PROCESSED`/`RESOLVED`/`ANNOTATED`) and permits the 7 legacy files, **plus** a one-time `git mv` of `docs/feedback/sec-consumer-feedback.md` into `new/` (making `new/` non-empty until processed), or (b) a stricter rule that first normalizes all legacy markers — and if (b), may I write dispositions for those 7 old files without your per-item review?~~ routed — TODO D7 (epoch 2026-08-01 + SEC move ratified in the packet; checker builds after the tick)
~~3. **Heavy-gate policy under sustained fleet load.** Load sat at 20–51 for the entire session, so I refused every heavy gate (per the runbook's "load > 20 = noise" rule). If you want composite proof on my config/docs/script-only change despite the load, say so and I'll run it and annotate load-suspect results — otherwise I'll continue defaulting to "offline-green only, composite owed at the next quiet window". Which do you prefer?~~ resolved in practice + recorded as the round-15 e2 recommendation (correctness gates load-insensitive, measurement gates not); the runbook line is still owed (TODO)

---

_Point-in-time snapshot — 2026-10-01 17:36 CEST. Living state lives in `TODO_LIST.md` / `CHANGELOG.md` / `AGENTS.md` / `FEATURES.md` / `ROADMAP.md`. Later sessions should ANNOTATE, never rewrite, this file. §f is the HARVEST input. WAITING FOR INSTRUCTIONS._
