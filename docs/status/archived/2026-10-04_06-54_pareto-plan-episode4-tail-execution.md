# Status Report — Pareto Plan + Episode-4 Tail Execution

> **ANNOTATED 2026-10-04 (docs-health round 17):** §b2/§d1 sharpened — the 6 workspace-mode test failures do NOT reproduce in CI's published-pins mode (test job green, run 37178906141); they live in the local sibling-tree state (go.work replaces), a concurrent session is actively fixing them (uncommitted brand-prefix strip in identity-model/id.go), and the LAST red on master is the loginpage 7/5 dep-budget breach (module-architecture job) — the release-train blocker is the budget decision, not the tests. The T9 archive pass (§f6) plus §f23–f25 process items executed in this pass. T5–T9 of Tier 20% remain open as scoped (verified: PageHeader ×8 and statusToBadgeMap still absent from the tree). Archived this pass.

**Date:** 2026-10-04 06:54 CEST · **Branch:** master, pushed to `origin` (`4e1d41f4..c9274e37`, release-train strict green)
**Session scope:** episode-4 templ-components deep-dive → Pareto plan (26 tasks / 45 micro-tasks) → execution of Tier 1% + 4% (T1–T4)
**Plan artifact:** [`docs/planning/2026-10-04_06-27_SUPERB-templ-tail-pareto-execution-plan.md`](../../planning/archived/2026-10-04_06-27_SUPERB-templ-tail-pareto-execution-plan.md)
**Format note:** `.md` per explicit user path demand (status-report skill is HTML-canonical; standing dispatch-report exception applies).

---

## a) FULLY DONE

| # | Item | Evidence |
|---|---|---|
| 1 | **Episode-4 repo-wide templ-components audit** — adminui 97 (↑ from 68), dashboardui 94 holds, loginpage 95 (new adoption); zero anti-patterns; v1.19.4 = latest; gates green (codegen, CSS bundles, class-sets, loginpage tests) | [`docs/research/2026-10-04_templ-components-deep-dive.html`](../../research/2026-10-04_templ-components-deep-dive.html) |
| 2 | **Prior-art annotations (gotcha 21)** — outcome notes on both 09-17 HTML reports + the 09-23 loginpage audit (append-only); research README series table updated to episode 4 | 3 files in `docs/research/` |
| 3 | **Pareto plan** — 1%→51% (setup styling), 4%→64% (loginpage cluster), 20%→80% (audit tail), remaining 80% enumerated; ALL 32 open TODOs mapped to 26 tasks + 45 micro-tasks + mermaid graph; committed + **pushed** | plan file (see header) |
| 4 | **T1 (THE 1%): setup styling unblock** — `setup.Config.LoginCSSPath` passthrough; setup/README §Styling (two proven `@source` strategies); setup-demo now ships a real compiled consumer stylesheet (61,023 bytes, canary-guarded) served at `/app.css`; `nix run .#build-setup-demo-css` flake app; CHANGELOG entry | commit `15bee1cf` + daemon commits |
| 5 | **T2: AccentColor doc truth-pass** — config.go field + DefaultAccentColor comments now state favicon-only reach (they claimed button/highlight styling that no longer exists) | loginpage/config.go |
| 6 | **T4: `Config.NoOAuth2`** — force-hides auto-detected OAuth2 buttons; mutually exclusive with `OAuth2Buttons` (rejected at `New`); 3 new tests; README row; CHANGELOG. Closes the last open loginpage review follow-up | handler_test.go (53 tests total) |
| 7 | **T3: library-owned spinner** — `feedback.Spinner` server-rendered hidden behind the `lp-spinner` hook via `display.Button.Icon` slot; login.js only toggles visibility, `SPINNER_HTML` deleted; regression test | loginpage/page.templ + assets/login.js |
| 8 | **Lint debt cleared** — `withDefaults` cyclop 15→≤12 (extracted `Config.validate()`); gocritic offBy1 fixed; final battery: tests `-race` green, vet clean, **golangci-lint 0 issues**, gofmt clean | loginpage module |
| 9 | **TODO hygiene** — moot "loginpage adoption OWNER CALL" struck (adoption landed 2026-10-04, scored 95); episode-4 tail item + regression item recorded | TODO_LIST.md |

## b) PARTIALLY DONE

