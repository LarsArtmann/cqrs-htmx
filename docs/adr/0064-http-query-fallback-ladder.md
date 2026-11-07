# ADR 0064: The HTTP QUERY Fallback Ladder as a Library Contract

**Status:** Accepted
**Date:** 2026-11-07
**Related:** RFC 10008 (The HTTP QUERY Method), [ADR 0055](0055-owner-session-gated-credential-ceremonies.md) (middleware placement as a security decision), [ADR 0062](0062-direct-imports-only.md) (additive-only in v4), plan of record [`2026-11-07_07-21_SUPERB-http-query-method-pareto-plan.md`](../planning/2026-11-07_07-21_SUPERB-http-query-method-pareto-plan.md), guide [`docs/guides/http-query-method.md`](../guides/http-query-method.md)

## Context

RFC 10008 standardizes `QUERY`: a request method that is safe, idempotent, and
carries a request body — exactly the CQRS read side. Before it, every complex
read was forced into one of two lies: `POST` (loses safety, retryability, and
cacheability, and drags CSRF along) or URL-encoded `GET` (length and encoding
limits; filters leak into logs, proxies, and browser history).

The server side already worked by accident: `App.Query`/`QueryTyped` never
inspect `r.Method`, the JSON decoders read the body regardless of verb, and Go's
stdlib `ServeMux` registers `QUERY /path` patterns on stock Go 1.27. The work
was making the story honest and complete: advertisement (`Accept-Query`),
hostile-infrastructure survival (the fallback ladder), spec-truth (OpenAPI 3.2),
and conditional requests (`ETagResults`).

The hard part is not the native method — it is that real deployments sit behind
proxies, WAFs, and corporate filters that reject methods they do not know. A
library that only speaks native QUERY is unusable exactly where production runs.

## Decision

1. **One handler serves the whole ladder.** A single `QUERY /contacts` mux
   pattern plus `MethodOverrideMiddleware` serves the native method AND the
   fallback transport from one registration. The consumer never writes a
   twin handler for the fallback — byte-parity is structural, not tested-in
   (and where it IS tested, `integration_test/query_method_test.go` pins it).

2. **The ladder has three rungs, in order of preference:**
   native `QUERY` → `POST` + `X-HTTP-Method-Override: QUERY` → `GET` with
   URL parameters (for small queries, consumer's choice). The library ships
   rungs one and two; rung three is a documented recipe, not code — a
   polyfill that rewrites QUERY into GET-with-params would re-create the
   URL-length problem QUERY exists to solve.

3. **QUERY is the ONLY override target, by design.** The middleware exists
   for safe read fallbacks. An override to `DELETE` (or any unsafe verb) is
   ignored and the request stays a POST. Method smuggling through a
   read-fallback seam is a non-feature.

4. **The middleware mounts INSIDE the security stack** (it wraps the mux, so
   session/CSRF/rate-limit layers see the wire method). The fallback POST is
   token-checked as a POST today and forever — we never bypass CSRF to "make
   QUERY work", and we never couple fallback auth to how any CSRF
   implementation's safe-method list treats QUERY later. An outermost
   placement was rejected: it would validate the *intended* method instead of
   the *wire* method, silently changing what the security layers believe they
   are authorizing.

5. **Advertisement is decoder-driven.** `DecodeJSONQuery`/`DecodeJSONQueryTyped`
   set `Accept-Query: application/json` themselves — including on error
   responses, so discovery works before the first success. Form decoders
   advertise nothing. The component that knows what it accepts is the
   component that speaks; `AcceptQuery(...)` only appends. The advertisement
   cannot drift from the implementation because the implementation writes it.

6. **Spec-truth over spec-compat.** A `query` operation renders in an
   OpenAPI 3.2.0 document or not at all: `openapi.Query()` auto-bumps the
   spec version on registration, and `Spec.Validate` rejects a query
   operation riding any other version. A 3.1 document carrying a 3.2 field
   lies to every toolchain that reads it.

7. **Divergences are documented, not hidden.** The content-type default is
   deliberately lenient (a body-carrying request without `Content-Type` is
   decoded, not rejected) — `RequireContentType("application/json")` opts a
   route into the RFC 10008 media-type ladder (missing type + body → 400,
   wrong type → 415). Validation failures stay 400 in v4 (422 is a v5
   decision, not a quiet break).

## Consequences

- Consumers behind hostile infrastructure ship QUERY today: the fallback POST
  rides every proxy ever shipped, and the upgrade to native QUERY is a
  client-side change (drop the override header) with zero server edits.
- The ladder is additive v4 API (`MethodOverrideMiddleware`,
  `RequireAnyMethod`, `RequireContentType`, `AcceptQuery`, `openapi.Query`,
  `ETagResults`); nothing existing changes behavior, and the v5 window owns
  the default flips (htmx 4 default, 422) already recorded in the plan.
- huma-based APIs cannot serve QUERY until upstream huma grows a `query` Path
  Item case (verified: v2.40.0 panics `unknown method QUERY`); the verdict is
  pinned by test so the day it changes, the docs change with it.
- The upstream legs stay open by design: httputil's CSRF safe-list should
  eventually treat QUERY as spec-safe (an upstream ask, not a local hack),
  and the proxy-compatibility table in the guide lists only verified rows.
