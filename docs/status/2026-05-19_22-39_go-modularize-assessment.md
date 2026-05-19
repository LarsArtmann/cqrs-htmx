# Go Modularize Assessment — cqrs-htmx

**Date:** 2026-05-19_22-39

## Current State

| Property         | Value                                               |
| ---------------- | --------------------------------------------------- |
| Go modules       | 2 (root + usermgmt)                                 |
| go.work          | Yes, at root, includes `.` and `./usermgmt`         |
| Root packages    | 1 flat package (`github.com/larsartmann/cqrs-htmx`) |
| Production files | 15                                                  |
| Exported symbols | ~100                                                |

### Module Landscape

| Module               | Path         | Direct Deps                                                                       | Internal Deps              | Replace |
| -------------------- | ------------ | --------------------------------------------------------------------------------- | -------------------------- | ------- |
| `cqrs-htmx`          | `./`         | go-cqrs-lite/core, casbin/v3, gorilla/csrf, cockroachdb/errors, golang.org/x/time | 0                          | None    |
| `cqrs-htmx/usermgmt` | `./usermgmt` | casbin/v3, golang.org/x/crypto                                                    | 0 (does NOT import parent) | None    |

### Starting State Classification: **Partial split** — usermgmt already has its own go.mod. Root is a flat single-package library.

## Existing Proposal Review

A modularization proposal exists at `docs/modularization/PROPOSAL.md` (dated 2026-05-14). It proposed splitting the root into 4 sub-modules:

1. `cqrs-htmx` (root) — App builder, dispatch pipeline
2. `cqrs-htmx/htmx` — HTMX request/response (zero external deps)
3. `cqrs-htmx/authz` — Authorization (only cockroachdb/errors)
4. `cqrs-htmx/middleware` — Middleware chain (zero external deps)

## Assessment: Recommendation to NOT Proceed

### Why the split is NOT recommended now

1. **Library, not application**: This is a library consumers import. Splitting into sub-modules means consumers must manage multiple import paths (`cqrs-htmx`, `cqrs-htmx/htmx`, `cqrs-htmx/authz`). For a ~5000 LOC library, this adds friction without proportional benefit.

2. **Tight coupling in dispatch pipeline**: `handler.go` uses `authMode`, `Enforcer`, `UserID`, `HTMXResponse`, `ErrorHandler`, `CSRFConfig` — all from different conceptual areas. Splitting these into modules would require the root module to import ALL sub-modules, defeating the isolation goal.

3. **Flat package = Go convention for small libraries**: The Go standard library and most successful Go libraries (e.g., `chi`, `gorilla/mux`) use a flat package for cohesive APIs. Splitting is only justified when the module has >10K LOC or genuinely independent subsystems.

4. **The 80-symbol surface is manageable**: All symbols are in one namespace, but they're well-named with clear prefixes (CSRF*, HTMX*, Notify*, Authorize*, Enforce\*). IDE autocompletion handles this fine.

5. **usermgmt split was correct**: The usermgmt submodule has a genuinely separate concern (user management with bcrypt, sessions, RBAC). It has its own types and doesn't import the parent. This is the right granularity for a sub-module.

### What SHOULD be done instead

1. **Keep the flat package** — it's the right structure for this library size
2. **Split `csrf.go` (445 lines) into two files**: `csrf.go` (middleware + config) and `csrf_helpers.go` (template helpers) — package-level split, not module-level
3. **Address the duplication from the code quality scan** — this improves the flat package without splitting

### When to reconsider modularization

- Library exceeds 10K LOC
- New subsystem with zero overlap (e.g., SSE/WebSocket helpers)
- Consumer feedback about dependency overreach
- Extracting `htmx` as a standalone library for non-CQRS use cases
