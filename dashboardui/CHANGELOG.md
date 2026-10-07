# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **Read-only telemetry phase 1 — `/telemetry` panel + engine-health probe composition (M26):** four optional Config providers (`Topology`, `EngineHealths`, `Placements`, `EngineStats`) light up a Telemetry nav entry + route with three sections: the system topology table (instances/buses/projection-host), the metaengine query-placement table (query → engine + ADT + volume + est. latency), and engine latency cards (RTT EWMA/P95/samples, freshness badge at 5 minutes, explicit "no tracker"/"no samples" states). `EngineHealths` also composes into the probes: `/-/healthz` carries an `engines` array (liveness stays 200 — a sick engine is not a dead process) and `/-/readyz` returns 503 `engine_unhealthy` while any engine is down, so orchestrators keep the pod out of rotation. Providers return dashboard-owned `core` view structs (no system/metaengine types in the rendering layer); sections render independently with inline errors; without providers the route, nav, and probe payloads are byte-for-byte unchanged. `systembridge.WireTelemetry(&cfg, sys)` (a new optional sub-package — the one place dashboardui imports the system/metaengine packages) maps a real `*system.System` onto all four in one call. Pinned by panel/capability-gating/inline-error/probe-composition tests and the `TestDashboard_TelemetryFromSystem` integration proof.

- **`FromSystem(sys)` — mount the dashboard over a go-cqrs-lite `system.New` instance in one call (M25):** maps the system's event store (EventSource + whatever Journal/Seekable/EventByID/StreamReader its engine implements — engines without a native stream reader get a journal-derived one, so the aggregates panel lights up everywhere), bus (SSE live updates), projection host, snapshot store, and command/query stores onto a ready Config; the caller layers Title/BasePath/Authorizer on top. The seam is a duck-typed `System` accessor interface satisfied structurally by `*system.System` and any wrapper forwarding the accessors — dashboardui gains NO system/metaengine dependency (capability seams stay leaf-typed), and wrappers/decorators stay probeable where a concrete assertion on the store would fail. `Autodetect` recognizes `System` values in its source list (mixing stores and systems under the usual first-wins merge). Pinned by duck-fake unit tests (full/partial mapping, first-wins) plus the integration test `TestDashboard_FromSystem_PanelsLightUp` — a real `system.New` on the memory preset renders Overview/Events/Aggregates/Projections through the mounted routes. README § Integration Modes gains the "From a go-cqrs-lite system" mode.
- **`Routes()` exposes the dashboard's HTTP route manifest (M21):** `[]Route` entries carry `Method`, `Pattern` (consumer-facing, wildcards in braces), `Panel`, and `Write` — the exact surface the instance serves, honoring capabilities and ReadOnly (write routes vanish in read-only mode). Built for consumer-side allowlists (reverse proxies, policy engines) and route-surface diffs in upgrade tests. Internally, route registration and the manifest are now generated from ONE route table (`routeSpec` rows with guard mode + capability conditions) — the eleven-block if-chain in `routes()` and its cyclop suppression are gone, a new guard mode defaults to fail-closed, and a `mux.Handler`-based test pins manifest ≡ registration (every listed route resolves to its exact ServeMux pattern). README § Route Manifest shows the allowlisting pattern.
- **`Autodetect` accepts multiple sources and merges them first-wins (M20):** the signature is now `Autodetect(sources ...any) (Config, error)` — pass the primary store first and auxiliary stores (a dedicated snapshot store, a projection host) after it; each capability is claimed by the earliest source implementing it, and conflicts never error. Single-source calls are unchanged (the variadic is backward compatible). Internally the probe chain became a table-driven `fieldProbe[T]` design (one generic probe over `readProbes`/`optionalProbes` tables), replacing the eleven-branch type-assertion chain and its cyclop suppression. The read-interface gate is aggregate (any source satisfying it is enough), zero sources or a value implementing nothing is a Rejection naming the source types, and first-wins/complementary-merge/read-from-later-source behavior is pinned by tests. README § Integration Modes documents the merge semantics.
- **`Config.Validate()` exported — pre-validate without constructing a Dashboard (M19):** returns the normalized config (defaults applied) or a Rejection-family error naming the first invalid field; `New` applies the same path automatically, and consumers can now fail fast at startup. The page-size default/max (50/200) are single-sourced from `core.DefaultPageSize()`/`core.MaxPageSize()` — the root package's duplicate constants are deleted, so the core values are the only source. Pinned by normalization/clamp/rejection/accent-color tests plus a New≡Validate equivalence check. (setup needs no call-site change: it composes the dashboard config at runtime from the store and already propagates `dashboardui.New`'s validation error.)
- **Audit log entries carry WHO performed write operations, and a new actor-aware Authorizer (M18):** `Config.ActorAuthorizer func(*http.Request) (Actor, error)` returns an `Actor{ID, Name}` on success; the dashboard injects it into the request context and every `dashboardui.audit` log line (projection reset, DLQ replay/delete/purge, snapshot delete — success AND failure paths) now ends with `actor_id`/`actor_name` plus `request_id` when the consumer runs httputil's request-ID middleware. Consumers using their own middleware can inject the actor directly via the exported `WithActor`/`ActorFromContext` context helpers — attribution works with no dashboard Authorizer at all. The legacy `Authorizer` keeps working unchanged (audits then record `"actor":"anonymous"` unless middleware injects one); when both are configured, `ActorAuthorizer` takes precedence. Pinned by guard tests (deny/allow/inject/precedence) and slog-capture tests asserting the exact audit attributes for the identified and anonymous cases.
- **`/-/versionz` reports real build metadata and can be auth-guarded (M17):** the `module` field is now derived from the binary's build info via reflection instead of a hardcoded constant — a fork reports its own module path — and is joined by `version` (published semver when linked as a tagged dependency, `"(devel)"` in-module, empty for local directory replaces) and the binary's VCS stamp (`vcsRevision`, `vcsTime`, `vcsModified`; omitted when built without VCS metadata). New `Config.VersionzRequireAuth` opts the endpoint into the configured `Authorizer` (403 on denial); the default stays public like healthz/readyz so load-balancer probes need no credentials — the trade-off (Title, BasePath, capabilities, module version, VCS revision are readable unauthenticated) is documented in README § Observability Endpoints. Resolution logic pinned by table tests over synthetic `debug.BuildInfo` (main-module, dependency, versioned replace, directory replace, absent) and endpoint tests for all three guard outcomes.

