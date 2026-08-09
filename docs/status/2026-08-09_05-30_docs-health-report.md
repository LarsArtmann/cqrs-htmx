# Docs Health Report — 2026-08-09

**Mode:** AUDIT (BUILD + HARVEST + VERIFY + ANNOTATE)
**First audit with this skill — no baseline.**

---

## Scores

| Dimension | Score | Summary |
| --------- | ----- | ------- |
| **Accuracy** | 9.5/10 | All split brains fixed; claims verified against code and gate output. Minor: coverage percentages are ephemeral. |
| **Fitness** | 9.0/10 | All must-have docs present. No structural decay. TODO_LIST 100% actionable. AGENTS.md above size flag (pre-existing). |

---

## What was stale and fixed

### Critical — split brains across living docs

| Finding | Files affected | Fix applied |
| ------- | -------------- | ----------- |
| Datastar test count wrong (43/71 claimed, actual 54) | AGENTS.md, ROADMAP.md, FEATURES.md | Updated to 54 |
| Datastar coverage wrong (84.6%/96.7% claimed, actual 97.4%) | AGENTS.md, ROADMAP.md, FEATURES.md, TODO_LIST.md | Updated to 97.4% |
| Module count wrong (20 claimed, actual 21) | AGENTS.md, ROADMAP.md, TODO_LIST.md | Updated to 21 |
| "19 modules lint-clean" (ROADMAP) — should be "11 lint-checked modules" | ROADMAP.md | Fixed |
| "All inter-module version refs resolved" (ROADMAP) — false, cross-module drift remains | ROADMAP.md | Changed to "Cross-module dep version drift remains (see TODO_LIST)" |
| FEATURES Metrics table stale test counts (root ~160/~133, dashboardui ~50/~153, datastar ~29/~54) | FEATURES.md | Updated to actual `func Test` counts |
| Datastar test files wrong (8 claimed, actual 5) | FEATURES.md | Updated to 5 |
| go-sse version v0.3.0 (actual v0.4.0) | ROADMAP.md | Updated |
| Examples count 8 (actual 9 — samber-do-demo added) | ROADMAP.md, CHANGELOG.md | Updated to 9 |
| ROADMAP module list missing samber-do-demo | ROADMAP.md | Added |
| Coverage values drifted from gate output (dashboardui 84.0%/83.8%, root 93.3%/93.5%) | ROADMAP.md, FEATURES.md, TODO_LIST.md | Aligned with `nix run .#coverage-gate` |

### Medium — annotation placement violations

| Finding | Files affected | Fix applied |
| ------- | -------------- | ----------- |
| 6 archived reports had banners between H1 and body (anti-pattern per skill) | 6 Aug 7-9 reports in `docs/status/archived/` | Moved banners to end-of-file `## Resolution` appendices |

### Structural — archive backlog

| Finding | Count | Fix applied |
| ------- | ----- | ----------- |
| Pre-August status reports cluttering `docs/status/` | 157 files | Archived to `docs/status/archived/` via `git mv` |
| Old planning docs cluttering `docs/planning/` | 44 files | Archived to `docs/planning/archived/` via `git mv` |

---

## What was already fresh

- **TODO_LIST.md:** No `[x]` items, no "Previously Completed" section, no TODO↔CHANGELOG duplication, no TODO↔ROADMAP duplication. All 6 open items verified as genuinely open against code.
- **FEATURES.md statuses:** 10/10 spot-checked claims verified against actual code (`New()`, `Command()`, `HealthHandler()`, `DecodeJSON[T]`, `RenderTempl()`, `HTMXMiddleware`, `Authorize()`, `RecommendedSecurityMiddleware()`, `ScriptHandler()`, WS symbols removed).
- **AGENTS.md:** No temporal pollution (0 matches), no commit hashes, no content misplacement.
- **All 7 must-have docs exist:** README.md, AGENTS.md, FEATURES.md, TODO_LIST.md, ROADMAP.md, CHANGELOG.md, docs/DOMAIN_LANGUAGE.md.
- **Cross-file consistency:** Version numbers, lint claims, coverage values now consistent across all living docs.
- **HTML/D2/SVG reference docs** (10 files in docs/research/, docs/reviews/, docs/modularization/, docs/architecture-understanding/): These are point-in-time analysis documents, NOT status reports. They contain stale metadata (v4.6.1, 19 modules) but this is expected — they're historical analyses. Most recommendations have been resolved: re-export retirement (ADR-0047), Casbin decision (ADR-0044), WS removal (ADR-0046), SecurityHeaders split-brain, httpspec compliance, leveraging-httputil.md guide, NewServer migration.

---

## What could not be verified

- **Historical CHANGELOG entries** (pre-v4.7.0): Point-in-time values (test counts, coverage percentages) are correct as historical record. Not modified.
- **Coverage percentages in headers/metrics tables:** Verified against `nix run .#coverage-gate` at time of audit. These are inherently ephemeral — they will drift on the next commit. The docs already include "recompute via `nix run .#coverage-gate`" notes.

---

## Verification gates (all canonical nix gates)

| Gate | Result |
| ---- | ------ |
| `nix run .#lint` | 0 issues / 11 modules |
| `nix run .#coverage-gate` | All 11 gates PASSED |
| `nix run .#test` | All 14 suites pass |
| `nix run .#check-codegen` | PASSED |
| `nix run .#check-cqrs-lint` | All modules pass strict |
| `bash scripts/check-docs-links.sh` | 181 links, all resolve |
| `nix fmt` | 0 files changed (already formatted) |

---

## Remaining concerns (not blocking)

1. **AGENTS.md is 46.5 KB** (verify-checklist flags >30 KB). Pre-existing accumulated context — not structural decay. A future session could split gotchas into a separate reference file.
2. **48 unannotated archived August reports:** Per the docs-health skill, "Leave it alone" is always valid for old reports. The harvest guide says "Reading every historical report" is an anti-pattern — "recent reports carry the signal; old ones carry noise." The 6 most recent (Aug 7-9) were annotated; the rest are left as historical snapshots.
3. **Coverage percentages are ephemeral:** Every living doc header includes them. The common-mistakes guide recommends deleting them from AGENTS.md. A future cleanup could replace hardcoded values with a pointer to `nix run .#coverage-gate`.
