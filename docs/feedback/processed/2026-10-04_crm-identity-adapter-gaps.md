# Consumer Feedback — What the Ledger CRM's Identity Adapter Has to Reinvent Around `setup`/`usermgmt`

> **PROCESSED** (2026-10-04): all five items acted on in the library; see ADR-0055 and `usermgmt/CHANGELOG.md` [Unreleased].
>
> 1. **DONE** — WebAuthn enrollment ceremonies enforce the owner-match rule (401 no session / 403 mismatched `user_id`); no HTTP opt-out; service-level API stays open for headless enrollment. loginpage's register→begin→finish flow verified compatible (cookie set at register; `credentials: "same-origin"`).
> 2. **DONE (wider than reported)** — ALL `currentUser`-reading routes (`/auth/me`, credentials ×2, verify/send, TOTP ×4, export, import, OAuth2 unlink) are self-wrapped with an enrich-only session pass inside `RegisterRoutes`; setup inherits the fix with zero consumer action. Mechanical invariant gate (`nix run .#check-session-route-wrappers`) keeps the class dead.
> 3. **DONE** — `setup.RequireSession` (401/JSON) and `setup.RequireSessionRedirect(loginURL)` (303/browser) exported, both mounting after the enrich-only `bundle.SessionMiddleware()`.
> 4. **DONE** — `Service.DisplayName(ctx, userID string)` with the two-shape contract (bare ULID + `user:`-prefixed actor strings), name→email→"" ladder, tombstone-filtered by both read-model backends.
> 5. **DONE** — `newLimiterFromConfig` keys on `httputil.ClientIP` (X-Forwarded-For → X-Real-IP → RemoteAddr host); proxy-trust caveat documented on `RateLimitConfig`. Upstream httputil doc note drafted separately.
>
> Acceptance path for the CRM ("delete CredentialsGate, nothing reddens"): items 1+2 make the gate redundant — M21 of the execution plan runs the CRM-side check when the repo is reachable (`docs/planning/2026-10-04_12-12_SUPERB-identity-auth-hardening-pareto-plan.md`).

**From:** Ledger CRM (private, event-sourced personal CRM on go-cqrs-lite; single-user, `-auth` opt-in WebAuthn posture)
**Date:** 2026-10-04
**Version evaluated:** cqrs-htmx local checkout HEAD (setup/v4 + usermgmt/v4); consumer adapter at `crm/internal/identity/identity.go`
**Consumer:** Crush (AI assistant) + Lars
**Provenance:** session question "what does internal/identity do that identity-model doesn't, and could setup handle any of these responsibilities"; every claim below re-verified against cqrs-htmx source today (file:line cited), plus the CRM's pinned tests (`credentials_gate_test.go`, `display_bdd_test.go`, `TestRegistrationRateLimitContract`).

---

## Context: how the CRM consumes the stack

The CRM embeds `setup.Bundle` via a ~260-line adapter (`internal/identity`) that exists
only in `-auth` mode. The layering question behind this report: identity-model is the
pure domain, usermgmt is the implementation, setup is the composition root — and the
adapter sits on top, currently holding several responsibilities that are **generic
library-shaped, not CRM-shaped**. Each of those is a candidate to move down a layer.
We ranked them by how much they cost us to own (weight) and how universally other
consumers will hit them (reach).

## 1. WebAuthn registration ceremonies bind to an arbitrary `user_id` — SECURITY (top weight)

`POST /auth/webauthn/register/begin` reads the target user from the JSON body
(`webauthn_http.go:34-43`, `req.UserID`), and `register/finish` reads it from the
`user_id` **query parameter** (`webauthn_http.go:51-57` via `requireUserIDFromQuery`,
`webauthn_http.go:14-25`). Neither handler consults the session. The routes are
mounted by `RegisterRoutes` with no middleware wrapper (`usermgmt/http.go:199-209`).

Consequence for any network-reachable consumer that mounts the bundle's routes as-is:
**anyone can attach a passkey to a known user id, then log in as that user.** Possession
of the user id (a ULID — sequential-ish, enumerable) is the only "authorization."

The CRM wraps these routes in a mux-level re-gate (`internal/identity`'s
`CredentialsGate`: session middleware + 401 when `UserIDFromRequest` is empty). This
is the workaround we cannot drop on a library bump — our AGENTS.md carries a standing
"do NOT remove the gate unless the library starts gating them itself" warning, which
is exactly the kind of consumer-side landmine a library fix would disarm.

Bootstrap note for the fix design: `POST /auth/register` issues a session cookie before
any credential ceremony, so session-gating the register routes does not break
first-user bootstrapping — verified from the CRM side 2026-09-19. A library-side
session gate (or at minimum `Config` opt-out for exotic flows) is the single highest-
value change in this report.

