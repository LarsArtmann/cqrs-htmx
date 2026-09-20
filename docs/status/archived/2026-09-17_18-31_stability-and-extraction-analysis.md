# Status Report — cqrs-htmx Stability & Extraction-Readiness Analysis

**Date:** 2026-09-17 18:31 CEST
**Session scope:** Analysis-only session (run from go-hotspot workspace). No production code in cqrs-htmx was modified. Two questions answered: (1) which module is most stable, (2) which parts are ready to be extracted into their own repos.
**Method:** git churn census, complexity×churn hotspot scoring (ran go-hotspot itself against cqrs-htmx), internal import graph, temporal coupling (strict + loose thresholds), external consumer census, live per-module test/coverage verification.

> **ANNOTATED 2026-09-20** (docs-health sweep): analysis-only session; no extraction has been started (awaiting the owner go/no-go + naming policy).
> - **§b:** b7 DONE (`check-release-train` green); b1–b6 remain open/partial.
> - **§c:** c2 DONE (harvested); the extraction pilot + `openapi`/`transport` analyses remain open.
> - **§f:** struck rows confirmed done (release-train, fresh coverage, lint, styles.css policy, tag-cache note, harvest); unmarked rows remain open — the extraction program is decision-gated → `ROADMAP.md` (owner calls g-1/g-2/g-3).
> - **§g:** Q1–Q3 are owner calls (consumer universe, sequencing strategy, naming/versioning policy) — unresolved; the analysis stands as-is.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | Module inventory: 27 real Go modules mapped (36 `go.mod` paths found incl. examples/testdata) | `find -name go.mod` sweep |
| 2 | Per-module git churn census (commits, added+deleted lines, last touch, 90-day window) over all 2,392 commits | bash/git aggregation, 2026-09-17 |
| 3 | Refined metrics: root-only churn (1,342 commits / 147,348 lines via pathspec exclusions), adminui churn excluding regenerated styles.css (83,522 → 42,571), repo age (started 2026-05-03) | git log pathspec runs |
| 4 | LOC census per module (src vs test, generated `_templ.go` excluded): usermgmt 12.5k/19.7k, root 6.5k/15k, dashboardui 5.4k/5.9k, identity-model 2.5k/1.9k, … auditlog 70/104, totp 96/139 | find+wc sweep |
| 5 | go-hotspot built and run against cqrs-htmx: 587 files scored, hotspot CSV + JSON with coupling captured | `/tmp/hotspot-stable.csv`, `/tmp/hotspot-full.json` (rc=0) |
| 6 | Per-module hotspot aggregation (avg/max score, churn) | awk aggregation of CSV |
| 7 | Finalist verification — live runs, all green: auditlog **100% cov**, health **100% cov**, totp **88.2%**, openapi **99.0%**, datastar **97.4%** | `GOWORK=off go test ./... -cover`, rc=0 each, 2026-09-17 |
| 8 | totp verified as a real implementation (RFC 6238 via pquerna/otp, structural-typing plugin), not a stub | read `usermgmt/totp/provider.go` |
| 9 | **Stability verdict delivered: `auditlog` is the most stable module** (15 commits, 816 churn lines, 70 LOC, 100% coverage, zero internal deps, zero dependents) | final answer, turn 1 |
| 10 | go.work + go.mod replace mapping: **every module go.mod has zero local replaces** (all resolve from published tags); only systemadapter carries 4 local replaces (→ go-cqrs-lite sibling) | grep sweep over 13 go.mods + go.work |
| 11 | Internal import graph (non-test files only): 6 modules with zero internal deps (auditlog, datastar, totp, webauthn, oauth2, identity-model); setup imports 10 modules (composition root) | corrected find+grep sweep |
| 12 | Temporal coupling: **71/71 pairs intra-module, zero cross-module** at default thresholds (≥5 shared commits, ≥30% degree); loose run (≥2 shared, ≥10%) yields 209 cross pairs, all confined to the ROOT↔usermgmt↔identity-model core, max degree 29% | go-hotspot JSON couplings, both threshold runs |
| 13 | External consumer census across `~/projects`: root v4 required by ~45 repos, usermgmt 28, identity(-model) 20, dashboardui 12, adminui 10, webauthn 8, loginpage 8, datastar 6, oauth2 5, totp 4, systemadapter 4, health 3, auditlog 3 | go.mod grep sweep |
| 14 | Extraction tier list + migration mechanics delivered: Tier 1 auditlog+health, Tier 2 totp/webauthn/oauth2, Tier 3 datastar, defer UI trio + identity-model, keep root/usermgmt/setup; import-path breaking change identified with FM#5 mitigation (final shim release + consumer sweep) | final answer, turn 2 |

