# Comprehensive Status Report — 2026-06-14 18:17

> **Audit date:** 2026-06-14T18:17:45+0200
> **Branch:** `master` (1 unpushed commit)
> **Total commits:** 450 (50 today)
> **Total Go LOC:** 20,948 across 4 modules
> **Go version:** 1.26.3

---

## Project Health Summary

| Metric         | Root                 | usermgmt  | integration_test | datastar-demo |
| -------------- | -------------------- | --------- | ---------------- | ------------- |
| Coverage       | **96.4%**            | **90.3%** | N/A              | N/A           |
| Test functions | 78                   | 191       | 8                | 0 (main pkg)  |
| Lint issues    | **9** (pre-existing) | **0**     | **0**            | **0**         |
| `go vet`       | clean                | clean     | clean            | clean         |
| Race detector  | ✅ pass              | ✅ pass   | ✅ pass          | N/A           |
| Build          | ✅ clean             | ✅ clean  | ✅ clean         | ✅ clean      |
| Benchmarks     | 27                   | 5         | 0                | 0             |
| Prod files     | 34                   | 18        | 0                | 8             |
| Prod LOC       | 4,532                | 1,964     | 0                | ~1,100        |
| Test LOC       | 9,310                | 3,474     | 366              | 0             |
| Total LOC      | 13,842               | 5,438     | 366              | 1,302         |

**Total Go code:** 4,532 + 1,964 + 0 + 1,100 = ~7,596 prod LOC + ~13,350 test LOC = **~20,948 lines** across 4 modules.

**Direct dependencies (root):** 11 (casbin/v3, nosurf, go-cqrs-lite ×4, go-error-family, httputil, ginkgo, gomega, x/time)

---

## Session Work (2026-06-14 Afternoon/Evening)

### Commits This Session (5 commits)

