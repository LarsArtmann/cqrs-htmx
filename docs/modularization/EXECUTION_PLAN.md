# Modularization Execution Plan: cqrs-htmx

> **Status:** Complete — all tasks executed and verified
> **Date:** 2026-05-27
> **Based on:** PROPOSAL.md (2026-05-27 revision)

---

## Summary

Fix remaining module hygiene: upgrade datastar-demo dependency versions, run full test suite, and update documentation. No further module extraction — the coupling analysis confirms the current structure is optimal.

---

## Task List

### Tier 1: Foundational

| #  | Task                                      | Dependencies | Effort | Verification                                | Rollback              |
| -- | ----------------------------------------- | ------------ | ------ | ------------------------------------------- | --------------------- |
| T1 | Fix integration_test go.mod (go mod tidy) | None         | 2 min  | `go build ./...` passes in integration_test | ✅ **Done** (776f101) |

### Tier 2: High Leverage

| #  | Task                                       | Dependencies | Effort | Verification                               | Rollback                                  |
| -- | ------------------------------------------ | ------------ | ------ | ------------------------------------------ | ----------------------------------------- |
| T2 | Upgrade datastar-demo deps to match root   | None         | 5 min  | `go build ./...` passes in datastar-demo   | `git checkout -- examples/datastar-demo/` |
| T3 | Update go.work to include integration_test | T1           | 2 min  | `go work sync` succeeds, all modules build | ✅ **Done**                               |

### Tier 3: Broad Value

| #  | Task                                   | Dependencies | Effort | Verification                         | Rollback                     |
| -- | -------------------------------------- | ------------ | ------ | ------------------------------------ | ---------------------------- |
| T4 | Update CI to test all 4 modules        | T1, T2       | 10 min | CI pipeline passes                   | ✅ **Done**                  |
| T5 | Fix lint warnings                      | None         | 15 min | `golangci-lint run` reports 0 issues | ✅ **Done**                  |
| T6 | Run full test suite across all modules | T2           | 5 min  | All tests pass with race detection   | N/A (read-only verification) |

### Tier 4: Polish

| #  | Task                                          | Dependencies | Effort | Verification                     | Rollback    |
| -- | --------------------------------------------- | ------------ | ------ | -------------------------------- | ----------- |
| T7 | Update AGENTS.md with modularization findings | T6           | 5 min  | AGENTS.md reflects current state | ✅ **Done** |
| T8 | Update modularization docs                    | T6           | 5 min  | Docs reflect final state         | In progress |

---

## Execution Order (Remaining)

1. ~~T1: Fix integration_test go.mod~~ ✅
2. T2: Upgrade datastar-demo deps ← **NEXT**
3. ~~T3: Update go.work~~ ✅
4. ~~T4: Update CI~~ ✅
5. ~~T5: Fix lint warnings~~ ✅
6. T6: Full test suite verification
7. ~~T7: Update AGENTS.md~~ ✅
8. T8: Update docs ← **IN PROGRESS**

---

## Per-Task Detail

### T2: Upgrade datastar-demo deps

Upgrade versions to match root module:

```bash
cd examples/datastar-demo

# Upgrade go directive
# Edit go.mod: change `go 1.26.2` to `go 1.26.3`

# Upgrade direct deps
GOWORK=off go get github.com/larsartmann/go-cqrs-lite/core@latest
GOWORK=off go mod tidy

# Verify
GOWORK=off go build ./...
```

This will also pull in updated indirect deps (go-branded-id, go-error-family) transitively.

### T2b: Upgrade usermgmt deps

**Self-review finding:** usermgmt also has version misalignments:

- Go version: 1.26.2 → 1.26.3
- go-cqrs-lite/core: v1.5.0 → v1.5.1-pre
- go-error-family: v0.1.0 (indirect) → v0.1.1

```bash
cd usermgmt

# Upgrade go directive
# Edit go.mod: change `go 1.26.2` to `go 1.26.3`

# Upgrade direct deps
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go get github.com/larsartmann/go-cqrs-lite/core@latest
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go mod tidy

# Verify
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go build ./...
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race
```

### T6: Full test suite

```bash
# Root
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Usermgmt
cd usermgmt && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Integration test
cd integration_test && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Datastar-demo (build only — main package, no tests)
cd examples/datastar-demo && GOWORK=off go build ./...
```

---

## Risk Assessment

| Risk                                           | Likelihood | Mitigation                                                        |
| ---------------------------------------------- | ---------- | ----------------------------------------------------------------- |
| datastar-demo breaks on newer go-cqrs-lite     | Low        | Demo already uses v1.5.0; upgrading to v1.5.1-pre is a minor bump |
| Version alignment causes indirect dep conflict | Very Low   | `go mod tidy` resolves automatically                              |

---

## Completion Criteria

- [x] integration_test go.mod tidy
- [x] go.work includes integration_test
- [x] CI tests all 4 modules
- [x] Zero lint warnings
- [x] datastar-demo deps aligned with root (migrated to v1.5.1 APIs: command.BasicCommand/query.BasicQuery)
- [x] usermgmt deps aligned with root (go version, go-cqrs-lite)
- [x] Full test suite passes across all modules
- [x] AGENTS.md updated
- [x] Modularization docs updated
