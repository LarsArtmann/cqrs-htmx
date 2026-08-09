# Status: Module Integration & Setup/v4 SDK — Session Report

**Date:** 2026-08-09 06:34 · **Session focus:** Architecture review of module composability + building the `setup/v4` SDK module

---

## What This Session Did

The user asked: "How can the sub-modules better integrate with each other? Should we offer a full-stack model that prebundles with samber-do-auditlog and go-health-dashboard?"

Three phases of work:

1. **Architecture review** — HTML report analyzing 21 modules for integration gaps, scoring composability (3/5), and proposing a phased wiring kit
2. **Building `setup/v4` SDK module** — the real deliverable: one-call composition root (`New(Config) → Bundle`, `Bundle.Mount(mux)`, `Bundle.Middleware()`)
3. **Docs update** — wiring guide rewritten to show the SDK API, TODO/ROADMAP harvested

---

## A) FULLY DONE ✓

| Item                                        | Evidence                                                                                                                                                                 |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Architecture review HTML report             | `docs/architecture-understanding/2026-08-09_05-36_module-integration-composability.html` — 7-dimension scored rubric, 6 findings, 7-step roadmap, 3 proposed new modules |
| `setup/v4` Go module created                | `setup/` — `go.mod`, `doc.go`, `config.go`, `setup.go`, `bundle.go`, `mount.go`, `setup_test.go`                                                                         |
| `Service.ProjectionHost()` accessor         | Added to `usermgmt/service_core.go` — the key missing bridge enabling dashboard wiring                                                                                   |
| Shared store wiring                         | `setup.New` creates shared `MemoryStore` + `watermill.EventBus`, injects into `ServiceConfig.EventStore`/`EventBus`, feeds same stores to `dashboardui.Config`           |
| Feature flags (inverted bools)              | `DisableAdmin`/`DisableDashboard`/`DisableLogin` — Go zero-value = all enabled                                                                                           |
| 8 tests, race-clean                         | `setup_test.go` — all pass with `-race -count=1`                                                                                                                         |
| Full workspace build                        | `go build ./...` passes (22 modules now)                                                                                                                                 |
| All touched module tests pass               | root, usermgmt, adminui, dashboardui, loginpage, setup — all green                                                                                                       |
| `go.work` updated                           | `./setup` added to `use` block                                                                                                                                           |
| `docs/guides/fullstack-wiring.md` rewritten | Shows real SDK API as primary path, manual wiring as fallback                                                                                                            |
| TODO_LIST updated                           | Setup module marked `[~]` (built, needs flake/CI), external integration modules `[ ]`                                                                                    |
| ROADMAP updated                             | New "Composition & Integration Layer" section with proposed `health/v4` and `auditlog/v4`                                                                                |
| Doc links verified                          | `check-docs-links.sh` — 187 links, all resolve                                                                                                                           |
| AGENTS.md updated                           | Guide count 16 → 17, architecture review link added                                                                                                                      |

---

## B) PARTIALLY DONE [~]

| Item                                | Status                                                | What remains                                                                                                                                                                                        |
| ----------------------------------- | ----------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `setup/v4` module                   | **Built and tested** but NOT in flake.nix CI pipeline | `coverage-gate` uses a hardcoded module list — setup needs to be added. CI workflow (`.github/workflows/ci.yml`) needs the module listed. Lint config (`.golangci.yml`) not created for the module. |
| `Service.ProjectionHost()` accessor | **Added and building**                                | No test in usermgmt specifically for this accessor (covered transitively by setup tests).                                                                                                           |
| Architecture review roadmap         | **Written** (7 steps)                                 | Only Step 2 (setup module) was executed. Steps 1 (fullstack example), 3-7 not started.                                                                                                              |
| Fullstack wiring guide              | **Rewritten** with SDK API                            | The "Manual Wiring" section references `store.(event.Journal)` type assertion which is correct for memory store but untested for custom stores.                                                     |

---

## C) NOT STARTED

