# Status — 18-20 Plan Execution Wave + Master-Red Repair (2026-10-12 20:01)

> **ANNOTATED 2026-10-25** — ~~this report~~ **SUPERSEDED** by [`2026-10-14_20-54_wave-tail-execution-gate-bugs-systemadapter-root-cause-self-review_status.md`](2026-10-14_20-54_wave-tail-execution-gate-bugs-systemadapter-root-cause-self-review_status.md), which 14-41 cut v4.14.1; its tooling rows (bump-dep lint leg, StateShape audit, treadmill script, templ chain) are TODO rows citing it. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-25 (round-23 (2026-10-24)).**

> **Point-in-time snapshot.** Session scope: executing
> [`docs/planning/2026-10-12_18-20_SUPERB-post-alignment-execution-wave-pareto-plan.md`](../../planning/2026-10-12_18-20_SUPERB-post-alignment-execution-wave-pareto-plan.md)
> (A1–A8 ungated tiers) — interrupted at ~90% by the discovery that **the alignment
> push had red'ed master CI**, which became the session's real work.
>
> **Repo state at writing:** HEAD `1bc4ce48`, tree clean, **3 unpushed commits**
> (`46ed1cec` docs batch 1, `0d473d99` master-red repair, `1bc4ce48` ROADMAP batch 2
> — daemon-captured, message unamended). Machine load 25–30 all session.
>
> **Format note:** the status-report skill is HTML-canonical; the operator's prompt
> explicitly demanded `.md` at this path — honored per the skill's user-override
> rule (one-off, not propagated).

---

## a) FULLY DONE (verifiable, committed)

