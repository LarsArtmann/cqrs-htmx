# Pareto Execution Plan — cqrs-lint Silent-Failure Fix & Build Unblock

**Created:** 2026-07-17 09:31
**Source:** `docs/status/2026-07-17_09-24_cqrs-lint-silent-failure-feedback-session.md` (50 TODOs)
**Rule:** Every task ≤ 12 min. ALL 50 source TODOs covered (mapping in last column).

---

## Pareto Breakdown — Where the leverage is

| Tier          | Promise       | What's in it                                                                 | Why                                            |
| ------------- | ------------- | ---------------------------------------------------------------------------- | ---------------------------------------------- |
| **1% → 51%**  | Unblock       | Local `replace` directives + build green + AGENTS.md gotcha                  | Nothing else can happen while build is red     |
| **4% → 64%**  | Ship the fix  | All 5 cqrs-lint loader fixes + tests + rebuild + verify on cqrs-htmx         | This is the entire point of the session        |
| **20% → 80%** | Root cause    | Upstream go-cqrs-lite publish fix + remove replace + tidy + test suite green | Stops the bug recurring for every consumer     |
| **Next 20%**  | Harden        | Integration tests for the fixes + `--strict` + loader stats + message split  | Prevents regression; makes CI gate trustworthy |
| **Long tail** | Polish & docs | Memory, ADR, changelog, CI matrix, feedback-doc reconciliation               | Process debt + discoverability                 |

---

## TASK TABLE (sorted by impact/effort/customer-value within tier)

### 🔴 TIER 1 — 1% → 51% (Critical path: build must go green FIRST)

| ID   | Task                                                                         | Impact | Effort | Cust-value | Maps→ |
| ---- | ---------------------------------------------------------------------------- | ------ | ------ | ---------- | ----- |
| T1.1 | Identify all cqrs-htmx modules needing `replace` (grep zero pseudo-versions) | H      | 10     | H          | #9    |
| T1.2 | Write `replace` directives to root `go.mod` → `../go-cqrs-lite/*`            | H      | 8      | H          | #9    |
| T1.3 | Write `replace` directives to `usermgmt/go.mod`                              | H      | 8      | H          | #9    |
| T1.4 | Write `replace` directives to remaining affected submodules                  | H      | 10     | H          | #9    |
| T1.5 | Run `go build ./...` at root, fix resolver errors until exit 0               | H      | 12     | H          | #10   |
| T1.6 | Run `go build ./...` in each submodule, fix until exit 0                     | H      | 12     | H          | #10   |
| T1.7 | Add cqrs-lint silent-failure gotcha to cqrs-htmx `AGENTS.md`                 | H      | 8      | M          | #12   |

### 🟠 TIER 2 — 4% → 64% (Ship the 5 cqrs-lint fixes)

| ID    | Task                                                                                                  | Impact | Effort | Cust-value | Maps→  |
| ----- | ----------------------------------------------------------------------------------------------------- | ------ | ------ | ---------- | ------ |
| T2.1  | Add `PackageLoadError` type + `LoadErrors []PackageLoadError` field to `AnalysisContext` (`types.go`) | H      | 10     | H          | #1     |
| T2.2  | Collect top-level `loadFromDir` errors into `LoadErrors` (`loader.go` module loop)                    | H      | 10     | H          | #1     |
| T2.3  | Collect per-package `pkg.Errors` into `LoadErrors` (`loader.go` package loop)                         | H      | 8      | H          | #1     |
| T2.4  | Rewrite empty-`GoFiles` path in `main.go`: surface `LoadErrors` + return `errFindingsWithErrors`      | H      | 10     | H          | #2     |
| T2.5  | Add `LoadErrors` warning block to `doctor.go`; non-zero if all modules failed                         | H      | 8      | H          | #3,#49 |
| T2.6  | Filter errored packages out of `DetectFeatures` Pass 1 (`feature_detect.go`)                          | H      | 10     | H          | #4     |
| T2.7  | Add loader diagnostics (modules found/loaded/skipped + why) to `--verbose`                            | M      | 12     | M          | #5,#30 |
| T2.8  | Run cqrs-lint test suite, capture failures                                                            | H      | 12     | H          | #6     |
| T2.9  | Fix any regressions from loader changes                                                               | H      | 12     | H          | #6     |
| T2.10 | `go install` cqrs-lint from fixed source (binary = 0.2.1+fix)                                         | H      | 8      | H          | #7     |
| T2.11 | Run fixed cqrs-lint on cqrs-htmx, confirm it names the real error                                     | H      | 8      | H          | #8     |

