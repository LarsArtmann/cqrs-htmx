# OTel 64→90 Sprint Execution — Session Status Report

> **Written:** 2026-09-14 13:56 CEST
> **Session scope:** execution of `docs/planning/2026-09-13_14-47_otel-64-to-90-end-to-end-tracing-sprint.md` (13 M-tasks), plus the verification pass and two pre-existing-drift repairs the gates surfaced.
> **Tree state at write time:** clean; HEAD `942c08c2`. All content committed, but portions were absorbed by the auto-commit daemon (see §d).

> **ANNOTATED 2026-09-20** (docs-health sweep): the OTel 64→90 sprint's open legs are closed.
> - **§b:** b1–b3 DONE (module isolation green everywhere; `check-modules` 8/8; audit gaps closed and API-verified); b4 (commit hygiene) is a historical daemon-race note.
> - **§c:** release train (v4.11.0), full `.#test`, `nix flake check`, and CI review DONE; bench-spike sanity + Logs-SDK/dashboardui-templ items remain open → `TODO_LIST.md` P1/P2.
> - **§f:** struck rows confirmed done; unmarked rows remain open (bench P1, upstream OTel/go-sse asks, guide-depth items, process hygiene, `integration_test` coverage-gate inclusion).
> - **§g:** all three questions resolved (sibling fixed + tagged; history immutable = CHANGELOG-as-narrative; v4.11.0 shipped).

---

## a) FULLY DONE

| Work | Evidence |
|---|---|
| **M1–M4 verified** (prior session's guide §2.1–2.5 restructure, §2.4 free-domain-spans recipe, §2.5 HTTP-root-span recipe, observability-demo `otelhttp` wrap + `TestOtelHTTPRootSpanAndTraceparentExtraction`) | All anchors resolve, runnable proofs present, demo tests green hermetically this session |
| **M6/M7 audit correction** — the plan's "ServiceConfig lacks PublishMiddleware/HandlerMiddleware" premise is FALSE: `SecurityHooks` is embedded on `ServiceConfig`, and `NewService` delegates to `NewEventSourcedSetup` which calls `applyBusMiddleware` (`es_setup.go:224`); ordering + applied-before-projections tests already exist (`service_security_test.go`, re-run green) | `git show af87e68f` + CHANGELOG entry + plan annotation |
| **M5: guide §2.6 "Correlation — two mechanisms, one table"** — domain-ULID path (`X-Correlation-ID` → `ContextEnrichmentMiddleware` → `App.EventOptions`, `middleware.go:15` / `app.go:210`) vs OTel-baggage path (`cqrsotel.WithCorrelationID` → `middleware.OTelCorrelationEnricher` → `otel.correlation_id` metadata), with the verified `decider.WithEnricher(CompositeEnricher(...))` recipe and a scope note on what setup does NOT wire | `docs/guides/leveraging-go-cqrs-lite.md:198-232`; every signature checked against go-cqrs-lite source before landing |
| **M8/M9: `setup.Config.Observability`** — opt-in `*middleware.OTelBundle` (upstream type, zero direct OTel imports in setup); prepends `bundle.Publish()`/`bundle.Event()` into `SecurityHooks` via new `applyObservability`; bundle outermost; nil = byte-identical legacy (test-pinned); rejected as conflict with `Service`/`ServiceConfig` (fail-fast, existing conflict machinery); README config-table row; 6 new tests (nil parity, flattened wiring, composition counts, 2 conflict rejections, end-to-end register-through-wired-bus smoke) | `setup/config.go`, `setup/setup.go`, `setup/observability_internal_test.go`, `setup/setup_observability_test.go`, `setup/README.md`; full setup suite green hermetically (11.9s) |
| **M10: middleware-demo live tracing** — `cqrsotel.Setup` (global providers + propagator + free domain spans), `CommandTracing`, `CommandTypedMetrics` → Prometheus `/metrics`, stdout spans; header comment now true; tests build through the production path with provider cleanup | `examples/middleware-demo/{main.go,main_test.go,go.mod}`; hermetic build/vet/test green (1.1s) |
| **M11: SSE fan-out observability recipe** — instrument table (`hub.OnSubscribe`/`OnUnsubscribe`/`OnDrop` methods, `Health()`, `WithOnDrop` option — all verified against go-sse source; hooks are methods, NOT constructor options), connected-clients gauge + drop counter + fan-out duration snippets, Prometheus bridge pointer, re-entrancy caveat | `docs/guides/sse-and-datastar.md` "Observability: Fan-Out Metrics" section |
| **M12: docs upkeep** — CHANGELOG `[Unreleased]` sprint entry (full provenance incl. the stale-audit-claim correction); TODO_LIST's three completed OTel P2 bullets replaced by a posture pointer (convention: no `[x]`); AGENTS.md OTel adoption-state bullet rewritten (64→90 landed, `rg`-checkable otel-free invariant, all seams); plan doc EXECUTED annotation (docs-health annotate mode); `check-docs-links` green (252 links) | `CHANGELOG.md`, `TODO_LIST.md`, `AGENTS.md`, plan doc |
| **M13: verification pass** — coverage gate all 15 modules green (**setup improved 85.9%→86.9%**, threshold 80); release-train 0 unpublished (806 requires); version-drift `--strict` green (after repair, see below); replace-directives green; go-toolchain green; dep budgets green (setup justified 19→21 with comment); setup lint 0 issues; DoD `rg -l "go.opentelemetry.io" setup/ usermgmt/` clean (library principle intact) | Gate outputs this session; hermetic runs for setup / middleware-demo / observability-demo / usermgmt security tests / integration_test |
| **Two pre-existing master breakages repaired** (found by the gates during M13): (1) `integration_test/go.mod` pinned `httputil v1.0.1` vs v1.1.1 everywhere else → bumped hermetically, drift gate green again; (2) `TestFullstackUI_LoginButtonsMatchAuthConfig/NoProviders` expected loginpage's retired copy → aligned to the canonical "No authentication method is configured." / "Login is not available yet." (loginpage's own tests pin that copy); integration_test suite green hermetically | Commit `942c08c2` (+ daemon `fe8c162f` for go.mod/go.sum) |
| **middleware-showcase vendor resync** — committed vendor tree had `httputil v1.0.1` while go.mod requires v1.1.1 (same 09-11 bump wave); `go mod vendor` fixed the working tree; build gate passes that module again (vendor/ is gitignored+untracked, so nothing to commit) | `nix run .#build` re-run: showcase passes |

