# Deep Review Round 2: samber/do v2 × Health Checks in cqrs-htmx

> **Date:** 2026-10-30
> **Scope:** every samber/do v2 usage and every health-check surface in the repo,
> re-audited against (a) the DO-1→DO-6 anti-pattern rules, (b) the pinned
> upstream libraries' current capability surfaces, and (c) the four dimensions:
> service orientation, composability, resilience, self-health.
> **Series:** round 2 of
> [`2026-10-08_samber-do-v2-health-checks-deep-review.md`](2026-10-08_samber-do-v2-health-checks-deep-review.md)
> (round 1 verdict: 8/10, "bridges yes, demo no"; its SUPERB outcome annex is
> the baseline this round verifies). Every round-1 roadmap item was re-checked
> against source, not changelogs.
> **Verification performed this round:** health + auditlog + samber-do-demo +
> integration_test suites re-run green under `-race` today (health and auditlog
> still **100.0%** statement coverage); `check-upstream-treadmill` re-run;
> samber/do upstream tags ls-remote'd; go-health v0.8.1 sources read from the
> module cache.

---

## 1. Executive Verdict

**Question 1 — "Are we using samber/do v2 in combination with Health Checks superbly?"**

**Yes.** Round 1's answer was "bridges yes, demo no." The demo gap is closed
and test-pinned; the bridges got *better* (durations on every wire, upstream
chain composition proven end-to-end, fail-loud recorder mismatch). Every
roadmap item from round 1 is verified landed or explicitly owner-gated in
TODO_LIST — nothing silently rotted.

**Question 2 — "PROPER Service Oriented, composable, resilient, self-health architecture?"**

**Yes — 9/10, up from 8/10.** The layering (dependency-free readiness core in
root → opt-in composition in `setup` → opt-in ecosystem bridges in
`health`/`auditlog` → self-monitoring UIs) is intact and now demonstrably
*proven* rather than designed: the demo runs real Kubernetes probes, the
"one probe" composition (projections + DI audit) is a passing integration
test against a real `*usermgmt.Service`, and setup's readiness composition is
literally one function serving two surfaces. The remaining half-point-class
gaps are coherence nits: two error-code namespaces for the same domain event
(T12, owner-gated), a deprecated-alias still taught by two READMEs, and a
handful of wording drifts.

### Scores

| Dimension           | R1    | R2   | One-line justification                                                                                       |
| ------------------- | ----- | ---- | ------------------------------------------------------------------------------------------------------------ |
| Service Orientation | 4.5/5 | 5/5  | Zero DI contamination of the domain core (re-verified by import scan); demo now teaches interface consumption. |
| Composability       | 4.0/5 | 4.5/5 | RecorderChain real + integration-proven; ONE readiness composition, two surfaces; fail-loud recorder refusal. |
| Resilience          | 4.0/5 | 4.5/5 | Fail-closed nil-provider, fail-loud recorder mismatch, real probe lifecycle; plain-`Run` drain still docs-posture. |
| Self-Health         | 3.5/5 | 4.0/5 | Every layer reports truth and durations reach the wire; vocabularies/error codes still drift (T12).            |
| **Overall**         | **8/10** | **9/10** | The demo is now the strongest part of the story; what remains is naming coherence, not architecture.       |

---

## 2. Round-1 Roadmap Disposition — verified against source, item by item

