# Status: Fleet Library Utilization Audit — Delivery + Self-Review

> **ANNOTATED 2026-11-06** — ~~this report~~ **SUPERSEDED** by [`2026-11-06_22-03_fleet-plan-wave-5-partial-render-presets-charts-landed-a22-mid-flight_status.md`](2026-11-06_22-03_fleet-plan-wave-5-partial-render-presets-charts-landed-a22-mid-flight_status.md), which the 22-03 report carries the fleet-plan execution state the 00-27 audit delivery kicked off; the audit artifact itself lives in docs/review/libs/ and AGENTS.md's fleet-wide utilization bullet. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-06 (round-24 (2026-11-06)).**

**Session:** 2026-11-05 evening → 2026-11-06 00:27 · **Author:** Crush · **Series:** [docs/review/libs](../../review/libs/index.html)

**Scope line — VERIFIED this session:** 19 per-library reports + `index.html` exist and are structurally valid (python HTML-parser: balanced tags, all anchors + 19 md links resolve); docs gates green at audit HEAD (`check-docs-links` 1,230 links, `check-docs-corpus-budget --ratchet` after re-true, `check-docs-tail-budget --ratchet`, `check-docs-freshness`); every pin-vs-latest claim verified live via `git ls-remote` (2026-11-05); every report's "how to verify" block replayed literally (go-etag gap = 11 commits, templ = 261 .templ files, WithOnDrop/BroadcasterHealth prod = 0, deprecated-alias dogfooding = 0, error-family unadopted symbols = 0 — all matched).
**NOT verified:** visual/browser render of `index.html` (no chromium on host; firefox/chrome are shell-banned — structural validation substituted); the full `check-modules` aggregate at final HEAD (targeted docs gates only; docs-only change); daemon auto-commit content not diffed (gotcha 51 class); go-cqrs-lite "≈80 derived" is arithmetic over the 2026-11-03 audit + four landed waves, NOT a re-audit.

---

## a) FULLY DONE

1. **19 utilization reports** at `docs/review/libs/*.md` (979 lines): capability inventories with Fully/Partial/Missed/N-A grades (every N/A carries its reason), score derivations with inputs, version currency, should-we-use verdicts, verify blocks, prior-report citations.
2. **`docs/review/libs/index.html`** — html-report-kit dashboard template (copied, never transcribed), 19-row verdict table genuinely sorted to match its caption, stat cards, fleet issues, strengths, dependency posture, 5-step action plan.
3. **Version truth, live-verified:** 18/19 entities at latest tag; the one lag (go-etag v0.6.1 → v0.7.0, tagged 2026-11-03) verified as 11 chore/CI/docs commits — zero functional delta.
4. **Symbol-level usage census** per used library (e.g. `NewRejection` ×200, `sse.Event` ×242, httputil CSRF suite ×100+, templ display ×70 distinct symbols).
5. **Prior series incorporated, not duplicated:** go-cqrs-lite 73/100 → ≈80 derived (CBOR/ADR-0059, dedup.Ring, otter view cache, snapshot guard landed post-audit); templ-components 97/94/95; health bridge 9/10.
6. **Housekeeping:** corpus baseline 44,457 → 45,575 with ledger row; `docs/LIBRARY_INTEGRATIONS.md` (stale 2026-05) banner-marked SUPERSEDED; `docs/research/README.md` series row; AGENTS.md memory line + fixed the stale "CBOR decision still open" claim (ADR-0059 accepted).
7. **Honest catches during the run:** Server-Timing is the W3C spec (not an RFC — my wrong guess was caught and fixed pre-ship); a false test-file claim retracted; one upgrade-CLI `os.WriteFile` counter-example found and precisely documented instead of glossed.

## b) PARTIALLY DONE

1. **Render verification of index.html** — structural checks pass; visual/browser check blocked (see d/e).
2. **go-cqrs-lite standing** — derived, not re-audited (deliberate; the 3-day-old 47-row audit is the scored truth).
3. **Action items live in reports/index only** — NOT yet harvested into TODO_LIST.md (pending your routing decision; question g.2).
4. **Aggregate gate** — targeted docs gates green; full `nix run .#check-modules` not run at the final HEAD of this session.

## c) NOT STARTED

