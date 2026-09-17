# Status Report — Library-Adoption Plan Execution (Tier 1%/4%/20% start)

**Created:** 2026-09-17 21:04 CEST
**Session scope:** executing `docs/planning/2026-09-17_13-42_pareto-execution-plan-sibling-library-adoption.md` (created + pushed earlier today) after the "NOW GET SHIT DONE" order.
**Environment reality this session:** a second agent session is working the SAME tree concurrently (templ-components × dashboardui wave, narrative commits `56dcbfdd`, `81088b64`), plus an auto-commit daemon AND an auto-push mechanism. Everything below is written against that backdrop.

---

## a) FULLY DONE

### Tier 1% (M01+M02) — go-etag correctness fix — 100% ✅
- **F01–F06:** All hand-rolled `If-None-Match` exact-string checks replaced with `etag.ParseETag` + `etag.MatchesIfNoneMatch` (RFC 7232 weak comparison, wildcard, validator lists, lenient malformed handling). Three functions cover all five production sites: `serveImmutableJSON` (event catalog + OpenAPI spec), `serveJS` (htmx.js + extensions + sync-worker + sync-client), `ProjectionStatusHandler`. `adminui/assets.go` deliberately untouched (already RFC-correct via http.ServeContent).
- **F02:** go-etag v0.3.1 promoted from `// indirect` to direct require in root go.mod (hermetic tidy, verified).
- **F08–F10:** New `conditional_get_spec_test.go` — a reusable RFC 7232 spec harness (wildcard `*`→304, list→304, weak `W/`→304, mismatch→200, malformed→200) wired into 4 handler families = 25 spec cases, all passing under `-race`.
- **F07/F11/F12:** Root build + vet + full race suite green (4.0s); golangci-lint 0 issues; cqrs-lint green.
- Committed (content absorbed into daemon commits `f0c7d2ec` + `9a1f025b`, byte-verified intact via rg checks).

### Tier 4% (M04+M05+M06) — SSE drain, retry hint, httputil currency — 100% ✅
- **M04:** `setup/bundle.go` `Close()` now drains the SSE hub via `Broadcaster.Shutdown(5s ctx)` before the abrupt `Close()` — server restarts no longer drop queued in-flight events. Shared-hub design means the DataStar feed drains with the same call. Timeout path logs a warning and falls through (shutdown never hangs). Two new tests: queued-event delivery across Close; deadline honored under buffer overflow.
- **M05:** `retry: 5000` reconnect hint now sent before the first event by BOTH `transport.ServeDomainEvents` and `Broadcaster.ServeSSE` (via `sse.WriteRetry`, constant `transport.DefaultRetryHintMillis`). Browsers back off after server restarts instead of stampeding. Wire-order pinned by two tests (transport plain-test + Ginkgo It asserting `HavePrefix("retry: 5000\n\n")`).
- **M06:** httputil v1.2.0 sweep COMPLETED — the concurrent session had already done all modules except `integration_test` (still v1.1.1); bumped it hermetically (tidy+build+vet green) + hermetic spot-checks on usermgmt/adminui/setup + `check-release-train --refresh-cache` green (0 unpublished requires). The `scripts/testdata/verify-tag` v0.12.0 entry correctly left alone (frozen poison fixture).
- Committed: `68750ef8` (fix(sse): drain + retry hint) and `361135d5` (chore(deps): httputil sweep) — both with full narrative messages, `--no-verify` + justification (gate still broken), already PUSHED to origin by the auto-push mechanism.

### Tier 20% partials — 3 items ✅
- **M03:** `ProjectionStatusHandler` now wraps its inner handler in `etag.New(DefaultETagConfig{SkipIfPresent: true})` — the go-etag middleware owns conditional evaluation (buffer → adopt handler ETag → weak-match 304). Inner handler extracted to `serveProjectionStatus`. New test: stale ETag after data change gets a fresh 200 + new ETag. All 8 projection-status tests green under race. (Committed via daemon `372e3271`, verified intact.)
- **M11 (pulled forward from tail):** systemadapter + examples/system-demo had a WRONG-PATH replace (`cqrs-htmx => ../` instead of `cqrs-htmx/v4 => ../`) — hermetic builds silently resolved root v4 from the proxy instead of the local tree. Fixed in both files.
- **NEW (unplanned but necessary):** systemadapter's hermetic build was BROKEN by sibling go-cqrs-lite master drift (projectionadapter now calls `metaengine.Store.Reset`, a ResetResult API that published metaengine v4.13.0 lacks). Added temporary sibling-master `metaengine/v4` replaces with removal-condition comments to both modules (matches the existing projectionadapter replace pattern). systemadapter hermetic tidy+build+vet GREEN after the fix.

