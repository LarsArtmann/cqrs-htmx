# Status: SUPERB Execution — samber/do × Health (Phase 0+1 done, 2–5 pending)

> **Session:** 2026-10-09 16:52 → 17:19 CEST (SUPERB dispatch: plan → execute → commit/push)
> **Plan:** [`docs/planning/2026-10-09_16-52_SUPERB-samber-do-health-truth-and-composability-pareto-plan.md`](../planning/2026-10-09_16-52_SUPERB-samber-do-health-truth-and-composability-pareto-plan.md) — 18 medium tasks (T1–T18), 50 fine tasks (f01–f50).
> **Prior snapshot:** [`2026-10-09_16-21_samber-do-health-deep-review-session.md`](2026-10-09_16-21_samber-do-health-deep-review-session.md) (review + self-critique; its §g questions are decided in the plan §0).

---

## a) FULLY DONE

1. **The SUPERB plan (Phase 0)** — committed (`769e9aaf`→amended): Pareto tiers (1%/4%/20%/rest), 18 medium + 50 fine tasks both sorted by importance/impact/effort/customer-value, mermaid execution graph, decisions for the 3 open questions (Q1 execute; Q2/Q3 owner-gated with recommendations), verification battery, anti-verschlimmbesser contract.
2. **T1 — demo probe lifecycle (the 1% that delivers 51%)**: F1 live-reproduced (empty-green mechanism confirmed empirically), `probe.Start(ctx)` + `dashboard.Start(ctx)` at boot with error propagation, `probe.RegisterRoutes` mounts `/healthz` `/readyz` `/startupz`, static `{"status":"ok"}` `/health` DELETED. Evidence: `container.go`/`main.go` rewrite; tests below.
3. **T2 — grading + dashboard truth**: `WithCriticalServices("user-read-model","casbin-projection")` + `WithVersion` wired; Start validates critical names against real checks (typo = boot failure).
4. **T3 — eager-init errors propagate** from `NewContainer` (slog-and-continue is dead; every `do.Invoke`/`Start` failure returns a wrapped error).
5. **T10 — interface consumption**: TOTP wiring switched to canonical `do.Provide` concrete `*totp.Provider` + `do.As` + `do.InvokeAs[identitymodel.TOTPProvider]`.
6. **F10 — boot panic FIXED (new P1 found by live-repro)**: the demo **panicked at startup** — `mux.Handle("/audit/", ...)` conflicted with `mux.HandleFunc("GET /", ...)` under Go 1.22+ ServeMux rules; second conflict found while verifying (go-health's method-less `/healthz` vs `GET /`). Fixed: method-less `/` index + GET-scoped audit mount without `StripPrefix` (the viewer serves its own prefix — the old StripPrefix would have 404'd every viewer route even without the panic).
7. **Regression tests (f04–f07, f27–f28)**: probe cache non-empty with both critical checks; all three K8s routes serve real verdicts over a real socket AND latch the startup probe; `/health-ui` renders the projection check names; mux mounts without conflicts; override-on-concrete proves deterministic. **10/10 `-race` runs green** (was flaky before the alias-override fix), `go vet` clean, treefmt clean.
8. **Demo README rewritten** (endpoint table, 7 key decisions incl. the two upstream traps).
9. **Phase-1 commit landed** (`da710fe6` amended with the detailed message; the auto-commit daemon raced both this and the plan commit — messages amended over it both times per the documented pattern).

## b) PARTIALLY DONE

1. **Live end-to-end boot verification (f07)** — the fixed demo now constructs its mux and reaches `listen` (panic gone — captured in logs), but the final `curl`-level check could not complete: an unrelated **zombie process from a different user holds :8098** (found via `/proc/net/tcp` inode; no kill permission). Behavior proof rests on `TestHealthRoutes_ServeRealVerdicts` (real `httptest` sockets). Effort to close: S once the port frees (or parameterize the port).
2. **F12 upstream finding captured but not filed** — minimal repro exists (see §d3); filing needs the verify-before-filing + owner gates (plan f45/f46 owner-packet entries cover routing it).

## c) NOT STARTED

- **T5** — root: export projection drain-ready states (additive API) + health/v4 consumes them (kills the string mirror) + CHANGELOG receipt.
- **T4** — health/v4: `RecorderChain` decorator + `NewProbe` on `NewWithHealthCheck` + tests + CHANGELOG receipt.
- **T6** — integration_test proofs (probe.Start+RegisterRoutes E2E; plugin-as-recorder).
- **T7** — docs discoverability batch (health/README surface map + do extras; setup/root README cross-links; auditlog README wording; flightrecorder + batteries mentions).
- **T8** — setup drain posture docs (RunWithAppkit = production drain path).
- **T9** — TODO_LIST harvest + AGENTS.md pointer + doc gates.
- **T11** — full verification battery (`nix run .#test`, `.#lint`, gates) + **push** (explicitly authorized by the dispatch; not yet done — Phases 2–5 changes still pending).
- **T12–T18** — owner-gated + polish tails (error-code unification, MarkDraining, rubric re-score, setup seam spike, examples-tagging policy, research-report outcome notes).

## d) TOTALLY FUCKED UP

