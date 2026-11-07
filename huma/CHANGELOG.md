# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [v4.0.0] - 2026-11-07

First tagged release of the optional Huma v2 adapter (demand signal [cqrs-htmx#41](https://github.com/LarsArtmann/cqrs-htmx/issues/41), built 2026-11-06).

### Added

- **`cqrsadapter.New(Config)` — the same CQRS pipeline inside typed OpenAPI operations.** `RegisterCommand(api, pipe, op, build, opts...)` and `RegisterQuery(api, pipe, op, build, respond, opts...)` run the identical stage order the route builder runs — authorize → build → validate → guard → dispatch — with Huma owning decode, input validation, status codes, and schema generation. An anonymous caller answers 401, a policy denial 403, a validator failure 400 with code `cqrshtmx.validate.failed`, a conflict 409, and a broken policy engine 503 (never a silent denial). `Config` takes your command/query dispatchers plus a `cqrshtmx.PermissionChecker` — `identitymodel.Authz` satisfies it structurally, so the adapter composes published root API (`Decide`/`UserIDFromContext`/`CommandOptionsFromContext`) without a root change.
- **The shared RFC 9457 problem document.** `ProblemError` embeds `httperr.StructuredError` and renders `application/problem+json` with the same `code` vocabulary the route-based handlers emit — clients branch on one error shape across both transports. `MapError(ctx, err)` is exported for raw handlers.
- **Pipeline options:** `Authorize(resource, action)`, `AuthorizeIn(domain, resource, action)`, `AuthorizeOn[I](resolver)` (derive the whole authorization question from the typed input), `RequireAuth()`, `ValidateCommand(fn)`, `Guard(fn)`, `WithTimeout(d)`. Exported stages `Authorize`/`DispatchCommand`/`DispatchQuery` let raw Huma handlers reuse pieces.
- **Documented deliberate divergences** (see `README.md` § What the adapter does not do): no decode/render/HTMX options (Huma owns those) and no `BeforeDispatch`/`AfterDispatch` hooks (they take an `*http.Request` a Huma handler never sees — use Huma middleware).
- **QUERY method verdict pinned.** Huma v2.40.0 has no `query` Path Item case and panics on `unknown method QUERY`; `query_method_test.go` pins the verdict and turns red the day upstream ships support (ROADMAP question 35c).