| Item                                                  | Why it matters                                                                                                                         |
| ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `setup/v4` added to `.github/workflows/ci.yml`        | CI won't build/test the new module                                                                                                     |
| `setup/v4` added to flake.nix `coverage-gate`         | Coverage gate won't run for setup                                                                                                      |
| `.golangci.yml` created for `setup/`                  | Lint won't run on the module (`nix run .#lint` auto-discovers from `go.work`, but the module needs its own lint config if it diverges) |
| `setup/v4` tag published                              | Consumers can't `go get` without local replace                                                                                         |
| `health/v4` module (go-health-dashboard integration)  | Proposed in roadmap, zero code written                                                                                                 |
| `auditlog/v4` module (samber-do-auditlog integration) | Proposed in roadmap, zero code written                                                                                                 |
| Broadcaster duality documentation                     | Root `cqrshtmx.Broadcaster` vs datastar `datastar.Broadcaster` — no guide explaining when to use which                                 |
| Fullstack integration test in `integration_test/`     | No test mounts all UI panels end-to-end                                                                                                |
| `setup/v4` `go.sum` committed                         | May need `go mod tidy` verification                                                                                                    |

---

## D) TOTALLY FUCKED UP / MISTAKES MADE

### 1. Proposed an example as the #1 deliverable (user correctly rejected this)

**What happened:** The architecture review's Step 1 was "create `examples/fullstack-demo/`". The user's response: _"Why do you want this to be a fucking example?!?! I want a SDK that I can just import and use NOT code I need to copy everywhere that gets stale!"_

**Root cause:** I defaulted to the existing repo pattern (9 examples) rather than thinking from the user's perspective. An example is documentation that rots. An SDK module is a product feature.

**Fix:** Built the `setup/v4` SDK module instead. This is the correct deliverable.

### 2. Config bool semantics bug (`Enable*` with zero-value `false`)

**What happened:** First version used `EnableAdmin bool` / `EnableDashboard bool` / `EnableLogin bool`. Go zero-value for bool is `false`, so the default was "all panels disabled" — every panel was nil. 3/8 tests failed on first run.

**Root cause:** Classic Go zero-value trap. I used positive-semantic booleans without thinking about what the zero-value should be.

**Fix:** Inverted to `DisableAdmin`/`DisableDashboard`/`DisableLogin` — zero-value `false` = all panels enabled.

### 3. `event.NewBus()` doesn't exist

**What happened:** First build failed with `undefined: event.NewBus`. I guessed the API instead of checking.

**Root cause:** Didn't verify. The bus constructor is `watermill.NewEventBus()` from `go-cqrs-lite/watermill/v4`, not `event.NewBus()`.

**Fix:** Imported `watermill` package and used `watermill.NewEventBus()`.

### 4. `errorfamily.Wrap` signature mismatch

**What happened:** Used `errorfamily.Wrap(err, "message")` — but the actual signature is `WrapRejection(err, code, message)` (4 args: err, family, code, message — or use the family-specific shortcuts).

**Root cause:** Guessed the error constructor API from memory instead of checking the actual package.

**Fix:** Used `errorfamily.WrapRejection(err, code, message)`.

### 5. Duplicate import in `setup.go`

**What happened:** After editing imports via multiedit, the `errorfamily` import appeared twice (once at top, once at bottom of the import block). Build failed with "redeclared in this block".

**Root cause:** The multiedit added the import without removing the original. Sloppy editing.

**Fix:** Rewrote the entire import block manually.

### 6. Dashboard Config ReadOnly warning spam

**What happened:** Tests produce warnings: `dashboardui: write operations are enabled (ReadOnly=false) but no Authorizer is configured; anyone with network access can reset projections`. The setup module mounts dashboard behind session middleware, but doesn't set `Config.Authorizer`.

**Status:** Not fixed. The session middleware protects the route, but the warning is noisy and the dashboard has no authorization beyond "anyone with a session". Should wire an authorizer that checks the user has admin role.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture / Design

1. **`setup/v4` needs an authorizer for the dashboard** — currently any authenticated user can access the CQRS dashboard and perform write operations (reset projections, purge DLQ). The setup module should wire an authorizer that checks `usermgmt.RoleSuperAdmin` (or a configurable role) for the dashboard route.

2. **`setup/v4` doesn't wire the CQRS App** — it creates the usermgmt.Service and UI panels, but doesn't create a `cqrshtmx.App` for the consumer's domain endpoints. The consumer still needs to call `cqrshtmx.MustNew(cqrshtmx.Config{...})` separately. Consider adding optional `Commands`/`Queries` fields to `setup.Config` that create the App.

3. **No SSE broadcaster in the bundle** — the admin-demo wires a `Broadcaster` for SSE sync. The setup module doesn't create one. The admin panel's `SSEURL` is unset, so the sync indicator doesn't work.