| R1 # | Action                                                       | Verdict this round                                                                                                                                                                                                                     |
| ---- | ------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1    | Demo probe lifecycle (Start/RegisterRoutes, kill static /health) | ✅ **FIXED + PINNED.** `container.go:110` `probe.Start`, `:123` `dashboard.Start`, `main.go:114` `probe.RegisterRoutes(mux, gohealth.DefaultRoutes())`; static `{"status":"ok"}` gone. Tests: `TestHealthProbe_StartedByNewContainer` (non-empty cache, criticals present), `TestHealthRoutes_ServeRealVerdicts` (3 probes answer real verdicts + startup latch), `TestHealthDashboard_ShowsProjectionChecks` (empty-green regression pinned). |
| 2    | Propagate eager-init errors                                   | ✅ **FIXED.** All five boot steps in `NewContainer` (`container.go:79-129`) return `errorfamily`-wrapped errors; the comment states the policy: "an application whose lifecycle wiring is broken must not start serving."                   |
| 3    | `RecorderChain` helper                                        | ✅ **DONE, then SURPASSED.** `health.RecorderChain` now *deprecates into* upstream `gohealth.ChainRecorders` (`health/probe.go:102-104`; exists in go-health v0.8.1 `recorder.go:34`) because the upstream chain always satisfies `DetailedHealthRecorder` — the bridge's own wrapper silently dropped per-check durations. The projections+audit-plugin chain is proven against a real service in `integration_test/s46_s49_seams_test.go:89-120` (incl. nil-skip and the detailed-interface assertion). |
| 4    | Surface map doc                                               | ✅ **DONE at every entry point.** `health/README.md` "Which surface for which consumer" table; `setup/README.md:238-240` cross-link; root `README.md:113` ecosystem-bridges line + `:337` module-table row.                                 |
| 5    | Unify error codes + export worker states                      | 🟡 **HALF DONE.** Worker states now **exported from root** (`projection_readiness.go:14-32`, seven `ProjectionStatus*` consts) plus the `ProjectionDrainReady` predicate (`:42`) — health's private string mirror is dead; `checkError` switches on root consts. Error codes remain split: root `projection.failed`/`projection.draining` vs health `cqrshtmx.health.projection_failed`/`projection_draining` (+`projection_unknown_status`). T12, open. See F1. |
| 6    | Drain story for plain `Run`                                   | 🟡 **DOCS POSTURE (as decided).** `setup/README.md:92` §"Drain posture: `RunWithAppkit` is the production drain path" (readiness 503 → DrainDelay 2s → stop). `run.go:93-105` unchanged. `Bundle.MarkDraining()` stays owner-gated (T13).   |
| 7    | `NewProbe` on a purpose-built constructor                     | ✅ **DONE + SURPASSED.** Builds on `gohealth.NewWithDetailedCheck` (`probe.go:54`) — the throwaway `do.New()` injector is gone (the one DO-2 blemish, closed); nil provider rejected fail-fast (`:47-50`); `duration_ns` reaches the wire on every path (issue #31); tests pin durations through `WithHealthRecorder` and the chain. |
| 8    | Demo polish (`InvokeAs`, `WithHealthCheckTimeout`)            | ✅ **DONE.** TOTP wired `do.As` + `do.InvokeAs` (`container.go:180,189`) with compile-time usage in tests; `do.InjectorOpts{HealthCheckTimeout}` documented in `health/README.md:81-86`; `WithInstanceID` documented for replicas.          |

Also closed from the R1 findings list: F10 (boot panic — `TestAuditViewerMount_NoServeMuxConflict` pins the GET-scoped mount), F11 (method-less pattern note lives in `main.go:97-101`), F12 (`do.InvokeAs` interface-scan nondeterminism — promoted to **gotcha 37** with repro + safe pattern).

---

## 3. Usage Map — unchanged in shape, better in content

Four modules import `github.com/samber/do/v2`; **nowhere else** (re-scanned):

| Module                    | Role                        | do surface used (current)                                                                                                    |
| ------------------------- | --------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `health`                  | go-health bridge            | `do.Injector` param on the recorder merge only. No container is constructed anymore (R1's dummy injector eliminated).          |
| `auditlog`                | do-auditlog bridge          | Returns `*do.InjectorOpts` (the v2-native plugin seam); self-contained `Shutdown(ctx)` viewer drain.                           |
| `examples/samber-do-demo` | canonical wiring example    | `NewWithOpts`, `ProvideValue`, `Provide`, `do.As`+`InvokeAs`, lifecycle adapters, `OverrideNamed` (test only), HW-4 annotations. |
| `integration_test`        | cross-module proof          | Recorder merge, chain composition, plugin-as-recorder, probe Start→routes E2E — all against a real `*usermgmt.Service`.        |

**Import isolation re-verified:** root, usermgmt, identity-model, setup,
adminui, loginpage, dashboardui, datastar, systemadapter — zero `samber/do`
imports. The DI container remains an application-side concern; the library
never forces it. This is still the single most important service-orientation
property here, and it still holds.

---

## 4. DO-1 → DO-6 Audit — clean, with receipts

| Rule | Smell                                        | Verdict    | Evidence (re-scanned 2026-10-30)                                                                                   |
| ---- | -------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------- |
| DO-1 | `Must*` in runtime paths                     | ✅ CLEAN   | Zero `do.Must*` in non-test code repo-wide.                                                                         |
| DO-2 | `do.New()` without matching `Shutdown()`     | ✅ CLEAN   | The only production container construction is the demo composition root (`NewWithOpts` + cleanup func, `container.go:63,131-140`); health constructs none. |
| DO-3 | `Override*` outside tests/setup              | ✅ CLEAN   | Only `container_test.go:54`, and the test *teaches the safe seam* (override the concrete service, never an invoked alias). |
| DO-4 | Package-level injector / scope var           | ✅ CLEAN   | None; `Container` is the sole injector holder (the DO-8 holder rule).                                               |
| DO-5 | `Invoke` inside loops                        | ✅ CLEAN   | None.                                                                                                               |
| DO-6 | `Shutdown()` reaching other services         | ✅ CLEAN   | Both adapters self-contained: `serviceLifecycle.Shutdown → svc.Close()`, `broadcasterLifecycle.Shutdown → hub.Shutdown(ctx)`. |

Structural smells (service-locator, injector-stored-in-service): none.

**Gap note (F6 below):** these rules are enforced by *review* in this repo.
branching-flow's `doanalyzerv2` (the DO-1→DO-6 analyzer) is not wired into
`check-modules`/CI — the sweep above was manual `rg`. Mechanizing it is a
roadmap item, owner-gated on a Go-installable distribution.

---

## 5. Health-Check Surface Inventory — current state

Same seven-surface layering as R1 (deliberate: library principle), now with
two structural improvements:

1. **One readiness composition, two surfaces.** `bundleReadinessChecks()`
   (`setup/mount.go:137-149`: projection gate + hub check + consumer
   `Config.HealthChecks`) is served verbatim by the mounted `/health` handler
   AND adapted onto appkit's boolean `ReadyCheck` (`run_appkit.go:141-151`).
   The R1-era risk — appkit silently dropping the hub or consumer checks —
   is structurally dead.
2. **The demo serves real health.** Surface 7 (static `{"status":"ok"}`) is
   now only the e2e/server Playwright stub, which is deliberate. The demo
   serves go-health's `/healthz` `/readyz` `/startupz` plus `/health-ui`.

Unchanged and still correct: liveness/readiness separation everywhere
(`LivePath` is dependency-blind by design; dashboardui `healthz` never gates
on engines); probe-vs-introspection split (public probes minimal, inventory
session-gated); projection truth lives only in `ProjectionStatuses()`.

Version currency: `samber/do v2.1.0` is upstream's latest v2 tag (ls-remote
checked); `go-health v0.8.1` and `samber-do-auditlog v0.11.0` aligned on the
treadmill; **`go-health-dashboard` is LAG 1** (v0.12.1 pinned, v0.12.2 out —
advisory; see F5).

---

## 6. The Combination Today — do v2 × health, seam by seam

```text
consumer app
└── injector (samber/do v2.1.0)
    ├── services implement do.HealthcheckerWithContext
    │      (demo adapters delegate to the LIBRARY's gates:
    │       serviceLifecycle → ProjectionReadinessCheck,
    │       broadcasterLifecycle → HubReadinessCheck  → DI health and /health can never disagree, HW-1)
    ├── auditlog.WithAuditLog(cfg) → *do.InjectorOpts ──► do.NewWithOpts
    │      (plugin hooks every registration/invocation/shutdown/health-check)
    └── gohealth.NewWithDetailedCheck(...)               ← NewProbe path
      │ or gohealth.New(injector, WithHealthRecorder(
    │      gohealth.ChainRecorders(health.Recorder(svc), auditPlugin)))  ← merge path
    └── every projection check carries the shared ProjectionStatuses() read
           time as duration_ns, on EVERY path (detailed recorder survives the chain)
```

What changed since R1 inside these seams:

- **Durations end-to-end.** The detailed-recorder interface survives
  `NewProbe`, the plain recorder, and the upstream chain — pinned by
  `TestNewProbe_CarriesDurationsOnTheWire`,
  `TestRecorder_DetailedDurationsFlowThroughWithHealthRecorder`, and
  `TestRecorderChain_PreservesDetailedDurations`.
- **Fail-loud recorder mismatch.** `WithHealthRecorder` passed to `NewProbe`
  is refused at `Start` with go-health's `ErrRecorderNotApplicable` (verified
  in v0.8.1 source: `probe.go:456-483`) instead of silently probing nothing —
  the HW-4 false-green class is now structurally impossible on this path, and
  the health README documents the refusal.
