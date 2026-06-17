# ADR 0007: Dependency Upgrade to v2.4.0 and Idiomatic Error-Family Adoption

**Date:** 2026-06-17
**Status:** Accepted

## Context

Three upstream releases landed in close succession and our `go.mod` files were
pinned to the previous generation:

| Dependency      | Was     | Now     |
| --------------- | ------- | ------- |
| go-cqrs-lite    | v2.3.0  | v2.4.0  |
| go-error-family | v0.3.0  | v0.4.0  |
| go-branded-id   | v0.3.0  | v0.3.1  |
| ginkgo/v2       | v2.30.0 | v2.31.0 |
| gomega          | v1.41.0 | v1.42.0 |

Two pressures motivated the upgrade:

1. **go-error-family v0.4.0 introduced idiomatic wrapping APIs** — `event.Wrapf`
   and `event.WrapTransient` — that replace the verbose
   `event.NewTransient(fmt.Sprintf(...)).WithCause(err)` pattern we used in every
   authorization error site. Adopting them removes `fmt` imports, deletes the
   `policyWrapErr` helper, and makes error construction match the library's
   intended usage.

2. **go-cqrs-lite v2.4.0 ships performance optimizations and new sub-packages**
   (reactive APIs, a `kv` module, expanded `otel`/`schema` coverage). Even where
   we do not yet consume the new features, pinning the latest tags keeps the four
   modules consistent and removes the last v2.3.x transitive edges.

Separately, the codebase carried **16 lint warnings** (2 `errorlint`, 2
`exhaustruct`, 12 `contextcheck`) that were either genuine bugs (`%s` instead of
`%w` on inner errors) or false positives requiring configuration.

## Decision

Upgrade all four Go modules (root, `usermgmt`, `integration_test`,
`datastar-demo`) to the versions above and adopt the new APIs idiomatically.

### 1. go-error-family v0.4.0: `Wrapf` / `WrapTransient` / `Newf` / `IsRetryable`

- **`event.Wrapf` / `event.WrapTransient`**: Every
  `event.NewTransient(fmt.Sprintf(...)).WithCause(err)` and
  `event.NewRejection(fmt.Sprintf(...)).WithCause(err)` site in
  `authz_policies.go`, `authz_types.go`, and `authz_roles.go` now uses the
  idiomatic wrapping constructors. The `policyWrapErr` helper and all `fmt`
  imports in those files were deleted.
- **`event.Newf`**: The panic-recovery handler in `recovery.go` uses
  `event.Newf(event.Infrastructure, ...)` instead of `fmt.Sprintf`.
- **`event.IsRetryable`**: Test code in `coverage_gaps_test.go` uses
  `event.IsRetryable(err)` instead of manual `event.Classify(err) == event.Transient`.

### 2. go-cqrs-lite v2.4.0: pin latest per-module tags

- All `go.mod` files declare `v2.4.0` for every consumed sub-package
  (`command`, `event`, `id`, `query`, `codec`, `dispatcher`, `memory`,
  `snapshot`). No `replace` directives are needed.
- New sub-packages (`kv`, reactive APIs, expanded `otel`) are available to
  consumers but are not yet wired into `cqrs-htmx`. Adopting them is deferred to
  a future ADR once concrete use cases arise.

### 3. Lint: 16 → 0

| Linter         | Fix                                                                                                                                                                              |
| -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `errorlint`    | `%s` → `%w` for inner errors in `email.go` and `import_export.go` so inner errors remain wrappable.                                                                              |
| `exhaustruct`  | Third-party `pquerna/otp` struct literals excluded via `.golangci.yml` rule instead of inline `nolint` comments.                                                                 |
| `contextcheck` | False positives from the `withTimeout` helper excluded at the file level — the helper correctly threads `r.Context()`, but the linter cannot trace across the function boundary. |

### 4. Email validation consolidation

`ImportUser.Validate()` now delegates to `ParseEmail()` instead of reimplementing
identical format/length logic, eliminating a duplicate-validation clone.

## Rationale

### Why adopt `Wrapf` instead of keeping `NewTransient(...).WithCause`?

The two-step `NewTransient(fmt.Sprintf(...)).WithCause(err)` is verbose, forces a
`fmt` import purely for error formatting, and hides intent behind string
formatting. `WrapTransient(format, args...)` (and its rejection sibling
`Wrapf`) says exactly what it does in one call. Removing the `policyWrapErr`
indirection means readers see the error family and cause inline.

### Why fix `errorlint` with `%w` instead of silencing it?

The two `errorlint` hits were real: wrapping an inner error with `%s` breaks
`errors.Is`/`errors.As` chains for consumers. `%w` is the correct verb and costs
nothing.

### Why exclude `pquerna/otp` structs and `withTimeout` at config level?

`exhaustruct` fires on every `pquerna/otp` literal because we intentionally omit
optional fields the library populates with defaults. Inline `nolint` comments
would clutter every call site; a single `.golangci.yml` rule is cleaner.
`contextcheck` cannot follow `r.Context()` through the `withTimeout` helper, so
every call site would emit a false positive — a file-level exclusion documents
the limitation once.

### Why pin v2.4.0 before consuming the new features?

Consistency and supply-chain hygiene: all four modules sharing one pin removes
mixed-version edges and makes future feature adoption a single-version bump. We
adopt the APIs we need now (error wrapping) and defer the rest.

## Consequences

- All `go.mod` files now share a single go-cqrs-lite version (`v2.4.0`).
- Authorization error construction is shorter and `fmt`-free in the authz layer.
- Lint is at **0 issues** across all modules; the CLI is authoritative when the
  LSP cache shows stale warnings (see `AGENTS.md` gotcha #6).
- `ImportUser.Validate()` and `ParseEmail()` are now a single source of truth for
  email validation.
- Consumers depending on the old `policyWrapErr` helper are unaffected — it was
  unexported.
- New go-cqrs-lite v2.4.0 features (`kv`, reactive APIs) remain unused; a future
  ADR should evaluate them against concrete needs.

## References

- `docs/status/2026-06-17_12-19_dependency-upgrade-and-lint-zero-status.md` — the session that completed the upgrade
- Commit `f131b20` — dependency version bumps across all modules
- Commit `25ab685` — idiomatic `event.Wrapf`/`WrapTransient` adoption and lint elimination
- [ADR 0005](0005-go-cqrs-lite-v230-adoption.md) — prior go-cqrs-lite v2.3.0 adoption
- `authz_policies.go`, `authz_types.go`, `authz_roles.go` — `Wrapf`/`WrapTransient` call sites
- `recovery.go` — `event.Newf` panic handler
- `.golangci.yml` — `exhaustruct` and `contextcheck` exclusion rules
