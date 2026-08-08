# Library Integration Report: cqrs-htmx

> How well does cqrs-htmx leverage the LarsArtmann ecosystem — and where are the gaps?

**Date:** 2026-05-13 | **Project:** cqrs-htmx (library/SDK) | **Coverage:** 95.7% (150+ tests)

---

## Executive Summary

cqrs-htmx is a well-scoped integration library with **3 direct dependencies** (go-cqrs-lite, casbin, cockroachdb/errors) and **1 transitive ecosystem dependency** (go-branded-id via go-cqrs-lite). The existing integrations are **solid and idiomatic**. The main opportunities are: (1) upgrading error handling to use go-error-family directly for richer context, and (2) documenting templ-components as the recommended companion for consumers.

---

## Integration Summary Table

| Library                          | Status    | Used?      | Should Use?        | Integration Quality  | Action                                |
| -------------------------------- | --------- | ---------- | ------------------ | -------------------- | ------------------------------------- |
| **go-cqrs-lite**                 | ✅ Active | **Direct** | ✅ Yes (core)      | ⭐⭐⭐⭐⭐ Excellent | Maintain                              |
| **go-branded-id**                | ✅ Active | Transitive | ✅ Yes (core)      | ⭐⭐⭐⭐ Good        | Consider direct import                |
| **go-error-family**              | ✅ Active | Transitive | ⚠️ Should consider  | ⭐⭐⭐ Indirect      | Upgrade error model                   |
| **casbin/casbin/v3**             | ✅ Active | **Direct** | ✅ Yes (core)      | ⭐⭐⭐⭐⭐ Excellent | Maintain                              |
| **cockroachdb/errors**           | ✅ Active | **Direct** | ✅ Yes (adequate)  | ⭐⭐⭐⭐ Good        | Evaluate migration to go-error-family |
| **templ-components**             | ✅ Active | Not used   | 📋 Consumer-facing | N/A                  | Document as companion                 |
| **cmdguard**                     | ✅ Active | N/A        | ❌ No              | N/A                  | —                                     |
| **go-output**                    | ✅ Active | N/A        | ❌ No              | N/A                  | —                                     |
| **go-commit**                    | ✅ Active | N/A        | ❌ No              | N/A                  | —                                     |
| **universal-workflow**           | ✅ Active | N/A        | ❌ No              | N/A                  | —                                     |
| **ActaFlow**                     | ~80%      | N/A        | ❌ No              | N/A                  | —                                     |
| **go-filewatcher**               | ✅ Active | N/A        | ❌ No              | N/A                  | —                                     |
| **go-business-rules**            | ✅ Active | N/A        | ⚠️ Could document   | N/A                  | Optional: docs for consumers          |
| **go-composable-business-types** | ✅ Active | N/A        | ❌ No              | N/A                  | —                                     |
| **project-discovery-sdk**        | v0.1.0    | N/A        | ❌ No              | N/A                  | —                                     |
| **go-localfirst**                | ✅ Active | N/A        | ❌ No              | N/A                  | —                                     |
| **go-localsync**                 | ✅ Active | N/A        | ❌ No              | N/A                  | —                                     |
| **go-finding**                   | ✅ Active | N/A        | ❌ No              | N/A                  | —                                     |
| **gogenfilter**                  | ✅ Active | N/A        | ❌ No              | N/A                  | —                                     |
| **smart-configs**                | ✅ Active | N/A        | ⚠️ Could document   | N/A                  | Optional: docs for consumers          |

---

## Detailed Analysis

### Direct Dependencies (Currently Used)

#### 1. go-cqrs-lite — ⭐⭐⭐⭐⭐ Excellent

**How it's used:** Core CQRS dispatch for commands and queries.

| Area                           | Usage                                            | Quality                      |
| ------------------------------ | ------------------------------------------------ | ---------------------------- |
| `command.Dispatcher`           | `App.commands` — type-safe command routing       | ✅ Idiomatic                 |
| `query.Dispatcher`             | `App.queries` — type-safe query routing          | ✅ Idiomatic                 |
| `event.RegisterClassification` | Maps sentinels to families via `sync.Once`       | ✅ Correct — avoids `init()` |
| `event.Classify`               | Maps errors to HTTP status codes in `MapError()` | ✅ Clean 5-family mapping    |
| `event.WithUserID`             | Propagates identity into event metadata          | ✅ Good audit trail          |
| `command.Type` / `query.Type`  | Handler registration key                         | ✅ Type-safe                 |

**Verdict:** Model integration. go-cqrs-lite is the reason this library exists and every API is used well. The `sync.Once` lazy registration pattern is the right call for a library (avoids import side effects).

**Improvement:** None needed.

---

#### 2. casbin/casbin/v3 — ⭐⭐⭐⭐⭐ Excellent