| # | What | Evidence | Scope |
| --- | --- | --- | --- |
| a1 | **A1 docs disambiguation (G5–G8)** — the two-catalogs distinction (cqrs-htmx `EventCatalog` ≠ `go-cqrs-lite/catalog/v4`) documented in all three consumer-facing surfaces: guide Serving note + rewritten See Also (distinction, demo, integration-test, research-map links), setup README `EventCatalogPath` naming note, catalog-demo doc-comment vocabulary note | `46ed1cec` | `docs/guides/event-catalog-guide.md`, `setup/README.md`, `examples/catalog-demo/main.go` |
| a2 | **A2 truth-pass (G1–G4)** — research-note Outcome section appended (append-only: catalog v4.8.0 swept, disambiguation landed, DocsPath stays a decision), TODO_LIST `Updated:` stamp refreshed for the alignment + execution wave | `46ed1cec` (G1 strike + G2 header note were already done in the prior session — verified, not redone) | research note, TODO_LIST |
| a3 | **A5/G9 CI verdict** — both red runs (16:07 `291b2c51`, 16:27 `563b8dc8`) triaged to exactly 2 jobs: `checks` (dashboardui templ codegen drift) + `lint (usermgmt)` (exhaustruct `StateShape`) | `gh run view` 38275235749 / 38277550570 | CI runs on the alignment push |
| a4 | **Master-red repair, both causes** — (1) regenerated `dashboardui/aggregates_templ.go` module-canonically after the templ-fmt formatter reformat (`79711ae0`, daemon-captured mid-alignment) had gone out without regeneration; `nix run .#check-codegen` rc=0 after. (2) `cloneSnapshot` now copies `StateShape` through (snapshot v4.7.0 added the field; the copy-through is a correctness fix — a stamped snapshot keeps its stamp — not lint appeasement) | `0d473d99`; hermetic `golangci-lint run` usermgmt + dashboardui both **0 issues**; usermgmt Snapshot tests ok | dashboardui, usermgmt |
| a5 | **A6 catalog-demo smoke (G17–G18) resolved as stale-gap** — the "visual smoke" ROADMAP gap was **already closed** by pre-existing `TestHTTPHandlers_ServeSpecs` (`main_test.go:50`, since the v4.6 milestone `25cfd08b`: httptest boot asserting `/openapi.json` `/asyncapi.json` `/diagram.d2` `/health` 200s). Verified green hermetic today; ROADMAP row struck with evidence instead of writing a duplicate test | hermetic `go test ./...` in examples/catalog-demo rc=0; strike in `1bc4ce48` | ROADMAP, catalog-demo |
| a6 | **A3 DocsPath decision packet (G11–G13)** — three-shape table (A `DocsPath`+introspection = heavy dep; B `DocsHandler http.Handler` = zero-dep seam; C status-quo consumer mount = already works), validation/wiring rules, GO/NO-GO demand-signal criteria, verdict slot — consolidated INTO the ROADMAP data-mesh Approach-C section (no duplicate artifact, per guardrail 2) | `1bc4ce48` | ROADMAP.md |
| a7 | **A5/G10 upstream treadmill sweep (go-cqrs-lite family)** — all 12 swept pins match latest published tags: catalog v4.8.0, decider v4.8.0, sqliteengine v4.5.3 (tag prefix `metaengine/sqliteengine/`), metaengine v4.19.0, schema v4.7.0, snapshot v4.7.0, systemscenario v4.1.0, system v4.13.0, event v4.14.0, dedup v4.3.0, projectionhost v4.6.0, scheduling v4.7.0. Apparent lower pins traced to the deliberately-untidy `scripts/testdata/verify-tag/` poison fixture only. **The lag tail did NOT grow** (gotcha 39's worry, checked) | `git ls-remote` 3162 tags vs go.mod pins | go-cqrs-lite family |
| a8 | **A4 preflight legs completed (see b3 for the remainder)** — receipts verified: `ff36c975` (sync assets cursor/backendId defect fix), `a4f28aae` (CHANGELOG cut receipts), `3e2bf224` (floor 1.27.2, go.mod says `go 1.27.2`); tag `v4.14.1` confirmed absent on origin (cut available); root CHANGELOG `[Unreleased]` carries the full sync surface (AddSyncOpenAPI, sync-demo, defect fixes, ADR-0056); **root hermetic single-test rc=0** (4 packages, 23.4s) | git show receipts; ls-remote; CHANGELOG grep; GOWORK=off test log | root v4.14.1 preflight |
| a9 | **A7/G19 already satisfied** — gotcha 39 (lag tail grows mid-sweep; `$`-anchor title leak) was codified into AGENTS.md in the prior session; verified present at `AGENTS.md:135`, not duplicated | grep | AGENTS.md |

## b) PARTIALLY DONE

| # | What | Works now | Missing | Blocker / effort |
| --- | --- | --- | --- | --- |
| b1 | **The push itself** | 3 commits local, each verified (docs content-checked; repair verified by codegen check + lints + scoped tests) | Pre-push strict gates not yet run; CI watch after push not done | M — one push cycle; prefer `--refresh-cache` first (gotcha 39) |
| b2 | **F11 outcome annotations** | 17-33 plan header note (T3 consumed) + research-note outcome done | Inline strikes on 17-33 plan T1/T2 rows + the 18-20 plan's own A-row outcome blockquote + `$`-anchor leak note — owed AFTER the push (annotations cite the push) | S — one docs commit post-push |
| b3 | **A4 verify-tag dry-run (G16)** | Every preflight input verified (receipts, CHANGELOG, tag absence, hermetic root test) | `scripts/tools/verify-tag.sh . v4.14.1 --dry-run` not yet executed (needs the clean tree — now true) | S — one command, **prep-only, the cut itself stays owner-gated (Q2)** |
| b4 | **A5 treadmill completeness** | go-cqrs-lite family fully swept (12 modules) | Non-GCL upstreams (templ-components, httputil, go-sse, go-appkit, go-health-dashboard, go-codec, go-error-family, go-branded-id) not swept this session — the strict train gate at push re-verifies them mechanically, so this is report-scoping honesty, not a red | S — extend the sweep loop |
| b5 | **A7 hardening (G20/G21)** | G19 done (pre-existing) | rc-guard sweep→commit wrapper sketch (G20) and `bump-dep --commit-mine` upstream filing draft (G21, owner files) not started | S/M each |
| b6 | **Full battery legs** | Scoped verification green (2 module lints, 3 test runs, codegen check) | `nix run .#test` / `.#check-modules` / coverage-gate not run this session — machine at load 25–30 all day made the ~6-min battery unattractive mid-session | M — quiet window, or accept slower run now |

## c) NOT STARTED

| # | What | Why | Still wanted? |
| --- | --- | --- | --- |
| c1 | **A9 T5 implementation** (`Config.DocsPath`/`DocsHandler` — F12–F16: field+validation, machine wiring, serving test, README/CHANGELOG, verify-tag train + consumer sweep) | **Owner-gated (Q1)** — the ROADMAP graduation rule demands a demand signal; the A3 packet is the decision input | Yes — packet makes the GO/NO-GO one decision |
| c2 | **A10 go-appkit alignment lane** (their catalog v4.7.1→v4.8.0 + accumulated wave, their gates) + **A8/G28** docs README pointer | **Owner-gated (Q3)** — cross-repo, not mine by default (plan guardrail 7) | Yes — the last open piece of the catalog v4.8.0 story |
| c3 | **Q2 root v4.14.1 cut** (`verify-tag --push`) | Owner-gated; preflight deliberately makes it one command (guardrail 1: A4 never tags) | Yes — unblocks sync-demo's hermetic `AddSyncOpenAPI` break |
| c4 | **docs-health HARVEST** of this report's section (f) into TODO_LIST/ROADMAP | Report just written; harvest is the next action after it lands | Yes — section (f) items die entombed otherwise |
| c5 | The broader **staged multi-tag wave** (usermgmt 41-file delta mostly tests, dashboardui 31 templ+assets, identity-model/adminui/loginpage/systemadapter/health/setup/totp/webauthn/oauth2 deltas) | Separate train decision from the root cut; enumerated today for the first time with exact per-module counts | Yes — P1 row "staged 14-tag wave cut" already tracks it |

## d) TOTALLY FUCKED UP (radical honesty)

1. **Our own alignment push red'ed master CI and sat undetected for ~4 hours (16:07 → this session's A5 check).** Severity: master red blocks every concurrent session's green assumption. Root cause is a THREE-leg chain, and all three legs are ours:
   - **Leg 1 — bump-dep doesn't lint.** `bump-dep verify` = hermetic tidy + build + vet; vet does not run `exhaustruct`. The snapshot v4.7.0 `StateShape` addition therefore sailed through the sweep, the local battery (`nix run .#test` compiles tests but doesn't lint), and the strict pre-push gate (train + version-drift only). First linter to see it was CI. A post-sweep hermetic `golangci-lint run` on the 27 swept modules would have caught it in minutes.
   - **Leg 2 — the templ-fmt formatter mutated the tree mid-session and the daemon committed it (`79711ae0`) without the downstream artifact.** Third instance of the "formatter damage + daemon capture + stale derived artifact" class (gotcha 4/35 family). The .templ reformat was content-identical, but `_templ.go` line anchors went stale → codegen-drift red.
   - **Leg 3 — no post-push CI watch.** The alignment session pushed at 16:07 and stopped watching. A5 (CI watch) was third in the plan's execution order; had the session pushed more docs first, the red window would have grown. The runbook has no mandatory "watch the push's CI to green" step.