- **The auditlog plugin IS a recorder** — compile-time asserted and driven
  through a real probe in `TestHealthProbe_AuditlogPluginIsRecorder`.

---

## 7. Findings (ranked)

### F1 — P2 · Error-code namespace drift is now the top coherence gap (T12, open)

Root emits `projection.failed` / `projection.draining`
(`projection_readiness.go:97,106`); the health bridge emits
`cqrshtmx.health.projection_failed` / `projection_draining`
(`health/probe.go:171,174`) for the *same worker states*. Root itself
additionally mixes `projection.readiness.nil_provider`,
`cqrshtmx.readiness.check_timeout`, and `cqrshtmx.sse.hub_closed`. Anyone
alerting on code strings must match two-plus prefixes for one domain concept.
Wire-visible fix → next minor train, CHANGELOG receipt, one decision packet
covering all root + bridge codes (widen T12's scope beyond health-vs-root).
The *states* are already unified (exported consts) — only the codes lag.

### F2 — P3 · The deprecated alias is still what the docs teach

`health.RecorderChain` is deprecated (`probe.go:96-101`) in favor of
`gohealth.ChainRecorders`, yet `auditlog/README.md:46` and the health README
surface-map row 3 still show `health.RecorderChain(...)` in their examples.
Functionally identical today (it forwards), but a deprecation that its own
documentation doesn't follow will never die. Pick one truth: update the two
examples, or undeprecate the bridge alias (it exists so the composition reads
as one family; forwarding keeps it correct either way).

### F3 — P3 · Three texts describe two different TOTP resolution patterns

Gotcha 37 says "samber-do-demo's TOTP wiring follows this" — referring to the
`do.AsNamed` + `InvokeNamed` safe pattern — but the demo actually uses the
*type-keyed* `do.As` + `InvokeAs` under the single-satisfier rule (safe, and
the test comment documents why the override goes through the concrete seam).
Meanwhile `main.go:8`'s package comment still says "Named services (TOTP auth
provider via ProvideNamed)" — a pattern the code no longer uses. All current
code is deterministic; the docs claim mechanics the code doesn't exercise.
Cheapest fix: align the two doc texts with the actual pattern.

