# Dedup Acceptance Log

Accepted clone groups from `art-dupl --type-aware -t 2` that are intentional
boilerplate, not harmful duplication. Each entry lists the clone group
location and a one-line reason for accepting it.

## Context-with-cancel boilerplate (Groups 1, 10, 13, 22)

- `usermgmt/http.go:219,256`, `usermgmt/oauth2_http.go:29,57,85`,
  `usermgmt/verification_totp_http.go:118` — `ctx, cancel := h.withTimeout(r); defer cancel()`
- `ws_dispatch.go:122,174` — `ctx, cancel := a.timeoutCtx(ctx, nil); defer cancel()`
- `usermgmt/lockout.go:96,112` — `normalized, unlock := l.normalizeAndLock(email); defer unlock()`

**Reason:** `defer cancel()` must execute in the caller's scope. Extracting
into a helper would change the defer lifetime. This is idiomatic Go
resource-cleanup boilerplate, not semantic duplication.

## Guard clauses (Groups 5, 6, 7, 11, 17, 21, 25)

- `usermgmt/credential_http.go:37,94`, `usermgmt/http.go:269` —
  `user, ok := requireUser(w, r); if !ok { return }`
- `usermgmt/credential_http.go:95`, `usermgmt/oauth2_http.go:76` —
  extended guard with path-value validation
- `dashboardui/handlers.go:321,869` — stream-loading guard
- `usermgmt/webauthn_http.go:52,88` — rate-limited user guard
- `usermgmt/oauth2_http.go:20,41` — rate-limit + provider guard
- `usermgmt/verification_totp_http.go:97,130` — guard + defer cancel
- `usermgmt/service_misc.go:32,45` — classifyDispatchError guard

**Reason:** Standard Go guard clauses. The guard helper (requireUser,
requireUserIDWithWebAuthnRateLimit, etc.) is already extracted. The
`if !ok { return }` wrapper is the minimal caller-side boilerplate.

## Parameterized page setup (Group 3)

- `dashboardui/handlers.go:312,818,984` —
  `p := d.page(title, path, r); listings := d.listStreams(r)`

**Reason:** Each call site uses a different page title and route. The unique
values are parameters, not duplication.

## HTTP header setting with different values (Group 14)

- `adminui/render.go:16` — `Content-Type: text/html`, `Cache-Control: no-store`
- `event_catalog_handler.go:73` — `Content-Type: application/json`, `Cache-Control: immutable`

**Reason:** Completely different content types and cache policies across
different modules. The two-line header set is too small to abstract
meaningfully, and the values are intentionally unique.

## Basic error propagation in constructors (post-refactoring)

- `usermgmt/sql_readmodel.go:76,90` — `if err != nil { return nil, err }`
  in `NewSQLiteUserReadModel` / `NewSQLUserReadModel`
- `usermgmt/sql_readmodel_extra.go:46,60` — same in Membership constructors
- `usermgmt/sql_readmodel_extra.go:139,153` — same in Tenant constructors
- `usermgmt/sql_readmodel_extra.go:226,240` — same in Bot constructors

**Reason:** After extracting `newViewStoreOrFail` (which absorbs the
WrapTransient error-wrapping), the remaining `if err != nil { return nil, err }`
is basic Go error propagation. Extracting further would require a generic
constructor with 7+ parameters for 3 lines of saved code.

## Guard-method call sites (post-refactoring)

- `dashboardui/handlers.go:504,600` — `if !d.requireProjectionHost(w) { return }`
- `dashboardui/handlers.go:620,639` — `if !d.requireDeadLetterStore(w) { return }`

**Reason:** These 3-line guard calls replaced the original 6-line inline
nil-check blocks. The duplication is now minimal (calling the same guard
method) and each handler has different logic after the guard.