2. **Both planning artifacts (17-33, 18-20) carried a stale-on-arrival gap**: A6/F17–F18 specified building a catalog-demo smoke test that has existed since the v4.6 milestone (`25cfd08b`), and the ROADMAP gap row I struck today was the source of the stale claim. I caught it before writing a duplicate test THIS time, but the plan-generation turn violated the repo's own verify-before-acting doctrine — plans must grep the tree for the gap they intend to close. (This is also a fixed split brain: docs said "gap", code said "covered".)
3. **I lost the commit race to the auto-commit daemon twice** (`fac7102b` docs batch, `29366c1f` fix batch), both recovered by the documented amend ritual, one still unamended (`1bc4ce48` ROADMAP). Cause: I ran scoped fmt and git add+commit as two tool calls; the daemon polls between them. The session's own guardrail 8 says "commit immediately after fmt, one tool call" — I technically did add+commit in one call but let fmt precede it as a separate call. Fix is mechanical: fmt && add && commit chained in ONE command.
4. **The full pre-commit hook attempt under load 30 burned ~3.5 minutes and failed transiently** (test-compile killed after ~3.5s no-output, 4 modules; cascade cancels in brandid-lint/go-humanize-linter). I checked `uptime` only AFTER the failure — load was visible at session start. Known-class failure (gotcha 8); the retry-first rule is right, but the load check belongs BEFORE the attempt, not after.
5. **What I did NOT fuck up, stated to keep the record honest:** I did not revert foreign changes (the templ reformat was formatter-authored; I regenerated instead of reverting); I did not tag (guardrail 1 held); the docs claims are all evidence-cited; the treadmill "no fresh tags" claim is scoped to the family I actually swept (b4). No lies found on re-read. No new ghost systems (the A3 packet documents an intentionally-unbuilt seam in ROADMAP — that is the opposite of a ghost). No new split brains — and one existing one (stale smoke-gap row) was killed.

