# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **User search rides the HTTP QUERY ladder (RFC 10008).** The users panel's search was GET-only; complex criteria (name + status + page) had to squeeze into URL params. New routes: `QUERY /users/search` (the native body-carrying read) and `POST /users/search` (the fallback twin) register the SAME handler, so every rung of the fallback ladder answers byte-identically; `MethodOverrideMiddleware` is wired into the panel's own stack, so a POST + `X-HTTP-Method-Override: QUERY` reaches the native rung (only QUERY is an override target — never an unsafe verb). The search input upgrades Enter-submits to native QUERY via the embedded `cqrs-query` extension (served at `/-/ext/cqrs-query.js`, auto-downgrading only when an intermediary refuses the verb) and fires the POST twin on live typing; the GET route stays the bookmarkable rung the filter chips and history navigate. The criteria ride one shared data path with the GET route (`usersSearchData`), so the rungs cannot drift on filtering, pagination, or enrichment. Pinned by `user_search_test.go` (ladder byte-parity across QUERY/POST/override/legacy-override, body criteria parity with URL params, oversized + malformed body 400s, anonymous 401, extension-script asset posture, rendered ladder attributes) and the e2e chromium ladder tests (request-method assertions: Enter → native QUERY, typing → POST twin).

- **Bulk import form adopts the library's `forms.FileInput`.** The import page's hand-rolled file input (its own label element, a 12-class hand-rolled class string, and a hand-linked help paragraph) moved to templ-components' `forms.FileInput` — the same styled native button, ring focus, and help-text contract as every other form field in the panel, with the `import-file` DOM hook preserved. CSS bundle rebuilt in the same change (the library's file-button utilities are harvested now). Pinned by the existing `TestImportPage_Renders` render assertions (unchanged, they still hold) plus the class-set gate (`check-css-bundle-classes`).