## 2. `/auth/credentials*` handlers silently depend on the host wrapping them in SessionMiddleware

`handleListCredentials` and `handleDeleteCredential` read the user exclusively from
request context (`credential_http.go:36-42`, `:93-99` via `currentUser` →
`UserFromContext`). That context is only populated by `usermgmt.SessionMiddleware` —
which `RegisterRoutes` does **not** apply to these routes. Mounted bare (as the setup
bundle mounts them), they fail closed with 401 forever, even for a fully logged-in
user, because nothing ever parses the session cookie on that path.

So the credential-management UI a consumer ships (or the bundle's admin panel, if it
links there) is dead-on-arrival unless the host independently knows to wrap
`/auth/credentials*` in session middleware. Our `CredentialsGate` covers this prefix
too — half of its reason for existing is making these routes *work*, not just closing
the item-1 hole. The bundle already owns the service and cookie name; applying
`SessionMiddleware` to its own credential routes at mount time (or documenting the
requirement loudly in `RegisterRoutes`' doc comment) would remove the trap.

## 3. No "require session, else redirect to /login" middleware for HTML apps

`Bundle.SessionMiddleware()` enriches context but never challenges. Every HTMX/HTML
consumer needs the same 15-line wrapper: session middleware → if no user →
303-redirect to the login page. Ours is `Identity.Gate()`; PapDashboard hand-rolled
its own cookie gate; this will recur for every browser-facing consumer that is not a
pure SPA. A `Bundle.RequireSessionRedirect("/login")` (API consumers would want the
401/JSON sibling, arguably `RequireSession`) would delete the whole class.

## 4. No display-name resolver over the user read model

Deal cards, deal detail, and the journal timeline render owner/actor ids as human
labels. We maintain a `DisplayName(ctx, userID)` that queries
`SELECT display_name, email FROM users_view WHERE key = ? AND tombstoned = 0` with a
display ladder (name → email → "") and — the subtle part — tolerance for **two id
shapes**: bare user ids from deal views and `"user:<ulid>"`-prefixed actor strings
from journal entries (`PrefixedString()`), retried prefix-stripped as fallback.
Our rendering BDD suite pins this duality.

The read model and its schema live in usermgmt; every consumer that ever shows a user
id in HTML will re-derive this query and get the prefix-shape handling wrong the
first time (we did). A `Service.DisplayName(ctx, id)` or read-model helper — with an
explicit contract for which id shapes it accepts — belongs in usermgmt. The
prefixed-vs-bare duality is itself a fleet-level seam worth one canonical helper
(go-branded-id's `PrefixedString` meets HTTP-layer string ids).

## 5. Rate limiter keys on RemoteAddr including port (per-TCP-connection budget)

`httputil.KeyedRateLimiter` with `KeyExtractorFromRemoteAddr` keys on `RemoteAddr`
**including the port**, i.e. per connection, not per IP. Pool churn (every new
connection = fresh budget) makes the caps much weaker than the config suggests; our
rate-limit contract test had to pin `http.Transport{MaxConnsPerHost: 1}` just to
observe the limit deterministically. Acceptable for a single-user tool; wrong for any
real multi-client exposure. Flagging as upstream-issue material rather than a fix
demand — but if item 1 makes the auth surface more security-load-bearing, this
undermines the same surface.

## What we deliberately keep in the adapter (not library candidates)

For completeness, the responsibilities that are host-shaped and should stay:

- **The principal seam** (`OwnerIDFrom` → `Server.OwnerFrom`): what "owner of a deal"
  means is domain vocabulary; setup cannot know it.
- **Single-SQLite-file composition** (one stack + second WAL read-model handle on the
  same DSN, `errors.Join` cleanup): deployment choice, plain composition.
- **Journal separation** (identity journal ≠ CRM journal): bounded-context decision.
- **Cookie posture tri-state** (`Secure *bool`): already expressible via
  `AuthHandlerConfig.Secure` + `Config.CSRF`; we only map operator intent onto it.

## Summary table

| # | Item | Layer it belongs in | Severity |
|---|------|---------------------|----------|
| 1 | Session-bind WebAuthn register ceremonies (arbitrary `user_id` from body/query) | usermgmt handlers / setup mount | Security |
| 2 | Apply SessionMiddleware to `/auth/credentials*` at mount | usermgmt `RegisterRoutes` | UX trap (dead routes) |
| 3 | `RequireSessionRedirect` helper | setup Bundle | Boilerplate |
| 4 | `DisplayName` resolver with explicit id-shape contract | usermgmt service/read model | Boilerplate + correctness |
| 5 | Rate-limit key = IP, not RemoteAddr:port | httputil/usermgmt | Latent weakness |

Items 1 and 2 would let the CRM's `CredentialsGate` shrink to nothing — that is the
acceptance test from our side: "delete the gate on the next bump and nothing reddens."
