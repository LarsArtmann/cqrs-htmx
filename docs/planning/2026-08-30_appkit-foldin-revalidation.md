# Re-validation: go-appkit ADR-001 fold-in checklist (F22.6)

**Re-validated:** 2026-08-30 against current master of BOTH repos.
**Verdict: the fold-in is UNBLOCKED — the go-appkit wave is pushed.** Every
checklist item re-verified below; no item's premise has gone stale.

## Precondition (was "push pending, user gate") — ✅ MET

`~/projects/go-appkit`: master == origin (0 unpushed commits), tree clean,
tags present and pushed: core `v0.3.0` (`v0.1.0`/`v0.2.0`/`v0.3.0` line),
`cqrs/v0.3.0`, `realtime/v0.1.0`, `flightrecorder/v0.1.0` (+ health variant).
The "push the wave cut at `f938d65`" TODO is DONE.

## Checklist items (from TODO_LIST P2), re-checked

| # | Item                                                                          | Status on current master                                                                                                                                                                                                                               |
| - | ----------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| a | Drop the spike `replace`, point `go-appkit` at `v0.3.0`                       | setup/go.mod: require is ALREADY `go-appkit v0.3.0`; the `replace => ../../go-appkit` (line 123) is the only remnant — one-line strip at fold-in                                                                                                       |
| b | Fold `RunWithAppkit` into `RunHandler` behind the unchanged signature         | `setup/run_appkit.go` exports `RunWithAppkit(ctx, addr, handler)` — signature unchanged from the spike report; fold = internal swap, zero API change                                                                                                   |
| c | `/health` semantics — appkit readiness vs bundle readiness                    | still applies; decide whether `/health` stays bundle-owned (projection readiness via `ProjectionReadinessCheck`) and appkit's own health endpoint is disabled or remapped; document the LB target (`/health/ready` question from comparison finding 2) |
| d | Dedupe stacked chains — drop bundle recovery + duplicate security-header pass | still applies; `RecommendedSecurityMiddleware()` is the dedupe point on the cqrs-htmx side                                                                                                                                                             |
| e | Logging posture — wire setup's logger or configure the appkit level           | still applies; the bench now pins `appkit.LogLevelError` (or discards slog) so the FOLD-IN must pick one production answer; the hardened bench (load-guarded baseline) is the regression tripwire                                                      |
| f | Consider exposing `Addr()`                                                    | still open upstream-side; nice-to-have, non-blocking                                                                                                                                                                                                   |

## Execution shape (one change, per the original plan)

1. Strip the setup `go-appkit` replace (require already v0.3.0).
2. Merge the `RunWithAppkit` server swap behind the unchanged `Run`/`RunHandler`
   signatures (config flag or build-tag-free internal switch — the spike
   already carries the SSE-flush/readiness/parity tests).
3. Resolve (c)/(d)/(e) in the same change: single recovery + single
   security-header pass + one logging answer; document LB health targets.
4. Re-run the canonical bench (`nix run .#bench-spike`) — the appkit-vs-baseline
   delta (~17µs vs ~20µs, logging-isolated) is the accepted cost documented
   in ADR-001 Option A.
5. Update ADR-001 status + TODO_LIST; family-train the change via verify-tag.
