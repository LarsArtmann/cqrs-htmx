# Comprehensive Status Report — Domain Model Enrichment & Events

**Date:** 2026-05-28 13:08 UTC+2
**Branch:** master
**Commits since last report:** 6
**Total commits on master:** 15+ since v1.0.0

---

## Executive Summary

This session focused on eliminating CRUD patterns from the `usermgmt` submodule by enriching the `User` domain model with behavior methods, extracting timestamp ownership from the store layer to the domain entity, co-locating validation logic, and adding domain event emission. The codebase now has zero CRUD smells in production code — all mutations go through domain methods.

**Key outcomes:**

- 10 new domain methods on `User`
- 4 domain event types with optional `EventHandler` callback
- `validatePassword` co-located in `user.go` (where it's used)
- `InMemoryUserStore` is pure persistence (no timestamp side-effects)
- Zero production code directly mutates `User` fields
- All tests pass with race detector: root 96.5%, usermgmt 91.0%

---

## a) FULLY DONE

### Domain Model Enrichment

| Method                                | File      | Description                                                                                |
| ------------------------------------- | --------- | ------------------------------------------------------------------------------------------ |
| `User.SetRoles(roles)`                | `user.go` | Replaces entire role list with defensive copy, updates `UpdatedAt`                         |
| `User.ChangePassword(old, new, cost)` | `user.go` | Verifies old password, validates new, hashes, updates `UpdatedAt`. Returns `(bool, error)` |
| `User.SetEmail(email)`                | `user.go` | Updates email, records `UpdatedAt`. Caller validates format                                |
| `User.SetDisplayName(name)`           | `user.go` | Updates display name, records `UpdatedAt`                                                  |
| `User.AddRole(role)`                  | `user.go` | Appends role if not present, updates `UpdatedAt`                                           |
| `User.RemoveRole(role)`               | `user.go` | Removes first occurrence, updates `UpdatedAt`                                              |
| `User.HasRole(role)`                  | `user.go` | `slices.Contains` check                                                                    |
| `User.SetPassword(password)`          | `user.go` | Hashes with default cost (12)                                                              |
| `User.SetPasswordWithCost(pw, cost)`  | `user.go` | Hashes with specified cost                                                                 |
| `User.CheckPassword(pw)`              | `user.go` | bcrypt comparison                                                                          |
| `User.IsPasswordSet()`                | `user.go` | Replaces scattered `PasswordHash != ""` checks                                             |
| `User.touch()`                        | `user.go` | Private helper: `UpdatedAt = time.Now().UTC()` — used by all mutation methods              |

### Service Refactoring (CRUD Elimination)

| Before                                                   | After                      | File                        |
| -------------------------------------------------------- | -------------------------- | --------------------------- |
| `user.Roles = roles; user.UpdatedAt = now`               | `user.SetRoles(roles)`     | `service.go:UpdateRoles`    |
| `CheckPassword + validatePassword + SetPasswordWithCost` | `user.ChangePassword(...)` | `service.go:ChangePassword` |
| `validatePassword` in `service.go`                       | moved to `user.go`         | `user.go`                   |
| `PasswordHash != ""`                                     | `u.IsPasswordSet()`        | `user.go:MarshalJSON`       |

### Store Layer Cleanup

| Store                        | Before                                  | After                 |
| ---------------------------- | --------------------------------------- | --------------------- |
| `InMemoryUserStore.Save()`   | Set `user.UpdatedAt = time.Now().UTC()` | Pure persistence only |
| `InMemoryUserStore.Create()` | Set `user.UpdatedAt = time.Now().UTC()` | Pure persistence only |

### Domain Events

| Event Type             | Emitted From               | Fields                                  |
| ---------------------- | -------------------------- | --------------------------------------- |
| `UserRegisteredEvent`  | `Service.Register()`       | Email, DisplayName, Roles[], OccurredAt |
| `UserLoggedInEvent`    | `Service.Login()`          | Email, OccurredAt                       |
| `PasswordChangedEvent` | `Service.ChangePassword()` | OccurredAt                              |
| `RolesUpdatedEvent`    | `Service.UpdateRoles()`    | Roles[], Domain, OccurredAt             |

**Event handler characteristics:**

- Optional: `nil` means no events emitted (zero overhead)
- Panic-safe: `recover()` catches handler panics, logs at warn, never fails operation
- Defensive copies: role slices are copied before passing to handler

### Tests Added

| Test File           | New Tests | What They Cover                                                                                                                     |
| ------------------- | --------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `user_test.go`      | 8         | `SetRoles`, `SetRoles_Nil`, `SetRoles_Empty`, `ChangePassword` (success/wrong/short), `SetEmail`, `SetDisplayName`, `IsPasswordSet` |
| `events_test.go`    | 6         | Register event, Login event, ChangePassword event, UpdateRoles event, panic recovery, nil handler                                   |
| `fuzz_test.go`      | 1         | `FuzzUser_ChangePassword` — 38K+ executions in 3s, 12 interesting inputs                                                            |
| `benchmark_test.go` | 2         | `BenchmarkUser_ChangePassword` (~1.9ms/op), `BenchmarkUser_SetRoles` (~53ns/op)                                                     |

### Documentation

- `AGENTS.md`: Added "Domain Model (usermgmt)" section with timestamp ownership, service delegation, validation co-location
- `FEATURES.md`: Added features 37b (Domain Model) and 37c (Domain Events) as FULLY_FUNCTIONAL

### Commits

| Hash      | Message                                                                       |
| --------- | ----------------------------------------------------------------------------- |
| `57e8ef0` | refactor(usermgmt): enrich User domain model to eliminate CRUD patterns       |
| `fa84a91` | refactor(usermgmt): co-locate domain validation, DRY touch(), fix test bypass |
| `397a2ee` | feat(usermgmt): add SetEmail, SetDisplayName, IsPasswordSet domain methods    |
| `940c2ed` | docs: update AGENTS.md with enriched domain model details                     |
| `b3f45a5` | feat(usermgmt): add domain events with optional EventHandler                  |
| `8393b58` | feat(usermgmt): fuzz tests, benchmarks, domain events, FEATURES.md            |

---

## b) PARTIALLY DONE

### go-structure-linter Warnings

The pre-commit `go-structure-linter` step reports 21 structural issues (all pre-existing, not introduced by this session):

- **19 HIGH**: Root package files should be in `internal/` or `pkg/` — this is intentional for a flat library package
- **1 MEDIUM**: `coverage.out` in root — should move to `coverage/` directory
- **1 MEDIUM**: Consider `pkg/` directory for public library code — documented in ADR as intentional flat structure

These are architectural decisions, not bugs. The root module is intentionally a single flat package (19 files, ~10K lines) per ADR analysis.

### LSP Diagnostics

- ~31 stale LSP warnings (cache issue), CLI reports 0 issues
- New hint: `b.N` can be modernized to `b.Loop()` in benchmarks (Go 1.24+) — cosmetic, not a bug

---

## c) NOT STARTED

### From TODO_LIST.md

| Item                                    | Status      | Blocker                                                                    |
| --------------------------------------- | ----------- | -------------------------------------------------------------------------- |
| BrandNamer for root module marker types | BLOCKED     | upstream `go-cqrs-lite` — marker types are unexported                      |
| SQL store backend for usermgmt          | NOT_STARTED | Requires persistent storage decision (PostgreSQL? SQLite?)                 |
| OpenTelemetry integration               | NOT_STARTED | Hooks exist (`BeforeDispatchHook`/`AfterDispatchHook`), no OTel middleware |
| WebSocket/SSE helpers                   | NOT_PLANNED | Would require significant API surface                                      |

### Potential Enhancements (Not in TODO_LIST)

1. **Role type safety** — `Role` is currently `string`, could be branded type
2. **PasswordHash as named type** — currently raw `string` in struct
3. **Event serialization** — domain events are plain structs, no JSON marshaling tests
4. **Event ordering guarantees** — current handler is synchronous, no ordering tests
5. **Audit log from events** — example: `EventHandler` that writes to structured log

---

## d) TOTALLY FUCKED UP!

**Nothing.** All tests pass, lint is clean (CLI), build succeeds across all 4 modules.

One cosmetic issue: the sed command in the earlier session accidentally added `PasswordChangedEvent` to `Logout` and `UpdateRoles` save paths — caught and fixed immediately before commit. No bugs shipped.

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (Next Session)

1. **Write the status report file** — this document needs to be committed
2. **Update TODO_LIST.md** — mark completed items, add new findings
3. **service.go line count** — at 425 lines (+75 over 350 limit), should split into `register.go`, `login.go`, `password.go`, `roles.go`

### Short Term

4. **EventHandler async dispatch** — currently synchronous; add example of goroutine dispatch
5. **Event JSON marshaling tests** — ensure all event types marshal/unmarshal correctly
6. **BenchmarkUser_AddRole/RemoveRole** — fill benchmark gap for remaining domain methods
7. **Test event slice isolation** — verify handler can't mutate the emitted slice
8. **Move `maxDisplayNameLength` to user.go** — still in `service.go`, should be with `User` entity

### Medium Term

9. **SQL store backend** — implement `PostgresUserStore` / `SQLiteUserStore`
10. **Event store integration** — bridge domain events to `go-cqrs-lite` event bus
11. **Role as branded type** — `type Role = brandid.ID[roleBrand, string]` for compile-time safety
12. **Password strength scoring** — zxcvbn integration instead of just length checks
13. **TOTP/2FA support** — `EnableTOTP`, `ValidateTOTP` methods on `User`
14. **Account deletion / GDPR right-to-be-forgotten** — `Service.DeleteUser()` with cascading cleanup

### Architecture Questions

15. **Should `User` fields be private?** — Currently `Email`, `DisplayName`, `Roles`, `UpdatedAt` are exported. External packages can mutate them directly, bypassing domain methods. Tradeoff: convenience vs. encapsulation.
16. **EventHandler return value** — Should it return `error`? Currently panics are recovered but silent failures are possible.
17. **Event replay capability** — Should events be persisted for replay? Current design is fire-and-forget.

---

## f) Top #25 Things To Get Done Next

| #  | Priority | Task                                                              | Impact | Effort | Category      |
| -- | -------- | ----------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | H        | Commit this status report + update TODO_LIST.md                   | L      | 5min   | Docs          |
| 2  | H        | Split `service.go` (425 lines) into per-operation files           | M      | 20min  | Refactor      |
| 3  | H        | Add event JSON marshaling tests                                   | M      | 10min  | Test          |
| 4  | H        | Add `EventHandler` async dispatch example in docs                 | M      | 15min  | Docs          |
| 5  | M        | Move `maxDisplayNameLength` constant to `user.go`                 | L      | 3min   | Refactor      |
| 6  | M        | Add `BenchmarkUser_AddRole` + `BenchmarkUser_RemoveRole`          | L      | 5min   | Test          |
| 7  | M        | Test event slice isolation (mutability)                           | M      | 5min   | Test          |
| 8  | M        | SQL store backend (`PostgresUserStore`)                           | H      | 2h     | Feature       |
| 9  | M        | Event store bridge to `go-cqrs-lite` event bus                    | H      | 1h     | Feature       |
| 10 | M        | Role branded type (`brandid.ID[roleBrand, string]`)               | M      | 30min  | Type Safety   |
| 11 | M        | `PasswordHash` as named type                                      | L      | 15min  | Type Safety   |
| 12 | M        | TOTP/2FA support                                                  | H      | 2h     | Feature       |
| 13 | M        | Account deletion with cascading cleanup                           | H      | 1h     | Feature       |
| 14 | M        | Password strength scoring (zxcvbn)                                | M      | 30min  | Security      |
| 15 | L        | Update `CHANGELOG.md` with session changes                        | L      | 10min  | Docs          |
| 16 | L        | Add `go doc` example for `EventHandler` usage                     | L      | 10min  | Docs          |
| 17 | L        | `User` field privacy — make exported fields private with getters? | M      | 1h     | Architecture  |
| 18 | L        | Event ordering guarantee tests                                    | L      | 15min  | Test          |
| 19 | L        | Add `SessionRenew` operation (extend TTL)                         | M      | 30min  | Feature       |
| 20 | L        | `User.Deactivate()` / `User.Reactivate()`                         | M      | 30min  | Feature       |
| 21 | L        | Rate limiting per endpoint in `AuthHandler`                       | M      | 1h     | Security      |
| 22 | L        | OpenTelemetry middleware using lifecycle hooks                    | H      | 2h     | Observability |
| 23 | L        | WebSocket/SSE notification helpers                                | M      | 2h     | Feature       |
| 24 | L        | `go generate` for event type registry                             | L      | 1h     | Tooling       |
| 25 | L        | Property-based tests for `Register`/`Login` invariants            | M      | 1h     | Test          |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the `User` struct fields (`Email`, `DisplayName`, `Roles`, `UpdatedAt`) be made unexported with getter/setter methods?**

Currently, `Email`, `DisplayName`, `Roles`, and `UpdatedAt` are all exported (public). This means any external package can do:

```go
user.Email = "hacked@evil.com"
user.Roles = []Role{RoleAdmin}
user.UpdatedAt = time.Now()
```

...completely bypassing domain validation, timestamp updates, and intent-revealing methods.

**Arguments FOR making them private:**

- True encapsulation — impossible to bypass domain methods
- `SetEmail` can validate format
- `SetRoles` updates `UpdatedAt` automatically
- Consistent with `PasswordHash` which is already private

**Arguments AGAINST making them private:**

- Breaks JSON marshaling without custom `MarshalJSON` (already have one)
- Breaks direct struct literal creation in tests (need constructor or builder)
- This is a Go library — consumers may want to construct `User` directly for testing
- `Clone()` already returns copies, so external mutation of the _returned_ copy is harmless
- The store returns clones, so stored data is protected

**What I don't know:** What's the intended consumer usage pattern? Is `User` meant to be constructed directly by consumers, or always through `Service.Register()`? If the former, private fields are hostile. If the latter, private fields are correct.

---

## Metrics

| Metric            | Root    | Usermgmt | Integration | datastar-demo |
| ----------------- | ------- | -------- | ----------- | ------------- |
| Coverage          | 96.5%   | 91.0%    | —           | —             |
| Test funcs        | ~200    | 175      | 5           | —             |
| Benchmark funcs   | 8       | 5        | —           | —             |
| Fuzz funcs        | 2       | 3        | —           | —             |
| Example funcs     | 12      | 4        | —           | —             |
| Lines (prod)      | ~10,089 | ~1,836   | —           | —             |
| Lines (test)      | ~6,543  | ~3,208   | —           | —             |
| Lint issues (CLI) | 0       | 0        | —           | —             |
| Race detector     | Pass    | Pass     | Pass        | —             |

## Files Changed This Session

| File                         | +/-       | Description                                              |
| ---------------------------- | --------- | -------------------------------------------------------- |
| `usermgmt/user.go`           | +82 / -6  | New domain methods, `validatePassword`, `touch()`        |
| `usermgmt/service.go`        | +46 / -20 | Event emission, `EventHandler`, domain delegation        |
| `usermgmt/store.go`          | +0 / -2   | Removed `UpdatedAt` mutations                            |
| `usermgmt/user_test.go`      | +80 / -1  | Domain method tests                                      |
| `usermgmt/events.go`         | +60 / 0   | New — event types and handler interface                  |
| `usermgmt/events_test.go`    | +142 / 0  | New — event emission tests                               |
| `usermgmt/fuzz_test.go`      | +12 / 0   | `FuzzUser_ChangePassword`                                |
| `usermgmt/benchmark_test.go` | +20 / 0   | `BenchmarkUser_ChangePassword`, `BenchmarkUser_SetRoles` |
| `usermgmt/coverage_test.go`  | +0 / -1   | Fixed test bypass                                        |
| `AGENTS.md`                  | +5 / -2   | Domain model section                                     |
| `FEATURES.md`                | +2 / -0   | Features 37b, 37c                                        |
