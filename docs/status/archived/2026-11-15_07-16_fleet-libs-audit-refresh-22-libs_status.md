# Fleet-Libs Audit Refresh (22 Libraries) — Session Status

> **ANNOTATED 2026-11-16** — ~~this report~~ **SUPERSEDED** by [`2026-11-15_14-37_comprehensive-status-update-plan-tail-execution_status.md`](2026-11-15_14-37_comprehensive-status-update-plan-tail-execution_status.md), which the 22-lib refresh shipped into the living audit index docs/review/libs/index.html; its follow-through rows live in the fleet TODO row. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-16 (round-23 (2026-11-16)).**

> Session 2026-11-14 evening → 2026-11-15 07:16 · Thread: owner blanket directive "re-audit all LarsArtmann libs, SUPERBLY, latest versions, reports + index"
> Series: `docs/review/libs/` (supersedes-in-place the 2026-11-05 round; index regenerated)

## §a Context and scope

Owner directive: verify every named LarsArtmann library is used/leveraged to the max, write `docs/review/libs/<LIB>.md` per library plus an `index.html` covering all; use latest versions; use subagents smartly. The 2026-11-05 round covered 19 libraries; this round extends to 22 (three NEW: go-datastar, httputil/endpoints, cmdguard), refreshes all 19 existing reports against 549 commits of drift (2026-11-05 → 2026-11-14), and re-verifies every version claim against live tags (fetched 2026-11-14; treadmill re-run twice, second time with a refreshed cache that surfaced a NEW upstream lag).

Execution shape: one parent (me) + 12 subagents in three waves (6 + 5 + 1 held back for capacity), each owning disjoint report files; parent did version-truth sweep, drift analysis, index regeneration, and all repo-mechanics (corpus, tail, memory, gates, screenshot).

## §b Fully done (receipts)

1. **Version-truth sweep, all 22 targets** — fetched tags in every sibling checkout, enumerated submodule tag streams (httputil endpoints/etagmetrics/server_timing; go-etag server/entitytag; go-datastar broadcast/static; templ-components 6 trains; go-appkit 12 trains; go-cqrs-lite ~100 prefixes incl. the nested metaengine restructure), diffed against every go.mod pin in the 30-module workspace.
2. **All 22 reports at audit-date 2026-11-14** — 19 refreshed, 3 created (go-datastar, httputil-endpoints, cmdguard), every report keeps the fleet-series format, every verify block re-run literally by its author-agent. Scores: go-appkit 80→100, go-health-dashboard 67→100, templ-components 89→95, go-sse 72→94, go-cqrs-lite ~80→90.5, go-idempotency 88 held, cqrs-htmx self 83/~92 held (inventory 12→15 rows), go-etag 83 held (5 prod sites, partial-render ETag new), go-error-family 65→80, httputil 73→79, go-codec 44→71, go-datastar 71 (new), go-health 65 held (bridge 9/10 re-verified), server_timing wind-down verdict unchanged; 8 correctly-not-adopted verdicts re-proven at zero imports (cmdguard new, latest v5.1.0; httputil/endpoints new with a reasoned no-TODO verdict).
3. **New findings, source-verified** — (a) treadmill strict-lag-0 RED: go-cqrs-lite metaengine v4.23.0→v4.24.0 AND metaengine/sqliteengine v4.8.0→v4.9.0, both tagged after the M17 sweep, v4.24.0 nests the module into a nested directory with dual tags (not a bare bump-dep); (b) M2M bot path mints no idempotency key and CommandIdempotency is absent from the setup default factory list; (c) examples/datastar-demo serves the SDK unversioned behind a 24h cache (upstream ScriptTagVersioned/ETagFor unadopted, adapter does not re-export them; datastartest submodule unused); (d) templ-components popover CSS still unfixed at v1.27.0 (91 unreleased commits, none touch it); (e) two consumer→library loops CLOSED this cycle: httputil#48 (CSRF QUERY safe-list) shipped in v1.9.0 and swept same-day, go-cqrs-lite#113 shipped as the kv cache constructor and our 182-line adapter is deleted.
4. **index.html regenerated** — 22-row table with movement deltas, fleet average 68.4→84.6, all three 2026-11-05 issues marked resolved with receipts, new issues/strengths/posture/action-plan/matrix sections, per-module census re-measured (go-cqrs-lite 170 prod files, 38 trains — the old "15 trains" framing understated reality).
5. **Repo mechanics landed same-change** — TODO_LIST fleet row extended with the three new open items + upstream watches; AGENTS.md fleet-audit memory line rewritten (22 libs, 90.5, metaengine lag named, refresh triggers); docs/research/README.md series row 33 re-trued; stale datastar guide JS-version fixed on sight (v1.0.4 via static v0.7.0, sizes re-measured); corpus baseline re-trued 59431→59693 with ledger sentence in docs/status/README.md; render screenshot verified in headless Chromium 152 (DOM checks: 35 table rows, stat cards, zero page errors) and preserved in-repo as index.render.png, linked from the index footer.
6. **Gates green at close** — corpus ratchet (59693), tail ratchet, todo-hygiene (79 rows clean), docs-links (1723 links, after fixing 5 the go-datastar agent broke), docs-freshness.

