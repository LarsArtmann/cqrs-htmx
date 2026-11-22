# go-output Adoption Report (NOT adopted — correctly)

> Audit date: 2026-11-22 (prior: 2026-11-14) · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-output` (+ 11 submodules: d2, daghtml, delimited, escape, graph, markdown, markup, plantuml, serialization, table, tree) v0.38.7 · Checkout: `/home/lars/projects/go-output`

## Verdict

| Direct usage       | Transitive                                                                                                                                                     | Latest                        | Verdict                   |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------- | ------------------------- |
| **0 import sites** | yes — root + 11 submodules pinned indirect at v0.38.7 in `auditlog/go.mod` (12 pins) and `integration_test/go.mod` (12 pins, go.mod:126-137) — every samber-do-auditlog consumer; its viewer renders through go-output | v0.38.7 (verified 2026-11-22) | **Correctly not adopted** |

## What it is

"Write your data once, render it anywhere": structured data → 16 output formats (tables, trees, diagrams) with type-safe enums and zero-config color; root module has no heavy deps; renderers live in isolated submodules. Aimed at CLIs and human-facing terminal/report output.

## Why cqrs-htmx should NOT adopt it

cqrs-htmx renders **HTML over HTTP** through templ + templ-components (dedicated, golden-tested substrate) and JSON through go-codec; it produces no terminal output (no CLI output formats, no reports). The library's value proposition has no intersection with a web library's needs — and its table/tree/diagram renderers would only duplicate what templ-components and the auditlog bridge already own. Where the fleet needs it (samber-do-auditlog's viewer), it arrives transitively and correctly.

## Flip conditions

If `cmd/cqrs-htmx-upgrade` or a future fleet-inspection CLI grows human-facing tabular output (e.g. "list all mounted routes across modules"), go-output/table or /markdown would be the right tool **inside that CLI's go.mod**, not the library's. The recorded watch point (FEATURES "Worth Considering" table) is the adminui CSV-export asks: the day an adminui surface generates tabular exports server-side, re-ask this verdict. Upstream check (2026-11-22): `[Unreleased]` is empty ("Nothing yet" both sections) — nothing changes the calculus.

## How to verify

```bash
rg -n 'larsartmann/go-output' --type go -g '!scripts/**' . | wc -l      # 0
rg -c 'go-output' auditlog/go.mod                                       # 12 pins (root + 11 submodules), indirect
rg -c 'go-output' integration_test/go.mod                               # 12 pins (indirect, go.mod:126-137)
git -C ~/projects/go-output ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1   # v0.38.7
```

## Prior reports

- [docs/LIBRARY_INTEGRATIONS.md](../../LIBRARY_INTEGRATIONS.md) (2026-05-13, superseded) — same verdict.
- 2026-11-05 (superseded by this refresh) — same verdict; pin count corrected (12 pins, not 14).
- 2026-11-14 (prior state of this file): not-adopted verdict, no score (not-adopted convention) · delta: verdict unchanged; second transitive pin-set receipt added (integration_test/go.mod), upstream `[Unreleased]` empty.
