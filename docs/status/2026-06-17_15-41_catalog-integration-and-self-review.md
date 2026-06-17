# Status Report — 2026-06-17

**Date:** 2026-06-17 15:41
**Branch:** master (clean, pushed)
**Version:** v2.4.0 (+ unreleased catalog sub-package)
**Session scope:** go-cqrs-lite module analysis → catalog integration → brutal self-review → fixes

---

## Project at a Glance

| Metric                      | Value                                                        |
| --------------------------- | ------------------------------------------------------------ |
| Go modules                  | 5 (root, usermgmt, integration_test, catalog, datastar-demo) |
| Go files                    | 207                                                          |
| Lines of Go code            | ~29,300                                                      |
| Tests passing               | 487 (123 root + 27 catalog + 328 usermgmt + 9 integration)   |
| Root coverage               | 96.4%                                                        |
| usermgmt coverage           | 84.1%                                                        |
| catalog coverage            | 84.9%                                                        |
| Lint issues                 | 0 (all modules)                                              |
| ADRs                        | 9                                                            |
| Features (FULLY_FUNCTIONAL) | 70                                                           |
| Open TODOs                  | 5                                                            |
| Race detector               | Clean (all modules)                                          |

---

## a) FULLY DONE

### This Session

1. **Module analysis** — Investigated all 24 go-cqrs-lite modules. Determined 8 used directly, 3 indirect, 13 unused. Documented rationale in ADR 0009.
2. **Catalog sub-package** (5th Go module) — Complete API documentation generation:
   - Builder API with `Command[T]`, `Query[T]`, `Event[T]` generic functions
   - HTTP handlers: `OpenAPIHandler`, `AsyncAPIHandler`, `D2Handler`, `HealthCheckHandler`
   - JSON + YAML output via `WithFormat()`
   - Schema auto-derivation from struct tags
   - Catalog validation (`Build()` panics, `BuildValid()` returns violations)
   - 27 tests, 84.9% coverage, 0 lint issues
   - README with full API reference + quickstart
3. **ADRs 0008 + 0009** — Catalog sub-package decision + module selection rationale
4. **Middleware integration guide** — How go-cqrs-lite dispatch middleware composes with cqrs-htmx HTTP middleware
5. **Integration test** — Cross-module catalog generation with real `App` instance
6. **Brutal self-review fixes**:
   - Removed lying `EventCatalogHandler` ghost system (claimed ZIP, served JSON)
   - Fixed flake.nix split brain (catalog missing from test/lint/build/coverage)
   - Removed self-referential `replace` directive in catalog/go.mod
   - Fixed 7 lint issues (errchkjson, exhaustive, exhaustruct, forcetypeassert, gci)
   - Fixed `errors.Join` → `fmt.Errorf("...: %w", err)`

### Pre-Session (Stable)

- Event-sourced passwordless usermgmt (7 events, 7 commands, WebAuthn, TOTP, email verification)
- SSE + WebSocket protocol helpers
- CSRF (nosurf), rate limiting, security headers, panic recovery
- HTMX response builder, embedded HTMX JS, notifications
- Pagination (`DecodePagination` + `RenderPaginatedJSON[T]`)
- 9 ADRs documenting all major architectural decisions

---

## b) PARTIALLY DONE

| Item                           | Status                                                         | Gap                                                                                                                           |
| ------------------------------ | -------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| **usermgmt pre-built catalog** | Catalog sub-package exists, but no `usermgmtcatalog.Default()` | Consumers must manually register all 7 commands + 7 events. Deliberately deferred — would create usermgmt→catalog dependency. |
| **nix flake check**            | `nix flake check` passes for formatting + devShells + apps     | catalog module added to flake.nix but `nix flake check` not re-run this session after the edit                                |
| **flake.nix version**          | `packages.default` still says `version = "2.3.0"` (line 43)    | Should be 2.4.0 or unreleased                                                                                                 |
| **go.work vs flake.nix**       | go.work has catalog, flake.nix now has catalog                 | Aligned now, but was a split brain for ~30 min                                                                                |

---

## c) NOT STARTED

