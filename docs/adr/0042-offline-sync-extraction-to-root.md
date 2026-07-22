# ADR-0042: Extract Offline Sync from adminui to Root Module

## Status

ACCEPTED — 2026-07-22

## Context

ADR-0029 shipped Phase 2a (SharedWorker for offline command sync) and ADR-0040
shipped Phase 2b (IndexedDB persistence). Both lived entirely inside
`adminui/assets/` — `sync-worker.js` was embedded via `adminui/assets.go`, and
all tab-side sync logic (SSE connection, SharedWorker coordination, HTMX event
listeners, sync indicator, retry button) was mixed into `admin.js` (~400 lines).

This created two problems:

1. **Consumer lock-in.** Any consumer who wanted offline command sync — not
   just adminui users — had no way to access the SharedWorker or sync client.
   The offline sync stack was trapped behind an admin dashboard dependency.
   A consumer building their own UI (loginpage, custom dashboard, HTMX partial
   app) could not use the queue-with-IndexedDB-persistence feature without
   pulling in the entire adminui module.

2. **Concern mixing in admin.js.** The 400-line `admin.js` file mixed four
   distinct concerns: CSRF token injection, mobile sidebar toggle, toast
   notifications / confirm dialogs, and the entire offline sync stack (SSE,
   SharedWorker, HTMX events, indicator, retry). The sync code had zero
   cross-references with the admin code — they were already logically separate,
   just physically collocated.

The extraction boundary was clean: admin code (CSRF, sidebar, toast, confirm)
never referenced sync variables. Sync code never referenced admin variables.
The only shared call site was `boot()` which called both `connectSSE()` and
`initSyncWorker()` — both sync functions that moved together.

## Decision

**Move the offline sync stack from adminui to the root module, following the
same embedded-JS-handler pattern as `HTMXScriptHandler()` and
`HTMXExtensionHandler()`.**

Concrete changes:

- `sync/sync-worker.js` — Production-hardened SharedWorker moved to root.
  IndexedDB as single source of truth, hello/bye tab protocol with tabId-based
  port Map, retry count (MAX_RETRIES=10) + TTL (24h) eviction, staggered and
  targeted retry delivery, null-envelope guard, in-memory fallback when IDB
  unavailable, flush dedup, `store.count()` for pending count.

- `sync/sync-client.js` — Tab-side client extracted from admin.js (~390 lines).
  SSE connection, sync indicator, HTMX event listeners (beforeRequest,
  sendError, responseError), SharedWorker coordination, `rebuildAndRetry`,
  retry button handler. Auto-initializes on DOMContentLoaded if
  `<body data-sse-url>` is present.

- `sync_embed.go` — `//go:embed sync/sync-worker.js` and
  `//go:embed sync/sync-client.js`, `syncVersion = "1.0.0"` constant.

- `sync_serve.go` — `SyncWorkerHandler()`, `SyncClientHandler()`,
  `SyncClientScriptTag(path)`, `SyncVersion()`. All use the existing
  `serveJS()` helper (Content-Type, Cache-Control immutable, ETag, 304-on-match,
  GET/HEAD only).

- adminui delegates: `adminui/assets.go` provides thin wrappers
  (`syncWorkerHandler()` / `syncClientHandler()`) that call the root handlers.
  `adminui/handler.go` routes `GET /-/sync-worker.js` and `GET /-/sync-client.js`
  through these wrappers. `adminui/layout.templ` conditionally includes
  `<script src=".../-/sync-client.js">` when `SSEURL != ""`.

- `adminui/assets/admin.js` slimmed from ~400 to ~65 lines (CSRF, sidebar,
  toast, confirm only).

- `adminui/assets/sync-worker.js` **deleted** (moved to root).

## Why Root Module (not a Dedicated Module)

A dedicated `sync/` Go module (separate `go.mod`) was considered and rejected.
The sync system is only 2 JS files + 1 Go handler file. It has no Go
dependencies beyond what the root module already imports. It is tightly coupled
to HTMX events and the SSE ACK protocol — both root-module concerns. A separate
module for this would be over-modularization, adding versioning and workspace
complexity for zero isolation benefit.

The root module already serves embedded JavaScript via `HTMXScriptHandler()`,
`HTMXExtensionHandler()`, and `HTMXExtensionsHandler()`. The sync handlers
follow the exact same pattern.

## Why SharedWorker URL Derivation (not a Config Flag)

The sync client (`sync-client.js`) finds its own `<script src>` tag and replaces
"sync-client.js" with "sync-worker.js" to derive the worker URL. This means
both assets must be served under the same base path. This approach was chosen
over a config flag or separate URL parameter because:

1. It requires zero consumer configuration beyond mounting two handlers.
2. It naturally handles base paths and prefixes (adminui mounts at `/-/sync-*`,
   a consumer might mount at `/static/sync-*`).
3. The worker URL is always derivable from the client URL by convention.

## Consequences

- **Positive:** Any consumer — not just adminui users — can wire offline command
  sync with two handler mounts and one `<script>` tag. The feature is no longer
  trapped behind an admin dashboard dependency.

- **Positive:** `admin.js` is now ~65 lines of focused admin UI code. Each file
  has a single responsibility.

- **Positive:** The sync system follows the established embedded-JS-handler
  pattern. Consistency with HTMXScriptHandler/HTMXExtensionHandler reduces
  cognitive load.

- **Positive:** adminui's test surface is smaller (no sync-worker tests needed
  in adminui — they live in the root module).

- **Negative:** The `syncVersion = "1.0.0"` constant is manual. If the JS
  changes without a version bump, clients cache stale code for 1 year (same
  trade-off as HTMXScriptHandler's `htmxVersion` constant). Mitigated by the
  pre-commit hook and code review.

- **Negative:** No browser E2E tests. The `rebuildAndRetry` cross-session path
  and IndexedDB persistence are protocol-tested in Go unit tests but not
  verified in a real browser.

- **Negative:** `navigator.onLine` remains the only connectivity signal.
  Captive portals and DNS failures are undetected. The worker mitigates by
  flushing on enqueue-while-online, but this is imperfect.

## Relationship to ADR-0040

ADR-0040 confined persistence to "optional admin UI JavaScript assets." This ADR
moves those assets to the root module, broadening availability. The architectural
principle from ADR-0040 is preserved: all persistence logic lives in JavaScript
assets, not in Go code. The Go library remains free of client-side persistence
concerns — `SyncWorkerHandler()` merely serves a static JS file.
