# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **Datastar adapter module** (`datastar/v4`): Optional, fully-isolated Go module (`github.com/larsartmann/cqrs-htmx/datastar/v4`) that lets consumers use [Datastar](https://data-star.dev/) instead of (or alongside) HTMX for frontend reactivity. Depends only on `datastar-go` SDK + `go-cqrs-lite/event/v4` — zero transitive deps from root (no casbin, httputil, or go-sse). Features: self-hosted `datastar.js` v1.0.2 with ETag/1yr cache (`ScriptHandler`/`ScriptHandlerWith`/`ScriptTag`), `ReadSignals` decoder, fluent `Response` builder (PatchElements/PatchSignals/Redirect/ExecuteScript/ConsoleLog/ConsoleError/DispatchCustomEvent/ReplaceURL), `Broadcaster` with bounded patch ring buffer replay (default 256 patches, reconnects via `Last-Event-ID`) and optional SSE heartbeat keep-alive (`NewBroadcasterWithHeartbeat`), `EventBridge` for declarative domain-event-to-patch mapping with optional `OnError` callback, and ~50 SDK option re-exports for single-import convenience. 71 tests, 96.7% coverage (gate 90%), 0 lint issues. See `docs/guides/datastar-integration.md` and ADR-0045.
- **Datastar integration guide** (`docs/guides/datastar-integration.md`): Full walkthrough — installation, quick start (5 steps), patch types, replay/reconnection, HTMX+Datastar coexistence, SDK re-exports, and demo reference.
- **ADR-0045** (`docs/adr/0045-datastar-optional-frontend.md`): Documents the decision to build Datastar support as a standalone module with patch ring buffer replay (not event replay), and why typed decoders returning `HandlerOption` are architecturally impossible (root's `handlerConfig` is unexported).
- **Datastar demo migration** (`examples/datastar-demo/`): Migrated from direct `datastar-go` SDK usage to the adapter module. Replaced custom `Broadcaster` + `BroadcastEvent` types with `ds.Broadcaster` (with replay support). All handlers now use `ds.ReadSignals`, `ds.NewResponse`, `ds.ErrorResponse`. Self-hosts `datastar.js` instead of CDN.

- **Catalog-demo smoke test** (`examples/catalog-demo/main_test.go`): 4 tests covering catalog registration (service, commands, events, queries), catalog validation, HTTP handler endpoints (OpenAPI, AsyncAPI, D2, health), and OpenAPI JSON structure. Previously the only example module with zero test coverage.
- **Errorfamily AST scanner** (`scripts/errorfamily_scanner.go`): Go AST-based scanner replacing the ripgrep-based filter. Uses `go/parser` which inherently ignores ALL comment types (//, /\* \*/, inline, multi-line) — zero false positives from documentation or migration notes. File has `//go:build ignore` so it doesn't affect module builds or coverage.
- **CI workflow expansion** (`.github/workflows/ci.yml`): added build+test+lint for identity-model, dashboardui, and loginpage (previously missing from CI entirely). Added phantom-version and errorfamily quality gates to the security job. Fixed usermgmt coverage threshold from 78% to 74% (matching flake.nix gate). Added identity-model (70%) and dashboardui (60%) coverage checks. Expanded mod-tidy check to cover all 18 workspace modules.
- **loginpage CHANGELOG** (`loginpage/CHANGELOG.md`): created with entries for all 6 tags (v4.0.0 through v4.6.1), following the totp CHANGELOG pattern. loginpage was the only sub-module without a CHANGELOG.
- **ReadinessHandler + DebugHandler** (`readiness.go`): composite readiness checker that runs NamedChecks in parallel and returns 200/503 with per-check status JSON. DebugHandler serves arbitrary system metadata as cached JSON. Both include full test coverage.
- **MySQL read model constructors** (`usermgmt/sql_readmodel_mysql.go`): `NewMySQLUserReadModel`, `NewMySQLMembershipReadModel`, `NewMySQLTenantReadModel`, `NewMySQLBotReadModel` — using `storage.NewViewStoreWithDialect` with `MySQLDialect{}`.
- **MySQL setup template** (`usermgmt/mysql_setup.go`): `//go:build ignore` reference constructor mirroring postgres_setup.go pattern with `stackmysql.New(dsn)`.
- **MySQL dialect test** (`usermgmt/mysql_dialect_test.go`): verifies `MySQLDialect` produces `?` placeholders, `LONGBLOB`/`DATETIME(3)` schema types, no Postgres `$1` placeholders.
- **State cache benchmark** (`usermgmt/benchmark_test.go`): `BenchmarkStateCache_ColdVsWarm` — 50-event stream cold path 189μs vs warm path 14μs (13.7x speedup from `WithStateCache`).
- **TOTP replay documentation tests** (`usermgmt/totp/replay_test.go`): documents stateless TOTP design (RFC 6238 §5.2) and validates window behavior.
- **Correctness tests** (`usermgmt/correctness_test.go`): eviction goroutine lifecycle test + state cache invalidation test (proves cache serves updated state after sequential writes).
- **dashboardui projection host tests** (`dashboardui/handlers_projection_host_test.go`): 3 tests covering ProjectionHost branches in overviewStats and dlqIndexHandler (were 0% covered). Coverage improved 82%→84%.
- **CI gate apps** (`flake.nix`): `check-phantom-version` (scans go.mod for zero pseudo-versions), `check-cqrs-lint` (runs cqrs-lint --strict on all 9 modules).
- **E2E flake.nix fix** (`flake.nix`): 3-layer fix — replaced removed `pkgs.nodePackages.npm` with `pkgs.nodejs`, added `GOEXPERIMENT=jsonv2`, added NixOS Chromium auto-detection via `E2E_BROWSER_PATH`. All 4 Playwright tests now pass (was completely broken).
- **errorfamily gate rewrite** (`flake.nix`): replaced broken `branching-flow errorfamily` subcommand with ripgrep-based check scanning for `errors.New(`/`fmt.Errorf(`/`errors.Join(` in non-test Go files. Comment-line filtering prevents false positives.
- **MySQL setup guide** (`docs/guides/mysql-setup.md`): prerequisites, event store, read models, connection string tips, supported/unsupported matrix.
- **MySQL migration guide** (`docs/migrations/adding-mysql.md`): step-by-step Postgres→MySQL migration instructions.
- **Guide updates**: `leveraging-go-cqrs-lite.md` §9b (state cache & snapshotting), `projection-health-monitoring.md` (OnProjectionFailed callback), `event-store-storage-health.md` (MySQL section).
- **Docs freshness script fix** (`scripts/check-docs-freshness.sh`): `head -3 go.mod` replaced with `grep '^go '` to handle go.mod files with leading comments.
- **Lockout eviction wiring** (`usermgmt/service_core.go`): `AccountLockout.EvictStale` is now wired into `NewService` via `wireLockoutEviction()` helper, matching the existing pattern for WebAuthn, verification, TOTP, and OAuth2 ephemeral stores. Without this, lockout maps grew unbounded in production (memory leak). Test: `TestService_LockoutEvictionWired`.
- **Dispatch middleware documentation** (`doc.go`): 20-line section documenting the `.Use()` recipe — the #1 undocumented capability (27 middleware factories from `go-cqrs-lite/middleware/v4` compose with zero glue). Cross-references the leveraging guide and middleware-demo example.
- **TOTP replay-window documentation** (`usermgmt/totp/provider.go`): `ValidateCode` doc comment now explains the stateless design, RFC 6238 §5.2 replay protection recommendation, and the mitigation path for consumers.
- **`decider.WithStateCache` wired** (`usermgmt/snapshot.go`, `usermgmt/stack_repositories.go`): state cache added via shared `repositoryOptions[State]` function, used by all 4 aggregate repositories (User/Membership/Tenant/Bot). Eliminates full event replay on every Execute (O(total events) → O(new events)). Auto-invalidating (decider manages cache after writes). Zero consumer-visible API change.
- **`projectionhost.WithOnFailed` callback** (`usermgmt/es_setup.go`, `usermgmt/es_projection_setup.go`): opt-in `OnProjectionFailed` field on `EventSourcedConfig`/`ServiceConfig`. Enables alerting on terminal worker failures (after 5 crash-restarts). Nil = no callback (same behavior as before).
- **MySQL event-store dialect** (`go-cqrs-lite/storage/sql/dialect.go`, `usermgmt/sql_event_store.go`): `MySQLDialect` added (11 methods, MySQL-specific DDL with `?` placeholders, `LONGBLOB`/`JSON`/`DATETIME(3)` types, inline indexes). `IsDuplicateKeyError` extended with "Duplicate entry" string match. `dialectToUpstream` updated. `storage/v4` bumped to v4.5.0.
- **dashboardui extended handler coverage** (`dashboardui/handlers_coverage_ext_test.go`): 16 new tests covering overviewStats, eventDetailHandler, loadEventByID, dlqDetailHandler, dlqIndexHandler, renderDLQ, snapshotDetailHandler, loadRecentEvents. dashboardui coverage improved from 78.7% to 82.1%.
- **E2E Playwright flake.nix app** (`flake.nix`): `nix run .#e2e` app added — builds server, starts it, runs Playwright via bun/npx.
- **DashboardUI lint remediation** (workspace-wide): 69 pre-existing lint issues in dashboardui module resolved — goconst (badge CSS classes + status strings extracted to `constants.go`), mnd (magic numbers extracted), varnamelen (parameter renames `pg`→`page`, `f`→`filter`, `n`→`num`), nestif/gocognit/cyclop (nolint directives for domain-complex functions), wsl_v5/nlreturn/gofumpt/golines (auto-fixed). All 15 workspace modules now at 0 issues.
- **cqrs-lint adoption** (workspace-wide): adopted `//cqrs-lint:ignore(RULE)` inline suppression syntax across all 15 modules. 79 findings remediated: 20 deprecated `event.NewEvent` → `event.New` calls migrated across 8 usermgmt decide files (with `event.WithCodec(codec.JSONCodec{})` to preserve JSON encoding), BasicCommand embedding in examples (`createItemCmd`, `pingCmd`), `RegisterTyped` conversion in `examples/basic`, 19 intentional `Must*` panics suppressed with documented reasons, and module-level E003 finding suppressed via go.mod comment for identity-model (intentionally cohesive domain-types module). cqrs-lint exits 0 in `--strict` mode.
- **cqrs-lint round 2** (56 stale-suppression relocations across 11 files): 4 stale E006 → inline on `event.New(`, E003 → go.mod comment, 15 C009 (panic) → `panic(` line, 4 B005 (fold) → `func Fold*(` line, 3 B007 (catalog) → `catalog.Register(` line, 4 A017 (snapshot) → `decider.NewRepository(` line, A005/S003/E004/E006 relocations. Root cause: suppression parser checks line + line-above only; comments on closing `)` of multi-line constructs were on neither.
- **dashboardui improvement sprint** (18 P0 bugs + security + CSS + a11y): overview stats accuracy (TotalAggregates, TotalEvents), SSE JS reconnection rewrite, dead code removal (`var _ = id.NewStreamID`, dead `csrfMeta()`), XSS fixes (all event types/IDs/stream IDs escaped via `esc()`), 404 catch-all handler, method-not-allowed (405), CSRF token header-first reading. Security hardening: error sanitization (all `err.Error()` removed from user-facing messages), audit logging for write ops (`slog.InfoContext`), confirmation dialogs on destructive forms, path parameter validation. CSS overhaul: 191-line class system with CSS custom properties, dark mode via `@media (prefers-color-scheme: dark)`, focus-visible outlines, print stylesheet. Cursor-based pagination across events/aggregates/commands/queries. Mobile responsive design. Accessibility: aria-labels.
- **Go cache prewarming script** (`scripts/prewarm-gocache.sh`): builds all workspace modules before concurrent analysis tools (govalid) start, mitigating the `go/packages` build-cache collision flake (21% per-attempt failure rate). The script is opt-in (not wired into buildflow yet).
- **identity-model enhancements**: domain events for identity aggregate, email value object improvements, unified real-time event dispatch and HTMX integration.
- **dashboardui handler restructuring**: shared formatting utilities extracted (`format.go`), handler files split by domain, unified response patterns across CQRS dashboard endpoints.
- **`docs/guides/leveraging-go-cqrs-lite.md`** (10-section adoption map): comprehensive guide documenting how to leverage 58 go-cqrs-lite modules from cqrs-htmx. Covers dispatch middleware (the #1 undocumented capability — 27 factories via `dispatcher.Use(...)`), OTel/Prometheus wiring, durable scheduling, event signing/encryption, catalog docs generation, scenario-based testing, transport/http SSE (deliberate non-adoption), reactive sagas (deriver), schema evolution, and niche modules.
- **`check-docs-links` nix app + script** (`flake.nix`, `scripts/check-docs-links.sh`): permanent markdown link checker that scans all .md files for broken file-path links. Filters out URLs, code blocks, and Go generics false positives. Integrated into the composite `check-modules` app. Replaces ad-hoc throwaway scripts used in prior docs-health sessions.
- **Datastar added to CI workflow** (`.github/workflows/ci.yml`): build, test (with coverage), lint (golangci-lint), and coverage gate (90% threshold) steps added for the datastar module. Closes the last gap in CI module coverage.
- **`examples/middleware-demo/`** (runnable proof): demonstrates wiring `go-cqrs-lite/middleware/v4` onto a cqrs-htmx command dispatcher with zero glue — `dispatcher.Use(CommandRecovery(), CommandRetry(), CommandCircuitBreaker(), CommandLogging())`. Includes 3 tests (`main_test.go`) asserting retry→204 behavior with race detection. Verified: first call retries twice (transient failures) then returns 204; second call succeeds immediately.
- **Guide snippet API verification**: all code snippets in the leveraging guide verified against go-cqrs-lite source. 6 API inaccuracies corrected: §2 Prometheus `Setup()` return type (`*Provider, error` not `http.Handler`), §3 scheduling callback signature (`Timer[P]` not `P`), §4 signing/encryption constructors (split inline nesting — they return `(value, error)` with unexported types), §5 catalog API (`simple.New()` requires title+version, `Command` is generic, `NewDocsServer` takes provider function), §8 deriver API (`OnEvent` doesn't exist — uses `Deriver` type with `.Filter().AsHandler()`), §9 schema upcaster API (corrected `NewVersionedSeekableJournal` error return, removed non-existent `Register`, fixed `NewUpcaster` callback to `event.Event→event.Event`).
- **Projection upcasting gap REFUTED**: traced the projection read path and confirmed that §9's claim ("projections bypass upcasters") is **false**. All projections route through `unmarshalPayload` → `identitymodel.UnmarshalPayload` → `applyUpcasters`, which runs upcasters at decode time. The store-layer `schema.VersionedSeekableJournal` is an alternative approach, not a needed fix. Guide §9 corrected, TODO item closed.
- **`docs/planning/2026-07-30_00-28_leveraging-hardening-and-expansion.md`**: comprehensive Pareto plan with 25 medium-granularity tasks and 95 fine-grained sub-tasks for hardening and extending the go-cqrs-lite leveraging story.
- **identity-model coverage-gate threshold** (`flake.nix`): added `check_cov identity-model 70` to the coverage-gate app. Coverage verified at 74.9% (well above the 70% threshold). The coverage gate now checks 9 modules (was 8). This was flagged as open in 5+ status reports since 2026-07-23.
- **`.golangci.yml` exclusion audit** — reviewed ALL exclusion rules across 4 config files (root, identity-model, usermgmt, adminui) and 100 `//nolint:` directives across the workspace. **Verdict: zero masked bugs.** All exclusions are structurally justified (`unused` for `//go:build ignore` consumers, `nilnil`/`nilerr` for not-found conventions, `exhaustruct` for builder-pattern structs, `wrapcheck` for typed domain errors, `goconst` for metadata labels, test-file pragmatism exclusions).
- **dashboardui write-operation handler tests** (`dashboardui/handlers_write_test.go`): 16 new tests covering all previously-untested write-operation handlers — DLQ delete (3 scenarios), DLQ purge (3), DLQ replay (2), projection reset (2), snapshot delete (3), time-travel detail (3). Coverage improved from 55% to 66.5%.
- **dashboardui index handler tests** (`dashboardui/handlers_index_test.go`): 5 new tests covering the previously-untested `timeTravelIndexHandler` and `snapshotsIndexHandler` index views (empty state with no StreamReader, listings with multiple streams, version rendering). dashboardui coverage improved from 66.5% to 72.5%.
- **dashboardui coverage gap tests** (`dashboardui/handlers_coverage_test.go`): 30+ new tests covering previously-untested handlers and helpers — `aggregatesIndexHandler` (empty/listings/pagination), `projectionHealthPartialHandler` (nil/empty host), `projectionStatusKind` (all status branches), `renderProjectionRow` (all status kinds), `renderProjectionHealthPanel` (empty/with projections), `buildProjectionStats` (nil/empty host), `MustNew` (success/panic), `guard` (nil-authorizer/denied/allowed), `Handler`/`Middleware`/`Config` accessors, `relativeTime` (all time ranges), `humanByteSize` (B through GiB), `encodingBadgeClass` (all encodings), `parsePageSize` (default/valid/invalid/zero/negative/over-max), `renderPagination` (none/next-only/prev-only/both/extra-params), `listStreams`, `metaRow`/`metaRowCopyable`, `truncate`, `esc`, `emptyState`. All previously 0%-coverage functions now at 100%. dashboardui coverage improved from 68.1% to 78.7% (gate 60%).
- **CorrelationID recovery in panic responses** (`recovery.go`): `writePanicResponse` now recovers the CorrelationID from the `X-Correlation-ID` request header (same pattern as the existing RequestID recovery from the response header). Panic responses now carry the same correlation context as every other error path. New test in `recovery_test.go`.
- **identity-model Authz engine tests** (`identity-model/authz_engine_test.go`): comprehensive test coverage for all 18 `Authz` methods (Enforce, EnforceAny, EnforceEx, Authorize, AsEnforcer, Apply, AddPolicy, RemovePolicy, AddGroupPolicy, RemoveGroupPolicy, RemoveAllRolesForUser, RemoveAllRolesInDomain, Policies, GroupPolicies, RolesForUser, ImplicitRolesForUser, ImplicitPermissionsForUser, DomainsForUser, UsersForRole, RolesForActor, ImplicitRolesForActor). Coverage increased from 0 to full method coverage.
- **identity-model command constructor tests** (`identity-model/commands_events_test.go`): unit tests for all 19 command constructors (RegisterUser, ChangeEmail, ChangeDisplayName, DeleteUser, AddCredential, RemoveCredential, VerifyEmail, EnableTOTP, DisableTOTP, LinkExternalAccount, UnlinkExternalAccount, AddMember, UpdateMemberRoles, RemoveMember, CreateTenant, SuspendTenant, ReactivateTenant, DeleteTenant, RegisterBot, DeleteBot) plus 8 event payload round-trip tests (UserRegistered, EmailChanged, MemberAdded, TenantCreated, BotRegistered, CredentialAdded, ExternalAccountLinked, UserDeleted).
- **identity-model dedup helper tests** (`identity-model/authz_helpers_test.go`): 14 unit tests covering `mutatePolicy`, `mutateGroupPolicy`, `getPolicies`, `rolesForUser`, and `marshalJSONOrWrap` — all previously untested internal helpers.
- **usermgmt dedup helper tests** (`usermgmt/sql_helpers_test.go`): 4 unit tests covering `newViewStoreOrFail` and `wrapTransientOrOK`.
- **Root dedup helper tests** (`event_catalog_helpers_test.go`): 2 unit tests covering `serializeToImmutableHandler`.
- **dashboardui handler + payload tests** (`dashboardui/handlers_extra_test.go`, `dashboardui/handlers_helpers_test.go`): 13 tests covering payload rendering (DefaultPayloadRenderer empty/JSON/invalid/unknown-encoding), CSRF token extraction, DLQ index/detail handlers, projections index handler, and `requireProjectionHost`/`requireDeadLetterStore` guard helpers.
- **HTMX-aware panic recovery** (`recovery.go`): `writePanicResponse` recovers the RequestID from the `X-Request-ID` response header (written by `ContextEnrichmentMiddleware`) when it is absent from the request context. Because `RecoveryMiddleware`/`RecoverHandler` runs outside `ContextEnrichmentMiddleware` in the recommended stack order, its captured request lacked the RequestID — making panic 500 responses the only error path that dropped the correlation ID. The fix enriches the request before delegating to the error handler so panic responses now carry the same request_id as every other error path. New test coverage in `recovery_test.go`. No consumer changes required.
- **SQL read model query projections** (`usermgmt/sql_readmodel.go`, `usermgmt/sql_readmodel_extra.go`): extended the SQL read model with additional query projection methods and a shared `sql_view_marshal.go` helper for view marshalling. Corrects dependency drift in the SQL read model.
- **dashboardui guard methods** (`dashboardui/handlers.go`): extracted `requireProjectionHost` and `requireDeadLetterStore` guard helpers, replacing inline nil-checks across projection reset, DLQ replay, and DLQ delete handlers.
- **Datastar added to errorfamily gate** (`flake.nix`, `.github/workflows/ci.yml`): the errorfamily app and CI security job now include the datastar module in the AST-based scan for stdlib error constructors. Datastar has zero violations; adding it ensures consistency with all other linted modules.
- **check-docs-links test suite** (`scripts/test-check-docs-links.sh`): 13 test cases exercising the markdown link checker's awk extraction logic — known-good links, known-broken links, Go generics false positives (`[T](mapper)`), fenced code block isolation, anchor-only/query-string handling, and URL/mailto skipping. All tests pass.
- **Template compilation verification** (`scripts/check-templates.sh`, `flake.nix`): `nix run .#check-templates` verifies that `//go:build ignore` SQL setup template files (sqlite/postgres/mysql + shared helper) actually compile. Temporarily strips build tags, adds stack backend deps, builds the full usermgmt package, then restores originals via trap-based cleanup. Catches type errors, missing imports, and broken refactors in template files that are otherwise never compiled by normal `go build`.

### Changed

- **adminui toast migration to `feedback.ToastContainer`** (`adminui/components.templ`, `adminui/assets/admin.js`, `adminui/tailwind.css`): replaced the hand-rolled `<div class="toast-host">` + custom `toast()` JS function with the templ-components library's `@feedback.ToastContainer("")`. The `adminui:toast` HX-Trigger event flow is preserved via a bridge that maps kind values (`ok`/`err`/`warn`/`info`) to the library's toast types (`success`/`error`/`warning`/`info`) and calls `tcShowToast()`. Removed 30+ lines of custom CSS — styles now come from the library. Accessibility improved: library toasts include `aria-live`, dismiss buttons, and proper role attributes.
- **adminui global HTMX error handling** (`adminui/layout.templ`): added `@htmx.GlobalErrorHandling(htmx.DefaultErrorHandlingConfig())` to the admin layout. Provides network error detection, 5xx retry (2 retries, 1s base delay), session-expiry redirect on 401, JSON error body parsing (`{family, code, message, fix}`), and ARIA announcer for accessible error feedback. Previously adminui had zero HTMX error handling — network failures and 5xx responses silently failed.
- **templ-components v1.6.0 → v1.7.0** (`adminui/go.mod`, `examples/admin-demo/go.mod`): bumped for latest component additions (LineChart, PieChart, AreaChart, AuthLayout recipe, CollapsibleSection, Heatmap).
- **dashboardui nav icons migrated to `templ-components/icons`** (`dashboardui/layout.go`): replaced the 10-case hand-rolled `navIconSVG()` switch with `icons.IconPathData()` calls. Icon names mapped: `chart`→`Chart`, `queue`→`QueueList`, `cube`→`Cube`, `arrow-path`→`ArrowPath`, `bug`→`BugAnt`, `clipboard`→`Clipboard`, `magnifying-glass`→`Search`, `clock`→`Clock`, `archive`→`ArchiveBox`. Icons now use Heroicons (24x24 stroke) instead of custom 20x20 fill paths — consistent visual style with adminui.
- **adminui CSP nonce threading** (`adminui/models.go`, `adminui/components.templ`, `adminui/layout.templ`): added `Nonce` field to `pageData` and threaded it through `ToastContainer(nonce)` and `ErrorHandlingConfig{Nonce: nonce}`. Consumers with strict CSP can now set a per-request nonce; defaults to `""` (no change for existing consumers).

- **Submodule `.golangci.yml` Go version drift fixed** (adminui, usermgmt, loginpage, identity-model, integration_test): all updated from stale `1.26.3`/`1.26.4` to `1.26.5` matching the root config and actual toolchain.
- **SQL setup deduplication** (`usermgmt/sql_setup_shared.go`): extracted `buildSQLEventSourcedSetupCore` shared helper from 3 identical ~40-line setup sequences in sqlite/postgres/mysql setup templates. Moved `extractDB` and `createAuthzAndCasbin` from sqlite_setup.go to the shared file (they were already called cross-file by postgres/mysql templates — implicit hidden coupling). Renamed `createSQLReadModels` → `createSQLiteReadModels` for naming consistency. Clone count reduced from 5 to 3 groups (art-dupl). Verified by `nix run .#check-templates`.

- **Errorfamily gate upgraded** (`flake.nix`, `scripts/errorfamily_scanner.go`): replaced ripgrep-based filter with a Go AST-based scanner (`go/parser`). The AST approach inherently ignores ALL comment types (//, `/* */`, inline, multi-line) — eliminating the entire class of false-positive risk from documentation mentioning `errors.New`/`fmt.Errorf`/`errors.Join`. The scanner has `//go:build ignore` so it doesn't affect module builds or coverage.
- **TODO_LIST.md reconciled**: removed stale state cache invalidation TODO (both `TestStateCache_ServesUpdatedStateAfterWrite` and `BenchmarkStateCache_ColdVsWarm` already existed). Removed completed items: catalog-demo smoke test, errorfamily comment-awareness, loginpage CHANGELOG, phantom-version CI gate. Updated cqrs-lint CI gate target from `.buildflow.yml` to GitHub Actions (blocked on Nix-only binary).
- **httputil consolidation**: Server-Timing, CSRF core, and keyed rate limiting moved from cqrs-htmx root to `httputil`. cqrs-htmx now re-exports these via type/var aliases (`server_timing_reexport.go`, `csrf_reexport.go`, `ratelimit_reexport.go`). Consumer API is unchanged. Removes `justinas/nosurf` and `golang.org/x/time` as direct root dependencies (now transitive via httputil). **httputil v0.8.0 published**: `go.work` local replace removed. Root `go.mod` bumped v0.7.1 → v0.8.0. Hermetic `nix run .#test` now passes without the replace.
- **httputil adoption to 100%**: all 39 re-export symbols deprecated (see Deprecated section). All internal callers, 14 test files, all 7 examples, and all docs migrated to direct `httputil.*` imports. `docs/guides/leveraging-httputil.md` updated with full migration table. **SecurityHeaders split-brain resolved**: httputil's `SecurityHeadersConfig` enriched with `PermissionsPolicy`, `Custom`, `ContentTypeOptions`, `SecurityHeaderSkip`, `RecommendedHSTS`/`RecommendedCSP` (additive, backward-compatible); cqrs-htmx's `security.go` is now a deprecated alias + delegating wrapper over httputil. `adminui/handler.go` and `loginpage/handler.go` migrated from `cqrshtmx.CSRFTokenFormField`/`CSRFTokenHTMLMeta` to `httputil.*`. HTML report corrected: f-maxbody card retracted, scorecard recount, methodology + snapshot date disclaimer added. **Publish step required:** tag httputil v0.9.0, bump cqrs-htmx `go.mod`, remove `go.work` replace.
- **All 7 examples migrated to `httputil.NewServer`**: `dashboard-demo`, `middleware-demo`, `observability-demo` joined `admin-demo`, `basic`, `catalog-demo`, `datastar-demo` in replacing bare `&http.Server{}` / `http.ListenAndServe` with `httputil.NewServer(httputil.ServerConfig{...}, handler)` + `<-srv.Start()`. All example `go.mod` files tidied. Production-grade timeouts everywhere.
- **Coverage numbers updated**: Root 93.3% (gate 90%), openapi 99.0%, usermgmt 81.6% (gate 74%), identity-model 74.9% (gate 70%), dashboardui 84.0% (gate 60%), datastar 96.7% (gate 90%). All 10 coverage gates pass.
- **TODO_LIST.md reconciled**: removed 5 completed/resolved items that were split brains — 3 previously completed (identity-model coverage gate, .golangci.yml exclusion audit, dashboardui write-operation handler tests), 1 design-evaluated (durable-scheduling: moved to ROADMAP "Not Planned" per design doc conclusion), and 1 done (dashboardui index handler tests + coverage gap tests). All P1 and P2 sections now have no open items.
- **CONTRIBUTING.md version references updated** (stale `v4.5.0` → `v4.6.1` across the module tag table; dashboardui `v4.0.0` → `v4.1.1`, identity-model `v4.1.0` → `v4.1.1`; loginpage gate `80%` → `79%`). Publishing-bug section updated to reflect the ongoing 13-of-40 broken submodule tags and that `go.work` local replaces are still required.
- **Migrated `id.AggregateID` → `id.StreamID` across all modules** (root, usermgmt, dashboardui, identity-model): all `id.NewAggregateID()`, `id.AggregateID`, `id.ParseAggregateID`, `id.DeriveAggregateID`, and `.AggregateID()` method calls migrated to the non-deprecated `StreamID` equivalents. All SA1019 staticcheck deprecation warnings cleared across root, usermgmt, and dashboardui. Aligns with go-cqrs-lite v4.2.0 upstream naming.
- **dashboardui `handlers.go` split into domain files** (`dashboardui/handlers.go` + 7 new files): the 1180-line monolith split into `handlers_events.go`, `handlers_aggregates.go`, `handlers_projections.go`, `handlers_dlq.go`, `handlers_audit.go`, `handlers_timetravel.go`, `handlers_snapshots.go`. Shared helpers (`streamRefFromRequest`, `loadStreamFromRequest`, `streamPathValues`, `latestVersion`) remain in `handlers.go`.
- **Error swallowing fixed in Close() methods** (`usermgmt/service_core.go`, `usermgmt/es_setup.go`, `usermgmt/es_setup_core.go`, `dashboardui/dashboard.go`): all 6 `Close`/`GracefulClose` methods that silently discarded projection host stop errors now log via `slog.Warn` before discarding. The `Dashboard.Close()` method also warns when no broadcaster is configured.
- **Auth sub-module CHANGELOGs aligned** (`usermgmt/{totp,webauthn,oauth2}/CHANGELOG.md`): added `[v4.6.1]` entries for lockstep version alignment with root v4.6.1.
- **Dependency bumps**: go-cqrs-lite `v4.1.0` → `v4.2.0` (command, event, id, idempotency, query; storage/memory and snapshot remain at `v4.1.0`), go-branded-id `v0.3.2` → `v0.5.0`. All examples and sub-modules updated. No cqrs-htmx API change.
- **identity-model authorization policies refactored** (`identity-model/authz_policies.go`, `identity-model/authz_roles.go`): simplified role hierarchy and policy definitions.
- **Event catalog handler aligned** (`event_catalog_handler.go`): aligned with SQL-backed session and view marshalling patterns.
- **Root `sse_broadcaster.go` errcheck fix**: `stream.Close()` error return now checked in `ServeSSE`.
- **Completed SA1019 deprecation migration across all 15 modules**: `id.AggregateRef` → `id.StreamRef` (12 sites in usermgmt snapshot.go/test), `evt.AggregateType` → `evt.StreamType` (2 sites in service_security_test.go), `event.ErrAggregateNotFound` → `event.ErrStreamNotFound` (2 test files), `usermgmt.NewUserID` → marked deprecated with nolint (backward-compat shim), `examples/admin-demo` `NewUserID` → `SyntheticUserID` (2 sites). Zero SA1019 warnings remain across all 15 workspace modules.
- **All 15 modules now lint-clean (0 issues each)**: triaged and resolved all remaining lint findings across root, usermgmt, dashboardui, and all sub-modules. Fixes include: errcheck on `defer svc.Close()` in projection health tests (5 sites), exhaustive switch cases (2 sites: added `errorfamily.Orchestration` and nolint for default-handled worker statuses), exhaustruct exclusion patterns for `readModelCore`/`InMemoryStore`/`IndexSpec`/`identitymodel.User`, gochecknoglobals exclusion for state var aliases, wrapcheck exclusion for thin re-export wrappers, goconst exclusion for event catalog schema strings, funlen exclusion for event catalog registration, testpackage exclusion for white-box test files, and nolintlint cleanup (removed unused `funlen` from 2 nolint directives in dashboardui). Removed dead `var _ = fmt.Sprintf` import hack from `sql_helpers_test.go`.
- **Datastar module added to nix lint/test/build scripts** (`flake.nix`): the `datastar` module was only in the coverage gate but missing from `nix run .#lint`, `nix run .#test`, and `nix run .#build`. All three scripts now include `(cd datastar && ...)`.
- **Domain model counts corrected** (AGENTS.md, FEATURES.md, ROADMAP.md, docs/DOMAIN_LANGUAGE.md): event payloads 22→21, commands 19→20. `RolesUpdated` event and `UpdateRoles` command annotated as legacy (superseded by `MemberRolesChanged`/`UpdateMemberRoles`). FEATURES.md event breakdown corrected (removed double-counted ExternalAccount/Credentials subsets).
- **Datastar added to all remaining flake.nix scripts**: `test-flake` (3x race), `test-fuzz`, `coverage` (human-readable report), and `check-cqrs-lint`. Also added `loginpage` to `test-flake` and `test-fuzz` (was missing). All 11 production modules now appear consistently in every flake.nix script.
- **Module architecture scripts expanded** (`scripts/check-module-isolation.sh`, `scripts/check-dep-budgets.sh`): added `identity-model`, `dashboardui`, and `datastar` to the isolation check module list and dependency budget table. Budgets set with 20% headroom: identity-model 10 (8 current), dashboardui 16 (13 current), datastar 5 (4 current).
- **docs/guides count corrected in AGENTS.md** (12→14): added `datastar-integration.md` and `mysql-setup.md` to the guide list and categorization description.
- **go.work comment block** (`go.work`): added explanatory comment documenting the `go-idempotency` replace directive rationale (zero pseudo-version transitive dep of go-cqrs-lite/idempotency, no published tags, workspace-level replace needed because module-local replace is not propagated).
- **`.cqrs-lint.json` adopted with `library` preset** (workspace-wide): root-cause fix for workspace-wide feature profile leakage. cqrs-lint auto-detection merged all 19 modules into one profile (`server: true`, leaked from `examples/*`), producing false-positive findings for library code. The `library` preset pins `server=false`, `command-flow=read-only`, `tracing=off`, `snapshot=off` — correct for cqrs-htmx library modules. Disabled F002/F006/F009/F010/F011/S002/S003/S007 (adoption-coaching and consumer-responsibility rules that are false positives for library code) via config, then removed 9 stale inline suppressions for those rules. Result: **121 suppressed (was 130), 0 unsuppressed, 0 stale, clean exit**. Also corrected AGENTS.md: installed version is 4.3.0 (not v0.2.2), comma-separated suppressions ARE supported, end-of-line suppressions are NOT. Closes TODO_LIST P2#1.
- **cqrs-lint inline suppressions consolidated**: reduced total suppressions from 130 to 121 across 3 rounds. Round 1 removed 10 obsolete suppressions (rule already disabled or code fixed). Round 2 relocated 56 stale suppressions to the correct line (parser checks finding's line + line-above only). Round 3 adopted `.cqrs-lint.json` and removed 9 suppressions for config-disabled rules. All remaining 121 suppressions are intentional, documented, and non-stale.

### Deprecated

- **SSE re-export layer deprecated** (`sse_event.go`, `sse_store.go`): all type aliases and delegating functions that re-export go-sse symbols (`SSEEvent`, `SSEEventID`, `SSEStream`, `NewSSEStream`, `NewSSEEventID`, `ParseSSEEventID`, `MustParseSSEEventID`, `WriteSSEEvent`, `SetSSEHeaders`, `LastEventIDFromRequest`, `SSEEventConnected`, `SSEEventHeartbeat`, `ContentTypeSSE`, `SSEEventStore`, `ReplayEvents`) now carry `// Deprecated:` markers directing consumers to import `github.com/larsartmann/go-sse` directly. All internal consumers (root module, dashboardui, examples, e2e) migrated to use `sse.*` directly. Zero SA1019 warnings. The deprecated symbols remain as backward-compat shims (type aliases are transparent, no runtime impact). The genuinely coupled types stay: `Broadcaster`/`WSBroadcaster` (CQRS `AfterDispatchHook` integration), `JournalSSEStore` (go-cqrs-lite journal to `sse.EventStore` bridge), `ServeSSE` (convenience lifecycle handler), and `EventToSSEMapper`.
- **httputil re-export layer deprecated** (`csrf_reexport.go`, `ratelimit_reexport.go`, `server_timing_reexport.go`): all 39 type/var/const aliases that re-export CSRF, keyed rate limiting, and Server-Timing symbols from `github.com/larsartmann/httputil` now carry `// Deprecated:` markers directing consumers to import httputil directly. All internal callers (`csrf_handler.go`, `options_types.go`, `errors.go`, `response.go`, `usermgmt/http.go`, `usermgmt/verification_totp_http.go`), all 14 test files, all 7 examples, adminui/loginpage READMEs, `doc.go`, SKILL.md, and guides migrated to direct `httputil.*` imports. Removal bundled with v5 (parallel to the identity-model re-export deprecation). See `docs/guides/leveraging-httputil.md` for the migration table.

### Fixed

- **BuildFlow templ-generate version mismatch** (`.buildflow.yml`): added `skip_steps: [templ-generate]` to prevent buildflow's bundled templ binary from regenerating `_templ.go` files with directory-prefixed `FileName:` fields that conflicted with the nix-pinned templ version. This was forcing `--no-verify` on commits touching templ files. The `templ-fmt` step still runs (formats `.templ` source files); committed `_templ.go` files are verified by `nix run .#check-codegen`.
- **usermgmt golines lint failure** (`usermgmt/sql_readmodel_extra.go`): struct tags in `MembershipView` and `BotView` used inconsistent alignment spaces that golangci-lint's golines formatter flagged. Changed to single-space tags matching the rest of the struct.
- **dashboardui lint failures** (`dashboardui/config.go`, `dashboardui/handler_overview.go`, `dashboardui/go.mod`): (1) godoclint flagged `config.go` because the cqrs-lint:ignore(E014) comment acted as a second package doc — moved suppression to `go.mod` (module-level). (2) tagliatelle flagged snake_case JSON tags `stream_id`/`event_id` on the display-only `recentEvent` struct — changed to camelCase `streamId`/`eventId` (struct is never JSON-serialized, only rendered via templ).
- **Workspace build broken by go-idempotency zero pseudo-version** (`go.work`): added `replace github.com/larsartmann/go-idempotency => /home/lars/projects/go-idempotency`. Without this, `go build ./...` fails because `go-cqrs-lite/idempotency` depends on `go-idempotency v0.0.0` (zero pseudo-version, repo has no tags). Individual module builds (GOWORK=off) worked because the replace in `go-cqrs-lite/idempotency/go.mod` was visible only in that module's context.
- **7 broken markdown links fixed**: `docs/status/archived/2026-07-20_04-45_*.md` (planning doc moved to `archived/`), `docs/guides/auth-provider-fault-tolerance.md` (removed dead `docs/references/` links), `usermgmt/docs/SQL_STORES.md` (ADR paths `../adr/` → `../../docs/adr/`), `adminui/README.md` (`csrf_middleware.go` → `csrf_reexport.go`).
- **Dead JSON tags removed from dashboardui `recentEvent` struct** (`dashboardui/handler_overview.go`): the struct is never JSON-serialized (rendered only via `fmt.Fprintf` into HTML strings), making the JSON tags dead weight that caused lint friction. Removed all 5 JSON tags and 2 unnecessary `cqrs-lint:ignore(A032)` directives. Updated the A011 suppression comment for accuracy.
- **prewarm-gocache.sh comment drift** (`scripts/prewarm-gocache.sh`): header said "ROOT CAUSE"/"FIX" implying prewarm IS the fix, but `.buildflow.yml` says `max_concurrency: 1` is the definitive fix and prewarm is a performance optimization. Rewrote header to accurately reflect prewarm's role.

- **Offline sync retry pipeline** (`sync/sync-client.js`, `sync/sync-worker.js`): multiple fixes to the offline command retry mechanism discovered via E2E browser testing. (1) **Verb casing**: HTMX 2.x stores `requestConfig.verb` as lowercase ("post"); the envelope now uppercases it for consistency with HTTP method conventions. (2) **Retry trigger**: `retryQueuedCommand` now uses `htmx.ajax()` with the persisted envelope instead of `htmx.trigger(el, "click")`, which only worked if the element itself had an HTMX trigger attribute (forms trigger on submit, not click). (3) **Connectivity detection**: the sync-client now listens for the window `online` event and sends a `flush` message to the SharedWorker, because the SharedWorker scope may not receive `online`/`offline` events reliably in all browsers and test environments. (4) **Dead port resilience**: `pickPort` now uses pure round-robin (removed `originatingTab` optimization that targeted dead ports after tab close), `sendRetry` catches postMessage errors and falls back to round-robin, and `flush()` schedules a periodic 2-second re-flush while commands remain pending (ensuring delivery even when a dead port silently swallows the first attempt). (5) Removed premature flush from the enqueue handler that could burn through `MAX_RETRIES` before the user reconnects. syncVersion bumped to `1.3.0`. All 4 E2E Playwright tests now pass.
- **Offline sync FormData serialization bug** (`sync/sync-client.js`): in HTMX 2.x, `requestConfig.parameters` is a `FormData` object, not a plain object. `postMessage` cannot clone `FormData` across the SharedWorker boundary (structured clone algorithm does not support it), so the command envelope was silently lost and never reached the SharedWorker for persistence. The `htmx:sendError` handler now converts `FormData` to a plain `{key: value}` object before passing it to `postMessage`. This was a production bug that broke ALL offline HTMX form submissions. syncVersion bumped to `1.2.0`.
- **UserDelete cascade fix** (`usermgmt/service_misc.go`): `DeleteUser` now cascades membership removal and bot deletion, matching `DeleteTenant`'s best-effort pattern. Previously, deleting a user left orphaned memberships and bots (whose API tokens still worked). New helpers: `removeMembershipsForUserBestEffort`, `deleteBotsForUserBestEffort`. `FindByOwner(UserID)` added to BotReadModel. Tests: `TestService_DeleteUser_CascadeMemberships`, `TestService_DeleteUser_CascadeBots`.
- **Lockout memory leak fixed** (`usermgmt/service_core.go`): `AccountLockout.EvictStale` was never wired into `NewService` (unlike the 4 other ephemeral stores), causing unbounded growth of lockout maps in production. Fixed by wiring the eviction goroutine via `wireLockoutEviction()` helper.
- **decoder.go cleanup** (`decoder.go`): `readBodyForDecode[T]` simplified from `(T, []byte, error)` to `([]byte, error)` — the unused `T` return was flagged by `unparam`.
- **Canonical nix quality gates verified (pre-httputil-consolidation)**: `nix run .#test` (all 11 module groups pass), `nix run .#coverage-gate` (all 9 gated modules pass with verified thresholds) were run successfully BEFORE the httputil consolidation (see Changed section above). Hermetic builds (`GOWORK=off`) were temporarily blocked until httputil v0.8.0 was published — **subsequently resolved** (see Changed: httputil v0.8.0 published). The workspace build (`go build ./...`) and all tests (`go test ./... -race`) continued to pass via the `go.work` local replace throughout.
- **Nil-body decoder regression test** (`decoder_test.go`): added `TestReadBody_NilBodyDoesNotPanic`, `TestDecodeJSONBody_NilBodyDoesNotPanic`, and `TestDecodeFormBody_NilBodyDoesNotPanic` to lock in the `r.Body == nil` guard in `readBody` (previously caused a nil-pointer panic on hand-constructed or proxied requests). The fix was already in `decoder.go:60`; these tests prevent regression.

### Removed

- **WebSocket transport dropped — library is SSE-only (BREAKING):** the entire WebSocket surface was removed. Deleted source files `ws.go`, `ws_broadcaster.go`, `ws_dispatch.go`, `ws_encoder.go` and tests `ws_test.go`, `ws_dispatch_test.go`, `ws_encoder_test.go`, `ws_end_to_end_integration_test.go`, `example_ws_test.go`. Deleted embedded asset `extensions/ws.min.js`. Removed `HTMXExtWS` constant + CDN URL + extension registry entry. Removed from `ack.go`: `BroadcastOnAckWS()`, `BroadcastOnAckWSFunc()`, `ackToWSMessage`. Removed 14 exported symbols: `WSBroadcaster`, `NewWSBroadcaster`, `WSMessage`, `WSOOBHTML`, `ParseWSMessage`, `ParseWSMessageInto[T]`, `WriteWSMessage`, `WriteWSMessageInto[T]`, `DispatchWSCommand`, `DispatchWSQuery`, `DecodeWSJSON[T]`, `DecodeWSJSONQuery[T]`, `BroadcastOnSuccessWS`/`BroadcastOnSuccessWSFunc`, `BroadcastOnErrorWS`/`BroadcastOnErrorWSFunc`. **Migration:** `WSBroadcaster.Broadcast(msg)` → `Broadcaster.Broadcast(sse.Event{Data: msg})`; `ParseWSMessage(data)` → `json.Unmarshal` into a typed struct (HEADERS separation was a HTMX-WS-extension quirk, not a domain concern); `WSOOBHTML` → `OOBHTML`; `DispatchWSCommand`/`DispatchWSQuery` → `app.Command`/`app.Query` (HTTP POST endpoints, not bi-directional upgrade). The library intentionally avoids pulling in `gorilla/websocket` or similar — consumers needing bi-directional transport should integrate a dedicated WebSocket library directly. SSE covers server→client; HTTP POST covers client→server. ADR-0046 documents the decision; ADR-0004 (SSE/WebSocket Support) and ADR-0010 (Transport Parity) are superseded.

## [v4.6.1] - 2026-07-27

### Changed

- **Dependency bumps** (since v4.6.0): go-cqrs-lite `v4.0.0`/`v4.1.0` → `v4.1.0` (storage/memory `v4.0.0` and snapshot `v4.0.1` brought up to `v4.1.0`; all other modules already at `v4.1.0`), go-sse `v0.2.1` → `v0.3.0`. No cqrs-htmx API change.
- **identity-model event-handling refresh** (`identity-model/fold.go`, `identity-model/membership.go`): simplified `HasRole`/`HasAnyRole` to use `slices.Contains`/`slices.ContainsFunc` instead of manual loops.

### Added

- **identity-model module metadata files** (`identity-model/`): `.editorconfig`, `.gitattributes`, `.gitignore`, `CONTRIBUTING.md`, `CHANGELOG.md` — standard project hygiene for the independently versioned module.

## [v4.6.0] - 2026-07-26

### Added

- **Dedicated `carrierStatus` regression coverage** (`carrier_status_internal_test.go`, `errors_model_test.go`): an internal unit test for `carrierStatus` (nil, plain error, zero-status skip, valid/out-of-range status, chain-walk past a zero carrier to a deeper override, and the real `errorfamily.Rejection` trigger) plus public `MapError` family-default tests pinning the full matrix (Rejection→400, Conflict→409, Transient→503, Infrastructure→503, Corruption→500), a wrapped-Rejection case, and an explicit-override-wins case. Locks down the v4.5.0 fix so the go-error-family v0.8.0 regression (every errorfamily error short-circuiting to 500) cannot return silently.

- **Root HTMX redirect helpers** (`redirect.go`): `HTMXRedirect(w, r, path)` emits an HTMX-aware redirect (`HX-Redirect` for HTMX clients, standard `http.Redirect` otherwise); `SafeRedirectPath(path)` normalizes untrusted paths to site-relative URLs as an open-redirect guard. Extracted during the dedup sweep; exercised by adminui + loginpage handler tests.
- **dashboardui SSE bridge improvements** (`dashboardui/sse.go`, `dashboardui/config.go`): emitted SSE events now carry the domain event ID (enables reconnect dedup); `Config.SSEHeartbeatInterval` (15s default) drives a heartbeat to keep proxies from killing idle connections.
- **dashboardui SSE reconnect replay + lifecycle** (`dashboardui/sse.go`, `dashboardui/dashboard.go`): the SSE handler now reads `Last-Event-ID` on reconnect and replays missed events from the journal via `cqrshtmx.JournalSSEStore` + `cqrshtmx.ReplayEvents` before entering the live loop. On first connect (no cursor), recent history is backfilled (up to `DefaultMaxReplay=1000` events). `Dashboard.Close()` disconnects all SSE clients by closing the broadcaster. Subscribe-before-replay ordering prevents the race where events fire during replay and are lost.

### Changed

- **Dedup sweep — harmful clones driven to zero** (two rounds, 2026-07-26): consolidated the event-catalog and OpenAPI handlers into a shared `immutableJSONServer`; extracted `requireUser` guard, `formatFromQuery`, `setHTMLNoStoreHeaders`, `toMemberRows`, `parseTenantMemberPath`, `reasonedCommand`, shared `cqrshtmx.ToastDetail`, a free `rebuildProjection` function, `writeHTML` helper, and `wsContext` delegation. Clone groups 33→26, harmful clones → 0.
- **Inter-module go.mod version references resolved** (commit `e274540`, `59e33ef`): `usermgmt`, `adminui`, and `loginpage` now reference `cqrs-htmx/v4 v4.5.0` (was stale `v4.4.0`), and the identity-model pseudo-version was replaced with `v4.1.0`. Root cause fixed in `scripts/batch-release.sh` (re-resolves `require` lines after stripping `replace` directives). Workspace builds were unaffected (go.work local replaces); this unblocks `GOWORK=off` resolution for external consumers. The `examples/dashboard-demo/go.mod` zero pseudo-version for `dashboardui/v4` is also resolved in this release (`v4.0.0-000...` → `v4.0.0`).
- **Dependency bumps** (since v4.5.0, verified against the `v4.5.0` tag): go-error-family `v0.8.0` → `v0.10.0`, templ-components `v1.1.0` → `v1.2.0` (adminui + admin-demo), go-sse `v0.2.0` → `v0.2.1`, httputil `v0.6.0` → `v0.6.1`. No cqrs-htmx API change.
- **Release-checklist pre-tag lockstep detection** (`scripts/release-checklist.sh`): the script now detects when HEAD is not tagged and marks sub-module test/build/check-modules failures as EXPECTED (they resolve once modules are tagged and pushed). Lint failures are marked as pre-existing nits with a recompute command. Adds a `go.work` vs `go.mod` go-directive consistency check.

### Fixed

- **Eliminated 11 stdlib error constructor violations** (`event_store_sse.go`, `dashboardui/handlers.go`, `dashboardui/payload.go`, `usermgmt/store.go`): migrated all `fmt.Errorf`/`errors.New` calls in non-test code to `errorfamily` constructors (`WrapInfrastructure`, `WrapCorruption`, `Newf`, `NewConflict`). `nix run .#errorfamily` now reports 0 violations across all modules.
- **Fixed `Dashboard.Close()` event-bus subscription leak** (`dashboardui/dashboard.go`, `dashboardui/sse.go`): `Close()` now signals a done-channel that makes the event-bus handler a no-op before closing the broadcaster. Uses `sync.Once` for idempotent shutdown. The underlying `event.Bus` has no `UnsubscribeAll` (tracked in ROADMAP for upstream), so the handler remains registered but becomes inert.
- **Removed accidentally-tracked `examples/dashboard-demo/dashboard-demo` binary** (commit `f25599a`): the ~12 MB compiled demo had been committed to the repo; it is now gitignored and removed from history going forward.

### Documented

- **Data-mesh research + proposal** (`docs/research/2026-07-25_*`, `docs/proposals/2026-07-25_data-mesh-interchange.md`): landscape analysis (why event sourcing structurally prevents data-discovery pain) and a proposal recommending cqrs-htmx adopt go-cqrs-lite `catalog/v4` plus three small runtime pieces (Channel-to-Bus binding, CloudEvents envelope, pull-based transport) rather than building a bespoke mesh. Status: researched, not yet adopted — see ROADMAP.md.

## [v4.5.0] - 2026-07-24

### Added

- **dashboardui module** (`dashboardui/v4`): NEW MODULE — CQRS/ES observability dashboard (templ + HTMX + Tailwind v4). Overview dashboard with stats cards, event stream browser with payload inspection, aggregate browser with event timeline, projection health dashboard, command/query audit panels, snapshot inspector, time-travel inspector (unique differentiator — replay aggregate state at any historical version), SSE live updates. `examples/dashboard-demo/` demonstrates the integration. dashboardui/v4.0.0 tagged.
- **identity-model module** (`identity-model/v4`): NEW MODULE — pure domain-model module for event-sourced identity management (ADR-0043). Contains ALL domain types (UserID, TenantID, BotID, ActorID, Session, User, Membership, ExternalAccount, WebAuthnCredential), event payloads (22 structs), commands (19 structs with accessor methods), state structs + fold functions (FoldUser/FoldMembership/FoldTenant/FoldBot), Authz engine (Casbin-backed — ADR-0044), RBAC model + default policies + role hierarchy, domain errors (errorfamily-only, no HTTP dependency), crypto helpers, upcaster registry, exported constants (41 event/command/aggregate-type constants). Casbin is a first-class dependency. identity-model/v4.1.0 tagged.
- **Event catalog** (`event_catalog.go`, `event_catalog_handler.go`): `EventCatalog` type with `Register`/`Events`/`JSON` methods. `EventCatalogHandler(catalog)` serves immutable JSON with 1-year cache and FNV-1a ETag. Published Language for projection builders.
- **Projection status handler** (`projection_status_handler.go`): `ProjectionStatusHandler(provider)` serves live projection health as JSON with `no-cache` semantics. `ProjectionStatusProvider` interface implemented by `*usermgmt.Service` and `*usermgmt.EventSourcedSetup`.
- **usermgmt event catalog** (`usermgmt/es_event_catalog.go`): `DefaultEventCatalog()` pre-populates all 21 event types with descriptions and payload metadata. `EventCatalog()` method on `EventSourcedSetup` and `Service`.
- **usermgmt projection health and rebuild** (`usermgmt/es_projection_health.go`): `ProjectionStatuses()` and `RebuildProjection(ctx, name)`. Rebuild stops host, resets checkpoint + read-model, creates fresh host, replays journal.
- **Type-safe command/query HTTP handlers** (`app.go`, `handler.go`, `options_decode.go`): `CommandTyped[Q](app, type, opts...)` and `QueryTyped[Q, R](app, type, opts...)` — package-level generic functions. Paired with `DecodeJSONTyped[Q]` / `DecodeFormTyped[Q]` and `DecodeAndValidateJSON[T]` / `DecodeAndValidateForm[T]` for one-step decode + validate.
- **Partial rendering helpers** (`partial.go`, `options_render.go`): `RenderPartialOrFull[T](partial, full)`, `RenderPartialOrFullFunc`, `RenderIf`, `RenderTemplComponent`, `OOBHTML` — eliminates boilerplate HTMX partial/full branching.
- **OpenAPI 3.1 spec generation** (`openapi/`, `options_openapi.go`): Dependency-free fluent builder for OpenAPI 3.1 documents. `WithOpenAPI(op)` attaches metadata; `OpenAPISpecHandler(spec)` serves `/openapi.json` with eager serialization (concurrency-safe, immutable), FNV-1a ETag, 1-year cache.
- **Offline sync extracted to root module** (ADR-0042): `SyncWorkerHandler()` / `SyncClientHandler()` serve the SharedWorker + sync client from the root module. `With` variants (`SyncWorkerHandlerWith` / `SyncClientHandlerWith`) allow custom JS. Any consumer can wire offline sync with two handler mounts + one `<script>` tag. adminui delegates to root.
- **Offline command queue — production hardening** (`sync/sync-worker.js`, `sync/sync-client.js`): IndexedDB as single source of truth. Fixed double-retry race, port leak, null-envelope poison commands. Added retry count (max 10) + TTL (24h) eviction, staggered delivery, targeted retry. `data-sync-worker-url` attribute override. ADR-0040.
- **Opt-in aggregate snapshotting** (`usermgmt/snapshot*.go`, ADR-0041): `SnapshotConfig` (Store/Codec/Strategy) wired through every repository path. Zero behavior change when unconfigured.
- **Enum `Valid()` methods** (`usermgmt`): `Valid()` on `Action`, `Effect`, `Role`, `UserDataFormat`, `AckStatus`. Non-breaking. Also adds `errDecoderReturnedNil` (Corruption/500).
- **`SyntheticUserID` + `GenerateUserID`** (`usermgmt`, root): Explicit SHA-256-derive vs ULID generation. `NewUserID` deprecated for non-ULID strings.
- **Admin UI: OAuth2 external-account views** (`adminui`): External-account card on user detail page. `usermgmt.NewExternalAccount(...)` public constructor.
- **Keyboard accessibility + reduced-motion guard** (`adminui`, `loginpage`): `:focus-visible` outlines, `@media (prefers-reduced-motion: reduce)` guard.
- **7 operational guides** (`docs/guides/`): consistency-model, event-replay-and-rebuild, auth-provider-fault-tolerance, event-store-storage-health, event-catalog-guide, projection-health-monitoring, rebuild-projection-runbook.
- **ROADMAP v5 decomposition plan** (`ROADMAP.md`): Module boundaries, decomposition trigger criteria, cost/benefit analysis.
- **Sync handler `With` variants + godoc examples + delegation tests + version-drift test** (`sync_serve.go`, `sync_serve_test.go`, `example_htmx_test.go`, `adminui/coverage_gaps2_test.go`).
- **SSE broadcaster load benchmarks** (`sse_broadcaster_bench_test.go`): Fan-out across {1, 10, 100, 1000} subscribers.
- **Property-based tests for event folds** (`usermgmt/property_sequences_test.go`): 13 sequence-property tests.
- **dedup.Ring benchmark — 100K tier** (`usermgmt/es_dedup_benchmark_test.go`).
- **Projection replay benchmark** (`usermgmt/es_projection_replay_bench_test.go`): ~3µs/event, linear scaling, 10K events = 30ms.

### Changed

- **usermgmt wired to identity-model** (`usermgmt/`): All domain type definitions replaced with type aliases. Command dispatch uses accessor methods. Errors re-exported with `WithHTTPStatus`. Fold functions, upcaster registry, and constants aliased via `var`. Split brain eliminated.
- **go-cqrs-lite API updates propagated** (workspace-wide): `AggregateID()` → `StreamID()`, `AggregateType` → `StreamType`, `event.AggregateRef` → `id.StreamRef`, `snapshot.AggregateID/Type` → `StreamID/Type`.
- **Constants + fold functions + upcaster registry consolidated to identity-model**: 41 constants exported, fold functions replaced with aliases, upcaster registry moved. `es_upcaster.go` + `es_upcaster_test.go` deleted from usermgmt.
- **identity-model + dashboardui added to flake.nix**: Both modules included in all 7 flake targets.
- **ADRs**: Added ADR-0043 (identity-model extraction) and ADR-0044 (Casbin as first-class dependency).
- **Projection lifecycle: adopted `projectionhost/v4`** (`usermgmt/es_projection_setup.go`, ADR-0031 Superseded): Replaced 155-line hand-rolled logic with `projectionhost.Host`. Per-projection goroutines, per-projection checkpoints, DLQ, crash auto-restart, replay→live handoff via `WithSubscriber`. `StartProjections` returns `(*projectionhost.Host, error)`. **Breaking for checkpoint users:** per-projection keys replace former single key.
- **Projection host factory deduplicated**: Shared `startProjectionHost` helper used by both `StartProjections` and `RebuildProjection`.
- **go-error-family v0.7.0 → v0.8.0**: `errors.AsType[E]` generic helper adopted. `carrierStatus` chain-walking bug fixed (treats `HTTPStatus()==0` as "not set").
- **go-sse v0.2.0**: SSE library extracted to separate `go-sse` module. Root `go.mod` references published `v0.2.0` (no longer pseudo-version). go.work replace removed.
- **Sync JS modernized**: `var` → `const`/`let`, `VERSION = "1.1.0"`, `@ts-check`, JSDoc on all public functions.
- **Sync ETag prefix unified**: `"cqrshtmx-sync-worker-%s"` → `"sync-worker-%s"` (consistent with `"htmx-%s"`).
- **`WSOOBHTML` is now a delegate** (`ws.go`): Delegates to `OOBHTML`. Fully backward compatible.

### Fixed

- **`carrierStatus` chain-walking bug** (`errors_status.go`): `MapError` returned 500 for ALL errorfamily errors because `carrierStatus()` didn't treat `HTTPStatus()==0` as "not set". Fixed: zero status now falls through to family default.
- **Silent `UserIDExtractor` failures now logged at Error** (`enrich.go`): Failing extractor is now loud instead of silently degrading to anonymous.
- **CSRF trusted-proxies warning no longer global** (`csrf_middleware.go`): Dropped package-global `sync.Once`. Each construction evaluated independently.
- **`decode-nil` classified as Corruption (500)** (`handler.go`): Decoder returning `(nil, nil)` is now Corruption (500), not Infrastructure (503).
- **Magic numbers extracted** (`fanout.go`, `usermgmt/http.go`): Named constants + `nolint` annotations with justifications.

## [v4.3.0] - 2026-07-12

### Changed

- **go-cqrs-lite v3.7.4 → v4.0.0**: All modules now depend on go-cqrs-lite v4. Import paths change from `go-cqrs-lite/*/v3` to `go-cqrs-lite/*/v4`. The v4 API is backward-compatible (compatibility aliases for `event.AggregateType`, `event.AggregateRef`, etc.). Consumers must update import paths and run `go get` for all transitive go-cqrs-lite modules at v4.0.0 (publishing bug workaround). See `docs/migrations/v3-to-v4.md` for details.
- **`.vendor-local/eventtest` eliminated**: The vendored eventtest module is removed entirely. All `replace` directives for `eventtest` removed from go.mod files. Use `go mod tidy -e` for the root module under `GOWORK=off` (upstream publishing bug — non-fatal).

### Breaking

- **`CSRFTestToken` signature changed** (`csrf_testing.go`): Was `func(CSRFMiddleware) string`, now `func(CSRFMiddleware) (string, *http.Cookie)`. The old API only returned the masked token but not the CSRF cookie, making it impossible to construct a valid POST request in tests (nosurf requires both token in form/header AND cookie). Tests calling `CSRFTestToken(mw)` must update to unpack the tuple: `token, cookie := CSRFTestToken(mw)`.

### Added

- **`Code` field on `StructuredError`** (`structured_error.go`): Populated via `errorCode(err)`, matching the `"code"` field emitted by `JSONErrorHandler`. Clients now get the same machine-readable error code regardless of whether the response is `application/json` or `application/problem+json`. Empty when the error has no `Code()` method.
- **Error context logging** (`logging.go`): `RequestLoggingSlog` now includes `error_code`, `error_family`, and `error_ctx_*` attributes when a dispatch error occurs. Traverses the full error chain to extract context from wrapped errors.
- **`writeDispatchError` helper** (`usermgmt/http.go`): Consolidates the `writeDispatchError(w, r, err)` pattern across all usermgmt HTTP handlers. Includes the error's `code` and the request ID in the JSON response body when available. Replaces 15 `writeError(w, errorStatus(err), err.Error())` call sites.
- **CBOR codec tests** (`usermgmt/es_state_test.go`): 4 new tests verifying `codec.ForEncoding` round-trip, fold-user with CBOR-encoded events, and mixed JSON+CBOR event stream folding.
- **Request-aware decoder tests** (root `feedback_features_test.go`): 3 new BDD tests for `DecodeFormWithRequest`, `DecodeJSONQueryWithRequest`, `DecodeFormQueryWithRequest`.
- **ProblemDetailsErrorHandler Code field tests** (`errors_model_test.go`): Verifies `code` field is included for classified errors and omitted for plain errors.
- **writeDispatchError unit tests** (`usermgmt/http_dispatch_error_test.go`): 4 tests covering code field, request ID inclusion, conflict status derivation, and nil-request handling.
- **OAuth2 HTTP handler tests** (`usermgmt/oauth2_http_test.go`): 9 tests covering all OAuth2 HTTP handlers: begin success, callback missing code/state, callback invalid state, callback success flow, callback error redirect, callback success redirect, unlink unauthenticated, unlink not-linked, unlink success.
- **adminui coverage gap tests** (`adminui/coverage_gaps2_test.go`): 17 tests covering New() validation, auth checks, not-found paths, asset serving (CSS/JS/sync-worker/htmx), diverse user/tenant rendering.
- **SSE broadcaster lifecycle tests** (`sse_broadcaster_test.go`): 4 tests for OnSubscribe/OnUnsubscribe hooks: fire verification, no-op for unknown channels, concurrent race test (20 goroutines x 100 cycles).
- **OAuth2 integration tests** (`integration_test/oauth2_integration_test.go`): 3 integration tests for full end-to-end OAuth2 finish login with mock token exchange server, invalid state rejection, provider mismatch rejection.
- **Docs freshness checker** (`scripts/check-docs-freshness.sh`, wired as `nix run .#check-docs-freshness`): Checks AGENTS.md version strings against go.mod, Go version refs, HTMX version refs, and deprecated API references. Integrated into `nix run .#check-modules`.

### Changed

- **`writeDispatchError` wires request ID** (`usermgmt/http.go`): Changed `_ *http.Request` to `r *http.Request`, extracting the request ID from context and including it as `request_id` in the JSON error response body. The `requestIDKey` constant was added alongside `codeKey`.
- **`ErrorCode` exported** (`errors.go`): The internal `errorCode` function is now exported as `ErrorCode`. `writeDispatchError` in usermgmt now uses `cqrshtmx.ErrorCode(err)` instead of `errors.As`, matching root's deepest-code traversal pattern. This ensures the domain-specific code (e.g. `"usermgmt.email_exists"`) is surfaced, not the infrastructure wrapper code.
- **`ErrorRecorder` extracted from `StatusRecorder`** (`logging.go`): The dispatch-error capture concern is now a separate `ErrorRecorder` struct, embedded by `StatusRecorder` via composition. `SetDispatchError` and `DispatchError()` are exported. Fixes the SRP violation where `StatusRecorder` had two responsibilities.
- **OAuth2 error context** (`usermgmt/service_oauth2.go`): `BeginOAuthLogin` and `FinishOAuthLogin` now attach `provider` context to all error wrapping sites (5 call sites: state save/consume errors, provider mismatch rejection, userinfo unmarshal, missing-email rejection), enabling better debugging of provider-specific failures.
- **`errors.AsType` adoption** (`usermgmt/service_register.go`): Replaced `errors.As(err, &ee)` with `errors.AsType[*event.Error](err)` per gopls recommendation.
- **`UserReadModel.Handle` refactored** (`usermgmt/es_readmodel.go`): Extracted the 12-case switch into a dispatch table (`handlers` map) with per-event handler methods and a shared `decodePayload[T]` generic helper. Eliminates the `maintidx` lint warning (complexity 38). The last remaining lint issue in the entire codebase is now resolved.
- **coreos/go-oidc/v3 v3.19.0 → v3.20.0**: Bumped to latest released version in the `usermgmt/oauth2` module.
- **go-error-family v0.6.1 → v0.7.0**: Upgraded across all modules. v0.7.0 adds `errors.AsType[E]` generic helper and refines the `Family.HTTPStatus()` contract.
- **templ-components v0.15.0 → v0.16.0**: Bumped in `adminui` and `examples/admin-demo` for latest component additions and bug fixes.

### Fixed

- **`encoding/json` v2 migration reverted**: 26 source files reverted from `encoding/json/v2` + `jsontext` back to stdlib `encoding/json` after automated migration broke the build. The project uses `encoding/json/v2` via `GOEXPERIMENT=jsonv2` (a compiler flag, not an import path) — the migration tool confused the two.
- **OAuth2 HTTP handler nil panic** (`usermgmt/oauth2_http.go`): `oauth2Error()` method signature changed from `(w, status, message)` to `(w, r, status, message)`. The old code passed `nil` as the `*http.Request` to `http.Redirect()`, which panics in Go 1.26. All 3 call sites in `handleOAuth2Callback` updated.
- **TOTPPendingSecretTTL default** (`usermgmt/totp.go`, `usermgmt/service_core.go`): Extracted `defaultTOTPPendingTTL = 5 * time.Minute` as a named constant. The default is now set at `NewService` init time instead of as an inline fallback at the use site, making the configuration consistent with other TTL fields.
- **Exhaustive lint in `usermgmt/service_register.go`**: Added explicit `case event.Transient, event.Corruption, event.Infrastructure:` to the `classifyDispatchError` switch (same body as `default`). Eliminates the exhaustive linter warning.
- **Split brain: `ProblemDetailsErrorHandler` vs `JSONErrorHandler`**: `ProblemDetailsErrorHandler` (which uses `StructuredError`) was missing the `code` field that `JSONErrorHandler` emitted. Now both paths emit the same `code`.
- **Last remaining lint issue resolved**: `usermgmt/es_readmodel.go:Handle` `maintidx` warning (complexity 38) is eliminated. **All modules now report 0 lint issues.**

### Documented

- **Empty-body behavior in `DecodeJSON`/`DecodeJSONQuery` godoc**: Documented that an empty body on GET requests produces a zero-value `T` (not an error).
- **Provider implementation guide** (`docs/guides/provider-implementation.md`): Covers all 3 auth provider interfaces (`TOTPProvider`, `WebAuthnProvider`, `OAuth2Provider`) with method signatures, key design points, and references to existing implementations.
- **Release process documentation** (`CONTRIBUTING.md`): Pre-release checklist (8 verification steps: test, build, lint, errorfamily, check-modules, coverage-gate, fmt, flake-check), loginpage tagging instructions, go-cqrs-lite v4.0.0 publishing bug workaround, and `encoding/json/v2` usage documentation.

## [v4.2.1] - 2026-07-08

### Added

- **Per-module CHANGELOGs**: Created CHANGELOG.md files for all 6 sub-modules (usermgmt, usermgmt/totp, usermgmt/webauthn, usermgmt/oauth2, adminui, loginpage) so consumers can track changes per module independently.

### Fixed

- **go.work replace+use conflict**: Moved `eventtest` from go.work `replace` directive to `use` block. Having a module in both `use` and `replace` caused a Go workspace error (`workspace module is replaced at all versions`) that broke BuildFlow test-race, go-fix, and govalid-generate steps. Per-module go.mod `replace` directives retain GOWORK=off compatibility.
- **go-cqrs-lite version drift**: Aligned all go-cqrs-lite modules to their latest available tags. Several modules (decider, projection, stack, storage, watermill, listing, scheduling, scenario, stack/sqlite, stack/postgres, catalog, encryption, signing) were pinned at v3.7.0 while others were at v3.7.4. The `.vendor-local/eventtest` copy was also updated to v3.7.4 with `go-error-family` constructors replacing removed `event.*` error helpers.

## [v4.2.0] - 2026-07-07

### Added

- **RequestGuard** (`authz.go`): New `RequestGuardFunc(http.Request, any) error` custom auth guard that runs after decode but before dispatch. Enables auth models that don't fit the Casbin Enforcer pattern — cookie-based player IDs, ownership checks, API key validation — without needing middleware wrappers. The guard receives the decoded command/query so it can inspect fields directly.
- **Request-aware decoders** (`options_decode.go`): Four new `*WithRequest` variants (`DecodeJSONWithRequest`, `DecodeFormWithRequest`, `DecodeJSONQueryWithRequest`, `DecodeFormQueryWithRequest`) that pass the `*http.Request` to the mapper function, so consumers can extract cookies, headers, or path values during the mapping step.
- **SSE channel lifecycle** (`sse_stream.go`): `OnDisconnect(fn)` callback registration for cleanup when clients disconnect.
- **SSE event constants** (`sse_event.go`): `SSEEventConnected` and `SSEEventHeartbeat` named constants for the ACK/heartbeat protocol.
- **go-error-family adoption** (all modules): Migrated from transitive dependency to direct import across root, usermgmt, and all auth strategy modules. Enriched error contexts with domain-specific identifiers (user IDs, provider names, credential IDs) for better debugging.
- **`DefaultRateLimiterConfig()`** (`ratelimit_config.go`): Constructor returning a sensible default `RateLimiterConfig` (token-bucket, 10 req/s burst 20) so consumers don't have to build the struct from scratch.
- **`SecurityHeaderSkip` sentinel** (`security_headers.go`): Pass as a header value to `SecurityHeadersMiddleware` to skip all security headers on a specific response (e.g. for HTML pages that embed inline styles from a trusted source).
- **`RenderHTML(html)` HandlerOption** (`options_render.go`): Renders raw HTML strings (non-templ) as the handler response. Use for static HTML, redirects, or when using `html/template` instead of `templ`.
- **`Broadcaster.Close()` + `fanOut.Close()`** (`sse_broadcaster.go`, `fanout.go`): Graceful shutdown — closes all subscriber channels and stops accepting new subscribers. Idempotent. Enables clean shutdown of SSE broadcasters during `Service.GracefulClose`.

### Changed

- **go-error-family v0.5.1 → v0.6.1**: Upgraded from transitive (via go-cqrs-lite event/v3) to direct dependency. `ErrDispatchFailed` now natively classified; old `sync.Once` + `RegisterClassification` machinery removed.
- **go-cqrs-lite v3.5.0 → v3.7.4**: Adopted `dedup.Ring` (O(1) bounded memory for replay→live dedup), CBOR codec support (`codec.ForEncoding` for per-event codec resolution), and all v3.7.x improvements.
- **usermgmt projection dedup**: Replaced unbounded `map[id.EventID]struct{}` with `dedup.Ring` (1024 entries, ~90KB fixed) in `es_projection_setup.go`. Memory is now O(1) regardless of journal size.
- **usermgmt payload decoding**: `unmarshalPayload` now resolves codec per-event via `codec.ForEncoding(evt.Encoding())` instead of hardcoded `json.Unmarshal`. Consumers who set `event.DefaultCodec = codec.CBORCodec{}` get transparent CBOR support.

### Fixed

- **GET decoder bug**: Fixed in `options_decode.go` — query parameter decoding was not correctly handling all edge cases (discovered during SKILL.md rewrite).

## [v4.1.1] - 2026-07-04

### Changed

- **httputil v0.3.0 → v0.4.0**: Transitive dependency bump across all modules. The root module consumes only `httputil.ClientIP` (in `KeyExtractorFromClientIP`); v0.4.0 adds middleware, an `httpspec` subpackage, and infrastructure types — none on cqrs-htmx's consumed surface. No API or behavior change.
- **templ-components v0.6.0 → v0.6.1**: Patch bump of the `adminui` direct dependency (also picked up by `examples/admin-demo`).
- **flake.lock**: nixpkgs revision refresh (no package version changes).

## [v4.1.0] - 2026-07-04

### Added

- **Embedded HTMX extensions** (`htmx_extensions.go`): SSE (htmx-ext-sse 2.2.4), WS (htmx-ext-ws 2.0.4), and idiomorph (0.7.4) JS files embedded via `go:embed`. These are the three extensions with direct server-side counterparts in the library (SSEStream/Broadcaster, WSMessage/WSBroadcaster, morph-swap for SSE partials).
  - `HTMXExtensionHandler(name)`: Serve a single embedded extension. Constants: `HTMXExtSSE`, `HTMXExtWS`, `HTMXExtIdiomorph`.
  - `HTMXExtensionsHandler(names...)`: Serve a concatenated bundle (one HTTP request instead of N).
  - `HTMXExtensionVersion(name)`, `HTMXExtensionNames()`: Introspection helpers.
  - `HTMXExtensionCDNScriptTag(name)`: CDN `<script>` tag fallback for consumers who prefer CDN.
  - Same caching as `HTMXScriptHandler` (ETag, Cache-Control 1yr immutable, 304 Not Modified).

### Changed

- **HTMX core bumped 2.0.9 → 2.0.10**: Updated embedded `htmx.min.js` and all version references.
- **serveJS helper extracted**: `HTMXScriptHandlerWith` now delegates to shared `serveJS(js, etag)` function, eliminating duplication with extension handlers.

## [v4.0.1] - 2026-07-02

### Added

- **Configurable TOTP pending-secret TTL** (`ServiceConfig.TOTPPendingSecretTTL`): Was hardcoded to 5 minutes, now configurable. Defaults to 5 minutes when ≤ 0. Mirrors the `WebAuthnSessionTTL` pattern.
- **Configurable WebAuthn session TTL** (`ServiceConfig.WebAuthnSessionTTL`): Was hardcoded to 5 minutes, now configurable. Already shipped in v4.0.0 but missing from the v4.0.0 changelog.
- **TOTP cross-module integration test** (`integration_test/totp_integration_test.go`): Full Service.EnableTOTP → totp.Provider.GenerateSecret → VerifyTOTPSetup → VerifyTOTP chain through the provider boundary. Includes nil-provider guards.
- **OAuth2 cross-module integration test** (`integration_test/oauth2_integration_test.go`): Full Service.BeginOAuthLogin → oauth2.Provider.BeginLogin (PKCE + redirect URL) flow. Includes unknown-provider and nil-provider guards.
- **JSON serialization boundary benchmarks** (`usermgmt/webauthn_benchmark_test.go`): `BenchmarkMarshalWebAuthnUser` (2 creds: 1.2µs, 4 allocs) and `BenchmarkMarshalWebAuthnUser_NoCreds` (0 creds: 400ns, 3 allocs). Confirms negligible overhead for ceremony paths.
- **Coverage tests for parse helpers** (`usermgmt/coverage_parse_helpers_test.go`): `ParseUserID`, `MustParseUserID`, `ParseActorID` round-trip, `ParseActorID` no-prefix fallback, `MustParseEmail` panic.
- **Pre-generated RSA key fixture** for OAuth2 OIDC tests: `sync.Once` cache eliminates ~50ms RSA key generation per test invocation. Shared across all `fakeOIDCServer` tests.
- **Fuzz tests on JSON boundary** (`usermgmt/webauthn_fuzz_test.go`, `usermgmt/webauthn/provider_fuzz_test.go`): `FuzzMarshalWebAuthnUser`, `FuzzParseUser`, `FuzzParseSession` — crash-tested with 400K+ iterations.
- **Compile-time interface assertions** (`integration_test`): `var _ TOTPProvider = (*usermgmt.Service)(nil)` and equivalent assertions for `WebAuthnProvider` and `OAuth2Provider` prove at compile time that the core `Service` satisfies each auth strategy interface. Prevents silent interface drift.

### Changed

- **OAuth2 lint config**: Added `gochecknoglobals` exclusion for test files (RSA key fixture uses package-level `sync.Once` pattern).
- **CONTRIBUTING.md**: Updated module table from 8 to 11 modules. Added auth sub-module structural typing explanation and dependency direction.
- **DOMAIN_LANGUAGE.md**: Added `TOTPProvider`, `WebAuthnProvider`, `OAuth2Provider`, `WebAuthnSessionTTL` terms.
- **ADR-0035**: Updated JSON serialization section with benchmark data (400ns–1.2µs) and fuzz/integration test references.

### Fixed

- **Stale doc references**: AGENTS.md coverage stat (95.4%/80.1% → 94.3%/74.5%), README.md go-cqrs-lite version (v3.1.0 → v3.5.0), README.md file tree (WebAuthnConfig → WebAuthnSessionTTL, added totp/webauthn/oauth2 dirs), TODO_LIST.md coverage stat and deferred items.
- **Migration guide**: Removed stale "pending" references in dependency impact table.

## [v4.0.0] - 2026-07-02

### Breaking Changes — Auth Strategy Extraction (Sollbruchstellen)

All three auth strategies (TOTP, WebAuthn, OAuth2) extracted behind primitive-type interfaces as independent Go modules. Consumers import only the auth strategies they need — zero transitive dependencies on pquerna/otp, go-webauthn, golang.org/x/oauth2, coreos/go-oidc, or go-jose unless they explicitly opt in.

#### New Modules

- **`usermgmt/totp/v4`** — TOTP MFA (pquerna/otp). Implements `TOTPProvider`.
- **`usermgmt/webauthn/v4`** — WebAuthn passkeys (go-webauthn). Implements `WebAuthnProvider`.
- **`usermgmt/oauth2/v4`** — OAuth2/OIDC (golang.org/x/oauth2 + coreos/go-oidc). Implements `OAuth2Provider`.

#### TOTP Changes

- `ServiceConfig.TOTPConfig *TOTPConfig` → `TOTP TOTPProvider`
- `TOTPConfig`, `TOTPTimeStep`, `TOTPDigits`, `TOTPSecretLength` removed from core
- New `TOTPProvider` interface: `GenerateSecret(string) ([]byte, string, string, error)`, `ValidateCode([]byte, string) bool`
- `TOTPVerifier` interface removed (was a ghost — never wired)

#### WebAuthn Changes

- `ServiceConfig.WebAuthnConfig *WebAuthnConfig` → `WebAuthn WebAuthnProvider`
- `WebAuthnConfig` type removed from core (now `webauthn.Config` in the webauthn module)
- `webauthn_adapter.go` moved to webauthn module
- `WebAuthnSessionStore` interface changed: `*webauthn.SessionData` → `[]byte`
- `BeginRegistrationResponse.Options` / `BeginLoginResponse.Options` changed to `json.RawMessage`
- In-memory session store uses TTL-based eviction (5 min default)
- Virtual authenticator test removed from core (needs go-webauthn)

#### OAuth2 Changes

- `ServiceConfig.OAuth2Config *OAuth2Config` → `OAuth2 OAuth2Provider`
- `OAuth2Config`, `OAuth2ProviderConfig` removed from core (now `oauth2.Config`, `oauth2.ProviderConfig`)
- New `OAuth2Provider` interface: `BeginLogin(ctx, provider, state) (redirectURL, pkceVerifier, error)`, `FinishLogin(ctx, provider, code, pkceVerifier) (userInfoJSON, error)`
- `OAuth2UserInfo` exported in core (Service deserializes provider's JSON return)
- `OAuth2StateStore.Save` signature changed: `Save(state, provider, pkceVerifier, ttl) error`
- OIDC discovery moved from Service.initOAuth2 to `oauth2.New()`
- `initOAuth2` and `getOAuth2Provider` methods removed from Service

#### Module Paths

- All import paths changed `/v3` → `/v4` across all modules

See `docs/migrations/v3-to-v4.md` for detailed before/after examples.

## [3.5.0] - 2026-07-01

### Added

#### Error Model Overhaul

- **HTTPStatusCarrier interface** (`errors_status.go`): Errors can now pin a specific HTTP status via `WithHTTPStatus(err, status)`. The wrapper preserves the cause's error family + sentinel identity (`errors.Is` traverses). Replaces the old `errorStatus()` split brain in usermgmt — usermgmt sentinels now use `withStatus()` which delegates to `WithHTTPStatus`. See ADR-0034.
- **ProblemDetailsErrorHandler**: Emits `StructuredError` as `application/problem+json` — unified RFC 7807 shape across all transports (HTTP, SSE, WS). Opt in via `Config.ErrorHandler`.
- **StructuredError enrichment**: Now exposes `Message`, `Why`, `Fix` fields (RFC 7807 extensions from `Family.DefaultMessage/Why/Fix`). Same JSON shape across HTTP/SSE/WS.
- **5xx detail redaction**: `SafeDetail(err, status, includeInternal)` replaces 5xx error text with the family's public-safe default message. 4xx detail (raw error) is preserved. `Config.IncludeInternalDetails` opts back in for dev. `StructuredError.Detail` is also redacted for SSE/WS.
- **Exported auth error codes**: `CodeUnauthorized`/`CodeForbidden` — compile-time-safe shared between root and usermgmt.

#### Architecture Enforcement

- **CI module architecture scripts** (4 scripts): `check-module-isolation.sh` (GOWORK=off build+vet per module), `check-dep-budgets.sh` (per-module max production deps), `check-version-drift.sh` (sibling version consistency), `check-replace-directives.sh` (no absolute paths). Wired as `nix run .#check-modules` and as a `module-architecture` CI job in `.github/workflows/ci.yml`.
- **Auth strategy interfaces** (`usermgmt/auth_interfaces.go`): `TOTPVerifier`, `WebAuthnProvider`, `OAuth2Provider` interfaces. Non-breaking, additive. Compile-time assertion `var _ TOTPVerifier = (*Service)(nil)` proves Service already satisfies the TOTP contract. Prepares the v4 extraction seam.

### Changed

- **go-cqrs-lite upgraded to v3.5.0**: All modules updated. v3.5.0 ships CBOR codec support, stricter encoding validation, storage package restructure into view/relational/ subdirs (with backward-compatible aliases), and lint cleanup across 10 modules.
- **Generic `errors.AsType`**: Replaced manual error chain type assertions with the generic helper from go-error-family.
- **Command decode error wrapping**: Decode failures now wrapped with context and family parity (`event.Wrapf` with `event.Classify(err)`).

### Fixed

- **usermgmt `errorStatus` split brain eliminated**: The old `errorStatus()` function in usermgmt was a separate mapping path from root's `MapError`. Now unified through `HTTPStatusCarrier` — one resolution path across modules.
- **Root module go.sum gap**: Missing `storage/memory/v3` and `snapshot/v3` entries caused `GOWORK=off go vet` to fail. Fixed via targeted `go get` + `go mod tidy`.
- **Root dep budget regression**: v3.5.0 upgrade promoted 7 indirect deps to direct (23 vs budget 18). Fixed via `go mod tidy -e` — properly demoted back to 16 direct deps.
- **Version drift eliminated**: All go-cqrs-lite sibling modules (snapshot, schema, storage/memory, listing, scenario) now pinned to consistent versions across root, usermgmt, adminui, integration_test, and all examples.
- **wrapcheck lint in usermgmt**: `forbidden()` now routes through `withStatus()` like all its siblings; `withStatus` marked as wrapping boundary.

### Rejected

- **ADR-0030 (Phase 2b IndexedDB)**: Marked REJECTED. Client-side persistence for the SharedWorker queue is a fundamentally inconsistent API surface that doesn't belong in a server-side Go library.

## [3.3.0] - 2026-06-29

### Added

- **Offline command queue — Phase 2a** (ADR-0029): `adminui/assets/sync-worker.js` — a SharedWorker (~80 lines vanilla JS) that queues command IDs when the network is down and tells tabs to retry on reconnect. The worker is a coordinator, not a proxy: it does NOT send HTTP requests (HTMX does), does NOT own SSE (per-tab EventSource), does NOT persist to disk (in-memory). Reactive detection via `htmx:sendError`. Served at `GET /-/sync-worker.js` when `Config.SSEURL` is set. IndexedDB banned; OPFS deferred to Phase 2b. admin.js gains `initSyncWorker()`, `enqueueCommand()`, `retryQueuedCommand()`, and an `htmx:sendError` handler that queues instead of rejecting. CSS adds `[data-sync-queued]` (dimmer, slower pulse) and `.sync-bar[data-sync-status="offline"]` (amber).
- **Production SSEEventStore** (`JournalSSEStore`): Backed by `event.SeekableJournal` for efficient cursor-based replay. Falls back to `ReadAll` + in-memory filter when the journal doesn't support `ReadFrom`. `WithMaxReplay(n)` limits first-connection replay volume (default 1000). Consumer-provided `EventToSSEMapper` function converts domain events to SSE events.
- **ACK protocol** (command confirmation): `CommandAck` struct with `{commandId, status, error}` JSON. `BroadcastOnAck()` / `BroadcastOnAckFunc()` on `Broadcaster` (SSE) and `BroadcastOnAckWS()` / `BroadcastOnAckWSFunc()` on `WSBroadcaster` (WS parity). Opt-in via `X-Command-Id` header.
- **Integration tests**: 6 end-to-end tests prove `JournalSSEStore` + `Broadcaster` + ACK protocol work together in real HTTP handlers (replay, confirmed/rejected ACK, reconnect + live ACK, opt-in guard, concurrent race).
- **ADRs**: 0023 (command-sync — sync commands not events), 0024 (honest UI — never lie about pending state), 0026 (idempotency store), 0027 (decide stays on server), 0029 (SharedWorker Phase 2a).
- **Idempotency store** (`IdempotencyStore` interface + `MemoryIdempotencyStore`): Prevents duplicate command execution on client retry. `CheckAndRecord` interface method is truly atomic (single lock for check+record). `ErrDuplicateCommand` → HTTP 409 Conflict. TTL-based expiration with background sweep goroutine + lazy expiry in `Seen()`. See ADR-0026.
- **admin-demo idempotency wiring**: The admin-demo showcase now rejects duplicate mutations (same `X-Command-Id`) with HTTP 409 before they reach the panel handler. Proves the feature end-to-end.
- **ADR-0027**: Definitive decision that `decide()` stays on the server (Queue-Only client). The library provides the queue/sync/ACK protocol; pre-validation is a consumer concern. Unblocks all Phase 2 work.
- **Server-Timing API** (`server_timing.go`): W3C Server Timing response header support for debug-mode performance profiling. Emits `Server-Timing: total;desc="Total request";dur=12, db;dur=8` headers visible in browser DevTools and curl. Three entry points: (1) `ServerTimingMiddleware()` — standalone always-on middleware; (2) `ServerTimingMiddlewareWhen(pred)` — predicate-gated (e.g. `?debug=1`, admin role); (3) `Config.ServerTiming` — 1-line integration into `App.Command()`/`App.Query()` handlers. Thread-safe `*ServerTiming` collector uses nil-receiver pattern (disabled=nil=natural no-op). Helpers `MeasureServerTiming(ctx, name)` and `RecordServerTiming(ctx, name, desc, dur)` are nil-safe. Interface preservation: wrapper delegates `Flusher`/`Hijacker`/`Pusher`/`Unwrap` so SSE/WS/HTTP2 work transparently. Benchmark: disabled=3.6ns/0-allocs, enabled Measure=138ns/1-alloc.
- **Checkpoint-based projection replay** (`usermgmt/es_projection_setup.go`): `StartProjections` gains optional `event.CheckpointStore` parameter. When non-nil AND journal implements `SeekableJournal`, replay uses `ReadFrom(checkpoint.EventID, 0)` instead of `ReadAll()` — avoids full journal replay on every restart. Checkpoint saved after each replayed event. Graceful fallback: nil store or non-seekable journal → full replay (backward compatible). `EventSourcedConfig`, `ServiceConfig`, `SQLiteSetupConfig`, `PostgresSetupConfig` all gain `CheckpointStore` field.
- **Observability wiring guide** (`docs/observability-wiring.md`): Complete OTel tracing + Prometheus `/metrics` + Server-Timing wiring recipes using `BeforeDispatchHook`/`AfterDispatchHook`. References go-cqrs-lite upstream `otel/v3` + `middleware/v3` + `prometheus/v3` modules.
- **v3→v3.3 incremental migration guide** (`docs/MIGRATION-v3-incremental.md`): Documents checkpoint replay, BasicCommand embedding, Server-Timing, SQL read models, stack presets — all opt-in, backward compatible.
- **scenario/v3 BDD for Tenant, Bot, Membership** (3 new test files, 20 tests): Completes scenario/v3 BDD adoption for all 4 usermgmt aggregates. Tenant: 8 tests (create/suspend/reactivate/delete happy + error paths). Bot: 6 tests (register/delete happy + error paths). Membership: 6 tests (add/update-roles/remove happy + error paths).
- **scenario/v3 BDD on ChangeEmail** (`es_scenario_test.go`): Second decider scenario test demonstrating the BDD DSL adoption beyond RegisterUser.
- **Command ID regression tests** (`es_command_id_test.go`): `TestAllCommandsProduceDifferentIDs` — constructs every command twice (40 total), asserts all IDs are mutually unique. `TestMustCommand_PanicsOnZeroAggregateID` and `TestMustCommand_PanicsOnEmptyCommandType` — verify fail-fast behavior on programming bugs. Prevents recurrence of the zero-cmdID bug.
- **CheckpointStore usage example** (`es_checkpoint_test.go`): Demonstrates the opt-in checkpoint round-trip (fresh store → save → reload → resume from checkpoint).
- **Server-Timing fuzz tests** (`server_timing_fuzz_test.go`): Adversarial metric names/descs/durations, middleware fuzz, nil-receiver no-op verification. Found CRLF injection bug (see Fixed).
- **CI codegen drift guard** (`nix run .#check-codegen`): Verifies adminui `_templ.go` files match `.templ` sources by regenerating + diffing. Prevents silent codegen drift.
- **Templ CLI in devShell** (`flake.nix`): Pins `pkgs.templ` v0.3.1020 (matches go.mod) to eliminate codegen oscillation between CLI versions.
- **`nix run .#gen` app**: One-command templ codegen + gofmt normalization for adminui.
- **ADR-0032**: BasicCommand embedding decision — documents the structural fix for the zero-cmdID bug class and the rationale for `mustCommand` panic-on-construction-failure.
- **ADR-0031 status update**: PROPOSED → Accepted. Checkpoint-based replay shipped in StartProjections; CatchUpSubscriber migration deferred.
- **Authz doc comments**: `RemoveAllRolesForUser` and `RemoveAllRolesInDomain` now document why the `subject` parameter is intentionally `string` (Casbin subjects are polymorphic: user IDs, bot IDs, prefixed actors).

### Changed

- **ID types branded** (**BREAKING**): `ActorID`, `ImpersonatorID`, `SSEEventID` changed from `type X string` to `brandid.ID[brand, string]` — phantom-typed with `.Get()`, `.IsZero()`, `.Equal()`. `ImpersonatorID = ActorID` (type alias — an impersonator IS an actor). Use `NewActorID("...")` / `NewSSEEventID("...")` constructors instead of casts. `.String()` now returns brand-prefixed form for debug; use `.Get()` for raw value. See ADR-0028.
- **Pagination unified**: Both root `DecodePagination` and usermgmt `credential_http.go` now delegate to `query.NewPagination` from go-cqrs-lite. **BREAKING**: Requesting a page beyond the last page now returns an empty page (standard REST) instead of silently clamping to the last page. The response includes `total_pages` so clients can detect the valid range.
- **go-cqrs-lite upgraded to v3.4.0** across all 8 modules: command, event, idempotency, query, listing, projection, snapshot, stack, storage → v3.4.0; decider/id/otel/watermill/codec/dispatcher at v3.3.0/v3.3.1 (latest tags). v3.4.0 adds managed projection host, durable scheduling, scenario-testing DSL.
- **Command constructors embed `command.BasicCommand`**: All 20 usermgmt command structs now embed `*command.BasicCommand`, which promotes `Type()`, `AggregateID()`, `ID()` methods automatically. This structurally eliminates the zero-cmdID bug class (7 constructors previously returned zero command IDs, silently breaking idempotency dedup and Watermill message UUIDs). The `mustCommand` helper panics on construction failure — the only error cases (empty command type, zero aggregate ID) are programming bugs. See ADR-0032.
- **Form decoder upgraded** (`decoder.go`): Replaced allocation-heavy JSON round-trip (`url.Values → map[string]any → JSON → struct`) with `go-playground/form/v4` (zero transitive deps, `SetTagName("json")` for backward compat). Form keys normalized to lowercase for case-insensitive field matching.
- **Stdlib modernization**: `slices.Contains`, `min()`, `slices.IndexFunc` replace 5 manual loops across root/usermgmt/adminui/examples.
- **go.mod alignment**: All 8 modules aligned to Go 1.26.4.
- **SSE delegation lint cleared**: `sse_event.go` vars converted to proper wrapper functions (`gochecknoglobals`), wrapcheck annotated for pure delegation. All modules now report 0 lint issues.
- **CONTRIBUTING.md rewritten (root + usermgmt)**: Removed deleted catalog module references, fixed module count (5→8), updated test framework documentation (standard testing + scenario/v3 BDD, not Ginkgo), removed password auth references, synced file tree, added error-family enforcement, added templ codegen instructions, added Nix-first workflow.
- **usermgmt coverage gate raised**: 75% → 78% (actual: 79.3%). Locks in coverage gains.
- **ADR-0015 status table updated**: All 6 remaining "Planned" items (Session struct, Impersonation, Tenant, Bot, upcasters, Roles removal) marked Done with version references.
- **flake.nix package version bumped**: 3.1.0 → 3.3.0.
- **FEATURES.md synced**: Removed catalog column (module merged upstream), updated ClientIP to FULLY_FUNCTIONAL, synced coverage/test counts.
- **ROADMAP.md synced**: Marked scenario/v3 BDD, OTel seam guide, Prometheus seam guide as Done.
- **ROADMAP typo fixed**: "v3.30" → "v3.3.0".
- **TODO_LIST triaged**: All `[~]` (partially done) items resolved — marked `[x]` (BrandNamer wired, TenantState.IsValid added, Phase 2a shipped) or `[-]` (blocked: ActorID split brain, Email type, WebAuthn \*http.Request, snapshot integration).
- **Status reports archived**: 7 reports older than 2 weeks (pre-June-15) moved to `docs/status/archive/`.

### Fixed

- **Command ID minting bug** (CRITICAL): 7 of 20 usermgmt command constructors (`RegisterUserCmd`, `LinkExternalAccountCmd`, `UnlinkExternalAccountCmd`, `AddMemberCmd`, `UpdateMemberRolesCmd`, `RemoveMemberCmd`, `RegisterBotCmd`) returned a zero-value `cmdID`, silently breaking idempotency dedup and Watermill message UUIDs (which derive from `cmd.ID()`). All constructors now call `id.NewCommandID()`. Regression test added.
- **CRLF injection in Server-Timing `escapeQuotedString`**: CR and LF characters in metric descriptions are now replaced with spaces. Previously only `"` and `\` were escaped, leaving raw newlines that could enable HTTP header splitting via crafted descriptions. Found by `FuzzServerTimingHeaderValue`.
- **Idempotency `CheckAndRecord` atomicity**: The original free function called `Seen()` then `Record()` as two separate interface calls — a TOCTOU race. Fixed by moving `CheckAndRecord` into the `IdempotencyStore` interface; `MemoryIdempotencyStore` now does check+record under a single write lock. Proven by a 200-goroutine concurrency test (exactly 1 winner).
- **Idempotency memory leak**: `Seen()` now lazily deletes expired entries, preventing unbounded map growth when the sweep goroutine is disabled (`sweepInterval=0`).

## [3.2.0] - 2026-06-28

### Added

- **VERSIONING.md**: Documents the semver policy for the library (when major/minor/patch bumps occur, pre-release tagging, and consumer upgrade expectations).
- **Consumer migration guide (v2→v3)** (`docs/migrations/v2-to-v3.md`): Covers import path changes (`/v2` → `/v3`), bus replacement (manual → watermill EventBus), and projection replay changes (automatic → manual checkpoint-based).
- **Godoc examples**: Added runnable godoc examples for `App`, `Handler`, and `Service` entry points so consumers can discover the API from `pkg.go.dev`.
- **Service-level impersonation tests**: End-to-end tests through full dispatch (command → event → read model) verifying impersonation flows.
- **Service-level membership tests**: End-to-end tests through full dispatch verifying membership add/update/remove flows.
- **Projection replay integration test**: Verifies journal-vs-live dedup correctness when projections restart from a checkpoint.
- **Property-based tests for event folds** (`foldTenant`, `foldBot`, `foldMembership`): Sequence-property tests verifying associativity, tombstone idempotency, and aggregate-specific invariants across random event streams.
- **Fuzz tests for projection dedup + identity model deciders**: Adversarial input testing for the dedup ring and all four aggregate deciders.

### Changed

- **revive:exported linter enabled**: All exported symbols now have godoc comments. Fixes all revive:exported violations across the codebase.
- **catalog/ module merged into go-cqrs-lite** (**BREAKING**): The
  `github.com/larsartmann/cqrs-htmx/catalog/v3` module is deleted. Its
  single-service Builder facade and standalone HTTP handlers (D2, Health,
  EventCatalog) are now part of `go-cqrs-lite/catalog/v3` v3.2.0 (`simple` and
  `docserver` sub-packages). The redundant `OpenAPIHandler`/`AsyncAPIHandler`
  are removed — use the richer upstream `docserver.DocsServer` (HTML UIs,
  static assets, YAML+JSON). Migrate by swapping imports:
  `cataloghtmx.New` → `simple.New`, `cataloghtmx.Command[T]` → `simple.Command[T]`,
  `cataloghtmx.D2Handler` → `docserver.D2Handler`,
  `cataloghtmx.HealthCheckHandler` → `docserver.HealthCheckHandler`,
  `cataloghtmx.GenerateEventCatalog` → `docserver.GenerateEventCatalog`,
  `cataloghtmx.OpenAPIHandler(cat)` → `docserver.NewDocsServer(...).OpenAPISpec()`.
  See ADR 0020 for rationale.

### Fixed

- **ADR number collision** (0015): The catalog-merge ADR shared number 0015 with the identity-model-redesign ADR. Renumbered catalog-merge ADR → 0020. Cross-references in ADR 0008 and CHANGELOG updated.
- **Broken CHANGELOG link**: The v2.2.0 changelog entry linked to `catalog/README.md` (deleted by the merge). Replaced with a valid link to ADR 0008.
- **Stale status reports archived**: 51 status reports >2 weeks old moved to `docs/status/archive/`. Reduces noise from 96 → 45 visible reports.
- **Migration checklist added to AGENTS.md**: New "Module Deletion / Migration" gotchas (#22-26) encode the reference-sweep, ADR-collision-check, breaking-change-CHANGELOG, and write-once-report lessons. Prevents recurrence of the "code green but docs lie" failure mode.

## [3.1.0] - 2026-06-26

### Added

- **SQL-backed persistent read models**: `SQLUserReadModel`, `SQLMembershipReadModel`, `SQLTenantReadModel`, `SQLBotReadModel` — survive process restarts without rebuilding from the event journal. Each stores data as queryable scalar SQL columns + a JSON blob for full reconstruction. Dual SQLite and Postgres constructors (`NewSQLite*ReadModel` / `NewSQL*ReadModel`). Uses go-cqrs-lite `storage.SQLViewStore[V,K]` + `AutoMapperWithTombstone[V]`.
- **One-call stack presets**: `NewSQLiteEventSourcedSetup(cfg)` and `NewPostgresEventSourcedSetup(cfg)` — create event store, bus, repositories, SQL read models, and projections in a single call. Postgres preset supports multi-DB split (`EventDSN` / `QueryDSN`). Each returns a setup struct with `Close()`, `GracefulClose(ctx)`, and `Authz()`.
- **`OptimizeSQLiteDB(ctx, db)`**: Applies WAL mode, synchronous=NORMAL, busy_timeout=5000ms, 64 MB cache, temp_store=MEMORY, 256 MB mmap. 3-10x write throughput improvement. Call before creating stores. No-op for Postgres/MySQL.
- **`Service.Close()` / `Service.GracefulClose(ctx)`**: Full lifecycle shutdown — stops eviction goroutines, then closes bus and store (if they implement `io.Closer`). Idempotent.
- **`EventSourcedSetup.Close()`**: Closes the setup's bus and store. Idempotent.
- **CI gate apps**: `nix run .#coverage-gate` (fails if coverage drops below thresholds: root 90%, usermgmt 75%, catalog 90%) and `nix run .#errorfamily` (verifies zero stdlib error constructors).
- **Restart-survival test**: Proves read model persistence across database close/reopen via journal replay.

### Changed

- **go-cqrs-lite upgraded to v3.1.0** across all 7 modules. Adds SQL-backed view stores (`storage.SQLViewStore`), `AutoMapper`/`AutoMapperWithTombstone`, `stack/sqlite` and `stack/postgres` one-call presets, typed command/query metadata (`event.CustomData[K]`), multi-DB split presets, and SQLite WAL tuning. Zero breaking changes over v3.0.0.
- **go-error-family upgraded to v0.5.1** across all modules. `ErrDispatchFailed` is now natively classified (`event.NewTransient`) — the old `sync.Once` + `RegisterClassification` machinery was removed.
- **Coverage**: root 95.4%, usermgmt 79.5% (was 74.7%), catalog 95.3%. 697 usermgmt tests (was ~500).
- **`flake.nix` package version**: Updated from `2.4.0` → `3.1.0`.

### Fixed

- **Coverage CI gate was silently always passing**: `bc` was missing from `runtimeInputs` and `go test` stdout leaked into the coverage variable, corrupting the `bc` comparison. Fixed: added `pkgs.bc`, fully suppressed stdout.
- **foldUser now errors on unknown events**: Previously silently ignored. Now returns a `Rejection` error, matching upstream's strict event handling.
- **SSEStream race condition**: Fixed concurrent access to stream state during Send/Close.
- **Sweeper double-close panic**: Fixed panic when periodic eviction tried to close an already-closed store.
- **CSRF plaintext HTTP origin bypass**: Hardened — defaults to off unless explicitly trusted via `TrustedProxies`.
- **actorKindFromString validates input**: Previously accepted any string; now rejects unknown actor kinds.

## [3.0.0] - 2026-06-22

### Added

- **`migrations/0001_user_events_to_events.sql`**: Migration script for deployments that used the pre-delegation `user_events` table. Renames `user_events` → `events`, adds the upstream columns (`schema_version`, `payload_encoding`, `created_at`), and backfills them with defaults (`schema_version = 1`, `payload_encoding = 'json'`, `created_at = occurred_at`). Idempotent via `IF NOT EXISTS` on columns and a guard on the old table name.
- **Branded types for `AuditEntry`**: `AggregateID` is now `id.AggregateID`, `UserID` is now `usermgmt.UserID`, and `EventType` is now `event.Type`. Prevents accidental cross-assignment with other string-typed IDs. **Breaking:** `AuditLog.EntriesFor` now takes `id.AggregateID` instead of `string` — convert via `id.ParseAggregateID(user.ID.Get())`. JSON wire format: non-zero IDs still serialize as strings; zero-value `UserID` now serializes as `null` instead of `""`.
- **`SSEEventID` branded type**: A `type SSEEventID string` that wraps the `Last-Event-ID` / `id:` SSE field. Constructed via `NewSSEEventID` (no validation) or `ParseSSEEventID` / `MustParseSSEEventID` (rejects newlines/carriage returns that would corrupt the wire format). Has `String()` and `IsZero()` methods. **Breaking:** `SSEEvent.ID` is now `SSEEventID`, `SSEStream.LastEventID()` returns `SSEEventID`, `LastEventIDFromRequest` returns `SSEEventID`, `ReplayEvents` takes `SSEEventID`. The low-level `SSEEventStore.EventsAfter(string)` interface is unchanged — pass `.String()` at the boundary.
- **`JSONKeyError` / `JSONKeyStatus` constants in root**: Exported JSON map-key constants (`"error"`, `"status"`) for consistent error/status response shapes within the root module. Note: usermgmt and catalog are independent Go modules and cannot import these — they retain their own local `errorKey`/`statusKey` constants with matching values. True cross-module unification would require extracting a shared `jsonkeys` sub-module, which is not worth the added module complexity for two string constants.

### Changed

- **`SQLEventStore` is now a type alias** over `go-cqrs-lite/storage/v3`'s `storage.SQLEventStore`. The hand-rolled 413-LOC store was replaced by a 78-LOC facade. The upstream store is a strict superset: richer schema (`schema_version`, `payload_encoding`, `created_at`), `event.SeekableJournal` / `BackwardsSource` conformance, OpenTelemetry tracing, and `event.WrapInfrastructure` error wrapping.
- **go-error-family upgraded to v0.5.0** across all modules. `MapError` now delegates to upstream `Family.HTTPStatus()` — the hand-rolled `familyStatus()` switch (E04) is deleted. The upstream method returns identical status codes (Rejection→400, Conflict→409, Transient→503, Corruption→500, Infrastructure→503), confirming the ADR-0017 reconciliation was correct. The upgrade is now applied uniformly (root, usermgmt, catalog, integration_test) — the initial upgrade had left usermgmt and catalog stranded on v0.4.0.

### Fixed

- **HTTP status mapping reconciled with upstream `go-error-family`** (ADR-0017). `Corruption` errors now return **500** (was 422) — data integrity breaks are server-side, not client input errors. `Infrastructure` errors now return **503** (was 500) — the correct "service unavailable, retry later" semantic. Panics still return 500 via explicit override (a panic is a server bug, not service-unavailable). **Breaking:** any client checking for 422 on corruption or 500 on infrastructure must update to 500 and 503 respectively.
- **`usermgmt.UserID` unified with `id.UserID`** (ADR-0018, supersedes ADR-0002). `usermgmt.UserID` is now a type alias of `id.UserID` from go-cqrs-lite, backed by `ulid.ULID` — eliminating the string-backed/ULID-backed type split that forced manual `.String()`→`NewUserID()` conversion at every boundary. `NewUserID(s)` now accepts any string: valid ULIDs pass through directly, non-ULID strings are deterministically hashed to valid ULIDs (backward compat for tests). `MustParseUserID` added for strict ULID validation in production code. `.Get()` now returns `ulid.ULID` instead of `string` — use `.Get().String()` at string boundaries (SQL, Casbin, logging). **Breaking:** consumers storing or transmitting UserID values as raw strings must use `.Get().String()`.
- **Broadcaster send-on-closed-channel panic** (`fanout.go`): `Broadcast()` previously snapshted subscriber channels under RLock, released the lock, then iterated — a concurrent `Unsubscribe()` could `close()` a channel mid-iteration, panicking the entire process. Fixed by holding the RLock during the non-blocking fan-out (race-safe: `Unsubscribe()` cannot acquire the write lock to close a channel during iteration). Regression test added.

### Breaking Changes

- **`SQLEventStore.Close()` no longer closes the `*sql.DB`.** The delegated upstream store uses a borrowed DB handle — callers now own its lifecycle. **Migration:** any code that called `store.Close()` to clean up the DB must now call `db.Close()` separately:

  ```go
  // Before (v2.5.0):
  store, _ := usermgmt.NewSQLEventStore(ctx, db, "postgres")
  defer store.Close() // closed db too

  // After (v3.0.0):
  store, _ := usermgmt.NewSQLEventStore(ctx, db, "postgres")
  defer store.Close() // marks store closed; db stays open
  defer db.Close()    // caller owns the DB lifecycle
  ```

  Rationale: shared-DB usage (event store + session store on one `*sql.DB`) was already broken under the old behavior — `store.Close()` would close the DB out from under the session store. The upstream library's "borrowed handle, caller owns lifecycle" philosophy is `database/sql` best practice and is correct for shared-DB scenarios. No backward-compat wrapper is provided because silently closing the DB was itself a bug for shared-DB consumers.

- **`SQLEventStore.Load()` on empty aggregate** now returns `event.ErrAggregateNotFound` instead of an empty slice. The decider's `Repository` handles this transparently (returns `Initial` state), so consumers going through `Service` are unaffected. Direct callers of `store.Load()` should check `errors.Is(err, event.ErrAggregateNotFound)` if they need to distinguish "no events yet" from a real error.
- **MySQL dropped for event store.** `NewSQLEventStore(ctx, db, "mysql")` now returns an error — `go-cqrs-lite/storage` has no MySQL dialect. `SQLSessionStore` retains MySQL support (it manages its own simpler schema). Migration path: switch to Postgres or SQLite for the event store.

## [2.5.0] - 2026-06-18

### Added

- **StructuredError**: RFC 7807-shaped transport-agnostic error payload (`type`, `title`, `status`, `detail`, `instance`). `NewStructuredError(err, r)` maps via `MapError` and extracts request ID. `JSON()` method for SSE/WS data serialization. Used uniformly across SSE and WS error channels.
- **BroadcastOnError / BroadcastOnErrorFunc**: Symmetric to `BroadcastOnSuccess`. `AfterDispatchHook` factories that fire when `err != nil`, broadcasting a `StructuredError` JSON event. Closes the SSE error-reporting gap — clients now learn when commands fail.
- **SSEStream.Heartbeat(ctx, interval)**: Sends SSE comment-frame pings (`: keepalive\n\n`) on a ticker. Prevents reverse proxies (Nginx, Cloudflare, AWS ALB) from killing idle SSE connections after 30–60s of silence.
- **SSEStream.OnDisconnect(fn)**: Registers cleanup callbacks fired on `Close()`. Enables metrics, logging, session deregistration.
- **WriteWSMessage / WriteWSMessageInto[T]**: Outbound WebSocket message encoders. Counterparts to `ParseWSMessage` / `ParseWSMessageInto[T]`. Round-trip verified.
- **WSBroadcaster**: Thread-safe fan-out for WebSocket messages. Mirrors SSE `Broadcaster` API — `Subscribe` / `Unsubscribe` / `Broadcast` / `SubscriberCount`. O(1) unsubscribe via channel pointer identity. Buffered channels (64).
- **BroadcastOnSuccessWS / BroadcastOnErrorWS**: `AfterDispatchHook` factories for `WSBroadcaster`. WebSocket equivalents of SSE `BroadcastOnSuccess` / `BroadcastOnError`.
- **DispatchWSCommand / DispatchWSQuery**: WebSocket→CQRS dispatch bridge. `App.DispatchWSCommand(r, type, decoder, data)` decodes raw WS message bytes, runs lifecycle hooks, dispatches the command, and returns the error. `DispatchWSQuery` returns `(result, error)`. Does NOT handle auth/CSRF/response-writing — WS is authenticated at upgrade time. `DecodeWSJSON[T]` / `DecodeWSJSONQuery[T]` decoder factories mirror HTTP `DecodeJSON[T]` but operate on `[]byte`.
- **ADR 0010**: Documents transport parity decisions — StructuredError (RFC 7807), BroadcastOnError symmetry, WSBroadcaster mirroring SSE Broadcaster, and WS dispatch bridge design. Closes the SSE↔WebSocket feature gap.
- **WSBroadcaster benchmarks**: `BenchmarkWSBroadcasterBroadcastStress` and `BenchmarkWSBroadcasterConcurrentSubscribe` mirror the SSE Broadcaster benchmarks. Validated fan-out scaling (1 sub = 1.5μs, 1000 subs = 174μs).
- **Generic fanOut[T]**: Extracted the shared subscriber hub from SSE `Broadcaster` and WS `WSBroadcaster` into a single generic `fanOut[T any]` type. Both broadcasters embed it, gaining Subscribe/Unsubscribe/Broadcast/SubscriberCount via method promotion. Eliminates ~100 lines of copy-pasted code. Zero breaking changes — identical public API. `channelPtr` helper moved to `fanout.go` (was misplaced in `sse_broadcaster.go` despite being transport-agnostic).
- **WS Func variants**: Added `BroadcastOnSuccessWSFunc(fn)` and `BroadcastOnErrorWSFunc(fn)` for dynamic message generation, closing the parity gap with SSE's `BroadcastOnSuccessFunc` / `BroadcastOnErrorFunc`.
- **Unified timeout**: Removed `dispatchTimeout` split brain. WS dispatch now calls `timeoutCtx(ctx, nil)` — the same method the HTTP path uses, made nil-safe for `*handlerConfig`.
- **catalog sub-package** (5th Go module): Automatic API documentation generation from Go CQRS types. Produces OpenAPI 3.0, AsyncAPI 3.0, D2 diagrams, and EventCatalog MDX file trees from a single registration. Zero dependency on root or usermgmt modules — consumers opt in via `go get github.com/larsartmann/cqrs-htmx/catalog/v3`. Includes HTTP handlers (`OpenAPIHandler`, `AsyncAPIHandler`, `D2Handler`, `GenerateEventCatalog`), JSON/YAML output via `WithFormat()`, schema auto-derivation from struct tags, and catalog validation (`Build()` panics, `BuildValid()` returns violations). See [ADR 0008](docs/adr/0008-catalog-sub-package.md). _(Note: this module was later merged upstream into `go-cqrs-lite/catalog/v3` — see ADR 0020.)_
- **ADR 0009**: Documents the rationale for which go-cqrs-lite modules are used (8 direct) vs not used (13 excluded), with specific reasons for each exclusion. Cross-references the middleware integration guide.
- **Middleware integration guide** (`docs/integrations/go-cqrs-lite-middleware.md`): Documents how go-cqrs-lite dispatch middleware (retry, circuit breaker, metrics, tracing) composes with cqrs-htmx HTTP middleware. Different layers, no conflict.
- **`ForbiddenErrorHandler`**: Exported named CSRF error handler for `CSRFConfig.ErrorHandler` — writes HTTP 403 with no body. Convenient default for consumers wiring CSRF protection without rendering nosurf's default error page.
- **catalog-demo example**: `examples/catalog-demo/` — standalone server generating live OpenAPI, AsyncAPI, and D2 docs from Go struct tags via the catalog module. Includes graceful shutdown and `ReadHeaderTimeout` hardening.
- **`test-race` / `test-fuzz` nix apps**: `nix run .#test-race` runs the race detector across all four Go modules; `nix run .#test-fuzz` discovers and runs every `Fuzz*` target (default 30s each, overridable via `FUZZTIME`).

### Fixed

- **`applyConfigDefaults` silent field loss**: The `HandlerConfig` defaults logic created a fresh config and manually copied each field — any field not copied was silently zeroed. `WebAuthnRateLimit` was missing, silently disabling rate limiting on all 4 WebAuthn endpoints. Refactored to start from the caller's config and override only fields needing non-zero defaults, eliminating the entire class of "forgot to copy a field" bugs.
- **Removed lying `EventCatalogHandler`**: The handler set `Content-Type: application/zip` but served a JSON file listing — a critical honesty violation. Removed entirely; kept `GenerateEventCatalog()` (build-time/startup-time file generation, which is the correct EventCatalog model).
- **Fixed `flake.nix` split brain**: `nix run .#test` (and `.#lint`/`.#build`/`.#coverage`) silently skipped the catalog module. Catalog is now included in all four multi-module nix apps.
- **Removed self-referential `replace` directive** in `catalog/go.mod` (`replace github.com/larsartmann/cqrs-htmx/catalog/v3 => ./`) — pointed the module at itself.
- **Bumped `flake.nix` `packages.default` version** from stale `2.3.0` to `2.4.0` to match the current release.
- **Fixed `errors.Join` anti-pattern**: Replaced `errors.Join(errors.New("..."), err)` with idiomatic `fmt.Errorf("...: %w", err)` for proper error wrapping.
- **Fixed 7 lint issues** in the catalog module: errchkjson, exhaustive, exhaustruct×3, forcetypeassert, gci. All modules report 0 lint issues.
- **Removed duplicate import** in `catalog/example_test.go` (same package imported twice with different aliases).

## [2.4.0] - 2026-06-17

### Changed

- **go-cqrs-lite v2.4.0**: All modules upgraded to the latest per-module tags (command, event, id, query, codec, decider, dispatcher, memory, projection, snapshot, otel).
- **go-error-family v0.4.0**: Upgraded with idiomatic `event.Wrapf` / `event.WrapTransient` replacing all `NewTransient(fmt.Sprintf(...)).WithCause(err)` patterns.
- **go-branded-id v0.3.1**: Latest branded ID library.
- **ginkgo v2.31.0 + gomega v1.42.0**: Latest BDD test framework.
- **Lint: 0 issues**: Fixed all 16 pre-existing lint warnings (errorlint, exhaustruct, contextcheck false positives).
- **Email validation consolidated**: `ImportUser.Validate()` now delegates to `ParseEmail()` instead of reimplementing identical logic.

### Security

- **Import/export authorization**: Endpoints now require admin role by default. Previously any authenticated user could export all user data or create accounts.
- **TOTP disable protection**: `DisableTOTP` now requires a valid code, preventing MFA stripping if a session is hijacked.
- **Per-endpoint rate limiting**: `HandlerConfig.ImportRateLimit`, `TOTPRateLimit`, `VerificationRateLimit` for per-IP rate limiting.

### Changed

- **TOTP library**: Replaced hand-rolled HMAC-SHA1 with `pquerna/otp/totp`.
- **DisableTOTP signature**: Now takes a code parameter.
- **ExportFormat → UserDataFormat**: Honest naming for dual import/export use.
- **Email value type**: `ParseEmail`/`MustParseEmail` for centralized validation. Used in `ExportUser`.
- **withTimeout helper**: Eliminates duplicated context-timeout boilerplate.

### Added

- **TOTP multi-factor authentication**: RFC 6238 TOTP via `pquerna/otp/totp` library. Events: `TOTPEnabled`/`TOTPDisabled`. Two-phase setup: `EnableTOTP` → pending secret → `VerifyTOTPSetup` confirms. Generates `otpauth://` URIs for QR codes. `Service.VerifyTOTP` validates codes as second factor. `DisableTOTP` requires a valid code to prevent MFA stripping.
- **Email verification flow**: Token-based email confirmation with `EmailVerified` event. `SendVerificationEmail` generates token (configurable TTL, default 24h) with optional SMTP callback. `VerifyEmail` consumes single-use token. Email change resets verification status.
- **User import/export**: `Service.ImportUsersFromJSON`/`ImportUsersFromCSV` for batch user creation (skips existing emails). Input validation via `ImportUser.Validate()`. `Service.ExportUsersToJSON`/`ExportUsersToCSV` for data export. Admin-only by default via `ImportExportAuthorizer`. Per-IP rate limiting via `HandlerConfig.ImportRateLimit`.
- **Virtual authenticator integration test**: Full WebAuthn registration → login ceremony using W3C spec test vectors. Tests: successful full flow, registration replay rejection, login with wrong challenge.
- **SQL event store**: `SQLEventStore` implements `event.Store` + `event.Journal` for Postgres/SQLite/MySQL. Auto-migrates schema, supports optimistic concurrency, parameterized queries per dialect.
- **Audit log projection**: `AuditLog` records all user events as `AuditEntry` structs. Queryable by aggregate ID, recent N, or total count. Optional via `ServiceConfig.AuditLog`.
- **Rate-limited registration**: Per-IP fixed-window rate limiting on the registration endpoint. Configurable via `HandlerConfig.RegistrationRateLimitConfig`.
- **Session rotation on privilege change**: `UpdateRoles` deletes all user sessions after role change, forcing re-authentication.
- **Credential listing pagination**: `GET /auth/credentials?page=N&page_size=N` with configurable page size (default 20, max 100).
- **Structured logging in WebAuthn ceremonies**: All 4 ceremony methods log at debug/info/warn with user_id, email, error context.
- **CI pipeline (GitHub Actions)**: Multi-module CI with separate lint jobs for root and usermgmt, race detector, `GONOSUMCHECK` env vars, concurrency group.
- **doc.go**: Comprehensive root package documentation with Quick Start, middleware stack, response builder, error mapping, SSE, and submodule reference.
- **Property-based testing**: 8 rapid-based property tests for `foldUser` invariants.
- **Benchmarks**: 5 benchmarks for `foldUser`, `ReadModel.Handle`, `BeginRegistration`, `BeginLogin`.
- **Fuzz tests**: 3 fuzz tests for WebAuthn ceremony inputs and credential ID decoding.
- **Coverage tests**: 15 tests covering error paths, schema version, RegisterCommands, DefaultEventSourcedSetup, writeJSON, HTTP handlers.

- **AccountLockout wired into WebAuthn login**: `BeginLogin` checks lockout status; `FinishLogin` records failures and resets on success.
- **Credential management HTTP endpoints**: `GET /auth/credentials` (list), `DELETE /auth/credentials/{id}` (remove by base64url ID).
- **WebAuthn session proactive eviction**: Background goroutine (`webauthnEvictionInterval`) periodically removes expired challenge sessions. `Service.Stop()` terminates the goroutine.
- **CasbinProjection subscribes to CredentialAdded/Removed**: Ensures projection ordering without affecting policies.
- **`Service.Stop()` method**: Gracefully shuts down background resources (WebAuthn eviction goroutine). Safe to call multiple times.
- **`Service.Authz()` accessor**: Returns the underlying `*Authz` for direct policy queries.
- **Service source propagation**: `Config.ServiceName` (string) and `App.EventOptions(ctx)` inject `event.WithSource` into event options. `EventOptionsFromContextWithSource(ctx, name)` is the free-function equivalent for callers that don't hold an `*App`.
- **Typed-query examples in root**: `ExampleApp_Query_typedRegister` and `ExampleApp_Query_typedDispatch` demonstrate the `query.RegisterTyped[T]` / `query.DispatchTyped[T]` pattern crossing module boundaries.
- **`ExampleBroadcaster_BroadcastOnSuccessFunc`**: Documents the dynamic-event `AfterDispatchHook` factory.
- **`ExampleConfig_BeforeDispatch` / `ExampleChain` (tracing)**: Show timing/span integration via the lifecycle hooks; no OpenTelemetry dependency required.
- **Per-module nix apps**: `nix run .#test-root`, `nix run .#test-usermgmt`, `nix run .#test-integration`, `nix run .#build-datastar-demo` set `GOWORK=off` automatically — no more per-module `cd` dance.
- **UpdateTodo command in datastar-demo**: `UpdateTodoCmd` + `TodoUpdatedPayload` + `handleUpdateTodo` + `POST /api/todos/update` route + inline edit input in the UI. The read-model `Projector.Apply` now handles `TodoUpdated` and the broadcaster renders it as `todo_updated`.
- **`GetStatsQry` typed query in datastar-demo**: `Stats` struct + `GetStatsQry` + `renderStatsFromQuery` helper. Wired into `handleListTodos` and `handleEventStream` so the stats read goes through the typed query dispatcher (caching/authorization-friendly).
- **SSE replay endpoint in datastar-demo**: `GET /api/events/replay` reads `Last-Event-ID` and replays missed events from the in-memory event store.
- **`FuzzEventOptionsFromContext`**: Fuzzes `EventOptionsFromContext` with arbitrary combinations of (deadline, cancel, user ID, correlation ID, request ID) — verifies the bounded-options invariant (≤ 4 options).
- **Real-server integration tests**:
  - `TestRateLimiter_RealServer_AllowsThenBlocks` + `TestRateLimiter_RealServer_ConcurrentRequests`: end-to-end rate limiting over `httptest.NewServer` with a real `http.Client`.
  - `TestSSE_RealServer_ReconnectionWithLastEventID` + `TestSSE_RealServer_ReconnectionNoLastID`: end-to-end SSE reconnection with real `Last-Event-ID` header.
  - `TestTypedQueryDispatch_CrossModule` + `TestCrossModule_PaginationFlow`: cross-module CQRS + pagination.
- **Benchmarks**:
  - `BenchmarkDecodePagination` (8 URL shapes: no params, with page, with size, with both, with extras, invalid, zero, huge).
  - `BenchmarkCommandRegisterTypedVsRegister` (typed ≈ manual: 57.4 vs 56.9 ns/op).
  - `BenchmarkQueryDispatchTypedVsDispatch` (typed ≈ manual: 85.7 vs 81.0 ns/op).
- **usermgmt service transient-error context**: `withUserIDContext` helper annotates every `event.NewTransient` in the service with `user_id` context (no PII — only the user ID, not email/display name). New `transientErr(userID, msg, cause)` helper keeps service methods concise.
- **ADR 0005**: Documents the go-cqrs-lite v2.3.0 adoption decisions (IsZero validation, FromContext deadline propagation, RegisterTyped/DispatchTyped, replace-directive removal).

### Changed

- **Eliminated production panics**: `marshalPayload` and `aggIDFromUser` now return errors instead of panicking. All callers updated to handle errors gracefully.
- **RegisterCommands returns error**: Silently ignored command registration failures now propagate to `NewService`.
- **bus.Subscribe failures logged**: Event bridge subscription errors now logged at warn level instead of silently ignored.
- **Projection runner errors logged**: Background projection goroutine errors now logged at error level.
- **WebAuthn HTTP refactor**: Finish endpoints use query params (`?user_id=X`) instead of fragile double body read pattern.
- **Coverage**: 86.2% usermgmt, 96%+ root, zero 0% functions.

### Fixed

- **Goroutine leak**: All WebAuthn test services now call `t.Cleanup(svc.Stop)` to terminate eviction goroutines.
- **Stale password references**: Removed all `"password":"secret12"` from test helpers (passwordless migration).
- **gosec G101**: Credential event/command type constants now have proper nolint annotations.
- **wrapcheck**: `marshalPayload` and `Dispatch` errors are now properly wrapped.

## [2.3.0] - 2026-06-13

### Changed

- **go-cqrs-lite upgraded to v2.3.0** across all modules (root, usermgmt, integration_test, datastar-demo). Per-module tags now published — no go.work replace directives needed.
- **Empty command/query type validation**: `App.Command("")` and `App.Query("")` panic at handler registration time using `command.Type.IsZero()` / `query.Type.IsZero()` — fail-fast instead of silent empty dispatch.
- **Deadline propagation**: `EventOptionsFromContext` now propagates context deadline to event options via `event.FromContext(ctx)` when present. Downstream events inherit request timeouts.
- **Type-safe handlers in datastar-demo**: Refactored all 3 command handlers from manual type assertions to `command.RegisterTyped[T]` — eliminates `cmd.(*CreateTodoCmd)` boilerplate.
- **Local MustParse wrappers**: Reimplemented `MustParseUserID`, `MustParseCorrelationID`, `MustParseRequestID` locally (go-cqrs-lite removed `id.MustParse[T]`). Preserves API compatibility for consumers.
- **query.MustNew removed upstream**: Updated `datastar-demo` to use `query.New()` + error check.
- **Code deduplication**: Eliminated all clone groups at threshold 50 (industry standard). At threshold 30, remaining groups are all 2–7 line spans in test code. −104 lines net across 14 source files.
- **goconst warnings eliminated**: Extracted test constants in `sse_test.go` (`eventTodoCreated`, `eventUpdate`, `eventItem`, `dataFirst`) and `coverage_test.go` (`aliceName` usage). Added `goconst` exclusion for `example_test.go` (self-contained examples should not reference test constants).
- **nestif warning fixed**: Extracted `parseWSHeaders` helper from `ParseWSMessageInto` in `ws.go`, reducing nesting complexity from 7 to 1.
- **Test deduplication**: Deleted 6 duplicate ClientIP tests from `coverage_test.go` (already covered by `httputil_test.go`). Merged 3 `sanitizeRedirectURL` DescribeTables into one. Extracted `queryNamedResultHandler` helper for query-result fixtures.
- **`ws_test.go` coverage improved**: `ParseWSMessageInto[T]` from 86.7% → 93.3% by exercising the type-assertion failure path and non-string header values.
- **`usermgmt.UpdateRoles` refactor**: Extracted `transientErr` helper to keep the function under the 60-line `funlen` limit. Replaces inline `withUserIDContext(event.NewTransient(...).WithCause(err), userID)` boilerplate.

## [2.1.0] - 2026-06-08

### Added

- **SSE (Server-Sent Events) support**: Full SSE implementation for the HTMX SSE extension (`hx-ext="sse"`). `SSEStream` manages a single connection with correct headers, flush, and context-aware lifecycle. `SSEEvent` struct with `Event`, `Data`, `ID`, `Retry` fields. `WriteSSEEvent` for writing events to any `io.Writer`.
- **SSE Broadcaster**: Thread-safe fan-out `Broadcaster` with O(1) unsubscribe via channel identity, buffered channels (64), and non-blocking broadcast (drops to slow consumers). `SubscriberCount()` for monitoring.
- **SSE reconnection**: `LastEventIDFromRequest(r)` extracts `Last-Event-ID`. `SSEEventStore` interface and `ReplayEvents(stream, store, lastID)` for full SSE spec reconnection support.
- **SSE + CQRS bridge**: `BroadcastOnSuccess(event, data)` and `BroadcastOnSuccessFunc(fn)` — `AfterDispatchHook` factories that broadcast SSE events on successful command dispatch.
- **WebSocket message parser**: `ParseWSMessage(data)` parses HTMX WebSocket JSON into `WSMessage` with separated `Headers` and `Body` fields. `StringBody(key)` for typed string access.
- **Typed WebSocket parser**: `ParseWSMessageInto[T](data)` — generic typed parser that deserializes body into struct T while separating HEADERS. Compile-time safe.
- **WebSocket OOB HTML**: `WSOOBHTML(id, html, strategy)` wraps HTML with `hx-swap-oob` attributes for out-of-band swaps. Uses `SwapStrategy` type.
- **Pagination**: `DecodePagination(r)` extracts page/page_size from query params, delegates to `query.NewPagination` for defaults (page=1, page_size=20, max=100). `RenderPaginatedJSON[T]()` renders `query.PaginatedResult[T]` as JSON 200.
- **Embedded HTMX JavaScript**: `HTMXScriptHandler()` serves embedded HTMX v2.0.9 (minified, ~49KB) with `Content-Type`, long-lived `Cache-Control` (1 year, immutable), `ETag`, and `If-None-Match` support for 304 responses. `HTMXScriptTag(path)` generates `<script>` tags. `HTMXVersion()` returns `"2.0.9"`. Opt-in, zero CDN dependency.

### Changed

- **go-cqrs-lite upgraded to v2.2.0** across all modules. Adopts `query.Pagination` and `query.PaginatedResult[T]` from upstream.
- **httputil upgraded to v0.1.0** (→ v0.1.1 pending).
- Modernized Go idioms and normalized table formatting across codebase.

## [2.0.0] - 2026-05-27

### Changed

- **CSRF: replaced `gorilla/csrf` with `justinas/nosurf`**: Simpler API, no secret management required, no HMAC/securecookie. Token generation uses `crypto/rand` with origin validation instead. `CSRFConfig.Secret` field removed. `RotateCSRFToken` removed (nosurf rotates automatically). Custom header/field translation via `translateCSRFHeaders`.
- **Error handling: replaced `cockroachdb/errors` with `go-error-family`**: Error classification now uses `github.com/larsartmann/go-error-family` via `go-cqrs-lite/core/event`. `sync.Once` registers sentinels with `event.RegisterClassification`.
- **`ClientIP` delegated to `larsartmann/httputil`**: `httputil.ClientIP(r)` replaces the local implementation.
- **go-cqrs-lite/core upgraded to v1.5.1**: `CatalogEntry` → `HandlerMeta` for API compatibility.

### Added

- **Panic recovery middleware**: `RecoveryMiddleware` recovers from panics, logs stack traces via `slog.ErrorContext`, and writes 500. `App.RecoveryMiddleware()` uses the App's configured `ErrorHandler` for consistent error formatting (JSON, redirects, request ID correlation). `http.ErrAbortHandler` is re-raised without recovery (Go net/http convention).
- **JSON result rendering**: `RenderJSON[T]()` renders query results as JSON with 200 OK. `RenderJSONStatus[T](status)` renders with a custom status code (e.g., 201 Created). Both include runtime type assertion for compile-time documentation.
- **Request ID in error responses**: `JSONErrorHandlerWithRedirect` now includes `"request_id"` field when a `RequestID` is present in context. `DefaultErrorHandlerWithRequestID` and `DefaultErrorHandlerWithRedirectAndRequestID` prefix plain-text errors with `[request_id: RID]`.
- **Config field**: `IncludeRequestIDInErrors` — when `true` and no custom `ErrorHandler` is set, the default handler automatically includes request IDs in error responses.
- `CorrelationID` type alias (`type CorrelationID = id.CorrelationID`) with `NewCorrelationID()`, `ParseCorrelationID()`, `MustParseCorrelationID()` helpers
- E2E test verifying event.NewEvent accepts options from EventOptionsFromContext
- `ContextEnrichmentMiddleware` now validates `X-Correlation-ID` as ULID; non-ULID values are silently dropped

### Changed

- **`WithCorrelationID` / `CorrelationIDFromContext`**: context now stores strongly-typed `id.CorrelationID` (ULID-backed branded type) instead of `string`. **Breaking change for consumers** passing raw strings — use `MustParseCorrelationID()` in tests, `NewCorrelationID()` to generate, or `ParseCorrelationID()` in production.
- `AuthorizeMiddleware` now prefers branded `UserID` from context over raw extractor string; falls back to extractor + `ParseUserID()` validation. Unparseable ULIDs now return 401 instead of passing raw strings to Casbin.
- Context keys are now empty-struct sentinel types (`userIDKey{}`, `correlationIDKey{}`, `htmxKey{}`) instead of string-based types — standard Go pattern for collision-free context values.

### Fixed

- `CorrelationIDFromContext` → `EventOptionsFromContext` → `event.WithCorrelationID()` pipe now fully wired with branded `id.CorrelationID` type
- Dead test: "returns error when casbin enforce fails" now actually tests failure (asserts error instead of asserting success)
- `EventOptionsFromContext` now propagates `CorrelationID` alongside `UserID` into event metadata

## [1.0.0] - 2026-05-16

### Added

- BDD test suite using Ginkgo/Gomega (`bdd_test.go`)
- `DecodeFormQuery` handler option for query parameter form decoding (symmetry with `DecodeForm`)
- `docs/` directory with architecture reviews, planning docs, and status reports
- `NewUserID()`, `ParseUserID()`, `MustParseUserID()` helpers; `type UserID = id.UserID` re-export
- `NotificationLevel` type with `LevelSuccess`, `LevelError`, `LevelWarning`, `LevelInfo` constants
- `JSONErrorHandlerWithRedirect` for JSON error responses with custom login redirect
- Dispatch lifecycle hooks: `BeforeDispatchHook` / `AfterDispatchHook` on `Config`
- Request validation: `ValidateCommand` / `ValidateQuery` HandlerOptions with `ErrValidationFailed`
- Correlation ID propagation: `WithCorrelationID` / `CorrelationIDFromContext`
- Timeout propagation: `Config.Timeout` wraps dispatch with `context.WithTimeout`
- Godoc examples (7 `Example*` functions)
- Benchmark tests (10 sub-benchmarks)
- `CONTRIBUTING.md` contribution guide
- GitHub Actions CI pipeline (build + test + lint + coverage gate)
- `authMode` typed enum for handler authorization (makes impossible states unrepresentable)

### Changed

- **`WithUserID` / `UserIDFromContext`**: context now stores strongly-typed `id.UserID` (ULID-backed branded type) instead of `string`. `UserIDExtractor` still returns `string`; middleware parses to `UserID`. **Breaking change for consumers** passing string literals or plain strings — use `MustParseUserID()` in tests, `ParseUserID()` in production.
- Extract helper functions (`hasNoResponse`, `hasMinimalResponse`, `decodeJSONBody`, `decodeRequest`, `decodeFormBody`, `notifyOption`, `triggerNotification`) to reduce duplication
- Extract notification helpers to dedicated `notify.go` module
- Consolidate duplicate test types across test files into shared helpers
- Remove local-path `replace` directives from `go.mod` — resolve from GitHub
- Notification levels now use `NotificationLevel` type instead of magic strings
- Remove `headerTrue` alias — use `HeaderTrue` everywhere
- `"X-Correlation-ID"` header extracted to `headerCorrelationID` constant
- `JSONErrorHandler` now delegates to `JSONErrorHandlerWithRedirect` with default redirect
- Authorization config consolidated from 4 fields (`authorize bool` + `requireAuth bool` + `resource` + `action`) to typed `authMode` enum + `resource` + `action`

### Fixed

- Use `headerRedirect` constant instead of hardcoded `"HX-Redirect"` string in `DefaultErrorHandlerWithRedirect`
- Thread `Config.LoginRedirect` into per-App error handler (was dead code — `New()` now creates a closure that captures the resolved loginRedirect)
- Use `headerTrue` constant in `Response.Refresh()` instead of hardcoded `"true"`
- Fix README compile-breaking example (`cqrshtmx.LoginRedirect` → `Config.LoginRedirect`)
- Fix error wrapping: `errors.Wrapf` with `%s` on sentinels → `fmt.Errorf("%w: ...")` throughout
- `Enforce(nil, ...)` error now includes all three fields (subject, resource, action) — was missing subject
- `JSONErrorHandler` now respects `Config.LoginRedirect` (was hardcoded to `/login`)

### Removed

- Remove dead `enrichContext()` no-op stub
- Remove redundant gocritic `disabled-checks` entries (`dupImport`, `octalLiteral`, `whyNoLint` — already disabled by default)
- Remove unused `io` and `event` imports from test files
- Remove dead sentinels `ErrNoUserID` and `ErrRendererMissing` (exported but never returned by any code path)
- Remove deprecated `DefaultNotificationEvent` var (race risk, unexported constant used internally)
- Unexport internal sentinels: `ErrCommandsNil` → `errCommandsNil`, `ErrQueriesNil` → `errQueriesNil`, `ErrDecoderMissing` → `errDecoderMissing`

## [0.2.0] - 2026-05-07

### Added

- Eliminate all 103 golangci-lint issues → 0 issues, project is lint-clean
- `.golangci.yml` v2 format with proper exclusion rules
- Comprehensive test coverage at 93.5% (138 specs)

## [0.1.0] - 2026-05-04

### Added

- **App builder**: `App` struct with `Config`, `Command()`, `Query()`, per-App `ErrorHandler` and `LoginRedirect`
- **CQRS dispatch**: `handleCommandDispatch()` and `handleQueryDispatch()` with automatic error handling
- **Handler options**: `DecodeJSON`, `DecodeForm`, `Render`, `RenderTempl`, `RenderTemplResult`, `Authorize`, `Enforce`, `UserIDExtractor`
- **HTMX response builder**: Fluent API with `Response` struct — `StatusCode()`, `Header()`, `Redirect()`, `Refresh()`, `Retarget()`, `Reswap()`, `Trigger()`, `TriggerAfterSettle()`, `TriggerAfterSwap()`
- **HTMX middleware**: `HTMXMiddleware` parses `HX-*` headers once, stores `HTMXRequest` in context
- **HTMX context accessors**: `IsHTMXRequest()`, `GetHTMXPrompt()`, `GetHTMXTarget()`, `GetHTMXTrigger()`, `GetHTMXTriggerName()`, `RenderPartial()`
- **Notification system**: `NotifySuccess`, `NotifyError`, `NotifyWarning`, `NotifyInfo` — standard `{level, message}` trigger pattern via `notify.go`
- **Casbin authorization**: `Authorize()`, `Enforce()`, `AuthorizeMiddleware()` using `casbin/casbin/v3`
- **Context enrichment**: `ContextEnrichmentMiddleware` + `UserIDExtractor` → context → event metadata
- **Error classification**: CQRS error → HTTP status mapping with `RegisterClassification`, sentinel errors (`ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrBadRequest`, `ErrConflict`, `ErrInternal`), and `LoginRedirect` support
- **templ integration**: `TemplComponent` duck-typed interface (no `a-h/templ` import dependency) with `RenderTempl` and `RenderTemplResult` options
- **Middleware chain**: `Chain()` utility for composing `net/http` middleware
- **Git Town integration**: `git-town.toml` configuration

### Changed

- Deduplicate `UserIDExtractor` calls — handlers check context first, skip if middleware already set user ID
