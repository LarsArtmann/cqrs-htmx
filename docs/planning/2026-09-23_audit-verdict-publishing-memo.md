# Options Memo — Publish templ-components Audit Verdicts Publicly?

**Created:** 2026-09-23 (adoption-plan task T10)
**Decision owner:** Lars (status-report §g-Q1/Q3 — unresolved)
**Question:** should the dashboardui templ-components adoption-audit series
(14/100 → 85 → 94, rubric v1) and its verdicts be promoted into public surfaces
(dashboardui README, pkg.go.dev-visible docs, website), or stay internal
(`docs/research/` + `docs/status/`)?

## Option A — keep internal (status quo)

**What:** audits + rubric stay in `docs/research/`; only the distilled,
evergreen parts ship publicly (what ALREADY happened: the README adoption
table + ADR-0053 divergences are public and carry the durable conclusions).

- ✅ Point-in-time scores (14, 85, 94) age badly in public docs; scores without
  a dated series context invite "why is it only 94?" questions.
- ✅ The consumer-relevant verdicts are already public where they matter
  (README table, ADR-0053, Theming section).
- ❌ Upstream (templ-components) misses the consumer evidence for its
  roadmap; the asks filed there (ListNote X–Y variant, CopyButton hook) carry
  the signal instead.

## Option B — publish the series

**What:** link the audit series from the dashboardui README's docs section
(done for the series index already), optionally surface the rubric on the
website (lars.software) as a "how we evaluate integrations" post.

- ✅ Selling signal: a measured 94/100 adoption story with a published rubric
  is strong evidence the panels are maintained, not abandonware.
- ❌ Every future audit run must then update the public number or explain
  drift — a permanent documentation tax.
- ❌ Rubric v1's judgment bands (±5 on run 2) are honest internally but
  thin armor publicly.

## Recommendation

**A, with the existing exceptions.** The durable conclusions are already
public (README adoption table, ADR-0053, series index in
`docs/research/README.md`); the scoring series stays internal. Revisit if the
website launches an "engineering" section that wants integration-quality
narratives (then: publish the RUBRIC as a method post, not the scores).
