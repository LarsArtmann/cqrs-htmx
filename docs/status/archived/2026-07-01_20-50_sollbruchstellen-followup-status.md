# Status Report — cqrs-htmx Sollbruchstellen Follow-Up

> **Date:** 2026-07-01 20:50
> **Previous Report:** `2026-07-01_18-41_sollbruchstellen-status.md`
> **Session Goal:** Self-critique the previous session, fix what was missed, wire CI enforcement
> **Branch:** `master` (commit `b8cc03e`)
> **Working Tree:** Clean (after reverting BuildFlow auto-bump)
> **Build:** All 3 production modules pass
> **Tests:** All 3 production modules pass race tests
> **Coverage:** Root 94.2%, usermgmt 79.4%, adminui 66.8%
> **CI Scripts:** All 4 pass in strict mode (0 drifts, 0 violations)

---

## a) FULLY DONE

### Architecture Enforcement Infrastructure

| Deliverable                                                           | Commit    | Status                     |
| --------------------------------------------------------------------- | --------- | -------------------------- |
| `constants.go` — 6 shared constants extracted, cycle debunked         | `315a3fd` | ✅ Shipped                 |
| `scripts/check-module-isolation.sh` — GOWORK=off build+vet per module | `315a3fd` | ✅ All 4 modules pass      |
| `scripts/check-dep-budgets.sh` — per-module max production deps       | `315a3fd` | ✅ All within budget       |
| `scripts/check-version-drift.sh` — sibling version consistency        | `315a3fd` | ✅ 0 drifts (strict mode)  |
| `scripts/check-replace-directives.sh` — no absolute paths             | `315a3fd` | ✅ All clean               |
| `nix run .#check-modules` — one-command architecture audit            | `315a3fd` | ✅ Wired in flake.nix      |
| `module-architecture` job in `.github/workflows/ci.yml`               | `b8cc03e` | ✅ Runs on every PR/push   |
| Advisory mode for version-drift (`--strict` flag)                     | `b8cc03e` | ✅ CI advisory, nix strict |

### Version Drift Elimination

| Module Pair         | Drift            | Fix                                        | Status   |
| ------------------- | ---------------- | ------------------------------------------ | -------- |
| `snapshot/v3`       | v3.3.0 vs v3.4.0 | Pinned to v3.4.0 across all 8 go.mod files | ✅ Fixed |
| `schema/v3`         | v3.1.0 vs v3.3.0 | Pinned to v3.3.0 across all modules        | ✅ Fixed |
| `storage/memory/v3` | v3.1.0 vs v3.4.0 | Pinned to v3.4.0 in datastar-demo          | ✅ Fixed |
| `listing/v3`        | v3.4.0 vs v3.5.0 | Downgraded root from v3.5.0 auto-bump      | ✅ Fixed |
| `scenario/v3`       | v3.4.0 vs v3.5.0 | Downgraded root from v3.5.0 auto-bump      | ✅ Fixed |

### Auth Strategy v4 Preparation

| Deliverable                                                   | Status                                   |
| ------------------------------------------------------------- | ---------------------------------------- |
| `TOTPVerifier` interface (4 methods)                          | ✅ Matches Service exactly               |
| `WebAuthnProvider` interface (v4 target shape)                | ✅ Documented gap (uses json.RawMessage) |
| `OAuth2Provider` interface (v4 target shape)                  | ✅ Documented gap (uses ExternalAccount) |
| Compile-time assertion `var _ TOTPVerifier = (*Service)(nil)` | ✅ Proves contract                       |
| `usermgmt/auth_interfaces.go`                                 | ✅ Non-breaking, additive                |

### Analysis & Documentation

| Deliverable                                                | Status                       |
| ---------------------------------------------------------- | ---------------------------- |
| v3 Sollbruchstellen proposal (213KB HTML, 4 D2 SVGs)       | ✅ At `docs/modularization/` |
| Planning doc with Pareto breakdown + mermaid graph         | ✅ At `docs/planning/`       |
| v2 proposal marked superseded                              | ✅                           |
| AGENTS.md updated with CI scripts + interfaces + drift fix | ✅                           |
| TODO_LIST.md updated with completed items                  | ✅                           |
| Previous status report                                     | ✅ At `docs/status/`         |

---

## b) PARTIALLY DONE

### usermgmt v4 Auth Strategy Extraction