### Fixed

- **Asset serving delegates to the root module's `cqrshtmx.ServeAsset` (2026-10-07, art-dupl dedup round):** the immutable-asset posture (content-derived ETag, 1-year immutable caching, nosniff) now lives once in the root module (`cqrshtmx.ServeAsset`/`cqrshtmx.AssetETag`/`cqrshtmx.AssetFromFS`) instead of duplicated with adminui; the const-asset `/-/dashboard.js` route additionally gains `X-Content-Type-Options: nosniff` it previously lacked, only the `embed.FS` declaration and config-error wrapping stay local (a missing embedded asset now surfaces as a wrapped Infrastructure-family error). ETag values change format from `"dashboardui-<name>-<16-hex>"` to `"<name>-<unpadded-hex>"`; consumers see one revalidation, then behavior is identical. Pinned by the same 304 round-trip tests plus root's header/conditional-GET/read contract tests.

- **Asset ETags are content-derived and cache headers immutable (M16):** every served asset (`/-/dashboard-tw.css`, `/-/dashboard.css`, `/-/dashboard.js`) now carries an ETag computed from its own bytes (FNV-1a, `"dashboardui-<name>-<hash>"`) instead of the hand-bumped `dashboardui-v4.9.0` constant — the class of bug where the compiled bundle changes but the version const does not, and every consumer's cached copy silently diverges. `Cache-Control` moves from `public, max-age=86400` to `public, max-age=31536000, immutable` (the root module's `HTMXScriptHandler` posture); the JS route previously had NO ETag at all and now answers `If-None-Match` with 304 like the CSS routes. Pinned by round-trip 304 tests for both assets, a stale-ETag re-serves-full-body case, and a content-hash unit test.
- **URL-parameter injection hardened across every link builder (filter, sort, pagination):** generated links now percent-encode query values at construction time — `core.EventFilter.ExtraParams` (type/streamType/streamID), the sortable-header hrefs and the sort-state fragment (sort/dir), and `core.PaginationQuery` (after/prev — the prev history's embedded commas now travel as one value instead of splitting). Hostile filter input (`&`, `=`, `#`, space, unicode) can no longer smuggle extra parameters into sort headers, page-size options, or Prev/Next links; pinned by round-trip tests at the builder level and a handler-level test whose events literally carry the hostile value as their type.
- **Malformed `after` cursors are a 400, not a silent page reset:** the events/commands/queries index pages used to swallow cursor-parse failures, quietly dropping the user back to page one (a "Next" that never advanced). They now render a 400 error page and log the raw value.
- **ReadAll fallback pagination honors the requested page size:** the command/query journals' non-seekable fallback truncated before computing hasNext, hiding the Next link entirely — and the query fallback truncated with the CONFIG default instead of the requested `?limit=`. Both now share the normal truncate-after-hasNext path.
- **Detail handlers map error family to status (event/command/query):** a not-found (Rejection) is a 404, an infrastructure failure is a 500 with the cause logged — previously every load error read as "404 not found", hiding store outages behind a misleading page. Root cause fixed at the loader: `core.loadEventFromAll`'s terminal not-found error carried the Infrastructure family and a copy-pasted "no event source available" message.
- **`writeJSON` marshals before committing the status:** a marshal failure after `WriteHeader` used to strand the intended status (often 200) on the wire with an error body appended; it now writes 500 + `{"error":"marshal_failed"}` before any status is committed.
- **Write-action audit logs carry the error at Error level:** DLQ replay/delete/purge, snapshot delete, and projection reset failures now log `slog.ErrorContext` with the `error` attached (was Info with just `result=error`), so the cause is greppable next to the audit trail.
- **The sorted events view is honest about being a window:** sorting without filters loads the most recent `FilterScanLimit` (500) events, sorts in memory, and now shows the ENTIRE window with a "sorted view: first 500 events" badge — previously it silently truncated to the page size and offered a Next link that reloaded and re-truncated the same window forever. Pagination controls (including the page-size selector) and the misleading "of Z" total are hidden in this mode because cursor paging cannot follow a re-sorted order; filter+sort views keep cursor pagination (sorting applies within each page). README § Sorting documents the window semantics.

### Changed

- **`core.ListStreamsPaged` returns an error (breaking signature):** `([]listing.StreamListing, PageState)` → `([]listing.StreamListing, PageState, error)`. Reader failures (and a misbehaving nil page) now surface instead of silently rendering an empty table; the wrap is family-preserving, so a reader that classifies its error as a Rejection still maps to 400 at the stream-listing pages (aggregates/snapshots/time-travel) while everything else renders a 500 error panel.

## [v4.13.0] - 2026-10-04

### Added

- **Integration modes — destination by default, composable by choice (`Autodetect`, `Config.Layout`, `setup.Config.DashboardLayout`):** the dashboard can now be embedded instead of only mounted. `Autodetect(store any) (Config, error)` probes a store for every go-cqrs-lite introspection interface (EventSource, Journal, SeekableJournal, StreamReader, ProjectionHost, DeadLetterStore, command and query journals, SnapshotStore, EventBus, EventByIDLoader) and returns a ready `Config`, eliminating the hand-written type-assertion dance in consumer wiring; `Config.Layout` (`LayoutFunc`) is the embed seam — when set, every full-page render hands the page metadata (`PageMeta`: title, brand, base path, capability-filtered `Nav` as exported `NavLink` values with templ-components icon names, required CSS/script URLs, CSP nonce, capabilities) plus the ready-rendered content to a consumer-owned function whose component becomes the response, so panels render inside the consumer's app chrome instead of a standalone bolt-on page (HTMX partials bypass the shell; nil keeps the built-in standalone document byte-identical); setup exposes the same seam as `DashboardLayout`. Pinned by `autodetect_test.go` + `layout_embed_test.go` (shell replacement, meta/nav/asset URLs, HTMX bypass) and the setup passthrough test.
- **DLQ table gained a count notice (`display.ListNote` `ListNoteCount`, templ-components v1.19.1):** the dead-letter table for a projection now ends with "Showing N items." (pluralized, `aria-label="Dead letter count"`) — the DLQ page's key question is HOW MANY letters exist, and the count is the blast radius for the Replay All / Purge All confirmations. The hand-rolled `Showing X–Y of Z` pagination info stays (ListNote speaks N-of-M/N-items, not X–Y ranges); the old wholesale `display.ListNote` exclusion is lifted. Markup pinned by `list_note_count.golden` + DLQ handler assertions.

### Changed

- **Full templ rendering migration (2026-09-22):** every page now renders
  through `a-h/templ` — `layout.templ`, `components.templ`, and one
  page template per panel (overview, events, aggregates, projections,
  commands/queries audit, DLQ, snapshots, time-travel). The former
  `strings.Builder` + `fmt.Fprintf` HTML layer (49 render functions, 155
  manual `esc()` calls) is deleted; templ auto-escapes every interpolation
  (`templ.EscapeString` is `html.EscapeString`, so escaped content is
  byte-identical) and `href`-class attributes gain URL scheme sanitization
  (`javascript:`-style values render as inert `about:blank#blocked` links
  instead of raw anchors). Markup was transcribed faithfully — same elements,
  classes, and attributes — with three HTML5-equivalent cosmetic deltas:
  lowercase `<!doctype html>`, void elements without the self-closing slash,
  and inter-element whitespace. Handlers render components straight to the
  `http.ResponseWriter` (`renderPage`/`writeHTML` now take a
  `templ.Component`); generated `_templ.go` files are committed (consumers
  run no codegen) and `nix run .#gen` / `.#check-codegen` /
  `.#build-dashboardui-css` now cover dashboardui.

## [v4.11.0] - 2026-09-19

### Fixed

- **Time-travel slider worked only without CSP (2026-09-19, N17):** the version slider carried inline `onchange`/`oninput` attributes, which nonce-based CSP (the recommended `RecommendedSecurityMiddleware` posture) silently blocks — slider navigation and live version display were dead under a CSP-enforcing consumer. Replaced with `data-nav-base`/`data-slider-display` attributes plus CSP-safe external listeners in the layout script (same behavior: drag updates display + `aria-valuetext`, release navigates; arrow keys unchanged). The inline handlers had also escaped `TestCSP_NoInlineEventHandlers`, because the sweep rendered only listing routes — the test now seeds a stream, sweeps the slider detail page, and fails loudly if the slider stops rendering (mutation-verified).

### Added

- **Event filter bar renders `forms.Input` (N17.3–4):** the three filter fields (Type, Stream Type, Stream ID) route through the library's labeled input instead of hand-rolled `<input>` markup; explicit DOM ids (`filter-type`, `filter-stream-type`, `filter-stream-id`) and the `hx-get` partial-swap wiring are pinned by a new label-pairing contract test (`TestA11y_FilterInputsKeepLabelPairing`). Slider gained `aria-valuetext` and a `:focus-visible` ring; the Tailwind bundle was rebuilt for the new input utility classes.
- **Render benchmarks for every adopted family (N16.1–5):** `render_bench_test.go` now covers Button (link/submit), EmptyState, DefinitionList (pre-M20 meta-table baseline with honest copy-button parity), and the Table raw-body vs data-row paths. Findings recorded in `docs/benchmarks/dashboardui-render-2026-09-19.md`: the DefinitionList swap costs only ~1.3x, and the raw-body table path is ~4.3x cheaper than typed data-rows at 10 rows — keep `tableHTMLRaw` for string-built listings.
- **`templ.Component` added to the root `ireturn` allow list (N16.6):** the templ component contract (returning `templ.Component` from render helpers) is idiomatic; the two `//nolint:ireturn` directives in `buttons.go` are gone.

## [v4.10.1] - 2026-09-19

### Fixed

- **Compiled Tailwind bundle rebuilt against the templ-components v1.18.0 class set** (+2,104 bytes): v4.10.0's embedded `dashboard-tw.css` was generated before the v1.18.0 sweep and missed the new/renamed utilities the adopted components now emit (e.g. the 28px table-sort touch targets from the library's accessibility pack). Rule reinforced: rebuild `nix run .#build-dashboardui-css` after EVERY templ-components family bump, in the same change.