### 🟡 TIER 3 — 20% → 80% (Upstream root cause + verify green)

| ID    | Task                                                                             | Impact | Effort | Cust-value | Maps→   |
| ----- | -------------------------------------------------------------------------------- | ------ | ------ | ---------- | ------- |
| T3.1  | Audit `command/go.mod` (and all sibling go.mod) for zero pseudo-version requires | H      | 8      | H          | #18,#21 |
| T3.2  | Fix `command/go.mod` dispatcher require to a real version; verify build          | H      | 8      | H          | #18     |
| T3.3  | Tag `command/v4 v4.0.1` (⚠ needs publish rights — Q3)                            | H      | 5      | H          | #19     |
| T3.4  | Verify `dispatcher/v4 v4.0.0` exists as real tag; tag if missing                 | H      | 8      | H          | #20     |
| T3.5  | Script: list all go-cqrs-lite module versions, flag any untagged/zero            | H      | 12     | H          | #21     |
| T3.6  | Remove local `replace` directives from all cqrs-htmx go.mod                      | H      | 8      | H          | #22     |
| T3.7  | Bump cqrs-htmx deps to real upstream versions                                    | H      | 10     | H          | #22     |
| T3.8  | `go mod tidy` root                                                               | M      | 5      | M          | #23     |
| T3.9  | `go mod tidy` each of 11 submodules                                              | M      | 12     | M          | #23     |
| T3.10 | `nix run .#test` full cqrs-htmx suite, capture failures                          | H      | 12     | H          | #31     |
| T3.11 | Fix failing cqrs-htmx tests                                                      | H      | 12     | H          | #31     |
| T3.12 | Re-run cqrs-lint on green cqrs-htmx, capture real finding set                    | M      | 8      | M          | #11,#32 |

### 🟢 TIER 4 — Hardening (tests + flags so it can't regress)

| ID   | Task                                                             | Impact | Effort | Cust-value | Maps→ |
| ---- | ---------------------------------------------------------------- | ------ | ------ | ---------- | ----- |
| T4.1 | Write test fixture: project with deliberately broken go.mod      | H      | 12     | H          | #24   |
| T4.2 | Assert: cqrs-lint on broken project exits non-zero + names error | H      | 8      | H          | #24   |
| T4.3 | Test: `doctor` on broken project warns, not confident profile    | H      | 10     | H          | #25   |
| T4.4 | Test: `lint` and `doctor` agree on analyzable package set        | H      | 10     | H          | #26   |
| T4.5 | Audit every `continue` in `loader.go`, document skip paths       | M      | 10     | M          | #27   |
| T4.6 | Wire `--strict` flag in `AppConfig`                              | M      | 8      | M          | #28   |
| T4.7 | Implement `--strict`: treat any `LoadErrors` as fatal            | M      | 10     | M          | #28   |
| T4.8 | Split "no imports" vs "all filtered out" messages                | M      | 8      | M          | #29   |
| T4.9 | Loader statistics table in `--verbose` (counts per module)       | L      | 12     | L          | #30   |

### 🔵 TIER 5 — cqrs-htmx follow-up & triage

| ID   | Task                                                     | Impact | Effort | Cust-value | Maps→ |
| ---- | -------------------------------------------------------- | ------ | ------ | ---------- | ----- |
| T5.1 | Triage each real cqrs-lint finding (accept/suppress/fix) | M      | 12     | M          | #33   |
| T5.2 | grep all 12 submodule go.mod for zero pseudo-versions    | M      | 8      | M          | #34   |
| T5.3 | Diff all submodule go.mod cqrs-lite version consistency  | M      | 10     | M          | #35   |

### 🟣 TIER 6 — Memory & docs

