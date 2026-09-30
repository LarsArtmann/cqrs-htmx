# ADR-0054: Identity-External Shell Mode for setup

## Status

ACCEPTED — 2026-09-30

## Context

`setup.New` has exactly one shape: it always constructs (or adopts) a
`usermgmt.Service` and always builds the auth handler, whose `/auth/*`
routes mount unconditionally. A class of consumers does not want an
identity domain at all — apps that authenticate against an external
authority (a PBX directory, an OIDC provider with their own session
store, a corporate SSO) and still want the bundle's _runtime shell_:
the serve/drain/close lifecycle (`Run`/`RunHandler`), the one-surface
readiness composition (`HealthPath` + consumer `HealthChecks`), the
opt-in `LivePath`, and the shared `Stores` for their own event
infrastructure.

The blocking gaps, source-level:

1. **No way to omit the auth surface.** `New` builds
   `usermgmt.NewAuthHandler(...)` unconditionally and `Mount` registers
   `/auth/*` whenever the handler is non-nil — which is always. The
   routes include open registration writing into the service's user
   database; for an identity-external app that database is an unused
   second user store and the routes are pure attack surface.
2. **No way to omit the service.** `New` derives the shared stores
   from `svc.Journal()`/`svc.EventBus()` — the service is the bundle's
   spine. Every session-gated surface (feeds, machine endpoints,
   panels) authenticates against it.

