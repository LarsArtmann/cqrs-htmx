# Status Report — 2026-08-05 06:45 — Deduplication Pass

**Session focus:** Reduce code duplication flagged by `art-dupl --type-aware -t 1` in cqrs-htmx master.

---

## A) FULLY DONE

1. **Analyzed all 13 clone groups** reported by `art-dupl` in the initial scan (`-t 1` threshold, 17 + 3 + 3 + 3 + 2 + 2 + 2 + 2 + 2 + 2 + 2 + 2 + 2 = 13 groups).
2. **Eliminated the `logAuth` duplication** between `usermgmt.Service` and `usermgmt.OAuth2Service`:
   - New file: `usermgmt/service_logging.go` containing package-level helper `logAuthEvent(logger, event, userID, attrs...)`.
   - Both `(*Service).logAuth` (`service_register.go:147`) and `(*OAuth2Service).logAuth` (`service_oauth2_extracted.go:323`) now delegate to the helper with a 1-line body each.
   - Behavior preserved: identical key order (`"event"`, `"user_id"`, attrs…), identical message format (`"usermgmt: "+event`), identical capacity hint (`0, 4+len(attrs)`).
3. **Verified the refactor builds clean**: `GOEXPERIMENT=jsonv2 go build ./usermgmt/...` → no output.
4. **Ran the full usermgmt test suite**: `GOEXPERIMENT=jsonv2 go test ./usermgmt/... -count=1 -race` → `ok 21.430s`. No regressions.
5. **Re-ran `art-dupl`**: clone groups reduced from **13 → 12**. The `logAuth` clone group is gone.

## B) PARTIALLY DONE

6. **Judgment decisions documented** for all 12 remaining clone groups (ACCEPT rationale), but **no actual code change** applied to any of them — they were judged not worth deduping under the `<decision_making>` harm-vs-acceptable rubric.

## C) NOT STARTED

7. **`dashboardui/sse.go` SSE handler vs `Broadcaster.ServeSSE`**: identified as a candidate, but `dashboardui/sseHandler` adds replay (via `sseStore`) and heartbeat (via `SSEHeartbeatInterval`) on top of `ServeSSE`. Would require introducing extension hooks (`WithReplay`, `WithHeartbeat`) into `ServeSSE` — meaningful API design work, deferred.
8. **Cross-module icon `<path>` loop** in `adminui/icons.go:26-30` and `dashboardui/layout.go:146-150`. templ-components only exposes `IconPathData`/`IconPathJS` — no SVG wrapper. Would need either (a) adding an `IconSVG` helper to templ-components, or (b) maintaining separate copies in each module. Neither was attempted.
9. **nix-driven full test suite**: only ran usermgmt tests, not `nix run .#test` (covers all 18 modules).
10. **Lint verification**: did not run `nix run .#lint` after the refactor.
11. **Coverage gate check**: did not re-run `nix run .#coverage-gate` (the 81.6% usermgmt gate is enforced in CI).
12. **`cqrs-lint`**: did not run cqrs-lint — the new file `service_logging.go` might be picked up, but it contains zero `cqrs-lint:ignore` directives and zero CQRS event subscriptions, so should be clean.

## D) TOTALLY FUCKED UP

13. **Nothing is broken.** Build green, tests pass, no lint ran. The only risk surface is the new file `service_logging.go` containing a `*slog.Logger` parameter — type-checked, identical call site, identical output.

## E) WHAT WE SHOULD IMPROVE

### Process gaps I noticed

14. **No `//cqrs-lint:ignore` directive needed** in `service_logging.go` — it's a pure helper with no events, no projections, no HTTP entry points. Verified by reading the new file.
15. **Threshold was `-t 1` (single statement)** — this surfaced a LOT of idiomatic Go patterns (test `t.Parallel()`, single-line `var out T`, stdlib `http.NewServeMux()`) that are not real duplication. Default `-t 5` would have been more focused on actual maintenance burdens. Worth mentioning to the user.
16. **I did not write a CHANGELOG.md entry** for the `logAuth` refactor. Per the project's TODO_LIST convention (AGENTS.md), completed work goes to CHANGELOG, not TODO_LIST.
17. **I did not update AGENTS.md** with the new file location or the convention "logAuthEvent is the canonical structured-auth-log helper" — even though the change is small, this is the kind of cross-session context AGENTS.md exists for.