1. **The "flagship demo" was worse than the review claimed.** The 2026-10-08 review's 3 P1s understated it: the demo **panicked at boot** (F10) — nobody could ever have seen the false-green `/health-ui`. My review's source-reading missed it because `go test` never builds main's mux; only the live repro exposed it. Lesson burned in: the review's confidence tags ("source-verified only") exist for exactly this.
2. **I introduced a flaky test before catching it.** My first fix used `do.OverrideNamed` on the `do.As` alias — which exposed **F12: samber/do v2.1.0 alias-override nondeterminism** (18/20 runs return the original service in a minimal single-goroutine repro). I shipped it, watched it fail 1-in-3 suite runs, and had to bisect empirically (alone vs paired vs count=20) before the minimal repro pinned it on upstream, not my wiring.
3. **F12 is an upstream landmine now documented only in a test comment** — the repro lives in `/tmp/do-override-repro` (session-temp!) and the test comment references "session notes" that don't exist yet as a file. Until T9/T18 land, the durable record is incomplete. (Mitigation queued: harvest + research-report outcome note.)
4. **Two daemon commit races** — both my plan and Phase-1 commits were captured by the auto-commit daemon mid-hook before my detailed messages landed; both recovered via amend, but the second race only surfaced because I re-checked `git log` after a "nothing to commit" surprise. The `preflight-tree-check` app exists for exactly this and I didn't run it before committing.

## e) WHAT WE SHOULD IMPROVE

1. **`go run` (or an e2e boot smoke) belongs in the demo's test suite** — `go test` green while `main()` panics is the F10 class; a boot-smoke test (`exec go run .` + poll `/healthz`) would have mechanized the catch.
2. **Override-after-invoke is a do-v2 footgun worth a fleet note** — the demo now documents it; the upstream filing (F12) plus a line in the crush-config lessons file (cross-project class!) would prevent rediscovery. Candidate for `references/lessons.md` in crush-config.
3. **Run `preflight-tree-check` before every phase commit** — the daemon races are deterministic behavior now; stop being surprised.
4. **Port-parameterize examples** — `:8098` hardcoding made live verification hostage to a zombie socket; an env override (`PORT`) costs one line.

## f) Next tasks (ranked; feeds HARVEST)

| #   | Task                                                                                     | Impact | Effort | Category      |
| --- | ---------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1   | T5: root export of drain-ready states + health/v4 consumes them + CHANGELOG              | High   | 60m    | Feature       |
| 2   | T4: health/v4 `RecorderChain` + `NewWithHealthCheck` switch + tests + CHANGELOG           | High   | 90m    | Feature       |
| 3   | T6: integration_test — probe.Start+RegisterRoutes E2E + plugin-as-recorder proofs         | High   | 90m    | Test          |
| 4   | T7: docs batch — surface map, cross-links (setup/root README), auditlog wording, extras   | High   | 90m    | Documentation |
| 5   | T8: setup drain posture docs (RunWithAppkit = production drain path)                      | Medium | 30m    | Documentation |
| 6   | T9: TODO_LIST harvest + AGENTS.md pointer + **capture F12 repro durably** + doc gates     | High   | 30m    | Cleanup       |
| 7   | T11: `nix run .#test` + `.#lint` + gates, then PUSH (authorized)                          | Critical | 100m | Verification  |
| 8   | Add boot-smoke test to the demo (exec `go run .`, poll `/healthz`) — the F10 mechanization | High   | 30m    | Test          |
| 9   | Parameterize demo port (`PORT` env) — unblocks live verification                          | Low    | 15m    | Quality       |
| 10  | T12 owner packet: error-code unification proposal (+ breakage analysis)                   | High   | 30m    | Process       |
| 11  | T13 owner packet: MarkDraining vs docs posture                                            | Medium | 15m    | Process       |
| 12  | **File samber/do v2.1.0 alias-override nondeterminism upstream** (F12; verify-before-filing first) | High | 45m | Process |
| 13  | **File/note go-health RegisterRoutes method-less pattern conflict** (method-scoped-root consumers panic) | Medium | 30m | Process |
| 14  | crush-config `references/lessons.md`: override-after-invoke + method-less-mux-pattern classes | Medium | 15m    | Documentation |
| 15  | T14: rubric re-score + research-report outcome notes (incl. F10–F12 receipts, five-vs-seven fix) | Low | 30m | Documentation |
| 16  | T15: setup↔go-health seam spike → ROADMAP                                                | Low    | 60m    | Feature       |
| 17  | T17: examples-module tagging policy check + document disposition                          | Low    | 30m    | Process       |
| 18  | Kill/evict the zombie :8098 holder when possible; re-run the final live check             | Low    | 10m    | Verification  |

## g) Questions I cannot answer myself

1. **Push now or after Phases 2–5?** The dispatch authorizes push; Phase 1 is a self-contained, race-clean, all-green unit — but pushing now means CI runs on a tree whose health/v4 + docs phases are still pending in the same plan. My default: finish T4/T5 (published-module code, needs the battery) and push once — confirm or override?
2. **F12 upstream filing — in my name or yours?** The samber/do v2.1.0 alias-override nondeterminism (minimal repro ready) is a cross-fleet landmine. I can file it (verify-before-filing + github-voice skills) — but it names samber/do, a third-party upstream, and your GitHub identity. Who files?
3. **Boot-smoke test via `exec go run` (adds ~10s + toolchain dependency to the demo suite) vs. a `main()`-mux construction test (fast, but wouldn't have caught the ServeMux panic at registration time)?** — the panic WAS at mux construction, so a "build main's mux in a test" variant would catch F10-class cheaply; the exec variant additionally proves listen+serve. Which bar do you want?

---

*Report basis: this SUPERB execution run only (Phase 0 planning + Phase 1 demo truth). Phases 2–5 untouched. `.md` at the user-demanded path; both status gates run against this file before finishing.*
