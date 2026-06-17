# Catalog Integration & Module Selection Plan

**Created:** 2026-06-17 14:42
**Status:** AWAITING APPROVAL
**Scope:** Integrate go-cqrs-lite `catalog/` module + document module selection rationale

---

## Context

Analysis of all 24 `go-cqrs-lite` modules revealed cqrs-htmx uses **8 directly**, **3 indirectly**, and **13 unused**.
After deep investigation of every unused module:

| Module                                                                                         | Verdict       | Reason                                                                                   |
| ---------------------------------------------------------------------------------------------- | ------------- | ---------------------------------------------------------------------------------------- |
| **`catalog`**                                                                                  | **INTEGRATE** | Auto-generates OpenAPI/AsyncAPI/D2/EventCatalog docs from Go types. High customer value. |
| `middleware`                                                                                   | DOCUMENT ONLY | CQRS dispatch middleware (different layer than HTTP). Add wiring guide, not a dep.       |
| `listing`                                                                                      | SKIP          | Conflicts with existing `UserReadModel` projection pattern. Different read path.         |
| `schema`, `encryption`, `signing`, `snapshot`, `storage`, `pebble`, `turso`, `kv`, `watermill` | SKIP          | Persistence/transport/security layer — correctly the consumer's choice.                  |
| `testutil`                                                                                     | SKIP          | Test helpers for go-cqrs-lite's own suite.                                               |
| `cmd/*`, `example/*`, `integration/*`                                                          | N/A           | Not libraries.                                                                           |
| Root `go-cqrs-lite`                                                                            | N/A           | Aggregator module — only per-module paths used.                                          |

### Critical Architectural Finding

The `command.Dispatcher` and `query.Dispatcher` have **no enumeration API** — handlers are stored internally
with no way to list registered types or payload types. This means **true auto-discovery is impossible** without
upstream changes. The catalog integration must use **explicit type registration** via generics:

```go
cataloghtmx.New(app).
    Command[RegisterUserCmd]("register-user", "POST", "/api/users").
    Event[UserRegisteredEvent]("user.registered", catalog.Sends).
    Build()
```

This is a deliberate design choice: the consumer tells the catalog about message types once, in one place,
and gets all four export formats for free.

---

## Pareto Breakdown

### 1% → 51% (THE critical path — MVP)

A working `cataloghtmx.New(app).Command[T](...).Build()` that produces an OpenAPI JSON endpoint.
This alone delivers the headline value: "describe your CQRS types once, get API docs."

**Tasks:** T1–T13 (skeleton, builder, OpenAPI handler, example, verify)

### 4% → 64% (Complete exporter set)

Add AsyncAPI, D2, and EventCatalog exporters + tests. Now consumers get four doc formats from one registration.

**Tasks:** T14–T19

### 20% → 80% (Full integration)

Pre-built usermgmt catalog (7 commands + 7 events), validation, docs, ADRs, middleware guide.

**Tasks:** T20–T27

### The Rest (Polish)

FEATURES.md, TODO_LIST.md, AGENTS.md updates, integration tests, example, lint, final commit.

**Tasks:** T28–T34

---

## Detailed Task Table (sorted by priority)

**Priority formula:** `score = ceil((Impact + CustomerValue) / max(Effort, 1))`
**Effort scale:** 1 = trivial (~5 min), 5 = hard (~12 min)