## b) PARTIALLY DONE

| # | Item | Works | Missing | Effort |
|---|------|-------|---------|--------|
| 1 | cqrs-htmx AGENTS.md read | Lines 1–150 (architecture, key patterns, first gotchas) | Lines 150+ (remaining gotchas) never read — some may affect extraction sequencing | S |
| 2 | go-modularize `phases.md` read | Phases 1 + start of 2 (detect-state framework) | Phases 2–7 procedures (proposal format, self-review checklist, execution, reflection) unread — relevant when an actual extraction starts | S |
| 3 | Coverage verification | 5 finalist modules re-verified live today | Remaining ~10 modules' coverage (usermgmt 81.9%, adminui 68.5%, identity-model 75.5%, …) cited from AGENTS.md table dated 2026-09-10 — point-in-time claim, **not** re-verified | M |
| 4 | External consumer counts | Counts correct per module | Display truncation bug: regex char class `[a-z/0-9]*` drops `-`/`_`, so `identity-model` printed as `identity`, `integration_test` as `integration`; 2 bare `cqrs-htmx` requires of unknown provenance | S |
| 5 | Extraction cost model | External consumers counted; Tier list reflects them | In-monorepo consumer cost **not quantified**: setup, integration_test, 12 examples, e2e/server all require the modules — the sed-sweep surface inside the monorepo is unmeasured | S |
| 6 | Finalist verification depth | Plain `go test -cover` green | No `-race -gcflags=all=-l` runs, no fresh `golangci-lint` run on finalists (lint-clean claim rests on AGENTS.md + buildflow hooks) | S |
| ~~7~~ | ~~Release-train state~~ done — check-release-train green | ~~go.mod requires shown as published (zero local replaces)~~ | ~~`check-release-train` gate never actually run — train-lag between extracted module and its new require targets unverified (tag-cache TTL gotcha applies)~~ | ~~S~~ |

## c) NOT STARTED

| # | Item | Why |
|---|------|-----|
| 1 | **Actual extraction pilot (auditlog)** — subtree split, new repo, shim release, consumer sweep | Analysis-only session; awaiting your go/no-go |
| ~~2~~ | ~~HARVEST of this report's section (f) into cqrs-htmx TODO_LIST.md / ROADMAP.md~~ done — harvested 2026-09-20 (this sweep) | ~~Report written first; harvest is the documented next step~~ |
| 3 | Deprecation-shim design ADR (module-path transition policy) | Blocked on question g-3 (naming/versioning policy) |
| 4 | `openapi/` extraction-candidate analysis | Forgotten from the tier table — it is a root sub-package with zero deps, 99% cov, frozen since July; arguably Tier 1 material |
| 5 | `transport/` (root sub-package) extraction analysis | dashboardui + setup both depend on `v4/transport`; never examined as a seam |
| 6 | Daemon-vs-human commit attribution split | Churn counts include auto-commit-daemon noise; a `git log` author/subject filter was not applied |
| 7 | docs/DOMAIN_LANGUAGE.md read | Project-discovery checklist item skipped (not needed for the metric questions asked) |
| 8 | Bus-factor analysis per module | go-hotspot already emits authors/author_names — data was in hand, analysis not run |

