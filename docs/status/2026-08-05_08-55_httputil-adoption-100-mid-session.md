# Status — httputil Adoption 100% (Mid-Session)

**Date:** 2026-08-05 08:55 CEST\
**Trigger:** "Get us to 100! Oh and btw I kinda HATE re-exports (99% of the time)!"\
**Plan:** `docs/planning/2026-08-05_08-12_httputil-adoption-100-kill-reexports.md`\
**Current branch:** master\
**Session state:** Work in progress — significant code changes landed, several workstreams still open.

---

## Executive Summary

The re-export deprecation layer is fully in place and all internal consumers inside cqrs-htmx have been migrated to direct `httputil` imports. Root, usermgmt, adminui, loginpage, integration_test, and the admin-demo example all compile. Test files were bulk-migrated and the root test suite is green.

The remaining blocker is the **SecurityHeaders split-brain resolution**, which requires publishing a new `httputil` version (≥ v0.9.0) so cqrs-htmx can depend on richer SecurityHeaders fields without breaking the Nix hermetic build (GOWORK=off). I cannot publish remotely from this session, so that workstream is parked with a clear next step.

---

## a) Fully Done

### Code

| Area                           | What changed                                                                                                                                                                                                        | Files                                                                    |
| ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| Re-export deprecation          | All 39 symbols in `csrf_reexport.go`, `ratelimit_reexport.go`, `server_timing_reexport.go` now carry `// Deprecated:` markers pointing consumers to `github.com/larsartmann/httputil`.                              | `csrf_reexport.go`, `ratelimit_reexport.go`, `server_timing_reexport.go` |
| Internal CSRF callers          | `csrf_handler.go` and `options_types.go` now use `httputil.CSRFConfig` / `httputil.ErrCSRFInvalid` directly.                                                                                                        | `csrf_handler.go`, `options_types.go`                                    |
| Internal error mapping         | `errors.go` now references `httputil.ErrCSRFInvalid` directly.                                                                                                                                                      | `errors.go`                                                              |
| Response CSRF header           | `response.go` now uses `httputil.DefaultCSRFHeaderName` directly; removed the unexported `defaultCSRFHeaderName` alias and the unused `defaultCSRFCookieName` / `defaultCSRFFieldName` consts.                      | `response.go`, `csrf_reexport.go`                                        |
| usermgmt rate limiting         | `usermgmt/http.go` and `usermgmt/verification_totp_http.go` now use `httputil.KeyedRateLimiter`, `httputil.NewKeyedRateLimiter`, `httputil.KeyedRateLimiterConfig`, `httputil.KeyExtractorFromRemoteAddr` directly. | `usermgmt/http.go`, `usermgmt/verification_totp_http.go`                 |
| Test migration                 | Bulk-migrated root, adminui, loginpage, and integration_test test files from `cqrshtmx.CSRF*` / `cqrshtmx.RateLimiter*` / `cqrshtmx.ServerTiming*` to `httputil.*`.                                                 | 14 test files (see diff summary)                                         |
| Example migration (admin-demo) | Switched to `httputil.CSRFMiddleware`, `httputil.CSRFConfig`, `httputil.ServerTimingMiddlewareWhen`, and `httputil.NewServer`.                                                                                      | `examples/admin-demo/main.go`                                            |
| Verification                   | `go build ./...` passes for root, usermgmt, adminui, loginpage, integration_test, and examples/admin-demo. Root `go test ./... -count=1` is green. No SA1019 warnings from the re-export files themselves.          | —                                                                        |

### Verification commands run

- `GOEXPERIMENT=jsonv2 go build ./...` (root)
- `GOEXPERIMENT=jsonv2 go test ./... -count=1` (root) — green
- `GOEXPERIMENT=jsonv2 go build ./...` for usermgmt, adminui, loginpage, integration_test, examples/admin-demo
- `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` — no new SA1019/unused issues from changed files

---

## b) Partially Done

### Example migrations to `httputil.NewServer`

| Example                       | Status      | Notes                                                                                                                             |
| ----------------------------- | ----------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `examples/admin-demo`         | Done        | Compiles; uses `httputil.NewServer` + `srv.Start()` pattern.                                                                      |
| `examples/catalog-demo`       | Partial     | `httputil.NewServer` wiring added; needs `goimports` + build verification.                                                        |
| `examples/dashboard-demo`     | Partial     | Import added and server block partially rewritten; the last `multiedit` call failed due to malformed JSON and must be re-applied. |
| `examples/middleware-demo`    | Not started | Still uses `&http.Server{}`.                                                                                                      |
| `examples/observability-demo` | Not started | Still uses `&http.Server{}`; not in original plan but should be migrated for consistency.                                         |