## [v4.10.0] - 2026-09-19

### Added

- **templ-components adoption (M4–M8):** dashboardui now renders errors, 404s, status badges, and stat cards through `github.com/larsartmann/templ-components` (v1.17.0 family) instead of hand-rolled CSS, while keeping the strings.Builder hybrid render path (no templ conversion).
  - **Compiled Tailwind bundle** (`assets/dashboard-tw.css`, `assets.go`): embedded stylesheet served at `/-/dashboard-tw.css` with ETag/304 + cache headers; built by `nix run .#build-dashboardui-css`, which scans the library's `.templ` files AND errorpage's `styles.go` (runtime class strings live in Go source) and fails loudly on a utility-free output.
  - **Styled error pages** (`render.go`, `errors_test.go`): `renderError` maps HTTP status → error family (409 Conflict, 502/503/504 Transient, 4xx Rejection, else Infrastructure) and renders `errorpage.ErrorPage` — bare card for HTMX requests, full documented shell otherwise; `notFoundHandler` upgraded to `errorpage.NotFound404`.
  - **Library badges everywhere** (`badges.go`, `format.go`): every status/encoding/count badge routes through `display.StatusBadge`/`display.Badge` via `badgeHTML`; unknown kinds keep raw text as explicit neutral badges.
  - **Library stat cards with stable DOM hooks** (`stats.go`): overview + projection-detail stats render through `display.StatCard` with ValueIDs (`stat-total-events`, `stat-system-health`, `stat-<metric>-<slug>` …) so live-updating scripts survive future markup refactors.
  - **Definition lists** (`definitions.go`): all six detail pages (event, command, query, DLQ entry, projection, snapshot) render metadata through `display.DefinitionList` with inline copy buttons.
  - **Copy buttons** (`buttons.go`): every copyable ID and the event payload render `display.CopyButton`; the hand-rolled click-to-copy cell JS and `data-copyable` attribute protocol are gone.
  - **Library buttons** (`buttons.go`): all navigation and form buttons render `display.Button` (Secondary / OutlineInfo / OutlineDanger); disabled pager links use the library's aria-disabled treatment.
  - **Empty states** (`render.go`): every empty state renders `display.EmptyState` (`role="status"`, icon tile) with per-page nav icons.
  - **Library tables**: all listing tables (events, commands, queries, aggregates, projections, dead letters, snapshots, time travel, recent events) render `display.Table` with typed headers (aria-sort), server-side sort links, and LazyRows content-visibility.
  - **Toast host** (`layout.go`): `feedback.ToastContainer` mounted in the layout; write handlers emit a `dashboardui:toast` Hx-Trigger event bridged to `tcShowToast` by the external script (CSP-safe).
  - **Global error handling** (`layout.go`): `htmx.GlobalErrorHandling` retries 5xx swaps with backoff and announces failures (2 retries, 1s base delay).
  - **Page-size selector** (`pagination.go`): renders `forms.Select`.