- **Done:** Interfaces designed, TOTPVerifier compile-time assertion passes, gap analysis documented
- **Not Done:** Actual extraction of TOTP/WebAuthn/OAuth2 implementations behind interfaces
- **Blocker:** `ServiceConfig.TOTPConfig`, `ServiceConfig.WebAuthnConfig`, `ServiceConfig.OAuth2Providers` are public fields — changing them is v4-breaking
- **Next step:** Start v4 major version branch, extract TOTP first (simplest — Service already satisfies the interface)

### BuildFlow Auto-Bump Management

- **Done:** Reverted v3.5.0 bump twice (it has breaking `codec.ForEncoding` removal)
- **Not Done:** No permanent guard against BuildFlow re-bumping on every commit
- **Risk:** Every BuildFlow pre-commit run silently bumps go-cqrs-lite to latest, introducing breaking changes

---

## c) NOT STARTED

### v4 Breaking Changes (deferred to major version)

1. Extract `usermgmt/totp` behind `TOTPVerifier` (Service already implements it!)
2. Extract `usermgmt/webauthn` behind `WebAuthnProvider`
3. Extract `usermgmt/oauth2` behind `OAuth2Provider`
4. Extract `usermgmt/sql` as separate module/concern
5. Evaluate root module → separate Go modules (only real way to reduce consumer deps)

### Features (pre-existing backlog)

6. Phase 2b — IndexedDB persistent offline queue (ADR-0030)
7. adminui coverage gate in CI (currently 66.8%, no gate)
8. eventtest publishing blocker (prevents `go mod tidy` under GOWORK=off)

---

## d) TOTALLY FUCKED UP

### BuildFlow Auto-Bump (recurring nuisance)

**Problem:** BuildFlow pre-commit hook silently auto-bumps go-cqrs-lite from v3.4.0 to v3.5.0 on every commit. v3.5.0 has a breaking change (`codec.ForEncoding` removed from `event/v3`). This was caught and reverted **twice** this session.

**Impact:** If someone commits without checking the diff, they ship a broken build. The pre-commit hook is supposed to help, not sabotage.

**Mitigation applied:** Manual revert after each commit. Not sustainable.

**Needed:** Either (a) pin go-cqrs-lite in `.buildflow.yml` to prevent auto-bump, or (b) upgrade to v3.5.0 properly by fixing the `codec.ForEncoding` usage.

### Nothing Else Is Fucked Up

All builds pass. All tests pass. All CI scripts pass strict mode. No regressions.

---

## e) WHAT WE SHOULD IMPROVE

### Urgent

1. **Stop BuildFlow auto-bumping go-cqrs-lite** — It introduces breaking changes silently. Either pin in config or fix the v3.5.0 breaking change.
2. **adminui coverage is 66.8%** — No CI gate exists. Should add one or improve coverage.
3. **`OPEN` item in TODO_LIST.md** — There's a literal `- [ ] OPEN` that needs to be replaced with actual content or removed.

### Architecture

4. **Start v4 branch** — The TOTP extraction is ready to go. Service already satisfies `TOTPVerifier`. It's the lowest-risk, highest-value v4 change.
5. **Document the v3→v4 migration path** — Consumers need to know what changes and how to adapt.
6. **Dep budget headroom is tight** — root has 16/18, adminui has 5/7. Consider whether budgets are too generous or deps are too many.

### Process

7. **BuildFlow config** — Investigate `.buildflow.yml` to control auto-bump behavior.
8. **Test dep audit** — ginkgo/gomega/rapid appear in production go.mod files. Should be in test-only modules (pattern from go-cqrs-lite).
9. **Integration test for CI scripts** — The scripts themselves should have tests to prevent regressions.

---

## f) Top 25 Things to Do Next

