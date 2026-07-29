# Dedup Acceptance Log

Accepted clone groups from `art-dupl --type-aware -t 1` that are intentional
boilerplate, not harmful duplication. Each entry lists the clone group
location and a one-line reason for accepting it.

The current state at `-t 5` (the default meaningful threshold) is **0 clone
groups**; the report at `-t 1` lists only short, idiomatic fragments.

## Context-with-cancel boilerplate

- `usermgmt/http.go:219,256`, `usermgmt/oauth2_http.go:29,57,85`,
  `usermgmt/verification_totp_http.go:118` — `ctx, cancel := h.withTimeout(r); defer cancel()`
- `ws_dispatch.go:122,174` — `ctx, cancel := a.timeoutCtx(ctx, nil); defer cancel()`
- `usermgmt/lockout.go:96,112` — `normalized, unlock := l.normalizeAndLock(email); defer unlock()`

**Reason:** `defer cancel()` must execute in the caller's scope. Extracting
into a helper would change the defer lifetime. This is idiomatic Go
resource-cleanup boilerplate, not semantic duplication.

## Generic zero-value returns

- `decoder.go:24,90` — `var out T` in `decodeJSONBody[T]` and `decodeFormBody[T]`

**Reason:** Go generics require each generic function to declare its own zero
`T` for the failure path. Language constraint, not duplication.

## Path-value destructuring in dashboardui detail handlers

- `dashboardui/handlers_aggregates.go:36-41`, `dashboardui/handlers_timetravel.go:64-69` —
  `streamType, streamID := streamPathValues(r)`

**Reason:** Two detail-page handlers consume the same shared helper. The
two-call-site destructuring is the natural result of two consumers needing the
same two-segment path; `streamPathValues` itself is already factored.

## Guard clauses

- `usermgmt/credential_http.go:95`, `usermgmt/oauth2_http.go:76` — extended guard
  with path-value validation
- `usermgmt/verification_totp_http.go:97,130` — guard + defer cancel

**Reason:** Each guard runs a different preflight (`requireUser` vs
`currentUser` vs `authContext`) and writes a different error message. The
`if !ok { return }` wrapper is the minimal caller-side boilerplate; further
extraction would require a handler-shaped helper per call site with no shared
code inside.

## HTTP header setting with different values

- `adminui/render.go:16` — `Content-Type: text/html`, `Cache-Control: no-store`
- `event_catalog_handler.go:73` — `Content-Type: application/json`, `Cache-Control: immutable`

**Reason:** Completely different content types and cache policies across
different modules. The two-line header set is too small to abstract
meaningfully, and the values are intentionally unique.

## Basic error propagation in constructors (post-refactoring)

- `usermgmt/sql_readmodel.go:76,90` — `if err != nil { return nil, err }`
  in `buildSQLUserReadModel`
- `usermgmt/sql_readmodel_extra.go:46,60` — same in buildSQLMembershipReadModel
- `usermgmt/sql_readmodel_extra.go:139,153` — same in buildSQLTenantReadModel
- `usermgmt/sql_readmodel_extra.go:226,240` — same in buildSQLBotReadModel

**Reason:** After extracting `newViewStoreOrFail` (which absorbs the
WrapTransient error-wrapping) and per-aggregate `buildSQLXReadModel` (which
absorbs the SQLite/SQL pair), the remaining `if err != nil { return nil, err }`
is basic Go error propagation. Extracting further would require a generic
constructor with 7+ parameters for 3 lines of saved code.

## Guard-method call sites (post-refactoring)

- `dashboardui/handlers_dlq.go:43,63,82` — `if !d.requireProjectionHost/DeadLetterStore(w) { return }`
- `dashboardui/handlers_projections.go:60` — `if !d.requireProjectionHost(w) { return }`

**Reason:** These 3-line guard calls replaced the original 6-line inline
nil-check blocks. The duplication is now minimal (calling the same guard
method) and each handler has different logic after the guard.

## Setup file `_ = bundle.Close(); return nil, err` blocks

- `usermgmt/postgres_setup.go:56,62`, `usermgmt/sqlite_setup.go:58,64`

**Reason:** Both files carry `//go:build ignore` and are reference templates
consumers copy to wire their own SQL backend. The duplication is deliberate
to keep each file self-contained as a copy-and-customize starting point.
Production code is factored into `es_setup_core.go`.

## Refactors applied this session

- `dashboardui`: extracted `renderStreamIndex(title, basePath, render)` shared
  by `aggregatesIndexHandler`, `snapshotsIndexHandler`, `timeTravelIndexHandler`.
- `adminui`: extracted templ `stateBadge(cond, labelIf, kindIf, labelElse,
  kindElse)` shared by `verifiedBadge` and `totpBadge`.
- `usermgmt`: extracted `buildSQLUserReadModel`, `buildSQLMembershipReadModel`,
  `buildSQLTenantReadModel`, `buildSQLBotReadModel` (each backing the
  SQLite/generic SQL constructor pair).
- `usermgmt`: extracted `Service.dispatchUserCommand(ctx, userID, build, kv...)`
  for the 5-line `aggIDFromUser` + `Dispatch` + `classifyDispatchError` boilerplate
  in `ChangeEmail`, `ChangeDisplayName`, `DeleteUser`, `AddCredential`,
  `RemoveCredential`.

## Verification

- `art-dupl --type-aware -t 5` → 0 clone groups.
- `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race` → all 7+ tested modules pass
  (root, openapi, usermgmt, dashboardui, adminui, webauthn, totp, oauth2).