---

## b) PARTIALLY DONE

- **check-modules gate:** ran once → failed on setup (TRANSIENT — concurrent session's mid-flight state; passes manually now) and systemadapter (REAL — fixed by the replace work above). NOT yet re-run to green. Required before final push.
- **Root go.mod directive stability:** restored `go 1.26.7` THREE times; re-bumped to `go 1.27.1` FOUR times by the concurrent session's tooling. Root cause fully diagnosed (see d/e). Currently at 1.27.1 again = workspace-mode builds broken again. NOT durably resolved (see questions in g).
- **M07 (buildflow binary rebuild):** plan assumed `nix run .#reinstall` — NO such flake app exists. Real mechanism needs locating (likely `buildflow upgrade` or a nix build of the buildflow package). Not started.
- **Tier 1% F15 / M18 / F53 / F91 (AGENTS.md + CHANGELOG + TODO_LIST docs wave):** NOT done. The critical AGENTS.md gotcha entry (go-appkit→root directive propagation mechanism) exists only in my commit message `68750ef8`, not yet in AGENTS.md where the other session will see it.
- **Push state:** my 2 narrative commits already reached origin (auto-push); 13 LOCAL commits remain unpushed (mostly the concurrent session's dashboardui/templ work + heuristic mixtures including my absorbed M03/M11 files).

---

## c) NOT STARTED (from the plan)

- **M07/M08:** buildflow rebuild + 51 go-structure-linter findings triage (pre-commit gate still deterministic-fails on 103 pre-existing findings; every commit this session used documented `--no-verify` fallback).
- **M12/M13:** dashboardui SSE-hub health card; setup OnSubscribe/OnUnsubscribe gauges. HIGH COLLISION RISK — the concurrent session is actively editing exactly dashboardui (`handler_overview.go`, `stats.go`, `layout.go` in `edfdcb2c`) and setup.
- **M14:** ssetest.RequireDataJSON adoption (12 sites).
- **M15–M17, M19–M27 (tail):** ETag metrics hooks, etagclient recipe, docs wave B, If-Match recipe, benchstat 304-vs-200, e2e retry assertion, sweeps, upstream tracking, fixture policy, vulnix triage, rubric/checklist docs, roadmap fuel.
- **M09/M10:** go-mod findings triage (26+26), phantom `./v4` root-cause.

---

## d) TOTALLY FUCKED UP (honest list)

1. **The go-directive tug-of-war (biggest self-inflicted time sink).** I restored root go.mod to 1.26.7 three times; it re-bumped to 1.27.1 four times. I treated occurrences 1–2 as accidental one-offs and only root-caused after occurrence 3 collided with my system-demo verification. Root cause: `go-appkit/go.mod` (the sibling repo the concurrent session's setup spike replaces) says `go 1.27.1` — ANY workspace-mode tidy/sync propagates that into cqrs-htmx root's directive, breaking every workspace build against the pinned `go_1_26` flake toolchain (GOTOOLCHAIN=local, no go1.27 binary exists on the machine). Telling detail I found too late: ALL EIGHT go-appkit submodules are at 1.26.7 — only its root module says 1.27.1, i.e. almost certainly the SAME accidental-propagation class there, and probably fixable with a one-line change in that repo. **What I should have done:** pattern-match occurrence 1 (known accidental class per AGENTS.md daemon history) → investigate the SOURCE before the first restore → one fix instead of three restores + a war.
2. **Repeated narrative-commit losses to the daemon.** My Tier 1% file set, M03 middleware, and the M11/replace fixes all landed inside `chore: auto-commit` heuristic commits MIXED with foreign work (`372e3271` contains my projection_status_handler.go AND their flake.nix; `edfdcb2c` is entirely theirs). Three detailed commit messages were written and lost (one commit failed outright because the daemon had absorbed the files mid-command; its log file even vanished from /tmp). **What I should have done:** commit after EVERY file edit (not after each test batch), and treat "status clean when I expected dirty" as a daemon-absorption alarm.
3. **One malformed multiedit** on sse_broadcaster_test.go (second hunk removed a newline instead of adding the new It block) — caught immediately by viewing the result, fixed in the next edit. Cost: one wasted round trip.
4. **The PIPESTATUS-family trap bit me once more:** `go build ... | head -8; echo $?` printed `SYSAD=0` while the systemadapter build had ACTUALLY FAILED (exit code was head's). This is a DOCUMENTED gotcha in AGENTS.md and I still ran into it — I caught it only because I cross-checked the error text. **What I should have done:** `cmd > f 2>&1; echo $?` unconditionally, per the repo's own hard rule.
5. **Verification-order mistake on system-demo:** I added the `cqrs-htmx/v4 => ../..` replace (M11) BEFORE confirming root's directive state, so the tidy failed on a condition I had myself just observed and not controlled. Sequence control, not knowledge, was the failure.

---

## e) WHAT WE SHOULD IMPROVE

1. **Commit per file-edit, not per test-batch** — the daemon polls faster than any verification loop. Smallest possible phase = one file + its direct test.
2. **Root-cause before reverting, ALWAYS** — even when the foreign change looks like a known accidental class, one `git log -S` + sibling check saves three restore cycles.
3. **Never trust `cmd | filter; echo $?`** — the repo's hard rule exists because BOTH humans and agents keep tripping it. Capture to file first, then inspect.
4. **Cross-session collision protocol for feature work:** before touching dashboardui/setup (the concurrent session's active area), check `git log --oneline -5 -- <dir>`; defer if their wave is mid-flight.
5. **Go-directive drift guard (new gate idea):** `check-go-toolchain.sh` already fails when go.work > flake; it should ALSO fail when any member go.mod directive > go.work's (would have caught this whole class mechanically at the first occurrence).
6. **The `--no-verify` fallback is doing heavy lifting** (every commit this session). Until M07/M08 repair the gate, each commit message must keep carrying the justification — this held, keep it.
7. **AGENTS.md write-latency:** the go-appkit propagation mechanism is commit-message-only right now; the other session reads AGENTS.md, not my commit log. Memory updates must happen at discovery time, not phase end.

---

## f) NEXT 50 THINGS TO GET DONE

*Ordered: blockers → remaining Tier 20% → tail (mirrors the plan; new discoveries appended).*

1. Decide + execute the go-directive endgame (see g-Q1): restore 1.26.7 + instant commit, OR accept 1.27.1 if a toolchain bump is coming.
2. Write the AGENTS.md gotcha: "workspace tidy/sync propagates go-appkit's 1.27.1 directive into root; all 8 go-appkit submodules are 1.26.7 — its root directive looks accidental; fix at source or bump flake deliberately."
3. Re-run `nix run .#check-modules` to GREEN (validates my systemadapter/system-demo replace fixes + setup transient cleared).
4. Locate the real buildflow binary refresh mechanism (plan's `.#reinstall` doesn't exist) → M07.
5. M07: rebuild + reinstall buildflow binary; verify the stale-binary preflight warning is gone.
6. M08: export the 51 go-structure-linter findings to a triage file.
7. M08: classify batch 1 (fix / suppress-with-reason / defer) + apply.
8. M08: classify + fix batches 2–3; re-run the findings gate; record the delta.
9. M14: inventory the 12 hand-unmarshaling SSE test sites.
10. M14: adopt `ssetest.RequireDataJSON` batch 1 (6 sites).
11. M14: batch 2 (6 sites) + run the SSE suites.
12. M18: CHANGELOG entries for tiers 1% + 4% + landed 20% items (append-only).
13. M18: HARVEST audit §f items into TODO_LIST.md; speculative ones into ROADMAP.md.
14. M18: cross-link the two 2026-09-17 deep-dive reports (one line each).
15. M18: annotate AGENTS.md httputil version mentions as dated historical records.
16. M18: post-bump VERIFY of every version claim in plan + audit report (httputil v1.2.0 now true everywhere).
17. M12/M13 (GATED on g-Q3): dashboardui SSE-hub health card (payload struct + core/ bridge + render + CSP-safe test).
18. M13: setup OnSubscribe/OnUnsubscribe gauges behind Observability + gauge-wiring test.
19. M13: decide + implement (or document declining) hub summary in `/health` payload.
20. F51: full tier-20% gate run (build + test + lint + coverage-gate) → sub-phase commits.
21. M09: export go-mod-ignore-check (26) + gomod-check (26) findings; triage.
22. M09: fix mixed direct/indirect require blocks (go.mod:44, samber-do-demo:94).
23. M09: middleware-showcase vendor-dir ignore/commit decision per repo convention.
24. M10: capture hook-environment module state (go env, GOWORK, replaces inside a buildflow run).
25. M10: minimal repro of the phantom `./v4` golangci resolution; fix or file evidence-backed issue.
26. M15: wire ETag `On304`/`OnETagGenerated` hooks into logging/metrics (catalog + status endpoints) + firing test.
27. M16: `etagclient.NewTransport` recipe section + integration test against `/events/catalog` (second GET = 304).
28. M17: retry-hint section in sse-and-datastar guide (value, chosen 5000 ms, escape hatch).
29. M17: `AllowPrivateNetwork` opt-in guidance (LAN deployments; default-off).
30. M17: adminui ServeContent-vs-middleware decision note (stay on ServeContent; rationale).
31. M19: If-Match lost-update recipe sketch (`MatchesIfMatch`) for admin command endpoints.
32. M19: research note — does HTMX polling send If-None-Match?
33. M20: benchstat 304-vs-200 on the catalog endpoint per the canonical 5×2s pattern; commit machine-pinned baseline artifacts.
34. M21: e2e — assert `retry:` reaches the browser; extend the sticky-ID reconnect test; run with `PLAYWRIGHT_BROWSERS_PATH` guard.
35. M22: sweep loginpage + health for hand-rolled conditional-GET logic.
36. M22: rerun `cqrs-lint` across modules post-adoption; fix/nolint findings.
37. M22: rerun coverage-gate; confirm root ≥90%, usermgmt ≥74% hold with the new tests.
38. M23: go-etag upstream — typed-error release tracking + adoption-when-published task.
39. M23: go-sse upstream — Unreleased items; track release cut.
40. M23: ROADMAP entry — Vary-aware cache demand signal from the audit.
41. M24: decide + document the verify-tag testdata fixture policy (frozen vs tracking); apply.
42. M25: vulnix exposure triage — map store-closure CVEs to actual runtime surface.
43. M26: adoption-score rubric doc + sibling-library audit checklist + version-drift ritual line for AGENTS.md.
44. M27: AGENTS.md go-etag adoption note post-landing; datastar retry-hint consistency patch; WPT corpus eval notes; family-audit scheduling; polled-panel 304 wrap-up.
45. Fix the pre-existing `ServeSSE` heartbeat goroutine leak (not joined before handler return — net/http violation noticed during M05; unrelated so untouched, needs its own change).
46. Consider a `WithSSERetryHint(ms)` option if a consumer ever needs a non-5000 hint (currently hard-coded const by plan guard; note as possible API).
47. Add an OpenAPISpecHandler-specific spec test (it shares `serveImmutableJSON` so it's covered transitively; a direct test would pin the public entrypoint).
48. Tie the two wire tests to the constant: assert `transport.DefaultRetryHintMillis == 5000` so the hint value can't drift silently from the tests.
49. Sleep-based sync in the new retry/drain tests (50ms) — replace with event-driven signaling if CI ever flakes.
50. FINAL (F96): full gate run (build, test, lint, coverage-gate, check-modules, check-templates) → final commit → push all 13+ local commits (includes the concurrent session's — see g-Q2).

---

## g) QUESTIONS I CANNOT ANSWER MYSELF

**Q1 — go-appkit's `go 1.27.1` directive: deliberate or accidental?** All 8 of its submodules say 1.26.7; only its root says 1.27.1; it landed in a 37-file daemon heuristic commit. If accidental → the durable fix is a one-line downgrade THERE (or by whoever owns that session), and I keep restoring cqrs-htmx root to 1.26.7. If deliberate (a toolchain bump is being prepared) → the flake's `go_1_26` pin must move too, and I should STOP restoring 1.26.7 and let the repo move to 1.27 wholesale. Which is it?

**Q2 — Push now or wait?** 13 local commits are unpushed; most are the concurrent session's dashboardui/templ wave plus heuristic mixtures that also contain my absorbed M03/M11 work. Pushing publishes their half-finished wave under my push. My earlier commits already reached origin via the auto-push mechanism anyway. Push everything now, or hold until the concurrent session's wave lands?

**Q3 — Collide or defer on M12/M13?** The remaining Tier-20% feature work (dashboardui SSE-hub health card, setup gauges) targets exactly the files the concurrent session is editing right now (`dashboardui/stats.go`, `handler_overview.go`, `layout.go`, setup internals). Do I proceed and let the daemon sort merges, or defer those two until their templ-components wave commits are done?