- **Accessibility sweep over the touched surfaces (axe + keyboard), with the first findings fixed.** The panel had never been audited with axe-core; the new `admin-a11y` e2e spec sweeps the tenants/users lists and the tenant/user detail pages (critical/serious only, accepted findings must carry a written reason) and pins the keyboard path through the multi-role grant form: Tab reaches the tenant select, every role checkbox, and the submit; Space toggles; Enter submits — no mouse required. What the first sweep found and this wave fixed: the user-menu trigger was labelled with the bare email (WCAG 2.5.3 Label in Name — the accessible name must contain the full visible text, avatar initials included; it now names initials + email, and the tenant switcher's label names the tenant it currently shows), the users-list filter chips carried `aria-pressed` on links (invalid attribute mapping — the active chip uses `aria-current` now), two icon-only remove buttons (roster, user-page tenant access) had no accessible name at all (now named per target), and the new status chips' gray text missed 4.5:1 (Tailwind v4's gray-500; bumped to gray-600). Remaining debt documented, not hidden: ~115 pre-existing color-contrast findings (accent-tinted brand chrome + gray-400 metadata ramp) accepted with a written reason in the spec — a design pass for a later wave, and the ACCEPTED entry says so.

- **Tenant list polish: lifecycle chips + status filter and member-count sort.** The tenants overview mixed live, suspended, and deleted tenants with no way to narrow the view and only one fixed ordering. Filter chips (All / Active / Suspended / Deleted, each with the search-scoped live count) and a Name / Members ordering toggle ride the URL (`?status=`, `?sort=`), so a narrowed view is shareable; unknown filter values degrade to the unfiltered list rather than an empty one, and the chip precedence matches the status badge exactly, so a chip never promises a row its badge would render differently. Sorting by members counts every matched tenant before paging (the count is computed for the whole match set, not just the visible window). **Delete and suspend preflights name their blast radius** — the confirm dialog states exactly how many memberships the action revokes or cuts off, replacing the vague "remove all memberships" phrasing. Pinned by `tenant_polish_test.go` (status filter narrowing + unknown-value degradation through the HTMX partial, member sort with tiebreak, impact confirms carrying the real counts). NOT shipped from this wave: the per-tenant audit shortcut — the audit log records no tenant lifecycle entries (the audit middleware covers user/session commands only), so the shortcut would land on an empty search; recorded as the usermgmt ask in TODO_LIST item (h).

- **Header tenant switcher (super-admin mode).** Moving between tenants meant walking back to the tenants list every time; the header now carries a dropdown of the live tenants (display-name order) that jumps straight to each tenant's detail page and marks the one being viewed with a check icon — the trigger names it too, so "where am I" is answered at a glance. Suspended and deleted tenants stay out of the list (they are not navigable targets), and tenant-admin mode renders no switcher at all: that mode is scoped to exactly one tenant by construction, so a switcher there would offer navigation that does not exist. Same library Dropdown as the user menu, so native-Popover light-dismiss, Esc, and keyboard navigation come for free. Pinned by `tenant_switcher_test.go` (live-listing with suspended exclusion, tenant-admin absence, current-tenant marking on the detail page).

- **Bulk import/export of users (CSV + JSON, super-admin).** The panel could create users one at a time while the Service has shipped `ImportUsersFromJSON/CSV` and `ExportUsersTo*` since v4.20.0 with no surface — onboarding a team or migrating a directory meant scripting against the API. New routes: `GET /users/import` (the form), `POST /users/import` (multipart upload, dispatched to the published importer), `GET /users/export.csv` / `GET /users/export.json` (streamed straight from the Service — no whole-directory buffering — with `Content-Disposition` naming the file). Guardrails that keep the bulk path honest: the upload is size-capped before the parser reads it (413, not OOM); the CSV header must declare an `email` column or the import is refused up front (the importer maps columns by name and would silently land nothing); existing emails are SKIPPED by the importer itself, so re-running a file is safe and the result says so; and the result panel reports exact imported/skipped counts with a bounded per-row error list (counts exact, list readable). Every imported account is passwordless — the form says so where the create-user form does. Entry points on the users list (Import users, Export CSV). Zero usermgmt change: published v4.20.0 API only. Pinned by `import_export_test.go` (JSON+CSV round-trip through the real streaming handlers, duplicate re-run safety, malformed JSON / headerless CSV / unknown extension refusals, oversized-body 413, service-level end-to-end import, render bound).

### Fixed

- **The tenant/user surfaces now handle their edge states honestly (M03 wave).** Four gaps the tenant-integration wave left open: (1) the grant form's tenant select was a dead end for a user who belongs to every live tenant — the disabled prompt survived filtering and rendered a select with nothing selectable; `withoutTenantOptions` now returns empty when only the prompt remains and the card renders "member of every live tenant" instead of the form. (2) The true-empty users directory ("no users at all") rendered a dead-ended empty state; it now carries a **Create the first user** CTA into the existing new-user form (the search-miss variant keeps its Clear search escape hatch). (3) The inline unknown-email banner downgraded to `role="status"` — the library announces assertively (`role="alert"`) only for error-class feedback, and a failed grant IS an error-class miss, so the banner is now `FeedbackError` and screen readers announce it. (4) Member removal confirmed with one line no matter the blast radius: removing the tenant's ONLY admin-tier (admin/owner) holder now escalates the confirm dialog on both surfaces (roster + user page, passkey-case style: "the tenant would be left without an admin"), backed by a bounded per-tenant admin count the detail handler computes for the visible rows. Also in this change: `data-sync-target` markers on the tenant identity card and the user's tenant-access card — the ADR-0056 regions whose content moves outside a navigation. Pinned by `edge_states_test.go` (member-of-every-tenant, empty CTA, alert semantics, sync targets, last-admin escalation on both surfaces with its disappearance once a second admin exists) and the trued `user_tenant_access_test.go`/`handler_test.go` expectations.

- **A member can finally HOLD the two roles the API always allowed: the single role select became a checkbox group on both membership surfaces.** identity-model memberships are role SETS (`AssignableRoles()` = viewer/user/admin/owner), but both the tenant roster's per-row control and the user page's tenant-access row rendered ONE `<select>`, so correcting a multi-role membership meant two writes that fought each other — and the add-member form silently flattened a set to one role. The new `roleCheckboxGroup` primitive (one templ component, both surfaces: tenant roster rows, tenant add-member form, user-page grant + per-row controls) posts `roles` as a multi-value field — the same field name and parsing the platform-access card already uses. The handler-side `parseRolesForm` validates EVERY posted value against `AssignableRoles()` and refuses the whole request on none-checked or unknown values (a membership cannot be silently emptied; revoke is the explicit path), replacing the members path's former non-validating single-value read. No route, command, or API change — the same `AddMember`/`UpdateMemberRoles` dispatches, now carrying the full set. Pinned by `user_tenant_access_test.go` (set two → reduce to one → clear-all refused on the user surface), `flows_test.go` (the same trio on the roster surface), `TestPanel_TenantAdminUpdateMemberRole`, the unchecked-start render pins (`tenant_ux_test.go`, `seed_render_test.go`), and the re-templated goldens (`components_templ.go` regenerated with the canonical v0.3.1100 generator).

### Added

- **Tenant access can be managed from the user side (user detail), not only from each tenant's roster.** The user page is where an admin usually starts ("who is this person?"), yet memberships were read-only there and the only way to grant one was to navigate to the tenant first. The new **Tenant access** card lists every membership resolved to tenant identity (display name + slug, linked to the tenant page) with the roles held there, and offers three durable writes through the SAME membership commands the tenant roster dispatches: grant a tenant + role (`POST /users/{id}/tenants`), change a role in place (`POST /users/{id}/tenants/{tenant}`), revoke (`POST /users/{id}/tenants/{tenant}/delete`). The grant select offers only tenants the user does NOT belong to — a membership is added once (the decider refuses a duplicate), so suggesting an existing one would be a dead end; changing it is the row's own control. Unknown roles and unknown tenants are refused before any command is dispatched, and a "Tenants" stat card joins the detail header. Pinned by `user_tenant_access_test.go` (render, empty state, grant, role replacement, bad-input refusal, revoke) and the route-manifest golden.

- **The tenant roster names people, not actor tokens.** Member rows led with `user:01HZ…`, an identifier an admin cannot map to anybody; resolving it meant copying the id into the URL by hand. Rows now resolve the membership actor through the directory (`identitymodel.ActorIDAsUserID` → user read model) and render an avatar, the email (linked to the user page in super-admin mode), the display name, and when the membership started; the role cell shows the roles actually held as badges next to the role control. Bot/system/service actors keep their prefixed id (there is no directory entry to resolve). Tenant-admin mode renders no user links because that mode serves no user directory. Pinned by `tenant_ux_test.go`.

- **Tenant detail leads with the tenant's identity.** A new **Tenant** card answers "which tenant is this" without inferring it from the members below: copyable tenant id, immutable slug, display name, live status, and roster size.

- **Tenant roster search + per-tenant member counts.** The tenants list gained the same live HTMX search as the user list (slug or display name, 350 ms debounce, honest empty state, pagination that preserves `q`) and a Members column, so the overview answers "who is in what" before opening anything.

- **User list access column + Administrators chip.** Each row now names the platform (wildcard-domain) roles a user holds and how many tenants they belong to. The count only includes domains that resolve to a real tenant — registration grants default roles in a domain named after the user's own stream, and a raw domain count would report a phantom tenant for every user. A fifth filter chip narrows the directory to platform-role holders, composable with search; the user id joins email and display name as a search term (ids are usually pasted from logs or tickets).

### Changed

- **Add-member UX: pick a person, choose a role, hear the miss inline.** The email field autocompletes from the directory (`<datalist>`, zero JS, CSP-safe) and no longer suggests existing members; the role select carries a disabled "Choose role…" prompt so a distracted Enter cannot grant the first role; and an unknown email no longer vanishes into a transient toast — the redirect carries the typed address back (`member_error=unknown&member_email=…`), the card explains it inline, preserves the input, and (super-admin) links straight to user creation.

### Fixed

- **The tenant creation form has real labels.** The name and display-name fields carried only placeholders, leaving both fields unlabeled to assistive tech and ambiguous to a first-time admin; they now carry labels and help text (the slug is immutable and URL-safe, the display name is optional).

## [v4.17.0] - 2026-11-07

### Added

- **`Config.SSEHubStats`: the dashboard now shows live SSE hub health (fleet-plan A2 adminui leg, 2026-11-06).** The A2 drop-observability wave reached dashboardui's overview card but not the admin panel — the operator most likely to act on "the hub is dropping events" saw nothing. `SSEHubStatsSource` (`SubscriberCount() int`, `DroppedEvents() int64`) is a structural interface; when wired via `Config.SSEHubStats`, the dashboard stats region (and its 30s polled partial) gains two cards: "SSE live" (green) and "Dropped events" (yellow attention tone when nonzero — slow-consumer drops are silent data loss). Nil (default) adds nothing; `setup` wires its adapter automatically when the shared SSE/DataStar hub is enabled. Pinned by `hubstats_test.go` (cards + values + yellow tone on both the page and the polled partial, nil-absent by default, zero-drops stays green) and setup-side by `setup/sse_hub_stats_test.go` (live counts through a real flooded hub + hubless zero-safety). Also in this change: the module's four remaining deprecated root re-export call sites were repointed to the `htmx` subpackage (`ScriptHandler`, `IsRequest`, `RenderPartial` ×2) — ADR-0062 direct-imports, SA1019 clean.

- **The full user lifecycle (superbification waves 1–3, 2026-11-06).** The panel could list and delete users, but ~25 exported `Service` capabilities had no panel surface. New routes on the user detail page: create a user (`GET /users/new` + `POST /users` — the auto-login session is discarded honestly, and the form carries the passwordless-hint note), edit display name (`POST /users/{id}/display-name`) and sign-in email (`POST /users/{id}/email`), grant/revoke platform roles (`POST /users/{id}/roles`, `/roles/revoke` — always through the durable event-sourced `GrantRoles`/`RevokeRoles` path per ADR-0057, never `AddGroupPolicy`), delete an individual credential (`POST /users/{id}/credentials/{credential}/delete`, base64url path segment), and resend verification (`POST /users/{id}/resend-verification`, with a graceful not-configured error path). The platform-access card pre-sets checkboxes from the current wildcard roles and warns on self-lockout when the target is the session user; the new Passkeys card (name / added / backup-state badges) escalates its confirm dialog to the blast radius — the last passkey with no other provider reads "locked out entirely". Each new write escalates its confirmation dialog to match. Pinned by `user_lifecycle_test.go` (14 tests) and the route-manifest golden rows.

- **Audit superpowers (superbification wave 4, 2026-11-06).** `audit_labels.go` is the one display source of truth mapping every recorded action to a human label + badge tone (destructive=error, refused login=warning, sessions=quiet; unknown actions fall back to the raw code, degrading honestly), reused by the audit page, the dashboard's recent activity, and the new per-user **Activity timeline card** on user detail (≤8 recent events, "View all N" deep-links into the filtered audit page). The audit page gains an omnibox filter (action/email/aggregate/label substring) with an HTMX partial swap and honest filtered pagination; `login_failed` entries render their `Detail`; timestamps show absolute values on hover. Riding along: every identity use of `AuditEntry.AggregateID` now goes through `.Get()` — `String()` is the StreamMarker display form and had produced wrong hrefs/matches in four sites (dashboard link, audit table, timeline match, filter). Pinned by `audit_superpowers_test.go` (7 tests).

- **Dashboard enrichment (superbification wave 5, 2026-11-06).** Adoption stats — verified-email %, MFA %, suspended-tenant count — are computed, not decorative ("—" for empty sets); recent-activity rows now link to the entity they mention; and a no-admin-grant warning banner (mirroring `setup`'s boot diagnostic) appears while nobody holds the admin grant and disappears once one exists — proven by test.

- **Users-list filter chips (superbification wave 6, 2026-11-06).** Status filter chips — Verified / Unverified / MFA on / has external provider — with `aria-pressed` state, composable with the existing search (chips preserve `q` in both toggle directions; unknown filter values are refused), plus linked-provider badges in the status column. Pinned by `users_filter_test.go` (4 tests).

- **`Config.ExtraNav` + `Config.DashboardURL` (superbification wave 7, composability, 2026-11-06).** `ExtraNav []NavEntry{Label, Href, Icon}` appends consumer-owned entries after the built-in sidebar items — hrefs render VERBATIM (no `BasePath` prefix), so sibling surfaces (`/dashboard`, `https://docs.example.com`) are one click from the panel; empty `Label` or `Href` is rejected at `New` (a sidebar link with no text or destination is a silent lie). `DashboardURL` cross-links the sibling dashboardui panel as an "Event store" entry; empty hides it, and for more than one link prefer `ExtraNav`. `setup` passes both through as `AdminExtraNav`/`AdminDashboardURL`. Pinned by `extra_nav_test.go` (validation, verbatim rendering, absent-when-unset).

### Fixed

- **The user-menu dropdown anchors correctly again — e2e 76/76 (2026-11-05, round-46 M12 tail).** Since templ-components v1.27.0, the library's popover JS defers to a declarative CSS anchor layer on engines that support it — but that layer is broken on Chromium 156 (compound `position-area` values are rejected outright, and `anchor()` cannot see the implicit `popovertarget` invoker), so the open menu fell back to its static position at the viewport corner. `adminui/tailwind.css` now carries a corrected anchor layer: single-keyword `position-area` bands per `data-tc-position` plus an explicit `anchor-name`/`position-anchor` pair for the user menu — stylesheet-declared, so strict CSP holds. Full `nix run .#e2e` suite 76/76 green; root cause and empirical probes receipted in the root `CHANGELOG.md` `[v4.17.0]` section.

### Changed

- **Dependency/family-train rides (2026-11-07 train):** root v4.16.0 → v4.17.0, usermgmt v4.18.0 → v4.20.0, identity-model v4.15.0 → v4.17.0. templ-components stays v1.27.0 (no library change this tag).

## [v4.16.0] - 2026-11-05

### Added

- **Route table as the single registration source + `Handler.Routes()` posture manifest (authorization L4, 2026-11-03).** The imperative `routes()` mux wiring is now a declarative table: mux registration and the new manifest generate from the same rows, so a route outside the manifest cannot be served and a manifest row cannot describe a route that does not exist. Mode-scoped rows (users/tenants vs members) are absent in the other mode, and a row that forgets its mode is served by NO mode — absence fails loudly at runtime, silent over-exposure fails only in an audit. `Route{Method, Pattern, Panel, Write, Posture, Why}` reports each row's posture per instance: assets public, guarded rows role-gated by default. `Config.AuthorizerPosture` declares what a custom authorizer enforces (validated at `New`: only empty/authenticated/role — the guard always 401s anonymous callers, so public and ungated would be lies); set it to `cqrshtmx.PostureAuthenticated` when the authorizer is `RequireAuthenticated()`-shaped. Pinned by `TestRoutes_ManifestGolden` (super-admin), `TestRoutes_TenantModeManifest`, `TestRoutes_AuthorizerPosture`, and `TestRoutes_FallbackSemantics` (the guarded 404 catch-all, the methodless 405, the exact-match dashboard root, and the public assets — the conversions the table refactor risked). Cross-receipt: root `CHANGELOG.md` `[v4.16.0]` (route postures, authorization L4).

### Changed

- **Dependency/codegen alignment (2026-11-05 train):** templ-components v1.26.0 → v1.27.0, root v4.15.0 → v4.16.0, httputil v1.7.0 → v1.8.0, templ v0.3.1070 → v0.3.1100. All `_templ.go` files regenerated with the v0.3.1100 generator (dead `GetChildren` preamble removed, module-relative file names in runtime render errors, source re-indentation only — no markup change).

## [v4.13.0] - 2026-11-02

### Fixed

- **The third forbidden code is gone: `adminui.forbidden` is now identity-model's sentinel (authorization level-up, 2026-11-02).** The panel re-coded a denial under its own error code, so a consumer could not recognise an authorization failure the panel produced without importing adminui. It now returns the shared `ErrForbidden`, which makes `cqrshtmx.IsForbidden(err)` true for a panel denial — the code-based, chain-walking check that works across every module. Behaviour is otherwise unchanged (still 403); only the error identity moved.

- **`Config.AccentColor` is validated at `New` against the CSS-color grammar (S36, 2026-10-25).** The accent is spliced into a raw `<style>` block (`:root{--accent:...}`), which HTML escaping cannot sanitize — a value like `red;} body{background:url(x)}` injected page rules. `New` now rejects anything outside #hex (3/4/6/8 digits), numeric `rgb()/rgba()/hsl()/hsla()`, and named CSS colors, mirroring dashboardui's grammar exactly (fail-fast chosen over sanitize-to-default: a typo'd accent should surface at construction, not silently render indigo). Pinned by `TestConfig_AccentColor_RejectsAtNew`/`AcceptsGrammar`/`DefaultStillValid`. Breaking only for configs that were injecting CSS; every legitimate color passes unchanged.

- **Failed HTMX partial renders answer 500 instead of a truncated 200 (S36, 2026-10-25).** `renderPartial` logged render errors but left the response a 200 with a half-written fragment — htmx treats 200 as success and swaps the broken fragment into the DOM silently. The error path now mirrors `renderPage`'s explicit 500. Pinned by `TestRenderPartial_RenderError` (updated from the stale contract that asserted the bug).

### Changed

- **Dependency/family-train rides (2026-11-02 wave):** templ-components v1.23.0 → v1.26.0, root v4.14.1 → v4.15.0, usermgmt v4.14.3 → v4.18.0, identity-model v4.12.3 → v4.15.0, go-error-family v0.11.0 → v0.12.0. The two S36 fixes above were coded 2026-10-25 but ship receipt-less in this tag — no adminui tag was cut between v4.12.4 (2026-10-13) and this train; these CHANGELOG entries are the retroactive receipts.

## [v4.12.4] - 2026-10-11

### Changed

- **Dependency/codegen alignment (2026-10-11 wave):** templ-components v1.21.0, root v4.14.x, `_templ.go` regenerated to canonical module-dir form, and the 1.27.2 toolchain floor. The asset-dedup and content-ETag receipts above are retroactive notes for work that shipped inside v4.12.3 (receipt-less there); no behavior change in this tag beyond the dependency rides.

### Fixed

- **Asset serving delegates to the root module's `cqrshtmx.ServeAsset` (2026-10-07, art-dupl dedup round):** `/-/admin-tw.css` and `/-/admin.js` keep the immutable-asset posture (content-derived ETag, 1-year immutable caching, nosniff) but the read/serve/ETag plumbing now lives once in the root module (`cqrshtmx.ServeAsset`/`cqrshtmx.AssetETag`/`cqrshtmx.AssetFromFS`) instead of being duplicated with dashboardui — the posture can no longer drift between the two UI modules; only the `embed.FS` declaration and config-error wrapping stay local (a missing embedded asset now surfaces as a wrapped Infrastructure-family error). ETag values change format from `"adminui-<name>-<16-hex>"` to `"<name>-<unpadded-hex>"`; consumers see one revalidation, then behavior is identical. Pinned by the same 304 round-trip tests plus root's header/conditional-GET/read contract tests.

- **Asset ETags are content-derived and cache headers immutable (dashboardui M16 sweep):** `/-/admin-tw.css` and `/-/admin.js` now carry ETags computed from their own bytes (FNV-1a, `"adminui-<name>-<hash>"`) instead of the hand-bumped `adminui-v3.4.0` constant — the class where the compiled bundle changes but the version const does not, silently diverging every consumer's cached copy. `Cache-Control` moves from `public, max-age=86400` to `public, max-age=31536000, immutable` (the root module's `HTMXScriptHandler` posture). Pinned by a round-trip 304 test and the content-hash assertion in `assets_test.go`; mirrors dashboardui's `assets.go` so both UI modules follow one rule.

## [v4.12.1] – [v4.12.3] — 2026-10-01 → 2026-10-07

- Dependency-train riders only (root v4.13.0, usermgmt v4.13.x → v4.14.1, identity-model v4.12.1, go-cqrs-lite storage v4.10.2, and the workspace-wide sweep to current published versions), plus the canonical treefmt rewrap absorbed. No adminui-specific feature entries were recorded for these tags; the v4.12.4 section above carries the retroactive receipts for the v4.12.3 asset-dedup and content-ETag work, and the historical detail lives in the root CHANGELOG sections for the same versions.

## [v4.12.0] - 2026-09-22

### Changed

- **Rode the v4.12.0 family train** (release commit `90514c5a`, alongside loginpage + systemadapter), and the compiled Tailwind bundles moved to the flake CSS builders' canonical output — minified, committed as produced (`b0f265b6`), the shape `check-css-bundles` enforces today.

## [v4.11.0] - 2026-09-19

### Added

- **templ-components core-shell migration (2026-09-17→09-19):** the hand-rolled layout shell is retired — `Layout` is now `layout.Base` + `layout.AppShell` + `navigation.SidebarNav` (the drawer keeps the legacy `.admin-sidebar`/`.admin-toggle` class names so `admin.js` is unchanged), and every `@Layout(p)` wrapper call site became a `content templ.Component` parameter (templ 0.3.1020 has no children-as-value). Verified across desktop/dark/mobile-390/tablet-768 with a browser run against a throwaway demo built from HEAD.
- **Structured error pages:** new published require `github.com/larsartmann/templ-components/errorpage v1.18.0`; `Handler.writeErrorPage` renders the `ErrorPage` card (bare for HTMX swaps, inside `Layout` for navigations, HTTP statuses preserved) and a guarded 404 catch-all serves `NotFound404` — every bare `http.Error`/`http.NotFound` in the panel handlers is replaced (render.go's post-header fallback stays, deliberate). 4 new errorpage tests.
- **Audit "Who" columns resolve actor emails** via the shared `resolveAuditEmails` read-model helper (dashboard + audit page).

### Fixed

- **Sidebar transparency + dark-mode gray inversion (browser-evidence verification):** (1) SidebarNav's `bg-[var(--tc-sidebar-bg)]` needs the variable defined — adminui's compiled bundle has no library `:root` defaults, so `.admin-shell` sets it alongside the CSP-safe `--tc-sidebar-w` (AppShell's inline style attribute is dropped under the CSP nonce posture); (2) adminui's `@theme` maps gray-800/900 to `--text`, which flips light in dark mode, so the library's `dark:` dark surfaces (e.g. `thead dark:bg-gray-800`) painted as light bands — the dark-mode media block pins those two tokens to the literal dark palette (safe: all 115 library `text-gray-800/900` occurrences carry a `dark:` override). `build-adminui-css` now also scans the errorpage module dir including its `styles.go`.

## [v4.3.0] – [v4.6.1] — 2026-07-12 → 2026-07-27

- Coordinated lockstep bumps riding the monorepo release trains of the era (root/identity-model dependency updates, replace-directive release-prep strips, repo-wide chores). No adminui-specific feature entries were recorded in this file for those tags; the historical detail lives in the root `CHANGELOG.md` sections for the same versions. (Reconstructed 2026-11-08, M15/F66 — the tags predate per-module CHANGELOG discipline.)

## [v4.8.0] – [v4.10.0] — 2026-08-14 → 2026-09-07

- Coordinated lockstep bumps riding the family release trains (root v4.8.0–v4.10.0, usermgmt, templ-components v1.18.0 CSS-bundle era). No adminui-specific feature entries were recorded in this file for those trains; the historical detail lives in the root `CHANGELOG.md` sections for those versions.

## [v4.7.0] - 2026-08-07

### Added

- CSP nonce support for inline scripts.
- HTMX partial rendering support via `RenderPartialOrFull`.
- `feedback.ToastContainer` adoption (bridges `adminui:toast` HX-Trigger events).
- `htmx.GlobalErrorHandling` adoption (5xx retry, network error toast, session-expiry redirect).
- Offline sync served from root module (`sync-worker.js`, `sync-client.js`).

## [v4.6.0] - 2026-07-26

### Changed

- **Dependency bump**: templ-components `v1.1.0` → `v1.2.0`.
- **Dedup sweep**: `ToastDetail` extracted to root module (`cqrshtmx.ToastDetail`); adminui now re-exports the shared type. See root CHANGELOG `[v4.6.0]` for the full dedup sweep summary.

## [v4.5.0] - 2026-07-24

### Changed

- **Explicit dependencies on root + usermgmt**: go.mod now directly requires `cqrs-htmx/v4` and `cqrs-htmx/usermgmt/v4` (previously transitive). All transitive dependencies materialized in go.sum.
- **go-cqrs-lite v4.0.x dependency alignment**: Updated all go-cqrs-lite module references.

## [v4.2.0] - 2026-07-08

### Changed

- **go-cqrs-lite v3.5.0 → v3.7.4**: Aligned with root and usermgmt dependency upgrades.
- **go-error-family direct import**: Migrated from transitive dependency to direct import.
- Refreshed templ generated output to match latest CLI version.

### Fixed

- **go.work replace+use conflict**: Removed `replace` directive from go.work that conflicted with BuildFlow's `use` directive. Per-module go.mod files retain their own `replace` for GOWORK=off compatibility.

## [v4.1.1] - 2026-07-04

### Changed

- **httputil v0.3.0 → v0.4.0**: Transitive dependency bump. No API or behavior change.
- **templ-components v0.6.0 → v0.6.1**: Minor dependency bump.
- **HTMX v2.0.9 → v2.0.10**: Updated embedded HTMX JS. Extracted `serveJS` helper for shared caching logic between HTMX core and extensions.

## [v4.0.1] - 2026-07-02

### Fixed

- Normalized templ generated import style for BuildFlow compatibility.
- Corrected pseudo-versions in go.mod from `v0.0.0` to valid `v4.0.0` paths.

## [v4.0.0] - 2026-07-02

### Changed — v4 Module Path Bump

- **BREAKING**: Module path changed from `github.com/larsartmann/cqrs-htmx/adminui/v3` to `github.com/larsartmann/cqrs-htmx/adminui/v4`.
- Updated dependency on root `cqrs-htmx/v4` and `usermgmt/v4`.

## [v3.5.0] - 2026-07-01

### Changed

- Aligned with root v3.5.0 release (go-cqrs-lite v3.5.0).

## [v3.0.0] - 2026-06-27

### Added

- **First release of adminui module.** A ready-made, good-looking Admin Dashboard for usermgmt-backed apps.
- One-call mount behind session middleware via `New()` + `Mount()` / `Handler()`.
- Two scopes: `ModeSuperAdmin` (global dashboard/users/tenants/audit) and `ModeTenantAdmin` (tenant-scoped members/audit).
- Auth-agnostic: reads `*usermgmt.User` from context (consumer's session middleware).
- Modern CSS design system (light/dark via `prefers-color-scheme`, accent color injection).
- HTMX patterns: live search, `hx-confirm` destructive actions, `HX-Redirect`, toast notifications via `HX-Trigger`.
- Embedded assets via `go:embed` — no build step for consumers.
- Depends on root `cqrs-htmx/v3` (reuses `HTMXScriptHandler`) + `usermgmt/v3` + `a-h/templ`.