| Item                                 | Priority | Notes                                                    |
| ------------------------------------ | -------- | -------------------------------------------------------- |
| OAuth2/OIDC integration              | Medium   | Alternative auth to WebAuthn. No work done.              |
| Event schema versioning              | Medium   | Version field on events for future migrations            |
| CSRF on WebAuthn endpoints           | Low      | Not wired by default (library principle)                 |
| Rate limiting on WebAuthn endpoints  | Low      | Not wired by default (library principle)                 |
| Dispatcher enumeration API           | Low      | Upstream go-cqrs-lite change needed for auto-discovery   |
| `usermgmtcatalog.Default()`          | Medium   | Pre-built catalog for all usermgmt types                 |
| Swagger UI / AsyncAPI Studio serving | Low      | Consumers wire their own UI; we only serve the JSON/YAML |

---

## d) TOTALLY FUCKED UP (Fixed This Session)

| What                         | Severity | How                                                                                | Fixed?                                                |
| ---------------------------- | -------- | ---------------------------------------------------------------------------------- | ----------------------------------------------------- |
| **EventCatalogHandler lied** | CRITICAL | Set `Content-Type: application/zip` but served JSON file listing                   | YES — removed entirely, kept `GenerateEventCatalog()` |
| **flake.nix split brain**    | CRITICAL | `nix run .#test` silently skipped catalog module                                   | YES — catalog added to all 4 multi-module apps        |
| **Self-referential replace** | MEDIUM   | `catalog/go.mod` had `replace github.com/larsartmann/cqrs-htmx/catalog/v2 => ./`   | YES — removed                                         |
| **errors.Join anti-pattern** | LOW      | Used `errors.Join(errors.New("..."), err)` instead of `fmt.Errorf("...: %w", err)` | YES                                                   |
| **7 lint issues**            | MEDIUM   | errchkjson, exhaustive, exhaustruct×3, forcetypeassert, gci                        | YES — all fixed, 0 issues                             |
| **Duplicate import**         | LOW      | example_test.go imported the same package twice with different aliases             | YES                                                   |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **`toKebab` function is hand-rolled** — Should use an established case-conversion library. The catalog module upstream already has `caseutil` — we duplicate the logic. If go-cqrs-lite exposes `caseutil`, use it. Otherwise consider `samber/lo` or `golang.org/x/text/cases`.
2. **No `.golangci.yml` for catalog module** — Uses the root config by directory inheritance. This works but is fragile — a `cd catalog && golangci-lint run` picks up root config. Consider adding a catalog-specific config matching usermgmt pattern.
3. **`flake.nix` version is stale** — `packages.default` says `2.3.0` but the project is on `2.4.0` + unreleased changes.

### Testing

4. **Catalog coverage is 84.9%** — Good but below the 90%+ standard of root. Missing: `BuildValid()` error paths, `HealthCheckHandler` unhealthy path body assertions, `WithDescription` option effect verification.
5. **No snapshot/golden tests for catalog output** — The OpenAPI/AsyncAPI/D2 output should have golden files to catch breaking changes in the generated specs. go-cqrs-lite's own catalog module uses golden tests extensively.
6. **No BDD tests for catalog** — Root and usermgmt use Ginkgo/Gomega BDD style. Catalog tests are plain `testing.T`.

### Documentation

7. **No `go(pkg.go.dev)` badge link in catalog README** — Other modules have pkg.go.dev badges. Catalog README has one but it points to the right URL — verify it resolves once published.
8. **Catalog doc.go example doesn't show `BuildValid()`** — Only shows `Build()` (panic variant). Should show both.
9. **ADR 0009 doesn't mention the `middleware` integration doc** — The doc exists at `docs/integrations/go-cqrs-lite-middleware.md` but the ADR only says "DOCUMENT ONLY" for middleware. Should cross-reference.

### Process

10. **`nix fmt` not run after flake.nix edits** — The flake.nix edits this session were not formatted with `nix fmt`. BuildFlow's `nix-fmt` check passed, but we should verify.
11. **CHANGELOG.md `[Unreleased]` doesn't mention the self-review fixes** — Only mentions the initial catalog addition, not the ghost system removal or lint fixes.