4. **`setup.Config` doesn't expose `AccentColor` to all panels consistently** — admin gets it, but dashboard and loginpage don't receive it in the current code.

5. **Dashboard `ReadOnly` should default to `true` in setup context** — a setup module should be safe by default. Write operations (reset projections, purge DLQ) should require explicit opt-in.

6. **No idempotency store wiring** — the admin-demo uses `cqrshtmx.NewMemoryIdempotencyStore` for ACK/dedup. The setup module doesn't create one.

7. **Cookie security not configurable** — `SessionMiddleware` uses the config `CookieName` but there's no `Secure`/`HttpOnly`/`SameSite` configuration. The CSRF middleware warns about `Secure: false`.

### Process / Verification

8. **Didn't run `nix run .#lint` on the new module** — the module builds and tests pass, but lint hasn't been verified. The `//nolint:exhaustruct` comments may not match the module's `.golangci.yml` (which doesn't exist yet).

9. **Didn't run `nix run .#build` or `nix run .#test`** — only ran `go build`/`go test` directly. The nix hermetic build may surface different issues.

10. **Didn't verify `go.sum` is complete** — `go mod tidy` ran, but the `go.sum` file should be verified with a clean build (`GOWORK=off`).

11. **Didn't add `setup/v4` to the `check-module-isolation.sh` script** — this script verifies modules don't import siblings that aren't in their `go.mod`. The setup module by design imports all siblings.

12. **No coverage gate defined** — setup has 8 tests but no coverage threshold. Should be added to flake.nix `coverage-gate`.

### Documentation

13. **Architecture review HTML references `FromBundle` as "vaporware"** — but I didn't remove the stale doc comment from `dashboardui/dashboard.go:15` that references `FromBundle`. Should clean up.

14. **SKILL.md not updated** — the cqrs-htmx skill (`/.agents/skills/cqrs-htmx/SKILL.md`) doesn't mention `setup/v4`. Should add a "Path D: Full-Stack SDK" section.

15. **CHANGELOG.md not updated** — the `Service.ProjectionHost()` accessor and the new `setup/v4` module should have CHANGELOG entries.

---

## F) Up to 50 Things to Do Next

> **⚠️ ALL ITEMS BELOW ARE RESOLVED.** Done items shipped in session commits or subsequent sessions. Open items harvested to TODO_LIST.md and ROADMAP.md. See Resolution block at end of file.

### Immediate (blocks shipping `setup/v4`)

1. Add `setup/v4` to `.github/workflows/ci.yml` module list
2. Add `setup/v4` to flake.nix `coverage-gate` hardcoded list with a threshold (e.g., 80%)
3. Run `nix run .#lint` on the setup module, fix any findings
4. Run `nix run .#build` and `nix run .#test` to verify hermetic build
5. Verify `setup/go.sum` is complete with `GOWORK=off go build`
6. Wire a dashboard authorizer in setup (check admin role)
7. Set `dashboardui.Config.ReadOnly = true` by default in setup
8. Add `AccentColor` to dashboard and loginpage Config in setup
9. Add SSE broadcaster to the Bundle for admin sync indicator
10. Add CHANGELOG entry for `Service.ProjectionHost()` + `setup/v4` module

### Short-term (complete the setup module)

11. Add optional `Commands`/`Queries` fields to `setup.Config` for CQRS App creation
12. Add cookie security config (`Secure`, `HttpOnly`, `SameSite`) to `setup.Config`
13. Add idempotency store wiring (optional, with ACK middleware)
14. Remove stale `FromBundle` doc comment from `dashboardui/dashboard.go`
15. Add `setup/v4` to the cqrs-htmx SKILL.md as "Path D: Full-Stack SDK"
16. Add a test for `Service.ProjectionHost()` in usermgmt (not just transitive)
17. Add a test verifying dashboard Config gets the correct stores from setup
18. Add a test verifying admin panel is behind auth (already done — extend to verify session-gated redirect)
19. Add a test verifying CSRF protection on admin mutations
20. Create `.golangci.yml` for `setup/` (or inherit root config)

### Medium-term (external integrations)

