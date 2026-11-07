# docs/research/ — Deep-Dive & Research Reports

**Convention (settled 2026-09-23, adoption-plan task T18):** research reports are
date-prefixed flat files in this directory (`YYYY-MM-DD_<topic>.<md|html>`),
discovered by listing — there is no hand-maintained master index to go stale.
A report becomes **load-bearing** when a module's README or AGENTS.md links it
as design rationale; those links are the index. Series that span multiple runs
cross-link via "superseded-by" annotations in the older report's outcome note
(append-only — never rewrite a published report's findings). A report whose
outcome is fully dispositioned (every option decided, successor shipped) moves
to [`archived/`](archived/) — the same ritual as `docs/status/`: an in-file
`> ARCHIVED <date> (<who>):` blockquote naming the successor, links re-based
at move time, and the series table above keeps the entry point (2026-11-05,
docs-health round 46 set the precedent with the theme-toggle spike).

**Eviction (2026-10-24, master plan S21):** the round-14 LEAVE verdict governed
sweeps, not retention forever. Unreferenced closed runs were evicted by the same
measured rule as the archive eviction: every file a living doc cites survives
(the load-bearing set incl. the entry points below); the rest is restorable via
[`docs/ARCHIVE_EVICTED.md`](../ARCHIVE_EVICTED.md). Superseded runs named in the
series table are history-in-git; the entry-point files remain. `docs/benchmarks/`
and `docs/proposals/` keep their own standing conventions (benchmark raw-baseline
policy; owner-gated decision packets) and were NOT evicted.

## Load-bearing series

| Series                                            | Runs                                                                                                                                                         | Entry point                                                                                                                                                    |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| go-cqrs-lite adoption audits                      | 2026-06-14 (v2 era, 42%) → 2026-07-30 (produced `docs/guides/leveraging-go-cqrs-lite.md`) → **2026-11-03 (73/100 high; 47 capabilities; v5 posture per ADR-0051)**                            | [`2026-11-03_go-cqrs-lite-deep-dive.html`](2026-11-03_go-cqrs-lite-deep-dive.html); linked from [`AGENTS.md`](../../AGENTS.md)                                 |
| templ-components in dashboardui (adoption audits) | 2026-09-17 (14/100) → 2026-09-23 (85→94, rubric §06) → 2026-10-04 (94 holds, repo-wide episode 4)                                                            | [`2026-10-04_templ-components-deep-dive.html`](2026-10-04_templ-components-deep-dive.html); linked from [`dashboardui/README.md`](../../dashboardui/README.md) |
| templ-components in loginpage (adoption audit)    | 2026-09-23 (verdict: keep zero-dep; triggers in ROADMAP OQ21) → **overturned 2026-10-04**: adopted in-tree, scored 95 (episode 4)                            | [`2026-10-04_templ-components-deep-dive.html`](2026-10-04_templ-components-deep-dive.html) (the 09-23 audit itself was evicted as uncited; its outcome is recorded here and in the 10-04 report)                                   |
| Theme toggle strategy (adminui spike)             | 2026-09-20 spike (recommended `data-theme`) → superseded by the shipped class-driven pattern (adminui 2026-09-21, commit `82a5e878`; dashboardui 2026-09-23) | [`2026-09-20_theme-toggle-strategy-spike.md`](archived/2026-09-20_theme-toggle-strategy-spike.md) — every option dispositioned inline (A shipped, B/defer not taken); ARCHIVED 2026-11-04 (docs-health round 46)                             |
| Fleet-wide library utilization audits (19 libs)   | **2026-11-05**: 13 used / 6 correctly-not-adopted; 1 version lag (go-etag v0.7.0, chore-only); 0 misuse; incorporates + updates the go-cqrs-lite and templ-components series | [`../review/libs/index.html`](../review/libs/index.html) (19 md reports beside it); supersedes `LIBRARY_INTEGRATIONS.md` (2026-05-13)                         |
| DataStar client-side deep dives                   | **2026-11-07**: upstream v1.0.4 studied against the embedded v1.0.3 adapter; demo `data-on:load` defect found, verified in headless Chromium, and FIXED same-session; the M11–M14 (ADR-0050 Tier-4) fact base | [`2026-11-07_datastar-client-side-deep-dive.html`](2026-11-07_datastar-client-side-deep-dive.html) |