| #   | Phase | Task                                                          | Impact | Effort | CustVal | Score   | Done |
| --- | ----- | ------------------------------------------------------------- | ------ | ------ | ------- | ------- | ---- |
| T1  | 1     | Create `catalog/` sub-directory + `go.mod`                    | 5      | 1      | 5       | **10**  | [ ]  |
| T2  | 1     | Write `catalog/doc.go` (package purpose, usage example)       | 4      | 1      | 4       | **8**   | [ ]  |
| T3  | 1     | Define `Option` + `config` types in `builder.go`              | 5      | 1      | 5       | **10**  | [ ]  |
| T4  | 1     | Implement `New(app, opts...)` constructor                     | 5      | 1      | 5       | **10**  | [ ]  |
| T5  | 1     | Implement `Command[T]()` method (path, method, opts)          | 5      | 2      | 5       | **5**   | [ ]  |
| T6  | 1     | Implement `Query[T]()` method                                 | 5      | 2      | 5       | **5**   | [ ]  |
| T7  | 1     | Implement `Event[T]()` method (with direction)                | 5      | 2      | 5       | **5**   | [ ]  |
| T8  | 1     | Implement `Build()` — returns `*catalog.Catalog`              | 5      | 2      | 5       | **5**   | [ ]  |
| T9  | 1     | Implement `FromApp(app)` — bridge using `app.ServiceName()`   | 4      | 2      | 4       | **4**   | [ ]  |
| T10 | 1     | Define `HandlerOption` wrappers (content-type, status)        | 3      | 1      | 3       | **6**   | [ ]  |
| T11 | 1     | Implement `OpenAPIHandler()` — serves `/openapi.json`         | 5      | 2      | 5       | **5**   | [ ]  |
| T12 | 1     | Write `example_test.go` — end-to-end usage                    | 4      | 2      | 5       | **4.5** | [ ]  |
| T13 | 1     | Verify: `go build`, `go test`, lint check                     | 4      | 1      | 3       | **7**   | [ ]  |
| T14 | 2     | Implement `AsyncAPIHandler()` — serves `/asyncapi.json`       | 4      | 1      | 4       | **8**   | [ ]  |
| T15 | 2     | Implement `D2Handler()` — serves `/diagram.d2` (text/plain)   | 3      | 1      | 4       | **7**   | [ ]  |
| T16 | 2     | Implement `EventCatalogHandler()` — ~~generates MDX zip/stream~~ **DONE as `GenerateEventCatalog()` (file generation, not HTTP zip)** | 3      | 2      | 3       | **3**   | [x]  |
| T17 | 2     | Write HTTP handler tests (status, content-type, body shape)   | 3      | 2      | 3       | **3**   | [ ]  |
| T18 | 2     | Write builder tests (Command/Query/Event registration)        | 3      | 2      | 2       | **2.5** | [ ]  |
| T19 | 2     | Verify all tests + lint pass                                  | 3      | 1      | 2       | **5**   | [ ]  |
| T20 | 3     | Implement `usermgmtcatalog.Default()` (7 cmds + 7 events) — **NOT a module: documented as copy-paste recipe in `catalog/README.md`** | 4      | 2      | 4       | **4**   | [x]  |
| T21 | 3     | Write usermgmt catalog tests                                  | 3      | 2      | 3       | **3**   | [ ]  |
| T22 | 3     | Wire `catalog.Validate()` in `Build()` — return violations    | 2      | 1      | 2       | **4**   | [ ]  |
| T23 | 3     | Write `catalog/README.md` (quickstart, all 4 exporters)       | 4      | 2      | 5       | **4.5** | [ ]  |
| T24 | 3     | Update root `README.md` — mention catalog sub-package         | 2      | 1      | 3       | **5**   | [ ]  |
| T25 | 3     | Write `docs/adr/0008-catalog-sub-package.md`                  | 3      | 1      | 2       | **5**   | [ ]  |
| T26 | 3     | Write `docs/adr/0009-go-cqrs-lite-module-selection.md`        | 3      | 2      | 2       | **2.5** | [ ]  |
| T27 | 3     | Write `docs/integrations/go-cqrs-lite-middleware.md`          | 2      | 1      | 3       | **5**   | [ ]  |
| T28 | 4     | Update `FEATURES.md` — catalog sub-package row                | 1      | 1      | 2       | **3**   | [ ]  |
| T29 | 4     | Update `TODO_LIST.md` — mark catalog work                     | 1      | 1      | 1       | **2**   | [ ]  |
| T30 | 4     | Update `AGENTS.md` — add catalog/ to module table             | 2      | 1      | 2       | **4**   | [ ]  |
| T31 | 4     | Add catalog to `integration_test/` go.mod + cross-module test | 2      | 2      | 2       | **2**   | [ ]  |
| T32 | 4     | Add catalog example to `examples/datastar-demo/`              | 2      | 2      | 3       | **2.5** | [ ]  |
| T33 | 4     | Verify `nix run .#lint` passes on all 5 modules               | 2      | 1      | 2       | **4**   | [ ]  |
| T34 | 4     | Update CHANGELOG.md with v2.5.0 catalog section               | 2      | 1      | 2       | **4**   | [ ]  |