### `examples/basic` and `examples/datastar-demo`

These already use `httputil.NewServer`. They need a regression build/test check, which is not done yet.

---

## c) Not Started

| Task                                        | Plan ref  | Why not started                                                                      |
| ------------------------------------------- | --------- | ------------------------------------------------------------------------------------ |
| Nil-body decoder regression test            | M10 / F10 | Will add after examples are stable.                                                  |
| Update `docs/guides/leveraging-httputil.md` | M08       | Blocked until code migrations are complete so the migration table is accurate.       |
| Fix HTML report factual errors              | M07       | Blocked until the MaxBodySize/SecurityHeaders narrative is final.                    |
| Update `doc.go`                             | M13       | Waiting on final code state.                                                         |
| Update `AGENTS.md`                          | M14       | Waiting on final architecture decisions (SecurityHeaders).                           |
| Update `SKILL.md`                           | M16       | Waiting on final API surface.                                                        |
| Update submodule READMEs                    | M04       | adminui/README.md and loginpage/README.md still reference `cqrshtmx.CSRFMiddleware`. |
| CHANGELOG entry                             | M15       | Should be written once all code changes are in.                                      |
| ROADMAP v5 entry                            | M20       | Not written yet.                                                                     |
| TODO_LIST update                            | M26       | Not written yet.                                                                     |
| Cross-link production-readiness guide       | M19       | Not written yet.                                                                     |
| Final lint/coverage gate                    | M18 / M22 | Not run yet.                                                                         |
| Full workspace race test                    | M12       | Not run yet.                                                                         |
| Render-verify HTML report                   | M21       | Not done.                                                                            |

---

## d) Totally Fucked Up

Nothing is permanently broken. The one operational mistake was a malformed `multiedit` JSON payload for `examples/dashboard-demo/main.go` that caused the tool call to fail. The file is still in a partial state (import added, server block not fully rewritten) and needs a clean second pass. No data loss and no broken commits.

---

## e) What We Should Improve

1. **Finish the example migrations cleanly.** The dashboard-demo partial edit needs to be completed, then middleware-demo and observability-demo migrated. Catalog-demo needs formatting + build check.
2. **Decide on SecurityHeaders.** This is the biggest open question. The plan calls for porting cqrs-htmx's richer SecurityHeadersConfig into httputil and then aliasing in cqrs-htmx. That requires a new httputil release. We need a publish step or a documented decision to defer.
3. **Add the nil-body regression test.** The `decoder.go` fix is already in place; a test locks it in.
4. **Write the migration guide table.** Consumers need a clear `cqrshtmx.X` → `httputil.X` map for all 39 symbols.
5. **Run the full workspace test suite with race.** Only root has been tested so far.
6. **Run `nix run .#coverage-gate`.** This is the CI gate and has not been exercised.
7. **Update public docs and SKILL.md before finishing.** The re-exports are deprecated, so the public-facing cheat sheet must steer consumers to httputil.
8. **Consider removing the now-unused `defaultCSRFCookieName` / `defaultCSRFFieldName` aliases.** Already done; note for the record.
9. **Run `go mod tidy` in affected examples.** admin-demo, catalog-demo, dashboard-demo, middleware-demo, observability-demo.
10. **Verify adminui and loginpage still compile after test import changes.** Already done for adminui/loginpage build, but their tests should be run explicitly.

---

## f) Up to 50 Things To Get Done Next

### Immediate (next 30 min)

1. Fix the malformed `multiedit` and finish `examples/dashboard-demo/main.go` migration.
2. Run `goimports` on `examples/catalog-demo/main.go` and build it.
3. Migrate `examples/middleware-demo/main.go` to `httputil.NewServer`.
4. Migrate `examples/observability-demo/main.go` to `httputil.NewServer`.
5. Build all 6 examples after migrations.
6. Run `go mod tidy` in each example module.
7. Run root `go test ./... -count=1 -race`.
8. Run usermgmt tests.
9. Run adminui tests.
10. Run loginpage tests.

### Short-term (next 60 min)

