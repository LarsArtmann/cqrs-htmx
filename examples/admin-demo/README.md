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
  "OpenAPI spec".

Both rungs are session-gated: `usermgmt.NewSessionMiddleware` only enriches
the request context, so the API route carries its own `requireSession` gate
(401 + problem document) — the pattern any consumer API surface needs.
