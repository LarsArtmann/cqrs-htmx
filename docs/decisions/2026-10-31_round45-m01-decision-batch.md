# M01 decision batch — round-45 (2026-10-31)

Five verdicts recorded per plan §8 recommendation defaults (plan: `docs/planning/2026-10-31_00-36_SUPERB-round45-do-health-closure-and-backlog-burndown-pareto-plan.md`; context: round-2 review §7). Owner-approval class: round-45 instruction "execute the WHOLE list" + the 10-42 owner-verdict-defaults precedent.

| # | Decision | Verdict | Rationale |
| --- | --- | --- | --- |
| 1 | T12 scope + survivor namespace | **WIDEN** to every wire-visible projection-health/readiness code in root + health bridge. Survivor: `cqrshtmx.<domain>.<event>` — one namespace, domain-segmented. Root's `projection.failed`/`projection.draining`/`projection.readiness.nil_provider`/`readiness.check_timeout` and bridge's `cqrshtmx.health.projection_failed`/`projection_draining` all collapse into `cqrshtmx.projection.*` / `cqrshtmx.readiness.*` / `cqrshtmx.sse.*` stays. | Alerting must match ONE prefix per domain concept; root's existing `cqrshtmx.` family (readiness/sse) already uses it — extending it is the smaller wire change than minting a second bare-prefix family. `cqrshtmx.sse.hub_closed` already conforms. |
| 2 | `RecorderChain` fate | **MIGRATE DOCS** to `gohealth.ChainRecorders` in `auditlog/README.md` + health README surface-map row 3; KEEP the deprecated bridge alias (source compat shipped in health v4.14.0). | Undeprecating a symbol deprecated one train ago churns the API for aesthetics; the alias still forwards and the composition-as-family intent is preserved by the docs showing the canonical form. |
| 3 | T13 (setup drain refusal) | **CLOSE WON'T-FIX** (dated). `RunWithAppkit` is the documented production drain path; plain `Run` keeps lb-refusal semantics (fail-loud on unready under load balancers is the honest default). | No `MarkDraining()` API is added: the drain-aware production path exists and is documented; adding a second drain mechanism creates two truths for one lifecycle moment. |
| 4 | `InvokeAs` interface-scan filing | **FILE UPSTREAM** at samber/do (map-order scan is a resolution-contract bug; repro verified twice against v2.1.0 in round-2 §7/F3 chain + round-2 §8). Draft lands under `docs/proposals/upstream/` in M20 before filing (verify-before-filing). | Deterministic single-satisfier rule is a public contract of the repo's DO-pattern; map-order dependence breaks it silently for downstream consumers. |
| 5 | erraudit strict-sweep gate policy | **ADVISORY FLAKE APP FIRST** (`.#check-erraudit-strict`, wired into check-modules as a non-blocking stage). Blocking promotion only after one green advisory month, tracked in TODO_LIST. | linter-building guidance: gating on full strictness before an advisory period produces silent `nolint` farming; the 29/29 green sweep must be earned first. |

Downstream wiring: verdict 1 → M02 (rename) + M09 (vocabulary map); verdict 2 → M04 (two READMEs); verdict 3 → M08 (dated close, no code); verdict 4 → M20 (draft + file); verdict 5 → M12 (advisory app + self-test).

## Verdict-1 execution mapping (m02.2, 2026-10-31)

Enumerated from source (readiness.go, projection_readiness.go, health/probe.go, app.go):

| Old code | Site | New code |
| --- | --- | --- |
| `projection.readiness.nil_provider` | projection_readiness.go:85 | `cqrshtmx.projection.readiness.nil_provider` |
| `projection.failed` | projection_readiness.go:97 | `cqrshtmx.projection.failed` |
| `projection.draining` | projection_readiness.go:106 | `cqrshtmx.projection.draining` |
| `cqrshtmx.health.projection_failed` | health/probe.go:171 | `cqrshtmx.projection.failed` (same worker state as root → identical code) |
| `cqrshtmx.health.projection_draining` | health/probe.go:174 | `cqrshtmx.projection.draining` |
| `cqrshtmx.health.projection_unknown_status` | health/probe.go:180 | `cqrshtmx.projection.unknown_status` |
| `cqrshtmx.health.no_provider` | health/probe.go:48 | unchanged (already conforms) |
| `cqrshtmx.readiness.check_timeout` | readiness.go:145 | unchanged (already conforms) |
| `cqrshtmx.sse.hub_closed` / `cqrshtmx.sse.hub_draining` | readiness.go:175,178 | unchanged (already conforms) |
| `config_invalid` | app.go:150 | unchanged — out of T12 scope (construction-time config guard, not a health/readiness wire event); residual noted for a future namespace pass |

No Go test asserts any old code string (verified by grep 2026-10-31: health tests assert message substrings only; projection_readiness_test.go's `nil_provider_fails_closed` is a case name). Wire-visible → CHANGELOG receipts in root + health in the same change.