| #   | Commit    | Description                                                                                                                     |
| --- | --------- | ------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `d0dfb3b` | htmx 4.0 research report — comprehensive what's-new HTML doc                                                                    |
| 2   | `cc6398b` | Cleanup: benchmark `b.Context()`, go.sum pruning, status doc formatting, go-cqrs-lite audit HTML, fixed duplicate `</code>` tag |
| 3   | `39fb6e5` | `HTMXScriptHandlerWith()` + `HTMXCDNScriptTag()` — pluggable htmx serving (concurrent agent)                                    |
| 4   | `137bc7c` | Pluggable HTMX: examples, docs, AGENTS.md update (complementary to #3)                                                          |
| 5   | `321257a` | htmx 4.0 migration readiness audit — codebase-level impact analysis (1481-line HTML)                                            |

### What Was Accomplished

1. **htmx 4.0 Research** — Fetched and analyzed 13+ documentation pages from four.htmx.org. Documented all 9 breaking changes, 7 new features, 16 extensions, HCON notation, and the extension system rewrite. Compiled into a 1543-line visually-designed HTML document.

2. **Pluggable HTMX Script Serving** — Added `HTMXScriptHandlerWith(js []byte, version string)` to let consumers serve any htmx JS build (v4 beta, custom build) from their Go binary. Added `HTMXCDNScriptTag(version string)` for CDN loading. Extracted `htmxVersion` as a package-level const (was duplicated in 3 places). Full test coverage: 11 new test cases, 3 new examples.

3. **htmx 4.0 Migration Readiness Audit** — Mapped all 9 htmx 4.0 breaking changes to specific files and line numbers in cqrs-htmx. Analyzed CSRF transport risk (fetch() vs XMLHttpRequest), error response swap behavior, header/attribute/event compatibility. Result: ~90% compatible, no Go code changes strictly required.

4. **Cleanup** — Fixed `benchmark_server_test.go` to use `b.Context()`. Pruned 46 lines of unused go.sum transitive dependencies. Fixed a duplicate `</code>` tag in htmx-4-whats-new.html that was causing oxfmt to fail.

---

## a) FULLY DONE ✓

| Area                     | Item                                | Details                                                                                                                                                                                                               |
| ------------------------ | ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Core Library**         | CQRS dispatch                       | Command + Query handlers with decode → validate → auth → dispatch → render pipeline                                                                                                                                   |
| **HTMX Integration**     | Full HTMX response builder          | Fluent API: PushURL, ReplaceURL, Redirect, Refresh, Location, Reswap, Retarget, Reselect, Trigger, TriggerAfterSwap, TriggerAfterSettle, TriggerWithDetail                                                            |
| **HTMX Request Parsing** | All request headers                 | HX-Request, HX-Boosted, HX-Current-URL, HX-Target, HX-Trigger, HX-Trigger-Name, HX-Prompt, HX-History-Restore-Request                                                                                                 |
| **CSRF Protection**      | justinas/nosurf integration         | Middleware + per-handler option + helpers (HTML meta, hx-headers, form field) + TrustedProxies config                                                                                                                 |
| **SSE**                  | Full SSE stack                      | SSEEvent, SSEStream, Broadcaster (O(1) unsubscribe), SSEEventStore, ReplayEvents, BroadcastOnSuccess hooks                                                                                                            |
| **WebSocket**            | Protocol helpers                    | WSMessage, ParseWSMessage, ParseWSMessageInto[T], WSOOBHTML — no WS library dependency                                                                                                                                |
| **Auth**                 | Casbin authorization                | Enforcer interface, Authorize middleware, Enforce helper — `*casbin.Enforcer` satisfies it                                                                                                                            |
| **Rate Limiting**        | Token bucket + min-heap eviction    | Per-key, configurable rate/burst, automatic cleanup of stale entries                                                                                                                                                  |
| **Security Headers**     | SecurityHeadersMiddleware           | Configurable CSP, HSTS, X-Frame-Options, etc. RecommendedCSP/HSTS presets                                                                                                                                             |
| **Recovery**             | Panic recovery middleware           | Package-level + App.RecoverHandler() with configured error handler                                                                                                                                                    |
| **Embedded HTMX JS**     | v2.0.9 via go:embed                 | ~49KB, served with ETag/Cache-Control. Pluggable via HTMXScriptHandlerWith                                                                                                                                            |
| **Pagination**           | go-cqrs-lite v2.3.0 integration     | DecodePagination, RenderPaginatedJSON[T]                                                                                                                                                                              |
| **Logging**              | RequestLogging + RequestLoggingSlog | Text, JSON, and slog formatters. sync.Pool for JSON buffer                                                                                                                                                            |
| **Error Handling**       | go-error-family classification      | MapError → HTTP status. HTMX-aware auth redirects. LoginRedirect helper                                                                                                                                               |
| **Context Enrichment**   | UserID/CorrelationID/RequestID      | Branded types via go-cqrs-lite/pkg/id. Parse*/MustParse* helpers                                                                                                                                                      |
| **usermgmt**             | Full user management                | Rich User entity (behavior methods), Service layer, In-memory stores, HTTP handlers, Session middleware, Account lockout, RBAC via Casbin                                                                             |
| **Performance**          | 11 optimizations (T01-T11)          | CSRF sync.Once, HealthHandler pre-alloc, slog inlining, Broadcaster snapshot, auth decode, SSE single-write, splitSSELines fast-path, setTriggerWithDetail, ParseWSMessageInto, JSONLogFormatter pool, io.WriteString |
| **Testing**              | 269 test functions                  | BDD (Ginkgo/Gomega), fuzz tests, 27+5 benchmarks, race-safe, 96.4%/90.3% coverage                                                                                                                                     |
| **Build/CI**             | flake.nix + BuildFlow               | 25 pre-commit checks (go, nix). Per-module nix apps. treefmt + oxfmt + golangci-lint                                                                                                                                  |
| **Module Structure**     | 4 Go modules                        | Root + usermgmt + integration_test + datastar-demo. Clean boundaries, zero mutual imports root↔usermgmt                                                                                                               |
| **Documentation**        | 5 ADRs, 4 research HTMLs            | Architecture decisions, performance review, htmx 4 research + migration audit, go-cqrs-lite feature audit                                                                                                             |

---

## b) PARTIALLY DONE 🔄

| Item                        | Status                | Details                                                                                                                                                                  | Effort to Complete |
| --------------------------- | --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------ |
| **DOMAIN_LANGUAGE.md**      | Template only         | File exists but contains placeholder text ("Example Term", ". for project name"). Needs filling with actual domain vocabulary (Command, Query, Enforcer, Dispatch, etc.) | 30 min             |
| **ROADMAP.md**              | Stale                 | Shows v2.1.0, go-cqrs-lite v2.2.0. We're on v2.3.x. Coverage numbers outdated (96.9%/91.1% vs actual 96.4%/90.3%). Lists typed dispatch as "Open" but it's done          | 15 min             |
| **CHANGELOG.md**            | Partial               | [Unreleased] section has v2.2.0 entries but no release notes for v2.3.0 changes (SSE, WebSocket, pagination, performance optimizations, pluggable htmx)                  | 1-2h               |
| **FEATURES.md**             | Slightly stale        | Updated 2026-06-09. Missing: pluggable HTMX script serving, HTMXCDNScriptTag, HTMXScriptHandlerWith                                                                      | 30 min             |
| **csrf_middleware_test.go** | Over file limit       | 370 lines (limit: 350, +5.7%). Only test file exceeding the limit                                                                                                        | 15 min to split    |
| **Lint issues (root)**      | 9 pre-existing        | 2 errcheck (benchmark resp.Body.Close), 2 exhaustruct (http.Client), 1 forcetypeassert (sync.Pool), 4 gochecknoglobals (intentional package-level vars for perf)         | 30 min             |
| **datastar-demo binary**    | Committed by accident | `examples/datastar-demo/datastar-demo` (10MB compiled binary) is tracked in git. Should be in .gitignore and removed from history                                        | 15 min             |
| **go.work committed**       | Should be gitignored  | `go.work` is tracked in git but BuildFlow warns it shouldn't be committed for libraries                                                                                  | 5 min              |

---

## c) NOT STARTED 📋

| Item                                        | Priority | Notes                                                                                                                                 |
| ------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| **Form decode JSON round-trip elimination** | High     | `decoder.go` still marshals→unmarshals form data. #1 remaining allocation hot path. Risk: behavioral change for arrays/nested objects |
| **PostgreSQL UserStore implementation**     | High     | Pattern documented in `usermgmt/docs/SQL_STORES.md` + ADR 0003. Library principle: no SQL driver dep                                  |
| **PostgreSQL SessionStore implementation**  | High     | Same pattern as UserStore                                                                                                             |
| **govulncheck in CI**                       | Medium   | Not currently running in ci.yml                                                                                                       |
| **pprof handler in datastar-demo**          | Medium   | No `net/http/pprof` handler for production profiling                                                                                  |
| **Memory pressure tests**                   | Medium   | Rate limiter at 10K+ keys, broadcaster at 10K+ subs untested                                                                          |
| **Compression middleware**                  | Low      | SSE/HTTP responses uncompressed. Consumers must add gzip/brotli at proxy                                                              |
| **Native OTel middleware**                  | Future   | Hook-based pattern currently documented. v3.0.0 target                                                                                |
| **OPTIONS method handling**                 | Low      | For CORS preflight                                                                                                                    |
| **InMemoryUserStore reverse index**         | Low      | O(1) email updates via reverse index                                                                                                  |
| **benchstat CI job**                        | Medium   | No automated before/after benchmark regression detection                                                                              |
| **v2.2.0/v2.3.0 release tag**               | High     | No git tags exist. CHANGELOG not finalized                                                                                            |

---

## d) TOTALLY FUCKED UP 💥

| Issue                                | Severity | Details                                                                                                                                                                                                                                                                |
| ------------------------------------ | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **datastar-demo binary in git**      | Medium   | A 10MB compiled binary (`examples/datastar-demo/datastar-demo`) is tracked in git history. Every clone downloads it. The go-structure-linter flags this every commit. Should never have been committed.                                                                |
| **go.work committed for a library**  | Low      | `go.work` is tracked but BuildFlow warns it shouldn't be committed for libraries. It causes confusion for consumers who import the module.                                                                                                                             |
| **DOMAIN_LANGUAGE.md is a template** | Low      | Still has "." as the project name and "Example Term" as placeholder. Has been like this since project creation. Every status report mentions it. Nobody has fixed it.                                                                                                  |
| **ROADMAP.md is stale**              | Low      | Shows v2.1.0 when we're effectively on v2.3.x. Lists done items as open. Coverage numbers wrong. Misleads anyone reading it.                                                                                                                                           |
| **Concurrent agent conflicts**       | Low      | Commit `39fb6e5` was created by a concurrent agent implementing the same feature I was implementing in `137bc7c`. The commits overlap. The concurrent agent's version is slightly more complete but both exist in history. Not harmful but messy.                      |
| **9 lint issues never addressed**    | Low      | The same 9 golangci-lint issues have been reported in every status report since the performance optimization session. The gochecknoglobals issues are intentional (perf optimization), but errcheck/exhaustruct/forcetypeassert could be fixed or explicitly nolint'd. |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Design

1. **Stop writing status reports, start fixing**: There are 58 status reports in `docs/status/`. The DOMAIN_LANGUAGE.md template has been mentioned in at least 10 of them. Fix it or delete it.

2. **Decide on the lint issues**: The 9 root lint issues have persisted across every report. Either fix them, add `//nolint:` directives with justification, or disable the linters in `.golangci.yml`. The current state — "9 issues, all pre-existing" — is technical debt that compounds with every report.

3. **Tag a release**: 450 commits, 0 tags. There's no v2.0.0, no v2.1.0, no v2.2.0. Consumers have no stable version to pin. The CHANGELOG has unreleased entries going back weeks. Cut a release.

4. **Clean git history**: The datastar-demo binary should be removed from history with `git-filter-repo`. It's 10MB of compiled Go binary in every clone.

5. **Consolidate the flat package decision**: The root package has 34 prod files. AGENTS.md says this is intentional ("Root module is intentionally a single flat package"). But `go-structure-linter` reports 34 ERROR-level issues every commit. Either suppress the linter for this project or accept the structural decision document.

### Process

6. **Single-agent execution**: The concurrent agent conflict (commits `39fb6e5` and `137bc7c`) shows that parallel agents working on the same repo create messy history. If using multiple agents, coordinate via branches.

7. **Reduce status report noise**: 58 status docs is excessive. Consider a single `STATUS.md` that gets updated, or archive old reports more aggressively.

8. **ROADMAP or delete it**: A stale roadmap is worse than no roadmap. Either maintain it or remove it.

### Code Quality

9. **Form decode allocation**: The #1 remaining performance issue (form decode JSON round-trip in `decoder.go`). Every status report since the performance audit mentions it. It's a 4-8h design task. Do it.

10. **Test file size**: `csrf_middleware_test.go` is 370 lines. Split it. The limit is 350. It's been flagged in every report for weeks.

---

## f) Top 25 Things to Get Done Next

Sorted by **impact/effort ratio** (highest first).

| #   | Task                                                                  | Impact        | Effort | Category      |
| --- | --------------------------------------------------------------------- | ------------- | ------ | ------------- |
| 1   | **Tag v2.3.0 release** + finalize CHANGELOG                           | **Critical**  | 30 min | Release       |
| 2   | **Remove datastar-demo binary from git history** (`git filter-repo`)  | High          | 15 min | Hygiene       |
| 3   | **Add `go.work` and build outputs to .gitignore**                     | High          | 5 min  | Hygiene       |
| 4   | **Fix or nolint the 9 root lint issues**                              | Medium        | 30 min | Quality       |
| 5   | **Fill DOMAIN_LANGUAGE.md** with actual domain terms                  | Medium        | 30 min | Docs          |
| 6   | **Update ROADMAP.md** to reflect v2.3.x reality                       | Medium        | 15 min | Docs          |
| 7   | **Update FEATURES.md** with pluggable HTMX, CDN tag, SSE, WS features | Medium        | 30 min | Docs          |
| 8   | **Split csrf_middleware_test.go** (370 → 2 files ≤350)                | Low           | 15 min | Compliance    |
| 9   | **Add govulncheck to CI** (ci.yml)                                    | High          | 15 min | Security      |
| 10  | **Form decode JSON round-trip elimination** (design + implement)      | **Very High** | 4-8h   | Perf          |
| 11  | **Add pprof handler to datastar-demo**                                | Medium        | 15 min | Observability |
| 12  | **Recover root coverage to 96.9%+**                                   | Medium        | 1-2h   | Quality       |
| 13  | **Recover usermgmt coverage to 91%+**                                 | Medium        | 1h     | Quality       |
| 14  | **Add benchstat CI job** for benchmark regression                     | Medium        | 1h     | CI            |
| 15  | **Memory pressure test** for rate limiter (10K+ keys)                 | Medium        | 1h     | Testing       |
| 16  | **Memory pressure test** for broadcaster (10K+ subs)                  | Medium        | 1h     | Testing       |
| 17  | **PostgreSQL UserStore implementation** (documented pattern)          | High          | 4-8h   | Feature       |
| 18  | **PostgreSQL SessionStore implementation**                            | High          | 2-4h   | Feature       |
| 19  | **Integration tests against real PostgreSQL** (testcontainers)        | Medium        | 2-4h   | Quality       |
| 20  | **Add SSE connection count monitoring** to datastar-demo              | Low           | 15 min | Observability |
| 21  | **InMemoryUserStore.Save reverse index** for O(1) email updates       | Low           | 1h     | Perf          |
| 22  | **More cross-module integration tests** (CSRF+CQRS, rate-limit+SSE)   | Medium        | 1-2h   | Quality       |
| 23  | **Consider compression middleware** (optional gzip)                   | Low           | 2h     | Feature       |
| 24  | **Document the two UserID types** more prominently in README          | Low           | 15 min | Docs          |
| 25  | **Native OTel middleware** (hook-based pattern currently documented)  | Medium        | 2-4h   | Feature       |

---

## g) Top Question I Cannot Figure Out Myself

### "Should we cut the v2.3.0 release NOW with the current state, or wait?"

**The context:**

- 450 commits, 0 git tags. No release has ever been tagged.
- CHANGELOG has extensive [Unreleased] entries but no version header.
- 1 unpushed commit (htmx-4 migration readiness doc).
- 9 pre-existing lint issues (intentional globals, minor test issues).
- `csrf_middleware_test.go` is 370 lines (5.7% over limit).
- A 10MB binary is in git history.
- ROADMAP says v2.1.0; actual code is v2.3.x.

**Why I can't decide:**

- Cutting a release is a **business/product decision**, not a technical one. It depends on: whether consumers are waiting for a stable tag, whether the maintainers consider the API stable, and whether the 9 lint issues + binary-in-git are release-blockers or acceptable debt.
- I can prepare everything (CHANGELOG, tag, push) but only the project owner can say "this is v2.3.0, ship it."
- The alternative — fixing all 25 items first — could take days and there's no guarantee the result is "better enough" to justify the delay.

**What I'd do if you say "ship it":**

1. Finalize CHANGELOG with v2.3.0 header and date
2. `git tag v2.3.0 && git push --tags`
3. Create GitHub release with release notes from CHANGELOG

**What I'd do if you say "fix first":**

1. Items #2-8 from the top-25 list (binary removal, gitignore, lint, docs, file split) — ~2h total
2. Then tag and release

---

## Session Summary

| Metric                   | Value                                                                                       |
| ------------------------ | ------------------------------------------------------------------------------------------- |
| Commits this session     | 5                                                                                           |
| Commits today (total)    | 50                                                                                          |
| Total commits (all-time) | 450                                                                                         |
| Files created            | 3 (htmx-4-whats-new.html, htmx-4-migration-readiness.html, go-cqrs-lite-feature-audit.html) |
| New API functions        | 2 (HTMXScriptHandlerWith, HTMXCDNScriptTag)                                                 |
| New tests                | 11 test cases + 3 examples                                                                  |
| Build status             | ✅ 4/4 modules clean                                                                        |
| Test status              | ✅ 4/4 modules pass with race detector                                                      |
| Lint status              | 9 root (pre-existing), 0 usermgmt, 0 integration, 0 datastar                                |
| Coverage (root)          | 96.4%                                                                                       |
| Coverage (usermgmt)      | 90.3%                                                                                       |
| Unpushed commits         | 1 (`321257a`)                                                                               |