## §c Partially done

1. **Quality-gate coverage is docs-legs-only.** Corpus/tail/todo-hygiene/docs-links/freshness run green this session; the full `nix run .#check-modules` battery and `.#test` were NOT re-run. Justification: the session touched zero Go files (only docs + baselines + root TODO/AGENTS), so build/test/lint surfaces are unchanged from the last green run — but that is an argument, not a receipt.
2. **Verify-block replay relied on agent attestations.** All 12 agents swore literal replays and fixed their own failures (several did: stale ls-remote pipes, wrong expectations); I personally replayed only the go-etag commands. No independent spot-audit of a sample of their claims.
3. **templ-components/datastar v1.27.0 latest-ness not independently verified** (flagged inside the go-datastar report itself; root + 5 submodules were verified, this one train was inferred from the family sweep).
4. **Screenshot verified via DOM assertions only** — the parent model cannot view images, so "renders, no broken layout" is proven by selectors/counts/zero console errors, not by eyes. The PNG is in-repo for a human to eyeball.
5. **The metaengine treadmill red is surfaced and tracked but NOT fixed** — deliberate scope call (a dependency wave with a nested-dir restructure to verify, not a report edit), but it means the tree ships with a red strict treadmill until the bump wave runs.

## §d Not started (deliberately out of scope this session)

1. The metaengine-family bump wave itself (v4.24.0 + sqliteengine v4.9.0).
2. The M2M idempotency fix (recipe doc or default-on).
3. The datastar-demo versioned script tag / adapter re-export / datastartest adoption.
4. cqrs-lint gate binary rebuild at v4.21.0 (waiver-adoption delta).
5. All code-level stretch items recorded in the index action plan (ProblemDetailsAudienceErrorHandler wiring, ExpectNotModifiedDynamic spec, QUERY-ladder internal panel, shipped OpenAPI consumer, flightrecorder evaluation, SSE-drops OTel metric).

## §e Owned failures (self-caught, all repaired same session)

1. **go-etag.md was assigned to NO subagent.** My wave plan covered 21 of 22 files; the gap surfaced only when compiling the index score table (13 used libs, 12 agent reports + 1 missing). Process bug: I built the waves from the user prompt list + a mental used/not-used split and dropped one. Fixed by refreshing it myself; the failure mode (compile-the-final-table-as-checksum) is what caught it and is worth keeping.
2. **Three mechanical fumbles in my own edits:** one multiedit old-string mismatch on the index hero (transcribed `git ls-remote` where the file said `git ls-remote --tags` — round-trip lost); one denominator-drift error I INTRODUCED in the index matrix (go-etag root 3 files, truth 4 — exactly the self-consistency class the deep-dive skill warns about); one corpus ledger pin off by one (my "zero net new lines" sentence took a line). All three self-caught and fixed same session, but each was avoidable: view-then-copy verbatim, re-count before writing, measure-after-edit before pinning. A fourth instance of the same class occurred WHILE WRITING this report: the first ledger draft attributed a +91 corpus delta to sibling drift when the delta was almost exactly this report itself (~+90) — caught by measuring before pinning, corrected in the ledger with the erratum recorded inline.
3. **The go-datastar agent shipped 5 broken relative links** (`../../../research/` from a two-deep directory). Root cause is in MY prompt: I never stated the link-depth convention for the new file. The docs-links gate caught them; prompt-template lesson recorded (state the depth or tell agents to run the link gate).
4. **Capacity discipline cost a wave boundary:** holding the sixth wave-2 agent until a wave-1 slot freed serialized the tail of the run. With rate limits unknown, safe-beat-sorry was the right call, but wave 2 could have been pre-staged to start the moment a slot opened instead of waiting for a report-and-wake cycle.

## §f Next (ranked; 30 items — the top 3 are the index action plan verbatim)

