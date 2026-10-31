# Upstream draft: go-health — RegisterRoutes mounts method-less patterns

Target: github.com/larsartmann/go-health (v0.8.1). Draft status: VERIFY
CURRENT README FIRST — go-health-dashboard v0.12.2's release notes mention
upstream activity; if the README already documents the conflict, this draft
is moot and gets struck, not filed.

## Finding (verified against v0.8.1 source, 2026-10-09, F11)

`Probe.RegisterRoutes(mux, routes)` mounts its endpoints as method-less
patterns (e.g. `mux.Handle("/healthz", ...)`), not `GET /healthz`. Under
Go 1.22+ ServeMux precedence rules, a consumer mounting ANY other handler on
the same path with a method (e.g. `GET /healthz`) — or vice versa — hits the
conflict panic at mount time ("conflicts with pattern /healthz"). cqrs-htmx
hit this in the samber-do-demo: a method-less `/audit/` mount conflicted with
`GET /` (pinned by `TestAuditViewerMount_NoServeMuxConflict`).

## Ask

Mount with method-scoped patterns (`mux.Handle("GET "+path, ...)`) — the
Kubernetes probe endpoints are GET-only by contract — or document the
method-less contract on `RegisterRoutes` so consumers know to mount
everything around it method-less.

## Consumer workaround

Mount health routes FIRST and scope every other mount by method; never mount
a competing handler on the same path without a method prefix.
