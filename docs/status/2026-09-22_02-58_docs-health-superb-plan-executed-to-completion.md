# Status Report — Docs-Health SUPERB Plan Executed to Completion (session 2026-09-22 ~01:20 → 03:0x CEST)

**Date:** 2026-09-22
**Session scope:** resume the paused docs-health run (its §g Q1–Q3), then — per explicit user instruction — execute the ENTIRE superb-plan todo list (`docs/planning/archived/2026-09-20_15-43_docs-health-completion-and-backlog-superb-plan.md`) through Z, deconflicted against the round-6 sibling session.
**Governing rules honored:** verify before strike; atomic per-gate changes; no repo-wide `--fix`; no destructive git ops; blob purges prep-only; nothing pushed except through existing daemons/hooks.

---

## a) FULLY DONE (verified this session)

### The three questions, answered by execution

1. **Q1 (hook toolchain) — FIXED AND PROVEN.** `scripts/lib/go-cache-env.sh` (already sourced by the hook) now raises `GOTOOLCHAIN` to the go.work floor when the ambient toolchain is older — never downgrades, no network (module-cache-resolved). Four behaviors verified (raise / keep-explicit / never-downgrade / outside-repo no-op) + shellcheck clean + a T9 case in the hook self-test (12/12). Proven live: a bare-shell commit ran the FULL hook ladder (large-file ✓, release-train 0/0/0 ✓, buildflow ✓). Buildflow's own remaining step failures were diagnosed (no longer an assumption): deterministic environment-class — `tsc` without tsconfig, `go-licenses` missing outside devShell, govulncheck pattern errors, samber-linter 90% historical failure rate. Recorded in AGENTS gotcha 8.
2. **Q2 (python3 ownership) — DECIDED:** ambient `python3` is the contract (CI runners ship it; the self-test already used it); flake apps additionally pin `pkgs.python3`; README documents both.
3. **Q3 (priority) — (a) then everything:** wiring finished first, then A7/B1/B4/A8/A9/A10/B2/B3, then the full C/D tail, then Z.

### Gate wiring completed (the half-wired state closed)

- Gate 2 (`scripts/check-status-rows.py`) + its self-test added to `check-modules` (stages array + sequential path + hermetic `pkgs.python3` runtimeInput) and to CI beside Gate 1's steps. Both apps run for real: 413 files, 19 mixed tables.
- Self-test strengthened to assert exact PARTIAL counts, exact file:line, and per-row lines (17 cases; it immediately caught one wrong assumption of mine about the table-header line semantics).
- **Scope identity proven:** my 19 mixed tables = the skill asset's 19 INCOMPLETE tables, per file AND per line (regex-parsed diff, EQUAL).
- `docs/status/README.md` Gate 2 section rewritten: repo-owned command is the gate of record; skill asset = annotator's authoring aid.

### The docs loop (A7, B1, B4, A8, A9, A10, B2, B3)

