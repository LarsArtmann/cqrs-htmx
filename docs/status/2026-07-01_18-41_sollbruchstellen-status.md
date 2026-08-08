# Status Report — cqrs-htmx Sollbruchstellen Session

> **Date:** 2026-07-01 18:41
> **Session Goal:** Unix-style modularization review focused on finding clean extraction seams (Sollbruchstellen), not just documenting what's broken
> **Branch:** `master` (commit `315a3fd`)
> **Build:** All 3 production modules build clean
> **Tests:** All 3 production modules pass race tests
> **Coverage:** Root 94.3%, usermgmt 79.4%

---

## a) FULLY DONE

### v3 Sollbruchstellen Proposal (`2026-07-01_SOLLBRUCHSTELLEN.html`)

| Deliverable                                                                   | Status                                                      |
| ----------------------------------------------------------------------------- | ----------------------------------------------------------- |
| Cycle debunked (5 string constants, not structural)                           | ✅ Documented with file:line evidence                       |
| 7 root module clean extraction candidates identified (16 files, 0 cross-refs) | ✅ Each with file list, line count, consumer-value analysis |
| 3 usermgmt deep seams mapped (domain/auth/SQL)                                | ✅ With aggregate-level file lists                          |
| 10 Sollbruchstellen priority matrix                                           | ✅ Ranked by impact/effort/customer-value                   |
| Interface designs for auth strategies (TOTP/WebAuthn/OAuth2)                  | ✅ Concrete Go code, not vague "extract it"                 |
| 4 D2 architecture diagrams (root current/target, usermgmt current/target)     | ✅ Rendered to SVG, embedded in proposal                    |
| v2 proposal marked superseded                                                 | ✅                                                          |

### Code Changes Shipped

| Change                                | Impact                                         | Status                                                 |
| ------------------------------------- | ---------------------------------------------- | ------------------------------------------------------ |
| `constants.go`                        | Breaks the "indivisible cycle" with 6 lines    | ✅ Committed (315a3fd)                                 |
| `scripts/check-module-isolation.sh`   | GOWORK=off build+vet per module                | ✅ All 4 modules pass                                  |
| `scripts/check-dep-budgets.sh`        | Per-module max production deps                 | ✅ All within budget (root:18, usermgmt:28, adminui:7) |
| `scripts/check-version-drift.sh`      | Detects sibling version mismatches             | ✅ Found 3 pre-existing drifts                         |
| `scripts/check-replace-directives.sh` | No absolute paths                              | ✅ All clean                                           |
| `nix run .#check-modules`             | One-command architecture audit                 | ✅ Wired in flake.nix                                  |
| `usermgmt/auth_interfaces.go`         | TOTPVerifier, WebAuthnProvider, OAuth2Provider | ✅ Compiles, non-breaking                              |
| Planning doc with Pareto breakdown    | 65-task detailed plan, mermaid graph           | ✅ At `docs/planning/`                                 |

### Anti-Verschlimmbesserung Decisions

| Rejected Action                                               | Why                                                                                            |
| ------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| Root module sub-package extraction (SSE, WS, ratelimit, etc.) | Same go.mod = same dep tree. 30+ re-export wrappers = zero consumer benefit. Pure boilerplate. |
| usermgmt domain layer sub-package extraction                  | Same go.mod problem. Domain types deeply referenced by Service — moving creates import cycles. |
| TOTP proof-of-concept extraction                              | `ServiceConfig.TOTPConfig` is public — changing it is v4-breaking. Not a POC.                  |

---

## b) PARTIALLY DONE

### usermgmt Auth Strategy Extraction (v4 prep)

- **What's done:** Interfaces designed and compiling (`TOTPVerifier`, `WebAuthnProvider`, `OAuth2Provider`)
- **What's NOT done:** Actual extraction of implementations behind these interfaces. Requires changing public `ServiceConfig` fields from concrete types to interface types — that's v4-breaking.
- **Blocker:** v3 API stability commitment. `ServiceConfig.TOTPConfig`, `ServiceConfig.WebAuthnConfig`, `ServiceConfig.OAuth2Providers` are all public fields used by consumers.

### Version Drift

- **What's done:** CI script detects drift, reports exact modules and versions
- **What's NOT done:** Fixing the 3 drifts it found:
  - `go-cqrs-lite/snapshot/v3`: v3.3.0 vs v3.4.0
  - `go-cqrs-lite/schema/v3`: v3.1.0 vs v3.3.0
  - `go-cqrs-lite/storage/memory/v3`: v3.1.0 vs v3.4.0