### F4 — P3 · Vacuous-true readiness on a `DisableService` bundle

`healthHandler` with zero checks returns `ReadinessHandler()` — constant 200
`{"status":"ok"}`. With `DisableService` (no service, README: "health = your
HealthChecks only") and no consumer checks, `/health` green-lights
unconditionally. Arguably honest (nothing to be unready about), but a
consumer who expected projection gating and disabled the service gets silence.
One README clause on the `DisableService` row — or a `New`-time warning —
closes it.

### F5 — P3 · go-health-dashboard LAG 1

v0.12.2 is out; v0.12.1 pinned (treadmill advisory). Family-wave gotcha 38
applies: bump the whole family in one change, CSS/class drift check included.

### F6 — P3 · DO-1→DO-6 enforcement is manual

The repo's only mechanized `samber/do` protection is review + the demo's
annotated patterns. branching-flow's `doanalyzerv2` exists but is not wired
into `check-modules`/CI here (no installable distribution; `check-branching-flow`
runs the go-design-smells set only). When the analyzer ships Go-installable,
a `check-do-antipatterns` stage mechanizes §4 permanently.

---

## 8. Capability Gap Analysis — pinned libraries vs what's used (updated)

### samber/do v2.1.0 (latest upstream tag)

| Capability                                | Status            | Note                                                                               |
| ----------------------------------------- | ----------------- | ------------------------------------------------------------------------------------ |
| `Provide`/`ProvideValue`/lifecycle + guards | 🟢 Leveraged      | demo remains the reference implementation                                             |
| `As`/`InvokeAs` (interface consumption)    | 🟢 Now used       | R1's 🟡 closed; scan-contract caveat documented (gotcha 37)                           |
| `InjectorOpts` plugin hooks                | 🟢 via auditlog   | correct library posture — no other consumer                                           |
| `Override*` in tests                       | 🟢 Leveraged      | safe-seam override taught in the test itself                                          |
| `InjectorOpts{HealthCheckTimeout}`         | 🟢 Documented     | health README §samber/do users (R1's 🟡 closed)                                       |
| `do.Package` / scopes / `InvokeStruct`     | ⚪ N/A            | application-side concerns; right for this library                                     |

### go-health v0.8.1 (latest) + dashboard v0.12.1 (LAG 1)

| Capability                              | Status        | Note                                                                    |
| --------------------------------------- | ------------- | ------------------------------------------------------------------------ |
| `NewWithDetailedCheck` + `DetailedHealthRecorder` | 🟢 Core of the bridge | duration_ns on every path (issue #31)                          |
| `ChainRecorders`                        | 🟢 Used + proven | integration-pinned with the audit plugin over a real service           |
| `ErrRecorderNotApplicable` fail-loud    | 🟢 Leveraged + documented | silent-recorder-drop class structurally dead                 |
| 3-probe routes / startup latch          | 🟢 Demo + integration E2E | R1's 🔴 closed; latch flip proven from the endpoint          |
| `WithCriticalServices`/`WithVersion`/`WithInstanceID` | 🟢 Demo + README | R1's F9 closed                                                 |
| batteries (`checks` pkg), `WithEvaluationHook` | 🟡 Documented | consumer-side; README shows the flightrecorder fleet pattern     |
| `MarkShuttingDown` two-phase drain      | ⚪            | setup/appkit own drain (documented posture, T13)                        |
| dashboard `Register(injector, probe)` do-native lifecycle | 🟡 Not used | demo uses `Provide` + explicit `Start` + native Shutdowner cascade — equivalent, documented |

**Version currency:** samber/do ✅, go-health ✅, samber-do-auditlog ✅,
dashboard LAG 1 (F5). No capability gap from being behind anywhere in the
health family except the dashboard patch.

---

## 9. Scoring Detail

### Service Orientation — 5/5 (was 4.5)

The last "against" was a demo that never showed interface consumption; TOTP is
now `do.As` + `InvokeAs` with the resolution contract's sharp edge captured as
gotcha 37 and a deterministic rule. The domain core remains container-free;
bridges export only upstream types (`*do.InjectorOpts`, `gohealth.*`); a
consumer never learns a cqrs-htmx container abstraction.

### Composability — 4.5/5 (was 4.0)

Chain recorders are real, duration-preserving, and integration-proven; setup's
readiness composition is one function serving two surfaces; the recorder
mismatch fails loud instead of silently probing nothing. Remaining: the
`setup.Bundle` → go-health bridge is still consumer-assembled
(`health.NewProbe(bundle.Service)` — three documented lines, and `HealthPath:
"-"` is the explicit opt-out seam); the deprecated-alias doc drift (F2) muddies
the composition story's one wart.

### Resilience — 4.5/5 (was 4.0)

Fail-closed nil-provider in the readiness gate; fail-loud recorder refusal;
per-check abandon-timeouts; projectionhost restart budgets + DLQ beneath
everything; go-health panic recovery + fail-closed classification; and the
demo's resilience story is no longer theater — a failed critical projection
flips `/readyz` to 503 and the dashboard shows it. The remaining gap is T13:
plain `Run` drains without flipping readiness (documented posture:
`RunWithAppkit` is the production drain path).

### Self-Health — 4.0/5 (was 3.5)

The system knows itself end-to-end and now *measures* itself too (durations on
the wire, startup latch proven, DI services health-checked through their
library gates so DI health and `/health` cannot disagree). What holds it under
5: the vocabulary/error-code drift (F1) and the absence of an in-repo
wire-format map for consumers aggregating across surfaces (the health README
maps by *consumer intent*, not by response schema).

---

## 10. Action Roadmap (ranked by impact × effort)

| # | Action                                                                                                                                         | Files                                          | Impact | Effort |
| - | ---------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- | ------ | ------ |
| 1 | **T12: unify projection-health error codes** — one namespace across root readiness + health bridge; widen the packet to root's own mixed prefixes (`projection.*` / `cqrshtmx.readiness.*` / `cqrshtmx.sse.*`). Wire-visible → next minor train, CHANGELOG receipt. | `projection_readiness.go`, `health/probe.go`, setup consumers | Medium — single-prefix alerting | S |
| 2 | **Deprecation truth** — either update `auditlog/README.md` + health README surface-map row to `gohealth.ChainRecorders`, or drop the deprecation on `health.RecorderChain` (it forwards and is the family-readable name). Docs-only. | 2 READMEs or `health/probe.go` | Low — kills the split-brain | XS |
| 3 | **InvokeAs wording alignment** — fix gotcha 37's parenthetical + `main.go:8` package comment to describe the actual pattern (type-keyed `do.As`, single-satisfier rule, concrete-seam override). | `docs/gotchas.md`, demo comment | Low | XS |
| 4 | **Vacuous-readiness note** — document that `DisableService` + empty `HealthChecks` = constant-200 `/health` (or warn at `New`). | `setup/README.md` (or `config_validate.go`) | Low | XS |
| 5 | **go-health-dashboard v0.12.2** — family-wave bump per gotcha 38 (one change, class-set check). | go.mod sweep | Low | S |
| 6 | **doanalyzerv2 gate** — wire DO-1→DO-6 into check-modules/CI once branching-flow ships an installable binary. Owner-gated. | `scripts/checks/`, flake | Medium — mechanizes §4 forever | M |

Items 2-4 are XS doc fixes; 1 and 6 are the only ones needing a train or an
owner decision.

---

## 11. Answers, Restated

- **"Superbly?"** Yes. The bridges were already at the ceiling for their size;
  the demo — R1's failure — is now the repo's best DI×health teaching asset,
  every claim is test-pinned, coverage is 100% on both bridges, and the
  composition the READMEs advertise is a passing integration test rather than
  prose.
- **"PROPER service-oriented, composable, resilient, self-health?"**
  **Yes — 9/10.** Dependency isolation is total, composition seams are real
  and proven, failure paths fail closed or fail loud, and every layer reports
  its truth with durations. The residual 1.0 is naming coherence (error codes,
  status vocabularies, three doc texts), not structure — and it is already
  ticketed (T12/T13 + three XS items above), not latent.

---

## Outcome note (append-only)

Round 2 (this file) supersedes round 1's *verdicts*, not its history: R1's
findings F1-F3/F5-F9 are closed and test-pinned (§2), F8 survives as T12 (F1
here), F4/F12 remain owner-gated docs posture. Next round should open by
checking T12's train status and re-running the §4 sweep — ideally mechanized
by then (roadmap 6).
