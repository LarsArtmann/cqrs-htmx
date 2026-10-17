# The GOTH Stack

**G**O + **T**empl + **H**TMX — a server-rendered Go web stack where the server
owns the markup, HTMX diff-speaks HTML over the wire, and templ makes the
templates typed Go. No SPA, no client-side state mirror, no JSON API you did not
ask for. This library is one of three repositories that compose it.

## The three repositories

| Repository                                                          | Role                                                                                                                                                             | Depends on                  |
| ------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite)         | The CQRS + event-sourcing core: deciders, event store, projections, the `system` composition root, the `metaengine` query planner                                | (standalone domain library) |
| [cqrs-htmx](https://github.com/larsartmann/cqrs-htmx) (this repo)   | The HTTP/HTMX binding for that core: declarative command/query handlers, Casbin authz, SSE, event-sourced user management, admin UI, dashboard, one-call `setup` | go-cqrs-lite                |
| [templ-components](https://github.com/larsartmann/templ-components) | UI components for templ + HTMX + Tailwind v4: layouts, forms, tables, badges, dialogs, polled regions — CSP-safe, dark-mode aware                                | (standalone UI library)     |

The dependency direction is deliberately a tree, not a graph: templ-components
knows nothing about CQRS, go-cqrs-lite knows nothing about HTTP, and cqrs-htmx
is the only place the two meet. Every piece is usable without the others.

## How they compose

- **cqrs-htmx → go-cqrs-lite:** every command/query handler dispatches through
  go-cqrs-lite's dispatchers; events flow through its store and projections.
  The optional `setup` module wires the whole identity domain (passkeys, OAuth2,
  TOTP, sessions, audit trail) in one call.
- **cqrs-htmx's UI modules → templ-components:** the ready-made surfaces
  (adminui, dashboardui, loginpage) render through templ-components, so they
  pick up your Tailwind theme instead of shipping their own CSS framework.
- **Your app → any subset:** bring your own router, your own templ components,
  your own persistence. The stack never picks for you — see the
  [README](../README.md) for the library principle behind this.

## Why server-rendered

One source of truth (the server), one wire format (HTML fragments), one
language (Go). Interactivity arrives via HTMX attributes and SSE streams —
both are progressive enhancements over working HTML, not prerequisites. The
event-sourced backend means every UI state is a projection you can rebuild,
replay, and time-travel (the [dashboardui](../dashboardui) module exists
precisely to watch that happen live).

## Pointers

- cqrs-htmx: [README](../README.md) · [docs/guides/](guides/) (wiring, SSE,
  frontend sync, projections)
- go-cqrs-lite: see its README for the decider/system/metaengine surface
- templ-components: see its README for the 110+ component catalog