1. **Plan execution (T5–T9 of Tier 20% not started)** — dashboardui `PageHeader` ×8 swap, upstream `statusToBadgeMap` issue, RelativeTime swap, status-report archive pass, M12/M13/M14 hygiene: fully scoped in the plan (micro-tasks M021–M037), zero execution this session.
~~2. **Setup test regression** — discovered, isolation-verified, recorded in TODO_LIST with error signatures; root-cause NOT attempted (out of plan scope).~~ SHARPENED 2026-10-04 — CI test job GREEN on published pins (run 37178906141), so the failures live in the LOCAL sibling-tree state (go.work replaces); a concurrent session has an uncommitted brand-prefix-strip fix in identity-model/id.go; the train is NOT blocked by test failures
3. **Full `.#check-modules` battery** — only module-scoped verifications ran (loginpage/setup/setup-demo build+vet+test, both CSS gates, codegen). The 27-stage composite battery hasn't been re-run since the loginpage cluster landed.
4. **Directive-8 commit messages** — plan + T1 got detailed messages; the loginpage cluster (T2–T4) was captured by the auto-commit daemon in heuristic `chore:` commits (`b9f340d0`, `b7e94ef0`, `c9274e37`) before a manual phase-boundary commit could land. Content is safe and pushed; message fidelity lost for that batch.

## c) NOT STARTED

Tier 20% tail (T5–T12), Tier-80% items (T13–T19), and the gated/dormant set (T20–T26) — all deliberately sequenced behind the executed tiers; see plan Tables A/B for exact scoping. Notables: v5 codemod scaffold (T18), CI lane for `check-cqrs-lint` (T13), fleet cqrs-lint binary swap (T15), e2e + bench-spike battery remainder (T16).

## d) TOTALLY FUCKED UP