## d) TOTALLY FUCKED UP

No data lost, no repo state damaged, no wrong conclusion shipped as final. But radical honesty about what went wrong *during* the session:

| # | What broke | Severity | Root cause | Mitigation |
|---|-----------|----------|------------|------------|
| 1 | **First import graph was wrong**: `grep -v '_test.go'` filtered file *content*, not filenames — test-only imports leaked into the "non-test" graph (e.g. phantom `usermgmt → root` edge list was inflated) | Medium — was caught by self-check before any conclusion used it; a wrong intermediate existed for one turn | Filter applied after `grep -h` (no filenames available to filter on) | Correct pattern: `find -not -name '*_test.go'` first, then grep. Redone same turn; final graph verified clean |
| 2 | **Consumer-census regex truncation**: `[a-z/0-9]*` drops `-` and `_`, mislabeling `identity-model` → `identity` (20), `integration_test` → `integration` (1) | Low — deterministic truncation, no two modules collide into one label, counts stand | Same root cause: sloppy character class | Fix pattern to `[a-z/0-9_-]*`; 2 bare `cqrs-htmx` entries still unexplained |
| 3 | **First go-hotspot invocation rejected** (target dir + flags parsed as one argument) | None — errored loudly, fixed on retry (flags before positional) | Invocation-order assumption | None needed |
| 4 | **Methodology caveats under-communicated in final answers**: (a) avgHS is relative-score averaged with test-file churn dilution — weakest metric shown; (b) the coupling mega-commit guard (commits >30 files excluded) interacts with this repo's auto-commit daemon mega-commits, so "zero cross-module coupling" partly reflects guard behavior on a 4.5-month-old hyperactive repo, not proof of perfect seams; (c) stability ranking on a repo this young is provisional | Medium — conclusions directionally right (corroborated by imports + replaces) but stated more confidently than the method warrants | Final-answer brevity pressure | This report documents the limits; loose-threshold run partially de-risks (b) |

## e) WHAT WE SHOULD IMPROVE

1. **Re-verify doc claims before citing them.** The coverage table and lint-clean claims were quoted from AGENTS.md (2026-09-10) without fresh runs, except finalists. Rule: a number I didn't produce today gets labeled as doc-claim in the answer. Impact: medium — a stale coverage claim could mis-rank an extraction candidate.
2. **Quantify in-repo migration surface before recommending tiers.** External consumers were counted; setup/examples/integration_test/e2e (the in-monorepo sweep) were not. Tier ordering could shift once the internal sed surface is known. Fix: one `grep -rl` per module + count, ~10 min.
3. **Use existing gates instead of inferring.** `check-release-train` and `check-module-isolation` exist precisely to answer questions I answered by grepping go.mods. Running them is both faster and authoritative (modulo the documented tag-cache TTL gotcha — re-run with `--refresh-cache`).
4. **Split daemon noise out of churn metrics.** Per AGENTS.md, heuristic auto-commits dominate recent history (`1,749 commits/90d`). A human-vs-daemon split (`git log --author`/subject filter) would make churn — and thus "stability" — meaningfully more accurate. go-hotspot could gain a `--exclude-author`/`--subject-filter` flag for this.
5. **Caveat methodology in the answer, not just in the working notes.** The avgHS dilution and coupling-guard confounder were visible in my working data but not surfaced. Fix: one "Limitations" block in any analysis answer.
6. **Read skill references fully before partial application.** phases.md Phases 2–7 contain the execution checklists; reading halves of references invites rework when execution starts.
7. **Memory hygiene:** the stability table + extraction tier list are durable project knowledge; they should land in cqrs-htmx AGENTS.md/TODO_LIST via harvest, not just in a timestamped report.

## f) 50 THINGS WE SHOULD GET DONE NEXT