The first external consumer asking for exactly this is webphone
(single-binary unified-communications app; identity = PBX extension +
directory password proven by the island's SIP REGISTER). Its adoption
plan (webphone `docs/planning/2026-09-30_10-37_SUPERB-setup-shell-adoption.html`)
documents why usermgmt itself can never be its identity: sessions must
carry the PBX directory password, which usermgmt sessions cannot.

## Decision

Two independent, composable `Config` flags:

### `DisableAuth` — no auth handler, service stays

`New` skips the `usermgmt.NewAuthHandler` construction; `Bundle.Auth`
is nil, so `Mount` registers no `/auth/*` routes. The service is still
built or adopted, so panels and `SessionMiddleware` keep working —
sessions can be minted by consumer code against the service API
(power-consumer mode: your own login endpoint, the bundle's panels).

Validation rejects two combinations that would silently break:

- `DisableAuth` with `DisableLogin=false` — the login page's form
  posts to `/auth/*`; without the routes it is a dead form.
- `DisableAuth` with `AuthHandlerConfig` — it configures endpoints
  that will not exist.

### `DisableService` — the shell

`New` builds no `usermgmt.Service` at all: `Bundle.Service` and
`Bundle.Auth` are nil, no panels are constructed, and `Stores` comes
from `Config.EventStore`/`Config.EventBus` — defaulting to
`memory.NewMemoryStore()` and `watermill.NewEventBus()`, the same
defaults the service path applies. The bundle keeps everything that
never needed the service: the lifecycle (`Run`/`RunHandler`/`Close`),
`HealthPath` with consumer `HealthChecks` (the built-in
`projections`/`sse-hub` checks are skipped naturally — nil service,
nil hub), opt-in `LivePath`, and `Handler`/`Middleware` composition.

`Close` is a no-op on a shell bundle (nothing owns a service, no hub
exists) and remains idempotent.

Validation rejects every surface whose session gate would dereference
the service that does not exist — `usermgmt.NewSessionMiddleware(nil, …)`
panics on the first request carrying the configured cookie, so a
half-mounted session-gated endpoint is a latent panic, not a feature:

- `Service`/`ServiceConfig` and all service-construction fields
  (`TOTP`, `WebAuthn`, `OAuth2`, `SessionTTL`, `Logger`,
  `AsyncStartup`, `OnProjectionFailed`, `Observability`,
  `ReadModelDB`) — except `EventStore`/`EventBus`, which become the
  shell's own inputs.
- All three panels enabled (`DisableAdmin`/`DisableDashboard`/
  `DisableLogin` must all be true) — the dashboard dereferences
  `Service.ProjectionHost()`, admin and login need the service.
- The event feeds (`SSEPath`, `DataStarPath`) — their gate is the
  bundle session middleware.
- The machine endpoints (`EventCatalogPath`, `ProjectionStatusPath`,
  `DebugPath`) — same gate, and the catalog describes the identity
  domain the app does not have.
- `AuthHandlerConfig` — nothing to configure.

The feeds and machine endpoints return in a future release behind an
injectable gate (`Config.SessionGate func(http.Handler) http.Handler`
or similar) so identity-external apps can mount them with their own
auth; rejected-now beats a latent-panic endpoint.

### Constructor: `NewShell`

`setup.NewShell(cfg)` is `New` with `DisableService` forced — the shell
constructor direct. Functionally identical, but a shell-only consumer
importing only `NewShell` keeps the service path unreachable, so the
linker prunes the usermgmt/casbin/panels tree a call to `New` would
retain. Shell consumers should prefer it; webphone's adoption plan
records the binary-size gate this answers.

### Composability

`DisableService` implies `DisableAuth` (auth without a service is
meaningless; `Auth` is nil regardless of the flag). `DisableAuth`
alone composes with a live service for the own-login-endpoint mode.

### Naming

`DisableAuth`/`DisableService` continue the established
`DisableAdmin`/`DisableDashboard`/`DisableLogin` flag family rather
than introducing a mode enum; each flag independently shrinks the
constructed surface, and the combination table is small enough to
validate explicitly.

## Alternatives considered

- **Split a zero-usermgmt `shell` submodule** — best binary footprint,
  but forks the composition root (two `New`s, two validation sets,
  double maintenance). MEASURED 2026-09-30 by webphone, the first
  identity-external consumer: importing the setup PACKAGE links its
  whole import graph (+72 modules — usermgmt, adminui, dashboardui,
  loginpage, casbin, appkit, datastar) even through `NewShell`, because
  Go links the init functions and package-level state of every
  transitively imported package regardless of function reachability.
  Their shell-mode binary: 15.25 → 25.65 MB, +10.40 MB = +68.2%,
  against a recorded ≤ +8 MB / ≤ +20% gate — NO-GO; the consumer kept
  the rejection and took the lifecycle value from `httputil.NewServer`
  (the primitive `RunHandler` wraps) at +8 KB instead. A future
  `shell` submodule that imports only event/storage/watermill + root
  would dodge the graph; build it when a second consumer needs more
  shell than the lifecycle.
- **`ShellMode bool` single flag** — one flag is simpler to set but
  cannot express the own-login-endpoint mode (service yes, auth
  routes no), which is a real composition; two orthogonal flags with
  validation beat one flag plus hidden coupling.
- **Serve feeds/machine endpoints ungated in shell mode** — silently
  changes documented wire contracts (the `/sse` 401 precedent);
  rejected.

## Consequences

### Positive

- The identity-external class of consumers can adopt the tested
  lifecycle and readiness composition without forking or carrying a
  dormant second user database.
- webphone becomes the reference consumer, hardening the seams for
  every future identity-external app.
- The rejection errors name the exact conflicting fields — the
  fail-fast posture the module already applies to service sources.

### Negative

- Five session-gated surfaces stay unavailable in shell mode until
  the injectable gate lands; consumers mount their own.
- Two more flags in an already-wide `Config`; the validation matrix
  grows (pinned by tests).

### Verification

`setup/setup_identity_external_test.go` pins both flags: the shell
bundle builds and closes idempotently with nil `Service`/`Auth`,
serves health from consumer checks only, mounts no `/auth/*` (404),
serves no session-gated surface, and every documented rejection fires;
the `DisableAuth` bundle keeps the service, mounts no `/auth/*`, and
fails closed on the auth routes.

## See also

- ADR-0048 — liveness/readiness decoupling (the `HealthPath`/`LivePath`
  semantics the shell composes)
- ADR-0050 — dual frontend protocol (the `/sse` 401 gate precedent)
- `docs/guides/setup-vs-hand-wiring.md` — when to use setup at all