### Architectural opportunities I deferred

18. **`Broadcaster.ServeSSE` extension hooks** (see #7) — would benefit dashboardui and any future consumer that wants replay + heartbeat on top of `ServeSSE`. The current dashboardui handler is ~45 lines that could be 3 lines + 2 hook options if `ServeSSE` exposed `ServeSSEOpts`.
19. **templ-components gap**: `icons.IconSVG(name, opts)` would eliminate 5+ lines of boilerplate per UI consumer (adminui, dashboardui, and likely future loginpage/themed components). Worth proposing to the templ-components repo.

### Code health signals I noticed

20. **`//go:build ignore` SQL setup files** are template-style ("Copy this file alongside whichever template you use") and shouldn't be counted as production code — but `art-dupl` doesn't know that. They will keep appearing in dedup reports until either (a) they're moved to a separate `templates/` directory, or (b) art-dupl learns to exclude `//go:build ignore` files.
21. **Example programs (`examples/*/main.go`) intentionally duplicate `pingRequest`, `mux := http.NewServeMux()`, `httptest.NewRequest(...)`** — they are copy-paste-friendly teaching artifacts. Worth excluding via `art-dupl --exclude-pattern "examples/**"` going forward.
22. **Test files (e.g. `datastar/response_test.go`)** produce 17 of the 17-clone `t.Parallel()` group. Test boilerplate is by design — exclude `*_test.go` from dedup scans with `--exclude-pattern "*_test.go"` for cleaner reports.

## F) UP TO 50 THINGS TO DO NEXT (Pareto-ordered by impact)

23. **Run `nix run .#test`** — full workspace coverage, not just usermgmt. Catches cross-module regressions.
24. **Run `nix run .#lint`** — verify `service_logging.go` passes golangci-lint (revive, gochecknoglobals, etc.).
25. **Run `nix run .#coverage-gate`** — confirm 81.6% usermgmt gate still holds (the new file has 4 statements, no test branch coverage — likely tiny impact).
26. **Add CHANGELOG.md entry** under v4.x: "Extract `logAuthEvent` shared helper; `*Service.logAuth` and `*OAuth2Service.logAuth` now delegate to it."
27. **Update AGENTS.md** with the new "logAuthEvent is the canonical structured-auth-log helper" note.
28. **Re-run art-dupl with `--exclude-pattern "*_test.go" --exclude-pattern "examples/**" --exclude-pattern "*/sql_setup_*.go"`** — get a cleaner signal focused on production code.
29. **Add `ServeSSEOpts` to `Broadcaster.ServeSSE`** with `WithReplay(sseStore)` and `WithHeartbeat(interval)` hooks; refactor `dashboardui.sseHandler` to use it (eliminate ~30 LOC, close a maintenance gap).
30. **Propose `icons.IconSVG(name, opts)` to templ-components** (or vend a copy into cqrs-htmx's root module as `cqrshtmx.SVGIcon(name, opts)` for consumers).
31. **Investigate whether `dashboardui/icons.go` and `adminui/icons.go` are really the only icon consumers** — if yes, factor the loop into a single shared package (likely `templ-components`).
32. **Add `art-dupl --exclude-pattern "*_test.go" --exclude-pattern "examples/**" --exclude-pattern "*/sql_setup_*.go"` to `nix run .#lint`** as a new `dedup-scan` app — surface only production-code duplication in CI.
33. **Audit `datastar/response_test.go`** — 17 `t.Parallel()` calls + similar body shape. Could be table-driven via `tests := []struct{...}{...}` to reduce from ~270 LOC to ~100 LOC while preserving coverage.
34. **Audit `dashboardui/handler_overview.go` and `dashboardui/layout.go`** for the `var b strings.Builder` + `b.WriteString(...)` pattern — already noted as idiomatic, but a `render.Builder` helper could centralize the Escape/Esc calls (currently spread across 6+ files).
35. **Audit the SSE handler extension points** — `sse_broadcaster.go`, `dashboardui/sse.go`, `e2e/server/main.go` all hand-roll SSE streams. A single `cqrshtmx.SSEHandler(broadcaster, opts)` would unify them.
36. **Add `//cqrs-lint:ignore` annotations to `service_logging.go`** if lint flags it (it shouldn't, but verify).
37. **Consider moving the `//go:build ignore` SQL setup files to `usermgmt/templates/`** so they're physically separated from production code and excluded from dup scans.
38. **Add a `// Deprecated:` migration note** in the existing logAuth methods pointing consumers at `logAuthEvent`? — No, both methods remain public API used by 36 call sites. Don't deprecate.
39. **Verify `examples/middleware-demo` and `examples/observability-demo` still pass** with the dedup changes (they import nothing from usermgmt's internal `logAuth`, so should be unaffected, but worth checking).
40. **Re-run `cqrs-lint`** with the new file present — confirm zero new findings.
41. **Add a doc note in `docs/guides/leveraging-go-cqrs-lite.md`** (or new `docs/guides/structured-logging.md`) explaining the `logAuthEvent` pattern so consumers writing custom auth flows know to use it.
42. **Audit `usermgmt/service_register.go:147-149`** post-refactor — the `logAuth` method now has a single line; consider inlining at all 36 call sites (would trade method-dispatch for explicitness, debatable — probably keep the method for readability).
43. **Profile the `logAuth` call path** — `make([]any, 0, 4+len(attrs))` allocates on every call. Could be replaced with a sync.Pool or struct-array, but this is over-optimization for a debug-log path.
44. **Compare `logAuth` with the 16+ other slog call sites in usermgmt** that don't follow the `("event", event, "user_id", userID)` convention — could the helper be generalized to a `logAuthStage(logger, stage, kv...)` with a structured `AuthEvent` struct?
45. **Audit `slog.Default()` usage in usermgmt** — the `logAuthEvent` helper takes a `*slog.Logger` (not the global default). Verify all callers pass a non-nil logger; nil-deref would crash.
46. **Run `go vet ./...` workspace-wide** — the refactor is a no-op for vet, but cheap to check.
47. **Document the dedup methodology in AGENTS.md**: "When reducing duplication, run art-dupl with `-t 5` and exclude test files + examples + //go:build ignore templates for production-focused reports."
48. **Investigate whether `dashboardui/handlers_dlq.go:87-88` and `dashboardui/handlers_snapshots.go:34-35` could share a `pathValues(r, "projection", "type", "id")` helper** — would centralize validation/error rendering.
49. **Check if `dashboardui/render.go` and `adminui/render.go` could share a `setContentType(w, mime)` helper** — would eliminate the 3-clone group at minimal cost.
50. **Schedule the next dedup pass after #28 lands** — run with the new exclude patterns to surface the next tier of harmful duplication.

## G) QUESTIONS I CAN'T FIGURE OUT MYSELF

**Q1: Should I commit this work now?**
The refactor is minimal (one new file + two 1-line method bodies), tests pass, no regressions. But: AGENTS.md says "NEVER COMMIT: Unless user explicitly says 'commit'." I won't commit without a yes. However, the change is also at risk of being overwritten by the auto-git daemon if I leave it unstaged.

**Q2: Should the CHANGELOG entry go under v4.x.x or a new v4.y.0 minor?**
The `logAuthEvent` refactor is a no-op behavior change. Conventionally this is a `chore(refactor):` patch bump. But cqrs-htmx is still pre-1.0 semver-strict; I don't know which version bucket this belongs in without checking the latest released tag.

**Q3: Should I add `//nolint:dupl` directives to the 12 remaining `ACCEPT` clones as documentation of intentional similarity?**
golangci-lint's `dupl` linter (configured separately from art-dupl) might already be flagging some of these. Adding `//nolint:dupl // <reason>` at each site would be self-documenting but adds noise. I don't know your preference without checking `.golangci.yml`'s `dupl` config and your existing nolint comment density.

---

**Awaiting instructions.**
