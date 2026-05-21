# Modularization Execution Plan: cqrs-htmx

> **Status:** Phase 5 — Ready for execution
> **Date:** 2026-05-22
> **Based on:** PROPOSAL.md (2026-05-22 revision)

---

## Summary

Fix module hygiene across all 4 Go modules: tidy integration_test, upgrade datastar-demo deps, update go.work, update CI, and fix lint warnings. No further module extraction — the coupling analysis shows it's not warranted.

---

## Task List

### Tier 1: Foundational (1% → 51% impact)

| #  | Task | Dependencies | Effort | Verification | Rollback |
|----|------|-------------|--------|-------------|----------|
| T1 | Fix integration_test go.mod (go mod tidy) | None | 2 min | `go build ./...` passes in integration_test | `git checkout -- integration_test/go.mod integration_test/go.sum` |
| T2 | Upgrade datastar-demo deps to match root | None | 5 min | `go build ./...` passes in datastar-demo | `git checkout -- examples/datastar-demo/go.mod examples/datastar-demo/go.sum` |

### Tier 2: High Leverage (4% → 64% impact)

| #  | Task | Dependencies | Effort | Verification | Rollback |
|----|------|-------------|--------|-------------|----------|
| T3 | Update go.work to include integration_test | T1 | 2 min | `go work sync` succeeds, all modules build | `git checkout -- go.work go.work.sum` |
| T4 | Update CI to test all 4 modules | T1, T2 | 10 min | CI pipeline passes | `git checkout -- .github/workflows/ci.yml` |

### Tier 3: Broad Value (20% → 80% impact)

| #  | Task | Dependencies | Effort | Verification | Rollback |
|----|------|-------------|--------|-------------|----------|
| T5 | Fix 10 lint warnings (revive, errcheck, forcetypeassert, recvcheck, exhaustruct, noctx) | None | 15 min | `golangci-lint run` reports 0 issues | Fix forward or `git checkout -- <files>` |
| T6 | Run full test suite across all modules | T1, T2, T3 | 5 min | All tests pass with race detection | N/A (read-only verification) |

### Tier 4: Polish

| #  | Task | Dependencies | Effort | Verification | Rollback |
|----|------|-------------|--------|-------------|----------|
| T7 | Update AGENTS.md with modularization findings | T6 | 5 min | AGENTS.md reflects current state | `git checkout -- AGENTS.md` |
| T8 | Update modularization docs | T6 | 5 min | Docs reflect final state | `git checkout -- docs/modularization/` |

---

## Execution Order (Linear)

1. T1: Fix integration_test go.mod
2. T2: Upgrade datastar-demo deps
3. T3: Update go.work
4. T5: Fix lint warnings
5. T6: Run full test suite
6. T4: Update CI
7. T7: Update AGENTS.md
8. T8: Update docs

---

## Per-Task Detail

### T1: Fix integration_test go.mod

```bash
cd integration_test && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go mod tidy
cd integration_test && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go build ./...
cd integration_test && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race
```

### T2: Upgrade datastar-demo deps

Upgrade go-cqrs-lite/core from v1.5.0 to match root's v1.4.0? Actually, root uses v1.4.0 but demo uses v1.5.0 (newer). The demo should use the newer version. Root should upgrade too eventually. For now, just ensure the demo builds.

```bash
cd examples/datastar-demo && GOWORK=off go mod tidy
cd examples/datastar-demo && GOWORK=off go build ./...
```

### T3: Update go.work

Add integration_test to go.work:

```go
go 1.26.2

use (
    .
    ./usermgmt
    ./integration_test
)
```

Do NOT add datastar-demo (it doesn't import sibling modules).

### T5: Fix lint warnings

10 warnings to fix:
1. logging.go:150 — WriteHeader missing doc
2. logging.go:158 — Push missing doc
3. logging.go:213 — Flush missing doc
4. logging.go:219 — Hijack missing doc
5. httputil.go:15 — errcheck on json.Encode
6. ratelimit.go:240 — forcetypeassert on heap.Pop
7. ratelimit.go:260 — forcetypeassert in Push
8. ratelimit.go:255 — recvcheck mixed receivers
9. logging_test.go:235 — exhaustruct mockPusher
10. testing_test.go:280 — noctx httptest.NewRequest

### T4: Update CI

Add steps for:
- Build + test integration_test
- Build datastar-demo

### T6: Full test suite

```bash
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race
cd usermgmt && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race
cd integration_test && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race
cd examples/datastar-demo && GOWORK=off go build ./...
```

---

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| go.work breaks consumer builds | Low | go.work ignored by consumers; only affects local dev |
| datastar-demo v1.5.0 has breaking changes from v1.4.0 | Low | Demo already builds and runs on v1.5.0 |
| integration_test replace directives conflict with go.work | Medium | Remove replace directives from integration_test after adding to go.work |
| CI changes break pipeline | Low | Test locally first |
