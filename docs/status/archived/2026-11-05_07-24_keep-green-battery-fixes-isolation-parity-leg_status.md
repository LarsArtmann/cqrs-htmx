# Status: keep-green after the v4.16.0 train — battery fixes (css bundles, branching-flow re-pin), the isolation-parity leg, and a reconciliation-first session

> **ANNOTATED 2026-11-06** — ~~this report~~ **SUPERSEDED** by [`2026-11-05_09-53_docs-health-round47-corpus-annotate-battery-archive-and-six-docs-truth_status.md`](2026-11-05_09-53_docs-health-round47-corpus-annotate-battery-archive-and-six-docs-truth_status.md), which isolation-parity leg and keep-green fixes held in every later green aggregate run; the round-47 report records the aggregate gate state that superseded them. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-06 (round-23 (2026-11-06)).**

**Date:** 2026-11-05 07:24 CET · **Session task:** "Execute the round-46-tail pareto plan (READ, UNDERSTAND, … Execute and Verify one step at a time). Repeat until done."
**Head at report time:** `f83dfda4` (daemon-cadence ~2-4 min all session; ≥1 sibling session active on the same tree until ~07:10).
**Entry state:** the plan of record (`docs/planning/2026-11-05_03-11_SUPERB-round46-tail-train-green-and-backlog-pareto-plan.md`) was written at HEAD `1d8c8b68`; by session start a sibling had already executed M01 (the train: root/setup/usermgmt/adminui/dashboardui/loginpage `v4.16.0` + `usermgmt/v4.19.0`) and two post-train CI fixes (`96e90a69`, `714a6db9`).
**Verification basis:** every claim below was re-derived in-session (commands named inline); nothing is trusted from the plan text alone — the M04 finding below is what that discipline bought.

## What the session found (the recon part)

- **The train is tagged and healthy:** `verify-remote-tags` over all 104 workspace tags = 104 ok / 0 missing / 0 mismatch (F011).
- **CI red is exactly ONE upstream class:** newest run `41967052538` — `checks`+`security`+`mod-tidy` GREEN; `build`/`test`/`module-architecture` all fail on precisely `catalog/v4@v4.12.0/docserver/*_templ.go: undefined: templ.JoinURLErrs|ResolveAttributeValue` (build: catalog-demo example; test/module-architecture: integration_test vet). Zero non-catalog failures.
- **The upstream fix is in motion elsewhere:** the go-cqrs-lite sibling checkout holds 6 unpushed commits + a fresh interim status (`2026-11-05_06-39_post-sweep-plan-execution-interim-status.md`) — an ACTIVE session owns the catalog release. Left untouched per the one-writer rule (CURRENT_FOCUS ground rule 6).

## a) FULLY DONE