11. Add `TestDecode_NilBodyDoesNotPanic` regression test.
12. Update `adminui/README.md` CSRF references to `httputil.CSRFMiddleware`.
13. Update `loginpage/README.md` CSRF references to `httputil.CSRFMiddleware`.
14. Update `doc.go` HTTP Middleware section with deprecation note and migration pointer.
15. Update `docs/guides/leveraging-httputil.md` MaxBodySize row.
16. Add re-export deprecation migration table to `docs/guides/leveraging-httputil.md`.
17. Fix HTML report `f-maxbody` card: retract and mark "Not Applicable".
18. Recompute HTML report scorecard numbers.
19. Add methodology note to HTML report.
20. Add snapshot date disclaimer to HTML report hero.
21. Update `docs/guides/leveraging-httputil.md` SecurityHeaders section with current split-brain status.
22. Update `docs/guides/production-readiness.md` cross-link.
23. Update `SKILL.md` cheat sheet to use `httputil.CSRFMiddleware`.
24. Add deprecation note to `SKILL.md`.
25. Update `AGENTS.md` httputil leverage entry.
26. Add `CHANGELOG.md` [Unreleased] entry covering re-export deprecation, decoder nil-body fix, example migrations.
27. Add `ROADMAP.md` v5 entry for httputil re-export removal.
28. Update `TODO_LIST.md` with httputil re-export deprecation item.
29. Mark MaxBodySize as resolved in `TODO_LIST.md`.
30. Run `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

### Medium-term (next 90 min)

31. Run `nix run .#test` for the full workspace.
32. Run `nix run .#coverage-gate`.
33. Run `nix run .#build`.
34. Run `nix run .#lint`.
35. Verify `examples/basic` and `examples/datastar-demo` still compile and run.
36. Render-verify `docs/research/2026-08-05_httputil-deep-dive.html`.
37. Check for any remaining `cqrshtmx.CSRF*` / `cqrshtmx.RateLimiter*` / `cqrshtmx.ServerTiming*` references outside comments.
38. Check for stale doc comments referencing deprecated symbols.
39. Ensure all changed files are `gofmt`/`goimports` clean.
40. Run `go work sync` if needed.

### Blocked / Strategic

41. **Publish httputil v0.9.0** with richer SecurityHeadersConfig (PermissionsPolicy, Custom, SecurityHeaderSkip, RecommendedHSTS, RecommendedCSP, ContentTypeOptions) or decide to defer SecurityHeaders resolution to v5.
42. After httputil v0.9.0 is published, update cqrs-htmx `go.mod` and either alias or remove `security.go`.
43. Re-run Nix hermetic build after SecurityHeaders resolution.
44. Consider whether to temporarily add a `go.work` replace for local httputil during development of the SecurityHeaders port.
45. Update httputil README / DOMAIN_LANGUAGE if SecurityHeaders fields are added.
46. Add httputil tests for new SecurityHeaders fields.
47. Update cqrs-htmx `security_test.go` if SecurityHeaders implementation changes.
48. Verify backward compatibility of cqrs-htmx SecurityHeaders API for consumers.
49. Decide whether to deprecate or delete cqrs-htmx `SecurityHeadersMiddleware`.
50. Final end-to-end verification: all 19 modules green, coverage gate passes, lint clean, docs accurate.

---

## g) Up to 3 Questions I Cannot Figure Out Myself

1. **SecurityHeaders:** Should I proceed with porting the richer SecurityHeadersConfig into httputil and cut/publish httputil v0.9.0 now, or defer that cross-repo release and keep cqrs-htmx's current SecurityHeaders implementation for v4? I cannot push tags/releases remotely from this session.
2. **Example server lifecycle:** For the examples migrated to `httputil.NewServer`, should I use the simple `<-srv.Start()` blocking pattern everywhere, or add graceful-shutdown handling (SIGINT/SIGTERM) to the simpler demos for consistency with catalog-demo?
3. **SecurityHeaders API direction:** If we do resolve the split brain, do you prefer (a) thin deprecated aliases in cqrs-htmx pointing to httputil, or (b) full deletion of `cqrshtmx.SecurityHeadersMiddleware` and friends as a v4.x breaking change because you "hate re-exports"?

---

## Notable Observations

- The `httputil.Server` type returned by `httputil.NewServer` uses `Start() <-chan error` and `Shutdown(ctx)`, not `ListenAndServe()`. Example migrations must account for this.
- The root module's `decoder.go` already contains the nil-body guard (`if r.Body == nil { return nil, nil }`). Only the regression test is missing.
- `goimports` is available in the environment and should be run on every touched Go file before final verification.
- Pre-existing lint findings (canonicalheader, exhaustive in `ws_dispatch.go`) are unrelated to this session's work and should not be "fixed" here.
- The auto-git commit daemon is active; unexpected commits are normal.

---

## Files Changed So Far (high-level)

- `csrf_reexport.go`
- `ratelimit_reexport.go`
- `server_timing_reexport.go`
- `csrf_handler.go`
- `options_types.go`
- `errors.go`
- `response.go`
- `usermgmt/http.go`
- `usermgmt/verification_totp_http.go`
- `examples/admin-demo/main.go`
- `examples/catalog-demo/main.go` (partial)
- `examples/dashboard-demo/main.go` (partial)
- 14 test files across root, adminui, loginpage, integration_test