| ID    | Task                                                                           | Impact | Effort | Cust-value | Maps→ |
| ----- | ------------------------------------------------------------------------------ | ------ | ------ | ---------- | ----- |
| T6.1  | Add build-state note to AGENTS.md (only if still broken)                       | M      | 5      | M          | #13   |
| T6.2  | `git diff` cqrs-lint 0.2.0 vs 0.2.1, verify/correct parity claim               | M      | 10     | M          | #14   |
| T6.3  | Search go-cqrs-lite issues/commits for prior silent-failure report             | M      | 8      | M          | #15   |
| T6.4  | Annotate feedback doc / open issue upstream                                    | M      | 10     | M          | #16   |
| T6.5  | Append "Fixed in commit X" to feedback doc                                     | L      | 5      | L          | #17   |
| T6.6  | Add "ship fixes not feedback" lesson to working memory                         | M      | 5      | L          | #36   |
| T6.7  | Write pre-yield checklist (build green? AGENTS.md? fix-or-describe?) to memory | M      | 8      | L          | #37   |
| T6.8  | Write go-cqrs-lite publish-procedure doc (replace-stripping trap)              | M      | 12     | M          | #39   |
| T6.9  | Write pre-publish check script: fail on zero pseudo-version post-strip         | H      | 12     | H          | #40   |
| T6.10 | Add changelog entry for cqrs-lint loader-error-reporting fix                   | L      | 5      | L          | #42   |
| T6.11 | Write ADR: replace directives + the publish pipeline                           | L      | 12     | L          | #43   |
| T6.12 | Write session summary to `TODO_LIST.md`                                        | L      | 8      | L          | #50   |

### ⚪ TIER 7 — Nice-to-haves

| ID   | Task                                                            | Impact | Effort | Cust-value | Maps→ |
| ---- | --------------------------------------------------------------- | ------ | ------ | ---------- | ----- |
| T7.1 | Add cqrs-lint step to cqrs-htmx CI (once non-zero-on-broken)    | M      | 12     | M          | #38   |
| T7.2 | Add `cqrs-lint self-check` subcommand (lint own examples)       | L      | 12     | L          | #41   |
| T7.3 | Generate `.cqrs-lint.json` for cqrs-htmx via doctor, commit     | L      | 8      | L          | #44   |
| T7.4 | Verify `pgregory.net/rapid` indirect dep is expected            | L      | 8      | L          | #45   |
| T7.5 | Scan 6 existing feedback docs for silent-failure overlap        | L      | 12     | L          | #46   |
| T7.6 | Read bank-sync feedback for same class of issue                 | L      | 8      | L          | #47   |
| T7.7 | Add known-consumers CI matrix (cqrs-htmx/bank-sync/DiscordSync) | L      | 12     | L          | #48   |

---

## Coverage check — all 50 source TODOs mapped

Every original TODO (#1–#50) appears in the **Maps→** column. Two merges:

- `#30` (loader stats) folded into `T2.7` (verbose diagnostics).
- `#49` (doctor non-zero all-failed) folded into `T2.5`.
- `#32` (run cqrs-lint for real) folded into `T3.12`.
- `#11` (re-run cqrs-lint) folded into `T3.12`.

**Total: 57 tasks. Max effort 12 min/task. Sum ≈ 515 min (~8.6h) of heads-down work.**

---

## Critical path (must be sequential)

```
T1.1→T1.2→T1.5 (build green)
   → T2.1→T2.2→T2.3→T2.4 (fix ships)
   → T2.8→T2.10→T2.11 (verify)
   → T3.3 (publish, needs Q3) → T3.6→T3.7 (remove replace) → T3.10 (tests green)
```

Everything in Tiers 4–7 parallelizes after its tier-3 dependency lands.

---

## Open questions blocking execution (from status report)

1. **Fix vs feedback division of labor** — implement fixes in go-cqrs-lite directly?
2. **Local `replace` hack acceptable** in cqrs-htmx go.mod (dev-only)?
3. **Publish rights** on go-cqrs-lite for tagging `command/v4 v4.0.1` / `dispatcher/v4 v4.0.0`?