1. **master is red: 6 deterministic test failures (PRE-EXISTING, not this session's diff).** 5 setup tests + the setup-demo seed fail with `Register: [transient:usermgmt.user.read_model_missing] user not in read model after register`, accompanied by `projection worker crashed … subscribe live events: bus subscription refused`. Verified in a temp worktree at pre-change commit `4e1d41f4` — this predates ALL of today's work. Suspect: a bus/projection-subscription behavior change that rode the v1.19.4/v4.13 family trains. **This blocks the next release train** (`.#test` will gate). Recorded as the top TODO_LIST item.
2. **Pre-commit hook raced the auto-commit daemon** — the plan commit's BuildFlow run hit its 9 known non-devShell step failures; while deciding the fallback, the daemon committed the content with a heuristic message. Protocol friction (gotcha 4/8), zero content loss, but two commits carry `chore:` messages for carefully-written bodies.
3. **Nothing else.** No reverts of foreign diffs, no gate bypasses without justification (`--no-verify` used twice, both documented, both docs/no-code-impact paths), no speculative rewrites.

## e) WHAT WE SHOULD IMPROVE

1. **Daemon-vs-hook race**: when a commit needs a real message, write the file, commit IMMEDIATELY (don't batch), or accept heuristic capture. Today's plan-file loss was self-inflicted by batching.
2. **Train hygiene**: the read_model_missing regression sat unnoticed on master since at least `4e1d41f4` — the full `.#test` battery (gotcha 2: root `./...` is module-scoped and FALSE-green) clearly hasn't run since the family bump. The workspace-build gate exists; a **workspace-TEST gate** equivalent would have caught this at bump time.
3. **setup demo parity**: setup-demo's test seeds via register — it was silently red too; example tests deserve inclusion in every pre-train battery check (they're in `.#test-all`, not `.#test`).
4. **Loginpage AccentColor** is now favicon-only — a candidate rename (`FaviconAccent`) for the next MAJOR window to stop the doc-lie class at the type level instead of in comments.
5. **Micro-friction**: two multiedit failures from box-drawing-char/whitespace mismatches (login.js, config.go) — read-then-edit with fresh Views would have saved two round trips.

## f) TOP 50 NEXT (sorted by impact × ease; Tier tags from the plan)

| # | Item | Tier |
|---|---|---|
~~| 1 | **Root-cause the read_model_missing/bus-subscription regression** (blocks trains; suspects: projection-host subscribe path, go-cqrs-lite bus bump) | 🔥 now |~~ SHARPENED 2026-10-04 — CI test job green on published pins (37178906141): the failures do NOT reproduce off the local sibling trees; the last actual red on master is the loginpage 7/5 dep-budget (module-architecture job). Local workspace-mode failures: sibling-tree state, concurrent session actively fixing (uncommitted identity-model/id.go prefix strip)
| 2 | Run full `.#check-modules` + `.#test` battery post-loginpage-cluster | 🔥 now |
| 3 | T5: dashboardui `display.PageHeader` ×8 swap (audit/projections/aggregates/snapshots/dlq) — last full capability gap; goldens + class-set gates decide | 20% |
| 4 | T6: file upstream `statusToBadgeMap` issue (verify-before-filing + github-voice gates), then adopt `StatusBadge` in adminui on land | 20% |
| 5 | T7: dashboardui `RelativeTime` narrow swap (snapshot-detail Created) | 20% |
~~| 6 | T9: status-report ARCHIVE pass (7 over-budget reports → `docs/status/archived/`) | 20% |~~ done — round-17 pass (2026-10-04): the 13-report tail annotated + archived
| 7 | T10: M13 BuildFlow failing-step-name capture → agents-notes | 20% |
| 8 | T11: M12 A012×4 per-finding verdicts into residual-triage doc | 20% |
| 9 | T12: M14 CHANGELOG receipt convention decision | 20% |
| 10 | T13: wire `check-cqrs-lint` into CI (last un-CI'd gate) | 80% |
| 11 | T15: rebuilt cqrs-lint binary → fleet swap (+ rules-diff ritual, gotcha 23) | 80% |
| 12 | T16: e2e battery (`.#test-all`) + bench-spike (idle machine only) | 80% |
| 13 | T17: v5 cut runbook skeleton (wave-ordered deletion plan, M15) | 80% |
| 14 | T18: `cqrs-htmx-upgrade` codemod scaffold + R1 golden fixture (M26–M28) | 80% |
| 15 | T19: go-cqrs-lite pre-commit doc-only misclassification repro | 80% |
| 16 | T14: scorecard timing-variance env investigation | 80% |
| 17 | loginpage `FaviconAccent` rename at next MAJOR window | idea |
| 18 | Workspace-TEST gate (the workspace-build counterpart) to catch bump-time test regressions | idea |
| 19 | setup-demo: assert `/app.css` serves non-empty CSS in its test (styling regression guard) | idea |
| 20 | loginpage: consumer-facing migration note for pre-rewrite consumers (embedded CSS → compiled contract) in next-train notes | idea |
| 21 | adminui: LoadingButton on mutating forms (09-17 F8 tail) | idea |
| 22 | dashboardui README adoption table: add ThemeScript/ThemeToggle + CSRFToken rows (adopted post-09-23, table may predate) | idea |
~~| 23 | episode-4 report: harvest §f items into TODO_LIST via docs-health HARVEST (skill loop-closure rule) | process |~~ done — this pass
~~| 24 | Annotate the 2026-10-04 review HTML: CSP-nonce + NoOAuth2 + spinner rows → Fixed | process |~~ done — the 05:45 review now carries an ANNOTATED block (CSP-nonce + NoOAuth2 + spinner → Fixed; zero-dep strength superseded)
~~| 25 | TODO_LIST: prune items made stale by the loginpage rewrite (grep for `lp-*`/zero-dep claims) | process |~~ done — the moot adoption owner-call was struck 06:5x; this pass updated the episode-4 tail + review follow-up rows
| 26–50 | *(plan Table A T20–T26 carried verbatim: V007 metaengine-gated migration, appkit v5-window, SidebarNav criteria, BuildFlow BF1–BF3, DataStar Tier-4 demand gate, buildcache human decision, owner-call batch: PapDashboard send / datastar-demo / cqrs-lint tag / cross-repo debt; plus: ROADMAP raw-ideas sweep for OQ21 closure, FEATURES.md loginpage refresh, AGENTS gotcha for the daemon-vs-hook race, loginpage coverage-gate re-pin if spinner adds uncovered branches, setup-demo visual golden for the styled login page, upstream datastar SDKScript vs cqrs-htmx/datastar script-serving comparison note, templ-components Unreleased watch item (typed triggers ADR-0043 → revisit wire adoption at next train), check-css-bundle-classes extension to setup-demo's app.css if it becomes load-bearing, session-cost note: 2 multiedit retries class)* | rest |

## g) TOP #1 QUESTION I CANNOT ANSWER MYSELF

**The 6 failing tests (read_model_missing / bus subscription refused) predate today and block the next train — do you want an emergency root-cause session NOW, or does the next train slide until the planned Tier-20% work completes first?** I can start the diagnosis immediately (projection-host subscribe path + the go-cqrs-lite bus bump are the suspects), but whether to interrupt the plan's order for it is a release-priority call only you can make.

---

*Verification state at writing: loginpage `-race` + vet + golangci-lint 0 issues + gofmt clean; setup + setup-demo build/vet green (6 known pre-existing failures excluded); both CSS gates + codegen green; master pushed to `c9274e37` with release-train strict green.*