**How it's used:** RBAC/ABAC authorization via `Enforcer` interface.

| Area                  | Usage                                               | Quality                          |
| --------------------- | --------------------------------------------------- | -------------------------------- |
| `Enforcer` interface  | Duck-types `*casbin.Enforcer`                       | ✅ Enables mocking               |
| `Authorize()` option  | Pre-dispatch policy check                           | ✅ Clean handler option          |
| `Enforce()` helper    | Subject/resource/action check with context wrapping | ✅ Error includes who/what       |
| `AuthorizeMiddleware` | Standalone middleware for non-CQRS routes           | ✅ HTMX-aware error handling     |
| `RequireAuth()`       | Auth without specific permission                    | ✅ Useful for login-gated routes |

**Verdict:** Textbook interface abstraction. The `Enforcer` interface means consumers can provide fakes for testing without importing Casbin at all. Error messages include subject/resource/action for debugging.

**Improvement:** None needed.

---

#### 3. cockroachdb/errors — ⭐⭐⭐⭐ Good

**How it's used:** Sentinel error creation, error wrapping with stack traces.

| Area             | Usage                                                     | Quality                          |
| ---------------- | --------------------------------------------------------- | -------------------------------- |
| `errors.New()`   | 7 sentinel errors                                         | ✅ Stack traces on all sentinels |
| `errors.Wrapf()` | Casbin enforce failure wrapping                           | ✅ Preserves cause chain         |
| `errors.Is()`    | Error identification in `MapError`, `DefaultErrorHandler` | ✅ Standard Go pattern           |

**Verdict:** Solid usage. cockroachdb/errors provides better stack traces than stdlib `errors`, which is valuable for a library consumed by others.

**Improvement opportunity:** go-error-family (now transitive via go-cqrs-lite) provides richer error context (`Coded`, `Classified`, `Contextual` interfaces). The current sentinels use `errors.New()` — they could use `errorfamily.NewRejection(...)` etc. to gain machine-readable error codes, structured context, and automatic family classification. This would eliminate the manual `registerErrorClassifications()` `sync.Once` block entirely.

---

### Transitive Dependencies (Used Indirectly)

#### 4. go-branded-id — ⭐⭐⭐⭐ Good (Indirect)

**How it's used:** Via `go-cqrs-lite/core/pkg/id` — `UserID` is a type alias for `id.UserID` (branded ULID).

| Area                      | Usage                      | Quality                        |
| ------------------------- | -------------------------- | ------------------------------ |
| `type UserID = id.UserID` | Type alias in `context.go` | ✅ Zero-cost abstraction       |
| `id.NewUserID()`          | Random ULID generation     | ✅ Used in `NewUserID()`       |
| `id.ParseUserID()`        | String → UserID validation | ✅ Used in `ParseUserID()`     |
| `id.MustParseUserID()`    | Panic variant for tests    | ✅ Used in `MustParseUserID()` |

**Verdict:** go-branded-id is the right primitive — compile-time type safety preventing `UserID`/`OrderID` mixing. Using it via go-cqrs-lite's `id` package is the correct dependency path (avoids coupling to branded-id directly).

**Improvement:** Consider adding go-branded-id as a direct dependency if the project ever needs branded IDs beyond `UserID` (e.g., `SessionID`, `RequestID`). Currently the transitive path is fine.

---

#### 5. go-error-family — ⭐⭐⭐ Indirect (Opportunity)

**How it's used:** Transitive via `go-cqrs-lite/core/event` — the `Family` type, `Classify`, and `RegisterClassification` are all go-error-family APIs re-exported by go-cqrs-lite.

**Why this matters:** go-cqrs-lite's `event/errors.go` is a thin wrapper around go-error-family:

```
event.Rejection = errorfamily.Rejection
event.Classify   → errorfamily.Classify
event.RegisterClassification → errorfamily.RegisterClassification
```

cqrs-htmx already depends on go-error-family concepts (families, classification) but only accesses them through go-cqrs-lite's re-exports.

**Opportunity:** cqrs-htmx could use go-error-family directly to:

- Create structured errors with `errorfamily.NewRejection("unauthorized", "...")` instead of bare `errors.New()`
- Get automatic family classification without the `sync.Once` registration block
- Add `ErrorCode()` and `ErrorContext()` to all sentinels — consumers get machine-readable error details
- Use `HandleError()` in the default error handler for Wix-quality messages

**Risk:** Low — go-error-family is already a transitive dep with zero external dependencies. Adding it as a direct dep only formalizes what's already there.

**Recommendation:** ⚠️ **Consider for next breaking change.** Not urgent — the current `RegisterClassification` + `sync.Once` pattern works correctly. But migrating to go-error-family constructors would simplify `errors.go` and provide richer error metadata for consumers.

