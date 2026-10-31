# Round 45 — samber/do × health closure, decision batch, backlog burn-down (2026-10-31)

**Scope:** the round-45 Pareto plan executed end-to-end: M01–M27 (decision batch, T12 wire rename, 3-family dependency train, failure-mode tests, five new gates, RFC 7807 context, doc-truth pack, TODO truth pass, upstream drafts, delivery prep, parked-row audit, final battery). **Push landed:** `158e86f6..82ddd1b8` (48 commits) — the session-opening push blocker (92 train-lag requires, rc=3) resolved by the M06 family train, pre-push gates green.

## a. What shipped (receipts = commit subjects; this is the index, not the detail)

- **M01/M02** — five owner decisions recorded (defaults per plan §8; `docs/decisions/2026-10-31_round45-m01-decision-batch.md`); T12 wire rename: root + health projection-health codes collapsed into `cqrshtmx.projection.*` / `cqrshtmx.readiness.*` / `cqrshtmx.sse.*` (CHANGELOG receipt; no test asserted old strings).
- **M03/M05/M23/M24** — verified as ALREADY LANDED (S36 seams, S39 tails, S38 a–g, S42 a–c) — the anti-verschlimmbesserung guardrail against re-doing done work; one stale test repaired (adminui unlink: S38e drift; now family-derived, hermetic+workspace green).
- **M06** — the family train: templ-components v1.26.0 (7 prefixes), go-codec v0.3.3, go-health-dashboard v0.12.2; CSS bundles rebuilt (class-set gate green); the one golden drift = the documented amber hue; **train lag 0 → push unblocked**.
- **M07/M08** — demo failure-path test (failed critical → `/readyz` 503 + dashboard verdict); T13 closed won't-fix (dated).
- **M09** — `docs/guides/health-surfaces.md`: the 7-surface wire map (path/auth/schema/vocabulary/drain) + vocabulary mapping + post-T12 code list.
- **M10–M13** — bump-dep lint default-on + collision warnings; check-modules default = report-then-fail (72→78 stages) + parity lock; erraudit strict advisory tooling (TOTAL=303 baseline); RFC 7807 `StructuredError.Context` (5xx-redacted).
- **M14** — TODO truth pass: 5 rows closed against landed receipts.
- **M15–M19** — four NEW gates (readme-badges, pr-secrets + probe-semantics fixtures, dependabot watch advisory, coverage-threshold self-test) + the coverage-gate repair (two latent breaks since 2026-10-27: measure-site context + root-row profile clobber).
- **M20/M21** — five upstream filing drafts (`docs/proposals/upstream/`); release-notes draft; templ chain verified live.
- **M25/M26** — go-error-modernization skill updated (owning repo); fleet sweep clean; 15 parked rows re-audited with dated verdicts (decision record §M26).

## b. What broke and was fixed in-session

- The env-leaks meta-gate caught BOTH new gates (bash-interpreter staging; un-stripped CI) — fixed same-commit (78/78).
- coverage-gate red since 2026-10-27 un-noticed: measure-site context + root-row profile clobber — root-caused, fixed, 15/15 green (commit `06443b3c`).
- Pre-existing malformed flake apps (`check-train-consumers-hermetic`, `train-lag-report` — no `program` attribute) — `nix flake check` green now.
- adminui suite sat red on master since the S38e wave (stale 503 expectation) — repaired.

## c. What remains (honest)

- **doanalyzerv2 gate** stays owner-gated (branching-flow ships no installable distribution).
- **Erraudit strict promotion** needs its green advisory month (TOTAL=303 baseline; TODO watch row).
- **M22 browser legs** are CI-gated: local NixOS has no runnable Chromium (documented `E2E_BROWSER_PATH` escape); specs exist (76 Playwright tests listed, browser launch crashes locally).
- **Skill evals** (row 73 remainder) stay owner-scale; the quarterly ritual row (70) stands.
- **Upstream filings** are DRAFTED, not filed (verify-before-filing; M01 verdict 4 authorizes the samber/do one).
- The 303-finding strict bill is now VISIBLE but untriaged by design (advisory month first).

## d. Verification battery (all green at push time)

check-modules 78/78 · `.#test` rc=0 · coverage-gate 15/15 · `.#lint` rc=0 · erraudit-inventory --gate TOTAL=0 · links 933 · corpus 36088 · todo-hygiene 74/0 · tail 3/3 · release-train strict lag 0 · pre-push battery green (push `82ddd1b8`).
