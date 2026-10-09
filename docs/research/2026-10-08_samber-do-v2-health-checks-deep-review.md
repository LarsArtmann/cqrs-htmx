# Deep Review: samber/do v2 × Health Checks in cqrs-htmx

> **Date:** 2026-10-08
> **Scope:** every samber/do v2 usage and every health-check surface in the repo,
> audited against (a) the samber/do canonical patterns and DO-1→DO-6 anti-pattern
> rules, (b) the full capability surface of the pinned upstream libraries, and
> (c) the four architecture dimensions the question names: service orientation,
> composability, resilience, self-health.
> **Series:** first report on this topic (`ls docs/research/` checked per the
> gotcha-21 process guard; nearest neighbors are the 2026-09-17 setup deep-dive
> and the 2026-10-06 dashboardui/system improvements note — neither covers DI ×
> health).

---

## 1. Executive Verdict

**Question 1 — "Are we using samber/do v2 in combination with Health Checks
superbly?"**

**The bridges: yes, close to superb. The flagship demo: no — it skips the
probe lifecycle and serves two false-green health endpoints.**

- `health/v4` (the go-health bridge) and `auditlog/v4` (the do-auditlog bridge)
  are small, sharp, 100%-coverage modules that integrate through the *correct*
  upstream seams: `gohealth.HealthRecorder` and `do.InjectorOpts`. Import
  isolation is perfect — zero `samber/do` imports in root/usermgmt/setup/UI
  modules. Consumers who don't opt in pay nothing.
- `examples/samber-do-demo` — the canonical wiring consumers copy — constructs
  a go-health `Probe` and dashboard but **never calls `probe.Start(ctx)`**,
  **never calls `dash.Start(ctx)`**, **never calls `probe.RegisterRoutes`**, and
  mounts a **static `{"status":"ok"}` `/health`**. Because
  `Probe.CachedResponse()` returns a zero-value `StatusPass` with empty checks
  before the first refresh (go-health `probe.go:706-716`), the demo's
  `/health-ui` renders a permanently **empty-but-green** page — the exact
  false-green class (samber-linter HW-4) its own comments cite as the reason
  for eager construction. The health module's README documents the correct
  wiring; the demo contradicts it.

**Question 2 — "PROPER Service Oriented, composable, resilient, self-health
architecture?"**

**Yes at the module layer (8/10 overall), with drift at the surface layer.**
The layered design is right: dependency-free readiness core in root, opt-in
composition in `setup`, opt-in ecosystem bridges in `health`/`auditlog`,
self-monitoring UIs in `dashboardui`. What keeps it from "superb":

- **Five production health surfaces with three status vocabularies and two
  error-code namespaces** for the same domain concept (projection health),
  with no cross-surface map anywhere in the docs.
- **One recorder ceiling**: go-health accepts exactly one
  `HealthRecorder`, so "DI audit trail + projection checks in one probe"
  (claimed by `auditlog/README.md`) is only achievable via the demo's
  lifecycle-wrapper trick, not by direct composition.
- **One drain gap**: `setup.Run`'s plain path never flips readiness to 503
  before `server.Shutdown` (only `RunWithAppkit` and `dashboardui` do
  two-phase drain).

### Scores

| Dimension          | Score | One-line justification                                                                    |
| ------------------ | ----- | ----------------------------------------------------------------------------------------- |
| Service Orientation | 4.5/5 | Opt-in modules, duck-typed seams, zero DI contamination of the domain core.               |
| Composability      | 4.0/5 | Recorder merge + InjectorOpts + NamedCheck append are real seams; single-recorder ceiling. |
| Resilience         | 4.0/5 | Per-check timeouts, classification, DLQ/restart budgets, panic-hardened recorder path; Run-path drain gap. |
| Self-Health        | 3.5/5 | Every infra layer reports in; but the flagship demo false-greens and vocabularies drift.  |
| **Overall**        | **8/10** | Excellent bones; the demo and surface coherence are what's missing.                     |