### Changed

- **Projection status semantics:** `stopped` workers now classify as healthy (`StatusGood`), aligning the dashboard with the root library's readiness gate (`ProjectionReadinessCheck` treats live/stopped as ready). Journal-only hosts drain to `stopped` when fully caught up and no longer show a false "Unhealthy"; only `failed` marks a projection unhealthy.
- **Context plumbing:** render handlers thread `r.Context()` end-to-end (no synthetic `context.Background()` in render closures).

### Fixed

- **CONSUMER-VISIBLE semantic change (surfaced from Changed):** `stopped` projection workers now classify as HEALTHY. If you alert on "Unhealthy" badges, journal-only setups that false-alarmed before will now show green - that is the intended alignment with `ProjectionReadinessCheck`; only `failed` marks unhealthy.
- **SSE stream URL 404 on every deployment:** the client-side base derivation in `dashboardJS` stripped the script's `/-` suffix with `/\/-\/$/`, a pattern requiring a trailing slash that never exists once `/dashboard.js` is removed — every page computed `<base>/-/-/events/stream` and the EventSource 404'd (connect/reconnect loop, never live). The pattern now strips the trailing `/-` (`/\/-$`); pinned by `TestDashboardJSSSEBaseURL`.
- **Health card test false-positive:** the overview health test's `stat-card ok` assertion had been matching the _Projections_ card's CSS class since inception; it now asserts the actual health value scoped to the `stat-system-health` element. The event-count test similarly asserts the exact value ("10") inside the `stat-total-events` element instead of page-wide string containment.