1. Metaengine-family bump wave: v4.23.0→v4.24.0 + sqliteengine v4.8.0→v4.9.0; verify the nested-dir module path resolves in usermgmt/dashboardui/systemadapter/root; treadmill strict-lag 0 green as the done-when.
2. M2M bot idempotency: document the Idempotency-Key keyExtractor recipe for bot routes, or default CommandIdempotency on for token-authenticated dispatch (note: entangled with the pending M2M owner-decision D1).
3. datastar-demo: versioned script tag via ScriptTagVersioned/ETagFor; consider adapter re-export; adopt datastartest for the adapter SSE assertions.
4. Rebuild the cqrs-lint gate binary at v4.21.0 (waiver adoption; the recipe from gotcha-era rebuilds applies).
5. Wire httputil v1.9.0 ExpectNotModifiedDynamic as an httpspec spec for the ETag-serving handlers (event catalog, OpenAPI, projection status, sync pull, partial render).
6. Wire ProblemDetailsAudienceErrorHandler (the last go-error-family partial) or record a decision not to.
7. Evaluate go-cqrs-lite flightrecorder (the single remaining missed row at 90.5) — likely pairs with appkit flightrecorderhealth.
8. SSE drop counters → OTel metric (the guide recipe) once drops have burned in.
9. A QUERY-ladder internal panel (self-audit row 13 is examples/integration-only today).
10. Ship one internal OpenAPI spec consumer (self-audit row 9: WithOpenAPI option form is test-only).
11. M06 follow-up: adopt upstream forms.InputFile instead of the hand-rolled file input in adminui users.
12. Upstream ask: templ-components popover CSS fix (compound position-area values rejected by Chromium 156) — until then adminui vendored layer stays load-bearing.
13. Archive round: this report makes the live tail 6 (a sibling archive round had already drained the 13 the 08-41 ledger described — the tail was 5 on arrival); TAIL_BASELINE 5 to 6 same-change; next docs-health pass should drain the settled 2026-11-14 set (four adminui-era reports).
14. Run the full check-modules battery + docs-gates run on the post-session tree (receipt for §c.1).
15. Independently verify templ-components/datastar v1.27.0 latest-ness (one ls-remote).
16. Human eyeball of index.render.png (I could not view it).
17. Spot-audit two agent-authored reports' verify blocks independently (trust-but-verify receipt).
18. e2e/server + examples consumer pin trues after the next family train (e2e pins dashboardui v4.17.0 / loginpage v4.18.0 vs adminui at v4.20.0 — family-tag lag is expected mid-train, re-check at next tag).
19. htmx 4-era extension seed (NV4 flip precondition, ROADMAP q35a) — decide seed vs wait for upstream extension registry.
20. system.New migration waves W2–W5 execution (W1 probes GO; hinge on owner D1/D2).
21. M2M owner decisions D1–D4 tick (blocks plan waves W1/W2; pre-staged recommendations exist).
22. The render-lane owner question (§g.2 of the 22-42 report) still open.
23. A17 (owner-gated) from the leverage plan.
24. Consider a small gate: "docs/review/libs report count == index row count" — would have caught failure §e.1 mechanically.
25. Add the link-depth convention to the fleet-series format block (prompt/library lesson from §e.3).
26. go-filewatcher v2.5.0 Healthy() noted in its report — no action while zero adoption; re-read if a watching concern ever lands.
27. Re-check go-credentials-store v0.4.3 system.New|metaengine anchor claim at the next audit (its README evolves fast).
28. erraudit inventory TOTAL=332 re-baselined at the next code wave (it will grow with any new code).
29. Watch for the next templ-components release (unreleased fixes listed in its report: StatCard focus, CopyButton fragments, BarChart, Sparkline, Stacked forms, ARIA/keyboard) and re-run the CSS-bundle class-set gates with it.
30. Refresh triggers honored: next fleet audit at any family train, a new sibling lib, or the next owner request.

## §g Owner questions (cannot be answered from the tree)

1. **Metaengine bump timing:** run the metaengine-family wave NOW (it reddens the strict treadmill and touches 4 modules with a nested-dir restructure to verify), or fold it into the next family train where the sweep rides existing release machinery?
2. **M2M bot idempotency shape:** recipe-documentation only (consumer opts in per route) vs default-on for token-authenticated dispatch (library opinion: bots are the replay-prone caller class). Note D1 (is bot M2M supported at all) is still unticked and gates the deeper shape.
3. **Fleet-audit cadence:** keep refresh-on-trigger (any family train / new sibling / owner request, as AGENTS.md now records) or also wire a periodic advisory gate (e.g. treadmill already covers lag; a staleness check on the report dates)?

## §h Verification receipts (this session)

- `bash scripts/checks/check-docs-corpus-budget.sh --ratchet` → green at 59693 (re-trued with ledger sentence in docs/status/README.md).
- `bash scripts/checks/check-docs-tail-budget.sh --ratchet` → green after TAIL_BASELINE 5→6 same-change (the 08-41-era "13" was stale; a sibling archive round had drained it).
- `bash scripts/checks/check-todo-hygiene.sh` → clean (79 rows).
- `bash scripts/checks/check-docs-links.sh` → 1723 links OK (after the 5 go-datastar link fixes).
- `bash scripts/checks/check-docs-freshness.sh` → PASSED.
- `nix run .#check-upstream-treadmill -- --refresh-cache --strict-lag 0` → RED on metaengine family (the finding, not a session breakage; advisory run is green apart from it).
- Headless Chromium 152 DOM verification of index.html: h1, 35 table rows (22 libs + 13 matrix), stat cards 22/14/8/84.6, zero page errors; screenshot preserved at docs/review/libs/index.render.png.