*HARVEST note: items 1–15 are TODO_LIST-grade (bounded, actionable); 16–50 are ROADMAP fuel (larger, decision-dependent). Impact/Critical-High-Medium-Low, Effort/S-M-L, Category.*

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Decide go/no-go on auditlog extraction pilot | High | S | Decision |
| 2 | Run extraction pilot: `git subtree split` auditlog → new repo, new module path | High | M | Feature |
| 3 | Cut final monorepo release with `// Deprecated:` re-export shim at `auditlog/v4` | High | S | Feature |
| 4 | Sweep the 3 consumer repos importing auditlog (sed + `GOWORK=off go build` each) | High | M | Cleanup |
| 5 | Same pilot loop for health (Tier 1 #2) | High | M | Feature |
| ~~6~~ | ~~Run `check-release-train` (+`--refresh-cache`) to map train-lag before sequencing extractions~~ done — check-release-train green (0 unpublished/0 lag, 2026-09-20) | ~~High~~ | ~~S~~ | ~~Quality~~ |
| ~~7~~ | ~~Fresh coverage run for all ~15 modules; replace the 2026-09-10 AGENTS.md table~~ done — fresh coverage run 2026-09-18/20; 15/15 gates green | ~~High~~ | ~~M~~ | ~~Quality~~ |
| 8 | Quantify in-monorepo requires per module (setup, integration_test, 12 examples, e2e/server) | High | S | Quality |
| 9 | Fix consumer-census regex (`[a-z/0-9_-]*`); re-run census | Medium | S | Bug |
| 10 | Chase the 2 bare `github.com/larsartmann/cqrs-htmx` requires — legacy v1-era paths? | Medium | S | Bug |
| 11 | Decide repo + module-path naming for extracted repos (g-3) | High | S | Decision |
| 12 | Write ADR: module-path transition policy (shim sunset timeline, /vN reset or continuity) | High | S | Documentation |
| 13 | Extract totp (Tier 2 pilot — smallest, zero deps) | Medium | M | Feature |
| 14 | Extract webauthn | Medium | M | Feature |
| 15 | Extract oauth2 | Medium | M | Feature |
| 16 | Extract datastar (standalone value beyond CQRS) | High | M | Feature |
| 17 | Analyze `openapi/` as extraction candidate (was forgotten; zero deps, 99% cov, frozen) | Medium | S | Feature |
| 18 | Analyze `transport/` (root sub-package) as a seam — dashboardui+setup depend on it | Medium | S | Feature |
| 19 | Race-test finalists (`-race -gcflags=all=-l`) to extend verification depth | Medium | S | Quality |
| ~~20~~ | ~~Fresh `golangci-lint run` on finalist modules (re-verify 0-issues claim today)~~ done — lint 0 issues / 15 modules | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| 21 | Define identity-model extraction trigger: "domain API freeze" criteria | High | S | Decision |
| 22 | Identity-model extraction ADR (co-change argument for/against, 20 consumers) | High | S | Documentation |
| 23 | UI-trio extraction plan (loginpage → adminui → dashboardui, when UI work stabilizes) | Medium | S | Documentation |
| 24 | Human-vs-daemon commit split for churn truth (filter or go-hotspot flag) | Medium | M | Quality |
| 25 | Add `--exclude-author`/`--subject-regex` to go-hotspot (daemon-noise-aware churn) | Medium | M | Feature |
| 26 | Bus-factor audit per module using existing go-hotspot authors data | Medium | S | Quality |
| 27 | usermgmt god-module risk review (12.5k src LOC, 599 commits) — split deeper? | High | L | Quality |
| 28 | dashboardui/core independence check (916 LOC pure layer — own module worth it?) | Low | S | Feature |
| 29 | Consumer-driven contract tests for extracted seams (shim vs new repo drift) | High | L | Quality |
| 30 | Extraction runbook doc (scripts + checklist, per collector-extraction precedent) | High | M | Documentation |
| 31 | Per-extracted-repo scaffolding: flake.nix (BuildFlow), CI workflow, .golangci.yml, .cqrs-lint.json | Medium | M | Feature |
| 32 | README + CHANGELOG seed per extracted repo | Medium | S | Documentation |
| 33 | Update `check-module-isolation`/`check-replace-directives` gates for cross-repo shape | Medium | M | Quality |
| 34 | Update `check-docs-freshness` gate for extracted-repo awareness | Low | M | Quality |
| 35 | go.work + go.work.sum cleanup after each extraction; `go work sync` | Medium | S | Cleanup |
| 36 | Delete moved dirs from monorepo; verify dead-replace guard passes | Medium | S | Cleanup |
| 37 | Re-run go-hotspot on cqrs-htmx post-extraction; publish delta report | Medium | S | Quality |
| 38 | Periodic stability dashboard (30/60/90-day churn trend per module) | Medium | M | Feature |
| ~~39~~ | ~~adminui styles.css churn-noise policy (build artifact regen vs commit)~~ done — AGENTS.md styles.css orphan-output policy recorded | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| 40 | Investigate usermgmt → root `v4` import usage (which symbols; boundary hygiene note) | Low | S | Quality |
| 41 | Test-only deps audit per module go.mod (FM#3 leak check) | Medium | M | Quality |
| 42 | examples/ import-path sweep post-extraction | Medium | M | Cleanup |
| 43 | e2e/server import-path sweep post-extraction | Medium | S | Cleanup |
| 44 | Decide integration_test module fate in the cross-repo world (bridge tests → which repo?) | Medium | S | Decision |
| 45 | setup/ official "distribution/bundle" role ADR (it already imports everything) | Medium | S | Documentation |
| 46 | Release-train docs for the new family shape (per-repo trains, train-lag semantics) | Medium | S | Documentation |
| 47 | Verify consumer repos post-sweep: full test suites green on new paths | High | M | Quality |
| ~~48~~ | ~~Tag-cache TTL gotcha: add `--refresh-cache` note to release checklist~~ done — tag-cache TTL note in AGENTS.md/runbook | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| 49 | Per-extracted-repo CODEOWNERS + module ownership map | Low | S | Documentation |
| ~~50~~ | ~~HARVEST this report: route items 1–15 → TODO_LIST.md, 16–50 → ROADMAP.md~~ done — harvested 2026-09-20 (this sweep) | ~~High~~ | ~~S~~ | ~~Cleanup~~ |

## g) QUESTIONS I CANNOT ANSWER MYSELF

**Q1 — Consumer universe beyond `~/projects`:** Are there production/external consumers of `auditlog` or `health` outside your local repos (public GitHub dependents, org repos, dependabot-managed projects)? I scanned only `/home/lars/projects`; if public consumers exist, the deprecation-shim sunset window for the pilot must be longer, and the pilot choice might shift. What I tried: local go.mod census (found 3 + 3 consumers), no public registry/dependents scan was in scope.

**Q2 — Sequencing strategy:** Stability-first (auditlog/health: zero risk, near-zero value — they're 70/150-line bridges) or value-first (identity-model/datastar: real standalone value, real coordination cost)? The data supports both; the right order depends on why you're extracting (repo hygiene vs. enabling independent ecosystems vs. shrinking the monorepo). This is a goals call only you can make.

**Q3 — Naming + versioning policy:** For extracted repos, do you want module-path continuity (`github.com/larsartmann/cqrs-htmx/auditlog/v4` impossible post-move → must change) or your standard family pattern (`go-auditlog/v1`, reset to v1)? And repo naming: `go-auditlog`, `cqrs-htmx-auditlog`, or `auditlog-viewer`? This determines the shim design, the consumer sweep diff size, and whether the release train resets.

---

*Point-in-time snapshot. Generated 2026-09-17 18:31 CEST. Section (f) is the input for docs-health HARVEST.*