---

## 2. Method and Version Currency

Everything below was verified against **source**, not changelogs:

- `samber/do v2.1.0` — **latest upstream tag** (`git ls-remote` checked;
  v2.0.0/v2.1.0 are the only v2 tags). Pinned in all 4 consuming modules.
- `larsartmann/go-health v0.5.0` — **latest tag**. Pinned in `health`,
  `examples/samber-do-demo`, `integration_test`.
- `larsartmann/go-health-dashboard v0.10.2` — **latest tag**.
- `larsartmann/samber-do-auditlog v0.11.0` (`auditlog/go.mod`).

Files read in full: `health/{probe,dashboard,doc}.go`, `health/README.md`,
`auditlog/auditlog.go`, `auditlog/README.md`,
`examples/samber-do-demo/{container,main,seed}.go` + tests,
`integration_test/health_probe_test.go`, root `readiness.go`,
`projection_readiness.go`, `projection_status_handler.go`, `app.go`
(HealthHandler), `setup/{mount,machine,run,run_appkit}.go`, `setup/config.go`
(health fields), `dashboardui/handlers_health.go`, plus the pinned upstream
sources in the module cache (do v2.1.0 `di*.go`/`injector.go`, go-health v0.5.0
README + `probe.go`, dashboard v0.10.2 `di.go`/`handlers.go`/`dashboard.go`,
samber-do-auditlog v0.11.0 `plugin.go`).

Tests executed during this review: `health` (PASS, **100.0% coverage**),
`auditlog` (PASS, **100.0% coverage**), `examples/samber-do-demo` (PASS —
note: passing, because nothing tests the probe lifecycle or dashboard
serving).

---

## 3. Usage Map — Where samber/do v2 Lives

Four modules import `github.com/samber/do/v2`; **nowhere else**:

| Module                     | Role                        | do surface used                                                                                                                     |
| -------------------------- | --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `health`                   | go-health bridge            | `do.New()` (dummy, inside `NewProbe`), `do.Injector` param on the recorder merge (`HealthCheckWithContext`)                          |
| `auditlog`                 | do-auditlog bridge          | Returns `*do.InjectorOpts` (plugin hooks) — the do v2-native integration point                                                        |
| `examples/samber-do-demo`  | canonical DI wiring example | `NewWithOpts`, `ProvideValue`, `Provide`, `ProvideNamed`, `Invoke`, `InvokeNamed`, `Override*` (tests only), lifecycle adapters      |
| `integration_test`         | cross-module proof          | `do.New` + `ProvideValue` + `Shutdown` in `t.Cleanup` proving the recorder merge against a real `*usermgmt.Service`                   |

**Import isolation verified:** `rg -l 'samber/do'` over root, usermgmt, setup,
adminui, loginpage, dashboardui, datastar, systemadapter → zero matches. This
is the single most important service-orientation property in the review: the
DI container is an *application-side* concern, and the library never forces it
on consumers. It matches the repo's library principle exactly.

---

## 4. DO-1 → DO-6 Anti-Pattern Audit

Audited with the samber-do-best-practices checklist. Repo-wide grep evidence
in each row.