1. TODO_LIST.md / ROADMAP.md harvest of section (f).
2. The action items themselves (go-etag bump, SSE drop observability, dashboard nonce, dogfood showcase…).
3. go-daemon sd_notify "running under systemd" docs note in `docs/guides/production-readiness.md`.
4. Next docs-health archive round (live tail is at budget).
5. The separate `docs/reviews/*_brutal-self-review.html` artifact — folded into this report per your one-file instruction (deviation from the self-review skill's default output path is instruction-driven).

## d) TOTALLY FUCKED UP (all caught and fixed in-session; listed so the process lessons survive)

1. **Score arithmetic churn:** templ-components shipped 89 → "fixed" to 88 → re-fixed to 89; go-appkit 80 → 75 → 80. Root cause: I wrote derivations from intended counts instead of counting the tables mechanically first. Final numbers are grep-verified against the emoji statuses; two wrong intermediate states still hit disk.
2. **Thinking artifact in prose:** `go-cqrs-lite.md` shipped with a literal `+ 0.5×?` scratch fragment in its derivation line. Caught late by the consistency pass.
3. **Baseline bumped too early:** corpus baseline was trued to 45,572, then my own housekeeping edits grew it +3 → re-trued to 45,575. Self-inflicted second cycle; bumps must come LAST.
4. **Two unverified external claims were written before verification:** "httpspec_compliance_test.go pins the header contract" (false — zero server_timing refs there) and "RFC 9111" for Server-Timing (wrong — W3C Server Timing API). Both caught by verification minutes later, but the discipline (verify-THEN-write) slipped under writing momentum.
5. **Sub-agent plan failed at launch:** 3 of 4 research agents died to rate limits; recovered by serial direct research (output unaffected, throughput hit).

## e) WHAT WE SHOULD IMPROVE

1. **Mechanical-first scoring:** run the row/status counters BEFORE writing any derivation — would have prevented d.1 entirely. Make it a fixed step of the library-deep-dive loop.
2. **Baseline bumps last:** finish every corpus-touching edit, bump once, ledger once.
3. **Verify-then-write as a hard gate:** no sentence about an external artifact lands before its primary-source check (d.4 proved the slip is real, not theoretical).
4. **Pre-flight the render path before choosing an HTML deliverable** — the browser ban made the done-bar screenshot item unachievable from the start; asking up front beats discovering at the end.
5. **Harvest discipline:** a report that generates action items should route them in the same session, not leave routing as a question.

## f) Next tasks (ranked; 40)

**Wave 1 — trivial/high-certainty:** 1. Bump go-etag v0.6.1→v0.7.0 (chore-only; `bump-dep` sweep). 2. Wire `sse.WithOnDrop` (log/metric) at the root Broadcaster seam. 3. Surface `sse.BroadcasterHealth` beside adminui sync indicator / dashboardui. 4. Thread a nonce (`WithNonce`/`WithNonceExtractor`) through `health.NewDashboard`. 5. Add `RenderPartialOrFull` + `OOBHTML` showcase page to `examples/basic`. 6. Build `RecommendedSecurityMiddleware()` via `httputil.NewMiddlewareStack` (inherit validation).
**Wave 2 — error-family stretch:** 7. `Audience`-filtered `SafeDetail` in httperr (design first). 8. `LogError`-style classified logging seam in middleware. 9. Survey double-wrap sites for `WrapOnce`. 10. Message templates (`RegisterTemplate`) for consistent SafeDetail strings.
**Wave 3 — codec stretch:** 11. `AutoDetect` for honest sync-pull stamps (ADR-0056 contract check first). 12. `TranscodeToJSON` legacy-client serve path (only if asked for). 13. `EncodePooled`/buffer pooling behind a bench-spike. 14. `ObservableCodec` metrics in observability-demo. 15. `Diagnose` in dashboardui's CBOR payload inspector. 16. dashboardui charts panel (first templ-components/charts consumer).
**Wave 4 — go-cqrs-lite open tail:** 17. systemcatalog-derived event catalog + drift gate (blocked: upstream tag ask — file it). 18. commandlifecycle wiring + FailureLog/RetryCount dashboardui views. 19. SQLite DLQ default for SQL setups. 20. Checkpoint batching (`WithCheckpointEvery/Interval`). 21. Engine-breadth presets in systemadapter (pebble/bbolt/postgres/duckdb/turso). 22. Rename 3 deprecated `meta.CausationID` dashboardui reads. 23. v5 exposure: disposition the 68 ADR-0051-gated SQLViewStore findings when cluster-1 fires.
**Wave 5 — bridges:** 24. go-health `WithEvaluationHook` audit integration. 25. go-health metrics section wiring. 26. `VersionHandler` /version endpoint in health bridge. 27. `NewChecks` batteries evaluation for consumer dependency checks. 28. appkit lifecycle-hook forwarding in setup (unified drain story). 29. go-health-dashboard option pass (`WithRateLimit` etc.) — document direct-consumer path in bridge README.
**Wave 6 — v5 prep:** 30. Inventory + delete root deprecated re-exports (CSRF/rate-limit/server-timing/security-headers/idempotency aliases) behind the v5 train. 31. Keep admin-demo as canonical go-idempotency direct-usage demo through the cutover.
**Docs/infra:** 32. HARVEST this section into TODO_LIST/ROADMAP (question g.2). 33. Run full `nix run .#check-modules` at HEAD. 34. Verify daemon auto-commit content for this session's files (gotcha 51). 35. Sanctioned render verification of index.html (question g.1). 36. go-daemon sd_notify systemd note in production-readiness guide. 37. `cmd/cqrs-htmx-upgrade` atomic-write hardening (single `os.WriteFile` site). 38. httputil `Logging` vs `RequestLoggingSlog` consolidation evaluation (N/A-with-reason today). 39. Next docs-health archive round (tail at budget; oldest reports first). 40. Eventually archive (not just banner) `docs/LIBRARY_INTEGRATIONS.md` once the new audit proves durable.

## g) Questions I cannot answer myself

1. **Render verification:** is there a sanctioned browser path on this host for local-file screenshots (an allowed chromium, a hyperframes browser tool install, or a policy exception) — or should the library-deep-dive done-bar screenshot item be waived for local-file renders? I cannot verify visual rendering myself; structural checks are all I have.
2. **Harvest routing:** should section (f)'s 40 items be routed into TODO_LIST.md/ROADMAP.md now (docs-health HARVEST), and if so at what rigor — all 40, or only waves 1–2 (the rest as ROADMAP fuel)?
3. **go-etag bump timing:** standalone tiny `bump-dep` wave now, or batched into the next family sweep? Both are defensible; the train rhythm is your call.

---

**Observed, untouched (not mine):** pre-existing working-tree changes at session start — `adminui/assets/admin-tw.css` + `adminui/tailwind.css` modified (the parallel dropdown-fix session, see the 22-31 report) and untracked `e2e/probe-explicit.mjs` (the 23-57 session). Left exactly as found.

**Report chain:** fleet audit → `docs/review/libs/index.html` (+19 md) · this report self-reviews that delivery. **Waiting for instructions.**
