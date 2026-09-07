# setup-demo

Runnable showcase of the [`setup/v4`](../../setup) one-call composition root:
event-sourced user management, auth API, login page, admin panel, CQRS
observability dashboard, health endpoint, and the dual SSE feeds — all from a
single `setup.New` call.

```sh
go run .
# open http://localhost:8099/dev-login  (signs in as admin@demo.dev)
```

## Routes

| Route            | Auth    | What it is                                                        |
| ---------------- | ------- | ----------------------------------------------------------------- |
| `/`              | public  | Login page (the bundle's catch-all)                                |
| `/dev-login`     | public  | Demo shortcut: sets the session cookie (dev only, never in prod)   |
| `/dev-logout`    | public  | Clears the session                                                 |
| `/auth/*`        | public  | Register / login / logout / me API                                 |
| `/admin/`        | session | Admin panel (users, tenants, memberships)                          |
| `/dashboard/`    | session | CQRS/ES observability dashboard                                    |
| `/health`        | public  | Projection readiness (503 until projections are live)              |
| `/sse`           | session | Shared SSE feed: every committed domain event, journal replay      |
| `/ds/events`     | session | DataStar SSE feed: same events, DataStar patch encoding (ADR-0050) |
| `/datastar.js`   | public  | DataStar SDK script (auto-mounted with `DataStarPath`)             |
| `/ds-demo`       | public  | Minimal DataStar client page showing the live broadcast counter    |
| `POST /broadcast`| public  | ONE action fanned out twice: raw SSE event + DataStar signal patch |

## The dual-transport point

`/sse` and `/ds/events` serve from the SAME fan-out hub
(`bundle.Broadcaster`): one `Broadcast` reaches HTMX and DataStar clients
simultaneously — no bridging code. `POST /broadcast` demonstrates it: HTMX
clients (the admin panel's sync indicator, any SSE listener) receive the raw
event while DataStar clients receive a signal patch that updates the counter
on `/ds-demo`.

This is a demo only — in-memory storage, dev-only login shortcut. Real
applications configure a WebAuthn/TOTP/OAuth2 provider and persist the event
store.