## e) WHAT WE SHOULD IMPROVE

1. **Give bump-dep a lint leg** (biggest systemic fix from d1): post-sweep, run hermetic `golangci-lint run` on each bumped module (or a `--lint` flag). Impact: kills the "CI finds what our gates missed" class that has now fired twice (exhaustruct today; earlier exhaustruct reds in the 10-11 wave). Effort M.
2. **Mandatory post-push CI watch in the runbook**: after every push, `gh run watch` (or poll) until terminal state before declaring done; red within the window = fix forward immediately (d1 leg 3). Effort S; candidate for a `scripts/tools/push-and-watch.sh` wrapper.
3. **Close the formatter→derived-artifact chain** (d1 leg 2): when BuildFlow's templ-fmt (or any formatter) mutates a `.templ`, the same run should regenerate `_templ.go` (or the repair step must be disabled for templ files so damage never lands). At minimum: AGENTS gotcha line + a check-modules leg that diffs `templ generate` output (CI already has this — the gap is only local). Effort M.
4. **One-command fmt+commit ritual**: codify `nix run .#fmt -- <paths> && git add <paths> && git commit …` as a single chained call in AGENTS gotcha 4 (the daemon-poll window between fmt and commit is the whole race). Effort S.
5. **Load-aware commit strategy**: check `uptime` before choosing full-hook vs `--no-verify`+justification (gotcha 8's transient class), not after. Effort S (a line in the same gotcha).
6. **Verify-before-planning for gap claims**: plan turns must grep the tree (tests, ROADMAP) before filing "missing X" rows — the A6 miss cost plan credibility, even though execution caught it. Effort S (behavioral).
7. **Daemon-captured content deserves immediate message amends** — `1bc4ce48` is still carrying a heuristic title; the amend ritual works, the discipline slipped at the end of the session. Effort S.
8. **StateShape round-trip audit (found while writing this report)**: the copy-through fix covers the in-memory store's clone; whether the PERSISTED snapshot paths (SQL store / codec) round-trip `StateShape` for usermgmt consumers is unverified. If the codec drops the stamp, stamped snapshots silently degrade to unstamped across restarts. Verify or file upstream. Effort S to check.
9. **Treadmill sweep as a script**, not an ad-hoc grep pipeline (the first ls-remote attempt silently failed through a bad awk; the tag-prefix surprise `metaengine/sqliteengine/` cost a second look). A `scripts/checks/check-upstream-treadmill.sh` with per-repo prefix maps would make A5-class checks one command. Effort M.

## f) Up to 50 things to get done next (ranked; HARVEST input)

| # | Task | Impact | Effort | Category |
| --- | --- | --- | --- | --- |
| 1 | Push the 3-commit pile (`46ed1cec` + `0d473d99` + `1bc4ce48`) through strict pre-push, `check-release-train --refresh-cache` first | **Critical** (un-reds the red-class half: lint fix + codegen fix reach CI) | S | Release |
| 2 | Watch the repair push's CI to terminal state (both previously-red jobs) — first live use of improvement e2 | **Critical** | S | Process |
| 3 | `verify-tag.sh . v4.14.1 --dry-run` — completes A4/G16; preflight then fully done | High | S | Release prep |
| 4 | Amend `1bc4ce48`'s heuristic message (detailed ROADMAP batch-2 description) | Low | S | Hygiene |
| 5 | F11 outcome annotations: 18-20 plan A-row blockquote (a1–a8 + gated holds), 17-33 plan T1/T2 inline strikes — post-push so they cite it | High | S | Docs |
| 6 | **OWNER Q2: cut root v4.14.1 now?** (one `verify-tag --push` after #3) — unblocks sync-demo hermetic break, closes the pre-existing root-require gap | High | S | Decision |
| 7 | Post-cut consumer sweep: bump-dep root-require sweep to v4.14.1 + sync-demo hermetic build/test green (the `AddSyncOpenAPI` undefined breaks resolves) | High | M | Release |
| 8 | **OWNER Q1: build T5 (DocsPath/DocsHandler) or keep the ROADMAP packet as the verdict?** (A9 F12–F16 ready to execute on YES) | High | — | Decision |
| 9 | **OWNER Q3: go-appkit catalog v4.8.0 lane — mine or yours?** (A10/G27 + A8/G28 pointer) | Medium | — | Decision |
| 10 | bump-dep lint leg (e1) — hermetic golangci per bumped module, `--lint` flag or always-on | High | M | Tooling |
| 11 | Post-push CI watch runbook step + optional `push-and-watch.sh` (e2) | High | S | Process |
| 12 | Formatter→regen chain guard for templ (e3): check-modules leg locally or disable templ-fmt repair | High | M | Tooling |
| 13 | StateShape persistence round-trip audit (e8): SQL store + codec path; file upstream if the stamp drops | High | S | Bug-hunt |
| 14 | Full battery legs when machine allows: `nix run .#test` + `.#check-modules` + coverage-gate re-run (last 15/15 on 10-01; root surface grew since) | High | M | Quality |
| 15 | docs-health HARVEST of this section (f) into TODO_LIST/ROADMAP | High | S | Docs |
| 16 | Codify one-command fmt+commit + load-check in AGENTS gotcha 4 (e4/e5) | Medium | S | Docs |
| 17 | Extend treadmill sweep to non-GCL upstreams (b4) + script it as `check-upstream-treadmill.sh` (e9) | Medium | M | Tooling |
| 18 | A7/G20: rc-guard sweep→commit wrapper sketch (one page, agents-notes or scripts sketch) | Medium | S | Process |
| 19 | A7/G21: `bump-dep --commit-mine` upstream filing draft (owner files at verify-before-filing gates) | Medium | S | Upstream |
| 20 | The staged multi-tag wave cut (c5): precise consumer-visible delta per module → tag plan (P1 row) | High | L | Release |
| 21 | Root coverage-gate: tests for the new sync/openapi surface or explicit threshold adjudication (P1 row; verify current CI `test` state first — it was green on the red runs) | High | M | Quality |
| 22 | P1 large-file row: `git rm --cached` the two committed binaries (examples/sync-demo/sync-demo 12.1 MB, cqrs-htmx-upgrade 3.7 MB) + `.gitignore` entries (forward-only) | High | S | Cleanup |
| 23 | Verify no NEW stray binaries landed from today's failed BuildFlow run (checked at 20:01: none — keep checking after each hook failure) | Medium | S | Hygiene |
| 24 | AGENTS gotcha candidate: templ-fmt + daemon capture + stale `_templ.go` as the named third instance of the class (with this session's hashes) | Medium | S | Docs |
| 25 | Verify the `$`-anchor strip fix actually landed in bump-dep's message template (gotcha 39 companion; two pushed titles still carry literal `$` — history, forward-fix only) | Medium | S | Tooling |
| 26 | Diagnose `brandid-lint` cascade failures from the 19:24 pre-commit run (environment class or real?) | Medium | S | Tooling |
| 27 | Triage `type-check` findings: tests/sync.spec.ts 8× 'unknown' (pre-existing Playwright TS strictness) | Low | S | Quality |
| 28 | Triage `pytest-test` "no tests ran" warning (noise class?) | Low | S | Quality |
| 29 | Triage vulnix 21 CVEs in nix derivations (known-noise vs actionable) | Low | M | Security |
| 30 | go-licenses E-lines for catalog-demo + catalog/v4.8.0 (no LICENSE found) — confirm the gotcha-22b devShell shim still covers; consider upstream LICENSE addition ask | Medium | S | Compliance |
| 31 | lychee policy decision (authenticate-vs-exclude) — standing warn, two exclusions proposed by the tool itself | Low | S | Tooling |
| 32 | `interrogate` missing from BuildFlow PATH — trivial skip; install or silence | Low | S | Tooling |
| 33 | Inline-strike 17-33 plan T3 row (header note exists; convention prefers inline) — fold into #5 | Low | S | Docs |
| 34 | Status-tail archive cadence: 3 reports today alone; round-21 full-tail archive owed (TODO stamp) | Medium | M | Docs |
| 35 | `check-docs-tail-budget` advisory run after this report + the F11 annotations land | Low | S | Docs |
| 36 | Re-run `bash scripts/checks/check-release-train.sh --refresh-cache` before ANY next push (gotcha 39 ritual; the go-appkit lane will need it cross-repo too) | Medium | S | Process |
| 37 | Consider CI-verdict polling line in pre-push hook output (informational `gh run list` post-push hint) | Low | S | Tooling |
| 38 | adminui/dashboardui CSS bundle check after any future templ-components family bump (standing rule — not due today, recorded for the next train) | Medium | — | Process |
| 39 | Annotate prior-session artifacts that referenced "catalog-demo visual smoke" as a gap (17-33/18-20 plans) so no future session rebuilds it | Low | S | Docs |
| 40 | Decide: does `check-codegen` belong in bump-dep verify for templ-bearing modules (dashboardui/adminui/loginpage)? (CI has it; local gap confirmed today) | Medium | S | Tooling |
| 41 | Sweep TODO_LIST P2 row #1 (docs disambiguation) to `[~]`→done-at-push evidence once #1–#2 land | Low | S | Docs |
| 42 | Same for P2 row #2 (DocsPath packet → done, verdict open) after #5 | Low | S | Docs |
| 43 | Bench-spike: standing quiet-window re-pin discipline unchanged; next train reminder only | Low | — | Process |
| 44 | systemadapter Go-mod-level replace hygiene: none this session; standing gotcha-36 resolution confirmed complete (no action, recorded) | Low | — | Cleanup |
| 45 | Explore gh-native `gh run watch` vs polling in the CI-watch step (#2/#11) — pick one, document | Low | S | Tooling |
| 46 | Re-check `.buildflow.yml` skip list intactness (gotcha-35 daemon-capture class; not checked today) | Medium | S | Hygiene |
| 47 | loginpage/adminui: no work this session; next docs pass owns the datastar-demo rebrand (standing ROADMAP item, untouched) | Low | — | Cleanup |
| 48 | File the "gap rows must be tree-verified at plan time" doctrine line into AGENTS or the planning checklist (e6) | Medium | S | Docs |
| 49 | After Q3 YES: run `check-release-train --refresh-cache` in THIS repo after go-appkit's cross-repo release (gotcha 27e/39 cross-repo form) | Medium | S | Process |
| 50 | Celebrate-and-keep: the A6 near-miss proves the strike-don't-rebuild instinct works — codify it as the default in planning turns, not a lucky catch | Low | S | Process |

## g) Three questions I cannot answer myself

1. **Cut root `v4.14.1` now (Q2), or stay staged?** Everything verifiable is verified: receipts (`ff36c975`, `a4f28aae`), CHANGELOG completeness, tag absence on origin, root hermetic test green, only the `--dry-run` formalities left. The cut un-blocks sync-demo's hermetic `AddSyncOpenAPI` break and lets the consumer sweep close the last pre-existing root-require gap. I will not tag without your YES (guardrail 1), and the timing (before vs after the bigger staged wave) is a release-strategy call only you can make.
2. **Build T5 (`Config.DocsPath`/`DocsHandler`) or close the question with the ROADMAP packet?** The packet (A3, in ROADMAP now) gives you the three shapes and the GO/NO-GO demand criteria. If your answer is YES, which shape — A (introspected catalog, heavy dep) or B (consumer-supplied `DocsHandler`, zero-dep)? That sub-choice changes the whole F12–F16 implementation and I will not guess your dependency-appetite for `setup`.
3. **The go-appkit catalog v4.8.0 alignment (Q3): mine or yours?** If mine, I need to know whether to run it under go-appkit's own gates in a dedicated session there (and whether the A8 docs-README pointer should ride that session). If yours, this repo's catalog story is DONE at v4.8.0 and I close the TODO row as owner-owned.

---

**Next action on resume:** #1 (push through strict gates) → #2 (watch CI) → #3–#5 (finish A4/F11) — then the three answers above unblock the remaining 80%.
