# cqrs-htmx

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.** Consumers import this package into THEIR projects.

Go library that makes it easy to use go-cqrs-lite with HTMX, templ, and Casbin authorization. Framework-agnostic — works with `net/http`, Gin, Chi, etc.

## Quick Reference

| Item     | Value                                                                     |
| -------- | ------------------------------------------------------------------------- |
| Language | Go 1.26.4 (`GOEXPERIMENT=jsonv2` required)                                |
| Module   | `github.com/larsartmann/cqrs-htmx/v4`                                     |
| Test     | `nix run .#test` or `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race`    |
| Build    | `nix run .#build` or `GOEXPERIMENT=jsonv2 go build ./...`                 |
| Lint     | `nix run .#lint` or `GOEXPERIMENT=jsonv2 golangci-lint run`               |
| Coverage | `nix run .#coverage` / `nix run .#coverage-gate` (root 90%, usermgmt 74%) |
| Fmt      | `nix fmt`                                                                 |
| DevShell | `nix develop`                                                             |

## Architecture

**Multi-module Go workspace.** 12 independent Go modules under one `go.work`:

- **Root** (`cqrs-htmx/v4`): Core library — HTTP handler builder, HTMX/SSE/WS helpers, authz (Casbin), CSRF, rate limiting, security headers, error mapping, pagination, `openapi/` sub-package (dependency-free OpenAPI 3.1 spec builder + `WithOpenAPI`/`OpenAPISpecHandler`)
- **usermgmt** (`usermgmt/v4`): Event-sourced CQRS user management (aggregates: User, Membership, Tenant, Bot). Passwordless — auth via WebAuthn/TOTP/OAuth2 behind interfaces
- **usermgmt/totp**, **usermgmt/webauthn**, **usermgmt/oauth2**: Independent auth strategy modules — satisfy `usermgmt` interfaces via structural typing
- **adminui** (`adminui/v4`): Ready-made admin dashboard (templ + HTMX)
- **loginpage** (`loginpage/v4`): Ready-made passwordless login page
- **integration_test**: Cross-module bridge tests
- **examples/**: `basic`, `datastar-demo`, `catalog-demo`, `admin-demo`

**Dependency direction:** Root → usermgmt is zero imports (clean boundary). Auth strategies → root/usermgmt via interfaces only. adminui/loginpage → root + usermgmt. Nothing depends on adminui or loginpage.

**Key dependencies:** go-cqrs-lite v4 (CQRS/event sourcing), casbin/v3 (authz), justinas/nosurf (CSRF), go-error-family (error classification), go-branded-id (typed IDs), a-h/templ (HTML templating), ginkgo/gomega (BDD testing).

## Key Patterns

- **Library principle:** Never enforce defaults consumers might disagree with (no mandatory CSP/HSTS/CSRF — all opt-in)
- **No stdlib error constructors:** `errors.New`/`fmt.Errorf` banned in non-test code. Use `event.New*/Wrap*/Wrapf/Newf` from go-cqrs-lite.
- **Error families → HTTP status:** Rejection(400), Conflict(409), Transient(503), Corruption(500), Infrastructure(503)
- **Duck-typing for external libs:** `Enforcer` (← casbin), `TemplComponent` (← templ), auth provider interfaces — consumers import what they need
- **authMode enum:** `authNone`/`authRequired`/`authAuthorized` — impossible states unrepresentable
- **templ `_templ.go` committed:** Generated files are committed so consumers run no codegen
- **Coverage gate:** Root ≥90%, usermgmt ≥74% (CI-enforced)

## Gotchas

- **`GOEXPERIMENT=jsonv2` is mandatory:** Build fails without it. Never add depguard rules banning `encoding/json/v2`.
- **`ActorID` differs by module:** Root's is `brandid.ID` (use `NewActorID("...")`). usermgmt's is a kind-discriminated struct (use `NewActorID(kind, raw)`).
- **CSRF middleware ordering:** `Chain(CSRFMiddleware, HTMXMiddleware, app.Middleware())` — CSRF first.
- **`HandlerConfig.Secure` is `*bool`:** nil defaults to true. Use `new(bool)` for false.
- **Registration is email-only:** No password field. Auth is exclusively via WebAuthn passkeys.
- **UserID types are aliased:** Root defines `type UserID = id.UserID` and usermgmt aliases the same type — they interoperate directly, no bridging needed.
- **Module path casing:** go-cqrs-lite uses lowercase `github.com/larsartmann/go-cqrs-lite`.
- **go-cqrs-lite v4.0.0+ publishing bug (STILL ACTIVE):** go.mod files originally referenced internal siblings with zero pseudo-versions. As of 2026-07-18, 13 of ~40 go-cqrs-lite submodule tags STILL have broken zero pseudo-versions (including v4.0.1/v4.0.2). The `go.work` local replaces (see below) are REQUIRED until upstream cuts a clean consolidated release (v4.0.3+ or v4.1.0). Root currently uses: command/v4 v4.0.0, event/v4 v4.0.2, id/v4 v4.0.1, query/v4 v4.0.0, transport/http/v4 v4.0.1.
- **go.work local replaces (STILL REQUIRED):** `go.work` contains `replace` directives pointing all go-cqrs-lite modules to `/home/lars/projects/go-cqrs-lite/*`. These are NOT optional — 13 of ~40 submodule tags still have broken zero pseudo-versions (see comment block in go.work for the full audit). Do NOT remove these replaces until go-cqrs-lite cuts a clean consolidated release (v4.0.3+ or v4.1.0) from a tree where every submodule go.mod has been `go mod tidy`-resolved.
- **cqrs-lint silent-failure on broken builds (fixed in v0.2.1):** cqrs-lint < v0.2.1 silently reports "No Go files importing go-cqrs-lite found" when the project doesn't compile, instead of surfencing the real error. Always ensure `go build ./...` passes before trusting cqrs-lint output. v0.2.1+ exits non-zero and names the load errors.
- **GOWORK=off for submodules:** `go.work` covers workspace; use `GOWORK=off` for per-module go.mod commands.
- **TODO_LIST convention (no `[x]` items):** `TODO_LIST.md` contains ONLY `[ ]` (open) and `[~]` (partially done) items. Completed work goes to `CHANGELOG.md` (append-only, per-version). Rejected/deferred ideas go to `ROADMAP.md` → "Not Planned". Never add `[x]` items to TODO_LIST — they re-introduce a split brain between "what's done" (CHANGELOG) and "what's open" (TODO). If you finish a task during a session, add a CHANGELOG entry, do not mark it `[x]` in TODO.
- **`NewUserID(string)` is deprecated (v4.2.x):** silently SHA-256-hashes non-ULID strings, masking invalid input and producing colliding IDs. Use `ParseUserID` (strict, returns error on non-ULID), `SyntheticUserID` (explicit stable hash for known-arbitrary input), or `GenerateUserID` (root module, fresh ULID). A `// Deprecated:` marker is on the declaration — staticcheck SA1019 flags every call site.
- **`SnapshotConfig` is opt-in (v4.2.x):** zero-value = full-replay mode (zero behavior change for existing consumers). To enable, set `Store` + `Codec` + `Strategy` together on `ServiceConfig.SnapshotConfig` (or `EventSourcedConfig`/SQLite/Postgres setup). `MemorySnapshotStore` is dev/test only. Verified: write path consults the snapshot and loads only the tail via `LoadFromVersion` (no full replay). See ADR-0041.
- **adminui offline queue uses IndexedDB (v4.2.x, Phase 2b / ADR-0040):** `sync-worker.js` persists failed command envelopes to IndexedDB (`cqrshtmx-sync` DB, `commands` store), drains on spawn for cross-session retry, deletes on ACK. Degrades to in-memory when IDB unavailable (private browsing). Protocol: tabs send `{type:"hello",tabId}` on connect and `{type:"bye",tabId}` on `beforeunload`; worker tracks ports by tabId in a Map, replaces stale ports on reconnect, and removes dead ports on `postMessage` failure. IDB is the single source of truth (no parallel in-memory queue). Commands have `retries` count (max 10) and `queuedAt` timestamp (24h TTL) — exceeding either marks the command dead (worker sends `{type:"dead",commandId}` and deletes the record). Retry delivery is staggered (100ms per command, capped 2s) and targeted (originating tab preferred, round-robin fallback) to avoid thundering herd. The `rebuildAndRetry` cross-session path is untested in a real browser — adminui unit tests verify the protocol, not the rendered output.
- **BuildFlow pre-commit hook + `golangci-lint --fix` re-formats staged files:** `.git/hooks/pre-commit` runs `buildflow --build-mode pre-commit --staged-only` then `git add`s the result. Two specific linters silently re-introduce bugs by auto-"fixing" intentional patterns: (1) **`fatcontext`** converts `captured = ctx` (assignment to an outer test var) into `captured := ctx` (a fresh unused local), breaking context-capture assertions; (2) **`dupword`** deletes the repeated `data:` prefix in multi-line SSE wire-format test strings, producing malformed expectations. **Root cause pinned (empirically verified 2026-07-20):** both confirmed via an isolated `golangci-lint run --fix` test. **Permanent fix applied:** the affected lines in `hooks_test.go`, `ws_dispatch_test.go`, and `sse_event_test.go` now carry `//nolint:fatcontext` / `//nolint:dupword` directives (with reasons), so `--fix` leaves them stable. For new code that fights these fixers, add the same `//nolint:<linter> // <reason>` on the line. Fallback workarounds if a nolint is missing: (a) `git commit --no-verify` for such fixes (then re-verify HEAD via tests), (b) `git restore <file>` to undo unwanted fixer reverts, (c) NEVER run `golangci-lint run --fix` repo-wide. Do NOT bypass on commits that don't touch this pattern.
- **`openapi/` sub-package is part of the root module (not a separate Go submodule):** It lives at `github.com/larsartmann/cqrs-htmx/v4/openapi`. Uses builder-pattern partial initialization — the entire package is excluded from exhaustruct in `.golangci.yml`. `WithOpenAPI(op)` stores metadata on `handlerConfig.openapiMeta` but no collector reads it yet (pure documentation). `OpenAPISpecHandler(spec)` returns `(http.HandlerFunc, error)` and serializes eagerly at construction (so serialization errors surface at startup, and the returned handler is immutable and concurrency-safe — no lazy caching, no locks); it serves `/openapi.json` with a stdlib FNV-1a ETag, 1-year immutable cache, and 304-on-`If-None-Match`.
- **`usermgmt.NewExternalAccount(...)` is the only way to construct `ExternalAccount` from outside usermgmt:** The embedded `externalAccountCore` is unexported, so struct literals fail. Use `NewExternalAccount(provider, sub, email, displayName, linkedAt)`. Needed by adminui tests and any consumer building test fixtures.