1. **Plan reconciliation before execution** — re-derived M01–M05 state from git/CI/reports instead of re-running the sibling's work; M01 ✅ (their lane), M05 ✅ (`714a6db9`), M02 partially, M03 mostly, M04 mis-shaped (see d4/e4).
2. **F023 — the 23-05 report annotated (21 items)** via `status-items --verify` first, then apply; links re-checked (1057 OK). Exactly the item 02-47 §f18 owned. Verdicts: #40-merge supersessions (publishing, merge order, upstream asks, pin lags, tag train), routed items (catalog residual → TODO P1 row; e4 → M04), owner-gated (g3 daemon → OQ20).
3. **F012 — TODO P1 CI row trued** with receipts (104/104 tags; signature-verified 3-job residual; sibling lanes named).
4. **css-bundle-classes FIXED** — both bundles rebuilt (the v4.16.0 family's templ-components class set incl. `.blur`; additive-only, 0 tokens lost), committed; `check-css-bundles` + `check-css-bundle-classes` GREEN (1056 tokens exact). The sibling's sweep had missed the same-change rule (AGENTS templ-components bullet).
5. **branching-flow FIXED** — +10/−9 decomposed by id-diff: 9 line-shift re-anchors of frozen findings + 1 genuinely-new `PHANTOM-PLUMBING internal/hx/immutable.go:24` — the verbatim move the branch-side 2026-11-02 amendment already adjudicated (its re-pin never landed on master; the #40 merge resolved the baseline to master's 202). Amendment row written; baseline re-pinned 202 → 203; gate GREEN (+0 added).
6. **M02 battery receipt** — full composite re-run with the full log kept: **86/88 stages green**; the only 2 reds (`module-isolation`, `workspace-build`) are the upstream catalog class = CI parity (CI's same legs are red on the same class). All other legs green in-run: coverage-threshold self-test, quickstart (GOWORK=off, real graph), badges, docs gates, route-postures, permission gates, shellcheck, dep budgets, release-train strict (897 requires, 0 unpublished).
7. **M04 RESHAPED + IMPLEMENTED** — the plan's "new API-vs-tag drift gate" would have DUPLICATED `check-module-isolation.sh` (per-member GOWORK=off build+vet, in composite+CI since 2026-10-04 — and it DID catch `Posture*` hermetically; the 23-05 report proves it). The real gap: `e2e/`+`examples/` had NO local hermetic leg (CI's build job covers them; the catalog-demo red was invisible to the local composite). Implemented `--examples` mode on the existing gate; **live-proven** (it fails on exactly catalog-demo build + integration_test vet — the documented class); wired into the composite stages array + `--sequential` path + a named CI step in `module-architecture` + the AGENTS Gates row (coverage note trued: composite covers hermetic builds now, the pre-push hook still doesn't).

## b) PARTIALLY DONE

1. **M04 wiring is UNCOMMITTED** — script/flake/ci.yml/AGENTS edits sit in the tree (daemon swept my earlier attempts into its commits twice; the final four-file set awaits a daemon-aware pathspec commit — NEXT item 1).
2. **F020 (PR-#40 receipt comment)** — the row now carries the battery receipts, but posting the PR comment waits for a green CI to point at (honest version of the box).
3. **This report's baseline bumps** — tail 2→3 and corpus +this-report are ledgered in the same commit as this file (see NEXT 2).

## c) NOT STARTED (this session; plan C/P4 tiers)

M06–M09 consumer waves (other repos) · M10 cqrs-lint 4.19.1 triage (8 modules) · M11 remainder (gotchas 51/52 — 49/50 landed by the sibling, verified) · M12 L1 tail (B55/B58/B59/B60) · M13 remainder (F070 landed by sibling — verified in the AGENTS row; research-archived conventions, guide sweeps, lychee excludes remain) · M14–M27 (all P4) · F019 bench-spike (correctly refused: load 70–82 all session).

## d) TOTALLY FUCKED UP (own mistakes, in the open)

1. **Piped-tail blindness AGAIN** — `nix run .#check-modules | tail -60` ate the failure head; cost ~15 min re-deriving the 6 failures by running 12 gates individually. Gotcha 3, confessed in round-46's report, still stepped on it. The full-log rerun is the receipt that shames it.
2. **`PIPESTATUS[0]` receipts were empty** under this host's shell (mvdan/sh) — two "rc=" lines I printed were noise; switched to `cmd >file 2>&1; echo rc=$?` only after noticing.
3. **Invented a flag** (`--include-detail`) for the branching-flow gate and broke its `cd` — read-the-script-first should have preceded invocation, not followed the error.
4. **Three consecutive commit races** with the daemon (hook 155/140/174 s vs 2-4 min cadence) before conceding to the documented gotcha-50b path (`--no-verify`, docs-only, hook-green-3× receipt in the message). The first attempt never checked the daemon cadence — predictable from `git log` timestamps alone.
5. **Battery/edits ordering contradiction** — I reasoned "docs edits before battery so gates read the final tree", then started the battery in background and edited anyway (the zero-line-annotation logic was sound; the stated plan and execution disagreed, and a sibling report landing mid-battery moved the tail/corpus numbers under me — resolved by the sibling, not by me).

## e) WHAT WE SHOULD IMPROVE

1. **Composite logs go to files, never through `tail`** — the #1 self-inflicted cost this session; `>file 2>&1` + grep is the standard.
2. **Commit-vs-daemon protocol:** check daemon cadence before EVERY commit; when cadence < ~2× hook duration, go straight to fetch → pathspec add → documented bypass with the hook-green receipt. Three dead hook runs at load 80 is ~8 minutes of wasted fleet compute.
3. **mvdan/sh rc discipline:** PIPESTATUS is unreliable here; `go-cache-env`/docs should standardize the redirect-then-echo form.
4. **Plans should name failure modes, not mechanisms** — M04 named "a new gate" that already existed; the executor found the real gap (examples coverage) only by refusing the plan's wording. Plan-errata annotation is queued (NEXT 14).
5. **Sibling heartbeat:** two sessions on one tree all morning produced daemon-swept staging twice and near-misses on TODO/AGENTS. A one-line "this session owns files X" marker (in CURRENT_FOCUS or the commit pathspec convention) would make the blast radius visible.

## f) NEXT (impact-ordered)

1. Commit the M04 four-file wiring (script, flake.nix, ci.yml, AGENTS) — pathspec, daemon-aware.
2. Commit this report + ledgered baseline bumps (tail 2→3, corpus 43992→+N).
3. Await/verify the go-cqrs-lite catalog release (sibling session owns it) → bump `examples/catalog-demo` + `integration_test` → CI's 3 red jobs AND the composite's 2 red legs go green together.
4. M10: cqrs-lint 4.19.1 triage — `cqrs-lint version`, rules diff 4.16.1→4.19.1, per-module fix/suppress/hold with receipts (8 red modules; the usermgmt upstream-catalog decision folds in R2).
5. M11 remainder: gotcha 51 `BUILDFLOW_MAX_CONCURRENCY=1` under load>30; gotcha 52 phantom Tailwind token (today's `.blur` drift is its live instance) + AGENTS index rows.
6. M12: B55 concurrent audit-log `-race` test; B58 `e2e/server/sync_types.go:19` + B59 `examples/sync-demo/main.go:44` value-embeds; B60 both suites; L1-complete commit.
7. M13 remainder: research-archived convention + in-file blockquotes; route-posture guide owning-package sentence; pre-split-layout guide grep; lychee excludes for the two archived dirs.
8. M06 browser-history: `reconcileSuperAdmins` → `GrantRoles`, `authztest.Suite`, delete the loop (own repo, own tests).
9. M07 InboxClean: delete `sseDeadlineFree`, bump dashboardui pin.
10. M08 SwettySwipperWeb: durable idempotent bootstrap, strip local replaces.
11. M09 crm + github-local-sync + webphone: seed/disable/declare postures; fix `docs/PERMISSIONS.md` `?` rows.
12. F020: post the PR-#40 aggregate-battery receipt comment once CI is green.
13. Harvest the two sibling exit reports (06-38: 94 open items, 06-41: 73) into TODO/ROADMAP at the next docs-health pass; archive when resolved.
14. Annotate the pareto plan's M04 errata (mechanism existed; examples leg was the gap) + strike M05 as sibling-done.
15. Owner parking lot (§5 of the plan): next-train GO, 7 parked artifacts, branching-flow owner review of provisional accepts, daemon policy OQ20, corpus posture a/b/c.
16. erraudit strict re-baseline (TOTAL=303 drift post-merge) + promotion-watch.
17. `check-shellcheck` receipt for the edited isolation script (CI covers it; local nix run for the same-commit proof).
18. bench-spike at the next quiet window (load < 30; refused honestly all session).
19. e2e/server `main.go` split (M25 slice 4 of 4; the other three splits are P4).
20. Re-run the full composite after items 1-3 land; expect 88/88 (the two catalog legs flip with the upstream bump).

## g) QUESTIONS (cannot answer from the repo)

1. **Pre-push hook scope:** extend "a push can never be greener than CI" to the hermetic legs (add module-isolation to `.githooks/pre-push`)? It would have blocked every push during this documented-red window — which is the point, or the bug, depending on the posture you want.
2. **Session ownership:** ≥2 sessions shared this tree all morning. Do I claim M06–M09 (consumer waves in OTHER repos: browser-history, InboxClean, SwettySwipperWeb, crm) now, or are those owned by the still-active sibling sessions?
3. **Tail posture:** the two sibling exit reports hold 167 open §f items between them and my report puts the live tail at 3 (=budget, >ratchet-baseline 2). Default: ledgered bump to 3 and adjudicate/archive at the next report cycle. Alternative you may prefer: force their full annotation+archive FIRST (heavy pass, ~167 verdicts). Which?