| Rule | Smell                                              | Verdict | Evidence                                                                                                                              |
| ---- | -------------------------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| DO-1 | `Must*` in runtime paths                            | ✅ CLEAN | Zero `MustInvoke*`/`MustProvide` anywhere in non-test code; `main.go:41` even documents the deliberate choice.                        |
| DO-2 | `do.New()` without matching `Shutdown()`            | 🟡 ONE BENIGN CASE | Demo returns a cleanup func (`container.go:93-100`); integration_test uses `t.Cleanup`; **`health/probe.go:46` creates a dummy `do.New()` that is never shut down** — harmless (it holds zero services) but a cleaner constructor exists (see §8). |
| DO-3 | `do.Override*` outside `_test.go`/setup             | ✅ CLEAN | Only in `container_test.go:35` — the explicitly allowed location, and the demo *comments* the rule.                                   |
| DO-4 | Package-level injector / `*do.RootScope` var        | ✅ CLEAN | None. The injector lives inside `Container` (the documented "DO-8" holder rule, `container.go:36-38`).                                |
| DO-5 | `do.Invoke` inside loops                            | ✅ CLEAN | None.                                                                                                                                  |
| DO-6 | `Shutdown()` reaching other services/injector       | ✅ CLEAN | Both adapters are self-contained: `serviceLifecycle.Shutdown` → `svc.Close()`; `broadcasterLifecycle.Shutdown` → `hub.Shutdown(ctx)`.  |

### Canonical patterns applied (all present in the demo, correctly)

- **Composition root + cleanup function** — `NewContainer(cfg) (*Container, func(), error)`, cleanup deferred in `main`.
- **Eager foundation** — `AppConfig` via `ProvideValue` (`container.go:109`).
- **Lazy singletons** — `*usermgmt.Service`, `*cqrshtmx.App`, `*gohealth.Probe`, dashboard via `Provide`.
- **Named services** — `"auth.totp"` via `ProvideNamed`/`InvokeNamed`.
- **Lifecycle adapters with compile-time guards** — `var _ do.ShutdownerWithContextAndError = (*serviceLifecycle)(nil)` etc. (`container.go:264-267, 289-292`), including the canonical *third-party type adaptation* pattern (Close-based → Shutdowner; hub → HealthcheckerWithContext).
- **Typed accessors** — `Container.Service()/App()/Broadcaster()/Logger()/HealthDashboard()`.
- **Eager invocation of health-gated services** with explicit HW-4 rationale (`container.go:81-91` + `//samber-linter:allow hw-4` annotations) — the pattern is right; the follow-through (Start) is missing (§7, F1).
- **Test container** — production wiring + `OverrideNamed` stub in tests.

### One anti-pattern the DO list doesn't name (Medium, demo)

**F5 — swallowed eager-init errors** (`container.go:71-91`): four eager
`do.Invoke` calls log failures with `slog.Error` and *continue*. If the
serviceLifecycle invoke fails, `injector.Shutdown()` will never call
`Service.Close()`; if the probe invoke fails, the demo serves without health —
silently. `NewContainer` already returns an `error`; propagating is a two-line
fix. This is the DO-1 *mindset* (no runtime panics/aborts) over-applied to
boot, where fail-fast is correct.

---

## 5. Health-Check Surface Inventory

The repo serves/seven distinct health surfaces. Layering is deliberate
(library principle: dependency-free core, opt-in everything); the issue is
coherence, not count.

| # | Surface | Module | Semantics | Shutdown-aware? | Status vocabulary |
| - | ------- | ------ | --------- | --------------- | ----------------- |
| 1 | `App.HealthHandler()` | root | dispatcher-presence only (nil check) | no | `ok` / 503 |
| 2 | `ReadinessHandler` + `NamedCheck{Timeout}` + `ProjectionReadinessCheck` + `HubReadinessCheck` | root | parallel named checks, per-check abandon-timeout, 503 body per check | no (stateless by design) | `ok`/`fail`/`degraded` |
| 3 | `/health` (readiness: built-ins + `Config.HealthChecks` appended) + opt-in `/live` (dependency-blind 200) + session-gated `ProjectionStatusHandler` | setup | built on #2; one probe surface for consumers (`setup/mount.go:132-160`) | plain `Run`: no; `RunWithAppkit`: yes (drain) | `ok`/`fail`/`degraded` |
| 4 | `NewProbe`/`Recorder`/`NewDashboard` → go-health 3-probe (liveness/readiness/startup) + dashboard | health | K8s pattern, critical vs non-critical classification, background cache, startup latch, panic recovery | yes (`MarkShuttingDown`/`Shutdown`) | `pass`/`warn`/`fail` |
| 5 | `/-/healthz`, `/-/readyz`, `/-/versionz` | dashboardui | engine-gated readiness, liveness never depends on engines | yes (`done` channel → `shutting_down`) | `ok`/`shutting_down`/`ready`/`no_data_source`/`engine_unhealthy` |
| 6 | `/health`, `/health/live`, `/health/ready` + drain delay | setup `RunWithAppkit` (appkit-owned) | readiness 503 → DrainDelay → stop | yes | appkit's |
| 7 | static `{"status":"ok"}` | examples (samber-do-demo `main.go:126-129`; e2e/server:219) | none — **demo: false green; e2e: deliberate Playwright stub** | no | — |

