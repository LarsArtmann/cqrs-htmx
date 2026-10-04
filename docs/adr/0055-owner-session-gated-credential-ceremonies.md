# ADR-0055: Owner-Session-Gated Credential Ceremonies

## Status

ACCEPTED — 2026-10-04

## Context

Reported by the Ledger CRM identity-adapter feedback
(`docs/feedback/2026-10-04_crm-identity-adapter-gaps.md`, item 1) and
verified against source: the WebAuthn enrollment ceremonies took their
target user from the request itself — `POST /auth/webauthn/register/begin`
read `user_id` from the JSON body, `register/finish` from the `user_id`
query param — with no session check anywhere in the handler or the service
layer (the service checks existence only). `UserID`s are time-ordered
ULIDs, so an unauthenticated caller could enumerate accounts and enroll a
passkey onto any of them: full account takeover, shipped by every consumer
that mounts the auth routes.

The same audit exposed a second, related defect (feedback item 2, wider
than reported): every `RegisterRoutes` handler that reads the current user
from the request context (`/auth/me`, credentials, TOTP, email
verification send, import/export, OAuth2 unlink) was dead when the mux was
mounted without external session middleware — the handlers assumed a
context nothing populated. The library documented "wrap with
`NewSessionMiddleware`" as the consumer's job, and the ready-made
`setup.Bundle` did not do it either (`Bundle.Middleware()` adds security +
logging, no session).

Constraints on the fix design:

1. **loginpage** (first-party) drives `register → registerBegin →
   registerFinish` in one browser flow (`loginpage/assets/login.js`); its
   fetches send `credentials: "same-origin"`, so any gate must accept the
   cookie that `POST /auth/register` has just set.
2. **First-user bootstrap** must keep working: there is no admin yet who
   could enroll the first credential.
3. **Headless/administrative enrollment** (scripts, provisioning) is a
   legitimate need.
4. The library principle — never enforce defaults consumers might disagree
   with — applies to conveniences, not to authorization invariants.

## Decision

1. **The enrollment ceremonies enforce the owner-match rule.** Without a
   session: **401**. With a session whose user is not the requested
   `user_id`: **403**. Self-targeted enrollment proceeds normally. The
   body/query `user_id` parameters stay (compat, explicitness) but are now
   authorization-checked rather than trusted.
2. **Bootstrap is carried by ordering, not by an opt-out.** `POST
   /auth/register` issues the session cookie in its response; the login
   page's same-origin follow-up requests carry it, so the very first user
   passes the gate. There is **no HTTP-level flag** to disable it.
3. **Headless/admin enrollment uses the service-level API.**
   `Service.BeginRegistration(ctx, userID)` / `FinishRegistration` remain
   open for provisioning code running inside the trust boundary. The gate
   lives in the HTTP layer only.
4. **The session-dependent route subset self-wraps.** `RegisterRoutes`
   wraps its `currentUser`-reading registrations (and the gated
   ceremonies) with an enrich-only session pass built from the same
   service + cookie name the handler already owns. Handlers keep their
   fail-closed 401s; an external `NewSessionMiddleware` remains supported
   and becomes a redundant second enrichment. The dead-route class —
   "context-only handler + bare mount" — is thereby fixed for every
   consumer, including `setup`, with zero consumer action.
5. **setup exports its gates.** `setup.RequireSession` (401, JSON
   convention) and `setup.RequireSessionRedirect(loginURL)` (303, browser
   convention) are now public for consumer-owned session-gated surfaces;
   both mount after the enrich-only `bundle.SessionMiddleware()`.
6. **Rate-limit keys extract the client IP** (`httputil.ClientIP`
   precedence) instead of `RemoteAddr` verbatim, which keyed per TCP
   connection. (Feedback item 5; same auth surface.)

## Consequences

- Unauthenticated enrollment requests that used to return 200 now return
  401; mismatched-target requests return 403. Consumers that scripted the
  HTTP ceremonies with arbitrary user ids must move to the service-level
  API. This is the point.
- Requests with an invalid body but no session get 401 (gate runs before
  decode) — fail-closed ordering, pinned by tests.
- `/auth/me` and friends start working on bare mounts where they
  previously 401'd forever; consumers who wrapped the mux themselves see
  one extra (idempotent) session-store lookup per request on those routes.
- Every session-dependent route registration is mechanically inventoried
  by the dead-route invariant gate (`scripts/check-session-route-wrappers.sh`),
  so the class cannot silently regress.
- The `withSession` enrichment trusts `X-Forwarded-For`-independent cookie
  auth only; the rate-limit key's proxy-trust caveat is documented on
  `RateLimitConfig`.