---

## f) Top 25 Things to Get Done Next

Sorted by impact/effort ratio (highest first).

| #   | Task                                                                            | Impact | Effort  | Ratio      |
| --- | ------------------------------------------------------------------------------- | ------ | ------- | ---------- |
| 1   | Fix `flake.nix` version from `2.3.0` → `2.4.0`                                  | Medium | Trivial | **∞**      |
| 2   | Update CHANGELOG `[Unreleased]` with self-review fixes                          | Medium | Trivial | **∞**      |
| 3   | Run `nix fmt` to verify all formatting                                          | Low    | Trivial | **High**   |
| 4   | Run `nix flake check` after flake.nix edits                                     | Low    | Trivial | **High**   |
| 5   | Add golden/snapshot tests for catalog OpenAPI output                            | High   | Medium  | **High**   |
| 6   | Catalog coverage → 90%+ (test BuildValid errors, HealthCheck body)              | Medium | Low     | **High**   |
| 7   | `usermgmtcatalog.Default()` — pre-built catalog for 7 cmds + 7 events           | High   | Medium  | **High**   |
| 8   | Bump `flake.nix` `packages.default` version to match release                    | Medium | Trivial | **High**   |
| 9   | Cross-reference ADR 0009 → middleware integration doc                           | Low    | Trivial | **Medium** |
| 10  | Add catalog to `CONTRIBUTING.md` build/test/lint section                        | Low    | Trivial | **Medium** |
| 11  | Consider `lo.KebabCase` or upstream `caseutil` instead of hand-rolled `toKebab` | Medium | Low     | **Medium** |
| 12  | Add BDD test for catalog builder (Ginkgo/Gomega to match project style)         | Medium | Medium  | **Medium** |
| 13  | Swagger UI handler — serve OpenAPI + embedded Swagger UI HTML                   | High   | Medium  | **Medium** |
| 14  | `usermgmtcatalog` integration test — verify catalog matches actual endpoints    | High   | Low     | **Medium** |
| 15  | Catalog example in `examples/datastar-demo/`                                    | Medium | Medium  | **Medium** |
| 16  | Event schema versioning — `SchemaVersion` field on event payloads               | High   | High    | **Medium** |
| 17  | OAuth2/OIDC integration as alternative to WebAuthn                              | High   | High    | **Medium** |
| 18  | CSRF protection wiring helper for WebAuthn endpoints                            | Medium | Low     | **Medium** |
| 19  | Rate limiting wiring helper for WebAuthn endpoints                              | Medium | Low     | **Medium** |
| 20  | Catalog doc.go: add `BuildValid()` example alongside `Build()`                  | Low    | Trivial | **Low**    |
| 21  | Property-based testing for catalog schema reflection                            | Medium | Medium  | **Low**    |
| 22  | Upstream go-cqrs-lite: request dispatcher enumeration API for auto-discovery    | High   | High    | **Low**    |
| 23  | Benchmark catalog builder + handlers (allocation targets)                       | Low    | Medium  | **Low**    |
| 24  | Catalog handler middleware support (auth on doc endpoints)                      | Low    | Low     | **Low**    |
| 25  | D2 diagram SVG rendering endpoint (via d2 CLI or WASM)                          | Low    | High    | **Low**    |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `usermgmtcatalog.Default()` live in the catalog module, in usermgmt, or in integration_test?**

The problem:

- If it lives in **catalog/**, the catalog module gains a dependency on usermgmt types → breaks the "zero dep on root or usermgmt" principle
- If it lives in **usermgmt/**, usermgmt gains a dependency on catalog → drags `go-faster/yaml` into every usermgmt consumer
- If it lives in **integration_test/**, it's test-only and consumers can't use it
- If it's a **6th module** (`usermgmtcatalog/`), it's heavy for a convenience wrapper

The cleanest answer seems to be a 6th module that depends on both usermgmt + catalog, but that's a lot of module overhead. Alternatively, document the pattern in the catalog README and let consumers write the ~14 lines themselves.

This is a packaging/architecture decision I can't resolve without understanding your preference on module count vs consumer convenience.