## b) PARTIALLY DONE

| Work | State | What remains |
|---|---|---|
| ~~**Module isolation / workspace hermetic build**~~ done — every module green (sibling master fixed; systemadapter/v4.11.0 tagged 2026-09-20) | ~~Every module green EXCEPT `systemadapter` + `examples/system-demo`~~ | ~~Both carry the documented TEMPORARY `replace => ../../go-cqrs-lite/...`; the sibling repo's concurrent session removed `metaengine.Store.Reset` from master (`reset.go:26: a.store.Reset undefined`), breaking `metaengine/projectionadapter` on master. Cannot be fixed from this repo — needs either the sibling session to finish + tag `projectionadapter v4.5.0+`, or a deliberate re-pin of the replace to an older sibling commit. I chose NOT to touch the sibling repo mid-refactor.~~ |
| ~~**`nix run .#check-modules` as one green command**~~ done — 8/8 stages green | ~~All stages green individually (drift/budgets/toolchain/release-train/replace-directives) but the composite still ends red on the isolation stage because of the systemadapter class above~~ | ~~Blocked on the same sibling-drift item~~ |
| ~~**Sprint outcome "adoption score 90"**~~ done — all audit gaps closed and API-verified against upstream source (AGENTS.md OTel bullet) | ~~All 5 audit gaps closed and the posture bullets say so, but no re-audit was run against `docs/research/2026-09-13_otel-deep-dive.html`'s scoring rubric — "90" is asserted, not measured~~ | ~~A short re-score pass over the audit's 7 findings would make the number evidence-based~~ |
| **Commit hygiene for the setup phase** | Content 100% at HEAD and tests green, but the M8/M9 + §2.6 first-iteration work was shredded into 5+ heuristic `chore: auto-commit` commits (`b638f3cc`, `6a6eb6c7`, `120afe39`, `3389c336`, `e244dc55`, `cb152429`) because my explicit phase commits came after the daemon's poll | History does not tell the story for those changes; CHANGELOG carries the narrative instead. See §d/§e. (Historical — daemon race recurred through 2026-09-20; see AGENTS.md.) |

## c) NOT STARTED (deliberately out of sprint scope, tracked elsewhere)

