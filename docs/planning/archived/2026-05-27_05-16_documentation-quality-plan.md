# Comprehensive Documentation & Quality Plan

**Date:** 2026-05-27_05-16
**Status:** Planning
**Goal:** Fix all stale documentation, broken examples, and quality issues across the cqrs-htmx library

## Pareto Analysis

### 1% → 51% Impact

- Fix 2 broken README code examples (won't compile — `UserIDExtractor` signature changed)
- These block anyone trying to use the library from the README

### 4% → 64% Impact

- Fix all gorilla/csrf → nosurf references in README, SECURITY.md, CONTRIBUTING.md
- Fix all cockroachdb/errors → go-error-family references
- These undermine trust for production evaluators

### 20% → 80% Impact

- Complete README refresh: missing features, wrong return types, stale deps table
- Run go mod tidy to clean stale go.sum entries
- Fix missing godoc on StatusRecorder.Write

---

## Execution Graph

```mermaid
graph TD
    subgraph Phase 1: 1% → 51%
        A1[A1: Fix README UserIDExtractor examples] --> A2[A2: Fix README usermgmt bridge example]
    end

    subgraph Phase 2: 4% → 64%
        B1[B1: Fix README CSRF section - nosurf] --> B2[B2: Fix SECURITY.md stale refs]
        B3[B3: Fix CONTRIBUTING.md stale refs] --> B4[B4: Fix README deps table]
    end

    subgraph Phase 3: 20% → 80%
        C1[C1: Fix README catalog return types] --> C2[C2: Add missing features to README]
        C3[C3: Fix README CSRFConfig.Secret] --> C4[C4: Fix README config section]
        C5[C5: Fix CONTRIBUTING.md dir tree] --> C6[C6: Run go mod tidy all modules]
        C7[C7: Add godoc to StatusRecorder.Write] --> C8[C8: Verify all tests pass]
    end

    A2 --> B1
    B2 --> B3
    B4 --> C1
    C4 --> C5
    C6 --> C7
    C8 --> DONE[Done - commit & push]
```

---

## Task Breakdown: ~30min Each (27 tasks)

| #  | Task                                                                                                           | Impact   | Effort | File            |
| -- | -------------------------------------------------------------------------------------------------------------- | -------- | ------ | --------------- |
| 1  | Fix README Quick Start UserIDExtractor to return (UserID, error)                                               | Critical | 15min  | README.md       |
| 2  | Fix README usermgmt bridge example (UserIDExtractor signature)                                                 | Critical | 15min  | README.md       |
| 3  | Rewrite README CSRF section for justinas/nosurf                                                                | High     | 30min  | README.md       |
| 4  | Remove CSRFConfig.Secret from README (field removed)                                                           | High     | 10min  | README.md       |
| 5  | Fix README catalog return types (CatalogMeta → HandlerMeta)                                                    | High     | 10min  | README.md       |
| 6  | Update README dependencies table (nosurf, go-error-family, httputil, remove gorilla/csrf & cockroachdb/errors) | High     | 15min  | README.md       |
| 7  | Add missing handler options to README (WithMaxBodySize, WithSuccessStatus, RequireMethod, OnError)             | Medium   | 15min  | README.md       |
| 8  | Add HealthHandler, MustNew, IsAuthenticated, HasCommands/HasQueries to README                                  | Medium   | 20min  | README.md       |
| 9  | Add Config.MaxBodySize and Config.IncludeRequestIDInErrors to README config section                            | Medium   | 15min  | README.md       |
| 10 | Fix README architecture tree (add recovery.go, update httputil.go desc)                                        | Medium   | 10min  | README.md       |
| 11 | Fix README go-cqrs-lite version (v1.4.0 → v1.5.1)                                                              | Medium   | 5min   | README.md       |
| 12 | Fix SECURITY.md: replace all gorilla/csrf → nosurf references                                                  | High     | 20min  | SECURITY.md     |
| 13 | Fix SECURITY.md: replace cockroachdb/errors → go-error-family in deps table                                    | High     | 10min  | SECURITY.md     |
| 14 | Fix CONTRIBUTING.md: remove cockroachdb/errors note, update error wrapping section                             | High     | 15min  | CONTRIBUTING.md |
| 15 | Fix CONTRIBUTING.md: update architecture dir tree (add missing files)                                          | Medium   | 15min  | CONTRIBUTING.md |
| 16 | Fix CONTRIBUTING.md: update errors.go description (go-error-family classification)                             | Medium   | 10min  | CONTRIBUTING.md |
| 17 | Run go mod tidy in root, usermgmt, integration_test modules                                                    | Low      | 10min  | go.mod/go.sum   |
| 18 | Add godoc to StatusRecorder.Write in logging.go                                                                | Low      | 5min   | logging.go      |
| 19 | Update AGENTS.md Key Decisions: verify nosurf/go-error-family accuracy                                         | Low      | 10min  | AGENTS.md       |
| 20 | Update CHANGELOG.md: add nosurf migration, go-error-family migration entries                                   | Medium   | 20min  | CHANGELOG.md    |
| 21 | Update FEATURES.md metrics (benchmark count, file counts) if changed by tidy                                   | Low      | 10min  | FEATURES.md     |
| 22 | Verify root tests pass after all changes                                                                       | Critical | 10min  | -               |
| 23 | Verify usermgmt tests pass after all changes                                                                   | Critical | 10min  | -               |
| 24 | Verify integration_test passes after all changes                                                               | Critical | 10min  | -               |
| 25 | Verify lint passes (0 issues) in root and usermgmt                                                             | Critical | 10min  | -               |
| 26 | Verify datastar-demo builds                                                                                    | Low      | 5min   | -               |
| 27 | Final cross-doc consistency check (coverage, versions, deps)                                                   | Medium   | 15min  | All docs        |

---

## Fine-Grained Breakdown: ~15min Each (75 tasks)

| #  | Task                                                                         | Parent | Est   |
| -- | ---------------------------------------------------------------------------- | ------ | ----- |
| 1  | Find exact line of broken UserIDExtractor in README Quick Start              | T1     | 5min  |
| 2  | Write correct UserIDExtractor func returning (cqrshtmx.UserID, error)        | T1     | 5min  |
| 3  | Update surrounding example code for new signature                            | T1     | 5min  |
| 4  | Find exact line of usermgmt bridge UserIDExtractor                           | T2     | 5min  |
| 5  | Write correct bridge using MustParseUserID + usermgmt.UserIDFromRequest      | T2     | 5min  |
| 6  | Update any related context enrichment example                                | T2     | 5min  |
| 7  | Find CSRF section in README, identify all gorilla/csrf claims                | T3     | 5min  |
| 8  | Rewrite CSRF intro paragraph for nosurf                                      | T3     | 10min |
| 9  | Rewrite CSRF token generation description (crypto/rand, one-time-pad)        | T3     | 5min  |
| 10 | Update CSRF usage examples if referencing gorilla-specific APIs              | T3     | 10min |
| 11 | Remove CSRFConfig.Secret row from CSRF config table                          | T4     | 5min  |
| 12 | Add any new CSRFConfig fields that nosurf uses                               | T4     | 5min  |
| 13 | Find CatalogEntries examples, identify CatalogMeta references                | T5     | 5min  |
| 14 | Replace CatalogMeta with dispatcher.HandlerMeta in examples                  | T5     | 5min  |
| 15 | Find deps table in README                                                    | T6     | 5min  |
| 16 | Remove gorilla/csrf row                                                      | T6     | 2min  |
| 17 | Remove cockroachdb/errors row                                                | T6     | 2min  |
| 18 | Add justinas/nosurf row                                                      | T6     | 2min  |
| 19 | Add go-error-family row                                                      | T6     | 2min  |
| 20 | Add larsartmann/httputil row                                                 | T6     | 2min  |
| 21 | Update go-cqrs-lite version to v1.5.1                                        | T6     | 2min  |
| 22 | Add WithMaxBodySize to handler options table                                 | T7     | 5min  |
| 23 | Add WithSuccessStatus to handler options table                               | T7     | 5min  |
| 24 | Add RequireMethod to handler options table                                   | T7     | 5min  |
| 25 | Add OnError to handler options table                                         | T7     | 5min  |
| 26 | Add HealthHandler section                                                    | T8     | 5min  |
| 27 | Add MustNew mention                                                          | T8     | 5min  |
| 28 | Add IsAuthenticated helper mention                                           | T8     | 5min  |
| 29 | Add HasCommands/HasQueries mention                                           | T8     | 5min  |
| 30 | Add MaxBodySize to Config example                                            | T9     | 5min  |
| 31 | Add IncludeRequestIDInErrors to Config example                               | T9     | 5min  |
| 32 | Add RecoveryMiddleware to Config if not present                              | T9     | 5min  |
| 33 | Add recovery.go to architecture tree                                         | T10    | 5min  |
| 34 | Update httputil.go description (WriteJSON local, ClientIP delegated)         | T10    | 5min  |
| 35 | Update errors.go description (go-error-family, not just sentinels)           | T10    | 5min  |
| 36 | Find and replace go-cqrs-lite v1.4.0 → v1.5.1                                | T11    | 5min  |
| 37 | Find SECURITY.md line 7 gorilla/csrf header                                  | T12    | 5min  |
| 38 | Replace with nosurf description                                              | T12    | 5min  |
| 39 | Find SECURITY.md line 126 gorilla/csrf update note                           | T12    | 5min  |
| 40 | Replace with nosurf update guidance                                          | T12    | 5min  |
| 41 | Find SECURITY.md line 134 gorilla/csrf HTTPS limitation                      | T12    | 5min  |
| 42 | Replace with nosurf behavior                                                 | T12    | 5min  |
| 43 | Replace gorilla/csrf in SECURITY.md deps table with nosurf                   | T13    | 5min  |
| 44 | Replace cockroachdb/errors in SECURITY.md deps table with go-error-family    | T13    | 5min  |
| 45 | Add missing deps to SECURITY.md table                                        | T13    | 5min  |
| 46 | Find CONTRIBUTING.md line 76 cockroachdb/errors note                         | T14    | 5min  |
| 47 | Remove or replace with go-error-family note                                  | T14    | 5min  |
| 48 | Find CONTRIBUTING.md line 73 errors.Wrapf anti-pattern                       | T14    | 5min  |
| 49 | Update to show current anti-pattern                                          | T14    | 5min  |
| 50 | Find CONTRIBUTING.md dir tree section                                        | T15    | 5min  |
| 51 | Add csrf.go, csrf_handler.go, csrf_helpers.go                                | T15    | 3min  |
| 52 | Add decoder.go, httputil.go, logging.go                                      | T15    | 3min  |
| 53 | Add ratelimit.go, recovery.go, security.go                                   | T15    | 3min  |
| 54 | Update errors.go description in tree                                         | T16    | 5min  |
| 55 | Verify tree matches actual file list                                         | T16    | 5min  |
| 56 | Run go mod tidy in root                                                      | T17    | 3min  |
| 57 | Run go mod tidy in usermgmt                                                  | T17    | 3min  |
| 58 | Run go mod tidy in integration_test                                          | T17    | 3min  |
| 59 | Check if any go.sum entries changed                                          | T17    | 3min  |
| 60 | Add godoc comment to StatusRecorder.Write                                    | T18    | 5min  |
| 61 | Verify AGENTS.md nosurf/go-error-family accuracy                             | T19    | 5min  |
| 62 | Check AGENTS.md architecture tree completeness                               | T19    | 5min  |
| 63 | Add CHANGELOG entry for nosurf migration                                     | T20    | 10min |
| 64 | Add CHANGELOG entry for go-error-family migration                            | T20    | 10min |
| 65 | Verify root tests: go test ./... -count=1 -race                              | T22    | 5min  |
| 66 | Verify usermgmt tests: cd usermgmt && go test ./... -count=1 -race           | T23    | 5min  |
| 67 | Verify integration_test: cd integration_test && go test ./... -count=1 -race | T24    | 5min  |
| 68 | Verify lint root: golangci-lint run                                          | T25    | 5min  |
| 69 | Verify lint usermgmt: cd usermgmt && golangci-lint run                       | T25    | 5min  |
| 70 | Build datastar-demo                                                          | T26    | 5min  |
| 71 | Cross-check coverage numbers across all 4 doc files                          | T27    | 5min  |
| 72 | Cross-check dep versions across all doc files                                | T27    | 5min  |
| 73 | Cross-check feature claims FEATURES.md vs actual code                        | T27    | 5min  |
| 74 | Update FEATURES.md metrics if go mod tidy changed anything                   | T21    | 5min  |
| 75 | Final read-through of README for any remaining inconsistencies               | T27    | 5min  |
