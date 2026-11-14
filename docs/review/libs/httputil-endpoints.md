# httputil/endpoints Utilization Report

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/httputil/endpoints` (separate module + tag stream from the root httputil module; stdlib-only, zero dependencies — `endpoints/go.mod` declares only `go 1.27`) · Checkout: `/home/lars/projects/httputil`

## Verdict at a glance

| Pin  | Latest (verified 2026-11-14)                       | Prod import sites | Adoption                                     | Score                                     |
| ---- | -------------------------------------------------- | ----------------- | -------------------------------------------- | ----------------------------------------- |
| none | `endpoints/v0.1.0` (tagged 2026-11-06; only tag)   | **0 files**       | **Not adopted — defensible, seam identified** | **0/100 utilization · correctly-not-adopted class** |

## What the library is

Live route discoverability for `http.ServeMux`: `*http.ServeMux` cannot enumerate its patterns, so `Registry.Handle/HandleFunc` mount a route on the mux AND record it in one declaration (the registry, not the mux, is the source of truth); `Register` mounts `GET /endpoints`, which serves the recorded routes as a self-describing listing — content-negotiated per RFC 9110 §5.3 from `Accept` (browsers get an HTML table, API clients JSON), one URL with `Vary: Accept`, `no-store`, never 406 (falls back to JSON), never panics (mux rejections surface as errors), and atomic snapshots so routes can be added while a listing serves.

## How cqrs-htmx uses it today

Not at all — zero import sites (`rg -l 'httputil/endpoints' --type go .` → nothing) and no pin in any `go.mod`. Released upstream alongside the root module's v1.9.0 cycle.

## Capability inventory

| #   | Capability                                                    | Status       | Evidence / reason                                                                                                                                                                                  |
| --- | ------------------------------------------------------------- | ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Registration-through-`Registry` (single source: mux + listing) | 🔴 Declined  | cqrs-htmx already solved the single-source problem, richer: in adminui and dashboardui **the route table IS the registration source** — mux and manifest are generated from the same rows, so "a route outside the manifest cannot be served" ([docs/guides/route-posture.md](../../guides/route-posture.md)); rows carry posture + `Why` + mode, not just a summary |
| 2   | `GET /endpoints` self-describing listing                      | 🔴 Declined  | A live HTTP route inventory is an information-disclosure decision cqrs-htmx deliberately leaves to consumers: the posture manifests have **no HTTP surface** and feed CI instead (`nix run .#check-modules` route-postures stage + the authztest behavioral matrix — proof, not display). The opt-in machine endpoints that DO expose introspection (debug, docs, event catalog) are session-gated (`setup/machine.go:70`). Also mechanically thin here: setup mounts panels as one wrapped pattern (`mux.Handle(cfg.AdminPath, ...StripPrefix...)`, setup/mount.go:61-66), so a setup-level registry would only ever record the ~10 layer rows — exactly `Bundle.RoutePostures()`' granularity ([setup/route_postures.go:19](../../../setup/route_postures.go)) — never the panels' per-route truth |
| 3   | Accept content negotiation (HTML/JSON, `Vary: Accept`)        | 🔴 Declined  | No listing surface to negotiate; cqrs-htmx's machine endpoints are JSON-only by contract, and hx partial-vs-full selection is HTMX-header-driven, not Accept-driven                                   |
| 4   | Stdlib-only embeddability                                     | ⚪ N/A       | A property, not a capability — valuable for deps-free contexts, but cqrs-htmx already depends on root httputil, so zero-dep embeddability buys nothing here                                          |

## Score derivation

4 rows − 1 N/A = **3 applicable**; 0 full + 0 partial + 3 declined ×0 = **0/100 utilization** — fleet class "correctly not adopted" (cf. [go-daemon](go-daemon.md), [go-credentials-store](go-credentials-store.md)): non-adoption is a verdict, not a gap.

## Version currency

Nothing pinned, so nothing to sweep (no treadmill entry — correct for an unimported module). Latest tag: `endpoints/v0.1.0` (verified `git ls-remote --tags origin 'refs/tags/endpoints/*'` and local `for-each-ref`, 2026-11-14; tagged 2026-11-06, the only tag in the stream).

## Should cqrs-htmx use it?

No — `RoutePostures()` is the better answer for this repo: declared manifests feed drift gates and a behavioral authz matrix (compile/CI-time proof), while `/endpoints` is a display page that proves nothing and would publish a route inventory over HTTP. The one honest seam was evaluated and rejected on the merits: `setup.Mount(mux *http.ServeMux)` (setup/mount.go:35) is the only mux cqrs-htmx owns, and an opt-in `Config.EndpointsListing`-style flag COULD thread an `endpoints.Registry` through it — but per capability #2 the registry would capture only layer-granularity rows that `RoutePostures()` already states with richer columns. If a live listing is ever asked for, the right shape is rendering `Bundle.RoutePostures()` (posture + `Why` included) at an opt-in, session-gated path — stdlib-only, no new dependency, no information cqrs-htmx doesn't already declare. Until that ask exists: no adoption, no TODO row.

## How to verify

```bash
rg -l 'httputil/endpoints' --type go .                    # nothing (exit 1)
rg -n 'httputil/endpoints' go.mod */go.mod               # no pin
git -C ~/projects/httputil for-each-ref --sort=-v:refname --format='%(refname:short)' refs/tags | grep '^endpoints/'   # endpoints/v0.1.0
git -C ~/projects/httputil ls-remote --tags origin 'refs/tags/endpoints/*' | grep -v '\^{}'                            # fc94885 endpoints/v0.1.0
rg -n 'RoutePostures\(\)' setup/route_postures.go        # the incumbent, stronger answer
```

## Prior reports

- None specific. Root module: [httputil.md](httputil.md); `server_timing` submodule: [httputil-server-timing.md](httputil-server-timing.md).