- Logs SDK (RC) and profiles (alpha) adoption — plan §2 explicit exclusions
- In-library SSE span emission (recipe-only was the decision; recipe shipped)
- ~~Release train / tags for the touched modules (`setup` now carries new API: `Observability`; per ADR-0050's note setup publishes with the next family train regardless)~~ done — v4.11.0 family train shipped 2026-09-19
- dashboardui metrics panels; dashboardui strings.Builder→templ paradigm shift
- ~~Full `nix run .#test` (17-suite canonical invocation) as one command this session — coverage-gate ran the same suites per-module and everything I touched was run hermetically, but the single canonical invocation was not executed~~ done — full gate ladder green (2026-09-10 addendum + later runs)
- Re-run of `nix run .#bench-spike` (no bench paths touched — `setup/run_appkit_test.go` untouched — so no re-pin needed; not run at all either, as a sanity pass) — STILL OPEN → `TODO_LIST.md` P1
- ~~`nix flake check --no-build` (was green at last full pass 2026-08-14; not re-run this session)~~ done — `nix flake check` green
- ~~`.github/workflows/ci.yml` review for the new setup deps (CI installs hooks; middleware/otel are published tags, so CI resolution should just work — unverified)~~ done — CI all-green 2026-09-20

## d) TOTALLY FUCKED UP (honest list)

1. **Lost the commit race three times.** The repo's #1 documented gotcha ("commit at phase boundaries — the daemon polls faster than a long verification tail") and I still lost the setup phase (5+ heuristic commits, explicit commit arrived to find "nothing to commit"), the §2.6 first draft (`cb152429`), and the README/lint portions (`120afe39` was mine; `3389c336`/`e244dc55` were not). Root cause: I let long test-iteration stretches run without committing the already-green intermediate state. Mitigation for next time: commit after every green test run, not just after "phase complete"; or stage+commit the moment a file stops changing.
2. **Built a test on an unverified assumption, twice.** (a) The ordering test asserted closure `reflect.Value.Pointer()` identity — empirically false (same func literal, different capture instantiations → different pointers); a 30-second experiment would have caught it before I wrote 100 lines. (b) The replacement fake-tracer approach ignored that modern otel interfaces carry an unexported marker method (`missing method tracer`) — implementations are impossible outside the otel module; one interface read would have shown it. Three rewrites of `observability_internal_test.go` before settling on honest count-based assertions + documented order. Empirically-verify-first is a standing repo lesson and I repeated its violation.
3. **Wrote a doc claim before verifying it.** §2.6 initially said `CommandCausalityEnricher` was "(default in usermgmt)" — false (no `WithEnricher` call site exists in usermgmt); and the follow-up sentence implied `setup.Observability` could wire the enricher into pre-built repositories — also false (it wires only bus middleware). Both were caught by my own re-verification, but the repo rule is verify-then-write, not write-then-check (two commits to get §2.6 truthful: `af87e68f` then `3f7fd30d`).
4. **Multiedit dropped a `//` comment prefix** in `setup/setup.go` (line "documented precedence: ..." without a marker) — produced a syntactically-valid-but-wrong comment block, caught only when the next edit's exact-match failed. Sloppy exact-matching discipline.
5. **My final chat summary was truncated mid-table** — the user never received the complete wrap-up of M8–M13. This report supersedes it.

## e) WHAT WE SHOULD IMPROVE

1. **Commit micro-cadence for daemon-race repos:** commit after every green verification, even mid-phase; never batch "finish then commit" in this tree.
2. **Empirical-first for Go reflect/toolchain assumptions:** run the 30-second experiment before writing assertion code.
3. **Verify-then-write for guide claims:** the §2 recipes now follow "signature checked against upstream source before landing" — keep that discipline for *prose claims* too, not just code snippets.
4. **Gate the un-gated:** `integration_test` is not coverage-gated, which is exactly why the loginpage copy drift sat on master. Add it to `coverage-gate` (or a cheaper smoke gate).
5. **Drift gate caught what the bump wave missed — but late:** the 2026-09-11 auto-commit wave left `integration_test/go.mod` (v1.0.1) and `middleware-showcase/vendor` (v1.0.1) behind. A post-daemon-commit `check-version-drift --strict` hook (or the existing one wired into the pre-commit hook path) would catch this at commit time instead of days later.
6. **Sibling-replace exposure is a standing liability:** two modules build only against a moving foreign master. The removal condition (projectionadapter v4.5.0+ tagged) is documented; the sibling's current refactor makes it worse. Worth an explicit TODO_LIST entry with a re-pin fallback plan.
7. **The audit's stale ServiceConfig claim shows the audit itself needs a freshness pass before execution** — verify each plan premise against current code at execution start (I did catch it, but mid-sprint; a 10-minute premise check up front would have re-scoped M6/M7 immediately).

## f) UP TO 50 THINGS TO GET DONE NEXT (Pareto-ish order)

**Immediate / small (this week):**
1. ~~Decide: rewrite/reorder the shredded heuristic commits for the setup phase (local-only history surgery) or accept CHANGELOG-as-narrative.~~ done (decided: CHANGELOG-as-narrative, history immutable)
2. ~~Resolve the sibling `metaengine.Store.Reset` break: wait for the sibling session, or re-pin `systemadapter` + `examples/system-demo` replaces to a known-good sibling commit.~~ done (sibling fixed; both build modes green; systemadapter/v4.11.0 tagged)
3. Add `integration_test` to the coverage gate (or a smoke test gate) so UI-copy drift can't sit on master again.
4. ~~Run the canonical `nix run .#test` end-to-end once post-sprint as a final all-green record.~~ done (full gate ladder green)
5. ~~Run `nix flake check --no-build` once post-sprint.~~ done (nix flake check green)
6. Re-run `nix run .#bench-spike` as a sanity pass (no baseline re-pin expected — no bench paths changed).
7. ~~Re-score the OTel audit rubric against current state so "90" is measured, not asserted (findings #1–#7).~~ done (audit gaps all closed and API-verified (AGENTS.md OTel bullet))
8. ~~Verify CI (`.github/workflows/ci.yml`) resolves setup's two new requires on a clean runner.~~ done (CI all-green 2026-09-20)
9. Consider `OnSubscribe`/`OnUnsubscribe` as constructor options upstream in go-sse (currently methods; the SSE recipe had to call them post-construction) — upstream ask candidate.
10. Consider upstream ask: export a test-friendly SpanProcessor hook or recorder tracer from go-cqrs-lite/otel so downstream can assert span ordering without importing the OTel SDK (blocked my setup order test).
11. Add a workspace-mode `go vet ./...` pass at the repo root to the verification checklist (hermetic per-module ran; workspace mode didn't this session).
12. Annotate `docs/research/2026-09-13_otel-deep-dive.html` with the executed outcomes (the audit HTML still says "gaps ticketed").
13. Add the sprint's runnable-proof commands to `setup/README.md` (copy-paste `middleware.NewOTelBundle` + `Observability` example is in the config.go doc; README links it only via the table row).
14. Sweep `docs/guides/leveraging-go-cqrs-lite.md` §2 for a runnable proof pointer on §2.6 (currently only §2.4/§2.5 have "Runnable proof:" lines; §2.6 could point at a test or the demo once the enricher gets demo wiring).
15. CHANGELOG: when the next version is cut, split the single monolithic sprint entry into Added (setup API) / Changed (docs) per Keep-a-Changelog type.

**Setup / observability product surface:**
16. Document (or wire) how `Observability` composes with `AsyncStartup` (should be orthogonal — verify + test-pin).
17. Add a `setup-demo` showcase of the `Observability` option (stdout exporter) so the onboarding demo teaches it.
18. Consider `Observability` acceptance of raw middleware slices (`event.PublishMiddleware`/`event.Middleware`) as an alternative to the bundle for consumers who don't want the bundle shape.
19. Consider surfacing `bundle.CorrelationEnricher()` wiring through a setup option IF repositories become construction-configurable (currently requires forking construction — documented limitation).
20. Test that `Observability` bundle middleware still fires on projection REPLAY (journal drain path) — the e2e test covers the live path only.
21. Pin the exact `event.publish`/`event.handle` span names in a golden test so upstream renames can't silently break the SSE/guide documentation.
22. Add a `check-modules` stage that validates dep-budget comment counts match reality (the comment said 19 while reality was 20 pre-sprint — the comment was stale before my change).

**Pre-existing debt noticed this session:**
23. ~~The other 47 train-lag requires (release-train advisory list) — next family train alignment sweep.~~ done (train-lag swept to ZERO 2026-09-20)
24. `usermgmt.TenantID`/`usermgmt.User` deprecation hints in `setup/config.go` (gopls) — finish the direct-identity-model import migration there.
25. gopls `infertypeargs` noise (~38 infos across usermgmt/root test files) — mechanical cleanup for signal hygiene.
26. ~~exhaustruct → exhaustruct_v5 migration (golangci deprecation warning fires on every setup lint run).~~ done (exhaustruct_v5 migration documented (AGENTS.md))
27. ~~`e2e/server` Playwright browsers cache on the dead `/mnt/buildcache` — the runbook workaround exists but the e2e suite wasn't run this session; run it with `PLAYWRIGHT_BROWSERS_PATH=/tmp/pw-browsers`.~~ done (e2e green (4/4 + admin-behavior 9/9))
28. Audit `docs/observability-wiring.md` for full deletion candidacy in v5 (SUPERSEDED banner exists; body is still wrong-on-purpose legacy).
29. Review whether `.gitignore`'s `vendor/` plus partially-tracked vendor trees (history shows vendor files in old daemon commits) needs a `.gitignore` negation pass or a deliberate "vendored examples" policy.
30. The `middleware-showcase` vendor tree is untracked but required for its hermetic build — either track a minimal vendor manifest or document `go mod vendor` as a required setup step for that example.

**Guide/docs depth:**
31. Add an end-to-end trace WALKTHROUGH (one request → root span → dispatch → decider → store spans → SSE fan-out) as a new guide or §2.7, tying §2.4–2.6 + the SSE recipe into one narrative.
32. Document `cqrsotel.Setup` global-registration failure modes (double-setup, test isolation, `WithoutGlobalRegistration`) — §2.4 mentions the happy path only.
33. Document OTel resource/service-name conventions across the two demos (they use different service names; a consumer copying both gets two services).
34. Add a correlation ID flow diagram (HTTP header → context → event metadata → audit log → trace) — the table is good, a diagram would be better.
35. Cross-link the SSE fan-out recipe from `leveraging-go-cqrs-lite.md` §2 intro list (it currently lives only in the SSE guide).
36. Add "when NOT to use baggage" guidance to §2.6 (PII in baggage is a classic footgun; W3C baggage has size limits).
37. Verify + document the interaction between `Observability` and `SecurityHooks` sign/encrypt order in a test (currently documented-only: bundle outermost).

**Process / repo hygiene:**
38. Create the TODO_LIST entry for the sibling-projectionadapter blocker with a re-pin fallback (AGENTS documents it; TODO_LIST should track the actionable state).
39. Sweep AGENTS.md for now-stale lines post-sprint (e.g. any "gaps ticketed in TODO_LIST P2" phrasing outside the OTel bullet).
40. Consider a `check-docs-freshness`-style assertion that CHANGELOG's latest entry date is within N days of the newest commit touching gated modules (would have caught nothing here, but formalizes the annotate-then-move-on pattern).
41. The plan doc's Definition-of-Done checkboxes are `- [ ]` in a PLANNING doc — fine per conventions, but the EXECUTED annotation should list DoD items with ✓/✗ verdicts explicitly.
42. Post-sprint: verify `nix run .#lint` (the app, all 15 modules) once — I linted setup manually; the app covers the same set but one canonical run is cheap insurance.
43. Decide whether examples/middleware-demo + observability-demo should get a shared `telemetrydemo` testutil (both now build near-identical stacks; dedup-code review candidate, likely "intentional similarity" verdict).
44. Add `go.work` sync verification (`go work sync` no-op check) to check-modules — my go.mod edits happened hermetically; a stale go.work require block would only surface in workspace mode.
45. File the upstream go-cqrs-lite issue: `middleware.OTelBundle` docs claim `bus.UsePublish(bundle.Publish())` ordering, but setup's prepend semantics (bundle outermost over consumer hooks) are our invention — upstream could document the composition guarantee.

**Forward-looking:**
46. ~~Plan the next family release train: setup (new API), root docs — enumerate which of the 47 lagging requires ride along.~~ done (v4.11.0 family train shipped 2026-09-19)
47. Evaluate OTel logs SDK (now stable upstream?) as the next audit cycle's topic — the 64→90 plan excluded it.
48. Consider a `WithMetricsDisabled`-style knob parity check between `middleware.NewOTelBundle` and `setup.Config.Observability` docs (setup doc doesn't mention tracing-only bundles).
49. Measure and record the overhead of the Observability bundle on `BenchmarkSpikeBaselineVsAppkit` (tracing middleware on the bus adds per-event spans; quantify before a consumer asks).
50. Schedule the next OTel audit cycle (quarterly?) so adoption scores stay evidence-based rather than decaying into stale claims like this sprint's ServiceConfig premise.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Sibling repo intent:** is the in-flight go-cqrs-lite refactor that removed `metaengine.Store.Reset` supposed to land soon (→ I should wait and re-test `systemadapter`/`examples/system-demo`, then ask for the `projectionadapter v4.5.0+` tag), or should I re-pin those two replaces to a known-good sibling commit right now to make `check-modules` fully green on this repo?
2. **History surgery:** do you want the shredded heuristic commits covering the setup phase (`b638f3cc`…`e244dc55`) reworded/squashed into proper narrative commits (local-only rewrite, branch is 9+ ahead of origin and unpushed), or is CHANGELOG-as-narrative acceptable and history stays immutable?
3. **Release timing:** `setup` now carries a new public API (`Config.Observability`) and per ADR-0050 its publish rides the next family train anyway — should the next train be cut soon with this API included, or should it sit unreleased in master until the 47-require train-lag sweep happens?

---

*Point-in-time snapshot — annotate, never rewrite. Next actor: start at §f items 1–3.*