---

## c) NOT STARTED

### v4 Breaking Changes (deferred)

1. Extract `usermgmt/totp` behind `TOTPVerifier` interface (1 file, 244 lines, pquerna/otp dep)
2. Extract `usermgmt/webauthn` behind `WebAuthnProvider` interface (3 files, 369 lines, go-webauthn dep)
3. Extract `usermgmt/oauth2` behind `OAuth2Provider` interface (2 files, 608 lines, oauth2+oidc+jose deps)
4. Extract `usermgmt/sql` as separate concern (9 files, ~1,500 lines, modernc/sqlite+pgx deps)
5. Root module → separate Go modules for SSE/WS/ratelimit (only this actually reduces consumer deps)

### Not Started (pre-existing)

6. Phase 2b — Persistent offline queue (IndexedDB, ADR-0030)
7. Root module → usermgmt zero imports invariant (usermgmt currently imports root for RateLimiter)

---

## d) TOTALLY FUCKED UP

### Nothing is fucked up.

All builds pass. All tests pass. No regressions introduced. The one thing that caught us was the BuildFlow pre-commit hook auto-reformatting `adminui/*_templ.go` files — but that's just a formatting churn, not a bug. There are still 3 auto-reformatted templ files in the working tree from the last commit hook.

### Pre-Existing Issues Found (NOT introduced this session)

- **Version drift:** 3 go-cqrs-lite sibling modules referenced at different versions across root and usermgmt go.mod files. This existed before this session. The new CI script surfaces it.
- **Untracked junk files:** `audit-log.html` and `workflow-audit-log.html` appeared in the repo root at some point (BuildFlow artifacts). They're in `.gitignore` scope but still on disk. Cleaned up in a previous commit.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Fix the 3 version drifts** — snapshot/schema/storage-memory are referenced at different versions by root vs usermgmt. Run `go mod tidy` in each module or pin explicitly.
2. **Dep budget headroom is tight** — root module has 16/18 deps (2 slots), adminui has 5/7 (2 slots). Any new dependency requires a budget justification.
3. **usermgmt has 25/28 deps** — the god-package dep bloat is real. v4 auth extraction would remove 5 deps (go-webauthn, pquerna/otp, oauth2, oidc, jose), bringing it to 20/28.

### Process

4. **BuildFlow auto-fixes on commit** — pre-commit hook reformats templ files, creating churn commits. Should run `templ generate` before staging, not during commit.
5. **`nix run .#check-modules` should be in CI** — currently it's a manual command. Should run on every PR.
6. **Version drift script should be advisory, not blocking** — it currently exits 1, which would block CI. Should warn until drift is fixed.

### Documentation

7. **Sollbruchstellen proposal is long** — 213KB HTML. Should have a 1-page TL;DR for quick consumption.
8. **AGENTS.md is getting large** — ~500 lines. Key Decisions section could be split into Architecture/Process/Testing subsections.

---

## f) Top 25 Things to Do Next