---

### Libraries NOT Used (With Rationale)

#### Should NOT Be Used (Correct Exclusions)

These libraries solve problems outside cqrs-htmx's scope:

| Library                          | Why Not                                | Scope Mismatch                                  |
| -------------------------------- | -------------------------------------- | ----------------------------------------------- |
| **cmdguard**                     | CLI framework                          | cqrs-htmx is a web library, not a CLI tool      |
| **go-output**                    | Output formatting (12 formats)         | cqrs-htmx writes HTTP responses, not CLI output |
| **go-commit**                    | AI git commit messages                 | CI/DevX tool, not a library dependency          |
| **universal-workflow**           | Multi-step workflow orchestration      | cqrs-htmx dispatches single commands/queries    |
| **ActaFlow**                     | Actor model / concurrency              | In-memory actor system, not HTTP handler layer  |
| **go-filewatcher**               | File system watching                   | No file system interaction in cqrs-htmx         |
| **go-composable-business-types** | Domain primitives (Money, Email, etc.) | cqrs-htmx is transport/auth, not domain         |
| **project-discovery-sdk**        | Project scanning                       | Completely different domain                     |
| **go-localfirst**                | Offline-first sync                     | Consumer app concern, not library concern       |
| **go-localsync**                 | External API sync                      | Consumer app concern                            |
| **go-finding**                   | Static analysis findings               | Completely different domain                     |
| **gogenfilter**                  | Generated code detection               | No code generation in cqrs-htmx                 |

#### Could Document for Consumers

| Library               | Why                                                                                                                                          | What to Do                                               |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| **templ-components**  | Natural companion — 53 UI components, HTMX helpers, dark mode, CSP-compliant. cqrs-htmx's `TemplComponent` duck-typing is designed for this. | Add "Recommended Companion" section to README            |
| **go-business-rules** | Consumers often validate commands before dispatch. go-business-rules provides severity-aware validation with JSON-serializable results.      | Add example in docs showing validation in command mapper |
| **smart-configs**     | Consumers need to configure Casbin model/policy paths, database URLs, etc. smart-configs provides actionable error messages.                 | Add example in docs showing config resolution            |

---

## Integration Quality Scorecard

| Dimension                  | Score      | Notes                                                                  |
| -------------------------- | ---------- | ---------------------------------------------------------------------- |
| **Core dependency usage**  | ⭐⭐⭐⭐⭐ | go-cqrs-lite APIs used fully and idiomatically                         |
| **Authorization design**   | ⭐⭐⭐⭐⭐ | Interface abstraction enables testing, error messages include context  |
| **Error handling**         | ⭐⭐⭐⭐   | Good classification, could be richer with go-error-family direct usage |
| **Type safety**            | ⭐⭐⭐⭐⭐ | Branded IDs, typed enums, generic decoders                             |
| **Dependency footprint**   | ⭐⭐⭐⭐⭐ | Minimal — 3 direct deps, no bloat                                      |
| **Consumer documentation** | ⭐⭐⭐     | Good README, missing ecosystem companion docs                          |
| **Ecosystem alignment**    | ⭐⭐⭐⭐   | Uses core deps well, one upgrade opportunity (error-family)            |

---

## Recommended Actions (Priority Order)

| # | Priority     | Action                                                                                                                 | Impact                                              | Effort                                                               |
| - | ------------ | ---------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------- | -------------------------------------------------------------------- |
| 1 | **Low**      | Add `go-error-family` as direct dependency; migrate sentinels from `errors.New()` to `errorfamily.New*()` constructors | Removes `sync.Once` block, adds error codes/context | Medium (breaking if consumers use `errors.Is` on sentinels — verify) |
| 2 | **Low**      | Document templ-components as recommended companion in README                                                           | Better consumer experience                          | Low (docs only)                                                      |
| 3 | **Optional** | Add `go-business-rules` example showing validation in `DecodeJSON` mapper                                              | Educational for consumers                           | Low (docs only)                                                      |
| 4 | **Optional** | Add `smart-configs` example for Casbin/config setup                                                                    | Educational for consumers                           | Low (docs only)                                                      |

---

## Conclusion

cqrs-htmx has a **lean, well-targeted dependency profile**. It correctly avoids pulling in libraries that don't serve its core mission (HTMX + CQRS + Authz integration). The three direct dependencies are used idiomatically with no obvious gaps.

The one meaningful improvement opportunity is **direct go-error-family usage** — the library already depends on it transitively and uses its concepts, but creates sentinels with `cockroachdb/errors.New()` instead of `errorfamily.New*()` constructors. This would be a low-risk, high-clarity change that eliminates boilerplate and adds structured error metadata.

The ecosystem libraries that are NOT used are correctly excluded — they solve different problems at different architectural layers.