## [v4.2.0] - 2026-08-07

### Added

- **Detail views for commands, queries, and DLQ entries** (`handlers_audit.go`, `handlers_dlq.go`, `handler.go`): New routes `/commands/{id}`, `/queries/{id}`, `/dead-letters/{projection}/{eventID}` render full detail pages with metadata tables, copyable IDs, and pretty-printed JSON payloads.
- **Projection detail view** (`handlers_projections.go`, `handler.go`): New route `/projections/{name}` shows checkpoint, processed/errors/restarts counters, lag, last error, DLQ link, and reset action for a single projection. Projection names in the index table are now clickable links.
- **Total count display** (`pagination.go`, all index handlers): Paginated pages now show "Showing X-Y of Z" when the total is known (last page). Uses `WithCountInfo()` builder pattern — no extra count queries for append-only logs.
- **Page-size selector** (`pagination.go`): Dropdown in the pagination bar lets users choose 25/50/100/200 items per page via `?limit=` query param.
- **Sortable column headers** (`sort.go`, `handlers_events.go`): Events table columns (Time, Type, Stream Type, Version) are clickable for ascending/descending sort via `?sort=` and `?dir=` query params. Headers show arrow indicators.
- **HTMX-powered filter form** (`handlers_events.go`, `layout.go`): Event filter form uses `hx-get`/`hx-target`/`hx-select`/`hx-swap`/`hx-push-url` for partial content swapping without full page reloads.
- **HTMX partial rendering** (`render.go`, `layout.go`): Server detects `HX-Request` header and renders title+main only (no full HTML document). Layout includes `data-hx-boost` for progressive enhancement.
- **Payload copy and download** (`handlers_events.go`, `layout.go`): Event detail page has Copy (clipboard API with toast) and Download JSON (Blob download) buttons for event payloads.
- **Keyboard navigation for time-travel slider** (`handlers_timetravel.go`, `layout.go`): Arrow keys anywhere on the time-travel page move the version slider. Live value display updates as the slider moves.
- **CSV export** (`export.go`, all index handlers): `?format=csv` on events, commands, and queries index pages exports up to 10,000 rows as a downloadable CSV file.
- **JSON API mode** (`export.go`, all index handlers): `?format=json` on events, commands, and queries index pages returns a JSON array of row objects.
- **SSE live event injection** (`layout.go`): JavaScript listener for `dashboard:event` custom events prepends new rows to the events table and refreshes projection health (capped at 50 rows).
- **Core data layer** (`core/` package): Capabilities, pagination, events, overview, payload, and format functions extracted into a pure-data `core` subpackage, bridged via type aliases in `core_bridge.go`.
- **Core package unit tests** (`core/*_test.go`): Comprehensive tests for DetectCapabilities, EventFilter, pagination cursor math, FetchOverview, LoadRecentEvents/LoadFilteredEvents/LoadEventByID, FindEventNeighbors, RelativeTime, HumanByteSize, DefaultPayloadRenderer, PrettyJSON, and DLQProjectionLinks.
- **CSP-safe rendering** (`layout.go`, `handlers_dlq.go`, `handlers_projections.go`, `handlers_snapshots.go`): All 6 inline `onsubmit="return confirm(...)"` handlers replaced with `data-confirm` attribute pattern. Inline toast `<script>` block moved to external `dashboardJS`. Single delegated `submit` listener for `data-confirm` forms. `dashboard.js` now always loaded (was conditional on EventBus).
- **Demo projection host** (`examples/dashboard-demo/main.go`): Demo now includes a projection host with a `user-read-model` projection so the projections panel and detail view show live data.
- **Templ migration evaluation** (`docs/planning/templ-migration-evaluation.md`): Documents the tradeoffs of migrating from strings.Builder to templ, with a recommendation to defer.

