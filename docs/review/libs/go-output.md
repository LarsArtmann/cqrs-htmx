# go-output Adoption Report (NOT adopted — correctly)

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-output` (+ 13 submodules: d2, daghtml, delimited, escape, graph, markdown, markup, plantuml, serialization, table, tree, …) v0.38.7 · Checkout: `/home/lars/projects/go-output`

## Verdict

| Direct usage | Transitive | Latest | Verdict |
| --- | --- | --- | --- |
| **0 import sites** | yes — 14 submodules pinned indirect in `auditlog/go.mod` via `samber-do-auditlog` (its viewer renders through go-output) | v0.38.7 (verified 2026-11-05) | **Correctly not adopted** |

## What it is

"Write your data once, render it anywhere": structured data → 16 output formats (tables, trees, diagrams) with type-safe enums and zero-config color; root module has no heavy deps; renderers live in isolated submodules. Aimed at CLIs and human-facing terminal/report output.

## Why cqrs-htmx should NOT adopt it

cqrs-htmx renders **HTML over HTTP** through templ + templ-components (dedicated, golden-tested substrate) and JSON through go-codec; it produces no terminal output (no CLI output formats, no reports). The library's value proposition has no intersection with a web library's needs — and its table/tree/diagram renderers would only duplicate what templ-components and the auditlog bridge already own. Where the fleet needs it (samber-do-auditlog's viewer), it arrives transitively and correctly.

## Flip conditions

If `cmd/cqrs-htmx-upgrade` or a future fleet-inspection CLI grows human-facing tabular output (e.g. "list all mounted routes across modules"), go-output/table or /markdown would be the right tool **inside that CLI's go.mod**, not the library's.

## How to verify

```bash
rg -n 'larsartmann/go-output' --type go -g '!scripts/**' . | wc -l      # 0
rg -c 'go-output' auditlog/go.mod                                       # 14 submodule pins, indirect
git -C ~/projects/go-output ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- [docs/LIBRARY_INTEGRATIONS.md](../../LIBRARY_INTEGRATIONS.md) (2026-05-13, superseded) — same verdict.