| #  | Task                                                                                              | Impact   | Effort   | Type           |
| -- | ------------------------------------------------------------------------------------------------- | -------- | -------- | -------------- |
| 1  | **Stop BuildFlow auto-bumping go-cqrs-lite** (pin in .buildflow.yml or fix v3.5.0 compat)         | Critical | 30 min   | Infrastructure |
| 2  | **Remove `OPEN` placeholder from TODO_LIST.md**                                                   | Low      | 1 min    | Hygiene        |
| 3  | **Start v4 branch: extract `usermgmt/totp`** behind TOTPVerifier (Service already implements it!) | Critical | 2h       | Feature        |
| 4  | **Add adminui coverage gate to CI** (currently 66.8%, no gate)                                    | High     | 15 min   | CI             |
| 5  | **v4: Extract `usermgmt/webauthn`** behind WebAuthnProvider                                       | Critical | 8h       | Feature        |
| 6  | **v4: Extract `usermgmt/oauth2`** behind OAuth2Provider                                           | Critical | 6h       | Feature        |
| 7  | **Write v3→v4 migration guide** for consumers                                                     | High     | 2h       | Docs           |
| 8  | **Phase 2b: IndexedDB persistent offline queue** (ADR-0030)                                       | High     | 8h       | Feature        |
| 9  | **Audit test deps in production go.mod** (ginkgo/gomega/rapid)                                    | Medium   | 1h       | Hygiene        |
| 10 | **Investigate eventtest publishing blocker**                                                      | Medium   | Research | Infrastructure |
| 11 | **Evaluate root → separate Go modules** for v4 (consumer dep reduction)                           | High     | Research | Architecture   |
| 12 | **Add architecture ADR for Sollbruchstellen decision**                                            | Medium   | 30 min   | Docs           |
| 13 | **Write consumer dep guide: "What deps do I pull in?"**                                           | Medium   | 1h       | Docs           |
| 14 | **Profile and parallelize `check-module-isolation.sh`** (currently ~30s)                          | Low      | 30 min   | CI             |
| 15 | **Add `check-dep-budgets.sh --strict` mode** with justification requirement                       | Low      | 30 min   | CI             |
| 16 | **Consider splitting AGENTS.md** (500+ lines)                                                     | Low      | 30 min   | Docs           |
| 17 | **Add mermaid module DAG to AGENTS.md**                                                           | Low      | 30 min   | Docs           |
| 18 | **Write integration tests for CI scripts**                                                        | Medium   | 1h       | Testing        |
| 19 | **v4: Extract `usermgmt/sql`** as separate concern                                                | High     | 6h       | Feature        |
| 20 | **Audit replace directives for stale entries**                                                    | Medium   | 1h       | Hygiene        |
| 21 | **Upgrade to go-cqrs-lite v3.5.0 properly** (fix codec.ForEncoding usage)                         | Medium   | 2h       | Maintenance    |
| 22 | **Add dep budget documentation** (why each dep exists)                                            | Low      | 1h       | Docs           |
| 23 | **Document auth strategy registration pattern** for v4 consumers                                  | Medium   | 1h       | Docs           |
| 24 | **Consider `nix run .#check-modules -- --fix`** auto-pinning                                      | Low      | 2h       | CI             |
| 25 | **Review adminui test coverage gaps** (66.8% → target 75%+)                                       | Medium   | 3h       | Testing        |

---

## g) Top Question I Cannot Answer Myself

**How do we stop BuildFlow from auto-bumping go-cqrs-lite to v3.5.0 on every commit?**

BuildFlow's pre-commit hook silently updates all `github.com/larsartmann/*` deps to their latest published tag. go-cqrs-lite v3.5.0 introduced a breaking change (`codec.ForEncoding` was removed from `event/v3`), so the auto-bump breaks the build every time.

I've reverted it manually twice, but this is not sustainable — any future commit without manual review will re-introduce the breakage.

**What I've tried:**

- Checked `.buildflow.yml` — it exists but I don't know its schema for pinning deps
- Checked if there's a `go.mod` pin mechanism that BuildFlow respects

**What I need:** Either (a) knowledge of how to configure BuildFlow to stop auto-bumping specific deps, or (b) a decision to upgrade to v3.5.0 and fix the `codec.ForEncoding` usage in our code.

---

## Session Metrics

| Metric                      | Value                                                                                                             |
| --------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| Total session commits       | 5 (5ba6acd → b8cc03e)                                                                                             |
| Files created               | 10 (constants.go, auth_interfaces.go, 4 scripts, proposal HTML, planning doc, status report, 4 D2+SVG)            |
| Files modified              | 12 (response.go, flake.nix, AGENTS.md, TODO_LIST.md, ci.yml, 5 go.mod files, check-version-drift.sh, v2 proposal) |
| Version drifts fixed        | 5 (snapshot, schema, storage/memory, listing, scenario)                                                           |
| CI scripts passing (strict) | 4/4                                                                                                               |
| Build status                | ✅ All pass                                                                                                       |
| Test status                 | ✅ All pass (race)                                                                                                |
| BuildFlow                   | ✅ 30/30 checks pass                                                                                              |
| Coverage                    | Root 94.2%, usermgmt 79.4%, adminui 66.8%                                                                         |
| Open TODO items             | 3 (Phase 2b, v4 auth extraction, root module v4 eval)                                                             |