### Changed

- **`IMPROVEMENT_IDEAS.md` pruned** from 883 lines to ~60 lines after implementing the high-value ideas.
- **Generic journal scanning** (`handlers_audit.go`): `loadCommandByID`/`loadQueryByID` refactored to share `scanJournalByID`/`findInAll` generic helpers, eliminating code duplication.
- **Lint config updates** (`.golangci.yml`): Fixed broken canonicalheader exclusion patterns, added exhaustruct excludes for dashboardui types, added wrapcheck/revive exclusions for core_bridge.go.

### Fixed

- **Mobile responsive design** (`layout.go`, all handler files): Hamburger menu with slide-in sidebar drawer and backdrop overlay for screens <768px. All buttons have 44px minimum touch targets (WCAG 2.5.5). Data tables wrapped in horizontal scroll containers. Filter bar controls stack vertically on mobile. Stat card grid collapses to 2 columns.
- **Accessibility aria-labels** (`handlers_projections.go`, `handlers_dlq.go`, `handlers_snapshots.go`): All interactive elements (Reset, Replay, Delete, Purge buttons and forms) now have descriptive `aria-label` attributes for screen reader users.
- **Skip-to-content link** (`layout.go`): Keyboard users can bypass the sidebar navigation directly to main content.
- **Copy-to-clipboard** (`layout.go`, `handler_overview.go`, `handlers_events.go`, `handlers_aggregates.go`, `handlers_audit.go`, `handlers_timetravel.go`, `handlers_snapshots.go`): Identifiers (event IDs, stream IDs, correlation/causation IDs, user IDs, request IDs) are click-to-copy with toast confirmation.
- **Health/filter/CSS-JS/pagination tests** (`handlers_health_test.go`, `handlers_security_test.go`): 8 new tests covering healthz/readyz/versionz endpoints, event filtering by type, CSS/JS serving headers, and pagination cursor preservation with active filters.
- **EventBus live updates in demo** (`examples/dashboard-demo/main.go`): Demo now wires `eventtest.FakeBus` for SSE live updates and publishes new events every 5 seconds. Expanded seed data to 8 users and 6 orders (28 events for pagination testing).
- **ROADMAP** (`ROADMAP.md`): Documents the future templ + Tailwind v4 migration plan.

