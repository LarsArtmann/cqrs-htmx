# Self-Review Fix Plan — 2026-06-25

> The v3.1.0 adoption work left **57 lint issues** (was 0), a **breaking API change**
> on `EventSourcedSetup`, **SecurityHooks silently ignored** in stack presets,
> a **copy-paste bug** in `postgres_setup.go`, and **968 lines of duplicated code**.

## Brutal Self-Review — What Went Wrong

### CRITICAL (broke things that worked)

1. **57 lint issues introduced** — went from 0 to 57 by writing sloppy code
   - 19 exhaustruct, 15 wrapcheck, 8 errcheck, 6 musttag, 4 staticcheck, 2 cyclop, 2 goconst, 1 golines
2. **Breaking API change on `EventSourcedSetup`** — changed `ReadModel *UserReadModel` to `event.Projection`. Every consumer accessing `setup.ReadModel.FindByID()` breaks.
3. **SecurityHooks lie** — `NewSQLiteEventSourcedSetup` accepts `SecurityHooks` in config but **never calls `wrapEventStore` or `applyBusMiddleware`**. The doc comment claims they're applied.
4. **Postgres copy-paste bug** — `postgres_setup.go` calls `NewSQLiteMembershipReadModel` (SQLite dialect) instead of Postgres equivalents. No Postgres variants exist for Membership/Tenant/Bot.

### BAD (code quality)

5. **968 lines of duplicated code** — 4 near-identical SQL read model implementations (User/Membership/Tenant/Bot). Same pattern repeated: `View` → `viewFrom*` → `*FromView` → `SQL*ReadModel` → `Handle` → `FindBy*SQL`.
6. **Unwrapped errors** — 15 `wrapcheck` violations. The project wraps ALL external errors with `event.WrapTransient`. I returned raw errors from `store.Set/Delete/Get`, `bundle.Close()`, etc.
7. **Missing json tags** — View structs have `view:"..."` tags but no `json:"..."` tags, despite being serialized via `json.Marshal`/`json.Unmarshal`. `musttag` linter catches this.
8. **Unchecked test errors** — 8 `errcheck` violations in test files.

### Type Model Issues

9. **No generic abstraction** — The SQL read model pattern should be `SQLProjection[View, Key]`, eliminating 4 copy-pasted implementations.
10. **Time as string** — Using `string` for time fields in views because SQLite TEXT doesn't scan into `time.Time`. Better: use `int64` Unix timestamps or `datetime` text format.
11. **Double storage** — Each view stores data both as scalar columns AND as a JSON blob in `Data`. Intentional but undocumented.

---

## Execution Plan — 26 tasks, all ≤12 min

Sorted by: **Damage fix first** → **Architecture** → **Polish**

| #  | Task                                                                                | Category     | Impact | Effort | Min |
| -- | ----------------------------------------------------------------------------------- | ------------ | ------ | ------ | --- |
| 1  | Revert `EventSourcedSetup` to concrete types; keep `*UserReadModel` etc.            | CRITICAL     | HIGH   | LOW    | 10  |
| 2  | Add `ReadModelDB` to config WITHOUT changing existing field types                   | CRITICAL     | HIGH   | LOW    | 8   |
| 3  | Fix SecurityHooks: apply `wrapEventStore` + `applyBusMiddleware` in sqlite/postgres | CRITICAL     | HIGH   | MED    | 12  |
| 4  | Fix postgres_setup.go copy-paste: create NewSQL\*ReadModel for Postgres dialect     | CRITICAL     | HIGH   | MED    | 12  |
| 5  | Consolidate 4 SQL read models into generic pattern (one file, less duplication)     | ARCHITECTURE | HIGH   | HIGH   | 12  |
| 6  | Add `json:"..."` tags to all View structs                                           | LINT FIX     | MED    | LOW    | 5   |
| 7  | Wrap all external errors with `event.WrapTransient` (15 wrapcheck fixes)            | LINT FIX     | HIGH   | MED    | 12  |
| 8  | Fix staticcheck QF1008: remove embedded field from selectors (4 fixes)              | LINT FIX     | LOW    | LOW    | 3   |
| 9  | Fix exhaustruct: add new types to exclusion list OR use nolint                      | LINT FIX     | MED    | LOW    | 8   |
| 10 | Fix goconst: extract string column names to constants                               | LINT FIX     | LOW    | LOW    | 5   |
| 11 | Fix golines: break long line in newSQLMembershipReadModel                           | LINT FIX     | LOW    | LOW    | 2   |
| 12 | Fix cyclop: extract helper from NewSQLiteEventSourcedSetup                          | LINT FIX     | MED    | LOW    | 8   |
| 13 | Fix errcheck in tests: check all m.Handle() returns                                 | LINT FIX     | MED    | LOW    | 8   |
| 14 | Fix cyclop in restart test: extract helper                                          | LINT FIX     | LOW    | LOW    | 5   |
| 15 | Run `nix run .#lint` — must show 0 issues                                           | VERIFY       | HIGH   | LOW    | 5   |
| 16 | Run `nix run .#test` — all 4 modules pass with race                                 | VERIFY       | HIGH   | LOW    | 5   |
| 17 | Run `branching-flow errorfamily .` — 0 violations                                   | VERIFY       | HIGH   | LOW    | 3   |
| 18 | Run `nix fmt && nix flake check` — formatting clean                                 | VERIFY       | MED    | LOW    | 3   |
| 19 | Commit fix batch: "fix: resolve 57 lint issues + breaking API + SecurityHooks bug"  | COMMIT       | HIGH   | LOW    | 3   |
| 20 | Update AGENTS.md with corrected v3.1.0 adoption details                             | DOCS         | MED    | LOW    | 8   |
| 21 | Update FEATURES.md / TODO_LIST.md with adoption status                              | DOCS         | LOW    | LOW    | 8   |
| 22 | Push all changes                                                                    | COMMIT       | HIGH   | LOW    | 2   |
| 23 | Review: does the generic SQL read model eliminate the DRY violation?                | REVIEW       | MED    | LOW    | 5   |
| 24 | Review: are all SecurityHooks actually wired end-to-end?                            | REVIEW       | HIGH   | LOW    | 5   |
| 25 | Review: does Postgres setup use Postgres dialect everywhere?                        | REVIEW       | HIGH   | LOW    | 5   |
| 26 | Final full pipeline verification                                                    | VERIFY       | HIGH   | LOW    | 5   |

**Total**: ~160 min (~2.7h) · 0 tasks >12 min

### Dependency chain

```
T1 (revert API) → T2 (config) → T3 (hooks) → T4 (postgres) → T5 (consolidate)
                                                              ↓
T6-T14 (lint fixes, parallel) → T15-T18 (verify) → T19 (commit) → T20-T22 (docs+push)
                                                                               ↓
                                                                         T23-T26 (review+verify)
```