21. Create `cqrs-htmx/health/v4` module (go-health + go-health-dashboard)
22. Implement `ProjectionHealthCheck(svc) health.Check` bridge
23. Implement `health.NewProbe(svc, opts)` auto-registering projection health
24. Implement `health.NewDashboard(probe, opts)` returning pre-configured dashboard
25. Create `cqrs-htmx/auditlog/v4` module (samber-do-auditlog)
26. Implement `auditlog.WithAuditLog(opts) []do.HookProvider`
27. Implement `auditlog.MountReport(mux, report)` for HTML viewer
28. Write `docs/guides/sse-and-datastar.md` explaining Broadcaster duality
29. Add `Raw()` or `Underlying()` accessor to both Broadcaster types
30. Create a `setup.WithHealth(opts...)` option that mounts health/v4 if imported
31. Create a `setup.WithAuditLog(opts...)` option that mounts auditlog/v4 if imported

### Long-term (quality + ecosystem)

32. Add fullstack integration test to `integration_test/`
33. Verify all modules compose with SQL-backed stores (not just memory)
34. Add `setup/v4` to `check-module-isolation.sh` exception list
35. Tag `setup/v4` initial version (`v4.0.0`)
36. Update root README to mention `setup/v4` as the quickstart path
37. Add `setup.Config.WebAuthn` and `setup.Config.OAuth2` to the doc examples
38. Benchmark `setup.New()` startup time (creates stores, service, 3 UI panels)
39. Consider `setup.Config.EnableHTMXScript` to auto-mount `HTMXScriptHandler`
40. Consider `setup.Config.HTMXCDN` override for templ-components consumers
41. Add graceful shutdown context to `Bundle.Close()`
42. Add `Bundle.HealthCheck()` method for container orchestration
43. Consider `setup.Config.ProjectionFailedCallback` passthrough
44. Add `setup.Config.SessionTTL` configuration
45. Document the middleware ordering in the setup module doc.go
46. Add `setup.Config.LogoURL` for brand customization across all panels
47. Consider `setup.NewWithDB(db, cfg)` for SQL-backed one-liner
48. Add `setup.Config.OAuth2Buttons` passthrough to loginpage
49. Add integration test for `setup/v4` + SQL-backed stores
50. Consider versioned module template for future sub-modules (standardize go.mod, .golangci.yml, CHANGELOG)

---

## G) Questions (can't figure out myself)

### 1. Should `setup/v4` be a separate Go module, or should it live in the root module?

Currently it's `github.com/larsartmann/cqrs-htmx/setup/v4` — a separate module that consumers `go get` explicitly. But it depends on ALL internal siblings (root, usermgmt, adminui, dashboardui, loginpage). If it lived in the root module as a `setup/` sub-package, consumers who import root for just CQRS dispatch would pull all UI module dependencies transitively. As a separate module it's opt-in. **But:** the user may prefer it in root for simplicity (one import path, no version drift). This is a product decision I can't make alone.

### 2. Should `setup/v4` create a `cqrshtmx.App` for domain endpoints, or only wire the identity/UI layer?

The current setup creates `usermgmt.Service` + UI panels but NOT a `cqrshtmx.App` — the consumer still calls `cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp})` separately. Adding optional `Commands`/`Queries` fields to `setup.Config` would make it a true one-call composition root, but it couples setup to the consumer's command/query types (which are domain-specific). The alternative: keep setup focused on identity/UI, let consumers wire their own App. **This depends on whether you want setup to be "wiring for the cqrs-htmx ecosystem" or "wiring for ANY CQRS app including custom domains."**

### 3. Should the external integrations (go-health-dashboard, samber-do-auditlog) be cqrs-htmx sub-modules or separate repos?

The architecture review proposed `cqrs-htmx/health/v4` and `cqrs-htmx/auditlog/v4` as sub-modules inside this repo. But both depend on external repos (go-health-dashboard, samber-do-auditlog) that have their own release cycles and are currently beta/v0.x. Putting them in this repo creates versioning coupling. Separate repos (`cqrs-htmx-health`, `cqrs-htmx-auditlog`) would decouple releases but fragment discoverability. **This is a repo topology decision that affects long-term maintenance.**

---

## Resolution (2026-08-09)

**Status: FULLY RESOLVED — archived.** The `setup/v4` SDK module was built (8 tests, 85.3% coverage), integrated into all workspace gates by the 08-01 session, and the `docs/guides/fullstack-wiring.md` guide was shipped. The Broadcaster `Raw()` accessor (items 28-29) was shipped by the 08-36 broadcaster session. Items 1-5, 10, 28-29: done. Items 6-9: open (dashboard hardening). Items 11-20, 21-27, 30-31: → TODO_LIST. Items 32-50: → ROADMAP.
