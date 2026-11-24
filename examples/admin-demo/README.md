# admin-demo

Runnable showcase of the [adminui](../../adminui/README.md) panel: an
in-memory usermgmt service with seeded demo users and tenants, the admin
panel mounted at `/admin` behind cookie session auth. Open the root URL and
you are signed in as the demo admin automatically.

```bash
GOEXPERIMENT=jsonv2 go run .   # http://localhost:18930
```

## Machine clients: the JSON directory search + served spec

The panel is HTML-over-the-wire; the demo also ships the machine story for
the same directory:

- `QUERY /api/users/search` — the RFC 10008 native rung: a safe,
  idempotent read whose criteria travel as a JSON body
  (`{"q": "alice", "filter": "verified", "page": 1}`). Dispatched through a
  root App (`SearchDirectory`), rendered as JSON, capped at 50 users per page.
- `POST /api/users/search` — the fallback twin: the same JSON body for
  clients and proxies that cannot send QUERY (byte-identical response).
- `GET /openapi.json` — the served OpenAPI 3.2 spec, built from the
  handler's own `WithOpenAPI` metadata (`App.OpenAPIRoutes()`), ETag-cached
  (conditional GETs answer 304). Linked from the panel sidebar as
  "OpenAPI spec" (machine clients need a panel session — see below).

Both rungs are session-gated: `usermgmt.NewSessionMiddleware` only enriches
the request context, so the API route carries its own `requireSession` gate
(401 + problem document) — the pattern any consumer API surface needs.

## The seeded demo cast

Every boot seeds the same deterministic cast (in-memory state, fixed IDs via
`SyntheticUserID("seed-"+email)`), so screenshots and e2e runs stay stable:

| Who | State it demos |
| --- | --- |
| `admin@demo.dev` | super admin — auto sign-in at the root URL (`/dev-login`) |
| `alice@acme.dev` | acme owner — `/dev-login-tenant` signs into the tenant-admin panel |
| `bob@acme.dev` | acme admin (role changed from user by the seed) |
| `carol@other.dev` | globex user (role changed from viewer by the seed) |
| `erin@acme.dev` | fully equipped: seeded passkey fixture + TOTP MFA enabled, plus a real login-audit trail (one rejected verify, then a success) |
| `mallory@acme.dev` | registered with no sign-in method — the panel's can't-sign-in edge state |
| `teamNN@acme.dev` ×60 | pagination filler past one 50-row page |
| bots `demo-ci-bot`, `demo-deploy-bot` | machine actors (scope-less and `users:read`+`tenants:read`) |
| tenants `acme`, `globex`, `initech` | healthy tenants for the panel lists |
| tenant `suspendo` | suspended with a reason — the Suspended chip/badge |
| tenant `vanisho` | deleted — the `TenantDeleted` event is journalled; the pinned in-memory read model drops the row, so it demos the delete path, not a visible tombstone |

Startup output prints each bot's API token once (`SHOWN ONCE` note): the
service stores only an HMAC hash (bot registration needs the demo's
`ServiceConfig.TokenPepper` — a fixed dev value, safe only because state is
in-memory). Bots are reachable via `GetBot`/`ResolveBotByToken` and
`usermgmt.NewAPITokenMiddleware`; the panel does not render a bots page yet.

The audit page shows the seeded activity produced through real published
paths only: registrations, session starts, passkey add, MFA enable, and
erin's `login_failed` → `login_succeeded` pair. Membership role changes
journal `MemberRolesChanged` on the membership stream — at this pin the
audit projection maps no audit action to them, so they change the members
table but not the audit page. No lockout rows are faked: the demo
configures no `LockoutStore`. External-account linking is likewise not
seeded: `LinkExternalAccount` exists only as an OAuth2-flow-internal
command at this pin, with no public `Service` method to drive it honestly.
