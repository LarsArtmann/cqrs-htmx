# Owner decision packet — usermgmt → system/metaengine migration (D1–D5)

**Filed:** 2026-11-07 (blanket-directive session) · **Plan:** [`docs/planning/2026-11-07_10-45_SUPERB-usermgmt-system-metaengine-migration-plan.md`](../planning/2026-11-07_10-45_SUPERB-usermgmt-system-metaengine-migration-plan.md) §8 · **Status: RESOLVED — defaults adopted** under the owner's standing blanket directive ("NOW GET SHIT DONE! The WHOLE TODO LIST!", the same tick that resolved D18), per that packet's "Default if unanswered" column.

| #   | Question (abridged)                                                                                 | Decision (the plan's default)                                                                                                        |
| --- | --------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| D1  | `usermgmt.NewService` façade over `system.New`, or composition moves to consumers?                  | **Façade.** Public API stability wins; OQ 14 (a `setup.NewFromSystem`-shaped composition entry) rides W5, not a v4 break.            |
| D2  | V007 cluster 1 SQL view tier: evaporate at v5 or keep?                                              | **Evaporate** (plan §5) — the SQL view tier is deleted in the v5 window; read models go through the projection/query layer.          |
| D3  | SecurityHooks/ADR-0011 has no `system.New` seam: upstream ask + interim dual-path, or block?        | **Upstream ask + dual-path.** Encrypted deployments keep `NewEventSourcedSetup` until the seam ships upstream; no fork, no block.    |
| D4  | Authz hot path: declarative `Enforce` query, or live Casbin enforcer hydrated from a query at boot? | **Live enforcer in the façade** (µs-range checks stay); the declarative query is the rebuild/hydration source, not the request path. |
| D5  | Startup read-your-writes: blocking drain in the façade, or `sys.Start` + poll?                      | **Poll + readiness gate** — matches the `AsyncStartup` posture and `cqrshtmx.ProjectionReadinessCheck`; no hidden blocking.          |

## Execution consequences

- W0 (truth on disk) is COMPLETE this session: W0.1 (rows + inventory errata, morning session), W0.2 (systemadapter doc-rot counts 20/21/24/25 → 21/21/27/27, this session), W0.3 (this packet).
- W1–W2 (probes + façade) are GO — reversible, test-gated work that follows the plan.
- W3–W4 (read-tier cutover, file deletion) stay gated on the plan's technical gates (the W1.2 starvation probe; the criterion in §6), not on further owner input — D1–D5 were the only owner-held keys.