- **A7:** AUDIT health report printed inline (skill format): **Accuracy 8.0/10, Fitness 7.75/10**, per-doc findings table, visible math, first-scored-audit honesty. Every finding had an owning phase later this session.
- **B1:** CHANGELOG `[Unreleased]` filled (gates + hook fix + README truth).
- **B4:** TODO_LIST purged of done-work-in-open-items (theme residual, /sse posture, usermgmt replace — all shipped in v4.12.0) and re-harvested (strict-lag split brain, post-train consumer smoke, dep-budget/VCS-cache self-tests, cqrs-lint snapshot, X-Client-Id sweep, requestContextEnricher + system.New asks folded); ROADMAP updated to v4.12.0 + ADR-0051 cross-link + OQ15 (GitHub Releases posture) + OQ16 (bench gate automate-or-retire).
- **A8:** 11 executed planning docs annotated + archived, plus 16 pre-September d2/svg/html artifacts; the one real inbound markdown link repaired (5 dupes in an archived report); relative links inside moved files fixed; gated-work-index refreshed (buildcache DORMANT — mount healthy since 09-15; upstream drafts #25–#28 filed); planning README updated.
- **A9:** DOMAIN_LANGUAGE verified against code: Actor/ActorID now list all 5 kinds (ADR-0111), SQLEventStore + MySQL, Session fields (ActorID/Origin/timestamps), UserID context; Broadcaster + Scoped Feed terms added; 21 events / 20 commands / role constants confirmed.
- **A10:** **AGENTS.md split: 123.5 KB → 28.1 KB** enduring core + `docs/agents-notes.md` (verbatim archive, zero knowledge loss). Every retained path mechanically verified to resolve (24/24). Pollution inventory: 63 dated/qualifier hits in the old file.
- **B2:** `/v4`-suffix import-path rule shipped as `scripts/lib/docs-import-paths.sh`, wired into `check-docs-freshness.sh`, with a 5-case fixture self-test (CI + check-modules wired). Adjudicated exemptions: nested-module shape (`usermgmt/webauthn/v4`), GitHub web URLs (`/commits/`), workspace-only modules, migration docs.
- **B3:** FEATURES re-audit: all 195 green rows' path refs verified mechanically (1 stale cell fixed: `server_timing.go` → re-export); v4.12.0 surface added (theme-toggle row, SSEFilter in setup row, v1.19.0 pins, Trigger-slot identity menu corrected); header → v4.12.0.

### The C/D tail (deconflicted)

- **C1:** bench-spike refused under load (26.9–31.2 vs limit 8) — 9th documented refusal; automate-or-retire routed to ROADMAP OQ16.
- **C10:** the dangling "ADR-001 (appkit)" ghost reference resolved by writing **ADR-0052** (appkit adoption posture: opt-in stands, default flip deferred to v5, (b)–(f) verdicts with rationale); TODO item now points at it.
- **D1:** asks recorded where the owner acts — templ-components TODO items 230–232 (ListNote semantics, Grid hybrid children, CopyButton span color) + go-cqrs-lite TODO section (requestContextEnricher, system.New checkpoint/DLQ, Explain Volume display — verified absent empirically). Both repos are Lars-owned; no external issues needed (stack/v4 precedent).
- **D2:** tag-message guard (`scripts/lib/tag-message-guard.sh`, wired into verify-tag post-creation; 3 new fixture cases, 15/15), MD024 `siblings_only` in `.markdownlint.json`, and a real bug fixed: coverage-gate's shared `/tmp/cov` profile raced across concurrent sessions → per-run mktemp. LICENSE-presence + flake-check-with-builds routed as policy TODOs.
- **D3:** `docs/runbooks/dependency-train-bump.md` (wave order, verification discipline, failure-class table, **never repo-wide `go work sync`** policy) + `scripts/bump-dep.sh` (digit-safe matching, hermetic tidy+build+vet per module, absence assertion, dirty-tree refusal — the refusal itself verified live).
- **D4:** both purge plans re-verified with fresh census data (v4: 3 blobs = 27.56 MiB at `339ce82b`; setup-demo range: FOUR revisions ≈ 105 MiB). Prep-only; nothing pushed.
- **D5:** all four named example suites green (basic, datastar-demo, catalog-demo, samber-do) + new `TestSmoke_AuditViewerServes` (the auditlog SSE viewer serves HTML; first attempt caught the `/audit` → `/audit/` 307). Remaining example gaps (async-startup-demo, middleware-showcase) noted in ROADMAP micro-ideas.
- **D6:** ROADMAP candidates triaged with verdicts: hydrator Option B **SHIPPED** (2026-08-16), SQLite CheckpointStore **SHIPPED**, Option C **REJECTED** (superseded by B), SQL-defaults **REJECTED** (library principle), MetricsRecorder + setup ideas **KEPT** (demand-gated). Micro-ideas list struck for everything this session did.
- **D8:** `systemadapter/declarations_volume_test.go` — per-query Volume magnitudes + declaration-count pin (the first draft pinned MY guesses and failed 10 assertions; corrected to pin the CODE, which is the point). Demo verified: `sys.Explain` shows 12 collections (count confirmed) but no Volume display → routed upstream.

### Z: the final battery (all green, 2026-09-22)

| Gate | Result |
| --- | --- |
| `nix run .#check-modules` | ✓ all 14 stages (incl. both status gates + both self-tests + docs-freshness self-test) |
| `nix run .#coverage-gate` | ✓ 15/15 (the sibling's documented ladder gap, closed) |
| `nix run .#lint` | ✓ 0 issues / 15 modules (the sibling's second gap, closed) |
| `nix flake check --no-build` | ✓ |
| actionlint / shellcheck (new+edited scripts) | ✓ |
| Annotation gate | ✓ 48 gated reports, 413 scanned |
| Row gate | ✓ 413 files, 0 PARTIAL, 19 mixed (baseline held) |
| Link gate / freshness gate | ✓ 281 links / ✓ |

Also this session: the corpus was swept — 8 status reports + the superb plan annotated and archived (unarchived tail: this report only), README counts refreshed, and `/mnt/buildcache` hit 100% mid-battery — the go-cache-env fail-fast caught it exactly as designed; build cache cleaned (17G freed), battery re-run green.

---

## b) PARTIALLY DONE / routed (honest)

1. **Narrative commits largely lost to the daemon again** — three phase commits attempted; the hook now WORKS (buildflow passes; remaining step failures are the diagnosed env-class), but the daemon's poll frequency still beats multi-minute verification tails. Content landed intact via heuristic commits (verified by stat diffs).
2. **D2 remainders routed:** per-module LICENSE presence (licensing policy — owner call) and `nix flake check` WITH builds (needs a nix-capable CI decision).
3. **Example gaps:** async-startup-demo + middleware-showcase still have no `*_test.go` (ROADMAP micro-ideas).
4. **Bench-spike** remains unrun (9 refusals); the gate's future is OQ16.
5. **Sibling's (f)-harvest:** high-value items harvested (see TODO_LIST/ROADMAP); the long tail (mobile-viewport checks, loginpage adoption, GitHub Releases…) stays in their reports/ROADMAP as recorded.

## c) NOT STARTED (deliberately)

- C9 (cqrs-lint Go distribution) and D7 (datastar-demo rebrand) — explicitly unauthorized; drafts/plans stay active in `docs/planning/`.
- Z.3 push — no push performed (not requested; daemon owns the origin state).

---

## d) TOTALLY FUCKED UP (honest)

1. **Two tool-driven near-misses, both caught by my own guards:** (a) an A8 bash function prepended verdicts from the wrong CWD — the `&&` chain prevented the bad mv, leaving only `.tmp` stubs (trashed); the originals were already safely in `archived/` and re-annotated in place. (b) `git mv` with a full-path destination failed loudly (basename fix). Zero data loss, but both were avoidable with `set -e`-style discipline in sweep scripts.
2. **The Volume test pinned my assumptions first** — 10 failing assertions taught (again) that a regression test must pin the CODE's current behavior, not the author's memory of it.
3. **The B2 rule's first draft flagged 10+ false positives** (nested-module paths, GitHub web URLs, workspace-only modules, migration docs) — each adjudicated before shipping; the self-test now pins every exemption.
4. **A subshell-variable bug in the first B2 lib design** (`CHECK_FAILED` lost through command substitution) — the self-test caught it immediately; re-designed to exit-status signaling.
5. **The final battery's first run failed on a full cache disk** — not a code failure (the fail-fast guard worked as designed), but I ran three long gates against a 100% FS before noticing. `df` first, always.

## e) WHAT WE SHOULD IMPROVE

1. **The "atomic gate checklist" is now AGENTS gotcha 19** — this session shipped two gates the complete way; the checklist is written down.
2. Sweep scripts should use `mktemp` workdirs + `set -e` + absolute paths — the A8 stub incident is the template of what not to do.
3. The daemon race remains unsolved at the process level; the hook fix at least makes future narrative commits POSSIBLE.
4. Cache-disk hygiene: check `df /mnt/buildcache` before any long battery (now also implied by go-cache-env's fail-fast, which fired correctly).

## f) NEXT (already routed — TODO_LIST/ROADMAP are the trackers)

Strict-lag split brain + pre-push wiring · post-train consumer-eye smoke · dep-budget + VCS-cache self-tests · cqrs-lint zero-warning snapshot/blocking · X-Client-Id sweep · LICENSE policy · flake-check-with-builds CI · remaining example smokes · OQ15 GitHub Releases · OQ16 bench gate · V007 cluster 1 (metaengine-gated) · upstream asks in the two sibling repos.

---

*Superb plan archived with full verdicts. This report is the sole unarchived status file; archive it when its items resolve.*