### Fixed

- **DLQ format string crash** (`handlers_dlq.go`): Replay and Purge form opening tags had 4 `%s` placeholders but only 3 arguments — would have panicked at runtime. Fixed by adding the missing 4th `esc(proj)` argument.
- **Heading hierarchy** (all handler files): Corrected heading levels throughout — page titles use `<h2>`, section headers use `<h3>`, sub-sections use `<h3>` instead of incorrectly nested `<h4>` under `<h3>`.
- **`encodingBadgeClass` semantics** (`format.go`): JSON/empty encoding changed from `badge-ok` (green) to `badge-neutral` (gray) — JSON is not a "success" state.
- **JSON export encoder** (`export.go`): Removed `encoder.SetEscapeHTML(true)` — method does not exist on `jsontext.Encoder` (stdlib-only API). Replaced with `jsontext.NewEncoder(w, jsontext.WithIndent("  "))` matching the usermgmt export pattern.

## [v4.1.0] - 2026-07-26

### Added

- **SSE reconnect replay** (`sse.go`, `dashboard.go`): the SSE handler reads `Last-Event-ID` on reconnect and replays missed events from the journal via `cqrshtmx.JournalSSEStore` + `cqrshtmx.ReplayEvents`. On first connect, recent history is backfilled (up to `DefaultMaxReplay=1000`). `Dashboard.Close()` disconnects all SSE clients.
- **SSE event IDs** (`sse.go`): emitted SSE events now carry the domain event ID, enabling reconnect dedup.
- **Heartbeat config** (`config.go`): `Config.SSEHeartbeatInterval` (15s default) drives a heartbeat to keep proxies from killing idle connections.

### Fixed

- **`Dashboard.Close()` event-bus subscription leak** (`dashboard.go`, `sse.go`): `Close()` now signals a done-channel that makes the event-bus handler a no-op before closing the broadcaster. Uses `sync.Once` for idempotent shutdown.
- **ErrorFamily compliance** (`handlers.go`, `payload.go`): migrated 7 `fmt.Errorf` calls to `errorfamily` constructors (`WrapInfrastructure`, `WrapCorruption`, `Newf`). `nix run .#errorfamily` now reports 0 violations for dashboardui.

## [v4.0.0] - 2026-07-24

### Added

- **First release** of the dashboardui module: a ready-made CQRS/Event-Sourcing observability dashboard.
- **Dashboard panel** (`dashboard.go`, `handler.go`, `handlers.go`): Plug-in HTTP dashboard showing projection health, event catalog overview, and system status. SSE-powered real-time updates.
- **SSE integration** (`sse.go`): Server-Sent Events endpoint for live projection status streaming to the dashboard.
- **Layout and rendering** (`layout.go`, `render.go`, `handler_overview.go`): Full HTML layout with sidebar navigation and responsive design.
- **Config** (`config.go`): `Config` struct with capability-detected interfaces (EventSource, Journal, SeekableJournal, StreamReader, ProjectionHost, DeadLetterStore, CommandJournal, QueryJournal, SnapshotStore, EventBus), customizable title, accent color, page size, and read-only mode.
- **Payload rendering** (`payload.go`): `PayloadRenderer` interface and `DefaultPayloadRenderer` for JSON/CBOR pretty-printing of event payloads.