| #  | Task                                                                                             | Impact   | Effort   | Type           |
| -- | ------------------------------------------------------------------------------------------------ | -------- | -------- | -------------- |
| 1  | **Fix 3 version drifts** (pin snapshot/schema/storage-memory to same version across modules)     | High     | 30 min   | Hygiene        |
| 2  | **Add `nix run .#check-modules` to `.github/workflows/ci.yml`**                                  | High     | 15 min   | CI             |
| 3  | **Make version-drift script advisory (warn, don't fail)**                                        | Medium   | 10 min   | CI             |
| 4  | **Commit remaining auto-reformatted templ files**                                                | Low      | 2 min    | Hygiene        |
| 5  | **v4: Extract `usermgmt/totp` behind `TOTPVerifier`**                                            | Critical | 4h       | Feature        |
| 6  | **v4: Extract `usermgmt/webauthn` behind `WebAuthnProvider`**                                    | Critical | 8h       | Feature        |
| 7  | **v4: Extract `usermgmt/oauth2` behind `OAuth2Provider`**                                        | Critical | 6h       | Feature        |
| 8  | **v4: Extract `usermgmt/sql` as separate module**                                                | High     | 6h       | Feature        |
| 9  | **Write v3→v4 migration guide** for consumers                                                    | High     | 2h       | Docs           |
| 10 | **Add dep budget documentation** (why each dep exists, what would remove it)                     | Medium   | 1h       | Docs           |
| 11 | **Phase 2b: IndexedDB persistent offline queue** (ADR-0030)                                      | High     | 8h       | Feature        |
| 12 | **Investigate root → separate Go modules** (only real way to reduce consumer deps)               | High     | Research | Architecture   |
| 13 | **Add `check-dep-budgets.sh` --strict mode** (fail on any new dep without justification comment) | Medium   | 30 min   | CI             |
| 14 | **Audit all `replace` directives for necessity** (some may be stale)                             | Medium   | 1h       | Hygiene        |
| 15 | **Write integration test that runs all 4 CI scripts**                                            | Medium   | 1h       | Testing        |
| 16 | **Add coverage gate to CI for adminui** (currently only root + usermgmt)                         | Medium   | 30 min   | Testing        |
| 17 | **Investigate eventtest publishing blocker** (prevents `go mod tidy` under GOWORK=off)           | Medium   | Research | Infrastructure |
| 18 | **Document the auth strategy registration pattern** for v4 consumers                             | Medium   | 1h       | Docs           |
| 19 | **Add architecture decision record for Sollbruchstellen decision**                               | Low      | 30 min   | Docs           |
| 20 | **Profile `check-module-isolation.sh`** (currently ~30s, could parallelize)                      | Low      | 30 min   | CI             |
| 21 | **Add `nix run .#check-modules -- --fix`** mode (auto-pin versions)                              | Low      | 2h       | CI             |
| 22 | **Write consumer-facing guide: "Which deps do I actually pull in?"**                             | Medium   | 1h       | Docs           |
| 23 | **Audit usermgmt for internal test dep leaks** (ginkgo/gomega/rapid in prod go.mod)              | Medium   | 1h       | Hygiene        |
| 24 | **Consider splitting AGENTS.md** (architecture decisions vs operational commands)                | Low      | 30 min   | Docs           |
| 25 | **Add mermaid diagrams to AGENTS.md** for module DAG visualization                               | Low      | 30 min   | Docs           |

---

## g) Top Question I Cannot Answer Myself

**Should cqrs-htmx move to separate Go modules for SSE/WS/ratelimit/security (true v4 module split), or keep the flat single-module design?**

The analysis shows 16 root files with zero logic coupling to core. But sub-package extraction within the same go.mod provides zero consumer benefit (same dependency tree). The ONLY way to let consumers "import SSE without rate limiting" is to split into separate Go modules with separate go.mod files.

**Arguments for splitting:**

- Consumer who only needs HTTP POST handlers saves `golang.org/x/time`, `httputil`, `casbin/v3`, `nosurf` deps
- Matches go-cqrs-lite's 54-module model
- Enables independent versioning of transport vs core

**Arguments against splitting:**

- Consumer UX: `import cqrshtmx` → one import, everything available. Splitting means `import cqrshtmx/sse; import cqrshtmx` — more friction.
- The flat package is the library's contract: "HTMX-aware CQRS HTTP integration" is ONE thing.
- go-cqrs-lite splits because it's a framework with 54 distinct concerns. cqrs-htmx is an integration library with 1 concern.
- Module maintenance overhead: 8+ go.mod files to sync, version drift to manage, replace directives to maintain.

**I cannot decide this because it's a product/UX decision, not a technical one.** The technical analysis is complete. The question is: does Lars want cqrs-htmx to be "one import, batteries included" (like Go's `net/http`) or "compose only what you need" (like go-cqrs-lite)?

---

## Session Metrics

| Metric               | Value                                                                                   |
| -------------------- | --------------------------------------------------------------------------------------- |
| Commits this session | 4 (5ba6acd, e9a9a3b, 315a3fd, + this one)                                               |
| Files created        | 12 (constants.go, auth_interfaces.go, 4 scripts, planning doc, 4 D2+SVG, proposal HTML) |
| Files modified       | 5 (response.go, flake.nix, AGENTS.md, TODO_LIST.md, v2 proposal)                        |
| Build status         | ✅ All pass                                                                             |
| Test status          | ✅ All pass (race)                                                                      |
| Coverage             | Root 94.3%, usermgmt 79.4%                                                              |
| BuildFlow            | ✅ 30/30 checks pass                                                                    |
| CI scripts           | 3/4 pass, 1 finds pre-existing drift                                                    |
