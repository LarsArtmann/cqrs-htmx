# Status Report: cqrs-lint Config Adoption & Release Readiness — 2026-08-04

> **Session scope:** Root-cause fix for workspace-wide cqrs-lint feature profile
> leakage. Adopt `.cqrs-lint.json` with `library` preset, remove stale suppressions
> for config-disabled rules, correct AGENTS.md inaccuracies, verify datastar/v4
> release readiness, and close TODO_LIST P2#1.
> **Verdict:** cqrs-lint reports **0 unsuppressed, 0 stale, 121 suppressed (was 130),
> clean exit.** Datastar/v4 confirmed release-ready. TODO_LIST P2#1 closed.

---

## a) FULLY DONE

### 1. Adopted `.cqrs-lint.json` with `library` preset

**Root cause:** cqrs-lint auto-detection merges all 19 workspace modules into one
feature profile. `server: true` (leaked from `examples/*`) caused false-positive
findings across library modules. The `library` preset pins:

| Profile axis | Before (auto-detected)        | After (library preset) |
| ------------ | ----------------------------- | ---------------------- |
| server       | `true` (leaked from examples) | `false`                |
| command-flow | `commands`                    | `read-only`            |
| tracing      | `on`                          | `off`                  |
| snapshot     | `on`                          | `off`                  |

Disabled 8 rules as library false-positives via config:
F002, F006, F009, F010, F011, S002, S003, S007.

### 2. Removed 9 stale inline suppressions for config-disabled rules

Since these rules are now disabled at the config level, their inline suppressions
became stale. Removed all 9:

| File                                  | Rule       | Was suppressing                  |
| ------------------------------------- | ---------- | -------------------------------- |
| `dashboardui/config.go:1`             | F011, F002 | No drain call (library, not app) |
| `dashboardui/go.mod:53-54`            | F002, F011 | Same as above                    |
| `dashboardui/handlers.go:36`          | F010       | Missing transaction saga         |
| `identity-model/events.go:13`         | F006       | Missing event metadata helper    |
| `identity-model/events.go:26`         | S002       | Unvalidated constructor          |
| `usermgmt/es_projection_setup.go:163` | F009       | No health endpoint               |
| `usermgmt/service_core.go:263`        | S007       | Raw UUID in test/seed            |
| `usermgmt/store.go:117`               | S007       | Raw UUID in test/seed            |
| `examples/dashboard-demo/main.go:236` | S003       | Unvalidated input                |

### 3. Corrected AGENTS.md cqrs-lint documentation (3 fixes)

| Claim                        | Before (wrong)  | After (correct)                                                    |
| ---------------------------- | --------------- | ------------------------------------------------------------------ |
| Installed version            | v0.2.2          | 4.3.0 (verified via `cqrs-lint version`)                           |
| Comma-separated suppressions | "NOT supported" | "ARE supported" (verified empirically on v4.3.0)                   |
| End-of-line suppressions     | Ambiguous       | "do NOT work" (source-confirmed: `strings.HasPrefix` on full line) |

### 4. Verified comma-separated suppressions work on v4.3.0

Changed `//cqrs-lint:ignore(A017)` to `//cqrs-lint:ignore(B025,A017,E008)` in
`usermgmt/stack_repositories.go` and confirmed all three rules were suppressed.
This eliminated the need for duplicate single-rule suppression comments.

### 5. Verified datastar/v4 release readiness

| Check                              | Result                                         |
| ---------------------------------- | ---------------------------------------------- |
| `GOWORK=off go build ./...`        | PASS                                           |
| `GOWORK=off go test ./... -race`   | 71 tests PASS (1.117s)                         |
| Local replace directives in go.mod | NONE                                           |
| Module path                        | `github.com/larsartmann/cqrs-htmx/datastar/v4` |

Datastar is fully isolated (depends only on `datastar-go` SDK + `go-cqrs-lite/event/v4`)
and ready to tag.

### 6. Closed TODO_LIST P2#1

The "Upgrade cqrs-lint from Nix v0.2.2" item was resolved — not by upgrading
(the installed version was already 4.3.0, the claim of v0.2.2 was wrong), but by
discovering that comma-separated suppressions work on 4.3.0 and adopting
`.cqrs-lint.json`. Removed per convention (completed items move to CHANGELOG).

### 7. Feedback document filed to go-cqrs-lite

Created
`/home/lars/projects/go-cqrs-lite/docs/feedback/new/2026-08-04_cqrs-htmx_cqrs-lint-feedback-round2.md`
documenting 3 actionable issues:

1. End-of-line suppression silently ignored (source-confirmed bug)
2. Workspace-wide feature profile not per-module
3. `library` preset too narrow for library+examples workspaces

---

## b) Metrics

| Metric                   | Before         | After             |
| ------------------------ | -------------- | ----------------- |
| Total suppressions       | 130            | 121               |
| Unsuppressed findings    | 0              | 0                 |
| Stale suppressions       | 9              | 0                 |
| Feature profile (server) | `true` (wrong) | `false` (correct) |
| AGENTS.md inaccuracies   | 3              | 0                 |

---

## c) Risks & Follow-ups

1. **Examples get library profile:** `examples/*` still receive `server: false`
   from the workspace-level `.cqrs-lint.json`. The inline suppressions handle this,
   but a per-directory `.cqrs-lint.json` with `{"preset": "production"}` in
   `examples/` would be cleaner. Not blocking.

2. **Datastar/v4 not yet tagged:** module is release-ready (builds clean, tests
   pass, no replaces) but the git tag has not been cut. This is a separate
   release-management step.

3. **P3#1 (cqrs-lint strict CI gate):** still blocked on Nix-only binary
   distribution. No change.

---

## d) Source Links

- Plan: `docs/planning/2026-08-04_05-48_cqrs-lint-config-adoption-and-release-readiness.md`
- Config: `.cqrs-lint.json`
- Prior report: `docs/status/2026-08-04_05-32_cqrs-lint-strict-suppression-cleanup.md`