**What's genuinely good here:**

- The **probe vs introspection split** matches go-health's documented threat
  model: public probes stay minimal (`/health`, `/live`), detail surfaces are
  session-gated (`ProjectionStatusPath` behind `RequireSession`,
  `setup/machine.go:64-66`). No inventory data on unauthenticated endpoints.
- **Liveness/readiness separation is correct everywhere it exists**: setup's
  `LivePath` is dependency-blind by design (a draining journal must not
  trigger restarts — `setup/config.go:242-248`); dashboardui's healthz never
  gates on engine health (`handlers_health.go:10-13`).
- **The domain's real health semantics live in one place** —
  `projectionDrainReady` (root `projection_readiness.go:15-18`) — and every
  surface above derives from `ProjectionStatuses()`, the projectionhost
  worker states. No surface invents its own projection truth.

**The drift (all P2/P3):**

1. **Status vocabulary**: three vocabularies for the same concept
   (`ok/fail/degraded` vs `pass/warn/fail` vs dashboardui's five states).
   Each is defensible in isolation (K8s consumers see only 2xx/503); a
   consumer aggregating across surfaces gets no mapping.
2. **Error-code namespaces**: root uses `projection.failed` /
   `projection.draining` (`projection_readiness.go:52,61`); health uses
   `cqrshtmx.health.projection_failed` / `cqrshtmx.health.projection_draining`
   (`health/probe.go:91,94`). Same event, two codes — alerting on code
   strings must duplicate.
3. **Worker-state strings duplicated**: root's unexported
   `projectionDrainReady` map vs health's unexported `statusLive`/
   `statusStopped` consts. A new projectionhost state today lands in health's
   `default:` branch (transient) — behaviorally safe, but the mirroring is
   convention, not type. Exporting the states (or a
   `ProjectionReady(status) bool` helper in root) would collapse it.

---

## 6. The Combination — How do v2 × Health Checks Integrate Today

Three integration seams, all correct at the design level:

```text
consumer app
└── injector (samber/do v2.1.0)
    ├── services implement do.HealthcheckerWithContext
    │      (demo: serviceLifecycle → ProjectionReadinessCheck,
    │             broadcasterLifecycle → HubReadinessCheck)
    ├── auditlog/v4.WithAuditLog(cfg) → *do.InjectorOpts ──► do.NewWithOpts
    │      (plugin hooks every registration/invocation/shutdown/health-check)
    └── gohealth.New(injector, WithHealthRecorder(health.Recorder(svc)))
           └── projectionRecorder merges:
                 injector.HealthCheckWithContext(ctx)   ← do's own service checks
                 + one check per projection             ← ProjectionStatuses()
                 (projection names win on collision)
```

- **`health.Recorder`** (`health/probe.go:55-80`) is *the* merge seam: nil-safe
  on the injector, projection names take precedence, errorfamily-typed
  results mirroring readiness semantics. Proven against a real service in
  `integration_test/health_probe_test.go` (7 workers + 1 injector check).
- **`auditlog.WithAuditLog` returning `*do.InjectorOpts`** is the do v2-native
  seam: the plugin must exist at construction to observe invocations, and
  `Setup` returns opts + plugin handle + live viewer in one call. The demo
  wires it correctly (`container.go:55-63`).
- **Lifecycle adapters make library types do-citizens**: `usermgmt.Service`
  (Close-based) and `cqrshtmx.Broadcaster` (foreign Health shape) become
  `ShutdownerWithContextAndError` + `HealthcheckerWithContext` via two ~10-line
  wrappers with compile-time guards. This is the canonical third-party
  adaptation pattern, executed exactly right.

### The one-recorder ceiling (P2)

go-health's `WithHealthRecorder` takes exactly one recorder, and a recorder
*replaces* the raw-injector batch. Consequently
`auditlog/README.md:40-42`'s claim — *"composes with the `cqrs-htmx/health`
module when you want DI audit and projection health in one probe"* — is only
true **indirectly**: pass the *plugin* as the recorder and the projections are
checked solely if they were registered as injector services via the demo's
lifecycle wrappers (with type-derived check names like
`*main.serviceLifecycle`, unusable for `WithCriticalServices`); pass
`health.Recorder` and the audit trail no longer observes health batches. The
composable fix is a chained-recorder decorator in `health/v4`:

```go
// RecorderChain runs recorders in order, merging maps (later wins).
func RecorderChain(recorders ...gohealth.HealthRecorder) gohealth.HealthRecorder
```

That one function makes the README's claim literally true.

---

## 7. Findings (ranked)

### F1 — P1 · Demo: probe never `Start`ed → `/health-ui` is an eternal empty-green page

`examples/samber-do-demo/main.go` + `container.go:86-88` construct and eagerly
invoke the `*gohealth.Probe` but never call `probe.Start(ctx)`. Verified
against pinned sources: `Probe.CachedResponse()` (go-health `probe.go:706+`)
returns **"a zero-value Response with StatusPass"** before the first refresh,
and the dashboard renders from exactly that cache
(`dashboard.go:166-172` → `SanitizeResponse(probe.CachedResponse())`). So the
page the demo advertises at `/health-ui` (main.go:118-120) shows zero checks,
status pass, forever — while projections could be failing. The eager-invoke
comment (`container.go:81-85`) cites HW-4 as its rationale; construction
without `Start` does not deliver it. **Fix:** `probe.Start(ctx)` (+ defer
`probe.Shutdown()` or `AsShutdowner()` registration in the container) and
assert non-empty checks in `container_test.go`.

### F2 — P1 · Demo: dashboard never `Start`ed → no live SSE updates

`go-health-dashboard` requires `dash.Start(ctx)` to run the SSE push loop
(upstream `di.go` example wires it explicitly). The demo never calls it — the
dashboard is static even if F1 is fixed. Upstream even ships the do-native
path the demo should use: `dashboard.Register(injector, probe, ...)` +
`dash.Start(ctx)`, after which `injector.Shutdown()` cascades to
`dash.Shutdown()` and `do.HealthCheck[*Dashboard]` reports pusher liveness
(`dashboard@v0.10.2/di.go`). The demo hand-rolls a provider that registers the
dashboard *value* — losing the lifecycle cascade.

### F3 — P1 · Demo: static `/health` false-green + K8s probes never mounted

`main.go:62,126-129` serves a hand-rolled `{"status":"ok"}` — unwired to the
probe, the projections, or the hub — and `probe.RegisterRoutes(mux,
gohealth.DefaultRoutes())` (the exact snippet in `health/README.md:37`) is
never called, so `/healthz` / `/readyz` / `/startupz` don't exist in the
flagship demo. Net: a consumer copying the demo gets **two** false-green
surfaces and zero real ones. The library's own `HubReadinessCheck` doc
(`readiness.go:158-160`) names this class: *"a dead feed answering 200 is
worse than failing the readiness gate."*

### F4 — P2 · `setup.Run` plain path: drain without readiness flip

`setup/run.go:93-105` cancels ctx → `server.Shutdown` → `b.Close()`; readiness
keeps answering 200 until the listener closes (then connection-refused).
`RunWithAppkit` does it right (503 → DrainDelay → stop, `run_appkit.go:26-32`),
and `dashboardui` does it right (`shutting_down` 503 on `done`). The plain
path relies on LBs treating refusal as down — usually true, not guaranteed
(connections in keep-alive may complete; some LBs retry). Options: a
`Bundle.MarkDraining()` that the healthHandler honors, or documenting the
appkit path as the production drain story.

### F5 — P2 · Demo swallows eager-init errors — see §4.

### F6 — P2 · Discoverability: the health module is invisible from the setup path

`setup/README.md` and root `README.md` never mention `health/v4` or go-health.
A consumer who starts at `setup.New` (the documented one-call path) meets the
built-in `/health` and has no signpost that the K8s three-probe surface with
classification, caching, and a live dashboard exists as a drop-in
(`health.NewProbe(bundle.Service)`). One "Production health" paragraph with a
cross-link closes it.

### F7 — P3 · `NewProbe`'s dummy injector

`health/probe.go:46` creates `do.New()` solely to satisfy
`gohealth.New(injector, ...)` — an injector with zero services whose
`Shutdown` is never called (the one DO-2 blemish). go-health v0.5.0 ships the
purpose-built constructor: `gohealth.NewWithHealthCheck(func(ctx) map[string]error
{ return recorder.RecordHealthCheckWithContext(ctx, nil) }, opts...)` —
semantically identical (nil-injector merge), no throwaway container. The
current form does demonstrate the recorder pattern, so this is a judgment
call — but "library code should not allocate a DI container to avoid owning
one" is the cleaner rule.

### F8 — P3 · Vocabulary/error-code drift — see §5.

### F9 — P3 · No demo of go-health's grading knobs

Nowhere in-repo does anything pass `WithCriticalServices` (outside tests),
`WithVersion`, `WithRefreshInterval`, or an evaluation hook — all documented
in `health/doc.go`/README and passable through `NewProbe`. The bridge's option
passthrough is its best feature and the demo doesn't show it. One line:
`health.NewProbe(svc, gohealth.WithCriticalServices("user-read-model",
"casbin-projection"), gohealth.WithVersion(version))`.

---

## 8. Capability Gap Analysis — What the Pinned Libraries Offer vs What's Used

### samber/do v2.1.0 (latest)

| Capability | Status | Note |
| ---------- | ------ | ---- |
| `Provide`/`ProvideValue`/`ProvideNamed` | 🟢 Fully leveraged | demo shows all three with rationale comments |
| Lifecycle interfaces + compile-time guards | 🟢 Fully leveraged | both adapters, guards present |
| `InjectorOpts` hooks (before/after registration/invocation/shutdown) | 🟢 via auditlog bridge | plugin consumes them; no other consumer — correct (library) |
| `Override*` in tests | 🟢 Fully leveraged | canonical test-container pattern |
| `InvokeAs`/`As` (provide concrete, consume interface) | 🟡 Unused | demo resolves concrete types; the skill-canonical `InvokeAs[TOTPProvider]` form would demonstrate interface consumption |
| `ProvideTransient`/scopes | ⚪ N/A | request/tenant scoping is a real-app concern, not this library's |
| `do.Package` grouping | ⚪ N/A-ish | auditlog's `Opts()` supersedes it for bridge use |
| `InvokeStruct` (batch injection) | ⚪ Unused | no struct-shaped dependency clusters in the demo |
| `do.WithHealthCheckTimeout` (InjectorOpts) | 🟡 Undocumented here | per-service health timeout exists upstream; neither health module nor demo mentions it (go-health README documents it — cqrs-htmx consumers reading only cqrs-htmx docs won't find it) |

### go-health v0.5.0 (latest)

| Capability | Status | Note |
| ---------- | ------ | ---- |
| 3-probe routes (`/healthz` `/readyz` `/startupz`) | 🔴 Missed in demo | `RegisterRoutes` never called anywhere outside README/doc examples |
| Critical vs non-critical classification | 🟡 Bridge supports, nothing uses | `WithCriticalServices` only in tests + docs |
| Background cache (`WithRefreshInterval`) | 🔴 Missed in demo | the root cause of F1 |
| Startup latch, `AwaitReady`, `Healthz()`, programmatic API | ⚪ Unconsumed | app-side concern; fine |
| `MarkShuttingDown` two-phase drain | ⚪ Unconsumed | setup/appkit own drain today |
| `NewWithHealthCheck` / `NewWithDetailedCheck` / `NewChecks` | 🟡 Better fit for `NewProbe` | see F7 |
| `WithEvaluationHook` (metrics seam) | ⚪ Unconsumed | observability is opt-in upstream posture; OK |
| `aggregate`/`federation` sub-packages | ⚪ Unconsumed | one-process library; federation is a consumer-side concern |
| `checks` batteries (Disk/Memory/HTTP/Database) | ⚪ Unconsumed | consumer-side; worth a "batteries" mention in health/README |
| `AsShutdowner` adapter | 🔴 Missed in demo | the canonical way to make `injector.Shutdown()` stop the probe loop |
| dashboard `Register(injector, probe)` | 🔴 Missed in demo | do-native lifecycle for the dashboard (F2) |
| samber-do-auditlog `Plugin` as `HealthRecorder` | 🟡 Half-true claim | §6 one-recorder ceiling |

**Version currency: perfect.** Every family member is at its latest tag; no
feature gap from being behind.

---

## 9. Scoring Detail

### Service Orientation — 4.5/5

**For:** domain core (identity-model/usermgmt/root) contains zero container
knowledge; do enters only through bridges whose *exported types are upstream
types* (`*do.InjectorOpts`, `gohealth.HealthRecorder`) — a consumer never
learns a cqrs-htmx-specific container abstraction; duck-typed seams
(`ProjectionStatusProvider`, `Enforcer`, `TemplComponent`) keep the library
framework-agnostic. **Against:** the half-point is the demo teaching
`InvokeNamed` stringly-typed resolution where `InvokeAs` would show the
interface-consumption pattern.

### Composability — 4.0/5

**For:** `health.Recorder` merges two worlds with one function;
`Config.HealthChecks` appends consumer checks into the SAME probe surface
(`setup/mount.go:139-141` — "consumers keep ONE probe surface" is tested in
`setup_health_checks_test.go`); `NewProbe` passes all go-health options
through; `auditlog.Setup` composes opts/handle/viewer in one call;
`NamedCheck.Timeout` composes boundedness per check. **Against:** the
single-recorder ceiling (no chain helper); no bridge from `setup.Bundle` to
the go-health surface (manual assembly required, demo shows it wrong);
README discoverability gap (F6).

### Resilience — 4.0/5

**For:** projectionhost gives per-projection restart budgets + DLQ, and every
health layer reports the *real* failure mode (`failed` → infrastructure error
with `LastError`; drain → transient — correct errorfamily classes so HTTP
maps to 503-not-500); readiness checks run in parallel with per-check
abandon-timeouts (`readiness.go:129-148`); go-health adds panic recovery and
fail-closed classification on the recorder path; SSE hub health surfaces
subscriber/buffer counts for capacity sizing (`readiness.go:156-184`).
**Against:** F4 (plain-Run drain), F1-F3 (the flagship demo's resilience story
is theater right now).

### Self-Health — 3.5/5

**For:** the system knows itself end-to-end — projections (workers), SSE hub
(closed/draining), engines (dashboardui), DI services (do healthcheckers +
auditlog plugin observing them), all derived from one source of truth
(`ProjectionStatuses`); machine endpoints (`ProjectionStatusPath`,
`EventCatalogPath`, `DebugPath`) give operators gated introspection separate
from public probes. **Against:** the demo false-greens (F1-F3); vocabulary
drift (F8); no map of the seven surfaces for consumers (this report's §5
table doesn't exist in-repo).

---

## 10. Action Roadmap

Ordered by impact × effort. P0/P1 are demo-scoped (no published-module risk
beyond examples; `examples/samber-do-demo` is its own module — check its
version-train status before cutting a tag).

| # | Action | Files | Impact | Effort |
| - | ------ | ----- | ------ | ------ |
| 1 | **Fix the demo's probe lifecycle**: `probe.Start(ctx)` + `defer probe.Shutdown()` (or register via `gohealth.Probe.AsShutdowner()`), `dash.Start(ctx)` (or switch to `dashboard.Register(injector, probe)`), mount `probe.RegisterRoutes(mux, gohealth.DefaultRoutes())`, delete the static `/health`, pass `WithCriticalServices`+`WithVersion` (F9). Add tests: `/health-ui` renders ≥7 checks post-Start; `/readyz` 200. | `examples/samber-do-demo/{container,main,container_test}.go` | High — kills 3 P1s, the demo becomes the canonical proof | S |
| 2 | **Propagate eager-init errors** in `NewContainer` instead of `slog.Error`+continue. | `examples/samber-do-demo/container.go` | Medium — silent shutdown-ordering loss eliminated | XS |
| 3 | **`RecorderChain` helper** in health/v4 + fix `auditlog/README.md` composition wording (or link the chain). | `health/`, `auditlog/README.md` | Medium — makes the advertised composition literally true | S |
| 4 | **Surface map doc**: add §5-style "which surface for which consumer" table (mirroring go-health README's consumer matrix) to `health/README.md`; cross-link from `setup/README.md` + root README. | docs | Medium — ends the invisible-module problem | S |
| 5 | **Unify error codes + export worker states**: root exports the drain-ready states (or a helper); health/v4 references them instead of mirroring strings; pick one error-code prefix for projection health. | `root/projection_readiness.go`, `health/probe.go` | Medium — alerting-on-code strings becomes single-prefix; kills the mirror | S (breaking-ish: code strings are wire-visible — CHANGELOG) |
| 6 | **Drain story for plain `Run`**: `Bundle.MarkDraining()` honored by healthHandler, called at the top of RunHandler's ctx.Done branch; or document RunWithAppkit as the drain path. | `setup/` | Medium — LB-correct two-phase drain everywhere | M |
| 7 | **`NewProbe` on `NewWithHealthCheck`** (F7) — drops the throwaway injector; note in CHANGELOG (behavior identical). | `health/probe.go` | Low | XS |
| 8 | Demo polish: `InvokeAs` interface-resolution example next to `InvokeNamed`; mention `do.WithHealthCheckTimeout` in health/README's samber/do section. | demo + docs | Low | XS |

Items 1-2 are strictly better demo truth; 3-7 touch published module code and
ride the next release train (CHANGELOG receipts required for 3, 5, 7 per the
consumer-visible-change rule; item 4 is docs-only, receipts in commit message
only).

---

## 11. Answers, Restated

- **"Superbly?"** The bridge modules are as good as this integration gets at
  this size — right seams, right isolation, latest versions, 100% coverage,
  real-service integration tests. The demo — the thing consumers actually
  copy — is currently the weakest link in the repo's health story and the
  direct answer is **no, not until roadmap items 1-2 land**.
- **"PROPER service-oriented, composable, resilient, self-health?"**
  **Yes at the architecture layer** (8/10): the layering is textbook
  opt-in service orientation and nothing in the domain core knows any of this
  exists. The gaps are surface coherence (vocabularies/codes/map), one drain
  path, one recorder ceiling, and demo truth — all small, all listed, all
  actionable.