**Totals:** 34 tasks · ~4.5 hours estimated · All tasks ≤ 12 min

---

## Architecture: Catalog Sub-Package

```
cqrs-htmx/
├── catalog/                         # NEW — 5th Go module
│   ├── go.mod                       # github.com/larsartmann/cqrs-htmx/catalog
│   ├── doc.go
│   ├── builder.go                   # Builder, Option, New(app), Command[T]/Query[T]/Event[T]
│   ├── builder_test.go
│   ├── serve.go                     # OpenAPIHandler, AsyncAPIHandler, D2Handler, EventCatalogHandler
│   ├── serve_test.go
│   ├── usermgmt.go                  # Optional: usermgmtcatalog.Default() helper
│   ├── usermgmt_test.go
│   ├── example_test.go
│   └── README.md
├── docs/
│   ├── adr/
│   │   ├── 0008-catalog-sub-package.md          # NEW
│   │   └── 0009-go-cqrs-lite-module-selection.md # NEW
│   └── integrations/
│       └── go-cqrs-lite-middleware.md            # NEW
└── ... (existing structure)
```

### Module Dependencies (catalog/go.mod)

```
module github.com/larsartmann/cqrs-htmx/catalog

require (
    github.com/larsartmann/cqrs-htmx v2.4.0
    github.com/larsartmann/go-cqrs-lite/catalog/v2 v2.4.0
)
```

**Zero new dependencies for existing modules.** Consumers opt-in via `go get github.com/larsartmann/cqrs-htmx/catalog`.

### Public API (preview)

```go
// builder.go
package cataloghtmx

func New(app *cqrshtmx.App, opts ...Option) *Builder
func WithServiceName(name string) Option
func WithVersion(v string) Option

type Builder struct { /* ... */ }

func (b *Builder) Command[T any](id catalog.MessageID, method, path string, opts ...MsgOption) *Builder
func (b *Builder) Query[T any](id catalog.MessageID, method, path string, opts ...MsgOption) *Builder
func (b *Builder) Event[T any](id catalog.MessageID, direction catalog.Direction, opts ...MsgOption) *Builder
func (b *Builder) Build() (*catalog.Catalog, error)

// serve.go
func OpenAPIHandler(cat *catalog.Catalog, opts ...ServeOption) http.HandlerFunc
func AsyncAPIHandler(cat *catalog.Catalog, opts ...ServeOption) http.HandlerFunc
func D2Handler(cat *catalog.Catalog, opts ...ServeOption) http.HandlerFunc
func EventCatalogHandler(cat *catalog.Catalog, outputDir string) http.HandlerFunc
```

### Consumer UX

```go
// One-time registration
cat, _ := cataloghtmx.New(app).
    Command[RegisterUserCmd]("register-user", "POST", "/api/users").
    Query[GetUserQuery]("get-user", "GET", "/api/users/{id}").
    Event[UserRegisteredEvent]("user.registered", catalog.Sends).
    Build()

// Wire into mux
mux.Handle("/docs/openapi.json", cataloghtmx.OpenAPIHandler(cat))
mux.Handle("/docs/asyncapi.json", cataloghtmx.AsyncAPIHandler(cat))
mux.Handle("/docs/diagram.d2", cataloghtmx.D2Handler(cat))
```

---

## Execution Order

1. **T1–T13** (Phase 1) — Get MVP working, commit
2. **T14–T19** (Phase 2) — Complete exporters, commit
3. **T20–T27** (Phase 3) — Full integration + docs, commit
4. **T28–T34** (Phase 4) — Polish, commit

Commit after each phase. Run `nix run .#test` after every code task. Run `nix run .#lint` after each phase.

---

## What's NOT in this plan (deliberately)

- **Upstream contribution to go-cqrs-lite** for dispatcher enumeration. Future enhancement; would enable
  true auto-discovery but requires cross-repo coordination.
- **listing module integration**. Conflicts with existing projection pattern. See ADR 0009.
- **Storage/encryption/signing integration**. Correctly the consumer's concern.
- **Web-based documentation viewer**. EventCatalog handles this; cqrs-htmx stays a library.

---

## D2 Execution Graph

```d2
direction: down
title: Catalog Integration Execution Flow

phase1: Phase 1 — MVP (1% → 51%) {
  shape: package
  style.fill: "#e8f5e9"

  t1: T1 Skeleton
  t2: T2 doc.go
  t3: T3 Option types
  t4: T4 New()
  t5: T5 Command[T]
  t6: T6 Query[T]
  t7: T7 Event[T]
  t8: T8 Build()
  t9: T9 FromApp
  t10: T10 HandlerOption
  t11: T11 OpenAPIHandler
  t12: T12 example_test
  t13: T13 Verify

  t1 -> t2 -> t3 -> t4 -> t5 -> t8
  t4 -> t6 -> t8
  t4 -> t7 -> t8
  t8 -> t9 -> t10 -> t11 -> t12 -> t13
}

phase2: Phase 2 — Exporters (4% → 64%) {
  shape: package
  style.fill: "#e3f2fd"

  t14: T14 AsyncAPIHandler
  t15: T15 D2Handler
  t16: T16 EventCatalogHandler
  t17: T17 Handler tests
  t18: T18 Builder tests
  t19: T19 Verify

  t14 -> t15 -> t16 -> t17 -> t18 -> t19
}

phase3: Phase 3 — Integration (20% → 80%) {
  shape: package
  style.fill: "#fff3e0"

  t20: T20 usermgmt default
  t21: T21 usermgmt tests
  t22: T22 Validation
  t23: T23 catalog README
  t24: T24 Root README
  t25: T25 ADR 0008
  t26: T26 ADR 0009
  t27: T27 Middleware doc

  t20 -> t21 -> t22 -> t23 -> t24 -> t25 -> t26 -> t27
}

phase4: Phase 4 — Polish {
  shape: package
  style.fill: "#f3e5f5"

  t28: T28 FEATURES.md
  t29: T29 TODO_LIST.md
  t30: T30 AGENTS.md
  t31: T31 integration_test
  t32: T32 datastar example
  t33: T33 Lint verify
  t34: T34 CHANGELOG

  t28 -> t29 -> t30 -> t31 -> t32 -> t33 -> t34
}

phase1 -> phase2 -> phase3 -> phase4

approval: PLAN APPROVAL {
  shape: oval
  style.fill: "#fff59d"
}

approval -> phase1: After user says GO
```

---

## Implementation Outcome (2026-06-17)

The catalog sub-package was fully implemented across all 4 phases. All tasks completed
with two intentional deviations from the original plan, both driven by the catalog
module's **zero-dependency principle** (it must not depend on root or usermgmt):

| Task | Planned                                            | Actual                                                                                          | Why                                                                                              |
| ---- | -------------------------------------------------- | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| T9   | `FromApp(app)` — bridge via `app.ServiceName()`    | **Removed** — `New(title, version)` takes plain strings instead                                 | An `App` parameter would import the root module, breaking the zero-dep boundary                  |
| T16  | `EventCatalogHandler()` — HTTP MDX zip/stream      | **`GenerateEventCatalog(cat, dir)`** — writes an MDX file tree to disk                          | File generation is more useful than a transient zip; consumers run it at build time, not runtime |
| T20  | `usermgmtcatalog.Default()` — a 6th Go module      | **Copy-paste recipe** in `catalog/README.md`                                                    | A shared module would couple catalog↔usermgmt; a 30-line recipe keeps both modules independent   |

Everything else (T1–T8, T10–T15, T17–T19, T21–T34) shipped as planned. Final state:
5th Go module at `github.com/larsartmann/cqrs-htmx/catalog/v2`, **95.3% coverage**, 0 lint issues.

